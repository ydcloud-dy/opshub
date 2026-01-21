package model

import (
	"time"
)

// JobTask 任务作业
type JobTask struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	Name         string     `json:"name" gorm:"size:255;not null" binding:"required"`
	TemplateID   *uint      `json:"templateId,omitempty" gorm:"index"`
	TaskType     string     `json:"taskType" gorm:"size:50;not null;index" binding:"required"` // manual, ansible, cron
	Status       string     `json:"status" gorm:"size:50;not null;default:pending;index"` // pending, running, success, failed
	TargetHosts  string     `json:"targetHosts,omitempty" gorm:"type:text"` // JSON字符串
	Parameters   string     `json:"parameters,omitempty" gorm:"type:text"` // JSON
	ExecuteTime  *time.Time `json:"executeTime,omitempty"`
	Result       string     `json:"result,omitempty" gorm:"type:text"` // JSON
	ErrorMessage string     `json:"errorMessage,omitempty" gorm:"type:text"`
	CreatedBy    uint       `json:"createdBy" gorm:"not null"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	DeletedAt    *time.Time `json:"deletedAt,omitempty" gorm:"index"`
}

func (JobTask) TableName() string {
	return "job_tasks"
}
