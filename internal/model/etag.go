package model

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// BumpEtag 给指定节点生成新 etag，触发 agent 下次拉配置时重新渲染 xray.json
//
// 任何会影响某节点 xray 配置的写操作都应该在事务收尾后调用本函数：
//   - Inbound 增删改 → bump 该 NodeID
//   - 受影响 Inbound 的关联 Relay → bump 那些 RelayNodeID
//   - User UUID/启用/超额变更 → bump 用户授权范围内所有 NodeID
//   - Relay / RelayBinding 增删改 → bump 对应 RelayNodeID
func BumpEtag(nodeID uint) error {
	if nodeID == 0 {
		return nil
	}
	etag := newEtag()
	now := time.Now()
	return db.Save(&NodeConfigVersion{
		NodeID:    nodeID,
		Etag:      etag,
		UpdatedAt: now,
	}).Error
}

// BumpEtags 批量 bump，去重
func BumpEtags(nodeIDs []uint) error {
	seen := make(map[uint]struct{}, len(nodeIDs))
	for _, id := range nodeIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if err := BumpEtag(id); err != nil {
			return err
		}
	}
	return nil
}

// GetEtag 读当前 etag，没有则返回空串（agent 拿到空串表示首次拉）
func GetEtag(nodeID uint) string {
	var v NodeConfigVersion
	if err := db.First(&v, nodeID).Error; err != nil {
		return ""
	}
	return v.Etag
}

func newEtag() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
