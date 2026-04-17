package service

import (
	"errors"
	"strconv"
	"time"

	"nexcoreproxy-master/internal/model"

	"gorm.io/gorm"
)

// AgentConfigService 渲染 Master ↔ ncp-agent 协议数据
//
// 设计原则：Master 是唯一权威，但只下发"结构化数据"，xray.json 的最终拼装由 agent 完成。
// 这样 Master 不需要内嵌 xray 协议知识，新协议只在 agent 渲染层加。
type AgentConfigService struct{}

func NewAgentConfigService() *AgentConfigService { return &AgentConfigService{} }

// AgentConfig 是 GET /api/v1/server/config 的响应体
type AgentConfig struct {
	Etag     string             `json:"etag"`
	Node     AgentNodeMeta      `json:"node"`
	Inbounds []AgentInbound     `json:"inbounds"`
	Users    []AgentUser        `json:"users"`
	Relays   []AgentRelay       `json:"relays"`
	Settings AgentRuntimeConfig `json:"settings"`
}

type AgentNodeMeta struct {
	ID                uint   `json:"id"`
	Role              string `json:"role"`         // backend / relay
	Region            string `json:"region"`
	AgentTarget       string `json:"agentTarget"`  // 目标 agent 版本（不一致触发自升级）
	XrayTarget        string `json:"xrayTarget"`   // 目标 xray 版本
	AgentDownloadURL  string `json:"agentDownloadUrl,omitempty"` // 含 ${ARCH} 占位符
	AgentSHA256URL    string `json:"agentSha256Url,omitempty"`   // 可选；空表示不校验
}

type AgentRuntimeConfig struct {
	PullInterval int    `json:"pullInterval"` // 秒
	PushInterval int    `json:"pushInterval"`
	LogLevel     string `json:"logLevel"`
}

type AgentInbound struct {
	ID                uint   `json:"id"`
	Tag               string `json:"tag"`
	Protocol          string `json:"protocol"`
	Listen            string `json:"listen"`
	Port              int    `json:"port"`
	PortRange         string `json:"portRange,omitempty"`
	Network           string `json:"network"`
	Security          string `json:"security"`
	StreamJSON        string `json:"streamJson,omitempty"`
	TLSJSON           string `json:"tlsJson,omitempty"`
	SniffJSON         string `json:"sniffJson,omitempty"`
	SettingsJSON      string `json:"settingsJson,omitempty"`
	RealityPrivateKey string `json:"realityPrivateKey,omitempty"`
	RealityShortID    string `json:"realityShortId,omitempty"`
	RealitySNI        string `json:"realitySni,omitempty"`
	RealityDest       string `json:"realityDest,omitempty"`
	CertDomain        string `json:"certDomain,omitempty"`
	CertPEM           string `json:"certPem,omitempty"` // 自动签发的证书内容（agent 落盘）
	KeyPEM            string `json:"keyPem,omitempty"`
}

type AgentUser struct {
	ID             uint    `json:"id"`
	Email          string  `json:"email"`
	UUID           string  `json:"uuid"`
	TrojanPassword string  `json:"trojanPwd,omitempty"`
	SS2022Password string  `json:"ssPwd,omitempty"`
	LimitIP        int     `json:"limitIp,omitempty"`
	SpeedMbps      int     `json:"speedMbps,omitempty"`
	ExpiryMs       int64   `json:"expiryMs,omitempty"`
	InboundIDs     []uint  `json:"inboundIds"` // 该用户在本节点上能用哪些 inbound
}

// AgentRelay relay 节点上的转发条目，agent 根据 mode 渲染对应 xray inbound+outbound+routing
type AgentRelay struct {
	ID         uint   `json:"id"`
	Tag        string `json:"tag"`
	Mode       string `json:"mode"` // transparent | wrap
	ListenPort int    `json:"listenPort"`
	PortRange  string `json:"portRange,omitempty"`

	// 后端落地（递归解析多级中转后的最终目标）
	BackendIP   string `json:"backendIp"`
	BackendPort int    `json:"backendPort"`

	// transparent 模式：直接 dokodemo-door → backend
	// wrap 模式：暴露 wrap inbound → outbound 用 trunk 凭据连 backend
	WrapProtocol     string `json:"wrapProtocol,omitempty"`
	WrapStreamJSON   string `json:"wrapStreamJson,omitempty"`
	WrapTLSJSON      string `json:"wrapTlsJson,omitempty"`
	WrapRealityPriv  string `json:"wrapRealityPriv,omitempty"`
	WrapRealityShort string `json:"wrapRealityShort,omitempty"`
	TrunkUUID        string `json:"trunkUuid,omitempty"`
	TrunkPassword    string `json:"trunkPassword,omitempty"`

	// 后端 inbound 元信息（wrap 模式 outbound 渲染要用）
	BackendInboundProtocol string `json:"backendInboundProtocol,omitempty"`
	BackendInboundStream  string `json:"backendInboundStream,omitempty"`
	BackendInboundTLS     string `json:"backendInboundTls,omitempty"`
}

// Build 渲染指定节点的完整 config
//
// 调用方：handler.GetAgentConfig（被 agent 周期性调用）。
func (s *AgentConfigService) Build(node *model.Node) (*AgentConfig, error) {
	rt := GetRuntimeConfig()
	cfg := &AgentConfig{
		Node: AgentNodeMeta{
			ID:               node.ID,
			Role:             nodeRole(node),
			Region:           node.Region,
			AgentTarget:      rt.AgentTargetVer,
			XrayTarget:       rt.XrayVersion,
			AgentDownloadURL: rt.AgentBinaryURL,
		},
		Settings: AgentRuntimeConfig{
			PullInterval: 60,
			PushInterval: 60,
			LogLevel:     "warning",
		},
	}

	switch nodeRole(node) {
	case "relay":
		if err := s.buildRelay(cfg, node.ID); err != nil {
			return nil, err
		}
	default: // backend / standalone
		if err := s.buildBackend(cfg, node.ID); err != nil {
			return nil, err
		}
	}

	cfg.Etag = model.GetEtag(node.ID)
	if cfg.Etag == "" {
		// 首次没有版本号，立即写一个，让后续 304 能生效
		_ = model.BumpEtag(node.ID)
		cfg.Etag = model.GetEtag(node.ID)
	}
	return cfg, nil
}

func (s *AgentConfigService) buildBackend(cfg *AgentConfig, nodeID uint) error {
	db := model.GetDB()

	var inbounds []model.Inbound
	if err := db.Where("node_id = ? AND enable = ?", nodeID, true).
		Order("sort, id").Find(&inbounds).Error; err != nil {
		return err
	}
	cfg.Inbounds = make([]AgentInbound, 0, len(inbounds))
	inboundIDSet := make(map[uint]struct{}, len(inbounds))
	for _, inb := range inbounds {
		inboundIDSet[inb.ID] = struct{}{}
		ai := AgentInbound{
			ID:                inb.ID,
			Tag:               inb.Tag,
			Protocol:          inb.Protocol,
			Listen:            inb.Listen,
			Port:              inb.Port,
			PortRange:         inb.PortRange,
			Network:           inb.Network,
			Security:          inb.Security,
			StreamJSON:        inb.StreamJSON,
			TLSJSON:           inb.TLSJSON,
			SniffJSON:         inb.SniffJSON,
			SettingsJSON:      inb.SettingsJSON,
			RealityPrivateKey: inb.RealityPrivateKey,
			RealityShortID:    inb.RealityShortID,
			RealitySNI:        inb.RealitySNI,
			RealityDest:       inb.RealityDest,
			CertDomain:        inb.CertDomain,
		}
		// 注入证书（如有）
		if inb.CertDomain != "" {
			var cert model.Certificate
			if err := model.GetDB().Where("domain = ? AND status = ?", inb.CertDomain, "issued").
				First(&cert).Error; err == nil {
				ai.CertPEM = cert.CertPEM
				ai.KeyPEM = cert.KeyPEM
			}
		}
		cfg.Inbounds = append(cfg.Inbounds, ai)
	}

	if len(inboundIDSet) == 0 {
		cfg.Users = []AgentUser{}
		return nil
	}

	users, userInboundMap, err := s.activeUsersForInbounds(inboundIDSet)
	if err != nil {
		return err
	}
	cfg.Users = make([]AgentUser, 0, len(users))
	for _, u := range users {
		assigned := userInboundMap[u.ID]
		if len(assigned) == 0 {
			continue
		}
		var expiryMs int64
		if u.ExpireAt != nil {
			expiryMs = u.ExpireAt.UnixMilli()
		}
		cfg.Users = append(cfg.Users, AgentUser{
			ID:             u.ID,
			Email:          userEmail(u.ID),
			UUID:           u.UUID,
			TrojanPassword: u.TrojanPassword,
			SS2022Password: u.SS2022Password,
			LimitIP:        u.DeviceLimit,
			SpeedMbps:      u.SpeedLimit,
			ExpiryMs:       expiryMs,
			InboundIDs:     assigned,
		})
	}
	return nil
}

func (s *AgentConfigService) buildRelay(cfg *AgentConfig, nodeID uint) error {
	db := model.GetDB()

	var relays []model.Relay
	if err := db.Where("relay_node_id = ? AND enable = ?", nodeID, true).
		Order("sort, id").Find(&relays).Error; err != nil {
		return err
	}

	cfg.Relays = make([]AgentRelay, 0, len(relays))
	wrapInboundIDs := make(map[uint]struct{})

	for _, r := range relays {
		// 解析 backend 地址（递归处理多级中转）
		backendIP, backendPort, backendInb, err := resolveRelayTarget(&r)
		if err != nil {
			continue
		}
		ar := AgentRelay{
			ID:                     r.ID,
			Tag:                    relayTag(&r),
			Mode:                   r.Mode,
			ListenPort:             r.ListenPort,
			PortRange:              r.ListenPortRange,
			BackendIP:              backendIP,
			BackendPort:            backendPort,
			BackendInboundProtocol: backendInb.Protocol,
			BackendInboundStream:   backendInb.StreamJSON,
			BackendInboundTLS:      backendInb.TLSJSON,
		}
		if r.Mode == "wrap" {
			ar.WrapProtocol = r.WrapProtocol
			ar.WrapStreamJSON = r.WrapStreamJSON
			ar.WrapTLSJSON = r.WrapTLSJSON
			ar.WrapRealityPriv = r.WrapRealityPriv
			ar.WrapRealityShort = r.WrapRealityShort
			ar.TrunkUUID = r.TrunkUUID
			ar.TrunkPassword = r.TrunkPassword
			wrapInboundIDs[r.BackendInboundID] = struct{}{}
		}
		cfg.Relays = append(cfg.Relays, ar)
	}

	// wrap 模式 relay 上要下发用户列表（用户在 relay 这一跳认证）
	if len(wrapInboundIDs) > 0 {
		users, userInboundMap, err := s.activeUsersForInbounds(wrapInboundIDs)
		if err != nil {
			return err
		}
		cfg.Users = make([]AgentUser, 0, len(users))
		for _, u := range users {
			assigned := userInboundMap[u.ID]
			if len(assigned) == 0 {
				continue
			}
			var expiryMs int64
			if u.ExpireAt != nil {
				expiryMs = u.ExpireAt.UnixMilli()
			}
			cfg.Users = append(cfg.Users, AgentUser{
				ID:             u.ID,
				Email:          userEmail(u.ID),
				UUID:           u.UUID,
				TrojanPassword: u.TrojanPassword,
				SS2022Password: u.SS2022Password,
				LimitIP:        u.DeviceLimit,
				SpeedMbps:      u.SpeedLimit,
				ExpiryMs:       expiryMs,
				InboundIDs:     assigned,
			})
		}
	}
	return nil
}

// activeUsersForInbounds 找出"有效且至少授权了入参里某一个 inbound"的用户
//
// 返回 users 列表 + map[userID][]inboundID（每个用户在该节点上授权的 inbound 子集）。
func (s *AgentConfigService) activeUsersForInbounds(inboundIDs map[uint]struct{}) ([]model.User, map[uint][]uint, error) {
	if len(inboundIDs) == 0 {
		return nil, map[uint][]uint{}, nil
	}
	db := model.GetDB()

	ids := make([]uint, 0, len(inboundIDs))
	for id := range inboundIDs {
		ids = append(ids, id)
	}

	// 1. 找出关联了这些 inbound 的所有 PackageInbound
	var pis []model.PackageInbound
	if err := db.Where("inbound_id IN ?", ids).Find(&pis).Error; err != nil {
		return nil, nil, err
	}
	if len(pis) == 0 {
		return nil, map[uint][]uint{}, nil
	}
	// package_id → []inbound_id（该套餐覆盖到本节点上的哪些 inbound）
	pkgToInbounds := make(map[uint][]uint)
	for _, pi := range pis {
		pkgToInbounds[pi.PackageID] = append(pkgToInbounds[pi.PackageID], pi.InboundID)
	}

	pkgIDs := make([]uint, 0, len(pkgToInbounds))
	for id := range pkgToInbounds {
		pkgIDs = append(pkgIDs, id)
	}

	// 2. 找出"持有这些套餐且有效"的订单
	now := time.Now()
	var orders []model.Order
	if err := db.Where("package_id IN ? AND status = ?", pkgIDs, "paid").
		Find(&orders).Error; err != nil {
		return nil, nil, err
	}

	// user_id → 套餐 ID 集合
	userPkg := make(map[uint]map[uint]struct{})
	for _, o := range orders {
		if userPkg[o.UserID] == nil {
			userPkg[o.UserID] = make(map[uint]struct{})
		}
		userPkg[o.UserID][o.PackageID] = struct{}{}
	}
	if len(userPkg) == 0 {
		return nil, map[uint][]uint{}, nil
	}

	// 3. 拉用户基本信息 + 过滤过期/超额/禁用
	userIDs := make([]uint, 0, len(userPkg))
	for id := range userPkg {
		userIDs = append(userIDs, id)
	}
	var users []model.User
	if err := db.Where("id IN ? AND enable = ?", userIDs, true).Find(&users).Error; err != nil {
		return nil, nil, err
	}

	result := make([]model.User, 0, len(users))
	userInboundMap := make(map[uint][]uint, len(users))
	for _, u := range users {
		if u.UUID == "" {
			continue
		}
		if u.ExpireAt != nil && u.ExpireAt.Before(now) {
			continue
		}
		if u.TrafficLimit > 0 && u.TrafficUsed >= u.TrafficLimit {
			continue
		}
		// 收集该用户在本节点上能访问的 inbound 集合
		seen := make(map[uint]struct{})
		for pkg := range userPkg[u.ID] {
			for _, inbID := range pkgToInbounds[pkg] {
				seen[inbID] = struct{}{}
			}
		}
		if len(seen) == 0 {
			continue
		}
		assigned := make([]uint, 0, len(seen))
		for id := range seen {
			assigned = append(assigned, id)
		}
		userInboundMap[u.ID] = assigned
		result = append(result, u)
	}
	return result, userInboundMap, nil
}

// resolveRelayTarget 解析 relay 的最终落地目标
//
// 多级中转：A → B → backend，返回 A 的 outbound 应该连到 B.ListenPort 的中间地址；
// 顶层（ViaRelayID=0）直接返回 backend 节点 IP + Inbound 端口。
func resolveRelayTarget(r *model.Relay) (ip string, port int, backendInb model.Inbound, err error) {
	db := model.GetDB()
	if err = db.First(&backendInb, r.BackendInboundID).Error; err != nil {
		return
	}
	if r.ViaRelayID != 0 {
		var via model.Relay
		if err = db.First(&via, r.ViaRelayID).Error; err != nil {
			return
		}
		var viaNode model.Node
		if err = db.First(&viaNode, via.RelayNodeID).Error; err != nil {
			return
		}
		ip = viaNode.IP
		port = via.ListenPort
		return
	}
	var backendNode model.Node
	if err = db.First(&backendNode, backendInb.NodeID).Error; err != nil {
		return
	}
	ip = backendNode.IP
	port = backendInb.Port
	return
}

func nodeRole(n *model.Node) string {
	if n.Role != "" {
		return n.Role
	}
	if n.Type == "relay" {
		return "relay"
	}
	return "backend"
}

// userEmail 用户在 xray 里的 email tag，固定 <id>@nx，stats key 也用它
func userEmail(id uint) string {
	return strconv.FormatUint(uint64(id), 10) + "@nx"
}

func relayTag(r *model.Relay) string {
	prefix := "rl-r"
	if r.Mode == "wrap" {
		prefix = "wrap-r"
	}
	return prefix + strconv.FormatUint(uint64(r.ID), 10)
}

// LookupNodeByToken agent 鉴权用：拿 Bearer 里的 AgentKey 反查 Node
func LookupNodeByToken(token string) (*model.Node, error) {
	if token == "" {
		return nil, errors.New("empty token")
	}
	var node model.Node
	if err := model.GetDB().Where("agent_key = ?", token).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid token")
		}
		return nil, err
	}
	if !node.Enable {
		return nil, errors.New("node disabled")
	}
	return &node, nil
}
