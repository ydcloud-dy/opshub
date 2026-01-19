package model

import (
	"time"
)

// AlertConfig 告警配置
type AlertConfig struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	Name            string    `gorm:"type:varchar(100);not null" json:"name"`                  // 告警配置名称
	AlertType       string    `gorm:"type:varchar(20);not null" json:"alertType"`            // 告警类型: domain_down, high_response_time, ssl_expiring, ssl_expired, ssl_invalid
	Enabled         bool      `gorm:"type:tinyint(1);default:1" json:"enabled"`               // 是否启用

	// 触发条件配置
	Threshold       *int      `gorm:"type:int" json:"threshold"`                              // 阈值(如响应时间ms、过期天数)
	DomainMonitorID *uint     `gorm:"index" json:"domainMonitorId"`                           // 关联的域名监控ID，为空表示全局配置

	// 告警方式配置
	EnableEmail     bool      `gorm:"type:tinyint(1);default:0" json:"enableEmail"`           // 邮件通知
	EnableWebhook   bool      `gorm:"type:tinyint(1);default:0" json:"enableWebhook"`         // Webhook通知
	EnableWeChat    bool      `gorm:"type:tinyint(1);default:0" json:"enableWeChat"`          // 企业微信通知
	EnableDingTalk  bool      `gorm:"type:tinyint(1);default:0" json:"enableDingTalk"`         // 钉钉通知
	EnableFeishu    bool      `gorm:"type:tinyint(1);default:0" json:"enableFeishu"`           // 飞书通知
	EnableSystemMsg bool      `gorm:"type:tinyint(1);default:0" json:"enableSystemMsg"`       // 系统内消息

	// 告警频率控制
	AlertInterval  int       `gorm:"type:int;default:600" json:"alertInterval"`               // 告警间隔(秒)，默认10分钟

	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// TableName 指定表名
func (AlertConfig) TableName() string {
	return "alert_configs"
}

// AlertChannel 告警通道配置
type AlertChannel struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`              // 通道名称
	ChannelType string    `gorm:"type:varchar(20);not null" json:"channelType"`       // 通道类型: email, webhook, wechat, dingtalk, feishu
	Enabled     bool      `gorm:"type:tinyint(1);default:1" json:"enabled"`           // 是否启用

	// 通道配置（JSON格式存储）
	Config      string    `gorm:"type:text" json:"config"`                            // 通道配置

	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// TableName 指定表名
func (AlertChannel) TableName() string {
	return "alert_channels"
}

// AlertReceiver 告警接收人
type AlertReceiver struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	Name            string    `gorm:"type:varchar(100);not null" json:"name"`          // 接收人姓名
	Email           string    `gorm:"type:varchar(255)" json:"email"`                 // 邮箱地址
	Phone           string    `gorm:"type:varchar(20)" json:"phone"`                  // 手机号
	WeChatID        string    `gorm:"type:varchar(100)" json:"wechatId"`             // 企业微信ID
	DingTalkID      string    `gorm:"type:varchar(100)" json:"dingtalkId"`           // 钉钉ID
	FeishuID        string    `gorm:"type:varchar(100)" json:"feishuId"`             // 飞书ID
	UserID          *uint     `gorm:"index" json:"userId"`                           // 关联的系统用户ID

	// 告警方式偏好
	EnableEmail     bool      `gorm:"type:tinyint(1);default:1" json:"enableEmail"`   // 接收邮件
	EnableWebhook   bool      `gorm:"type:tinyint(1);default:0" json:"enableWebhook"` // 接收Webhook
	EnableWeChat    bool      `gorm:"type:tinyint(1);default:1" json:"enableWeChat"`  // 接收企业微信
	EnableDingTalk  bool      `gorm:"type:tinyint(1);default:1" json:"enableDingTalk"` // 接收钉钉
	EnableFeishu    bool      `gorm:"type:tinyint(1);default:1" json:"enableFeishu"`  // 接收飞书
	EnableSystemMsg bool      `gorm:"type:tinyint(1);default:1" json:"enableSystemMsg"` // 接收系统消息

	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// TableName 指定表名
func (AlertReceiver) TableName() string {
	return "alert_receivers"
}

// AlertLog 告警日志
type AlertLog struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	AlertType       string    `gorm:"type:varchar(50);not null;index" json:"alertType"`     // 告警类型
	DomainMonitorID uint      `gorm:"index;not null" json:"domainMonitorId"`                     // 域名监控ID
	Domain          string    `gorm:"type:varchar(255);not null" json:"domain"`                 // 域名
	Status          string    `gorm:"type:varchar(20);not null" json:"status"`                 // 状态: success, failed
	Message         string    `gorm:"type:text" json:"message"`                                 // 告警消息内容
	ChannelType     string    `gorm:"type:varchar(20)" json:"channelType"`                      // 发送通道
	ErrorMsg        string    `gorm:"type:text" json:"errorMsg"`                                // 错误信息
	SentAt          time.Time `gorm:"index" json:"sentAt"`                                     // 发送时间
	CreatedAt       time.Time `json:"createdAt"`
}

// TableName 指定表名
func (AlertLog) TableName() string {
	return "alert_logs"
}
