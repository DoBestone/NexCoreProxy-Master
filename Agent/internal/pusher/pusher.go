// Package pusher 周期性把流量统计 + 系统快照 push 给 Master，并消费返回的 kicks 名单
//
// Step 4 阶段先打通框架（系统快照 + 空 stats），Step 5 接入 xray Stats gRPC 后填充 stats。
package pusher

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"nexcoreproxy-agent/internal/config"
	"nexcoreproxy-agent/internal/protocol"
	"nexcoreproxy-agent/internal/upgrade"
	"nexcoreproxy-agent/internal/xraymgr"
)

const agentVersion = upgrade.CurrentAgentVersion

// StatsSource 抽象出"拿一批流量增量"的能力，由 statsclient 实现；本包不直接依赖 xray gRPC
type StatsSource interface {
	// Drain 返回自上次调用至今的累计增量并清零，同时给出在线 (email,ip) 列表
	Drain(ctx context.Context) (stats map[string]protocol.TrafficDelta, online map[string][]string, err error)
}

// KickHandler 收到服务端 kicks 名单时被调用；由 main 注入 xray client 删除 client 实现
type KickHandler func(ctx context.Context, emails []string)

// Pusher 单例
type Pusher struct {
	cfg       *config.Config
	client    *protocol.Client
	xray      *xraymgr.Manager
	src       StatsSource
	kick      KickHandler
	period    atomic.Int64
	etagFn    func() string
}

func New(cfg *config.Config, client *protocol.Client, xray *xraymgr.Manager,
	src StatsSource, kick KickHandler, etagFn func() string) *Pusher {
	p := &Pusher{
		cfg:    cfg,
		client: client,
		xray:   xray,
		src:    src,
		kick:   kick,
		etagFn: etagFn,
	}
	p.period.Store(int64(cfg.PushInterval))
	return p
}

// Run 阻塞循环，每 period 秒推一次
func (p *Pusher) Run(ctx context.Context) {
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
				log.Printf("[pusher] tick failed: %v", err)
			}
		}
	}
}

func (p *Pusher) tick(ctx context.Context) error {
	var (
		stats  map[string]protocol.TrafficDelta
		online map[string][]string
	)
	if p.src != nil {
		s, o, err := p.src.Drain(ctx)
		if err != nil {
			log.Printf("[pusher] drain stats failed: %v", err)
		} else {
			stats = s
			online = o
		}
	}

	payload := &protocol.PushRequest{
		EtagApplied: p.etagFn(),
		Stats:       stats,
		Online:      online,
		System: &protocol.SystemSnapshot{
			XrayVersion:  p.xray.Version(),
			AgentVersion: agentVersion,
		},
	}
	resp, err := p.client.PushReport(ctx, payload)
	if err != nil {
		return err
	}
	if len(resp.Kicks) > 0 && p.kick != nil {
		p.kick(ctx, resp.Kicks)
	}
	if resp.CurrentEtag != "" && resp.CurrentEtag != p.etagFn() {
		log.Printf("[pusher] master etag advanced (%s → %s), puller will catch up next tick",
			p.etagFn(), resp.CurrentEtag)
	}
	return nil
}
