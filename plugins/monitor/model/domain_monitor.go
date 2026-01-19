package model

import (
	"time"
)

// DomainMonitor 域名监控模型
type DomainMonitor struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	Domain        string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"domain"`          // 域名
	Status        string    `gorm:"type:varchar(20);not null;default:'unknown'" json:"status"`    // 状态: normal, abnormal, paused, unknown
	ResponseTime  int       `gorm:"type:int;default:0" json:"responseTime"`                       // 响应时间(ms)
	SSLValid      bool      `gorm:"type:tinyint(1);default:0" json:"sslValid"`                    // SSL是否有效
	SSLExpiry     *time.Time `gorm:"type:datetime" json:"sslExpiry"`                               // SSL过期时间
	CheckInterval int       `gorm:"type:int;not null;default:300" json:"checkInterval"`           // 检查间隔(秒)
	EnableSSL     bool      `gorm:"type:tinyint(1);default:1" json:"enableSSL"`                    // 是否启用SSL检查
	EnableAlert   bool      `gorm:"type:tinyint(1);default:0" json:"enableAlert"`                  // 是否启用告警
	LastCheck     *time.Time `gorm:"type:datetime" json:"lastCheck"`                               // 最后检查时间
	NextCheck     *time.Time `gorm:"type:datetime;index" json:"nextCheck"`                         // 下次检查时间

	// 告警配置
	AlertConfigID *uint     `gorm:"index" json:"alertConfigId"`                                 // 告警配置ID
	ResponseThreshold *int  `gorm:"type:int;default:1000" json:"responseThreshold"`              // 响应时间阈值(ms)
	SSLExpiryDays   *int  `gorm:"type:int;default:30" json:"sslExpiryDays"`                    // SSL过期提前告警天数

	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// TableName 指定表名
func (DomainMonitor) TableName() string {
	return "domain_monitors"
}
