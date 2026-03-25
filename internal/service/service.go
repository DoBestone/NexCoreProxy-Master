package service

import (
	"time"

	"github.com/robfig/cron/v3"
)

// Services 服务集合
type Services struct {
	Node    *NodeService
	User    *UserService
	Cron    *cron.Cron
}

// NewServices 创建服务集合
func NewServices() *Services {
	s := &Services{
		Node: NewNodeService(),
		User: NewUserService(),
		Cron: cron.New(cron.WithLocation(time.UTC)),
	}
	return s
}

// StartCron 启动定时任务
func (s *Services) StartCron() {
	// 每 30 秒同步所有节点状态
	s.Cron.AddFunc("@every 30s", func() {
		s.Node.SyncAll()
	})
	s.Cron.Start()
}

// StopCron 停止定时任务
func (s *Services) StopCron() {
	s.Cron.Stop()
}