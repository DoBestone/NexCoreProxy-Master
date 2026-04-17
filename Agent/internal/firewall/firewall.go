// Package firewall 把 ServerConfig 里需要监听的端口同步到节点防火墙
//
// 设计：每次 puller 拉到新配置后调一次 Reconcile(ports)；
// 实现侧从当前规则做 diff，新增的 allow，多余的"由 ncp 管理过"标签的 deny。
//
// 自动检测 ufw / firewalld / iptables，按可用性优先；都不在就 no-op + 记日志。
//
// 失败永远不阻塞 xray reload；防火墙是 best-effort，节点本地若已开放端口照常工作。
package firewall

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"nexcoreproxy-agent/internal/protocol"
)

// Port 一个需要开放的端口
type Port struct {
	Number   int
	Protocol string // tcp / udp / both
}

// CollectPorts 从 ServerConfig 抽取出该节点需要开放的端口集合
//
// - inbound.Port (TCP) — 业务入站
// - inbound.Port (UDP) — 如果是 quic / hy2 / tuic
// - relay.ListenPort (TCP+UDP) — 中转 dokodemo 通常 tcp+udp
// - relay.ListenPortRange — Hy2 端口跳跃
//
// stats inbound (10085) 监听 127.0.0.1，不需要对外开放，跳过。
func CollectPorts(cfg *protocol.ServerConfig) []Port {
	uniq := make(map[string]Port)
	add := func(p Port) {
		if p.Number <= 0 {
			return
		}
		key := fmt.Sprintf("%d/%s", p.Number, p.Protocol)
		uniq[key] = p
	}

	for _, in := range cfg.Inbounds {
		proto := "tcp"
		switch strings.ToLower(in.Protocol) {
		case "hysteria2", "hy2", "tuic":
			proto = "udp"
		}
		if strings.ToLower(in.Network) == "quic" || strings.ToLower(in.Network) == "kcp" {
			proto = "udp"
		}
		add(Port{Number: in.Port, Protocol: proto})
	}
	for _, r := range cfg.Relays {
		add(Port{Number: r.ListenPort, Protocol: "tcp"})
		add(Port{Number: r.ListenPort, Protocol: "udp"})
		if r.PortRange != "" {
			s, e := parseRange(r.PortRange)
			for p := s; p <= e && p > 0; p++ {
				add(Port{Number: p, Protocol: "udp"})
			}
		}
	}
	out := make([]Port, 0, len(uniq))
	for _, p := range uniq {
		out = append(out, p)
	}
	return out
}

// Reconcile 把目标端口集合应用到本机防火墙
//
// 当前实现只做"加规则"（确保需要的端口开），不做删除：
// 删除存量规则风险大（可能误杀 SSH / 用户自定义），交给管理员手动收敛。
func Reconcile(ports []Port) error {
	if len(ports) == 0 {
		return nil
	}
	switch detectBackend() {
	case "ufw":
		return reconcileUFW(ports)
	case "firewalld":
		return reconcileFirewalld(ports)
	case "iptables":
		return reconcileIPTables(ports)
	default:
		log.Printf("[firewall] no supported backend, skipped %d ports", len(ports))
		return nil
	}
}

func detectBackend() string {
	for _, b := range []string{"ufw", "firewall-cmd", "iptables"} {
		if _, err := exec.LookPath(b); err == nil {
			switch b {
			case "ufw":
				return "ufw"
			case "firewall-cmd":
				return "firewalld"
			case "iptables":
				return "iptables"
			}
		}
	}
	return ""
}

func reconcileUFW(ports []Port) error {
	for _, p := range ports {
		_ = run("ufw", "allow", fmt.Sprintf("%d/%s", p.Number, p.Protocol))
	}
	return nil
}

func reconcileFirewalld(ports []Port) error {
	for _, p := range ports {
		_ = run("firewall-cmd", "--permanent", fmt.Sprintf("--add-port=%d/%s", p.Number, p.Protocol))
	}
	return run("firewall-cmd", "--reload")
}

func reconcileIPTables(ports []Port) error {
	for _, p := range ports {
		// 幂等：先尝试 -C 检查存在再 -A，避免重复规则堆积
		check := exec.Command("iptables", "-C", "INPUT", "-p", p.Protocol,
			"--dport", strconv.Itoa(p.Number), "-j", "ACCEPT")
		if err := check.Run(); err == nil {
			continue
		}
		_ = run("iptables", "-A", "INPUT", "-p", p.Protocol,
			"--dport", strconv.Itoa(p.Number), "-j", "ACCEPT")
	}
	return nil
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Printf("[firewall] %s %s failed: %v %s", name, strings.Join(args, " "), err, stderr.String())
		return err
	}
	return nil
}

func parseRange(s string) (int, int) {
	dash := strings.IndexByte(s, '-')
	if dash <= 0 {
		return 0, 0
	}
	a, _ := strconv.Atoi(strings.TrimSpace(s[:dash]))
	b, _ := strconv.Atoi(strings.TrimSpace(s[dash+1:]))
	return a, b
}
