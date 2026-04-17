package model

import "time"

// NodeConfigVersion 节点配置版本号（agent 拉配置 etag 协商用）
//
// Master 任何会影响某节点 xray.json 渲染结果的写操作（Inbound/Relay/User/Package 变更）
// 都要 BumpEtag(nodeID)。Agent 携带本地 etag 来拉，未变化返回 304。
type NodeConfigVersion struct {
	NodeID    uint      `json:"nodeId" gorm:"primaryKey"`
	Etag      string    `json:"etag" gorm:"size:32;not null"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// UserTraffic 用户级流量日志（替代旧 TrafficLog 的入站级粒度）
//
// Agent 上报 push 时按 (user, node, hour-bucket) 聚合写入，避免每分钟一行造成的写放大。
// 旧的 TrafficLog 表保留以便老代码继续编译，新代码只用 UserTraffic。
type UserTraffic struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     uint      `json:"userId" gorm:"uniqueIndex:idx_user_node_bucket,priority:1"`
	NodeID     uint      `json:"nodeId" gorm:"uniqueIndex:idx_user_node_bucket,priority:2;index"`
	BucketHour time.Time `json:"bucketHour" gorm:"uniqueIndex:idx_user_node_bucket,priority:3"`
	Upload     int64     `json:"upload"`
	Download   int64     `json:"download"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// NodeOnlineIP 节点在线 IP 记录（Agent push 时一并上报，用于 device_limit 软限）
type NodeOnlineIP struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"userId" gorm:"uniqueIndex:idx_uniq_user_node_ip,priority:1;index"`
	NodeID    uint      `json:"nodeId" gorm:"uniqueIndex:idx_uniq_user_node_ip,priority:2"`
	IP        string    `json:"ip" gorm:"uniqueIndex:idx_uniq_user_node_ip,priority:3;size:45"`
	LastSeen  time.Time `json:"lastSeen" gorm:"index"`
	CreatedAt time.Time `json:"createdAt"`
}

// NodeEvent 节点事件流（升级 / 同步 / 自愈 / 告警）
type NodeEvent struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	NodeID    uint      `json:"nodeId" gorm:"index;not null"`
	Type      string    `json:"type" gorm:"size:30;not null"` // upgrade/heal/sync/alert/install
	Level     string    `json:"level" gorm:"size:10;default:'info'"` // info/warn/error
	Message   string    `json:"message" gorm:"size:1000"`
	Detail    string    `json:"detail" gorm:"type:text"`
	CreatedAt time.Time `json:"createdAt" gorm:"index"`
}
