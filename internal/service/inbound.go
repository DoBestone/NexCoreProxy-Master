package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"

	"nexcoreproxy-master/internal/model"
)

// realitySNIPool Reality 伪装域名池，新建时随机抽一个
var realitySNIPool = []struct {
	SNI  string
	Dest string
}{
	{"www.microsoft.com", "www.microsoft.com:443"},
	{"www.bing.com", "www.bing.com:443"},
	{"addons.mozilla.org", "addons.mozilla.org:443"},
	{"www.cloudflare.com", "www.cloudflare.com:443"},
	{"www.apple.com", "www.apple.com:443"},
	{"www.yahoo.co.jp", "www.yahoo.co.jp:443"},
}

// defaultPortForProtocol 没指定端口时给一个合理默认
func defaultPortForProtocol(proto string) int {
	switch strings.ToLower(proto) {
	case "vless":
		return 443
	case "vmess":
		return 2083
	case "trojan":
		return 8443
	case "ss", "shadowsocks":
		return 8388
	case "hysteria2", "hy2":
		return 443
	case "tuic":
		return 444
	}
	return 443
}

// InboundService 管理 Inbound 表
//
// 任何会改变 inbound 的操作都要 bump 该节点的 etag，并触发依赖该 inbound 的 RelayBinding 同步。
type InboundService struct {
	syncer *RelaySyncer
}

func NewInboundService() *InboundService {
	return &InboundService{syncer: NewRelaySyncer()}
}

// AttachSyncer 用于循环依赖时延迟注入；当前同进程构造能直接用 NewRelaySyncer，预留位
func (s *InboundService) AttachSyncer(syncer *RelaySyncer) { s.syncer = syncer }

// List 列出某节点的所有 inbound（管理用）
func (s *InboundService) List(nodeID uint) ([]model.Inbound, error) {
	var rows []model.Inbound
	q := model.GetDB().Order("sort, id")
	if nodeID > 0 {
		q = q.Where("node_id = ?", nodeID)
	}
	err := q.Find(&rows).Error
	return rows, err
}

// Get 查单条
func (s *InboundService) Get(id uint) (*model.Inbound, error) {
	var row model.Inbound
	if err := model.GetDB().First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// Create 创建 inbound
//
// 零配置模式：管理员只要选 protocol，其他字段全自动：
//   - port: 不填用协议默认
//   - tag: 不填按 <protocol>-<port>-<rand4> 生成
//   - name: 不填按 <Protocol>-<port> 生成
//   - security=reality 时自动生成 X25519 keypair + 8 字节 shortID + 随机池 SNI/dest
//   - ss 协议自动生成 method + psk
//   - trojan/vmess 有 ws 网络时自动生成随机 path
//   - hy2 自动生成 obfs password
//
// 完成后 bump etag 触发 agent 拉配置。
func (s *InboundService) Create(in *model.Inbound) error {
	if in.NodeID == 0 {
		return errors.New("node_id required")
	}
	if in.Protocol == "" {
		return errors.New("protocol required")
	}
	if err := s.autofill(in); err != nil {
		return err
	}
	if err := model.GetDB().Create(in).Error; err != nil {
		return fmt.Errorf("create inbound: %w", err)
	}
	if err := model.BumpEtag(in.NodeID); err != nil {
		return err
	}
	if s.syncer != nil {
		_ = s.syncer.OnInboundChanged(in.NodeID)
	}
	return nil
}

// autofill 给空字段填默认/随机值
func (s *InboundService) autofill(in *model.Inbound) error {
	proto := strings.ToLower(in.Protocol)
	in.Protocol = proto

	if in.Port <= 0 {
		in.Port = defaultPortForProtocol(proto)
	}
	if in.Listen == "" {
		in.Listen = "0.0.0.0"
	}
	if in.Network == "" {
		switch proto {
		case "ss", "shadowsocks", "hysteria2", "hy2", "tuic":
			// 这些协议没有 network 概念
		default:
			in.Network = "tcp"
		}
	}

	// security 默认：vless → reality（主推），trojan/vmess+ws → tls，其他 none
	if in.Security == "" {
		switch proto {
		case "vless":
			in.Security = "reality"
		case "trojan":
			in.Security = "tls"
		case "hysteria2", "hy2", "tuic":
			in.Security = "tls"
		default:
			in.Security = "none"
		}
	}

	// Reality 字段自动填
	if in.Security == "reality" {
		if in.RealityPrivateKey == "" || in.RealityPublicKey == "" {
			priv, pub, short, err := genRealityKeys()
			if err != nil {
				return fmt.Errorf("gen reality keys: %w", err)
			}
			in.RealityPrivateKey = priv
			in.RealityPublicKey = pub
			if in.RealityShortID == "" {
				in.RealityShortID = short
			}
		} else if in.RealityShortID == "" {
			in.RealityShortID = randomToken(4) // 8 hex chars
		}
		if in.RealitySNI == "" || in.RealityDest == "" {
			pick := realitySNIPool[rand.Intn(len(realitySNIPool))]
			if in.RealitySNI == "" {
				in.RealitySNI = pick.SNI
			}
			if in.RealityDest == "" {
				in.RealityDest = pick.Dest
			}
		}
	}

	// 协议 settings JSON（目前只给真的需要的协议填）
	if in.SettingsJSON == "" {
		switch proto {
		case "ss", "shadowsocks":
			settings, _ := json.Marshal(map[string]any{
				"method":   "2022-blake3-aes-128-gcm",
				"password": randomToken(16), // SS-2022 server key
			})
			in.SettingsJSON = string(settings)
		case "hysteria2", "hy2":
			settings, _ := json.Marshal(map[string]any{
				"obfs": map[string]any{"type": "salamander", "password": randomToken(16)},
			})
			in.SettingsJSON = string(settings)
		case "tuic":
			settings, _ := json.Marshal(map[string]any{"congestionControl": "bbr"})
			in.SettingsJSON = string(settings)
		}
	}

	// streamSettings JSON（ws 协议自动生成 path）
	if in.StreamJSON == "" && in.Network == "ws" {
		stream := map[string]any{
			"network":    "ws",
			"security":   in.Security,
			"wsSettings": map[string]any{"path": "/" + randomToken(4)},
		}
		b, _ := json.Marshal(stream)
		in.StreamJSON = string(b)
	}

	// Tag / Name 默认
	if in.Tag == "" {
		in.Tag = fmt.Sprintf("%s-%d-%s", shortProto(proto), in.Port, randomToken(2))
	}
	if in.Name == "" {
		in.Name = fmt.Sprintf("%s-%d", strings.ToUpper(shortProto(proto)), in.Port)
	}

	// 默认启用
	// (Enable bool 默认 false，要显式设 true，但 gorm default:true 会在插入时起作用)
	// 这里强制：新建就要启用
	in.Enable = true
	return nil
}

func shortProto(p string) string {
	switch p {
	case "vless":
		return "vl"
	case "vmess":
		return "vm"
	case "trojan":
		return "tj"
	case "ss", "shadowsocks":
		return "ss"
	case "hysteria2", "hy2":
		return "hy"
	case "tuic":
		return "tc"
	}
	return p
}

// Update 更新 inbound：写库 → bump 节点 etag → 同步 binding
func (s *InboundService) Update(in *model.Inbound) error {
	if in.ID == 0 {
		return errors.New("id required")
	}
	if err := model.GetDB().Save(in).Error; err != nil {
		return fmt.Errorf("update inbound: %w", err)
	}
	if err := model.BumpEtag(in.NodeID); err != nil {
		return err
	}
	if s.syncer != nil {
		_ = s.syncer.OnInboundChanged(in.NodeID)
	}
	return nil
}

// Delete 删除 inbound：先删库 → bump → 同步 binding（会清掉关联 relay）
func (s *InboundService) Delete(id uint) error {
	var in model.Inbound
	if err := model.GetDB().First(&in, id).Error; err != nil {
		return err
	}
	if err := model.GetDB().Delete(&in).Error; err != nil {
		return err
	}
	if err := model.BumpEtag(in.NodeID); err != nil {
		return err
	}
	if s.syncer != nil {
		_ = s.syncer.OnInboundChanged(in.NodeID)
	}
	return nil
}

// Toggle 切换 enable
func (s *InboundService) Toggle(id uint, enable bool) error {
	var in model.Inbound
	if err := model.GetDB().First(&in, id).Error; err != nil {
		return err
	}
	in.Enable = enable
	return s.Update(&in)
}
