package model

import "time"

// Inbound 节点入站定义（取代旧 InboundTemplate / UserNode 的"协议端"角色）
//
// 一条 Inbound = 一台 Backend 节点上的一个 xray inbound 实例。
// Tag 全局唯一，作为 xray inbound tag 直接落进 config.json。
type Inbound struct {
	ID     uint   `json:"id" gorm:"primaryKey"`
	NodeID uint   `json:"nodeId" gorm:"index;not null"`
	Tag    string `json:"tag" gorm:"size:64;uniqueIndex;not null"`
	Name   string `json:"name" gorm:"size:100;not null"`

	// 协议核心
	Protocol  string `json:"protocol" gorm:"size:20;not null"` // vless/vmess/trojan/ss/hysteria2/tuic/anytls
	Listen    string `json:"listen" gorm:"size:50;default:'0.0.0.0'"`
	Port      int    `json:"port" gorm:"not null"`
	PortRange string `json:"portRange" gorm:"size:50"` // Hy2 端口跳跃 "20000-30000"
	Network   string `json:"network" gorm:"size:20"`   // tcp/ws/grpc/h2/quic
	Security  string `json:"security" gorm:"size:20"`  // none/tls/reality

	// 各段完整 JSON（直接写入 xray inbound）
	StreamJSON   string `json:"streamJson" gorm:"type:text"`   // streamSettings
	TLSJSON      string `json:"tlsJson" gorm:"type:text"`      // tls/reality
	SniffJSON    string `json:"sniffJson" gorm:"type:text"`    // sniffing
	SettingsJSON string `json:"settingsJson" gorm:"type:text"` // 协议特定 settings 模板（不含 clients）

	// 自动化字段（自动生成的密钥/证书引用）
	RealityPrivateKey string `json:"-" gorm:"size:128"`
	RealityPublicKey  string `json:"realityPublicKey" gorm:"size:128"`
	RealityShortID    string `json:"realityShortId" gorm:"size:32"`
	RealitySNI        string `json:"realitySni" gorm:"size:255"`
	RealityDest       string `json:"realityDest" gorm:"size:255"`
	CertDomain        string `json:"certDomain" gorm:"size:255"` // 自动签发证书域名

	Enable    bool      `json:"enable" gorm:"default:true;index"`
	Sort      int       `json:"sort" gorm:"default:0"`
	Remark    string    `json:"remark" gorm:"size:255"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// PackageInbound 套餐 ↔ 入站 关联（取代 UserNode 的"授权关系"）
type PackageInbound struct {
	ID        uint `json:"id" gorm:"primaryKey"`
	PackageID uint `json:"packageId" gorm:"index:idx_pkg_inb,unique"`
	InboundID uint `json:"inboundId" gorm:"index:idx_pkg_inb,unique"`
}

// Relay 中转记录（一条 = relay 节点上一个转发端口指向某 backend inbound）
type Relay struct {
	ID               uint   `json:"id" gorm:"primaryKey"`
	Name             string `json:"name" gorm:"size:100"`
	RelayNodeID      uint   `json:"relayNodeId" gorm:"index;not null"`
	BackendInboundID uint   `json:"backendInboundId" gorm:"index;not null"`

	Mode            string `json:"mode" gorm:"size:20;not null"` // transparent | wrap
	ListenPort      int    `json:"listenPort"`
	ListenPortRange string `json:"listenPortRange" gorm:"size:50"`

	// wrap 模式字段
	WrapProtocol      string `json:"wrapProtocol" gorm:"size:20"`
	WrapSecurity      string `json:"wrapSecurity" gorm:"size:20"`
	WrapStreamJSON    string `json:"wrapStreamJson" gorm:"type:text"`
	WrapTLSJSON       string `json:"wrapTlsJson" gorm:"type:text"`
	WrapRealityPriv   string `json:"-" gorm:"size:128"`
	WrapRealityPub    string `json:"wrapRealityPub" gorm:"size:128"`
	WrapRealityShort  string `json:"wrapRealityShort" gorm:"size:32"`
	TrunkUUID         string `json:"trunkUuid" gorm:"size:36"`
	TrunkPassword     string `json:"trunkPassword" gorm:"size:64"`

	// 多级中转
	ViaRelayID uint `json:"viaRelayId" gorm:"index"`

	// 来源（自动绑定生成 vs 管理员手工建）
	Source     string `json:"source" gorm:"size:20;default:'manual'"`
	BindingID  uint   `json:"bindingId" gorm:"index"`
	PortLocked bool   `json:"portLocked" gorm:"default:false"`

	// 健康
	HealthStatus    string     `json:"healthStatus" gorm:"size:20;default:'unknown'"` // ok/bad/unknown
	HealthFailCount int        `json:"healthFailCount" gorm:"default:0"`
	LastHealthAt    *time.Time `json:"lastHealthAt"`

	Enable    bool      `json:"enable" gorm:"default:true;index"`
	Sort      int       `json:"sort" gorm:"default:0"`
	Remark    string    `json:"remark" gorm:"size:255"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// RelayBinding Relay 节点 → Backend 节点 的整体绑定，自动同步生成 Relay 记录
type RelayBinding struct {
	ID            uint `json:"id" gorm:"primaryKey"`
	RelayNodeID   uint `json:"relayNodeId" gorm:"uniqueIndex:idx_rb_pair;not null"`
	BackendNodeID uint `json:"backendNodeId" gorm:"uniqueIndex:idx_rb_pair;not null"`

	Mode string `json:"mode" gorm:"size:20;default:'transparent'"`

	PortStrategy  string `json:"portStrategy" gorm:"size:20;default:'same'"` // same | offset | pool
	PortOffset    int    `json:"portOffset" gorm:"default:0"`
	PortPoolStart int    `json:"portPoolStart" gorm:"default:0"`
	PortPoolEnd   int    `json:"portPoolEnd" gorm:"default:0"`

	WrapProtocol   string `json:"wrapProtocol" gorm:"size:20"`
	WrapSecurity   string `json:"wrapSecurity" gorm:"size:20"`
	WrapStreamTpl  string `json:"wrapStreamTpl" gorm:"type:text"`
	AutoGenReality bool   `json:"autoGenReality" gorm:"default:true"`

	ViaRelayID uint `json:"viaRelayId"`

	Enable   bool `json:"enable" gorm:"default:true"`
	AutoSync bool `json:"autoSync" gorm:"default:true"`

	Remark    string    `json:"remark" gorm:"size:255"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
