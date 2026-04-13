package service

import (
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

// Services 服务集合
type Services struct {
	Node  *NodeService
	User  *UserService
	Email *EmailService
	Agent *AgentManager
	Cron  *cron.Cron
}

// NewServices 创建服务集合
func NewServices() *Services {
	s := &Services{
		Node:  NewNodeService(),
		User:  NewUserService(),
		Email: NewEmailService(),
		Agent: NewAgentManager(),
		Cron:  cron.New(cron.WithLocation(time.UTC)),
	}
	return s
}

// StartCron 启动定时任务
func (s *Services) StartCron() {
	// 每 30 秒同步所有节点状态
	_, err := s.Cron.AddFunc("@every 30s", func() {
		s.Node.SyncAll()
	})
	if err != nil {
		log.Printf("[Cron] 注册节点同步任务失败: %v", err)
		return
	}
	s.Cron.Start()
}

// StopCron 停止定时任务
func (s *Services) StopCron() {
	s.Cron.Stop()
}
