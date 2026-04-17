package service

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"nexcoreproxy-master/internal/model"

	"golang.org/x/crypto/ssh"
)

// NodeService 节点服务
type NodeService struct {
	clients     sync.Map
	agentAPI    *AgentAPIClient
	provisioner *NodeProvisioner
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

// GetAll 获取所有启用的节点
func (s *NodeService) GetAll() ([]model.Node, error) {
	var nodes []model.Node
	err := model.GetDB().Where("enable = ?", true).Find(&nodes).Error
	return nodes, err
}

// GetAllAdmin 获取所有节点（包括禁用的，管理员用）
func (s *NodeService) GetAllAdmin() ([]model.Node, error) {
	var nodes []model.Node
	err := model.GetDB().Find(&nodes).Error
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
	// 自动生成 AgentKey（唯一索引，不能为空）
	if node.AgentKey == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return fmt.Errorf("生成 AgentKey 失败: %v", err)
		}
		node.AgentKey = fmt.Sprintf("%x", b)
	}
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

// Install SSH自动安装 NexCoreProxy Panel (内置 3x-ui + NCP)
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

	// 生成随机凭据
	port := 10000 + secureRandomInt(55000)
	user := generateRandomString(8)
	password := generateRandomString(16)

	// 一条命令完成：下载 + 清理旧装 + 安装 + 配置端口/账号/密码 + 生成 Token + 启动
	installCmd := fmt.Sprintf(
		`bash <(curl -fsSL https://raw.githubusercontent.com/DoBestone/NexCoreProxy/main/Panel/install.sh) --force -p %d -u '%s' -k '%s'`,
		port, shellEscape(user), shellEscape(password),
	)
	output, err := s.sshRun(sshClient, installCmd)
	if err != nil {
		return fmt.Errorf("安装失败: %v, 输出: %s", err, output)
	}

	// 等待服务完全启动
	time.Sleep(3 * time.Second)

	// 读取 API Token
	apiTokenOutput, _ := s.SSHRun(sshClient, "ncp-agent get-token 2>/dev/null")
	apiToken := strings.TrimSpace(apiTokenOutput)
	if len(apiToken) != 32 {
		apiToken = ""
	} else if matched, _ := regexp.MatchString("^[a-zA-Z0-9]{32}$", apiToken); !matched {
		apiToken = ""
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
		updates["api_port"] = 54322
	}

	err = model.GetDB().Model(&model.Node{}).Where("id = ?", node.ID).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("更新节点信息失败: %v", err)
	}

	return nil
}

// generateRandomString 生成加密安全的随机字符串
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			b[i] = charset[i%len(charset)]
			continue
		}
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// secureRandomInt 生成加密安全的随机整数
func secureRandomInt(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}

// SSHConnect SSH连接 (公开方法)
func (s *NodeService) SSHConnect(host string, port int, user, password string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
			ssh.KeyboardInteractive(func(name, instruction string, questions []string, echos []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range questions {
					answers[i] = password
				}
				return answers, nil
			}),
		},
		// 首次连接自动信任并记录主机密钥，后续连接验证一致性（TOFU 策略）
		HostKeyCallback: s.hostKeyCallback(),
		Timeout:         30 * time.Second,
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	return ssh.Dial("tcp", addr, config)
}

// hostKeyCallback 实现 Trust-On-First-Use (TOFU) 主机密钥验证
// 首次连接时记录密钥，后续连接时验证是否一致
func (s *NodeService) hostKeyCallback() ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		fingerprint := ssh.FingerprintSHA256(key)
		host := hostname
		if h, _, err := net.SplitHostPort(hostname); err == nil {
			host = h
		}

		// 检查数据库中是否有已记录的主机密钥指纹
		var node model.Node
		if err := model.GetDB().Where("ip = ?", host).First(&node).Error; err != nil {
			// 节点不在数据库中，允许连接（可能是新节点安装）
			log.Printf("[SSH] 新主机 %s 密钥指纹: %s", hostname, fingerprint)
			return nil
		}

		// 如果节点没有记录过密钥指纹，保存并放行（TOFU）
		if node.HostKeyFingerprint == "" {
			log.Printf("[SSH] 首次信任主机 %s 密钥: %s", hostname, fingerprint)
			if err := model.GetDB().Model(&node).Update("host_key_fingerprint", fingerprint).Error; err != nil {
				log.Printf("[SSH] 保存主机密钥指纹失败: %v", err)
			}
			return nil
		}

		// 验证密钥指纹是否一致
		if node.HostKeyFingerprint != fingerprint {
			return fmt.Errorf("SSH 主机密钥不匹配: %s (预期: %s, 实际: %s)，可能遭受中间人攻击", hostname, node.HostKeyFingerprint, fingerprint)
		}

		return nil
	}
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
//
// 自研 agent 节点：基于最近 push 时间判活（agent 主动推架构下无独立健康端点）
// 老 3x-ui 节点：TCP 探测面板端口
func (s *NodeService) TestConnection(id uint) error {
	node, err := s.GetByID(id)
	if err != nil {
		return err
	}

	if node.IsAgentBackend() {
		if node.LastSyncAt == nil {
			return fmt.Errorf("agent 尚未回连，请确认已执行一键部署且节点可访问 Master")
		}
		if age := time.Since(*node.LastSyncAt); age > 3*time.Minute {
			return fmt.Errorf("agent 超过 %v 未心跳（上次: %s）", age.Truncate(time.Second), node.LastSyncAt.Format(time.RFC3339))
		}
		return nil
	}

	addr := net.JoinHostPort(node.IP, strconv.Itoa(node.Port))
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}
	conn.Close()
	return nil
}

// SyncStatus 同步节点状态
//
// 自研 agent 节点：Master 不主动查，直接读 DB（agent 定时 push 已写入）
// 老 3x-ui 节点：优先 HTTP API，刷 token 重试，最后回退 SSH
func (s *NodeService) SyncStatus(id uint) (*NodeStatus, error) {
	node, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if node.IsAgentBackend() {
		return s.syncStatusFromDB(node), nil
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

// syncStatusFromDB 从 DB 读 agent 最近一次 push 写入的状态快照
//
// LastSyncAt 超过 3 分钟视为离线：xray_state = "unknown" + status = "offline"，
// 用于前端给节点挂"离线"标签；不报错以避免前端显示"请求失败"。
func (s *NodeService) syncStatusFromDB(node *model.Node) *NodeStatus {
	xrayState := "running"
	nodeStatus := "online"
	if node.LastSyncAt == nil || time.Since(*node.LastSyncAt) > 3*time.Minute {
		xrayState = "unknown"
		nodeStatus = "offline"
	}
	// 只更新 status 字段，其他指标保留 agent 上次 push 的值
	_ = model.GetDB().Model(&model.Node{}).Where("id = ?", node.ID).
		Update("status", nodeStatus).Error
	return &NodeStatus{
		CPU:           node.CPU,
		Memory:        node.Memory,
		Disk:          node.Disk,
		Uptime:        node.Uptime,
		XrayVersion:   node.XrayVersion,
		XrayState:     xrayState,
		UploadTotal:   node.UploadTotal,
		DownloadTotal: node.DownloadTotal,
	}
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

	// 如果所有关键指标为 0，说明响应格式可能不兼容（旧版 ncp-api 用 "obj" 而非 "data"）
	if status.CPU == 0 && status.Memory == 0 && status.Disk == 0 && status.Uptime == 0 {
		return nil, fmt.Errorf("API 返回数据为空，可能格式不兼容")
	}

	// 更新数据库
	now := time.Now()
	if err := model.GetDB().Model(&model.Node{}).Where("id = ?", node.ID).Updates(map[string]interface{}{
		"status":         "online",
		"cpu":            status.CPU,
		"memory":         status.Memory,
		"disk":           status.Disk,
		"uptime":         status.Uptime,
		"xray_version":   status.XrayVersion,
		"upload_total":   status.UploadTotal,
		"download_total": status.DownloadTotal,
		"last_sync_at":   now,
	}).Error; err != nil {
		log.Printf("[Sync] 更新节点 %d 状态到数据库失败: %v", node.ID, err)
	}

	return status, nil
}

// syncStatusViaSSH 通过 SSH 同步状态（回退方案）
func (s *NodeService) syncStatusViaSSH(node *model.Node) (*NodeStatus, error) {
	sshClient, err := s.SSHConnect(node.IP, node.SSHPort, node.SSHUser, node.SSHPassword)
	if err != nil {
		if dbErr := model.GetDB().Model(&model.Node{}).Where("id = ?", node.ID).Updates(map[string]interface{}{
			"status":       "offline",
			"last_sync_at": time.Now(),
		}).Error; dbErr != nil {
			log.Printf("[Sync] 更新节点 %d 离线状态失败: %v", node.ID, dbErr)
		}
		return nil, fmt.Errorf("SSH连接失败: %v", err)
	}
	defer sshClient.Close()

	// 通过 ncp-agent 获取状态
	output, err := s.SSHRun(sshClient, "ncp-agent info 2>/dev/null || echo 'error'")
	if err != nil || strings.Contains(output, "error") {
		model.GetDB().Model(&model.Node{}).Where("id = ?", node.ID).Updates(map[string]interface{}{
			"status":       "offline",
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

	// 获取系统资源（通过 /proc 读取，比 top 更可靠）
	cpuCmd := `cat /proc/stat | head -1 | awk '{idle=$5; total=0; for(i=2;i<=NF;i++) total+=$i; printf "%.1f", (1-idle/total)*100}'`
	cpuOutput, _ := s.SSHRun(sshClient, cpuCmd)
	status.CPU, _ = strconv.ParseFloat(strings.TrimSpace(cpuOutput), 64)

	memCmd := `awk '/MemTotal/{t=$2} /MemAvailable/{a=$2} END{if(t>0) printf "%.1f", (t-a)/t*100}' /proc/meminfo`
	memOutput, _ := s.SSHRun(sshClient, memCmd)
	status.Memory, _ = strconv.ParseFloat(strings.TrimSpace(memOutput), 64)

	diskOutput, _ := s.SSHRun(sshClient, "df / | tail -1 | awk '{print $5}' | tr -d '%'")
	status.Disk, _ = strconv.ParseFloat(strings.TrimSpace(diskOutput), 64)

	uptimeOutput, _ := s.SSHRun(sshClient, "awk '{print int($1)}' /proc/uptime")
	uptime, _ := strconv.ParseInt(strings.TrimSpace(uptimeOutput), 10, 64)
	status.Uptime = uint64(uptime)

	// 更新数据库
	if err := model.GetDB().Model(&model.Node{}).Where("id = ?", node.ID).Updates(map[string]interface{}{
		"status":       "online",
		"cpu":          status.CPU,
		"memory":       status.Memory,
		"disk":         status.Disk,
		"uptime":       uptime,
		"xray_version": status.XrayVersion,
		"last_sync_at": time.Now(),
	}).Error; err != nil {
		log.Printf("[Sync] 更新节点 %d SSH状态到数据库失败: %v", node.ID, err)
	}

	return status, nil
}

// apiTokenPattern 合法 API Token 只允许字母数字，长度 16-128
// 用于拒绝 SSH fallback 返回的 usage 帮助/错误文本污染数据库
var apiTokenPattern = regexp.MustCompile(`^[A-Za-z0-9]{16,128}$`)

// refreshAPIToken 刷新 API Token
func (s *NodeService) refreshAPIToken(node *model.Node) (string, error) {
	sshClient, err := s.SSHConnect(node.IP, node.SSHPort, node.SSHUser, node.SSHPassword)
	if err != nil {
		return "", err
	}
	defer sshClient.Close()

	// 优先读 TokenFile（ncp-agent gen-token 会同步写入），fallback 到 ncp-agent
	// 多行命令 + 严格比对输出格式，避免 usage 帮助文本被当成 token
	const probeCmd = `cat /usr/local/x-ui/API_TOKEN 2>/dev/null || ncp-agent get-token 2>/dev/null`
	output, err := s.SSHRun(sshClient, probeCmd)
	if err != nil {
		return "", err
	}

	token := strings.TrimSpace(output)
	if !apiTokenPattern.MatchString(token) {
		return "", fmt.Errorf("Token 格式无效或未设置（输出长度=%d）", len(token))
	}

	// 保存到数据库
	if err := model.GetDB().Model(&model.Node{}).Where("id = ?", node.ID).Update("api_token", token).Error; err != nil {
		log.Printf("[Token] 保存节点 %d API Token 失败: %v", node.ID, err)
	}

	return token, nil
}

// SyncAll 同步所有节点状态（限制并发数防止资源耗尽）
func (s *NodeService) SyncAll() {
	nodes, err := s.GetAll()
	if err != nil {
		return
	}

	const maxConcurrent = 20
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, node := range nodes {
		if !node.Enable {
			continue
		}
		wg.Add(1)
		sem <- struct{}{} // 获取信号量
		go func(n model.Node) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量
			s.SyncStatus(n.ID)
		}(node)
	}
	wg.Wait()
}

// GetInbounds 获取节点入站列表
func (s *NodeService) GetInbounds(nodeID uint) ([]map[string]interface{}, error) {
	node, err := s.GetByID(nodeID)
	if err != nil {
		return nil, err
	}

	// 优先使用 NCP API
	if node.APIToken != "" {
		inbounds, err := s.agentAPI.GetInbounds(node.IP, node.APIPort, node.APIToken)
		if err == nil && inbounds.Success && len(inbounds.Data) > 0 {
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

	// 回退到面板 API (通过 NodeClient 登录 3x-ui 获取结构化数据)
	client, err := s.getClient(nodeID)
	if err != nil {
		return nil, err
	}
	return client.GetInbounds()
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

	// 3x-ui 要求 settings/streamSettings 为 JSON 字符串，前端可能发送解析后的对象
	for _, key := range []string{"settings", "streamSettings"} {
		if val, ok := inbound[key]; ok {
			switch val.(type) {
			case string:
				// 已经是字符串，无需处理
			default:
				b, err := json.Marshal(val)
				if err != nil {
					return fmt.Errorf("序列化 %s 失败: %v", key, err)
				}
				inbound[key] = string(b)
			}
		}
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
	if err := model.GetDB().Model(&model.Node{}).Where("id = ?", nodeID).Update("api_token", token).Error; err != nil {
		log.Printf("[Token] 保存节点 %d API Token 失败: %v", nodeID, err)
	}

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
	if err := model.GetDB().Model(&model.Node{}).Where("id = ?", nodeID).Update("api_token", token).Error; err != nil {
		return "", fmt.Errorf("保存 Token 到数据库失败: %v", err)
	}

	return token, nil
}

// ========== SSH 直连控制方法 ==========

// SSHGetStatus 通过SSH获取节点完整状态
func (s *NodeService) SSHGetStatus(nodeID uint) (map[string]interface{}, error) {
	node, err := s.GetByID(nodeID)
	if err != nil {
		return nil, err
	}

	// agent 节点没有 3x-ui，用最近一次 push 的快照组装等价视图
	if node.IsAgentBackend() {
		info := map[string]interface{}{
			"service_status": "active",
			"version":        node.AgentVersion,
			"panel_port":     "-", // 无面板
			"admin_user":     "-",
			"xray_version":   node.XrayVersion,
			"cpu":            node.CPU,
			"memory":         node.Memory,
			"disk":           node.Disk,
			"uptime":         node.Uptime,
			"upload_total":   node.UploadTotal,
			"download_total": node.DownloadTotal,
			"backend":        "ncp-agent",
		}
		if node.LastSyncAt != nil {
			info["last_sync_at"] = node.LastSyncAt.Format(time.RFC3339)
			if time.Since(*node.LastSyncAt) > 3*time.Minute {
				info["service_status"] = "stale"
			}
		}
		return info, nil
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

	// CPU 使用率（通过 /proc/stat，比 top 更可靠）
	cpu, _ := s.SSHRun(sshClient, `cat /proc/stat | head -1 | awk '{idle=$5; total=0; for(i=2;i<=NF;i++) total+=$i; printf "%.1f", (1-idle/total)*100}'`)
	info["cpu"], _ = strconv.ParseFloat(strings.TrimSpace(cpu), 64)

	// 内存使用率（通过 /proc/meminfo）
	mem, _ := s.SSHRun(sshClient, `awk '/MemTotal/{t=$2} /MemAvailable/{a=$2} END{if(t>0) printf "%.1f", (t-a)/t*100}' /proc/meminfo`)
	info["memory"], _ = strconv.ParseFloat(strings.TrimSpace(mem), 64)

	// 磁盘使用率
	disk, _ := s.SSHRun(sshClient, "df / | tail -1 | awk '{print $5}' | tr -d '%'")
	info["disk"], _ = strconv.ParseFloat(strings.TrimSpace(disk), 64)

	// 运行时间（秒）
	uptime, _ := s.SSHRun(sshClient, "awk '{print int($1)}' /proc/uptime")
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
	if err := model.GetDB().Model(&model.Node{}).Where("id = ?", nodeID).Update("port", port).Error; err != nil {
		return fmt.Errorf("更新端口到数据库失败: %v", err)
	}
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

	// 自研 agent 节点：bump config etag，agent 下次心跳拉到新版本会热重载 xray
	if node.IsAgentBackend() {
		if err := model.BumpEtag(nodeID); err != nil {
			return fmt.Errorf("触发 agent 重载失败: %v", err)
		}
		return nil
	}

	// 老 3x-ui 节点：优先 SSH，兜底 panel API
	if node.SSHPassword != "" {
		return s.restartXrayViaSSH(node)
	}
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
		log.Printf("[SSH] 命令 %s 失败: %v, 输出: %s", cmd, err, output)
	}

	return fmt.Errorf("所有重启命令都失败")
}

// getClient 获取节点客户端
func (s *NodeService) getClient(nodeID uint) (*NodeClient, error) {
	if v, ok := s.clients.Load(nodeID); ok {
		client, ok := v.(*NodeClient)
		if !ok {
			s.clients.Delete(nodeID)
			return nil, fmt.Errorf("节点客户端类型错误")
		}
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
	loginURL := fmt.Sprintf("http://%s/login", net.JoinHostPort(c.IP, strconv.Itoa(c.Port)))

	data := map[string]string{
		"username": c.Username,
		"password": c.Password,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化登录数据失败: %v", err)
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient.Post(loginURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("连接节点失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	var result struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	if !result.Success {
		return fmt.Errorf("登录失败: %s", result.Msg)
	}

	// 从 Cookie 中获取 session（3x-ui 使用 "3x-ui" 作为 cookie 名）
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "3x-ui" || cookie.Name == "session" {
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
	url := fmt.Sprintf("http://%s%s", net.JoinHostPort(c.IP, strconv.Itoa(c.Port)), path)

	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("序列化请求数据失败: %v", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", c.Token)
		req.AddCookie(&http.Cookie{Name: "3x-ui", Value: c.Token})
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
	body, err := c.Request("GET", "/panel/api/server/status", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Success bool        `json:"success"`
		Obj     interface{} `json:"obj"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析状态响应失败: %v", err)
	}

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
	body, err := c.Request("GET", "/panel/api/inbounds/list", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Success bool                     `json:"success"`
		Obj     []map[string]interface{} `json:"obj"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析入站列表失败: %v", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("获取入站列表失败")
	}

	return result.Obj, nil
}

// AddInbound 添加入站
func (c *NodeClient) AddInbound(inbound map[string]interface{}) error {
	body, err := c.Request("POST", "/panel/api/inbounds/add", inbound)
	if err != nil {
		return err
	}

	var result struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}
	if !result.Success {
		return fmt.Errorf("添加入站失败: %s", result.Msg)
	}

	return nil
}

// DeleteInbound 删除入站
func (c *NodeClient) DeleteInbound(id int) error {
	body, err := c.Request("POST", fmt.Sprintf("/panel/api/inbounds/del/%d", id), nil)
	if err != nil {
		return err
	}

	var result struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}
	if !result.Success {
		return fmt.Errorf("删除入站失败: %s", result.Msg)
	}

	return nil
}

// RestartXray 重启 Xray
func (c *NodeClient) RestartXray() error {
	body, err := c.Request("POST", "/panel/api/server/restartXrayService", nil)
	if err != nil {
		return err
	}

	var result struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}
	if !result.Success {
		return fmt.Errorf("重启失败: %s", result.Msg)
	}

	return nil
}

// ========== SSH 管理功能 ==========

// shellEscape 转义 shell 单引号，防止命令注入
func shellEscape(s string) string {
	return strings.ReplaceAll(s, "'", "'\"'\"'")
}

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

	// 使用 ncp-agent 命令设置（转义防注入）
	_, err = s.SSHRun(sshClient, fmt.Sprintf("ncp-agent set-user '%s'", shellEscape(username)))
	if err != nil {
		return fmt.Errorf("设置用户名失败: %v", err)
	}

	_, err = s.SSHRun(sshClient, fmt.Sprintf("ncp-agent set-pass '%s'", shellEscape(password)))
	if err != nil {
		return fmt.Errorf("设置密码失败: %v", err)
	}

	// 重启服务使新凭证生效
	if _, err := s.SSHRun(sshClient, "ncp-agent restart"); err != nil {
		return fmt.Errorf("重启服务失败: %v", err)
	}

	// 更新数据库
	if err := model.GetDB().Model(&model.Node{}).Where("id = ?", nodeID).Updates(map[string]interface{}{
		"username": username,
		"password": password,
	}).Error; err != nil {
		return fmt.Errorf("更新凭证到数据库失败: %v", err)
	}

	return nil
}

// panelRepo 面板的 GitHub 仓库
const panelRepo = "DoBestone/NexCoreProxy-Panel"

// CheckUpdate 检查 Panel 更新（基于 DoBestone/NexCoreProxy-Panel 的 Release）
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

	// 获取当前版本
	currentOutput, _ := s.SSHRun(sshClient, "cat /usr/local/x-ui/VERSION 2>/dev/null")
	currentVersion := strings.TrimSpace(currentOutput)
	if currentVersion == "" {
		currentVersion = "未知"
	}

	// 服务状态
	statusOutput, _ := s.SSHRun(sshClient, "systemctl is-active x-ui 2>/dev/null || echo 'inactive'")

	// 从 GitHub API 获取最新 Release
	latestCmd := fmt.Sprintf(
		`curl -s https://api.github.com/repos/%s/releases/latest 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'`,
		panelRepo,
	)
	latestOutput, _ := s.SSHRun(sshClient, latestCmd)
	latestVersion := strings.TrimSpace(latestOutput)

	// 没有发布 Release 则不需要更新
	needUpdate := false
	if latestVersion != "" && currentVersion != "未知" && latestVersion != currentVersion {
		needUpdate = true
	}

	return map[string]interface{}{
		"currentVersion": currentVersion,
		"latestVersion":  latestVersion,
		"needUpdate":     needUpdate,
		"status":         strings.TrimSpace(statusOutput),
	}, nil
}

// UpdateAgent 更新 Panel 到最新 Release
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

	updateScript := fmt.Sprintf(`
set -e
REPO="%s"
ARCH=$(uname -m)
[[ "$ARCH" == "x86_64" ]] && ARCH="amd64"
[[ "$ARCH" == "aarch64" ]] && ARCH="arm64"

# 获取最新 Release 版本
LATEST=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
if [[ -z "$LATEST" ]]; then
    echo "错误: 仓库没有发布 Release，无法更新"
    exit 1
fi

CURRENT=$(cat /usr/local/x-ui/VERSION 2>/dev/null || echo "")
if [[ "$CURRENT" == "$LATEST" ]]; then
    echo "当前已是最新版本: $LATEST"
    exit 0
fi

echo "正在更新 NexCoreProxy Panel: $CURRENT -> $LATEST"

# 备份数据库和配置
mkdir -p /tmp/ncp-backup
cp -f /etc/x-ui/x-ui.db /tmp/ncp-backup/ 2>/dev/null || true
cp -f /usr/local/x-ui/bin/config.json /tmp/ncp-backup/ 2>/dev/null || true

# 下载新版本
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST}/x-ui-linux-${ARCH}.tar.gz"
wget -q -O /tmp/x-ui-update.tar.gz "$DOWNLOAD_URL" || {
    echo "下载失败: $DOWNLOAD_URL"
    exit 1
}

# 停止服务
systemctl stop x-ui 2>/dev/null || true

# 解压覆盖（保留数据库）
tar -xzf /tmp/x-ui-update.tar.gz -C /usr/local/
rm -f /tmp/x-ui-update.tar.gz

# 恢复数据库（如果被覆盖）
cp -f /tmp/ncp-backup/x-ui.db /etc/x-ui/ 2>/dev/null || true

# 写入版本 & 设置权限
echo "$LATEST" > /usr/local/x-ui/VERSION
chmod +x /usr/local/x-ui/x-ui
chmod +x /usr/local/x-ui/bin/xray-linux-${ARCH} 2>/dev/null || true

# 确保 ncp-agent 符号链接存在
ln -sf /usr/local/x-ui/x-ui /usr/bin/ncp-agent

# 启动服务
systemctl daemon-reload
systemctl start x-ui

sleep 3
if systemctl is-active --quiet x-ui; then
    echo "更新成功，版本: $LATEST"
else
    echo "更新完成，但服务启动失败，请检查日志"
fi
`, panelRepo)

	output, err := s.SSHRun(sshClient, updateScript)
	if err != nil {
		return output, fmt.Errorf("更新失败: %v", err)
	}

	return output, nil
}

