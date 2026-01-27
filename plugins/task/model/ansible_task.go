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

// AnsibleTask Ansible任务
type AnsibleTask struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	Name            string     `json:"name" gorm:"size:255;not null" binding:"required"`
	PlaybookContent string     `json:"playbookContent,omitempty" gorm:"type:longtext"`
	PlaybookPath    string     `json:"playbookPath,omitempty" gorm:"size:500"`
	Inventory       string     `json:"inventory,omitempty" gorm:"type:text"` // JSON字符串
	ExtraVars       string     `json:"extraVars,omitempty" gorm:"type:json"` // JSON
	Tags            string     `json:"tags,omitempty" gorm:"size:500"` // 逗号分隔
	Fork            int        `json:"fork" gorm:"default:5"`
	Timeout         int        `json:"timeout" gorm:"default:600"` // 秒
	Verbose         string     `json:"verbose" gorm:"size:20;default:v"` // v, vv, vvv
	Status          string     `json:"status" gorm:"size:50;not null;default:pending;index"` // pending, running, success, failed, cancelled
	LastRunTime     *time.Time `json:"lastRunTime,omitempty"`
	LastRunResult   string     `json:"lastRunResult,omitempty" gorm:"type:json"` // JSON
	CreatedBy       uint       `json:"createdBy" gorm:"not null"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	DeletedAt       *time.Time `json:"deletedAt,omitempty" gorm:"index"`
}

func (AnsibleTask) TableName() string {
	return "ansible_tasks"
}
