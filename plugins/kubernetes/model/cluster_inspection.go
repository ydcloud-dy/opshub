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

// ClusterInspection 集群巡检记录表
type ClusterInspection struct {
	ID           uint64     `gorm:"primaryKey" json:"id"`
	ClusterID    uint64     `gorm:"not null;index;comment:集群ID" json:"clusterId"`
	ClusterName  string     `gorm:"type:varchar(100);comment:集群名称" json:"clusterName"`
	Status       string     `gorm:"type:varchar(20);comment:巡检状态(running/completed/failed)" json:"status"`
	Score        int        `gorm:"type:int;comment:健康评分(0-100)" json:"score"`
	CheckCount   int        `gorm:"type:int;comment:检查项总数" json:"checkCount"`
	PassCount    int        `gorm:"type:int;comment:通过项数" json:"passCount"`
	WarningCount int        `gorm:"type:int;comment:警告项数" json:"warningCount"`
	FailCount    int        `gorm:"type:int;comment:失败项数" json:"failCount"`
	Duration     int        `gorm:"type:int;comment:巡检耗时(秒)" json:"duration"`
	ReportData   string     `gorm:"type:longtext;comment:巡检报告JSON数据" json:"reportData"`
	UserID       uint64     `gorm:"index;comment:巡检发起人ID" json:"userId"`
	StartTime    time.Time  `gorm:"comment:开始时间" json:"startTime"`
	EndTime      *time.Time `gorm:"comment:结束时间" json:"endTime"`
	CreatedAt    time.Time  `gorm:"type:datetime" json:"createdAt"`
	UpdatedAt    time.Time  `gorm:"type:datetime" json:"updatedAt"`
}

// TableName 指定表名
func (ClusterInspection) TableName() string {
	return "k8s_cluster_inspections"
}

// InspectionStatus 巡检状态常量
const (
	InspectionStatusRunning   = "running"
	InspectionStatusCompleted = "completed"
	InspectionStatusFailed    = "failed"
)

// CheckStatus 检查项状态
const (
	CheckStatusSuccess = "success"
	CheckStatusWarning = "warning"
	CheckStatusError   = "error"
)
