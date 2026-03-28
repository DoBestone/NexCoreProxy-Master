package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AgentAPIClient ncp-api 客户端
type AgentAPIClient struct {
	httpClient *http.Client
}

// NewAgentAPIClient 创建客户端
func NewAgentAPIClient() *AgentAPIClient {
	return &AgentAPIClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// AgentAPIStatus 状态响应
type AgentAPIStatus struct {
	Success bool `json:"success"`
	Data    struct {
		XrayVersion   string  `json:"xrayVersion"`
		CPU           float64 `json:"cpu"`
		Memory        float64 `json:"mem"`
		Disk          float64 `json:"disk"`
		Uptime        uint64  `json:"uptime"`
		UploadTotal   int64   `json:"uploadTotal"`
		DownloadTotal int64   `json:"downloadTotal"`
	} `json:"data"`
}

// AgentAPIInbound 入站信息
type AgentAPIInbound struct {
	ID          int    `json:"id"`
	Remark      string `json:"remark"`
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	Enable      bool   `json:"enable"`
	Tag         string `json:"tag"`
	TotalUp     int64  `json:"up"`
	TotalDown   int64  `json:"down"`
	TotalClient int    `json:"totalClient"`
}

// AgentAPIInbounds 入站列表响应
type AgentAPIInbounds struct {
	Success bool              `json:"success"`
	Data    []AgentAPIInbound `json:"data"`
}

// AgentAPIClientInfo 客户端信息
type AgentAPIClientInfo struct {
	ID         int    `json:"id"`
	Email      string `json:"email"`
	Enable     bool   `json:"enable"`
	TotalUp    int64  `json:"up"`
	TotalDown  int64  `json:"down"`
	ExpiryTime int64  `json:"expiryTime"`
}

// AgentAPIClients 客户端列表响应
type AgentAPIClients struct {
	Success bool                 `json:"success"`
	Data    []AgentAPIClientInfo `json:"data"`
}

// AgentAPIResponse 通用响应
type AgentAPIResponse struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
	Data    any    `json:"data"`
}

// doRequest 执行请求
func (c *AgentAPIClient) doRequest(url, token string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-API-Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// doRequestWithRetry 带重试的请求
func (c *AgentAPIClient) doRequestWithRetry(url, token string, maxRetries int) ([]byte, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		body, err := c.doRequest(url, token)
		if err == nil {
			return body, nil
		}
		lastErr = err
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	return nil, lastErr
}

// GetStatus 获取节点状态
func (c *AgentAPIClient) GetStatus(ip string, port int, token string) (*AgentAPIStatus, error) {
	url := fmt.Sprintf("http://%s:%d/api/status", ip, port)
	body, err := c.doRequestWithRetry(url, token, 2)
	if err != nil {
		return nil, err
	}

	var status AgentAPIStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// GetInbounds 获取入站列表
func (c *AgentAPIClient) GetInbounds(ip string, port int, token string) (*AgentAPIInbounds, error) {
	url := fmt.Sprintf("http://%s:%d/api/inbounds", ip, port)
	body, err := c.doRequestWithRetry(url, token, 2)
	if err != nil {
		return nil, err
	}

	var inbounds AgentAPIInbounds
	if err := json.Unmarshal(body, &inbounds); err != nil {
		return nil, err
	}

	return &inbounds, nil
}

// GetInbound 获取单个入站详情
func (c *AgentAPIClient) GetInbound(ip string, port int, token string, inboundID int) (map[string]interface{}, error) {
	url := fmt.Sprintf("http://%s:%d/api/inbound/%d", ip, port, inboundID)
	body, err := c.doRequestWithRetry(url, token, 2)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetClients 获取客户端列表
func (c *AgentAPIClient) GetClients(ip string, port int, token string, inboundID int) (*AgentAPIClients, error) {
	url := fmt.Sprintf("http://%s:%d/api/clients/%d", ip, port, inboundID)
	body, err := c.doRequestWithRetry(url, token, 2)
	if err != nil {
		return nil, err
	}

	var clients AgentAPIClients
	if err := json.Unmarshal(body, &clients); err != nil {
		return nil, err
	}

	return &clients, nil
}

// Restart 重启面板
func (c *AgentAPIClient) Restart(ip string, port int, token string) error {
	url := fmt.Sprintf("http://%s:%d/api/restart", ip, port)
	_, err := c.doRequest(url, token)
	return err
}

// RestartXray 重启 Xray
func (c *AgentAPIClient) RestartXray(ip string, port int, token string) error {
	url := fmt.Sprintf("http://%s:%d/api/restart-xray", ip, port)
	_, err := c.doRequest(url, token)
	return err
}

// IsAPIAvailable 检查 API 是否可用
func (c *AgentAPIClient) IsAPIAvailable(ip string, port int, token string) bool {
	url := fmt.Sprintf("http://%s:%d/api/status", ip, port)
	_, err := c.doRequest(url, token)
	return err == nil
}