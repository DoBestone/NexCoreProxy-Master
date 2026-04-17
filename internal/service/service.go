package service

import (
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

// Services 服务集合
type Services struct {
	Node         *NodeService
	User         *UserService
	Email        *EmailService
	Agent        *AgentManager
	AgentConfig  *AgentConfigService
	AgentPush    *AgentPushService
	Inbound      *InboundService
	RelayBinding *RelayBindingService
	RelaySyncer  *RelaySyncer
	Subscription *SubscriptionService
	RelayHealth  *RelayHealthChecker
	Provisioner  *NodeProvisioner
	Cert         *CertService
	Backup       *BackupService
	Alert        *AlertService
	Cron         *cron.Cron
}

// NewServices 创建服务集合
func NewServices() *Services {
	syncer := NewRelaySyncer()
	inbound := NewInboundService()
	inbound.AttachSyncer(syncer)
	provisioner := NewNodeProvisioner(inbound)
	nodeSvc := NewNodeService()
	nodeSvc.AttachProvisioner(provisioner)
	emailSvc := NewEmailService()
	s := &Services{
		Node:         nodeSvc,
		User:         NewUserService(),
		Email:        emailSvc,
		Agent:        NewAgentManager(),
		AgentConfig:  NewAgentConfigService(),
		AgentPush:    NewAgentPushService(),
		Inbound:      inbound,
		RelayBinding: NewRelayBindingService(syncer),
		RelaySyncer:  syncer,
		Subscription: NewSubscriptionService(),
		RelayHealth:  NewRelayHealthChecker(),
		Provisioner:  provisioner,
		Cert:         NewCertService(),
		Backup:       NewBackupService(),
		Alert:        NewAlertService(emailSvc),
		Cron:         cron.New(cron.WithLocation(time.UTC)),
	}
	return s
}

// StartCron 启动定时任务
func (s *Services) StartCron() {
	register := func(spec, name string, fn func()) {
		if _, err := s.Cron.AddFunc(spec, fn); err != nil {
			log.Printf("[Cron] register %s failed: %v", name, err)
		}
	}
	register("@every 30s", "node sync (legacy)", func() { s.Node.SyncAll() })
	register("@every 60s", "relay health check", func() { s.RelayHealth.CheckAll() })
	register("@every 5m", "expire scan", ScanExpired)
	register("@every 5m", "stale online IP cleanup", CleanStaleOnlineIPs)
	// 每天 00:05 跑月流量重置（5 字段：分 时 日 月 周）
	register("5 0 * * *", "monthly traffic reset", MonthlyReset)
	// 每天 03:00 扫描即将过期的证书并续签
	register("0 3 * * *", "acme renew", func() { s.Cert.RenewExpiring() })
	// 每天 02:00 跑数据库备份
	register("0 2 * * *", "db backup", func() { s.Backup.RunOnce() })
	// 每分钟检查节点离线 → 邮件告警
	register("@every 1m", "offline alert", func() { s.Alert.CheckOffline() })
	s.Cron.Start()
}

// StopCron 停止定时任务
func (s *Services) StopCron() {
	s.Cron.Stop()
}
