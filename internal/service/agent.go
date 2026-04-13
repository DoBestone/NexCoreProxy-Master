package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"nexcoreproxy-master/internal/model"
)

// AgentCommand Agent命令
type AgentCommand struct {
	ID     string      `json:"id"`
	Action string      `json:"action"`
	Params interface{} `json:"params"`
}

// AgentResponse Agent响应
type AgentResponse struct {
	ID      string      `json:"id"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error"`
}

// AgentStatus Agent状态上报
type AgentStatus struct {
	AgentKey      string  `json:"agentKey"`
	XrayVersion   string  `json:"xrayVersion"`
	CPU           float64 `json:"cpu"`
	Memory        float64 `json:"memory"`
	Disk          float64 `json:"disk"`
	Uptime        uint64  `json:"uptime"`
	UploadTotal   int64   `json:"uploadTotal"`
	DownloadTotal int64   `json:"downloadTotal"`
}

// AgentConnection Agent连接
type AgentConnection struct {
	AgentKey   string
	NodeID     uint
	Send       chan []byte
	Receive    chan []byte
	LastActive time.Time
}

// AgentManager Agent连接管理器
type AgentManager struct {
	connections map[string]*AgentConnection
	mu          sync.RWMutex
	pendingCmds map[string]chan *AgentResponse
}

// NewAgentManager 创建Agent管理器
func NewAgentManager() *AgentManager {
	return &AgentManager{
		connections: make(map[string]*AgentConnection),
		pendingCmds: make(map[string]chan *AgentResponse),
	}
}

// Register 注册Agent连接
func (m *AgentManager) Register(agentKey string, nodeID uint) *AgentConnection {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn := &AgentConnection{
		AgentKey:   agentKey,
		NodeID:     nodeID,
		Send:       make(chan []byte, 100),
		Receive:    make(chan []byte, 100),
		LastActive: time.Now(),
	}

	m.connections[agentKey] = conn

	// 更新节点状态
	db := model.GetDB()
	if err := db.Model(&model.Node{}).Where("agent_key = ?", agentKey).Updates(map[string]interface{}{
		"status":    "online",
		"connected": true,
	}).Error; err != nil {
		log.Printf("[Agent] 更新节点在线状态失败 (key=%s): %v", agentKey[:8], err)
	}

	return conn
}

// Unregister 注销Agent连接
func (m *AgentManager) Unregister(agentKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, ok := m.connections[agentKey]; ok {
		close(conn.Send)
		close(conn.Receive)
		delete(m.connections, agentKey)

		// 更新节点状态
		db := model.GetDB()
		if err := db.Model(&model.Node{}).Where("agent_key = ?", agentKey).Updates(map[string]interface{}{
			"status":    "offline",
			"connected": false,
		}).Error; err != nil {
			log.Printf("[Agent] 更新节点离线状态失败 (key=%s): %v", agentKey[:8], err)
		}
	}
}

// GetConnection 获取连接
func (m *AgentManager) GetConnection(agentKey string) (*AgentConnection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, ok := m.connections[agentKey]
	return conn, ok
}

// IsConnected 检查是否已连接
func (m *AgentManager) IsConnected(agentKey string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.connections[agentKey]
	return ok
}

// SendCommand 发送命令到Agent
func (m *AgentManager) SendCommand(agentKey string, cmd *AgentCommand) (*AgentResponse, error) {
	m.mu.RLock()
	conn, ok := m.connections[agentKey]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("agent未连接")
	}

	// 创建响应通道
	respChan := make(chan *AgentResponse, 1)

	m.mu.Lock()
	m.pendingCmds[cmd.ID] = respChan
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		delete(m.pendingCmds, cmd.ID)
		m.mu.Unlock()
	}()

	// 发送命令
	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	select {
	case conn.Send <- data:
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("发送命令超时")
	}

	// 等待响应
	select {
	case resp := <-respChan:
		return resp, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("等待响应超时")
	}
}

// HandleResponse 处理Agent响应
func (m *AgentManager) HandleResponse(resp *AgentResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ch, ok := m.pendingCmds[resp.ID]; ok {
		ch <- resp
	}
}

// UpdateStatus 更新Agent状态
func (m *AgentManager) UpdateStatus(status *AgentStatus) error {
	db := model.GetDB()

	updates := map[string]interface{}{
		"status":         "online",
		"xray_version":   status.XrayVersion,
		"cpu":            status.CPU,
		"mem":            status.Memory,
		"disk":           status.Disk,
		"uptime":         status.Uptime,
		"upload_total":   status.UploadTotal,
		"download_total": status.DownloadTotal,
		"last_sync_at":   time.Now(),
	}

	result := db.Model(&model.Node{}).Where("agent_key = ?", status.AgentKey).Updates(updates)
	if result.Error != nil {
		return result.Error
	}

	// 更新连接活跃时间
	m.mu.Lock()
	if conn, ok := m.connections[status.AgentKey]; ok {
		conn.LastActive = time.Now()
	}
	m.mu.Unlock()

	return nil
}

// GenerateAgentKey 生成Agent密钥
func GenerateAgentKey() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// 极端情况：随机源不可用时使用时间戳混合
		return hex.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	}
	return hex.EncodeToString(b)
}

// GetConnectedNodes 获取已连接的节点
func (m *AgentManager) GetConnectedNodes() []uint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	nodeIDs := make([]uint, 0, len(m.connections))
	for _, conn := range m.connections {
		nodeIDs = append(nodeIDs, conn.NodeID)
	}
	return nodeIDs
}