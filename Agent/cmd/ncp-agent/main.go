package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"nexcoreproxy-agent/internal/config"
	"nexcoreproxy-agent/internal/protocol"
	"nexcoreproxy-agent/internal/puller"
	"nexcoreproxy-agent/internal/pusher"
	"nexcoreproxy-agent/internal/statsclient"
	"nexcoreproxy-agent/internal/xraymgr"
)

var (
	cfgPath = flag.String("config", "/etc/ncp-agent/agent.yaml", "path to agent config yaml")
)

func main() {
	flag.Parse()
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("[main] load config: %v", err)
	}
	log.Printf("[main] starting ncp-agent node_id=%d master=%s xray=%s",
		cfg.NodeID, cfg.MasterURL, cfg.XrayBin)

	client := protocol.NewClient(cfg.MasterURL, cfg.NodeToken)
	xray := xraymgr.New(cfg.XrayBin, cfg.XrayConfigPath, cfg.XrayService, cfg.DryRun)

	pull, err := puller.New(cfg, client, xray)
	if err != nil {
		log.Fatalf("[main] init puller: %v", err)
	}

	stats := statsclient.New(cfg.XrayBin, cfg.XrayAPIPort)
	// kicks 处理：Phase 1 简单实现 — 让 puller 立即重拉 etag（Master 已在踢人时 bump etag）
	// 即时 client 删除走 Phase 2 通过 xray HandlerService gRPC 实现，避免在这里再开 gRPC client。
	kickHandler := func(ctx context.Context, emails []string) {
		log.Printf("[main] received %d kicks from master, will refresh on next pull", len(emails))
	}
	push := pusher.New(cfg, client, xray, stats, kickHandler, pull.Etag)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Printf("[main] received signal, shutting down")
		cancel()
	}()

	go pull.Run(ctx)
	go push.Run(ctx)

	<-ctx.Done()
	log.Printf("[main] exited")
}
