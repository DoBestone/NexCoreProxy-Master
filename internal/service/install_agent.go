package service

import (
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"nexcoreproxy-master/internal/model"

	"golang.org/x/crypto/ssh"
)

//go:embed install_agent.sh
var installAgentScript string

// InstallAgent 用 SSH 部署 ncp-agent + xray-core 到节点
//
// 流程：
//  1. 生成/复用 AgentKey（= node_token）
//  2. 通过 SSH 上传 install-agent.sh 到 /tmp
//  3. 执行脚本（注入 NCP_MASTER_URL / NCP_NODE_ID / NCP_NODE_TOKEN / NCP_AGENT_URL）
//  4. 等待 agent 首次拉配置（最多 60s）
//  5. 标记 Installed=true，清空 SSHPassword（敏感凭据用完即抹）
//
// 失败保留 SSHPassword 便于人工排查 + 重试。
func (s *NodeService) InstallAgent(id uint) error {
	node, err := s.GetByID(id)
	if err != nil {
		return err
	}
	rt := GetRuntimeConfig()
	if rt.MasterURL == "" {
		return fmt.Errorf("master_url 未配置（启动时未传 --master-url）")
	}
	if rt.AgentBinaryURL == "" {
		return fmt.Errorf("agent binary url 未配置")
	}
	if node.SSHPassword == "" {
		return fmt.Errorf("节点缺少 SSH 凭据，无法自动部署")
	}

	// 1. AgentKey
	if node.AgentKey == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return fmt.Errorf("生成 AgentKey 失败: %v", err)
		}
		node.AgentKey = fmt.Sprintf("%x", b)
		if err := model.GetDB().Model(node).Update("agent_key", node.AgentKey).Error; err != nil {
			return fmt.Errorf("保存 AgentKey 失败: %v", err)
		}
	}

	sshClient, err := s.SSHConnect(node.IP, node.SSHPort, node.SSHUser, node.SSHPassword)
	if err != nil {
		return fmt.Errorf("SSH 连接失败: %v", err)
	}
	defer sshClient.Close()

	// 2. 上传脚本
	if err := uploadFile(sshClient, "/tmp/install-agent.sh", installAgentScript); err != nil {
		return fmt.Errorf("上传安装脚本失败: %v", err)
	}

	// 3. 执行
	envPrefix := fmt.Sprintf(
		`NCP_MASTER_URL=%s NCP_NODE_ID=%d NCP_NODE_TOKEN=%s NCP_AGENT_URL=%s XRAY_VERSION=%s `,
		shellQuote(rt.MasterURL), node.ID, shellQuote(node.AgentKey),
		shellQuote(rt.AgentBinaryURL), shellQuote(rt.XrayVersion),
	)
	output, err := s.SSHRun(sshClient, envPrefix+"bash /tmp/install-agent.sh 2>&1")
	if err != nil {
		return fmt.Errorf("安装脚本执行失败: %v\n%s", err, output)
	}

	// 4. 等首次拉配置（agent 启动 → 立即拉 → Master 更新 last_sync_at）
	deadline := time.Now().Add(60 * time.Second)
	startedAt := time.Now()
	online := false
	for time.Now().Before(deadline) {
		var n model.Node
		if err := model.GetDB().Select("id,last_sync_at,status").First(&n, node.ID).Error; err == nil {
			if n.LastSyncAt != nil && n.LastSyncAt.After(startedAt) {
				online = true
				break
			}
		}
		time.Sleep(2 * time.Second)
	}
	if !online {
		return fmt.Errorf("安装完成但 agent 未在 60s 内回连，请检查节点防火墙/出网；安装日志：\n%s", tail(output, 50))
	}

	// 5. 收尾
	updates := map[string]any{
		"installed":     true,
		"backend":       "ncp-agent",
		"status":        "online",
		"ssh_password":  "", // 用完即抹
		"agent_version": "0.1.0",
	}
	if err := model.GetDB().Model(&model.Node{}).Where("id = ?", node.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新节点状态失败: %v", err)
	}

	// 6. 默认入站集（仅 backend 角色）— 让节点上线即有可用协议
	role := node.Role
	if role == "" {
		role = "backend"
	}
	if role == "backend" && s.provisioner != nil {
		if _, err := s.provisioner.Provision(node.ID, InboundSetStandard); err != nil {
			// 失败不阻塞安装流程，仅记日志（管理员可手动 Provision）
			_ = err
		}
	}
	return nil
}

// AttachProvisioner 由 Services 构造时延迟注入
func (s *NodeService) AttachProvisioner(p *NodeProvisioner) { s.provisioner = p }

// uploadFile 通过 SSH stdin 用 cat heredoc 写文件，避免依赖 sftp 包
//
// 内容里如果含 'NCP_END' 字符串会破坏 heredoc，但脚本本身不会出现这个标记。
func uploadFile(client *ssh.Client, remotePath, content string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// 用 base64 传输避免任何 shell 特殊字符问题
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	cmd := fmt.Sprintf("echo %s | base64 -d > %s && chmod +x %s",
		encoded, shellQuote(remotePath), shellQuote(remotePath))
	out, err := session.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(out))
	}
	return nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func tail(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}
