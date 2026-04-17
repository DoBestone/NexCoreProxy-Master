package service

import (
	"log"
	"time"

	"nexcoreproxy-master/internal/model"
)

// ScanExpired 扫描过期用户，标 enable=false + bump 受影响节点 etag
//
// 频率推荐：每 5 分钟一次。
func ScanExpired() {
	db := model.GetDB()
	now := time.Now()

	var expired []model.User
	if err := db.Where("enable = ? AND expire_at IS NOT NULL AND expire_at < ?", true, now).
		Find(&expired).Error; err != nil {
		log.Printf("[cron] ScanExpired query failed: %v", err)
		return
	}
	if len(expired) == 0 {
		return
	}

	ids := make([]uint, 0, len(expired))
	for _, u := range expired {
		ids = append(ids, u.ID)
	}
	if err := db.Model(&model.User{}).Where("id IN ?", ids).
		Update("enable", false).Error; err != nil {
		log.Printf("[cron] ScanExpired disable failed: %v", err)
		return
	}
	_ = bumpEtagsForUsers(ids)
	log.Printf("[cron] ScanExpired disabled %d users", len(ids))
}

// MonthlyReset 月流量重置
//
// 频率：每天 00:05 跑一次，命中当天 == users.reset_day 的用户重置 traffic_used 为 0。
// 兼容老用户 reset_day=0：跳过。
func MonthlyReset() {
	day := time.Now().Day()
	db := model.GetDB()

	var users []model.User
	if err := db.Where("reset_day = ? AND traffic_used > 0", day).Find(&users).Error; err != nil {
		log.Printf("[cron] MonthlyReset query failed: %v", err)
		return
	}
	if len(users) == 0 {
		return
	}
	ids := make([]uint, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}
	// 同步重新启用（之前因超额被自动停用的用户）
	if err := db.Model(&model.User{}).Where("id IN ?", ids).
		Updates(map[string]any{"traffic_used": 0, "enable": true}).Error; err != nil {
		log.Printf("[cron] MonthlyReset update failed: %v", err)
		return
	}
	_ = bumpEtagsForUsers(ids)
	log.Printf("[cron] MonthlyReset reset %d users for day %d", len(ids), day)
}

// CleanStaleOnlineIPs 清理超过 5 分钟没刷新的在线 IP 记录
//
// device_limit 软限基于 node_online_ips 行数；不清理会让 limit 越超越严。
func CleanStaleOnlineIPs() {
	cutoff := time.Now().Add(-5 * time.Minute)
	model.GetDB().Where("last_seen < ?", cutoff).Delete(&model.NodeOnlineIP{})
}
