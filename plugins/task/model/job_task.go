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
