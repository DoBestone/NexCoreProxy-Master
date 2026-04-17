package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"
)

func yamlMarshal(v any) (string, error) {
	b, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// SubscriptionFormat 订阅格式枚举
type SubscriptionFormat string

const (
	FormatV2rayN  SubscriptionFormat = "v2rayn"  // base64(URI list) — 兼容 v2rayN/NG/Shadowrocket/Quantumult/Loon
	FormatClash   SubscriptionFormat = "clash"   // Clash.Meta yaml
	FormatSingBox SubscriptionFormat = "singbox" // sing-box config json
)

// DetectFormat 按 UA 嗅探最适合的格式；如 query 强制 type 优先
func DetectFormat(ua, forceType string) SubscriptionFormat {
	if forceType != "" {
		switch strings.ToLower(forceType) {
		case "clash", "yaml":
			return FormatClash
		case "singbox", "sing-box", "json":
			return FormatSingBox
		case "v2rayn", "base64", "uri":
			return FormatV2rayN
		}
	}
	uaLower := strings.ToLower(ua)
	switch {
	case strings.Contains(uaLower, "clash"), strings.Contains(uaLower, "stash"):
		return FormatClash
	case strings.Contains(uaLower, "sing-box"), strings.Contains(uaLower, "sfa"),
		strings.Contains(uaLower, "sfi"), strings.Contains(uaLower, "sfm"):
		return FormatSingBox
	default:
		return FormatV2rayN
	}
}

// Render 把 ProxyNode 列表渲染成订阅文本（base64/yaml/json）
func Render(format SubscriptionFormat, nodes []ProxyNode) (body string, contentType string) {
	switch format {
	case FormatClash:
		return renderClash(nodes), "text/yaml; charset=utf-8"
	case FormatSingBox:
		return renderSingBox(nodes), "application/json; charset=utf-8"
	default:
		return renderV2rayN(nodes), "text/plain; charset=utf-8"
	}
}

// --- v2rayN base64 (URI list) ---

func renderV2rayN(nodes []ProxyNode) string {
	var lines []string
	for _, n := range nodes {
		uri := nodeToURI(&n)
		if uri != "" {
			lines = append(lines, uri)
		}
	}
	return base64.StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n")))
}

// nodeToURI 把单个 ProxyNode 转成对应协议的 URI
func nodeToURI(n *ProxyNode) string {
	switch n.Protocol {
	case "vless":
		return vlessURI(n)
	case "vmess":
		return vmessURI(n)
	case "trojan":
		return trojanURI(n)
	case "shadowsocks", "ss":
		return ssURI(n)
	case "hysteria2", "hy2":
		return hysteria2URI(n)
	case "tuic":
		return tuicURI(n)
	}
	return ""
}

// vmess://base64({v,ps,add,port,id,aid,scy,net,type,host,path,tls,sni,alpn,fp})
func vmessURI(n *ProxyNode) string {
	if n.UUID == "" || n.Host == "" {
		return ""
	}
	host, path := wsHostPath(n)
	body := map[string]any{
		"v":    "2",
		"ps":   n.Name,
		"add":  n.Host,
		"port": fmt.Sprintf("%d", n.Port),
		"id":   n.UUID,
		"aid":  "0",
		"scy":  "auto",
		"net":  valueOr(n.Network, "tcp"),
		"type": "none",
		"host": host,
		"path": path,
		"tls":  "",
	}
	if n.Security == "tls" {
		body["tls"] = "tls"
		if sni := pickStreamHost(n); sni != "" {
			body["sni"] = sni
		}
	}
	raw, _ := json.Marshal(body)
	return "vmess://" + base64.StdEncoding.EncodeToString(raw)
}

// ss://base64(method:[serverPSK:]userPSK)@host:port#name
//
// SS-2022 多用户模式需要 server_psk:user_psk 两段（xray/sing-box 规范）：
//   ss://base64(2022-blake3-aes-128-gcm:server_psk:user_psk)@...
// 老版 SS 单用户只一段：
//   ss://base64(aes-128-gcm:user_psk)@...
//
// 2022-blake3-* 协议必须带 server_psk 才能和 xray 多用户服务端对上。
func ssURI(n *ProxyNode) string {
	if n.Password == "" {
		return ""
	}
	method, _ := n.StreamSettings["method"].(string)
	if method == "" {
		method = "2022-blake3-aes-128-gcm"
	}
	serverPSK, _ := n.StreamSettings["password"].(string) // 从 inbound.settings 读 server-level PSK

	var userInfoRaw string
	if strings.HasPrefix(method, "2022-") && serverPSK != "" {
		userInfoRaw = method + ":" + serverPSK + ":" + n.Password
	} else {
		userInfoRaw = method + ":" + n.Password
	}
	userInfo := base64.StdEncoding.EncodeToString([]byte(userInfoRaw))
	return fmt.Sprintf("ss://%s@%s:%d#%s", userInfo, n.Host, n.Port, url.QueryEscape(n.Name))
}

// hysteria2://password@host:port?sni=...&insecure=0#name
func hysteria2URI(n *ProxyNode) string {
	if n.Password == "" {
		return ""
	}
	q := url.Values{}
	if sni := pickStreamHost(n); sni != "" {
		q.Set("sni", sni)
	}
	q.Set("insecure", "0")
	return fmt.Sprintf("hysteria2://%s@%s:%d?%s#%s",
		url.QueryEscape(n.Password), n.Host, n.Port, q.Encode(), url.QueryEscape(n.Name))
}

// tuic://uuid:password@host:port?congestion_control=bbr&alpn=h3#name
func tuicURI(n *ProxyNode) string {
	if n.UUID == "" || n.Password == "" {
		return ""
	}
	q := url.Values{}
	q.Set("congestion_control", "bbr")
	q.Set("alpn", "h3")
	if sni := pickStreamHost(n); sni != "" {
		q.Set("sni", sni)
	}
	return fmt.Sprintf("tuic://%s:%s@%s:%d?%s#%s",
		n.UUID, url.QueryEscape(n.Password),
		n.Host, n.Port, q.Encode(), url.QueryEscape(n.Name))
}

// vless://uuid@host:port?type=tcp&security=reality&pbk=xxx&sid=xx&sni=...&flow=xtls-rprx-vision#name
func vlessURI(n *ProxyNode) string {
	if n.UUID == "" || n.Host == "" || n.Port == 0 {
		return ""
	}
	q := url.Values{}
	if n.Network != "" {
		q.Set("type", n.Network)
	} else {
		q.Set("type", "tcp")
	}
	if n.Security != "" {
		q.Set("security", n.Security)
	}
	if n.Security == "reality" {
		if n.RealitySettings != nil {
			if v, _ := n.RealitySettings["publicKey"].(string); v != "" {
				q.Set("pbk", v)
			}
			if v, _ := n.RealitySettings["shortId"].(string); v != "" {
				q.Set("sid", v)
			}
			if v, _ := n.RealitySettings["sni"].(string); v != "" {
				q.Set("sni", v)
			}
		}
		q.Set("fp", "chrome")
		q.Set("flow", "xtls-rprx-vision")
	}
	if n.Security == "tls" {
		if sni := pickStreamHost(n); sni != "" {
			q.Set("sni", sni)
		}
		q.Set("fp", "chrome")
	}
	// 网络层补充
	switch n.Network {
	case "ws":
		if h, p := wsHostPath(n); h != "" || p != "" {
			if h != "" {
				q.Set("host", h)
			}
			if p != "" {
				q.Set("path", p)
			}
		}
	case "grpc":
		if sn := grpcServiceName(n); sn != "" {
			q.Set("serviceName", sn)
		}
	}
	tag := url.QueryEscape(n.Name)
	return fmt.Sprintf("vless://%s@%s:%d?%s#%s", n.UUID, n.Host, n.Port, q.Encode(), tag)
}

// trojan://password@host:port?type=tcp&security=tls&sni=...#name
func trojanURI(n *ProxyNode) string {
	if n.Password == "" || n.Host == "" {
		return ""
	}
	q := url.Values{}
	if n.Network != "" {
		q.Set("type", n.Network)
	}
	if n.Security == "" {
		q.Set("security", "tls")
	} else {
		q.Set("security", n.Security)
	}
	if sni := pickStreamHost(n); sni != "" {
		q.Set("sni", sni)
	}
	return fmt.Sprintf("trojan://%s@%s:%d?%s#%s",
		url.QueryEscape(n.Password), n.Host, n.Port, q.Encode(), url.QueryEscape(n.Name))
}

// --- Clash.Meta yaml ---

func renderClash(nodes []ProxyNode) string {
	var proxies []map[string]any
	var names []string
	for _, n := range nodes {
		p := nodeToClash(&n)
		if p == nil {
			continue
		}
		proxies = append(proxies, p)
		names = append(names, p["name"].(string))
	}
	root := map[string]any{
		"port":         7890,
		"socks-port":   7891,
		"mode":         "rule",
		"log-level":    "info",
		"proxies":      proxies,
		"proxy-groups": []map[string]any{
			{"name": "PROXY", "type": "select", "proxies": append([]string{"自动选择"}, names...)},
			{"name": "自动选择", "type": "url-test", "proxies": names,
				"url": "https://www.gstatic.com/generate_204", "interval": 300},
		},
		"rules": []string{"MATCH,PROXY"},
	}
	out, _ := yamlMarshal(root)
	return out
}

func nodeToClash(n *ProxyNode) map[string]any {
	switch n.Protocol {
	case "vmess":
		m := map[string]any{
			"name":   n.Name,
			"type":   "vmess",
			"server": n.Host,
			"port":   n.Port,
			"uuid":   n.UUID,
			"alterId": 0,
			"cipher":  "auto",
			"udp":     true,
			"network": valueOr(n.Network, "tcp"),
		}
		if n.Security == "tls" {
			m["tls"] = true
			m["servername"] = pickStreamHost(n)
		}
		if n.Network == "ws" {
			h, p := wsHostPath(n)
			m["ws-opts"] = map[string]any{"path": p, "headers": map[string]string{"Host": h}}
		}
		return m
	case "shadowsocks", "ss":
		method, _ := n.StreamSettings["method"].(string)
		if method == "" {
			method = "2022-blake3-aes-128-gcm"
		}
		return map[string]any{
			"name":     n.Name,
			"type":     "ss",
			"server":   n.Host,
			"port":     n.Port,
			"cipher":   method,
			"password": n.Password,
			"udp":      true,
		}
	case "hysteria2", "hy2":
		return map[string]any{
			"name":     n.Name,
			"type":     "hysteria2",
			"server":   n.Host,
			"port":     n.Port,
			"password": n.Password,
			"sni":      pickStreamHost(n),
		}
	case "tuic":
		return map[string]any{
			"name":               n.Name,
			"type":               "tuic",
			"server":             n.Host,
			"port":               n.Port,
			"uuid":               n.UUID,
			"password":           n.Password,
			"alpn":               []string{"h3"},
			"congestion-controller": "bbr",
			"sni":                pickStreamHost(n),
		}
	case "vless":
		m := map[string]any{
			"name":              n.Name,
			"type":              "vless",
			"server":            n.Host,
			"port":              n.Port,
			"uuid":              n.UUID,
			"udp":               true,
			"client-fingerprint": "chrome",
			"network":           valueOr(n.Network, "tcp"),
		}
		if n.Security == "reality" {
			m["tls"] = true
			m["servername"] = strFromMap(n.RealitySettings, "sni")
			m["flow"] = "xtls-rprx-vision"
			m["reality-opts"] = map[string]any{
				"public-key": strFromMap(n.RealitySettings, "publicKey"),
				"short-id":   strFromMap(n.RealitySettings, "shortId"),
			}
		} else if n.Security == "tls" {
			m["tls"] = true
			if sni := pickStreamHost(n); sni != "" {
				m["servername"] = sni
			}
		}
		return m
	case "trojan":
		m := map[string]any{
			"name":     n.Name,
			"type":     "trojan",
			"server":   n.Host,
			"port":     n.Port,
			"password": n.Password,
			"udp":      true,
		}
		if sni := pickStreamHost(n); sni != "" {
			m["sni"] = sni
		}
		return m
	}
	return nil
}

// --- sing-box JSON ---

func renderSingBox(nodes []ProxyNode) string {
	outbounds := []map[string]any{
		{"type": "selector", "tag": "PROXY", "outbounds": []string{"自动选择"}},
		{"type": "urltest", "tag": "自动选择", "outbounds": []string{}},
		{"type": "direct", "tag": "direct"},
		{"type": "block", "tag": "block"},
	}
	var names []string
	for _, n := range nodes {
		ob := nodeToSingBox(&n)
		if ob == nil {
			continue
		}
		outbounds = append(outbounds, ob)
		names = append(names, ob["tag"].(string))
	}
	// 把节点名字塞进 selector / urltest
	outbounds[0]["outbounds"] = append([]string{"自动选择"}, names...)
	outbounds[1]["outbounds"] = names

	root := map[string]any{
		"log":        map[string]any{"level": "info"},
		"outbounds":  outbounds,
		"route":      map[string]any{"final": "PROXY"},
	}
	out, _ := json.MarshalIndent(root, "", "  ")
	return string(out)
}

func nodeToSingBox(n *ProxyNode) map[string]any {
	switch n.Protocol {
	case "vmess":
		m := map[string]any{
			"type":        "vmess",
			"tag":         n.Name,
			"server":      n.Host,
			"server_port": n.Port,
			"uuid":        n.UUID,
			"security":    "auto",
			"alter_id":    0,
		}
		if n.Security == "tls" {
			m["tls"] = map[string]any{"enabled": true, "server_name": pickStreamHost(n)}
		}
		return m
	case "shadowsocks", "ss":
		method, _ := n.StreamSettings["method"].(string)
		if method == "" {
			method = "2022-blake3-aes-128-gcm"
		}
		return map[string]any{
			"type":        "shadowsocks",
			"tag":         n.Name,
			"server":      n.Host,
			"server_port": n.Port,
			"method":      method,
			"password":    n.Password,
		}
	case "hysteria2", "hy2":
		return map[string]any{
			"type":        "hysteria2",
			"tag":         n.Name,
			"server":      n.Host,
			"server_port": n.Port,
			"password":    n.Password,
			"tls":         map[string]any{"enabled": true, "server_name": pickStreamHost(n)},
		}
	case "tuic":
		return map[string]any{
			"type":              "tuic",
			"tag":               n.Name,
			"server":            n.Host,
			"server_port":       n.Port,
			"uuid":              n.UUID,
			"password":          n.Password,
			"congestion_control": "bbr",
			"tls":               map[string]any{"enabled": true, "alpn": []string{"h3"}, "server_name": pickStreamHost(n)},
		}
	case "vless":
		m := map[string]any{
			"type":        "vless",
			"tag":         n.Name,
			"server":      n.Host,
			"server_port": n.Port,
			"uuid":        n.UUID,
			"flow":        "",
		}
		if n.Security == "reality" {
			m["flow"] = "xtls-rprx-vision"
			m["tls"] = map[string]any{
				"enabled":     true,
				"server_name": strFromMap(n.RealitySettings, "sni"),
				"utls":        map[string]any{"enabled": true, "fingerprint": "chrome"},
				"reality": map[string]any{
					"enabled":    true,
					"public_key": strFromMap(n.RealitySettings, "publicKey"),
					"short_id":   strFromMap(n.RealitySettings, "shortId"),
				},
			}
		} else if n.Security == "tls" {
			m["tls"] = map[string]any{
				"enabled":     true,
				"server_name": pickStreamHost(n),
				"utls":        map[string]any{"enabled": true, "fingerprint": "chrome"},
			}
		}
		return m
	case "trojan":
		m := map[string]any{
			"type":        "trojan",
			"tag":         n.Name,
			"server":      n.Host,
			"server_port": n.Port,
			"password":    n.Password,
			"tls": map[string]any{
				"enabled":     true,
				"server_name": pickStreamHost(n),
			},
		}
		return m
	}
	return nil
}

// --- helpers ---

func pickStreamHost(n *ProxyNode) string {
	// 优先 TLSSettings.serverName，其次 stream.host
	if n.TLSSettings != nil {
		if v, _ := n.TLSSettings["serverName"].(string); v != "" {
			return v
		}
	}
	if n.StreamSettings != nil {
		if h, ok := n.StreamSettings["wsSettings"].(map[string]any); ok {
			if v, _ := h["host"].(string); v != "" {
				return v
			}
		}
	}
	return ""
}

func wsHostPath(n *ProxyNode) (string, string) {
	if n.StreamSettings == nil {
		return "", ""
	}
	if ws, ok := n.StreamSettings["wsSettings"].(map[string]any); ok {
		host, _ := ws["host"].(string)
		path, _ := ws["path"].(string)
		return host, path
	}
	return "", ""
}

func grpcServiceName(n *ProxyNode) string {
	if n.StreamSettings == nil {
		return ""
	}
	if g, ok := n.StreamSettings["grpcSettings"].(map[string]any); ok {
		s, _ := g["serviceName"].(string)
		return s
	}
	return ""
}

func strFromMap(m map[string]any, k string) string {
	if m == nil {
		return ""
	}
	v, _ := m[k].(string)
	return v
}

func valueOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
