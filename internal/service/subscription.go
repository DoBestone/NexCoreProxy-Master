// Package service · subscription.go — 用户订阅生成（自研 agent 架构版）
//
// 核心解析路径：
//
//	user
//	 └─ orders (status=paid)
//	     └─ packages
//	         └─ package_inbounds
//	             └─ inbounds (在 backend node 上)
//	                 ├─ 直连：inbound 自己的 host:port
//	                 └─ relays (BackendInboundID = inbound.id, enable, healthy)
//	                      ├─ transparent → 同协议 / 同 UUID，仅替换 host:port
//	                      └─ wrap        → 走 relay 自己的协议（用 wrap_* 字段）
//
// 每条 (inbound × path) 渲染成一个 ProxyNode（中间结构），再交给 format 渲染器拼最终格式。
package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"nexcoreproxy-master/internal/model"
)

type SubscriptionService struct{}

func NewSubscriptionService() *SubscriptionService { return &SubscriptionService{} }

// ProxyNode 是订阅链接前的中间表示。format 渲染器只读这个结构。
type ProxyNode struct {
	Name        string // 显示名，会进客户端节点列表
	Protocol    string // vless/vmess/trojan/ss/hysteria2/tuic
	Host        string // 客户端实际连接的地址（可能是 backend 也可能是 relay）
	Port        int
	Network     string // tcp/ws/grpc/h2/quic
	Security    string // none/tls/reality

	// 协议凭据
	UUID     string
	Password string // trojan/ss

	// stream 细节（直接传给客户端）
	StreamSettings map[string]any // 已渲染好的 streamSettings 子树
	TLSSettings    map[string]any
	RealitySettings map[string]any

	// 来源标记，渲染时排序/分组用
	IsRelay   bool
	RelayMode string // transparent / wrap
	Region    string // backend 节点 region
	Sort      int
}

// GenerateForUser 入口：返回该用户当前可见的所有 ProxyNode
func (s *SubscriptionService) GenerateForUser(userID uint) ([]ProxyNode, error) {
	db := model.GetDB()
	var user model.User
	if err := db.First(&user, userID).Error; err != nil {
		return nil, errors.New("用户不存在")
	}
	if !user.Enable {
		return nil, errors.New("用户已禁用")
	}
	if user.UUID == "" {
		// 老用户没 UUID 自动补
		user.UUID = randomUUID()
		_ = db.Model(&user).Update("uuid", user.UUID).Error
	}

	// 1. 找出该用户授权的 inbound
	var inbounds []model.Inbound
	err := db.Raw(`
		SELECT DISTINCT i.*
		FROM inbounds i
		JOIN package_inbounds pi ON pi.inbound_id = i.id
		JOIN orders o ON o.package_id = pi.package_id
		WHERE o.user_id = ? AND o.status = 'paid' AND i.enable = true
	`, userID).Scan(&inbounds).Error
	if err != nil {
		return nil, fmt.Errorf("query inbounds: %w", err)
	}
	if len(inbounds) == 0 {
		return []ProxyNode{}, nil
	}

	// 2. 拉这些 inbound 所在的节点信息（地址、region）
	nodeIDs := make([]uint, 0, len(inbounds))
	for _, inb := range inbounds {
		nodeIDs = append(nodeIDs, inb.NodeID)
	}
	var nodes []model.Node
	_ = db.Where("id IN ?", nodeIDs).Find(&nodes).Error
	nodeByID := make(map[uint]model.Node, len(nodes))
	for _, n := range nodes {
		nodeByID[n.ID] = n
	}

	// 3. 拉所有指向这些 inbound 的 healthy relay（含其 relay node）
	inboundIDs := make([]uint, 0, len(inbounds))
	for _, inb := range inbounds {
		inboundIDs = append(inboundIDs, inb.ID)
	}
	var relays []model.Relay
	_ = db.Where("backend_inbound_id IN ? AND enable = ? AND health_status != ?",
		inboundIDs, true, "bad").Find(&relays).Error
	relaysByInbound := make(map[uint][]model.Relay)
	relayNodeIDs := make([]uint, 0, len(relays))
	for _, r := range relays {
		relaysByInbound[r.BackendInboundID] = append(relaysByInbound[r.BackendInboundID], r)
		relayNodeIDs = append(relayNodeIDs, r.RelayNodeID)
	}
	var relayNodes []model.Node
	_ = db.Where("id IN ?", relayNodeIDs).Find(&relayNodes).Error
	relayNodeByID := make(map[uint]model.Node, len(relayNodes))
	for _, n := range relayNodes {
		relayNodeByID[n.ID] = n
	}

	// 4. 渲染：每个 inbound 一条直连 + 每个 relay 一条派生
	out := make([]ProxyNode, 0, len(inbounds))
	for _, inb := range inbounds {
		backendNode, ok := nodeByID[inb.NodeID]
		if !ok || !backendNode.Enable {
			continue
		}
		// 直连
		direct := buildDirectNode(&user, &inb, &backendNode)
		if direct != nil {
			out = append(out, *direct)
		}
		// 中转
		for _, r := range relaysByInbound[inb.ID] {
			relayNode, ok := relayNodeByID[r.RelayNodeID]
			if !ok || !relayNode.Enable {
				continue
			}
			pn := buildRelayNode(&user, &inb, &backendNode, &r, &relayNode)
			if pn != nil {
				out = append(out, *pn)
			}
		}
	}

	// 5. 排序：直连优先 → region → sort
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].IsRelay != out[j].IsRelay {
			return !out[i].IsRelay
		}
		if out[i].Region != out[j].Region {
			return out[i].Region < out[j].Region
		}
		return out[i].Sort < out[j].Sort
	})
	return out, nil
}

func buildDirectNode(u *model.User, inb *model.Inbound, n *model.Node) *ProxyNode {
	pn := &ProxyNode{
		Name:     fmt.Sprintf("%s | %s", n.Name, inb.Name),
		Protocol: inb.Protocol,
		Host:     n.IP,
		Port:     inb.Port,
		Network:  inb.Network,
		Security: inb.Security,
		Region:   n.Region,
		Sort:     inb.Sort,
	}
	fillCredsAndStream(pn, u, inb)
	return pn
}

func buildRelayNode(u *model.User, inb *model.Inbound, backend *model.Node,
	r *model.Relay, relayNode *model.Node) *ProxyNode {
	switch r.Mode {
	case "transparent":
		// 协议/凭据/stream 跟 backend 完全一致，仅替换 host:port
		pn := &ProxyNode{
			Name:      fmt.Sprintf("%s→%s | %s", relayNode.Name, backend.Name, inb.Name),
			Protocol:  inb.Protocol,
			Host:      relayNode.IP,
			Port:      r.ListenPort,
			Network:   inb.Network,
			Security:  inb.Security,
			Region:    relayNode.Region,
			Sort:      r.Sort,
			IsRelay:   true,
			RelayMode: "transparent",
		}
		fillCredsAndStream(pn, u, inb)
		return pn
	case "wrap":
		// 用 relay 自己的 wrap 协议参数
		pn := &ProxyNode{
			Name:      fmt.Sprintf("%s→%s [wrap] | %s", relayNode.Name, backend.Name, inb.Name),
			Protocol:  r.WrapProtocol,
			Host:      relayNode.IP,
			Port:      r.ListenPort,
			Security:  r.WrapSecurity,
			Region:    relayNode.Region,
			Sort:      r.Sort,
			IsRelay:   true,
			RelayMode: "wrap",
		}
		// 用户凭据复用（用户 UUID/trojan_pwd 在 wrap inbound 也有效）
		pn.UUID = u.UUID
		pn.Password = wrapPasswordFor(u, r.WrapProtocol)
		pn.StreamSettings = decodeJSONMap(r.WrapStreamJSON)
		pn.TLSSettings = decodeJSONMap(r.WrapTLSJSON)
		if r.WrapSecurity == "reality" {
			pn.RealitySettings = map[string]any{
				"publicKey": r.WrapRealityPub,
				"shortId":   r.WrapRealityShort,
			}
		}
		return pn
	}
	return nil
}

func fillCredsAndStream(pn *ProxyNode, u *model.User, inb *model.Inbound) {
	switch inb.Protocol {
	case "vless", "vmess":
		pn.UUID = u.UUID
	case "trojan":
		pn.Password = u.TrojanPassword
	case "ss":
		pn.Password = u.SS2022Password
	case "hysteria2", "tuic":
		pn.Password = u.UUID // hy2/tuic 共用 UUID 当 password
	}
	pn.StreamSettings = decodeJSONMap(inb.StreamJSON)
	pn.TLSSettings = decodeJSONMap(inb.TLSJSON)
	if inb.Security == "reality" {
		pn.RealitySettings = map[string]any{
			"publicKey": inb.RealityPublicKey,
			"shortId":   inb.RealityShortID,
			"sni":       inb.RealitySNI,
		}
	}
}

func wrapPasswordFor(u *model.User, proto string) string {
	switch proto {
	case "trojan":
		return u.TrojanPassword
	case "ss":
		return u.SS2022Password
	}
	return ""
}

func decodeJSONMap(s string) map[string]any {
	if s == "" {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil
	}
	return out
}
