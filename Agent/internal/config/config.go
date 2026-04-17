// Package config 加载和持久化 ncp-agent 自身配置（不含 xray.json）
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config 是 /etc/ncp-agent/agent.yaml 的内容
type Config struct {
	// Master 协议
	MasterURL string `yaml:"master_url"`           // 形如 https://master.example.com
	NodeID    uint   `yaml:"node_id"`
	NodeToken string `yaml:"node_token"`           // 等价于 Master.Node.AgentKey

	// 拉/推周期（秒），如未配则用 Master 下发的 settings
	PullInterval int `yaml:"pull_interval,omitempty"`
	PushInterval int `yaml:"push_interval,omitempty"`

	// xray 进程相关
	XrayBin        string `yaml:"xray_bin"`             // 默认 /usr/local/bin/xray
	XrayConfigPath string `yaml:"xray_config_path"`     // 默认 /usr/local/etc/xray/config.json
	XrayService    string `yaml:"xray_service"`         // systemd 服务名，默认 xray
	XrayAPIPort    int    `yaml:"xray_api_port"`        // stats gRPC 端口，默认 10085

	// 缓存目录（断网时复用上次配置）
	CacheDir string `yaml:"cache_dir"`                  // 默认 /var/lib/ncp-agent

	// 日志
	LogLevel string `yaml:"log_level"`                  // debug/info/warn/error，默认 info

	// 是否启用自动管理防火墙端口
	ManageFirewall bool `yaml:"manage_firewall"`

	// 调试：跳过 xray reload（dry-run）
	DryRun bool `yaml:"dry_run"`
}

// Load 读 yaml 并填充默认值
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	c.applyDefaults()
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) applyDefaults() {
	if c.XrayBin == "" {
		c.XrayBin = "/usr/local/bin/xray"
	}
	if c.XrayConfigPath == "" {
		c.XrayConfigPath = "/usr/local/etc/xray/config.json"
	}
	if c.XrayService == "" {
		c.XrayService = "xray"
	}
	if c.XrayAPIPort == 0 {
		c.XrayAPIPort = 10085
	}
	if c.CacheDir == "" {
		c.CacheDir = "/var/lib/ncp-agent"
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if c.PullInterval == 0 {
		c.PullInterval = 60
	}
	if c.PushInterval == 0 {
		c.PushInterval = 60
	}
}

func (c *Config) validate() error {
	if c.MasterURL == "" {
		return fmt.Errorf("master_url required")
	}
	if c.NodeID == 0 {
		return fmt.Errorf("node_id required")
	}
	if c.NodeToken == "" {
		return fmt.Errorf("node_token required")
	}
	return nil
}

// CachePath 返回缓存子文件绝对路径，自动 mkdir
func (c *Config) CachePath(name string) (string, error) {
	if err := os.MkdirAll(c.CacheDir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(c.CacheDir, name), nil
}
