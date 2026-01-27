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

// DomainCheckHistory 域名检查历史记录
type DomainCheckHistory struct {
	ID           uint       `gorm:"primarykey" json:"id"`
	DomainID     uint       `gorm:"index;not null" json:"domainId"`                              // 域名监控ID
	Domain       string     `gorm:"type:varchar(255);not null" json:"domain"`                   // 域名
	Status       string     `gorm:"type:varchar(20);not null" json:"status"`                    // 检查状态: normal, abnormal
	ResponseTime int        `gorm:"type:int;default:0" json:"responseTime"`                     // 响应时间(ms)
	SSLValid     bool       `gorm:"type:tinyint(1);default:0" json:"sslValid"`                  // SSL是否有效
	SSLExpiry    *time.Time `gorm:"type:datetime" json:"sslExpiry"`                             // SSL过期时间
	StatusCode   int        `gorm:"type:int;default:0" json:"statusCode"`                       // HTTP状态码
	ErrorMessage string     `gorm:"type:text" json:"errorMessage"`                              // 错误信息
	CheckedAt    time.Time  `gorm:"type:datetime;not null;index" json:"checkedAt"`              // 检查时间
	CreatedAt    time.Time  `json:"createdAt"`
}

// TableName 指定表名
func (DomainCheckHistory) TableName() string {
	return "domain_check_histories"
}
