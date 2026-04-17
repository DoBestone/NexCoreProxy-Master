// Package xrayrender 把 Master 下发的 ServerConfig 渲染成 xray-core 用的 config.json
//
// 只输出"够 xray 启动并正常工作"的最小集合，模板细节（streamSettings 完整结构）
// 由 Master 在 Inbound.StreamJSON 里填好；本包负责拼装、注入 clients、加 stats inbound。
//
// Phase 1 已实现：VLESS+Reality（XTLS-Vision）、Dokodemo-door（透传 relay）。
// Phase 1+ 将补：vmess、trojan、shadowsocks-2022、hysteria2、tuic 与 wrap 模式 outbound。
package xrayrender

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nexcoreproxy-agent/internal/protocol"
)

// CertDir 证书写盘的根目录；测试可重写
var CertDir = "/usr/local/etc/xray/certs"

// XrayConfig 是 xray-core 单文件配置的最小骨架（用 map 保留扩展性，避免给所有字段建 struct）
type XrayConfig struct {
	Log       map[string]any   `json:"log"`
	API       map[string]any   `json:"api"`
	Stats     map[string]any   `json:"stats"`
	Policy    map[string]any   `json:"policy"`
	Routing   map[string]any   `json:"routing"`
	Inbounds  []map[string]any `json:"inbounds"`
	Outbounds []map[string]any `json:"outbounds"`
}

// Render 入口：把 ServerConfig 转成 xray.json 字节流
func Render(cfg *protocol.ServerConfig, statsAPIPort int, logLevel string) ([]byte, error) {
	if logLevel == "" {
		logLevel = "warning"
	}
	x := &XrayConfig{
		Log: map[string]any{
			"loglevel": logLevel,
		},
		API: map[string]any{
			"tag":      "api",
			"services": []string{"HandlerService", "LoggerService", "StatsService"},
		},
		Stats: map[string]any{},
		Policy: map[string]any{
			"levels": map[string]any{
				"0": map[string]any{
					"statsUserUplink":   true,
					"statsUserDownlink": true,
				},
			},
			"system": map[string]any{
				"statsInboundUplink":   true,
				"statsInboundDownlink": true,
			},
		},
		Routing: map[string]any{
			"rules": []map[string]any{
				{
					"type":        "field",
					"inboundTag":  []string{"api"},
					"outboundTag": "api",
				},
			},
		},
		Outbounds: []map[string]any{
			{"protocol": "freedom", "tag": "direct"},
			{"protocol": "blackhole", "tag": "blocked"},
		},
	}

	// stats API inbound（必须，agent 通过 gRPC 拉流量）
	x.Inbounds = append(x.Inbounds, map[string]any{
		"tag":      "api",
		"listen":   "127.0.0.1",
		"port":     statsAPIPort,
		"protocol": "dokodemo-door",
		"settings": map[string]any{"address": "127.0.0.1"},
	})

	// 业务 inbound：根据 protocol 走不同渲染分支
	for _, inb := range cfg.Inbounds {
		rendered, err := renderInbound(&inb, cfg.Users)
		if err != nil {
			return nil, fmt.Errorf("render inbound %s: %w", inb.Tag, err)
		}
		if rendered != nil {
			x.Inbounds = append(x.Inbounds, rendered)
		}
	}

	// relay 节点：根据 mode 渲染 inbound + outbound + routing
	for _, r := range cfg.Relays {
		ins, outs, rules, err := renderRelay(&r, cfg.Users)
		if err != nil {
			return nil, fmt.Errorf("render relay %s: %w", r.Tag, err)
		}
		x.Inbounds = append(x.Inbounds, ins...)
		x.Outbounds = append(x.Outbounds, outs...)
		if existing, ok := x.Routing["rules"].([]map[string]any); ok {
			x.Routing["rules"] = append(existing, rules...)
		}
	}

	return json.MarshalIndent(x, "", "  ")
}

// renderInbound 根据协议把单个 Inbound 渲染成 xray inbound 对象（含 clients）
func renderInbound(inb *protocol.Inbound, users []protocol.User) (map[string]any, error) {
	clients := clientsForInbound(inb, users)

	switch strings.ToLower(inb.Protocol) {
	case "vless":
		return renderVLESS(inb, clients)
	case "vmess":
		return renderVMess(inb, clients)
	case "trojan":
		return renderTrojan(inb, clients)
	case "shadowsocks", "ss":
		return renderShadowsocks(inb, clients)
	case "hysteria2", "hy2":
		return renderHysteria2(inb, clients)
	case "tuic":
		return renderTUIC(inb, clients)
	case "dokodemo-door":
		// 一般用于 relay；保留入口便于直接配置
		return renderDokodemo(inb)
	default:
		// 兜底：未实现的协议，先跳过并记录（不阻断启动）
		return nil, nil
	}
}

func renderVMess(inb *protocol.Inbound, clients []map[string]any) (map[string]any, error) {
	settings := mergeSettings(inb.SettingsJSON, "clients")
	settings["clients"] = clients
	stream, err := buildStreamSettings(inb)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"tag":            inb.Tag,
		"listen":         valueOr(inb.Listen, "0.0.0.0"),
		"port":           inb.Port,
		"protocol":       "vmess",
		"settings":       settings,
		"streamSettings": stream,
		"sniffing":       decodeJSON(inb.SniffJSON),
	}, nil
}

func renderTrojan(inb *protocol.Inbound, clients []map[string]any) (map[string]any, error) {
	settings := mergeSettings(inb.SettingsJSON, "clients")
	settings["clients"] = clients
	stream, err := buildStreamSettings(inb)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"tag":            inb.Tag,
		"listen":         valueOr(inb.Listen, "0.0.0.0"),
		"port":           inb.Port,
		"protocol":       "trojan",
		"settings":       settings,
		"streamSettings": stream,
		"sniffing":       decodeJSON(inb.SniffJSON),
	}, nil
}

func renderShadowsocks(inb *protocol.Inbound, clients []map[string]any) (map[string]any, error) {
	// SS-2022 多用户：clients 转成 settings.clients；method 从 SettingsJSON 读
	settings := mergeSettings(inb.SettingsJSON, "clients", "password")
	if _, ok := settings["method"]; !ok {
		settings["method"] = "2022-blake3-aes-128-gcm"
	}
	// SS clients 形如 {email,password,method?}；这里复用 password 字段（trojan 字段没填时用 ss_pwd）
	ssClients := []map[string]any{}
	for _, c := range clients {
		ssClients = append(ssClients, map[string]any{
			"email":    c["email"],
			"password": c["password"],
		})
	}
	settings["clients"] = ssClients
	settings["network"] = "tcp,udp"
	return map[string]any{
		"tag":      inb.Tag,
		"listen":   valueOr(inb.Listen, "0.0.0.0"),
		"port":     inb.Port,
		"protocol": "shadowsocks",
		"settings": settings,
	}, nil
}

func renderHysteria2(inb *protocol.Inbound, clients []map[string]any) (map[string]any, error) {
	// xray 1.8.24+ 内置 hysteria2 inbound（实验性）；常见做法是在节点上用 sing-box 跑 hy2，
	// 这里先保留 stub，等 Phase 1+ 决定是否用 sing-box 旁挂。
	_ = clients
	settings := mergeSettings(inb.SettingsJSON)
	return map[string]any{
		"tag":      inb.Tag,
		"listen":   valueOr(inb.Listen, "0.0.0.0"),
		"port":     inb.Port,
		"protocol": "hysteria2",
		"settings": settings,
	}, nil
}

func renderTUIC(inb *protocol.Inbound, clients []map[string]any) (map[string]any, error) {
	_ = clients
	settings := mergeSettings(inb.SettingsJSON)
	return map[string]any{
		"tag":      inb.Tag,
		"listen":   valueOr(inb.Listen, "0.0.0.0"),
		"port":     inb.Port,
		"protocol": "tuic",
		"settings": settings,
	}, nil
}

// mergeSettings 解析 SettingsJSON 并删掉调用方明确接管的 key（防止用户配置覆盖 clients）
func mergeSettings(jsonStr string, drop ...string) map[string]any {
	m := decodeJSON(jsonStr)
	if m == nil {
		m = map[string]any{}
	}
	for _, k := range drop {
		delete(m, k)
	}
	return m
}

func renderVLESS(inb *protocol.Inbound, clients []map[string]any) (map[string]any, error) {
	settings := map[string]any{
		"clients":    clients,
		"decryption": "none",
	}
	// 如果 SettingsJSON 提供了 fallbacks 等额外字段，merge 进去
	if inb.SettingsJSON != "" {
		var extra map[string]any
		if err := json.Unmarshal([]byte(inb.SettingsJSON), &extra); err == nil {
			for k, v := range extra {
				if k == "clients" {
					continue // 永远以下发用户为准
				}
				settings[k] = v
			}
		}
	}

	streamSettings, err := buildStreamSettings(inb)
	if err != nil {
		return nil, err
	}

	out := map[string]any{
		"tag":            inb.Tag,
		"listen":         valueOr(inb.Listen, "0.0.0.0"),
		"port":           inb.Port,
		"protocol":       "vless",
		"settings":       settings,
		"streamSettings": streamSettings,
	}
	if sniff := decodeJSON(inb.SniffJSON); sniff != nil {
		out["sniffing"] = sniff
	}
	return out, nil
}

func renderDokodemo(inb *protocol.Inbound) (map[string]any, error) {
	settings := decodeJSON(inb.SettingsJSON)
	if settings == nil {
		settings = map[string]any{"network": "tcp,udp", "followRedirect": false}
	}
	return map[string]any{
		"tag":      inb.Tag,
		"listen":   valueOr(inb.Listen, "0.0.0.0"),
		"port":     inb.Port,
		"protocol": "dokodemo-door",
		"settings": settings,
	}, nil
}

// buildStreamSettings 优先用 Master 下发的 StreamJSON；如果是 reality 且字段缺失则补齐
func buildStreamSettings(inb *protocol.Inbound) (map[string]any, error) {
	stream := decodeJSON(inb.StreamJSON)
	if stream == nil {
		stream = map[string]any{}
	}
	if inb.Network != "" {
		stream["network"] = inb.Network
	}
	if inb.Security != "" {
		stream["security"] = inb.Security
	}
	if strings.EqualFold(inb.Security, "reality") {
		realitySettings := decodeJSONOr(inb.TLSJSON, map[string]any{})
		if inb.RealityPrivateKey != "" {
			realitySettings["privateKey"] = inb.RealityPrivateKey
		}
		if inb.RealityShortID != "" {
			if existing, ok := realitySettings["shortIds"].([]any); ok && len(existing) > 0 {
				// 已有则不覆盖
				_ = existing
			} else {
				realitySettings["shortIds"] = []string{inb.RealityShortID}
			}
		}
		if inb.RealityDest != "" {
			realitySettings["dest"] = inb.RealityDest
		}
		if inb.RealitySNI != "" {
			if _, ok := realitySettings["serverNames"]; !ok {
				realitySettings["serverNames"] = []string{inb.RealitySNI}
			}
		}
		stream["realitySettings"] = realitySettings
	} else if strings.EqualFold(inb.Security, "tls") {
		tls := decodeJSONOr(inb.TLSJSON, map[string]any{})
		// 自动签发的证书：落盘并写入 tlsSettings.certificates
		if inb.CertPEM != "" && inb.KeyPEM != "" && inb.CertDomain != "" {
			certFile, keyFile, err := writeCertFiles(inb.CertDomain, inb.CertPEM, inb.KeyPEM)
			if err == nil {
				tls["certificates"] = []map[string]any{{
					"certificateFile": certFile,
					"keyFile":         keyFile,
				}}
			}
		}
		stream["tlsSettings"] = tls
	}
	return stream, nil
}

// writeCertFiles 把 PEM 内容落到 CertDir/<domain>.{crt,key}
//
// 仅在内容变化时写盘（avoid xray reload 抖动）；权限 0600。
func writeCertFiles(domain, certPEM, keyPEM string) (certPath, keyPath string, err error) {
	if err = os.MkdirAll(CertDir, 0o755); err != nil {
		return
	}
	certPath = filepath.Join(CertDir, domain+".crt")
	keyPath = filepath.Join(CertDir, domain+".key")
	if err = writeIfChanged(certPath, []byte(certPEM), 0o644); err != nil {
		return
	}
	if err = writeIfChanged(keyPath, []byte(keyPEM), 0o600); err != nil {
		return
	}
	return
}

func writeIfChanged(path string, content []byte, mode os.FileMode) error {
	if old, err := os.ReadFile(path); err == nil && string(old) == string(content) {
		return nil
	}
	return os.WriteFile(path, content, mode)
}

// clientsForInbound 从用户列表里挑出授权了本 inbound 的，渲染成 xray clients
func clientsForInbound(inb *protocol.Inbound, users []protocol.User) []map[string]any {
	out := []map[string]any{}
	for _, u := range users {
		if !inboundAuthorized(inb.ID, u.InboundIDs) {
			continue
		}
		c := map[string]any{
			"email": u.Email,
		}
		// Reality 默认走 xtls-rprx-vision；其它协议字段不放
		switch strings.ToLower(inb.Protocol) {
		case "vless":
			c["id"] = u.UUID
			if strings.EqualFold(inb.Security, "reality") || strings.EqualFold(inb.Security, "tls") {
				c["flow"] = "xtls-rprx-vision"
			}
		case "vmess":
			c["id"] = u.UUID
		case "trojan":
			c["password"] = u.TrojanPassword
		case "shadowsocks", "ss":
			c["password"] = u.SS2022Password
		case "hysteria2", "hy2", "tuic":
			c["password"] = u.UUID
		}
		out = append(out, c)
	}
	return out
}

func inboundAuthorized(id uint, list []uint) bool {
	for _, x := range list {
		if x == id {
			return true
		}
	}
	return false
}

// renderRelay 渲染 relay 节点上一条转发记录对应的 xray 配置片段
//
// 返回 (inbounds, outbounds, routing rules)。
// Phase 1 实现 transparent 模式；wrap 模式留 stub，待 Step 10。
func renderRelay(r *protocol.Relay, users []protocol.User) ([]map[string]any, []map[string]any, []map[string]any, error) {
	switch r.Mode {
	case "transparent":
		inboundTag := r.Tag
		outboundTag := r.Tag + "-out"
		in := map[string]any{
			"tag":      inboundTag,
			"listen":   "0.0.0.0",
			"port":     r.ListenPort,
			"protocol": "dokodemo-door",
			"settings": map[string]any{
				"address": r.BackendIP,
				"port":    r.BackendPort,
				"network": "tcp,udp",
			},
		}
		out := map[string]any{
			"tag":      outboundTag,
			"protocol": "freedom",
		}
		rule := map[string]any{
			"type":        "field",
			"inboundTag":  []string{inboundTag},
			"outboundTag": outboundTag,
		}
		return []map[string]any{in}, []map[string]any{out}, []map[string]any{rule}, nil
	case "wrap":
		return renderWrapRelay(r, users)
	default:
		return nil, nil, nil, fmt.Errorf("unknown relay mode: %s", r.Mode)
	}
}

// renderWrapRelay 渲染 wrap 模式的中转
//
// 用户客户端连 Relay 节点的 wrap inbound（用户的 UUID/Trojan 密码），
// Relay 把流量经 trunk 凭据 outbound 到 Backend 的真正 inbound。
// 用户流量统计落在 Relay 这一跳；Backend 只看到一条 trunk 连接。
func renderWrapRelay(r *protocol.Relay, users []protocol.User) ([]map[string]any, []map[string]any, []map[string]any, error) {
	if r.WrapProtocol == "" {
		return nil, nil, nil, fmt.Errorf("wrap relay %s: wrap_protocol empty", r.Tag)
	}
	inboundTag := r.Tag
	outboundTag := r.Tag + "-out"

	// === Inbound：把 user 列表填入 clients ===
	clients := []map[string]any{}
	for _, u := range users {
		// wrap 模式下，user.InboundIDs 含被 wrap 的 backend inbound id
		// （Master agent_protocol.go buildRelay 中收集 wrapInboundIDs 时记录的就是 BackendInboundID）
		// 这里 r 没有直接持 backend inbound id，但全部 wrap 用户都对应本节点上的 wrap inbound，
		// 所以全量注入即可。
		c := map[string]any{"email": u.Email}
		switch r.WrapProtocol {
		case "vless":
			c["id"] = u.UUID
			if r.WrapStreamJSON != "" || r.WrapTLSJSON != "" {
				c["flow"] = "xtls-rprx-vision"
			}
		case "trojan":
			c["password"] = u.TrojanPassword
		}
		clients = append(clients, c)
	}

	// 构造 inbound
	inboundSettings := map[string]any{"clients": clients}
	if r.WrapProtocol == "vless" {
		inboundSettings["decryption"] = "none"
	}
	stream := decodeJSONOr(r.WrapStreamJSON, map[string]any{})
	if r.WrapStreamJSON == "" {
		// 兜底：reality + tcp + xtls-vision
		stream["network"] = "tcp"
		stream["security"] = "reality"
	}
	if _, ok := stream["security"]; ok && stream["security"] == "reality" {
		realitySettings := decodeJSONOr(r.WrapTLSJSON, map[string]any{})
		if r.WrapRealityPriv != "" {
			realitySettings["privateKey"] = r.WrapRealityPriv
		}
		if r.WrapRealityShort != "" {
			if _, has := realitySettings["shortIds"]; !has {
				realitySettings["shortIds"] = []string{r.WrapRealityShort}
			}
		}
		stream["realitySettings"] = realitySettings
	} else if _, ok := stream["security"]; ok && stream["security"] == "tls" {
		if tls := decodeJSON(r.WrapTLSJSON); tls != nil {
			stream["tlsSettings"] = tls
		}
	}
	in := map[string]any{
		"tag":            inboundTag,
		"listen":         "0.0.0.0",
		"port":           r.ListenPort,
		"protocol":       r.WrapProtocol,
		"settings":       inboundSettings,
		"streamSettings": stream,
	}

	// === Outbound：用 trunk 凭据连 backend ===
	out := map[string]any{
		"tag":      outboundTag,
		"protocol": r.BackendInboundProtocol,
	}
	switch r.BackendInboundProtocol {
	case "vless":
		out["settings"] = map[string]any{
			"vnext": []map[string]any{{
				"address": r.BackendIP,
				"port":    r.BackendPort,
				"users": []map[string]any{{
					"id":         r.TrunkUUID,
					"encryption": "none",
					"flow":       "xtls-rprx-vision",
				}},
			}},
		}
	case "vmess":
		out["settings"] = map[string]any{
			"vnext": []map[string]any{{
				"address": r.BackendIP,
				"port":    r.BackendPort,
				"users":   []map[string]any{{"id": r.TrunkUUID}},
			}},
		}
	case "trojan":
		out["settings"] = map[string]any{
			"servers": []map[string]any{{
				"address":  r.BackendIP,
				"port":     r.BackendPort,
				"password": r.TrunkPassword,
			}},
		}
	default:
		// 兜底：freedom，等于退化成 transparent
		out["protocol"] = "freedom"
		out["settings"] = map[string]any{}
	}
	if backendStream := decodeJSON(r.BackendInboundStream); backendStream != nil {
		out["streamSettings"] = backendStream
	}

	rule := map[string]any{
		"type":        "field",
		"inboundTag":  []string{inboundTag},
		"outboundTag": outboundTag,
	}
	return []map[string]any{in}, []map[string]any{out}, []map[string]any{rule}, nil
}

// --- 小工具 ---

func valueOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func decodeJSON(s string) map[string]any {
	if s == "" {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil
	}
	return out
}

func decodeJSONOr(s string, fallback map[string]any) map[string]any {
	if v := decodeJSON(s); v != nil {
		return v
	}
	return fallback
}
