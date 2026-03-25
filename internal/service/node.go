package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"nexcoreproxy-master/internal/model"
)

// NodeService 节点服务
type NodeService struct {
	clients sync.Map
}

// NewNodeService 创建节点服务
func NewNodeService() *NodeService {
	return &NodeService{}
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

	// 安装x-ui的命令
	installCmd := fmt.Sprintf(
		`bash <(curl -Ls https://raw.githubusercontent.com/vaxilu/x-ui/main/install.sh) << 'EOF'
y
%s
%s
54321
EOF`,
		node.Username, node.Password,
	)

	output, err := s.sshRun(sshClient, installCmd)
	if err != nil {
		return fmt.Errorf("安装失败: %v, 输出: %s", err, output)
	}

	node.Status = "online"
	model.GetDB().Save(node)
	return nil
}

// sshConnect SSH连接
func (s *NodeService) sshConnect(host string, port int, user, password string) (*ssh.Client, error) {
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

// sshRun 执行SSH命令
func (s *NodeService) sshRun(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	return string(output), err
}

// TestConnection 测试节点连接
func (s *NodeService) TestConnection(id uint) error {
	client, err := s.getClient(id)
	if err != nil {
		return err
	}
	_, err = client.GetStatus()
	return err
}

// SyncStatus 同步节点状态
func (s *NodeService) SyncStatus(id uint) (*NodeStatus, error) {
	client, err := s.getClient(id)
	if err != nil {
		return nil, err
	}

	status, err := client.GetStatus()
	if err != nil {
		model.GetDB().Model(&model.Node{}).Where("id = ?", id).Updates(map[string]interface{}{
			"status":    "offline",
			"last_sync": time.Now(),
		})
		return nil, err
	}

	model.GetDB().Model(&model.Node{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":         "online",
		"cpu":            status.CPU,
		"mem":            status.Memory,
		"disk":           status.Disk,
		"uptime":         status.Uptime,
		"xray_version":   status.XrayVersion,
		"upload_total":   status.UploadTotal,
		"download_total": status.DownloadTotal,
		"last_sync":      time.Now(),
	})

	return status, nil
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

// GetInbounds 获取节点入站列表
func (s *NodeService) GetInbounds(nodeID uint) ([]map[string]interface{}, error) {
	client, err := s.getClient(nodeID)
	if err != nil {
		return nil, err
	}
	return client.GetInbounds()
}

// AddInbound 添加入站
func (s *NodeService) AddInbound(nodeID uint, inbound map[string]interface{}) error {
	client, err := s.getClient(nodeID)
	if err != nil {
		return err
	}
	return client.AddInbound(inbound)
}

// DeleteInbound 删除入站
func (s *NodeService) DeleteInbound(nodeID uint, inboundID int) error {
	client, err := s.getClient(nodeID)
	if err != nil {
		return err
	}
	return client.DeleteInbound(inboundID)
}

// RestartXray 重启节点 Xray
func (s *NodeService) RestartXray(nodeID uint) error {
	client, err := s.getClient(nodeID)
	if err != nil {
		return err
	}
	return client.RestartXray()
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
	UserID      uint   `json:"userId"`
	Username    string `json:"username"`
	TrafficUsed int64  `json:"trafficUsed"`
	TrafficLimit int64 `json:"trafficLimit"`
	ExpireAt    string `json:"expireAt"`
	Nodes       []NodeConfig `json:"nodes"`
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
		Obj     struct {
			Session string `json:"session"`
		} `json:"obj"`
	}

	json.Unmarshal(body, &result)

	if !result.Success {
		return fmt.Errorf("登录失败: %s", result.Msg)
	}

	c.Token = result.Obj.Session
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