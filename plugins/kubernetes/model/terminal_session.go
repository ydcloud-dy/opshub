package model

import (
	"time"
)

// TerminalSession 终端会话录制记录
type TerminalSession struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time `gorm:"type:datetime" json:"createdAt"`
	UpdatedAt    time.Time `gorm:"type:datetime" json:"updatedAt"`

	// 集群信息
	ClusterID   uint   `gorm:"not null;index:idx_cluster_id" json:"clusterId"`
	ClusterName string `gorm:"size:100" json:"clusterName"`

	// Pod 信息
	Namespace     string `gorm:"size:100;not null;index:idx_namespace" json:"namespace"`
	PodName       string `gorm:"size:200;not null;index:idx_pod_name" json:"podName"`
	ContainerName string `gorm:"size:100;not null" json:"containerName"`

	// 用户信息
	UserID   uint   `gorm:"not null;index:idx_user_id" json:"userId"`
	Username string `gorm:"size:100" json:"username"`

	// 录制文件信息
	RecordingPath string `gorm:"size:500;not null" json:"recordingPath"` // asciinema录制文件路径
	Duration      int    `json:"duration"`                              // 会话时长（秒）
	FileSize      int64  `json:"fileSize"`                              // 文件大小（字节）

	// 状态
	Status string `gorm:"size:20;default:'completed'" json:"status"` // recording, completed, failed
}

// TableName 指定表名
func (TerminalSession) TableName() string {
	return "k8s_terminal_sessions"
}

// TerminalSessionStatus 会话状态常量
const (
	SessionStatusRecording = "recording" // 录制中
	SessionStatusCompleted = "completed" // 已完成
	SessionStatusFailed    = "failed"    // 失败
)
