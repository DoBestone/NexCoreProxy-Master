package service

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"nexcoreproxy-master/internal/model"
)

// AlertService 节点离线检测 + 邮件告警
//
// 触发规则：
//   - 节点 LastSyncAt > 5 分钟 → 视为离线
//   - 同一节点首次离线 → 立即告警；此后每小时最多重复一次（避免轰炸）
//   - 节点恢复 → 发"已恢复"邮件
type AlertService struct {
	email       *EmailService
	mu          sync.Mutex
	alerted     map[uint]time.Time // node_id → 上次告警时间
	offlineSince map[uint]time.Time
}

func NewAlertService(email *EmailService) *AlertService {
	return &AlertService{
		email:        email,
		alerted:      make(map[uint]time.Time),
		offlineSince: make(map[uint]time.Time),
	}
}

// Recipient 告警收件人；首选 ALERT_EMAIL 环境变量，回退第一个 admin
func (s *AlertService) Recipient() string {
	if v := os.Getenv("ALERT_EMAIL"); v != "" {
		return v
	}
	var u model.User
	if err := model.GetDB().Where("role = ?", "admin").Order("id").First(&u).Error; err == nil {
		return u.Email
	}
	return ""
}

// CheckOffline 周期任务（建议 1 分钟一次）
func (s *AlertService) CheckOffline() {
	to := s.Recipient()
	if to == "" || s.email == nil {
		return
	}
	cutoff := time.Now().Add(-5 * time.Minute)
	now := time.Now()

	var nodes []model.Node
	if err := model.GetDB().Where("enable = ?", true).Find(&nodes).Error; err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, n := range nodes {
		offline := n.LastSyncAt == nil || n.LastSyncAt.Before(cutoff)

		if offline {
			if _, was := s.offlineSince[n.ID]; !was {
				s.offlineSince[n.ID] = now
			}
			last, ever := s.alerted[n.ID]
			if !ever || now.Sub(last) >= time.Hour {
				s.sendOfflineMail(to, &n)
				s.alerted[n.ID] = now
				log.Printf("[alert] node %s offline, mail sent to %s", n.Name, to)
			}
		} else {
			// 恢复：之前告过警的发恢复邮件
			if _, ever := s.alerted[n.ID]; ever {
				s.sendRecoveredMail(to, &n)
				delete(s.alerted, n.ID)
				delete(s.offlineSince, n.ID)
				log.Printf("[alert] node %s recovered", n.Name)
			}
		}
	}
}

func (s *AlertService) sendOfflineMail(to string, n *model.Node) {
	last := "未知"
	if n.LastSyncAt != nil {
		last = n.LastSyncAt.Format("2006-01-02 15:04:05")
	}
	body := fmt.Sprintf(`<div style="font-family:system-ui;color:#1e293b;line-height:1.6">
<h3 style="color:#dc2626;margin:0 0 12px">节点离线告警</h3>
<table style="font-size:13px">
<tr><td style="color:#64748b;padding-right:12px">节点名</td><td>%s</td></tr>
<tr><td style="color:#64748b;padding-right:12px">IP</td><td>%s</td></tr>
<tr><td style="color:#64748b;padding-right:12px">类型</td><td>%s</td></tr>
<tr><td style="color:#64748b;padding-right:12px">最后心跳</td><td>%s</td></tr>
</table>
<p style="margin-top:14px;color:#475569">建议立即检查 ncp-agent 与 xray 服务状态。</p>
</div>`, n.Name, n.IP, n.Type, last)
	_ = s.email.Send(to, "[NexCore] 节点离线: "+n.Name, body)
}

func (s *AlertService) sendRecoveredMail(to string, n *model.Node) {
	body := fmt.Sprintf(`<div style="font-family:system-ui;color:#1e293b;line-height:1.6">
<h3 style="color:#16a34a;margin:0 0 12px">节点已恢复</h3>
<p style="font-size:13px">节点 <b>%s</b> (%s) 已恢复在线。</p>
</div>`, n.Name, n.IP)
	_ = s.email.Send(to, "[NexCore] 节点恢复: "+n.Name, body)
}

