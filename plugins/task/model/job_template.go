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

// JobTemplate 任务模板
type JobTemplate struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Name        string     `json:"name" gorm:"size:255;not null" binding:"required"`
	Code        string     `json:"code" gorm:"size:100;not null;uniqueIndex" binding:"required"`
	Description string     `json:"description" gorm:"type:text"`
	Content     string     `json:"content" gorm:"type:longtext;not null" binding:"required"`
	Variables   string     `json:"variables,omitempty" gorm:"type:text"` // JSON字符串
	Category    string     `json:"category" gorm:"size:50;not null;index" binding:"required"` // script, ansible, module
	Platform    string     `json:"platform,omitempty" gorm:"size:50"` // linux, windows
	Timeout     int        `json:"timeout" gorm:"default:300"` // 秒
	Sort        int        `json:"sort" gorm:"default:0;index"`
	Status      int        `json:"status" gorm:"default:1;index"` // 0-禁用, 1-启用
	CreatedBy   uint       `json:"createdBy" gorm:"not null"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty" gorm:"index"`
}

func (JobTemplate) TableName() string {
	return "job_templates"
}
