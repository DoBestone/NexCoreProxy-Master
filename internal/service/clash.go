package service

import (
	"encoding/json"
	"fmt"

	"nexcoreproxy-master/internal/model"

	"gopkg.in/yaml.v3"
)

// GenerateClashSubscription 生成 Clash YAML 订阅
func (s *NodeService) GenerateClashSubscription(userID uint) (string, error) {
	var user model.User
	if err := model.GetDB().First(&user, userID).Error; err != nil {
		return "", err
	}
	if !user.Enable {
		return "", fmt.Errorf("用户已禁用")
	}

	var userNodes []model.UserNode
	model.GetDB().Where("user_id = ? AND enable = ?", userID, true).Find(&userNodes)

	nodes, _ := s.GetAll()

	var allRelayRules []model.RelayRule
	model.GetDB().Where("enable = ?", true).Preload("RelayNode").Find(&allRelayRules)
	relayRulesByBackend := make(map[uint][]model.RelayRule)
	for _, rule := range allRelayRules {
		relayRulesByBackend[rule.BackendNodeID] = append(relayRulesByBackend[rule.BackendNodeID], rule)
	}

	var proxies []map[string]interface{}
	seenNames := make(map[string]int)
	addProxy := func(p map[string]interface{}) {
		if p == nil {
			return
		}
		name, _ := p["name"].(string)
		if name == "" {
			return
		}
		if seenNames[name] > 0 {
			p["name"] = fmt.Sprintf("%s #%d", name, seenNames[name]+1)
		}
		seenNames[name]++
		proxies = append(proxies, p)
	}

	for _, userNode := range userNodes {
		for _, node := range nodes {
			if node.ID != userNode.NodeID {
				continue
			}
			if node.Type == "relay" {
				continue
			}
			inbounds, err := s.GetInbounds(node.ID)
			if err != nil {
				continue
			}

			isBackend := node.Type == "backend"

			for _, inbound := range inbounds {
				if isBackend {
					addProxy(s.buildClashProxy(node.IP, inbound, node.Name+" [直连]"))

					protocol, _ := inbound["protocol"].(string)
					for _, rule := range relayRulesByBackend[node.ID] {
						if rule.Protocol != protocol || rule.RelayInboundPort == 0 {
							continue
						}
						relayRemark := fmt.Sprintf("%s→%s [中转]", rule.RelayNode.Name, node.Name)
						synth := map[string]interface{}{
							"protocol":       protocol,
							"port":           float64(rule.RelayInboundPort),
							"remark":         relayRemark,
							"settings":       inbound["settings"],
							"streamSettings": inbound["streamSettings"],
						}
						addProxy(s.buildClashProxy(rule.RelayNode.IP, synth, relayRemark))
					}
				} else {
					remark, _ := inbound["remark"].(string)
					if remark == "" {
						remark = node.Name
					}
					addProxy(s.buildClashProxy(node.IP, inbound, remark))
				}
			}
		}
	}

	names := make([]string, 0, len(proxies))
	for _, p := range proxies {
		if n, ok := p["name"].(string); ok {
			names = append(names, n)
		}
	}

	selectProxies := append([]string{"auto", "DIRECT"}, names...)
	proxyGroups := []map[string]interface{}{
		{
			"name":    "PROXY",
			"type":    "select",
			"proxies": selectProxies,
		},
	}
	if len(names) > 0 {
		proxyGroups = append(proxyGroups, map[string]interface{}{
			"name":     "auto",
			"type":     "url-test",
			"url":      "http://www.gstatic.com/generate_204",
			"interval": 300,
			"proxies":  names,
		})
	}

	config := map[string]interface{}{
		"port":        7890,
		"socks-port":  7891,
		"allow-lan":   false,
		"mode":        "rule",
		"log-level":   "info",
		"proxies":     proxies,
		"proxy-groups": proxyGroups,
		"rules":       []string{"MATCH,PROXY"},
	}

	out, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// buildClashProxy 根据 inbound 生成单个 Clash proxy 配置
func (s *NodeService) buildClashProxy(host string, inbound map[string]interface{}, remark string) map[string]interface{} {
	protocol, _ := inbound["protocol"].(string)
	portVal, ok := inbound["port"].(float64)
	if !ok || portVal < 1 || portVal > 65535 {
		return nil
	}
	port := int(portVal)
	settings := getInboundField(inbound, "settings")
	streamSettings := getInboundField(inbound, "streamSettings")

	switch protocol {
	case "vmess":
		return buildClashVMess(host, port, remark, settings, streamSettings)
	case "vless":
		return buildClashVLESS(host, port, remark, settings, streamSettings)
	case "trojan":
		return buildClashTrojan(host, port, remark, settings, streamSettings)
	case "shadowsocks":
		return buildClashSS(host, port, remark, settings)
	}
	return nil
}

func buildClashVMess(host string, port int, remark, settings, streamSettings string) map[string]interface{} {
	var sett struct {
		Clients []struct {
			ID string `json:"id"`
		} `json:"clients"`
	}
	json.Unmarshal([]byte(settings), &sett)
	if len(sett.Clients) == 0 {
		return nil
	}
	sc := parseStreamSettings(streamSettings)

	p := map[string]interface{}{
		"name":     remark,
		"type":     "vmess",
		"server":   host,
		"port":     port,
		"uuid":     sett.Clients[0].ID,
		"alterId":  0,
		"cipher":   "auto",
		"udp":      true,
		"network":  sc.Network,
		"tls":      sc.Security == "tls",
	}
	applyTLSCommon(p, sc)
	applyNetworkOpts(p, sc)
	return p
}

func buildClashVLESS(host string, port int, remark, settings, streamSettings string) map[string]interface{} {
	var sett struct {
		Clients []struct {
			ID   string `json:"id"`
			Flow string `json:"flow"`
		} `json:"clients"`
	}
	json.Unmarshal([]byte(settings), &sett)
	if len(sett.Clients) == 0 {
		return nil
	}
	sc := parseStreamSettings(streamSettings)

	p := map[string]interface{}{
		"name":    remark,
		"type":    "vless",
		"server":  host,
		"port":    port,
		"uuid":    sett.Clients[0].ID,
		"udp":     true,
		"network": sc.Network,
		"tls":     sc.Security == "tls" || sc.Security == "reality",
	}
	if flow := sett.Clients[0].Flow; flow != "" {
		p["flow"] = flow
	}
	applyTLSCommon(p, sc)
	applyRealityOpts(p, sc)
	applyNetworkOpts(p, sc)
	return p
}

func buildClashTrojan(host string, port int, remark, settings, streamSettings string) map[string]interface{} {
	var sett struct {
		Clients []struct {
			Password string `json:"password"`
		} `json:"clients"`
	}
	json.Unmarshal([]byte(settings), &sett)
	if len(sett.Clients) == 0 {
		return nil
	}
	sc := parseStreamSettings(streamSettings)

	p := map[string]interface{}{
		"name":     remark,
		"type":     "trojan",
		"server":   host,
		"port":     port,
		"password": sett.Clients[0].Password,
		"udp":      true,
		"network":  sc.Network,
	}
	applyTLSCommon(p, sc)
	applyNetworkOpts(p, sc)
	return p
}

func buildClashSS(host string, port int, remark, settings string) map[string]interface{} {
	var sett struct {
		Method   string `json:"method"`
		Password string `json:"password"`
	}
	json.Unmarshal([]byte(settings), &sett)
	if sett.Password == "" {
		return nil
	}
	return map[string]interface{}{
		"name":     remark,
		"type":     "ss",
		"server":   host,
		"port":     port,
		"cipher":   sett.Method,
		"password": sett.Password,
		"udp":      true,
	}
}

// applyTLSCommon 填充 servername / skip-cert-verify / client-fingerprint
func applyTLSCommon(p map[string]interface{}, sc streamConfig) {
	if sc.Security == "tls" {
		if tls := sc.TlsSettings; tls != nil {
			if sni, ok := tls["serverName"].(string); ok && sni != "" {
				p["servername"] = sni
			}
			if fp, ok := tls["fingerprint"].(string); ok && fp != "" {
				p["client-fingerprint"] = fp
			}
		}
		p["skip-cert-verify"] = false
	}
}

// applyRealityOpts 填充 reality-opts（仅 VLESS 适用）
func applyRealityOpts(p map[string]interface{}, sc streamConfig) {
	if sc.Security != "reality" || sc.RealitySettings == nil {
		return
	}
	r := sc.RealitySettings
	opts := map[string]interface{}{}
	if pk, ok := r["publicKey"].(string); ok {
		opts["public-key"] = pk
	}
	if sids, ok := r["shortIds"].([]interface{}); ok && len(sids) > 0 {
		if sid, ok := sids[0].(string); ok {
			opts["short-id"] = sid
		}
	}
	p["reality-opts"] = opts
	if sns, ok := r["serverNames"].([]interface{}); ok && len(sns) > 0 {
		if sn, ok := sns[0].(string); ok {
			p["servername"] = sn
		}
	}
	if fp, ok := r["fingerprint"].(string); ok && fp != "" {
		p["client-fingerprint"] = fp
	}
}

// applyNetworkOpts 填充 ws-opts / grpc-opts
func applyNetworkOpts(p map[string]interface{}, sc streamConfig) {
	switch sc.Network {
	case "ws":
		if sc.WsSettings == nil {
			return
		}
		opts := map[string]interface{}{}
		if path, ok := sc.WsSettings["path"].(string); ok && path != "" {
			opts["path"] = path
		}
		if headers, ok := sc.WsSettings["headers"].(map[string]interface{}); ok {
			if host, ok := headers["Host"].(string); ok && host != "" {
				opts["headers"] = map[string]interface{}{"Host": host}
			}
		}
		if len(opts) > 0 {
			p["ws-opts"] = opts
		}
	case "grpc":
		if sc.GrpcSettings == nil {
			return
		}
		if sn, ok := sc.GrpcSettings["serviceName"].(string); ok && sn != "" {
			p["grpc-opts"] = map[string]interface{}{"grpc-service-name": sn}
		}
	}
}
