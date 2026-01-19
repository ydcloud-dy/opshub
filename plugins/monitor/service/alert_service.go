package service

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"time"
)

// AlertService 告警服务
type AlertService struct {
	// 可以注入数据库连接等依赖
}

// NewAlertService 创建告警服务
func NewAlertService() *AlertService {
	return &AlertService{}
}

// AlertMessage 告警消息
type AlertMessage struct {
	AlertType   string `json:"alertType"`   // 告警类型
	Domain      string `json:"domain"`      // 域名
	Status      string `json:"status"`      // 状态
	Message     string `json:"message"`     // 消息内容
	ResponseTime int    `json:"responseTime"` // 响应时间
	SSLExpiry   string `json:"sslExpiry"`   // SSL过期时间
	Timestamp   string `json:"timestamp"`   // 时间戳
}

// AlertChannelConfig 告警通道配置
type AlertChannelConfig struct {
	// 邮件配置
	SMTPHost     string `json:"smtpHost"`
	SMTPPort     int    `json:"smtpPort"`
	SMTPUser     string `json:"smtpUser"`
	SMTPPassword string `json:"smtpPassword"`
	FromEmail    string `json:"fromEmail"`
	FromName     string `json:"fromName"`

	// Webhook配置
	WebhookURL string `json:"webhookUrl"`

	// 企业微信配置
	WeChatWebhook string `json:"wechatWebhook"`

	// 钉钉配置
	DingTalkWebhook string `json:"dingtalkWebhook"`
	DingTalkSecret  string `json:"dingtalkSecret"`

	// 飞书配置
	FeishuWebhook string `json:"feishuWebhook"`
}

// SendAlert 发送告警
func (s *AlertService) SendAlert(message AlertMessage, config AlertChannelConfig, receivers []string) error {
	var errors []error

	// 并发发送各种通知
	if len(receivers) > 0 {
		// 邮件通知
		if err := s.sendEmail(message, config, receivers); err != nil {
			errors = append(errors, fmt.Errorf("邮件发送失败: %w", err))
		}

		// 企业微信通知
		if err := s.sendWeChat(message, config); err != nil {
			errors = append(errors, fmt.Errorf("企业微信发送失败: %w", err))
		}

		// 钉钉通知
		if err := s.sendDingTalk(message, config); err != nil {
			errors = append(errors, fmt.Errorf("钉钉发送失败: %w", err))
		}

		// 飞书通知
		if err := s.sendFeishu(message, config); err != nil {
			errors = append(errors, fmt.Errorf("飞书发送失败: %w", err))
		}

		// Webhook通知
		if err := s.sendWebhook(message, config); err != nil {
			errors = append(errors, fmt.Errorf("Webhook发送失败: %w", err))
		}
	}

	// 如果所有发送都失败，返回错误
	if len(errors) > 0 && len(errors) == len(receivers)+3 { // +3是wechat/dingtalk/feishu
		return fmt.Errorf("部分或全部告警发送失败: %v", errors)
	}

	return nil
}

// sendEmail 发送邮件
func (s *AlertService) sendEmail(message AlertMessage, config AlertChannelConfig, receivers []string) error {
	if config.SMTPHost == "" || len(receivers) == 0 {
		return fmt.Errorf("邮件配置不完整")
	}

	// 构建邮件内容
	subject := fmt.Sprintf("[域名监控告警] %s - %s", message.Domain, message.getAlertTitle())
	body := s.buildEmailBody(message)

	// 设置SMTP认证
	auth := smtp.PlainAuth("", config.SMTPUser, config.SMTPPassword, config.SMTPHost)

	// 构建邮件头
	from := fmt.Sprintf("%s <%s>", config.FromName, config.FromEmail)
	to := receivers[0]
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	// 构建邮件内容
	var msg string
	for k, v := range headers {
		msg += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	msg += "\r\n" + body

	// 发送邮件
	addr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)
	err := smtp.SendMail(addr, auth, config.FromEmail, receivers, []byte(msg))
	if err != nil {
		return err
	}

	return nil
}

// buildEmailBody 构建邮件内容
func (s *AlertService) buildEmailBody(message AlertMessage) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #f44336; color: white; padding: 20px; text-align: center; }
        .content { background: #f9f9f9; padding: 20px; border-radius: 5px; margin-top: 20px; }
        .info-item { margin: 10px 0; padding: 10px; background: white; border-left: 4px solid #f44336; }
        .footer { text-align: center; margin-top: 20px; color: #999; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>域名监控告警</h2>
        </div>
        <div class="content">
            <h3>%s</h3>
            <div class="info-item"><strong>域名:</strong> %s</div>
            <div class="info-item"><strong>状态:</strong> %s</div>
            <div class="info-item"><strong>消息:</strong> %s</div>
            <div class="info-item"><strong>时间:</strong> %s</div>
            %s
        </div>
        <div class="footer">
            <p>此邮件由系统自动发送，请勿回复。</p>
        </div>
    </div>
</body>
</html>
`, message.getAlertTitle(), message.Domain, message.getStatusText(), message.Message, message.Timestamp, s.getDetailInfo(message))
}

// sendWeChat 发送企业微信通知
func (s *AlertService) sendWeChat(message AlertMessage, config AlertChannelConfig) error {
	if config.WeChatWebhook == "" {
		return fmt.Errorf("企业微信Webhook未配置")
	}

	content := fmt.Sprintf("**域名监控告警**\n\n**域名**: %s\n**状态**: %s\n**消息**: %s\n**时间**: %s",
		message.Domain, message.getStatusText(), message.Message, message.Timestamp)

	webhookData := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": content,
		},
	}

	return s.sendWebhookRequest(config.WeChatWebhook, webhookData)
}

// sendDingTalk 发送钉钉通知
func (s *AlertService) sendDingTalk(message AlertMessage, config AlertChannelConfig) error {
	if config.DingTalkWebhook == "" {
		return fmt.Errorf("钉钉Webhook未配置")
	}

	content := fmt.Sprintf("## 域名监控告警\n\n**域名**: %s\n**状态**: %s\n**消息**: %s\n**时间**: %s",
		message.Domain, message.getStatusText(), message.Message, message.Timestamp)

	webhookData := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"title": message.getAlertTitle(),
			"text":  content,
		},
	}

	return s.sendWebhookRequest(config.DingTalkWebhook, webhookData)
}

// sendFeishu 发送飞书通知
func (s *AlertService) sendFeishu(message AlertMessage, config AlertChannelConfig) error {
	if config.FeishuWebhook == "" {
		return fmt.Errorf("飞书Webhook未配置")
	}

	webhookData := map[string]interface{}{
		"msg_type": "text",
		"content":  struct {
			Text string `json:"text"`
		}{
			Text: message.getAlertTitle() + "\n\n域名: " + message.Domain + "\n状态: " + message.getStatusText() + "\n消息: " + message.Message + "\n时间: " + message.Timestamp,
		},
	}

	return s.sendWebhookRequest(config.FeishuWebhook, webhookData)
}

// sendWebhook 发送自定义Webhook
func (s *AlertService) sendWebhook(message AlertMessage, config AlertChannelConfig) error {
	if config.WebhookURL == "" {
		return fmt.Errorf("Webhook URL未配置")
	}

	webhookData := map[string]interface{}{
		"alertType":    message.AlertType,
		"domain":       message.Domain,
		"status":       message.Status,
		"message":      message.Message,
		"responseTime": message.ResponseTime,
		"sslExpiry":    message.SSLExpiry,
		"timestamp":    message.Timestamp,
	}

	return s.sendWebhookRequest(config.WebhookURL, webhookData)
}

// sendWebhookRequest 发送Webhook请求
func (s *AlertService) sendWebhookRequest(url string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// 创建HTTP客户端，跳过SSL验证
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Webhook返回错误状态: %d", resp.StatusCode)
	}

	return nil
}

// 辅助方法
func (m AlertMessage) getAlertTitle() string {
	switch m.AlertType {
	case "domain_down":
		return "域名无法访问"
	case "high_response_time":
		return "响应时间过高"
	case "ssl_expiring":
		return "SSL证书即将过期"
	case "ssl_expired":
		return "SSL证书已过期"
	case "ssl_invalid":
		return "SSL证书无效"
	default:
		return "域名监控告警"
	}
}

func (m AlertMessage) getStatusText() string {
	switch m.Status {
	case "normal":
		return "正常"
	case "abnormal":
		return "异常"
	default:
		return "未知"
	}
}

func (s *AlertService) getDetailInfo(message AlertMessage) string {
	var detail string
	if message.ResponseTime > 0 {
		detail += fmt.Sprintf(`<div class="info-item"><strong>响应时间:</strong> %d ms</div>`, message.ResponseTime)
	}
	if message.SSLExpiry != "" {
		detail += fmt.Sprintf(`<div class="info-item"><strong>SSL过期时间:</strong> %s</div>`, message.SSLExpiry)
	}
	return detail
}
