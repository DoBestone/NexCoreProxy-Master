package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"nexcoreproxy-master/internal/model"
)

// NodeService 节点服务
type NodeService struct {
	clients  sync.Map
	agentAPI *AgentAPIClient
}

// NewNodeService 创建节点服务
func NewNodeService() *NodeService {
	return &NodeService{
		agentAPI: NewAgentAPIClient(),
	}
}

// NodeClient 节点 API 客户端
type NodeClient struct {
	NodeID   uint
	IP       string
	Port     int
	Username string
	Password string
	Token    string
	ExpireAt time.Time
}

// GetAll 获取所有节点
func (s *NodeService) GetAll() ([]model.Node, error) {
	var nodes []model.Node
	err := model.GetDB().Where("enable = ?", true).Find(&nodes).Error
	return nodes, err
}

// GetByID 根据ID获取节点
func (s *NodeService) GetByID(id uint) (*model.Node, error) {
	var node model.Node
	err := model.GetDB().First(&node, id).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// Create 创建节点
func (s *NodeService) Create(node *model.Node) error {
	node.Status = "unknown"
	return model.GetDB().Create(node).Error
}

// Update 更新节点
func (s *NodeService) Update(node *model.Node) error {
	return model.GetDB().Save(node).Error
}

// Delete 删除节点
func (s *NodeService) Delete(id uint) error {
	s.clients.Delete(id)
	return model.GetDB().Delete(&model.Node{}, id).Error
}

// Install SSH自动安装x-ui
func (s *NodeService) Install(id uint) error {
	node, err := s.GetByID(id)
	if err != nil {
		return err
	}

	sshClient, err := s.sshConnect(node.IP, node.SSHPort, node.SSHUser, node.SSHPassword)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %v", err)
	}
	defer sshClient.Close()

	// 生成随机端口（10000-65000）
	port := 10000 + rand.Intn(55000)

	// 生成随机用户名（8位）
	user := generateRandomString(8)

	// 生成随机密码（16位）
	password := generateRandomString(16)

	// 使用 NexCoreProxy Agent 安装脚本
	installCmd := fmt.Sprintf(
		`bash <(curl -Ls https://raw.githubusercontent.com/DoBestone/NexCoreProxy-Agent/main/install.sh) -p %d -u %s -pass '%s'`,
		port, user, password,
	)

	output, err := s.sshRun(sshClient, installCmd)
	if err != nil {
		return fmt.Errorf("安装失败: %v, 输出: %s", err, output)
	}

	// 等待服务启动
	time.Sleep(5 * time.Second)

	// 获取 API Token (直接从文件读取，只取有效的 32 字符 token)
	apiTokenOutput, _ := s.SSHRun(sshClient, "cat /usr/local/x-ui/API_TOKEN 2>/dev/null")
	// 提取有效的 token（32 字符的字母数字字符串）
	lines := strings.Split(apiTokenOutput, "\n")
	var apiToken string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 匹配 32 字符的字母数字 token
		if len(line) == 32 {
			matched, _ := regexp.MatchString("^[a-zA-Z0-9]{32}$", line)
			if matched {
				apiToken = line
				break
			}
		}
	}

	// 更新节点信息
	updates := map[string]interface{}{
		"status":   "online",
		"port":     port,
		"username": user,
		"password": password,
	}
	if apiToken != "" {
		updates["api_token"] = apiToken
	}

	err = model.GetDB().Model(&model.Node{}).Where("id = ?", node.ID).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("更新节点信息失败: %v", err)
	}

	return nil
}

// generateRandomString 生成随机字符串
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// SSHConnect SSH连接 (公开方法)
func (s *NodeService) SSHConnect(host string, port int, user, password string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	return ssh.Dial("tcp", addr, config)
}

// SSHRun 执行SSH命令 (公开方法)
func (s *NodeService) SSHRun(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	return string(output), err
}

// 保留私有方法供内部使用
func (s *NodeService) sshConnect(host string, port int, user, password string) (*ssh.Client, error) {
	return s.SSHConnect(host, port, user, password)
}

func (s *NodeService) sshRun(client *ssh.Client, cmd string) (string, error) {
	return s.SSHRun(client, cmd)
}

// TestConnection 测试节点连接
func (s *NodeService) TestConnection(id uint) error {
	node, err := s.GetByID(id)
	if err != nil {
		return err
	}

	// 简单的 TCP 连接测试，不依赖面板凭据
	addr := fmt.Sprintf("%s:%d", node.IP, node.Port)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}
	conn.Close()
	return nil
}

// SyncStatus 同步节点状态 (优先 HTTP API，失败回退 SSH)
func (s *NodeService) SyncStatus(id uint) (*NodeStatus, error) {
	node, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// 优先尝试 HTTP API
	if node.APIToken != "" {
		status, err := s.syncStatusViaAPI(node)
		if err == nil {
			return status, nil
		}
		// API 失败，尝试刷新 Token
		newToken, tokenErr := s.refreshAPIToken(node)
		if tokenErr == nil && newToken != "" {
			node.APIToken = newToken
			status, err = s.syncStatusViaAPI(node)
			if err == nil {
				return status, nil
			}
		}
	}

	// 回退到 SSH
	return s.syncStatusViaSSH(node)
}

// syncStatusViaAPI 通过 HTTP API 同步状态
func (s *NodeService) syncStatusViaAPI(node *model.Node) (*NodeStatus, error) {
	apiStatus, err := s.agentAPI.GetStatus(node.IP, node.APIPort, node.APIToken)
	if err != nil {
		return nil, fmt.Errorf("API 调用失败: %v", err)
	}

	if !apiStatus.Success {
		return nil, fmt.Errorf("API 返回失败")
	}

	status := &NodeStatus{
		XrayVersion:   apiStatus.Data.XrayVersion,
		XrayState:     "running",
		CPU:           apiStatus.Data.CPU,
		Memory:        apiStatus.Data.Memory,
		Disk:          apiStatus.Data.Disk,
		Uptime:        apiStatus.Data.Uptime,
		UploadTotal:   apiStatus.Data.UploadTotal,
		DownloadTotal: apiStatus.Data.DownloadTotal,
	}

	// 更新数据库
	now := time.Now()
	model.GetDB().Model(&model.Node{}).Where("id = ?", node.ID).Updates(map[string]interface{}{
		"status":         "online",
		"cpu":            status.CPU,
		"memory":         status.Memory,
		"disk":           status.Disk,
		"uptime":         status.Uptime,
		"xray_version":   status.XrayVersion,
		"upload_total":   status.UploadTotal,
		"download_total": status.DownloadTotal,
		"last_sync_at":   now,
	})

	return status, nil
}

// syncStatusViaSSH 通过 SSH 同步状态（回退方案）
func (s *NodeService) syncStatusViaSSH(node *model.Node) (*NodeStatus, error) {
	sshClient, err := s.SSHConnect(node.IP, node.SSHPort, node.SSHUser, node.SSHPassword)
	if err != nil {
		model.GetDB().Model(&model.Node{}).Where("id = ?", node.ID).Updates(map[string]interface{}{
			"status":      "offline",
			"last_sync_at": time.Now(),
		})
		return nil, fmt.Errorf("SSH连接失败: %v", err)
	}
	defer sshClient.Close()

	// 通过 ncp-agent 获取状态
	output, err := s.SSHRun(sshClient, "ncp-agent info 2>/dev/null || echo 'error'")
	if err != nil || strings.Contains(output, "error") {
		model.GetDB().Model(&model.Node{}).Where("id = ?", node.ID).Updates(map[string]interface{}{
			"status":      "offline",
			"last_sync_at": time.Now(),
		})
		return nil, fmt.Errorf("获取状态失败")
	}

	status := &NodeStatus{XrayState: "running"}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "版本:") {
			status.XrayVersion = strings.TrimSpace(strings.TrimPrefix(line, "版本:"))
		}
	}

	// 获取系统资源
	cpuOutput, _ := s.SSHRun(sshClient, "top -bn1 | grep 'Cpu(s)' | awk '{print $2}' | cut -d'%' -f1 || echo 0")
	status.CPU, _ = strconv.ParseFloat(strings.TrimSpace(cpuOutput), 64)

	memOutput, _ := s.SSHRun(sshClient, "free | grep Mem | awk '{printf \"%.1f\", $3/$2 * 100}'")
	status.Memory, _ = strconv.ParseFloat(strings.TrimSpace(memOutput), 64)

	diskOutput, _ := s.SSHRun(sshClient, "df -h / | tail -1 | awk '{print $5}' | cut -d'%' -f1")
	status.Disk, _ = strconv.ParseFloat(strings.TrimSpace(diskOutput), 64)

	uptimeOutput, _ := s.SSHRun(sshClient, "cat /proc/uptime | awk '{print int($1)}'")
	uptime, _ := strconv.ParseInt(strings.TrimSpace(uptimeOutput), 10, 64)
	status.Uptime = uint64(uptime)

	// 更新数据库
	model.GetDB().Model(&model.Node{}).Where("id = ?", node.ID).Updates(map[string]interface{}{
		"status":       "online",
		"cpu":          status.CPU,
		"memory":       status.Memory,
		"disk":         status.Disk,
		"uptime":       uptime,
		"xray_version": status.XrayVersion,
		"last_sync_at": time.Now(),
	})

	return status, nil
}

// refreshAPIToken 刷新 API Token
func (s *NodeService) refreshAPIToken(node *model.Node) (string, error) {
	sshClient, err := s.SSHConnect(node.IP, node.SSHPort, node.SSHUser, node.SSHPassword)
	if err != nil {
		return "", err
	}
	defer sshClient.Close()

	output, err := s.SSHRun(sshClient, "ncp-agent get-token 2>/dev/null || cat /usr/local/x-ui/API_TOKEN 2>/dev/null")
	if err != nil {
		return "", err
	}

	token := strings.TrimSpace(output)
	if token == "" {
		return "", fmt.Errorf("Token 未设置")
	}

	// 保存到数据库
	model.GetDB().Model(&model.Node{}).Where("id = ?", node.ID).Update("api_token", token)

	return token, nil
}

// SyncAll 同步所有节点状态
func (s *NodeService) SyncAll() {
	nodes, _ := s.GetAll()
	var wg sync.WaitGroup
	for _, node := range nodes {
		if !node.Enable {
			continue
		}
		wg.Add(1)
		go func(n model.Node) {
			defer wg.Done()
			s.SyncStatus(n.ID)
		}(node)
	}
	wg.Wait()
}

// GetInbounds 获取节点入站列表 (通过SSH)
func (s *NodeService) GetInbounds(nodeID uint) ([]map[string]interface{}, error) {
	node, err := s.GetByID(nodeID)
	if err != nil {
		return nil, err
	}

	// 优先使用 HTTP API
	if node.APIToken != "" {
		inbounds, err := s.agentAPI.GetInbounds(node.IP, node.APIPort, node.APIToken)
		if err == nil && inbounds.Success {
			result := make([]map[string]interface{}, len(inbounds.Data))
			for i, inbound := range inbounds.Data {
				result[i] = map[string]interface{}{
					"id":          inbound.ID,
					"remark":      inbound.Remark,
					"port":        inbound.Port,
					"protocol":    inbound.Protocol,
					"enable":      inbound.Enable,
					"tag":         inbound.Tag,
					"up":          inbound.TotalUp,
					"down":        inbound.TotalDown,
					"totalClient": inbound.TotalClient,
				}
			}
			return result, nil
		}
	}

	// 回退到 SSH
	sshClient, err := s.SSHConnect(node.IP, node.SSHPort, node.SSHUser, node.SSHPassword)
	if err != nil {
		return nil, fmt.Errorf("SSH连接失败: %v", err)
	}
	defer sshClient.Close()

	output, err := s.SSHRun(sshClient, "ncp-agent list-inbounds")
	if err != nil {
		return nil, err
	}

	return []map[string]interface{}{{"output": output}}, nil
}

// DeleteInbound 删除入站 (通过SSH)
func (s *NodeService) DeleteInbound(nodeID uint, inboundID int) error {
	node, err := s.GetByID(nodeID)
	if err != nil {
		return err
	}

	sshClient, err := s.SSHConnect(node.IP, node.SSHPort, node.SSHUser, node.SSHPassword)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %v", err)
	}
	defer sshClient.Close()

	// 使用 ncp-agent 删除入站
	_, err = s.SSHRun(sshClient, fmt.Sprintf("ncp-agent del-inbound %d", inboundID))
	return err
}

// AddInbound 添加入站 (通过面板API)
func (s *NodeService) AddInbound(nodeID uint, inbound map[string]interface{}) error {
	client, err := s.getClient(nodeID)
	if err != nil {
		return err
	}
	return client.AddInbound(inbound)
}

// ========== API Token 管理 ==========

// GetAPIToken 通过 SSH 获取节点的 API Token
func (s *NodeService) GetAPIToken(nodeID uint) (string, error) {
	node, err := s.GetByID(nodeID)
	if err != nil {
		return "", err
	}

	sshClient, err := s.SSHConnect(node.IP, node.SSHPort, node.SSHUser, node.SSHPassword)
	if err != nil {
		return "", fmt.Errorf("SSH连接失败: %v", err)
	}
	defer sshClient.Close()

	output, err := s.SSHRun(sshClient, "ncp-agent get-token 2>/dev/null || cat /usr/local/x-ui/API_TOKEN 2>/dev/null")
	if err != nil {
		return "", fmt.Errorf("获取 Token 失败: %v", err)
	}

	token := strings.TrimSpace(output)
	if token == "" {
		return "", fmt.Errorf("API Token 未设置")
	}

	// 保存到数据库
	model.GetDB().Model(&model.Node{}).Where("id = ?", nodeID).Update("api_token", token)

	return token, nil
}

// GenAPIToken 生成新的 API Token
func (s *NodeService) GenAPIToken(nodeID uint) (string, error) {
	node, err := s.GetByID(nodeID)
	if err != nil {
		return "", err
	}

	sshClient, err := s.SSHConnect(node.IP, node.SSHPort, node.SSHUser, node.SSHPassword)
	if err != nil {
		return "", fmt.Errorf("SSH连接失败: %v", err)
	}
	defer sshClient.Close()

	output, err := s.SSHRun(sshClient, "ncp-agent gen-token")
	if err != nil {
		return "", fmt.Errorf("生成 Token 失败: %v", err)
	}

	// 提取 Token
	var token string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "API Token 已生成:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				token = strings.TrimSpace(parts[1])
				break
			}
		}
	}

	if token == "" {
		return "", fmt.Errorf("生成失败")
	}

	// 保存到数据库
	model.GetDB().Model(&model.Node{}).Where("id = ?", nodeID).Update("api_token", token)

	return token, nil
}

// ========== SSH 直连控制方法 ==========

// SSHGetStatus 通过SSH获取节点完整状态
func (s *NodeService) SSHGetStatus(nodeID uint) (map[string]interface{}, error) {
	node, err := s.GetByID(nodeID)
	if err != nil {
		return nil, err
	}

	sshClient, err := s.SSHConnect(node.IP, node.SSHPort, node.SSHUser, node.SSHPassword)
	if err != nil {
		return nil, fmt.Errorf("SSH连接失败: %v", err)
	}
	defer sshClient.Close()

	// 获取完整系统信息
	info := make(map[string]interface{})

	// 服务状态
	serviceStatus, _ := s.SSHRun(sshClient, "systemctl is-active x-ui 2>/dev/null || echo 'inactive'")
	info["service_status"] = strings.TrimSpace(serviceStatus)

	// 版本
	version, _ := s.SSHRun(sshClient, "cat /usr/local/x-ui/VERSION 2>/dev/null || echo 'unknown'")
	info["version"] = strings.TrimSpace(version)

	// 面板端口
	port, _ := s.SSHRun(sshClient, "ncp-agent get-port 2>/dev/null || echo '54321'")
	info["panel_port"] = strings.TrimSpace(port)

	// 管理员用户名
	user, _ := s.SSHRun(sshClient, "ncp-agent get-user 2>/dev/null || echo 'admin'")
	info["admin_user"] = strings.TrimSpace(user)

	// CPU 使用率
	cpu, _ := s.SSHRun(sshClient, "top -bn1 | grep 'Cpu(s)' | awk '{print $2}' | cut -d'%' -f1 || echo 0")
	info["cpu"], _ = strconv.ParseFloat(strings.TrimSpace(cpu), 64)

	// 内存使用率
	mem, _ := s.SSHRun(sshClient, "free | grep Mem | awk '{printf \"%.1f\", $3/$2 * 100}'")
	info["memory"], _ = strconv.ParseFloat(strings.TrimSpace(mem), 64)

	// 磁盘使用率
	disk, _ := s.SSHRun(sshClient, "df -h / | tail -1 | awk '{print $5}' | cut -d'%' -f1")
	info["disk"], _ = strconv.ParseFloat(strings.TrimSpace(disk), 64)

	// 运行时间（秒）
	uptime, _ := s.SSHRun(sshClient, "cat /proc/uptime | awk '{print int($1)}'")
	uptimeSec, _ := strconv.ParseInt(strings.TrimSpace(uptime), 10, 64)
	info["uptime"] = uptimeSec
	info["uptime_human"] = formatUptime(uptimeSec)

	// Xray 状态
	xrayPid, _ := s.SSHRun(sshClient, "pgrep -f xray-linux || echo ''")
	if strings.TrimSpace(xrayPid) != "" {
		info["xray_status"] = "running"
		// Xray 版本
		arch, _ := s.SSHRun(sshClient, "uname -m")
		archStr := strings.TrimSpace(arch)
		if strings.Contains(archStr, "x86_64") {
			archStr = "amd64"
		} else if strings.Contains(archStr, "aarch64") {
			archStr = "arm64"
		}
		xrayVer, _ := s.SSHRun(sshClient, fmt.Sprintf("/usr/local/x-ui/bin/xray-linux-%s version 2>/dev/null | head -1 | awk '{print $2}'", archStr))
		info["xray_version"] = strings.TrimSpace(xrayVer)
	} else {
		info["xray_status"] = "stopped"
		info["xray_version"] = ""
	}

	// 入站数量
	inboundCount, _ := s.SSHRun(sshClient, "sqlite3 /usr/local/x-ui/db/x-ui.db 'SELECT COUNT(*) FROM inbounds;' 2>/dev/null || echo 0")
	info["inbound_count"], _ = strconv.Atoi(strings.TrimSpace(inboundCount))

	// 总流量
	traffic, _ := s.SSHRun(sshClient, "sqlite3 /usr/local/x-ui/db/x-ui.db 'SELECT SUM(up)+SUM(down) FROM inbounds;' 2>/dev/null || echo 0")
	info["total_traffic"], _ = strconv.ParseInt(strings.TrimSpace(traffic), 10, 64)

	// 在线用户数
	onlineUsers, _ := s.SSHRun(sshClient, "sqlite3 /usr/local/x-ui/db/x-ui.db 'SELECT COUNT(*) FROM inbounds WHERE enable=1;' 2>/dev/null || echo 0")
	info["online_inbounds"], _ = strconv.Atoi(strings.TrimSpace(onlineUsers))

	return info, nil
}

// formatUptime 格式化运行时间
func formatUptime(seconds int64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	mins := (seconds % 3600) / 60
	if days > 0 {
		return fmt.Sprintf("%d天%d小时%d分钟", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%d小时%d分钟", hours, mins)
	}
	return fmt.Sprintf("%d分钟", mins)
}

// SSHRestartXray 通过SSH重启Xray
func (s *NodeService) SSHRestartXray(nodeID uint) error {
	node, err := s.GetByID(nodeID)
	if err != nil {
		return err
	}

	sshClient, err := s.SSHConnect(node.IP, node.SSHPort, node.SSHUser, node.SSHPassword)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %v", err)
	}
	defer sshClient.Close()

	_, err = s.SSHRun(sshClient, "ncp-agent restart-xray")
	return err
}

// SSHSetPort 通过SSH设置面板端口
func (s *NodeService) SSHSetPort(nodeID uint, port int) error {
	node, err := s.GetByID(nodeID)
	if err != nil {
		return err
	}

	sshClient, err := s.SSHConnect(node.IP, node.SSHPort, node.SSHUser, node.SSHPassword)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %v", err)
	}
	defer sshClient.Close()

	_, err = s.SSHRun(sshClient, fmt.Sprintf("ncp-agent set-port %d", port))
	if err != nil {
		return err
	}

	// 更新数据库
	model.GetDB().Model(&model.Node{}).Where("id = ?", nodeID).Update("port", port)
	return nil
}

// SSHEnableInbound 通过SSH启用入站
func (s *NodeService) SSHEnableInbound(nodeID uint, inboundID int, enable bool) error {
	node, err := s.GetByID(nodeID)
	if err != nil {
		return err
	}

	sshClient, err := s.SSHConnect(node.IP, node.SSHPort, node.SSHUser, node.SSHPassword)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %v", err)
	}
	defer sshClient.Close()

	cmd := fmt.Sprintf("ncp-agent enable-inbound %d", inboundID)
	if !enable {
		cmd = fmt.Sprintf("ncp-agent disable-inbound %d", inboundID)
	}

	_, err = s.SSHRun(sshClient, cmd)
	return err
}

// RestartXray 重启节点 Xray
// RestartXray 重启节点的 xray 服务 (通过 SSH)
func (s *NodeService) RestartXray(nodeID uint) error {
	node, err := s.GetByID(nodeID)
	if err != nil {
		return err
	}

	// 优先使用 SSH 方式重启
	if node.SSHPassword != "" {
		return s.restartXrayViaSSH(node)
	}

	// 备用：通过 x-ui API 重启
	client, err := s.getClient(nodeID)
	if err != nil {
		return err
	}
	return client.RestartXray()
}

// restartXrayViaSSH 通过 SSH 重启 xray
func (s *NodeService) restartXrayViaSSH(node *model.Node) error {
	sshClient, err := s.sshConnect(node.IP, node.SSHPort, node.SSHUser, node.SSHPassword)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %v", err)
	}
	defer sshClient.Close()

	// 尝试多种重启命令
	commands := []string{
		"xray restart",
		"systemctl restart xray",
		"x-ui restart",
		"systemctl restart x-ui",
	}

	for _, cmd := range commands {
		output, err := s.sshRun(sshClient, cmd)
		if err == nil {
			return nil
		}
		// 记录失败但继续尝试下一个命令
		fmt.Printf("命令 %s 失败: %v, 输出: %s\n", cmd, err, output)
	}

	return fmt.Errorf("所有重启命令都失败")
}

// getClient 获取节点客户端
func (s *NodeService) getClient(nodeID uint) (*NodeClient, error) {
	if v, ok := s.clients.Load(nodeID); ok {
		client := v.(*NodeClient)
		if client.Token != "" && client.ExpireAt.After(time.Now()) {
			return client, nil
		}
	}

	node, err := s.GetByID(nodeID)
	if err != nil {
		return nil, err
	}

	client := &NodeClient{
		NodeID:   node.ID,
		IP:       node.IP,
		Port:     node.Port,
		Username: node.Username,
		Password: node.Password,
	}

	if err := client.Login(); err != nil {
		return nil, err
	}

	s.clients.Store(nodeID, client)
	return client, nil
}

// ========== 订阅链接生成 ==========

// Subscription 订阅信息
type Subscription struct {
	UserID       uint         `json:"userId"`
	Username     string       `json:"username"`
	TrafficUsed  int64        `json:"trafficUsed"`
	TrafficLimit int64        `json:"trafficLimit"`
	ExpireAt     string       `json:"expireAt"`
	Nodes        []NodeConfig `json:"nodes"`
}

// NodeConfig 节点配置
type NodeConfig struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Link     string `json:"link"`
}

// GenerateSubscription 生成用户订阅链接
func (s *NodeService) GenerateSubscription(userID uint) (string, error) {
	// 获取用户信息
	var user model.User
	if err := model.GetDB().First(&user, userID).Error; err != nil {
		return "", err
	}

	// 获取用户分配的节点
	var userNodes []model.UserNode
	model.GetDB().Where("user_id = ? AND enable = ?", userID, true).Find(&userNodes)

	// 获取所有启用的节点
	nodes, _ := s.GetAll()

	var nodeConfigs []NodeConfig
	for _, userNode := range userNodes {
		for _, node := range nodes {
			if node.ID == userNode.NodeID {
				// 获取节点的入站配置
				client, err := s.getClient(node.ID)
				if err != nil {
					continue
				}

				inbounds, err := client.GetInbounds()
				if err != nil {
					continue
				}

				// 生成每个入站的链接
				for _, inbound := range inbounds {
					link := s.generateLink(node, inbound, user)
					if link != "" {
						nodeConfigs = append(nodeConfigs, NodeConfig{
							Name:     node.Name,
							Protocol: fmt.Sprintf("%v", inbound["protocol"]),
							Link:     link,
						})
					}
				}
			}
		}
	}

	// 如果用户没有分配节点，返回所有可用节点
	if len(nodeConfigs) == 0 {
		for _, node := range nodes {
			client, err := s.getClient(node.ID)
			if err != nil {
				continue
			}

			inbounds, err := client.GetInbounds()
			if err != nil {
				continue
			}

			for _, inbound := range inbounds {
				link := s.generateLink(node, inbound, user)
				if link != "" {
					nodeConfigs = append(nodeConfigs, NodeConfig{
						Name:     node.Name,
						Protocol: fmt.Sprintf("%v", inbound["protocol"]),
						Link:     link,
					})
				}
			}
		}
	}

	// 生成订阅内容
	var links []string
	for _, cfg := range nodeConfigs {
		links = append(links, cfg.Link)
	}

	// Base64编码
	subContent := ""
	for _, link := range links {
		subContent += link + "\n"
	}

	return base64.StdEncoding.EncodeToString([]byte(subContent)), nil
}

// generateLink 生成节点链接
func (s *NodeService) generateLink(node model.Node, inbound map[string]interface{}, user model.User) string {
	protocol, _ := inbound["protocol"].(string)
	port, _ := inbound["port"].(float64)
	remark, _ := inbound["remark"].(string)

	settings, _ := json.Marshal(inbound["settings"])
	stream, _ := json.Marshal(inbound["streamSettings"])

	// 根据协议生成链接
	switch protocol {
	case "vmess":
		return s.generateVMessLink(node.IP, int(port), remark, string(settings), string(stream))
	case "vless":
		return s.generateVLESSLink(node.IP, int(port), remark, string(settings), string(stream))
	case "trojan":
		return s.generateTrojanLink(node.IP, int(port), remark, string(settings))
	case "shadowsocks":
		return s.generateSSLink(node.IP, int(port), remark, string(settings))
	}

	return ""
}

// generateVMessLink 生成VMess链接
func (s *NodeService) generateVMessLink(host string, port int, remark, settings, stream string) string {
	var sett struct {
		Clients []struct {
			ID string `json:"id"`
		} `json:"clients"`
	}
	json.Unmarshal([]byte(settings), &sett)

	if len(sett.Clients) == 0 {
		return ""
	}

	uuid := sett.Clients[0].ID

	vmess := map[string]interface{}{
		"v":    "2",
		"ps":   remark,
		"add":  host,
		"port": port,
		"id":   uuid,
		"aid":  0,
		"net":  "tcp",
		"type": "none",
		"host": "",
		"path": "",
		"tls":  "",
	}

	data, _ := json.Marshal(vmess)
	return "vmess://" + base64.StdEncoding.EncodeToString(data)
}

// generateVLESSLink 生成VLESS链接
func (s *NodeService) generateVLESSLink(host string, port int, remark, settings, stream string) string {
	var sett struct {
		Clients []struct {
			ID string `json:"id"`
		} `json:"clients"`
	}
	json.Unmarshal([]byte(settings), &sett)

	if len(sett.Clients) == 0 {
		return ""
	}

	uuid := sett.Clients[0].ID
	return fmt.Sprintf("vless://%s@%s:%d?security=none#%s", uuid, host, port, remark)
}

// generateTrojanLink 生成Trojan链接
func (s *NodeService) generateTrojanLink(host string, port int, remark, settings string) string {
	var sett struct {
		Clients []struct {
			Password string `json:"password"`
		} `json:"clients"`
	}
	json.Unmarshal([]byte(settings), &sett)

	if len(sett.Clients) == 0 {
		return ""
	}

	password := sett.Clients[0].Password
	return fmt.Sprintf("trojan://%s@%s:%d?security=none#%s", password, host, port, remark)
}

// generateSSLink 生成Shadowsocks链接
func (s *NodeService) generateSSLink(host string, port int, remark, settings string) string {
	var sett struct {
		Method   string `json:"method"`
		Password string `json:"password"`
	}
	json.Unmarshal([]byte(settings), &sett)

	if sett.Password == "" {
		return ""
	}

	user := base64.StdEncoding.EncodeToString([]byte(sett.Method + ":" + sett.Password))
	return fmt.Sprintf("ss://%s@%s:%d#%s", user, host, port, remark)
}

// ========== NodeClient 方法 ==========

// NodeStatus 节点状态
type NodeStatus struct {
	CPU           float64 `json:"cpu"`
	Memory        float64 `json:"memory"`
	Disk          float64 `json:"disk"`
	Uptime        uint64  `json:"uptime"`
	XrayVersion   string  `json:"xrayVersion"`
	XrayState     string  `json:"xrayState"`
	UploadTotal   int64   `json:"uploadTotal"`
	DownloadTotal int64   `json:"downloadTotal"`
}

// Login 登录节点
func (c *NodeClient) Login() error {
	url := fmt.Sprintf("http://%s:%d/login", c.IP, c.Port)

	data := map[string]string{
		"username": c.Username,
		"password": c.Password,
	}
	jsonData, _ := json.Marshal(data)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("连接节点失败: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}

	json.Unmarshal(body, &result)

	if !result.Success {
		return fmt.Errorf("登录失败: %s", result.Msg)
	}

	// 从 Cookie 中获取 session
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "session" {
			c.Token = cookie.Value
			break
		}
	}

	if c.Token == "" {
		return fmt.Errorf("登录失败: 未获取到 session")
	}

	c.ExpireAt = time.Now().Add(24 * time.Hour)
	return nil
}

// Request 发送 API 请求
func (c *NodeClient) Request(method, path string, data interface{}) ([]byte, error) {
	url := fmt.Sprintf("http://%s:%d%s", c.IP, c.Port, path)

	var body io.Reader
	if data != nil {
		jsonData, _ := json.Marshal(data)
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", c.Token)
		req.AddCookie(&http.Cookie{Name: "session", Value: c.Token})
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// GetStatus 获取节点状态
func (c *NodeClient) GetStatus() (*NodeStatus, error) {
	body, err := c.Request("POST", "/server/status", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Success bool        `json:"success"`
		Obj     interface{} `json:"obj"`
	}

	json.Unmarshal(body, &result)

	status := &NodeStatus{}
	if obj, ok := result.Obj.(map[string]interface{}); ok {
		if v, ok := obj["cpu"].(float64); ok {
			status.CPU = v
		}
		if v, ok := obj["uptime"].(float64); ok {
			status.Uptime = uint64(v)
		}
		if mem, ok := obj["mem"].(map[string]interface{}); ok {
			if cur, ok := mem["current"].(float64); ok {
				if total, ok := mem["total"].(float64); ok && total > 0 {
					status.Memory = cur / total * 100
				}
			}
		}
		if disk, ok := obj["disk"].(map[string]interface{}); ok {
			if cur, ok := disk["current"].(float64); ok {
				if total, ok := disk["total"].(float64); ok && total > 0 {
					status.Disk = cur / total * 100
				}
			}
		}
		if xray, ok := obj["xray"].(map[string]interface{}); ok {
			if v, ok := xray["state"].(string); ok {
				status.XrayState = v
			}
			if v, ok := xray["version"].(string); ok {
				status.XrayVersion = v
			}
		}
		if traffic, ok := obj["netTraffic"].(map[string]interface{}); ok {
			if v, ok := traffic["sent"].(float64); ok {
				status.UploadTotal = int64(v)
			}
			if v, ok := traffic["recv"].(float64); ok {
				status.DownloadTotal = int64(v)
			}
		}
	}

	return status, nil
}

// GetInbounds 获取入站列表
func (c *NodeClient) GetInbounds() ([]map[string]interface{}, error) {
	body, err := c.Request("POST", "/inbound/list", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Success bool                     `json:"success"`
		Obj     []map[string]interface{} `json:"obj"`
	}

	json.Unmarshal(body, &result)
	if !result.Success {
		return nil, fmt.Errorf("获取入站列表失败")
	}

	return result.Obj, nil
}

// AddInbound 添加入站
func (c *NodeClient) AddInbound(inbound map[string]interface{}) error {
	body, err := c.Request("POST", "/inbound/add", inbound)
	if err != nil {
		return err
	}

	var result struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}

	json.Unmarshal(body, &result)
	if !result.Success {
		return fmt.Errorf("添加入站失败: %s", result.Msg)
	}

	return nil
}

// DeleteInbound 删除入站
func (c *NodeClient) DeleteInbound(id int) error {
	body, err := c.Request("POST", fmt.Sprintf("/inbound/del/%d", id), nil)
	if err != nil {
		return err
	}

	var result struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}

	json.Unmarshal(body, &result)
	if !result.Success {
		return fmt.Errorf("删除入站失败: %s", result.Msg)
	}

	return nil
}

// RestartXray 重启 Xray
func (c *NodeClient) RestartXray() error {
	body, err := c.Request("POST", "/xray/restart", nil)
	if err != nil {
		return err
	}

	var result struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}

	json.Unmarshal(body, &result)
	if !result.Success {
		return fmt.Errorf("重启失败: %s", result.Msg)
	}

	return nil
}

// ========== SSH 管理功能 ==========

// ResetCredentials 通过SSH重置面板账号密码
func (s *NodeService) ResetCredentials(nodeID uint, username, password string) error {
	node, err := s.GetByID(nodeID)
	if err != nil {
		return err
	}

	sshClient, err := s.SSHConnect(node.IP, node.SSHPort, node.SSHUser, node.SSHPassword)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %v", err)
	}
	defer sshClient.Close()

	// 使用 ncp-agent 命令设置
	_, err = s.SSHRun(sshClient, fmt.Sprintf("ncp-agent set-user '%s'", username))
	if err != nil {
		return fmt.Errorf("设置用户名失败: %v", err)
	}

	_, err = s.SSHRun(sshClient, fmt.Sprintf("ncp-agent set-pass '%s'", password))
	if err != nil {
		return fmt.Errorf("设置密码失败: %v", err)
	}

	// 重启服务
	s.SSHRun(sshClient, "ncp-agent restart")

	// 更新数据库
	model.GetDB().Model(&model.Node{}).Where("id = ?", nodeID).Updates(map[string]interface{}{
		"username": username,
		"password": password,
	})

	return nil
}

// CheckUpdate 检查 3x-ui 面板更新
func (s *NodeService) CheckUpdate(nodeID uint) (map[string]interface{}, error) {
	node, err := s.GetByID(nodeID)
	if err != nil {
		return nil, err
	}

	sshClient, err := s.SSHConnect(node.IP, node.SSHPort, node.SSHUser, node.SSHPassword)
	if err != nil {
		return nil, fmt.Errorf("SSH连接失败: %v", err)
	}
	defer sshClient.Close()

	// 获取 3x-ui 最新版本
	latestCmd := "curl -s https://api.github.com/repos/MHSanaei/3x-ui/releases/latest 2>/dev/null | grep '\"tag_name\"' | sed -E 's/.*\"([^\"]+)\".*/\\1/' || echo 'unknown'"
	latestOutput, _ := s.SSHRun(sshClient, latestCmd)
	latestVersion := strings.TrimSpace(latestOutput)

	// 检查 3x-ui 是否已安装
	statusOutput, _ := s.SSHRun(sshClient, "systemctl is-active x-ui 2>/dev/null || echo 'inactive'")

	// 检查当前版本
	currentOutput, _ := s.SSHRun(sshClient, "cat /usr/local/x-ui/VERSION 2>/dev/null || echo '已安装'")

	return map[string]interface{}{
		"currentVersion": strings.TrimSpace(currentOutput),
		"latestVersion":  latestVersion,
		"needUpdate":     latestVersion != "" && latestVersion != "unknown" && strings.TrimSpace(currentOutput) != latestVersion,
		"status":         strings.TrimSpace(statusOutput),
	}, nil
}

// UpdateAgent 更新 3x-ui 面板
func (s *NodeService) UpdateAgent(nodeID uint) (string, error) {
	node, err := s.GetByID(nodeID)
	if err != nil {
		return "", err
	}

	sshClient, err := s.SSHConnect(node.IP, node.SSHPort, node.SSHUser, node.SSHPassword)
	if err != nil {
		return "", fmt.Errorf("SSH连接失败: %v", err)
	}
	defer sshClient.Close()

	// 非交互式更新脚本
	updateScript := `
cd /usr/local/
ARCH=$(uname -m)
if [[ "$ARCH" == "x86_64" ]]; then
    ARCH="amd64"
elif [[ "$ARCH" == "aarch64" ]]; then
    ARCH="arm64"
fi

# 获取最新版本
LATEST=$(curl -s https://api.github.com/repos/MHSanaei/3x-ui/releases/latest | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
if [[ -z "$LATEST" ]]; then
    LATEST="v2.8.11"
fi

echo "正在更新 3x-ui 到版本: $LATEST"

# 备份数据库
mkdir -p /tmp/x-ui-backup
cp -f /usr/local/x-ui/db/x-ui.db /tmp/x-ui-backup/ 2>/dev/null || true
cp -f /usr/local/x-ui/bin/config.json /tmp/x-ui-backup/ 2>/dev/null || true

# 下载新版本
wget -q -O /tmp/x-ui-update.tar.gz "https://github.com/MHSanaei/3x-ui/releases/download/${LATEST}/x-ui-linux-${ARCH}.tar.gz" || {
    echo "下载失败"
    exit 1
}

# 停止服务
systemctl stop x-ui 2>/dev/null || true

# 解压覆盖
rm -rf /usr/local/x-ui/x-ui
tar -xzf /tmp/x-ui-update.tar.gz -C /usr/local/
rm -f /tmp/x-ui-update.tar.gz

# 恢复数据库和配置
cp -f /tmp/x-ui-backup/x-ui.db /usr/local/x-ui/db/ 2>/dev/null || true
cp -f /tmp/x-ui-backup/config.json /usr/local/x-ui/bin/ 2>/dev/null || true

# 写入版本文件
echo "$LATEST" > /usr/local/x-ui/VERSION

# 设置权限
chmod +x /usr/local/x-ui/x-ui
chmod +x /usr/local/x-ui/bin/xray-linux-${ARCH} 2>/dev/null || true

# 启动服务
systemctl daemon-reload
systemctl start x-ui

sleep 3
if systemctl is-active --quiet x-ui; then
    echo "3x-ui 更新成功，版本: $LATEST"
else
    echo "更新完成，但服务启动失败，请检查日志"
fi
`
	output, err := s.SSHRun(sshClient, updateScript)
	if err != nil {
		return output, fmt.Errorf("更新失败: %v", err)
	}

	return output, nil
}
