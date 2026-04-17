package service

import "sync/atomic"

// RuntimeConfig 启动期由 main 注入，全程只读
//
// 之所以放 atomic.Value 而不是直接 var：将来如果做"系统设置 → DB 持久化"想热更新，
// 可以平滑切换；当前 only main.go 写一次。
type runtimeConfig struct {
	MasterURL       string
	AgentBinaryURL  string // ncp-agent 二进制下载地址，含 ${ARCH} 占位符
	XrayVersion     string
	AgentTargetVer  string
}

var rtCfg atomic.Pointer[runtimeConfig]

// SetRuntimeConfig 由 main 调用一次
func SetRuntimeConfig(masterURL, agentBinaryURL, xrayVersion, agentTargetVer string) {
	rtCfg.Store(&runtimeConfig{
		MasterURL:      masterURL,
		AgentBinaryURL: agentBinaryURL,
		XrayVersion:    xrayVersion,
		AgentTargetVer: agentTargetVer,
	})
}

// GetRuntimeConfig 获取运行时配置；首次未设置时返回零值
func GetRuntimeConfig() runtimeConfig {
	if v := rtCfg.Load(); v != nil {
		return *v
	}
	return runtimeConfig{}
}
