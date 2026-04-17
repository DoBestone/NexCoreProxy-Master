package model

import (
	"fmt"
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
//
// 只迁移自研 agent 架构使用的表。旧表（UserNode / InboundTemplate / TrafficLog / RelayRule）
// 已从该列表移除：fresh install 不再创建；存量表用 DropLegacyTables 主动 DROP。
//
// 旧表的 struct 仍保留在 model 包内（供个别 legacy handler 编译过），但运行时不依赖。
func AutoMigrate() error {
	if err := db.AutoMigrate(
		// 业务核心
		&User{},
		&Node{},
		&Package{},
		&Order{},
		&Announcement{},
		&EmailConfig{},
		&NexCoreConfig{},
		// 自研 agent 架构表
		&Inbound{},
		&PackageInbound{},
		&Relay{},
		&RelayBinding{},
		&NodeConfigVersion{},
		&UserTraffic{},
		&NodeOnlineIP{},
		&NodeEvent{},
		&AcmeAccount{},
		&Certificate{},
	); err != nil {
		return err
	}
	// 历史数据回填：已部署 ncp-agent 的节点标记为新后端
	db.Model(&Node{}).
		Where("(backend IS NULL OR backend = '' OR backend = 'xui') AND installed = ?", true).
		Update("backend", "ncp-agent")
	return nil
}

// IsAgentBackend 是否跑自研 agent（vs 老的 3x-ui）
func (n *Node) IsAgentBackend() bool {
	return n.Backend == "ncp-agent"
}

// DropLegacyTables 一次性 DROP 旧表（user_nodes / inbound_templates / traffic_logs / relay_rules）
//
// 由 main.go 的 --drop-legacy 命令调用。运行前自动备份建议手动做。
func DropLegacyTables() error {
	for _, t := range []string{"user_nodes", "inbound_templates", "traffic_logs", "relay_rules"} {
		if db.Migrator().HasTable(t) {
			if err := db.Migrator().DropTable(t); err != nil {
				return fmt.Errorf("drop %s: %w", t, err)
			}
		}
	}
	return nil
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
	// SubscribeToken 持久化订阅令牌，粘贴到客户端后长期有效；通过"重置"接口主动换发
	SubscribeToken string `json:"-" gorm:"size:64;uniqueIndex"`

	// per-user 协议凭据（自研 agent 架构核心：Master 是唯一权威）
	UUID           string `json:"uuid" gorm:"size:36;uniqueIndex"`     // VLESS/VMess
	TrojanPassword string `json:"-" gorm:"size:64"`                    // Trojan
	SS2022Password string `json:"-" gorm:"size:64"`                    // SS-2022 user PSK (base64)
	SpeedLimit     int    `json:"speedLimit" gorm:"default:0"`         // Mbps，0=不限
	DeviceLimit    int    `json:"deviceLimit" gorm:"default:0"`        // 在线设备数，0=不限
	ResetDay       int    `json:"resetDay" gorm:"default:1"`           // 月流量重置日 1-28

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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
	Role          string     `json:"role" gorm:"size:20;default:'backend';index"`    // backend | relay (新版语义，Type 保留兼容)
	Region        string     `json:"region" gorm:"size:50"`                          // 自动绑定/订阅分组用 (cn/hk/us/...)
	Installed     bool       `json:"installed" gorm:"default:false"`                 // ncp-agent 是否已成功部署
	AgentVersion  string     `json:"agentVersion" gorm:"size:30"`                    // ncp-agent 版本（push 时回报）
	// Backend 指示节点后端类型，影响运维 API 走哪条路径。
	//   "xui"       — 老节点通过 3x-ui panel API 管控（遗留，待退役）
	//   "ncp-agent" — 自研 agent + xray，通过 /v1/server/* HTTP 协议管控
	Backend string `json:"backend" gorm:"size:16;default:'xui';index"`
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
	ID       uint    `json:"id" gorm:"primaryKey"`
	Name     string  `json:"name" gorm:"size:100;not null"`
	Protocol string  `json:"protocol" gorm:"size:20"` // 兼容字段，新版按 PackageInbound 关联
	Traffic  int64   `json:"traffic"`                 // 流量(字节), 0=无限（与 TransferGB 二选一）
	Duration int     `json:"duration"`                // 时长(天), 0=永久
	Price    float64 `json:"price"`
	Nodes    int     `json:"nodes"` // 兼容字段

	// per-package 限额（新增）
	TransferGB  int `json:"transferGb" gorm:"default:0"`  // 月流量 GB，0=不限（覆盖 Traffic）
	DeviceLimit int `json:"deviceLimit" gorm:"default:0"` // 在线设备数限制
	SpeedLimit  int `json:"speedLimit" gorm:"default:0"`  // 限速 Mbps

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
