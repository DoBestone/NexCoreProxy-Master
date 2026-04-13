package model

import (
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB

// InitDB 初始化数据库
func InitDB(dsn string) error {
	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return nil
}

// GetDB 获取数据库连接
func GetDB() *gorm.DB {
	return db
}

// AutoMigrate 自动迁移表结构
func AutoMigrate() error {
	return db.AutoMigrate(
		&User{},
		&Node{},
		&UserNode{},
		&Package{},
		&Order{},
		&Ticket{},
		&TicketReply{},
		&InboundTemplate{},
		&TrafficLog{},
		&Announcement{},
		&EmailConfig{},
		&NexCoreConfig{},
		&RelayRule{},
	)
}

// User 用户模型
type User struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	Username     string     `json:"username" gorm:"uniqueIndex;size:50"`
	Password     string     `json:"-" gorm:"size:255"`
	Email        string     `json:"email" gorm:"size:100;index"`
	Role         string     `json:"role" gorm:"size:20;default:'user';index"` // admin, user
	Balance      float64    `json:"balance" gorm:"default:0"`                 // 余额
	TrafficLimit int64      `json:"trafficLimit"`                             // 流量限制 (字节)
	TrafficUsed  int64      `json:"trafficUsed"`                              // 已用流量
	ExpireAt     *time.Time `json:"expireAt"`                                 // 到期时间
	Enable       bool       `json:"enable" gorm:"default:true;index"`
	Remark       string     `json:"remark" gorm:"size:255"`
	InviteCode   string     `json:"inviteCode" gorm:"size:20;uniqueIndex"` // 邀请码
	InvitedBy    uint       `json:"invitedBy"`                             // 邀请人ID
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

// Announcement 公告模型
type Announcement struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Title     string    `json:"title" gorm:"size:200;not null"`
	Content   string    `json:"content" gorm:"type:text"`
	Type      string    `json:"type" gorm:"size:20;default:'info'"` // info, warning, success
	Pinned    bool      `json:"pinned" gorm:"default:false"`
	Enable    bool      `json:"enable" gorm:"default:true"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// EmailConfig 邮件配置模型 (SMTP Lite API)
type EmailConfig struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	APIURL    string    `json:"apiUrl" gorm:"column:api_url;size:255"`  // SMTP Lite API 地址
	APIKey    string    `json:"-" gorm:"column:api_key;size:255"` // API Key
	FromName  string    `json:"fromName" gorm:"size:100"`               // 发件人名称
	Enable    bool      `json:"enable" gorm:"default:false"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Node 节点模型
type Node struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	Name          string     `json:"name" gorm:"size:100;not null"`
	IP            string     `json:"ip" gorm:"size:50"`
	Port          int        `json:"port" gorm:"default:54321"`
	Username      string     `json:"username" gorm:"size:50"`
	Password      string     `json:"-" gorm:"size:255"`
	SSHPort       int        `json:"sshPort" gorm:"default:22"`
	SSHUser       string     `json:"sshUser" gorm:"size:50"`
	SSHPassword   string     `json:"-" gorm:"size:255"`
	AgentKey      string     `json:"agentKey" gorm:"size:64;uniqueIndex"` // Agent连接密钥
	APIToken      string     `json:"-" gorm:"size:255"`                   // ncp-api Token
	APIPort       int        `json:"apiPort" gorm:"default:54322"`        // ncp-api 端口
	MasterURL     string     `json:"masterUrl" gorm:"size:255"`           // Master地址
	Type          string     `json:"type" gorm:"size:20;default:'standalone';index"` // standalone, relay, backend
	Enable        bool       `json:"enable" gorm:"default:true"`
	Remark        string     `json:"remark" gorm:"size:255"`
	Status        string     `json:"status" gorm:"size:20;default:'unknown'"`
	XrayVersion   string     `json:"xrayVersion" gorm:"size:20"`
	CPU           float64    `json:"cpu"`
	Memory        float64    `json:"memory"`
	Disk          float64    `json:"disk"`
	Uptime        uint64     `json:"uptime"`
	UploadTotal   int64      `json:"uploadTotal"`
	DownloadTotal int64      `json:"downloadTotal"`
	LastSyncAt    *time.Time `json:"lastSyncAt"`
	HostKeyFingerprint string `json:"-" gorm:"size:100"` // SSH 主机密钥指纹 (TOFU)
	Connected          bool   `json:"connected" gorm:"-"` // 是否在线（运行时）
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// UserNode 用户节点关联
type UserNode struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	UserID    uint       `json:"userId" gorm:"index:idx_user_enable"`
	NodeID    uint       `json:"nodeId" gorm:"index"`
	InboundID int        `json:"inboundId"`
	Remark    string     `json:"remark" gorm:"size:100"`
	Enable    bool       `json:"enable" gorm:"default:true;index:idx_user_enable"`
	ExpireAt  *time.Time `json:"expireAt"`
	CreatedAt time.Time  `json:"createdAt"`
	Node      Node       `json:"node" gorm:"foreignKey:NodeID"`
}

// Package 套餐模型
type Package struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:100;not null"`
	Protocol  string    `json:"protocol" gorm:"size:20"` // vmess, vless, trojan, all
	Traffic   int64     `json:"traffic"`                 // 流量(字节), 0=无限
	Duration  int       `json:"duration"`                // 时长(天), 0=永久
	Price     float64   `json:"price"`                   // 价格
	Nodes     int       `json:"nodes"`                   // 可用节点数, 0=无限
	Remark    string    `json:"remark" gorm:"size:255"`
	Sort      int       `json:"sort" gorm:"default:0"`
	Enable    bool      `json:"enable" gorm:"default:true"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Order 订单模型
type Order struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	OrderNo     string     `json:"orderNo" gorm:"uniqueIndex;size:32"`
	UserID      uint       `json:"userId" gorm:"index"`
	PackageID   uint       `json:"packageId"`
	PackageName string     `json:"packageName" gorm:"size:100"`
	Amount      float64    `json:"amount"`                                  // 金额
	Status      string     `json:"status" gorm:"size:20;default:'pending'"` // pending, paid, cancelled, refunded
	PayMethod   string     `json:"payMethod" gorm:"size:20"`                // alipay, wechat, balance
	PaidAt      *time.Time `json:"paidAt"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	User        User       `json:"user" gorm:"foreignKey:UserID"`
}

// Ticket 工单模型
type Ticket struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"userId" gorm:"index"`
	Subject   string    `json:"subject" gorm:"size:200"`
	Content   string    `json:"content" gorm:"type:text"`
	Status    string    `json:"status" gorm:"size:20;default:'open';index"` // open, closed
	Priority  int       `json:"priority" gorm:"default:0"`                  // 0=普通, 1=紧急
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	User      User      `json:"user" gorm:"foreignKey:UserID"`
}

// TicketReply 工单回复
type TicketReply struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	TicketID  uint      `json:"ticketId" gorm:"index"`
	UserID    uint      `json:"userId"` // 0 = 管理员回复
	Content   string    `json:"content" gorm:"type:text"`
	CreatedAt time.Time `json:"createdAt"`
}

// InboundTemplate 入站模板
type InboundTemplate struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:100;not null"`
	Protocol  string    `json:"protocol" gorm:"size:20;not null"`
	Port      int       `json:"port"`
	Settings  string    `json:"settings" gorm:"type:text"`
	Stream    string    `json:"stream" gorm:"type:text"`
	TLS       string    `json:"tls" gorm:"type:text"`
	Remark    string    `json:"remark" gorm:"size:255"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TrafficLog 流量日志
type TrafficLog struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	NodeID     uint      `json:"nodeId" gorm:"index:idx_node_recorded"`
	InboundID  int       `json:"inboundId"`
	Upload     int64     `json:"upload"`
	Download   int64     `json:"download"`
	RecordedAt time.Time `json:"recordedAt" gorm:"index:idx_node_recorded"`
	CreatedAt  time.Time `json:"createdAt"`
}

// RelayRule 中转规则
type RelayRule struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	RelayNodeID      uint      `json:"relayNodeId" gorm:"index:idx_relay_backend,priority:1"`
	BackendNodeID    uint      `json:"backendNodeId" gorm:"index:idx_relay_backend,priority:2"`
	RelayInboundPort int       `json:"relayInboundPort"`                  // 中转节点上的入站端口
	RelayInboundTag  string    `json:"relayInboundTag" gorm:"size:100"`  // Xray inbound tag
	RelayOutboundTag string    `json:"relayOutboundTag" gorm:"size:100"` // Xray outbound tag
	Protocol         string    `json:"protocol" gorm:"size:20"`           // vmess/vless/trojan/shadowsocks
	Enable           bool      `json:"enable" gorm:"default:true"`
	Remark           string    `json:"remark" gorm:"size:255"`
	SyncStatus       string    `json:"syncStatus" gorm:"size:20;default:'pending'"` // pending/synced/error
	SyncError        string    `json:"syncError" gorm:"size:500"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	RelayNode        Node      `json:"relayNode" gorm:"foreignKey:RelayNodeID"`
	BackendNode      Node      `json:"backendNode" gorm:"foreignKey:BackendNodeID"`
}

// NexCoreConfig NexCore 代理配置（用于在线更新）
type NexCoreConfig struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ProxyURL  string    `json:"proxyUrl" gorm:"column:proxy_url;size:255"`
	RepoToken string    `json:"-" gorm:"column:repo_token;size:255"`
	Owner     string    `json:"owner" gorm:"size:100"`
	Repo      string    `json:"repo" gorm:"size:100"`
	Enabled   bool      `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
