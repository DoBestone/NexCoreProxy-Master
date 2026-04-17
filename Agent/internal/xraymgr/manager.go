// Package xraymgr 负责 xray-core 进程的生命周期：
//   - 把渲染好的 config.json 安全地落盘（temp + atomic rename）
//   - 用 `xray test -c` 在替换前做语法校验，校验失败保留旧配置
//   - 通过 systemd 重启服务（最简单且最可靠）
//   - 暴露 Reload(bytes) 单一入口，供 puller 调用
package xraymgr

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Manager struct {
	XrayBin     string
	ConfigPath  string
	ServiceName string
	DryRun      bool
}

// New 构造 Manager。所有路径要求绝对路径。
func New(xrayBin, configPath, serviceName string, dryRun bool) *Manager {
	return &Manager{
		XrayBin:     xrayBin,
		ConfigPath:  configPath,
		ServiceName: serviceName,
		DryRun:      dryRun,
	}
}

// Reload 把 raw 写入新 config 并触发 xray 重启。
//
// 流程：
//  1. 写到 <config>.new
//  2. xray test -c <config>.new 校验语法
//  3. atomic rename <config>.new → <config>
//  4. systemctl restart <service>
//
// 失败回退：rename 之前的失败不影响线上；rename 之后的 systemd 失败由调用方决定回滚策略。
func (m *Manager) Reload(raw []byte) error {
	if m.DryRun {
		fmt.Println("[dry-run] xray reload skipped, config size =", len(raw))
		return nil
	}
	dir := filepath.Dir(m.ConfigPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	tmp := m.ConfigPath + ".new"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("write tmp config: %w", err)
	}

	if err := m.testConfig(tmp); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("xray config validation failed: %w", err)
	}

	if err := os.Rename(tmp, m.ConfigPath); err != nil {
		return fmt.Errorf("rename config: %w", err)
	}

	return m.restartService()
}

func (m *Manager) testConfig(path string) error {
	cmd := exec.Command(m.XrayBin, "test", "-c", path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (m *Manager) restartService() error {
	cmd := exec.Command("systemctl", "restart", m.ServiceName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("systemctl restart %s: %s", m.ServiceName, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// Version 调 xray version 并返回首行版本号；获取失败返回空串
func (m *Manager) Version() string {
	cmd := exec.Command(m.XrayBin, "version")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return ""
	}
	line := strings.SplitN(strings.TrimSpace(stdout.String()), "\n", 2)[0]
	// "Xray 1.8.24 (Xray, Penetrates Everything.)" → "1.8.24"
	parts := strings.Fields(line)
	if len(parts) >= 2 {
		return parts[1]
	}
	return line
}
