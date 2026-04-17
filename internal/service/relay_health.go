package service

import (
	"net"
	"strconv"
	"sync"
	"time"

	"nexcoreproxy-master/internal/model"
)

// RelayHealthChecker 周期 TCP 探测每条 Relay 的可达性
//
// 失败 3 次标 bad，恢复后立即标 ok。bad 状态会让订阅渲染时跳过该条目。
// 探测路径：probe relay 节点的监听端口 + probe backend inbound 端口；
// 多级中转还会检查 via relay 是否健康（任一环 bad → 整链 bad）。
type RelayHealthChecker struct {
	probeTimeout time.Duration
	failThresh   int
}

func NewRelayHealthChecker() *RelayHealthChecker {
	return &RelayHealthChecker{
		probeTimeout: 3 * time.Second,
		failThresh:   3,
	}
}

// CheckAll 跑一轮全量探测；通常由 cron 每 60s 调一次
func (c *RelayHealthChecker) CheckAll() {
	db := model.GetDB()
	var relays []model.Relay
	if err := db.Where("enable = ?", true).Find(&relays).Error; err != nil {
		return
	}

	// 缓存节点（同一 RelayNode 可能出现多次，避免重复查 DB）
	nodeCache := sync.Map{}
	loadNode := func(id uint) *model.Node {
		if v, ok := nodeCache.Load(id); ok {
			return v.(*model.Node)
		}
		var n model.Node
		if err := db.First(&n, id).Error; err != nil {
			return nil
		}
		nodeCache.Store(id, &n)
		return &n
	}
	loadInbound := func(id uint) *model.Inbound {
		var inb model.Inbound
		if err := db.First(&inb, id).Error; err != nil {
			return nil
		}
		return &inb
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 20) // 并发上限，避免节点多时网络过载

	for i := range relays {
		r := &relays[i]
		wg.Add(1)
		sem <- struct{}{}
		go func(r *model.Relay) {
			defer wg.Done()
			defer func() { <-sem }()
			c.probeOne(r, loadNode, loadInbound)
		}(r)
	}
	wg.Wait()
}

func (c *RelayHealthChecker) probeOne(r *model.Relay,
	loadNode func(uint) *model.Node,
	loadInbound func(uint) *model.Inbound) {

	relayNode := loadNode(r.RelayNodeID)
	backendInb := loadInbound(r.BackendInboundID)
	if relayNode == nil || backendInb == nil {
		return
	}
	backendNode := loadNode(backendInb.NodeID)
	if backendNode == nil {
		return
	}

	ok := true

	// 1. relay 监听端口
	if !tcpProbe(relayNode.IP, r.ListenPort, c.probeTimeout) {
		ok = false
	}
	// 2. backend inbound 端口
	if ok && !tcpProbe(backendNode.IP, backendInb.Port, c.probeTimeout) {
		ok = false
	}
	// 3. 多级：via 也要健康
	if ok && r.ViaRelayID != 0 {
		var via model.Relay
		if err := model.GetDB().First(&via, r.ViaRelayID).Error; err == nil {
			if via.HealthStatus == "bad" {
				ok = false
			}
		}
	}

	now := time.Now()
	updates := map[string]any{"last_health_at": now}
	prevStatus := r.HealthStatus
	if ok {
		updates["health_status"] = "ok"
		updates["health_fail_count"] = 0
	} else {
		updates["health_fail_count"] = r.HealthFailCount + 1
		if r.HealthFailCount+1 >= c.failThresh {
			updates["health_status"] = "bad"
		}
	}
	_ = model.GetDB().Model(r).Updates(updates).Error

	// 状态翻转 → bump etag，让订阅刷新
	newStatus, _ := updates["health_status"].(string)
	if newStatus != "" && newStatus != prevStatus {
		_ = model.BumpEtag(r.RelayNodeID)
	}
}

func tcpProbe(host string, port int, timeout time.Duration) bool {
	if host == "" || port <= 0 {
		return false
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
