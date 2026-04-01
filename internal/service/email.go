package service

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"os"
)

// EmailConfig 邮件配置
type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
	UseTLS   bool
}

// EmailService 邮件服务
type EmailService struct {
	config *EmailConfig
}

// NewEmailService 创建邮件服务
func NewEmailService() *EmailService {
	return &EmailService{
		config: &EmailConfig{
			Host:     os.Getenv("SMTP_HOST"),
			Port:     587,
			Username: os.Getenv("SMTP_USER"),
			Password: os.Getenv("SMTP_PASS"),
			From:     os.Getenv("SMTP_FROM"),
			FromName: os.Getenv("SMTP_FROM_NAME"),
			UseTLS:   true,
		},
	}
}

// LoadConfig 从数据库加载配置
func (s *EmailService) LoadConfig(host string, port int, username, password, from, fromName string, useTLS bool) {
	s.config = &EmailConfig{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
		FromName: fromName,
		UseTLS:   useTLS,
	}
}

// Send 发送邮件
func (s *EmailService) Send(to, subject, body string) error {
	if s.config.Host == "" {
		return fmt.Errorf("邮件服务未配置")
	}

	from := s.config.From
	if from == "" {
		from = s.config.Username
	}

	// 构建邮件头
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", s.config.FromName, from)
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	// 组装邮件
	var msg bytes.Buffer
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(body)

	// 发送邮件
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	var auth smtp.Auth
	if s.config.Username != "" {
		auth = smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	}

	if s.config.UseTLS {
		return s.sendWithTLS(addr, auth, from, []string{to}, msg.Bytes())
	}

	return smtp.SendMail(addr, auth, from, []string{to}, msg.Bytes())
}

// sendWithTLS 使用TLS发送邮件
func (s *EmailService) sendWithTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         s.config.Host,
	})
	if err != nil {
		return fmt.Errorf("TLS连接失败: %v", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("创建SMTP客户端失败: %v", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP认证失败: %v", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("设置发件人失败: %v", err)
	}

	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("设置收件人失败: %v", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("准备邮件数据失败: %v", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("写入邮件内容失败: %v", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("关闭邮件写入失败: %v", err)
	}

	return client.Quit()
}

// SendWithTemplate 使用模板发送邮件
func (s *EmailService) SendWithTemplate(to, subject, templateName string, data interface{}) error {
	tmpl, err := s.getTemplate(templateName)
	if err != nil {
		return err
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("渲染模板失败: %v", err)
	}

	return s.Send(to, subject, body.String())
}

// getTemplate 获取邮件模板
func (s *EmailService) getTemplate(name string) (*template.Template, error) {
	templates := map[string]string{
		"welcome": `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f8fafc; padding: 40px;">
<div style="max-width: 600px; margin: 0 auto; background: white; border-radius: 16px; padding: 40px; box-shadow: 0 4px 24px rgba(0,0,0,0.08);">
<div style="text-align: center; margin-bottom: 32px;">
<div style="width: 56px; height: 56px; background: linear-gradient(135deg, #1677ff, #4096ff); border-radius: 14px; margin: 0 auto 16px; display: flex; align-items: center; justify-content: center;">
<svg width="28" height="28" viewBox="0 0 24 24" fill="none"><path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="white" stroke-width="2"/><path d="M2 17L12 22L22 17" stroke="white" stroke-width="2"/><path d="M2 12L12 17L22 12" stroke="white" stroke-width="2"/></svg>
</div>
<h1 style="color: #1f1f1f; font-size: 24px; margin: 0;">欢迎注册</h1>
</div>
<p style="color: #595959; font-size: 16px; line-height: 1.6;">亲爱的 <strong>{{.Username}}</strong>，</p>
<p style="color: #595959; font-size: 16px; line-height: 1.6;">感谢您注册 NexCore 代理主机！您的账号已创建成功。</p>
<div style="background: #f8fafc; border-radius: 12px; padding: 24px; margin: 24px 0;">
<p style="color: #8c8c8c; font-size: 14px; margin: 0 0 8px;">您的账号信息：</p>
<p style="color: #262626; font-size: 16px; margin: 0;">用户名：<strong>{{.Username}}</strong></p>
</div>
<p style="color: #8c8c8c; font-size: 14px;">如有任何问题，请随时联系客服。</p>
<div style="text-align: center; margin-top: 32px; padding-top: 24px; border-top: 1px solid #f0f0f0;">
<p style="color: #bfbfbf; font-size: 12px; margin: 0;">NexCore 代理主机 © 2026</p>
</div>
</div>
</body>
</html>`,
		"order": `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f8fafc; padding: 40px;">
<div style="max-width: 600px; margin: 0 auto; background: white; border-radius: 16px; padding: 40px; box-shadow: 0 4px 24px rgba(0,0,0,0.08);">
<div style="text-align: center; margin-bottom: 32px;">
<h1 style="color: #1f1f1f; font-size: 24px; margin: 0;">订单确认</h1>
</div>
<p style="color: #595959; font-size: 16px; line-height: 1.6;">尊敬的用户，</p>
<p style="color: #595959; font-size: 16px; line-height: 1.6;">您的订单已创建成功！</p>
<div style="background: #f8fafc; border-radius: 12px; padding: 24px; margin: 24px 0;">
<p style="color: #262626; font-size: 16px; margin: 0 0 12px;">订单号：<strong>{{.OrderNo}}</strong></p>
<p style="color: #262626; font-size: 16px; margin: 0 0 12px;">套餐：<strong>{{.PackageName}}</strong></p>
<p style="color: #ff4d4f; font-size: 20px; margin: 0;">金额：${{.Amount}}</p>
</div>
<p style="color: #8c8c8c; font-size: 14px;">请及时完成支付，如有问题请联系客服。</p>
<div style="text-align: center; margin-top: 32px; padding-top: 24px; border-top: 1px solid #f0f0f0;">
<p style="color: #bfbfbf; font-size: 12px; margin: 0;">NexCore 代理主机 © 2026</p>
</div>
</div>
</body>
</html>`,
		"ticket_reply": `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f8fafc; padding: 40px;">
<div style="max-width: 600px; margin: 0 auto; background: white; border-radius: 16px; padding: 40px; box-shadow: 0 4px 24px rgba(0,0,0,0.08);">
<div style="text-align: center; margin-bottom: 32px;">
<h1 style="color: #1f1f1f; font-size: 24px; margin: 0;">工单回复</h1>
</div>
<p style="color: #595959; font-size: 16px; line-height: 1.6;">尊敬的用户，</p>
<p style="color: #595959; font-size: 16px; line-height: 1.6;">您的工单有了新的回复。</p>
<div style="background: #f8fafc; border-radius: 12px; padding: 24px; margin: 24px 0;">
<p style="color: #8c8c8c; font-size: 14px; margin: 0 0 8px;">工单主题：</p>
<p style="color: #262626; font-size: 16px; margin: 0 0 16px;"><strong>{{.Subject}}</strong></p>
<p style="color: #8c8c8c; font-size: 14px; margin: 0 0 8px;">回复内容：</p>
<p style="color: #595959; font-size: 15px; margin: 0; line-height: 1.6;">{{.Content}}</p>
</div>
<p style="color: #8c8c8c; font-size: 14px;">请登录系统查看详情。</p>
<div style="text-align: center; margin-top: 32px; padding-top: 24px; border-top: 1px solid #f0f0f0;">
<p style="color: #bfbfbf; font-size: 12px; margin: 0;">NexCore 代理主机 © 2026</p>
</div>
</div>
</body>
</html>`,
	}

	content, ok := templates[name]
	if !ok {
		return nil, fmt.Errorf("模板不存在: %s", name)
	}

	return template.New(name).Parse(content)
}

// IsConfigured 检查是否已配置
func (s *EmailService) IsConfigured() bool {
	return s.config != nil && s.config.Host != ""
}
