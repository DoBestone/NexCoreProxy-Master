// Package upgrade 处理 agent / xray 自动升级的版本对账
//
// Phase 1 实现"检测 + 上报"，自动下载替换留 Phase 2：
//   - puller 拉到 NodeMeta.AgentTarget / XrayTarget 与本地版本不一致时记日志 + 给 push 打标
//   - 真实滚动升级流程（下载 → sha256 校验 → systemd 重启 → 失败回滚）需要二进制 release 渠道就位
//
// 之所以拆出独立包：将来加版本下载 + GPG 校验 + 滚动策略时不污染 puller。
package upgrade

import (
	"log"

	"nexcoreproxy-agent/internal/protocol"
)

const CurrentAgentVersion = "0.1.0"

// Detect 比对 push 响应里期望的版本与本地版本
//
// 返回需要升级的组件名（agent / xray）；空列表表示无需升级。
func Detect(meta protocol.NodeMeta, currentXray string) []string {
	var needs []string
	if meta.AgentTarget != "" && meta.AgentTarget != CurrentAgentVersion {
		needs = append(needs, "agent")
		log.Printf("[upgrade] agent version mismatch: current=%s target=%s",
			CurrentAgentVersion, meta.AgentTarget)
	}
	if meta.XrayTarget != "" && currentXray != "" && meta.XrayTarget != currentXray {
		needs = append(needs, "xray")
		log.Printf("[upgrade] xray version mismatch: current=%s target=%s",
			currentXray, meta.XrayTarget)
	}
	return needs
}
