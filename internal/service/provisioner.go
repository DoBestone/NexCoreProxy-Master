package service

import (
	"encoding/json"
	"fmt"

	"nexcoreproxy-master/internal/model"
)

// NodeProvisioner 给新装好的 backend 节点自动生成"开箱即用"的入站集合
//
// 思路：管理员只输 IP/SSH 一键装完，系统自动塞一组主流入站（VLESS-Reality + Hy2 + ...），
// 客户端订阅里立刻就有可用节点，无需任何手工配置。
type NodeProvisioner struct {
	inboundSvc *InboundService
}

func NewNodeProvisioner(inb *InboundService) *NodeProvisioner {
	return &NodeProvisioner{inboundSvc: inb}
}

// InboundSet 预设方案
type InboundSet string

const (
	InboundSetMinimal  InboundSet = "minimal"  // 仅 VLESS-Reality
	InboundSetStandard InboundSet = "standard" // VLESS-Reality + Trojan-WS + SS-2022
	InboundSetFull     InboundSet = "full"     // standard + VMess-WS + Hy2 + TUIC
)

// Provision 给指定 backend 节点写入预设入站集
//
// 已经存在同名 inbound 的不会重复创建（按 Name 去重，幂等）。
// reality 密钥 / SS PSK / WS path 等敏感字段由系统自动生成。
func (p *NodeProvisioner) Provision(nodeID uint, set InboundSet) ([]model.Inbound, error) {
	defs := p.defsFor(set)
	if len(defs) == 0 {
		return nil, fmt.Errorf("unknown inbound set: %s", set)
	}

	existingNames := map[string]struct{}{}
	{
		var rows []model.Inbound
		_ = model.GetDB().Where("node_id = ?", nodeID).Find(&rows).Error
		for _, r := range rows {
			existingNames[r.Name] = struct{}{}
		}
	}

	created := []model.Inbound{}
	for _, d := range defs {
		if _, dup := existingNames[d.Name]; dup {
			continue
		}
		inb, err := p.materialize(nodeID, &d)
		if err != nil {
			return created, fmt.Errorf("materialize %s: %w", d.Name, err)
		}
		if err := p.inboundSvc.Create(inb); err != nil {
			return created, fmt.Errorf("create %s: %w", d.Name, err)
		}
		created = append(created, *inb)
	}
	return created, nil
}

// defs are blueprint structs (still need keys/passwords filled in)
type inboundDef struct {
	Name        string
	Tag         string
	Protocol    string
	Port        int
	PortRange   string
	Network     string
	Security    string
	RealitySNI  string
	RealityDest string
	SSMethod    string
	WSPath      string
}

func (p *NodeProvisioner) defsFor(set InboundSet) []inboundDef {
	reality := inboundDef{
		Name: "VLESS-Reality-443", Tag: "vl-reality-443",
		Protocol: "vless", Port: 443, Network: "tcp", Security: "reality",
		RealitySNI: "www.microsoft.com", RealityDest: "www.microsoft.com:443",
	}
	trojanWS := inboundDef{
		Name: "Trojan-WS-8443", Tag: "tj-ws-8443",
		Protocol: "trojan", Port: 8443, Network: "ws", Security: "tls",
	}
	ss := inboundDef{
		Name: "SS-2022-8388", Tag: "ss-8388",
		Protocol: "ss", Port: 8388, SSMethod: "2022-blake3-aes-128-gcm",
	}
	vmessWS := inboundDef{
		Name: "VMess-WS-2083", Tag: "vm-ws-2083",
		Protocol: "vmess", Port: 2083, Network: "ws", Security: "tls",
	}
	hy2 := inboundDef{
		Name: "Hysteria2-443", Tag: "hy2-443",
		Protocol: "hysteria2", Port: 443, PortRange: "20000-30000", Security: "tls",
	}
	tuic := inboundDef{
		Name: "TUIC-v5-444", Tag: "tuic-444",
		Protocol: "tuic", Port: 444, Security: "tls",
	}
	switch set {
	case InboundSetMinimal:
		return []inboundDef{reality}
	case InboundSetStandard:
		return []inboundDef{reality, trojanWS, ss}
	case InboundSetFull:
		return []inboundDef{reality, trojanWS, ss, vmessWS, hy2, tuic}
	}
	return nil
}

func (p *NodeProvisioner) materialize(nodeID uint, d *inboundDef) (*model.Inbound, error) {
	in := &model.Inbound{
		NodeID:    nodeID,
		Name:      d.Name,
		Tag:       d.Tag,
		Protocol:  d.Protocol,
		Listen:    "0.0.0.0",
		Port:      d.Port,
		PortRange: d.PortRange,
		Network:   d.Network,
		Security:  d.Security,
		Enable:    true,
	}

	switch d.Security {
	case "reality":
		priv, pub, short, err := genRealityKeys()
		if err != nil {
			return nil, err
		}
		in.RealityPrivateKey = priv
		in.RealityPublicKey = pub
		in.RealityShortID = short
		in.RealitySNI = d.RealitySNI
		in.RealityDest = d.RealityDest
		// stream / tls JSON 由 agent 渲染时根据 Reality* 字段补齐，无需写
	case "tls":
		// Phase 1 不预生成证书，留给 ACME 子系统（Step 15）；占位 CertDomain 为空
	}

	// 协议特定 settings
	switch d.Protocol {
	case "ss":
		settings, _ := json.Marshal(map[string]any{"method": d.SSMethod})
		in.SettingsJSON = string(settings)
	case "trojan", "vmess":
		if d.Network == "ws" {
			path := "/auto-" + randomToken(4)
			stream := map[string]any{
				"network":    "ws",
				"security":   d.Security,
				"wsSettings": map[string]any{"path": path},
			}
			b, _ := json.Marshal(stream)
			in.StreamJSON = string(b)
		}
	case "hysteria2":
		settings, _ := json.Marshal(map[string]any{
			"obfs": map[string]any{"type": "salamander", "password": randomToken(16)},
		})
		in.SettingsJSON = string(settings)
	case "tuic":
		settings, _ := json.Marshal(map[string]any{
			"congestionControl": "bbr",
		})
		in.SettingsJSON = string(settings)
	}
	return in, nil
}
