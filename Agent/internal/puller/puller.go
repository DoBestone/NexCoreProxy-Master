// Package puller 周期性向 Master 拉配置并触发 xray reload
package puller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"nexcoreproxy-agent/internal/config"
	"nexcoreproxy-agent/internal/firewall"
	"nexcoreproxy-agent/internal/protocol"
	"nexcoreproxy-agent/internal/upgrade"
	"nexcoreproxy-agent/internal/xraymgr"
	"nexcoreproxy-agent/internal/xrayrender"
)


// Puller 单例
type Puller struct {
	cfg     *config.Config
	client  *protocol.Client
	xray    *xraymgr.Manager
	cacheFp string

	mu      sync.RWMutex
	current *protocol.ServerConfig
	etag    atomic.Value // string

	period atomic.Int64 // 秒，可被 Master 下发的 PullInterval 覆盖
}

func New(cfg *config.Config, client *protocol.Client, xray *xraymgr.Manager) (*Puller, error) {
	cachePath, err := cfg.CachePath("config.json")
	if err != nil {
		return nil, err
	}
	p := &Puller{
		cfg:     cfg,
		client:  client,
		xray:    xray,
		cacheFp: cachePath,
	}
	p.etag.Store("")
	p.period.Store(int64(cfg.PullInterval))

	// 启动时尝试加载缓存（断网容灾：上次拉到的配置）
	if data, err := os.ReadFile(cachePath); err == nil {
		var cached protocol.ServerConfig
		if err := json.Unmarshal(data, &cached); err == nil {
			p.current = &cached
			p.etag.Store(cached.Etag)
			log.Printf("[puller] loaded cached config etag=%s inbounds=%d users=%d",
				cached.Etag, len(cached.Inbounds), len(cached.Users))
		}
	}
	return p, nil
}

// Current 返回最近一次成功拉到的配置（线程安全），可能为 nil
func (p *Puller) Current() *protocol.ServerConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.current
}

// Etag 当前 etag
func (p *Puller) Etag() string {
	return p.etag.Load().(string)
}

// Run 阻塞循环。ctx 取消时退出。
func (p *Puller) Run(ctx context.Context) {
	// 启动立即拉一次
	if err := p.tick(ctx); err != nil {
		log.Printf("[puller] initial tick failed: %v", err)
	}
	for {
		d := time.Duration(p.period.Load()) * time.Second
		if d <= 0 {
			d = 60 * time.Second
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(d):
			if err := p.tick(ctx); err != nil {
				log.Printf("[puller] tick failed: %v", err)
			}
		}
	}
}

func (p *Puller) tick(ctx context.Context) error {
	cfg, notModified, err := p.client.FetchConfig(ctx, p.Etag())
	if err != nil {
		return err
	}
	if notModified {
		return nil
	}
	if err := p.applyConfig(cfg); err != nil {
		return fmt.Errorf("apply config: %w", err)
	}
	return nil
}

// applyConfig 渲染 xray.json 并 reload；成功后写缓存 + 更新 etag
func (p *Puller) applyConfig(cfg *protocol.ServerConfig) error {
	raw, err := xrayrender.Render(cfg, p.cfg.XrayAPIPort, cfg.Settings.LogLevel)
	if err != nil {
		return err
	}
	if err := p.xray.Reload(raw); err != nil {
		return err
	}

	// 防火墙端口收敛：失败不阻塞主流程，仅写日志
	if p.cfg.ManageFirewall {
		ports := firewall.CollectPorts(cfg)
		if err := firewall.Reconcile(ports); err != nil {
			log.Printf("[puller] firewall reconcile warned: %v", err)
		}
	}

	p.mu.Lock()
	p.current = cfg
	p.mu.Unlock()
	p.etag.Store(cfg.Etag)

	if cfg.Settings.PullInterval > 0 {
		p.period.Store(int64(cfg.Settings.PullInterval))
	}

	if data, err := json.Marshal(cfg); err == nil {
		_ = os.WriteFile(p.cacheFp, data, 0o600)
	}
	log.Printf("[puller] applied config etag=%s inbounds=%d users=%d relays=%d",
		cfg.Etag, len(cfg.Inbounds), len(cfg.Users), len(cfg.Relays))

	// 升级版本对账：检测差异 → 进升级窗口（默认 03:00-05:00）→ 后台执行
	for _, comp := range upgrade.Detect(cfg.Node, p.xray.Version()) {
		if !inUpgradeWindow(time.Now()) {
			continue
		}
		switch comp {
		case "agent":
			go upgrade.PerformAgentUpgrade(cfg.Node.AgentDownloadURL, cfg.Node.AgentSHA256URL, cfg.Node.AgentTarget)
		case "xray":
			// xray 下载源固定走 GitHub release，不依赖 master 分发
			go upgrade.PerformXrayUpgrade(
				"https://github.com/XTLS/Xray-core/releases/download/v"+cfg.Node.XrayTarget+"/Xray-linux-${ARCH}.zip",
				"", cfg.Node.XrayTarget)
		}
	}
	return nil
}

// inUpgradeWindow 限制升级只在凌晨低峰跑（默认 03:00-05:00）
//
// 节点本地时区。后续可由 ServerConfig.Settings 下发参数化。
func inUpgradeWindow(t time.Time) bool {
	h := t.Hour()
	return h >= 3 && h < 5
}
