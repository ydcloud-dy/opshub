// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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

// ReceiverInfo 接收人信息（用于告警发送）
type ReceiverInfo struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	FeishuID  string `json:"feishuId"`
	DingTalkID string `json:"dingtalkId"`
	WeChatID  string `json:"wechatId"`
}

// ReceiverChannelRelation 接收人与通道的关联信息
type ReceiverChannelRelation struct {
	ReceiverID    uint          `json:"receiverId"`
	ChannelID     uint          `json:"channelId"`
	ChannelType   string        `json:"channelType"`
	Receiver      ReceiverInfo  `json:"receiver"`
	ChannelConfig string        `json:"channelConfig"` // 关联时的特定配置（如@信息）
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

// SendAlert 发送告警（支持接收人-通道关联）
// 如果receiverChannels不为空，使用接收人-通道关联关系发送；否则使用旧的方式发送
func (s *AlertService) SendAlert(message AlertMessage, config AlertChannelConfig, receivers []string, receiverChannels ...[]ReceiverChannelRelation) error {
	var errors []error
	var successCount int

	// 如果提供了接收人-通道关联信息，使用新的发送方式
	if len(receiverChannels) > 0 && len(receiverChannels[0]) > 0 {
		return s.sendAlertWithReceiverChannels(message, config, receiverChannels[0])
	}

	// 否则使用旧的方式发送（向后兼容）
	// 邮件通知
	if config.SMTPHost != "" && len(receivers) > 0 {
		if err := s.sendEmail(message, config, receivers); err != nil {
			errors = append(errors, fmt.Errorf("邮件发送失败: %w", err))
		} else {
			successCount++
		}
	}

	// 企业微信通知
	if config.WeChatWebhook != "" {
		if err := s.sendWeChat(message, config, nil); err != nil {
			errors = append(errors, fmt.Errorf("企业微信发送失败: %w", err))
		} else {
			successCount++
		}
	}

	// 钉钉通知
	if config.DingTalkWebhook != "" {
		if err := s.sendDingTalk(message, config, nil); err != nil {
			errors = append(errors, fmt.Errorf("钉钉发送失败: %w", err))
		} else {
			successCount++
		}
	}

	// 飞书通知
	if config.FeishuWebhook != "" {
		if err := s.sendFeishu(message, config, nil); err != nil {
			errors = append(errors, fmt.Errorf("飞书发送失败: %w", err))
		} else {
			successCount++
		}
	}

	// Webhook通知
	if config.WebhookURL != "" {
		if err := s.sendWebhook(message, config); err != nil {
			errors = append(errors, fmt.Errorf("Webhook发送失败: %w", err))
		} else {
			successCount++
		}
	}

	// 如果所有发送都失败，返回错误
	if successCount == 0 && len(errors) > 0 {
		return fmt.Errorf("所有告警通道发送失败: %v", errors)
	}

	return nil
}

// sendAlertWithReceiverChannels 基于接收人-通道关联发送告警
func (s *AlertService) sendAlertWithReceiverChannels(message AlertMessage, config AlertChannelConfig, relations []ReceiverChannelRelation) error {
	var errors []error
	var successCount int

	// 按通道类型分组接收人
	emailReceivers := make([]string, 0)
	feishuReceivers := make([]ReceiverInfo, 0)
	dingtalkReceivers := make([]ReceiverInfo, 0)
	wechatReceivers := make([]ReceiverInfo, 0)

	for _, relation := range relations {
		switch relation.ChannelType {
		case "email":
			if relation.Receiver.Email != "" {
				emailReceivers = append(emailReceivers, relation.Receiver.Email)
			}
		case "feishu":
			if relation.Receiver.FeishuID != "" {
				feishuReceivers = append(feishuReceivers, relation.Receiver)
			}
		case "dingtalk":
			if relation.Receiver.DingTalkID != "" {
				dingtalkReceivers = append(dingtalkReceivers, relation.Receiver)
			}
		case "wechat":
			if relation.Receiver.WeChatID != "" {
				wechatReceivers = append(wechatReceivers, relation.Receiver)
			}
		}
	}

	// 发送邮件
	if config.SMTPHost != "" && len(emailReceivers) > 0 {
		if err := s.sendEmail(message, config, emailReceivers); err != nil {
			errors = append(errors, fmt.Errorf("邮件发送失败: %w", err))
		} else {
			successCount++
		}
	}

	// 发送飞书
	if config.FeishuWebhook != "" && len(feishuReceivers) > 0 {
		if err := s.sendFeishu(message, config, &feishuReceivers); err != nil {
			errors = append(errors, fmt.Errorf("飞书发送失败: %w", err))
		} else {
			successCount++
		}
	}

	// 发送钉钉
	if config.DingTalkWebhook != "" && len(dingtalkReceivers) > 0 {
		if err := s.sendDingTalk(message, config, &dingtalkReceivers); err != nil {
			errors = append(errors, fmt.Errorf("钉钉发送失败: %w", err))
		} else {
			successCount++
		}
	}

	// 发送企业微信
	if config.WeChatWebhook != "" && len(wechatReceivers) > 0 {
		if err := s.sendWeChat(message, config, &wechatReceivers); err != nil {
			errors = append(errors, fmt.Errorf("企业微信发送失败: %w", err))
		} else {
			successCount++
		}
	}

	// 如果所有发送都失败，返回错误
	if successCount == 0 && len(errors) > 0 {
		return fmt.Errorf("所有告警通道发送失败: %v", errors)
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
func (s *AlertService) sendWeChat(message AlertMessage, config AlertChannelConfig, receivers *[]ReceiverInfo) error {
	if config.WeChatWebhook == "" {
		return fmt.Errorf("企业微信Webhook未配置")
	}

	content := fmt.Sprintf("**域名监控告警**\n\n**域名**: %s\n**状态**: %s\n**消息**: %s\n**时间**: %s",
		message.Domain, message.getStatusText(), message.Message, message.Timestamp)

	// 如果提供了接收人，添加@提醒
	if receivers != nil && len(*receivers) > 0 {
		content += "\n\n**告知对象**: "
		for i, receiver := range *receivers {
			if i > 0 {
				content += ", "
			}
			content += "<@" + receiver.WeChatID + ">"
		}
	}

	webhookData := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": content,
		},
	}

	return s.sendWebhookRequest(config.WeChatWebhook, webhookData)
}

// sendDingTalk 发送钉钉通知
func (s *AlertService) sendDingTalk(message AlertMessage, config AlertChannelConfig, receivers *[]ReceiverInfo) error {
	if config.DingTalkWebhook == "" {
		return fmt.Errorf("钉钉Webhook未配置")
	}

	content := fmt.Sprintf("## 域名监控告警\n\n**域名**: %s\n**状态**: %s\n**消息**: %s\n**时间**: %s",
		message.Domain, message.getStatusText(), message.Message, message.Timestamp)

	// 如果提供了接收人，添加@提醒
	var atMobiles []string
	if receivers != nil && len(*receivers) > 0 {
		content += "\n\n**告知对象**: "
		for i, receiver := range *receivers {
			if i > 0 {
				content += ", "
			}
			content += "@" + receiver.DingTalkID
			if receiver.Phone != "" {
				atMobiles = append(atMobiles, receiver.Phone)
			}
		}
	}

	webhookData := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"title": message.getAlertTitle(),
			"text":  content,
		},
	}

	// 添加@提醒信息
	if len(atMobiles) > 0 {
		webhookData["at"] = map[string]interface{}{
			"atMobiles": atMobiles,
			"isAtAll":   false,
		}
	}

	return s.sendWebhookRequest(config.DingTalkWebhook, webhookData)
}

// sendFeishu 发送飞书通知
func (s *AlertService) sendFeishu(message AlertMessage, config AlertChannelConfig, receivers *[]ReceiverInfo) error {
	if config.FeishuWebhook == "" {
		return fmt.Errorf("飞书Webhook未配置")
	}

	// 构建富文本内容
	textElements := []map[string]interface{}{
		{"tag": "text", "text": fmt.Sprintf("域名: %s\n", message.Domain)},
		{"tag": "text", "text": fmt.Sprintf("状态: %s\n", message.getStatusText())},
		{"tag": "text", "text": fmt.Sprintf("消息: %s\n", message.Message)},
		{"tag": "text", "text": fmt.Sprintf("时间: %s", message.Timestamp)},
	}

	// 如果有响应时间，添加该信息
	if message.ResponseTime > 0 {
		textElements = append(textElements, map[string]interface{}{
			"tag": "text", "text": fmt.Sprintf("\n响应时间: %d ms", message.ResponseTime),
		})
	}

	// 如果有SSL过期时间，添加该信息
	if message.SSLExpiry != "" {
		textElements = append(textElements, map[string]interface{}{
			"tag": "text", "text": fmt.Sprintf("\nSSL过期时间: %s", message.SSLExpiry),
		})
	}

	// 如果提供了接收人，添加@提醒
	if receivers != nil && len(*receivers) > 0 {
		textElements = append(textElements, map[string]interface{}{
			"tag": "text", "text": "\n\n告知对象: ",
		})
		for _, receiver := range *receivers {
			if receiver.FeishuID != "" {
				textElements = append(textElements, map[string]interface{}{
					"tag": "at",
					"user_id": receiver.FeishuID,
				})
				textElements = append(textElements, map[string]interface{}{
					"tag": "text",
					"text": " ",
				})
			}
		}
	}

	webhookData := map[string]interface{}{
		"msg_type": "post",
		"content": map[string]interface{}{
			"post": map[string]interface{}{
				"zh_cn": map[string]interface{}{
					"title":   message.getAlertTitle(),
					"content": [][]map[string]interface{}{textElements},
				},
			},
		},
	}

	jsonData, _ := json.MarshalIndent(webhookData, "", "  ")
	_ = jsonData // Keep marshaled data for webhook

	err := s.sendWebhookRequest(config.FeishuWebhook, webhookData)
	return err
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
		return fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体以获取更多信息
	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	if n > 0 {
		var respData map[string]interface{}
		if err := json.Unmarshal(body[:n], &respData); err == nil {
			// 飞书返回的格式: {"code": 0, "msg": "success"}
			if code, ok := respData["code"].(float64); ok && code != 0 {
				return fmt.Errorf("API返回错误: code=%v, msg=%v", code, respData["msg"])
			}
		}
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Webhook返回错误状态: %d, body: %s", resp.StatusCode, string(body[:n]))
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
