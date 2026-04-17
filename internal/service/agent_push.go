package service

import (
	"strconv"
	"strings"
	"time"

	"nexcoreproxy-master/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AgentPushService 处理 agent 上报的 push 请求
//
// 三件事：
//  1. 流量入账：按 (user, node, hour-bucket) 桶 UPSERT 到 user_traffic，并 atomic 累加 users.traffic_used
//  2. 在线设备：写 node_online_ips（覆盖式），用于 device_limit 软限
//  3. 计算 kicks：用户超额 / 过期 / 被禁 → 返回 email 列表让 agent 立即从 xray runtime 删 client
//
// 注：本服务无状态，把 IO 全压到 DB；agent 端按 60s 推一次，假设入参聚合后流量增量。
type AgentPushService struct{}

func NewAgentPushService() *AgentPushService { return &AgentPushService{} }

// IngestStats 把流量增量写库 + 累加 users.traffic_used
//
// stats key 格式 "<user_id>@nx"，trunk@nx 等内部账号会被跳过。
// 0 字节不写库，避免无意义行膨胀。
func (s *AgentPushService) IngestStats(nodeID uint, stats map[string]TrafficDeltaDTO) error {
	if len(stats) == 0 {
		return nil
	}
	bucket := time.Now().Truncate(time.Hour)

	db := model.GetDB()
	return db.Transaction(func(tx *gorm.DB) error {
		for email, d := range stats {
			if d.Up == 0 && d.Down == 0 {
				continue
			}
			uid, ok := parseUserEmail(email)
			if !ok {
				continue
			}
			row := model.UserTraffic{
				UserID:     uid,
				NodeID:     nodeID,
				BucketHour: bucket,
				Upload:     d.Up,
				Download:   d.Down,
				UpdatedAt:  time.Now(),
			}
			// UPSERT：相同 (user,node,bucket) 累加，不同则插入
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "user_id"}, {Name: "node_id"}, {Name: "bucket_hour"}},
				DoUpdates: clause.Assignments(map[string]any{
					"upload":     gorm.Expr("upload + ?", d.Up),
					"download":   gorm.Expr("download + ?", d.Down),
					"updated_at": time.Now(),
				}),
			}).Create(&row).Error; err != nil {
				return err
			}
			// 累加用户总流量（traffic_used 单位：字节）
			if err := tx.Model(&model.User{}).Where("id = ?", uid).
				UpdateColumn("traffic_used", gorm.Expr("traffic_used + ?", d.Up+d.Down)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// IngestOnline 覆盖式刷新该节点上的 online IP 表
//
// 老数据按 LastSeen 滚动清理：超过 5 分钟未刷新视为离线（由别处定时任务清）。
// 这里只做 UPSERT。
func (s *AgentPushService) IngestOnline(nodeID uint, online map[string][]string) error {
	if len(online) == 0 {
		return nil
	}
	now := time.Now()
	db := model.GetDB()
	return db.Transaction(func(tx *gorm.DB) error {
		for email, ips := range online {
			uid, ok := parseUserEmail(email)
			if !ok {
				continue
			}
			for _, ip := range ips {
				if ip == "" {
					continue
				}
				row := model.NodeOnlineIP{
					UserID:    uid,
					NodeID:    nodeID,
					IP:        ip,
					LastSeen:  now,
					CreatedAt: now,
				}
				if err := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "user_id"}, {Name: "node_id"}, {Name: "ip"}},
					DoUpdates: clause.AssignmentColumns([]string{"last_seen"}),
				}).Create(&row).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// CalcKicks 计算需要从 xray runtime 立即剔除的用户 email 列表
//
// 触发条件：
//   - 超额：traffic_used >= traffic_limit (limit > 0)
//   - 过期：expire_at < now
//   - 被禁：enable = false
//
// 返回的 email 用 "<id>@nx" 格式，与 agent 在 xray 里写入的 client.email 对齐。
// 同时把超额用户标记为 enable=false 并 bump 节点 etag，让下次 pull 不再下发。
func (s *AgentPushService) CalcKicks(nodeID uint) ([]string, error) {
	db := model.GetDB()
	now := time.Now()

	type row struct {
		ID           uint
		Enable       bool
		ExpireAt     *time.Time
		TrafficLimit int64
		TrafficUsed  int64
	}
	// 这里用 inbound + package 反向找出"在该节点上有授权"的所有用户，再过滤无效的
	var users []row
	err := db.Raw(`
		SELECT DISTINCT u.id, u.enable, u.expire_at, u.traffic_limit, u.traffic_used
		FROM users u
		JOIN orders o        ON o.user_id = u.id AND o.status = 'paid'
		JOIN package_inbounds pi ON pi.package_id = o.package_id
		JOIN inbounds i      ON i.id = pi.inbound_id
		WHERE i.node_id = ? AND i.enable = true
	`, nodeID).Scan(&users).Error
	if err != nil {
		return nil, err
	}

	var (
		kicks       []string
		toDisableID []uint
	)
	for _, u := range users {
		bad := false
		switch {
		case !u.Enable:
			bad = true
		case u.ExpireAt != nil && u.ExpireAt.Before(now):
			bad = true
		case u.TrafficLimit > 0 && u.TrafficUsed >= u.TrafficLimit:
			bad = true
			toDisableID = append(toDisableID, u.ID)
		}
		if bad {
			kicks = append(kicks, userEmail(u.ID))
		}
	}

	if len(toDisableID) > 0 {
		// 超额自动停用，避免续费前继续被尝试下发
		if err := db.Model(&model.User{}).Where("id IN ?", toDisableID).
			Update("enable", false).Error; err != nil {
			return kicks, err
		}
		// bump 该用户授权范围内所有节点的 etag（其他节点 pull 时也会摘掉）
		_ = bumpEtagsForUsers(toDisableID)
	}
	return kicks, nil
}

// TrafficDeltaDTO push payload 里的流量增量；与 handler.TrafficDelta 同形，避免 service 反向依赖 handler
type TrafficDeltaDTO struct {
	Up   int64
	Down int64
}

// parseUserEmail 解析 "<id>@nx" → uint，不匹配返回 false
func parseUserEmail(email string) (uint, bool) {
	at := strings.IndexByte(email, '@')
	if at <= 0 {
		return 0, false
	}
	if email[at:] != "@nx" {
		return 0, false
	}
	n, err := strconv.ParseUint(email[:at], 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(n), true
}

// bumpEtagsForUsers 找出受影响的所有节点（这些用户在哪些节点上有授权）并 bump etag
func bumpEtagsForUsers(userIDs []uint) error {
	if len(userIDs) == 0 {
		return nil
	}
	var nodeIDs []uint
	err := model.GetDB().Raw(`
		SELECT DISTINCT i.node_id
		FROM users u
		JOIN orders o        ON o.user_id = u.id AND o.status = 'paid'
		JOIN package_inbounds pi ON pi.package_id = o.package_id
		JOIN inbounds i      ON i.id = pi.inbound_id
		WHERE u.id IN ?
	`, userIDs).Scan(&nodeIDs).Error
	if err != nil {
		return err
	}
	// 同时 bump 所有指向这些 backend inbound 的 relay 节点
	var relayNodeIDs []uint
	_ = model.GetDB().Raw(`
		SELECT DISTINCT r.relay_node_id
		FROM relays r
		JOIN inbounds i ON i.id = r.backend_inbound_id
		JOIN package_inbounds pi ON pi.inbound_id = i.id
		JOIN orders o ON o.package_id = pi.package_id
		WHERE o.user_id IN ?
	`, userIDs).Scan(&relayNodeIDs).Error
	return model.BumpEtags(append(nodeIDs, relayNodeIDs...))
}
