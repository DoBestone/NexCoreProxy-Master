package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"time"
)

// EmailService 邮件服务 (基于 SMTP Lite API)
type EmailService struct {
	apiURL   string // SMTP Lite API 地址
	apiKey   string // API Key
	fromName string // 发件人名称
}

// NewEmailService 创建邮件服务
func NewEmailService() *EmailService {
	return &EmailService{
		apiURL:   os.Getenv("SMTP_API_URL"),
		apiKey:   os.Getenv("SMTP_API_KEY"),
		fromName: os.Getenv("SMTP_FROM_NAME"),
	}
}

// LoadConfig 从数据库加载配置
func (s *EmailService) LoadConfig(apiURL, apiKey, fromName string) {
	s.apiURL = apiURL
	s.apiKey = apiKey
	s.fromName = fromName
}

// sendRequest SMTP Lite API 请求体
type sendRequest struct {
	To       string `json:"to"`
	Subject  string `json:"subject"`
	Body     string `json:"body"`
	IsHTML   bool   `json:"is_html"`
	FromName string `json:"from_name,omitempty"`
}

// sendResponse SMTP Lite API 响应体
type sendResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	UsedSMTP string `json:"used_smtp"`
}

// Send 通过 SMTP Lite API 发送邮件
func (s *EmailService) Send(to, subject, body string) error {
	return s.sendMail(to, subject, body, true)
}

// SendText 发送纯文本邮件
func (s *EmailService) SendText(to, subject, body string) error {
	return s.sendMail(to, subject, body, false)
}

func (s *EmailService) sendMail(to, subject, body string, isHTML bool) error {
	if s.apiURL == "" || s.apiKey == "" {
		return fmt.Errorf("邮件服务未配置")
	}

	reqBody := sendRequest{
		To:       to,
		Subject:  subject,
		Body:     body,
		IsHTML:   isHTML,
		FromName: s.fromName,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %v", err)
	}

	url := s.apiURL + "/api/v1/send"
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", s.apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求 SMTP Lite API 失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SMTP Lite API 返回 %d: %s", resp.StatusCode, string(respBody))
	}

	var result sendResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	if !result.Success {
		return fmt.Errorf("发送失败: %s", result.Message)
	}

	return nil
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
		"welcome": `<!DOCTYPE html>
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
		"order": `<!DOCTYPE html>
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
		"ticket_reply": `<!DOCTYPE html>
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
	return s.apiURL != "" && s.apiKey != ""
}
