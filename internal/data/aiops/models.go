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
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package aiops

import (
	"time"

	"gorm.io/gorm"
)

// Provider 模型供应商配置，兼容 OpenAI-compatible API。
type Provider struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	Name            string         `gorm:"type:varchar(100);not null;comment:配置名称" json:"name"`
	Provider        string         `gorm:"type:varchar(50);not null;default:'openai-compatible';comment:供应商类型" json:"provider"`
	BaseURL         string         `gorm:"type:varchar(500);not null;comment:API地址" json:"baseUrl"`
	APIKey          string         `gorm:"type:varchar(1000);comment:API Key" json:"-"`
	Model           string         `gorm:"type:varchar(100);not null;comment:模型名称" json:"model"`
	Temperature     float64        `gorm:"type:decimal(4,2);default:0.20;comment:温度" json:"temperature"`
	MaxTokens       int            `gorm:"type:int;default:8192;comment:最大输出Token" json:"maxTokens"`
	Timeout         int            `gorm:"type:int;default:180;comment:超时时间秒" json:"timeout"`
	ReasoningEffort string         `gorm:"type:varchar(20);comment:推理强度" json:"reasoningEffort"`
	Enabled         bool           `gorm:"default:true;comment:是否启用" json:"enabled"`
	IsDefault       bool           `gorm:"default:false;comment:是否默认" json:"isDefault"`
	Remark          string         `gorm:"type:varchar(500);comment:备注" json:"remark"`
	LastTestAt      *time.Time     `gorm:"comment:最后测试时间" json:"lastTestAt,omitempty"`
	LastTestMsg     string         `gorm:"type:varchar(500);comment:最后测试结果" json:"lastTestMsg"`
}

func (Provider) TableName() string {
	return "ai_providers"
}

// Session AI 会话。
type Session struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	UserID    uint           `gorm:"index;comment:用户ID" json:"userId"`
	Username  string         `gorm:"type:varchar(100);comment:用户名" json:"username"`
	Title     string         `gorm:"type:varchar(200);comment:会话标题" json:"title"`
	Type      string         `gorm:"type:varchar(50);default:'chat';comment:会话类型 chat/diagnosis/log" json:"type"`
	Status    string         `gorm:"type:varchar(30);default:'active';comment:状态" json:"status"`
	Summary   string         `gorm:"type:text;comment:摘要" json:"summary"`
}

func (Session) TableName() string {
	return "ai_sessions"
}

// Message AI 消息记录。
type Message struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	SessionID  uint           `gorm:"index;comment:会话ID" json:"sessionId"`
	UserID     uint           `gorm:"index;comment:用户ID" json:"userId"`
	Role       string         `gorm:"type:varchar(30);not null;comment:角色 user/assistant/system" json:"role"`
	Content    string         `gorm:"type:longtext;comment:内容" json:"content"`
	Model      string         `gorm:"type:varchar(100);comment:模型" json:"model"`
	Status     string         `gorm:"type:varchar(30);default:'success';comment:状态" json:"status"`
	TokensIn   int            `gorm:"type:int;default:0;comment:输入Token" json:"tokensIn"`
	TokensOut  int            `gorm:"type:int;default:0;comment:输出Token" json:"tokensOut"`
	LatencyMS  int64          `gorm:"type:bigint;default:0;comment:耗时毫秒" json:"latencyMs"`
	Error      string         `gorm:"type:text;comment:错误信息" json:"error,omitempty"`
	ContextRef string         `gorm:"type:longtext;comment:上下文引用JSON" json:"contextRef,omitempty"`
}

func (Message) TableName() string {
	return "ai_messages"
}

// ToolCall AI 工具调用记录。
type ToolCall struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	SessionID uint           `gorm:"index;comment:会话ID" json:"sessionId"`
	MessageID uint           `gorm:"index;comment:消息ID" json:"messageId"`
	UserID    uint           `gorm:"index;comment:用户ID" json:"userId"`
	ToolName  string         `gorm:"type:varchar(100);not null;comment:工具名称" json:"toolName"`
	Params    string         `gorm:"type:longtext;comment:参数JSON" json:"params"`
	Result    string         `gorm:"type:longtext;comment:结果JSON" json:"result"`
	Status    string         `gorm:"type:varchar(30);default:'success';comment:状态" json:"status"`
	LatencyMS int64          `gorm:"type:bigint;default:0;comment:耗时毫秒" json:"latencyMs"`
	Error     string         `gorm:"type:text;comment:错误信息" json:"error,omitempty"`
}

func (ToolCall) TableName() string {
	return "ai_tool_calls"
}

// DiagnosisTask 智能诊断任务。
type DiagnosisTask struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	UserID       uint           `gorm:"index;comment:用户ID" json:"userId"`
	Username     string         `gorm:"type:varchar(100);comment:用户名" json:"username"`
	SessionID    uint           `gorm:"index;comment:会话ID" json:"sessionId"`
	ObjectType   string         `gorm:"type:varchar(50);comment:对象类型 pod/deployment/log" json:"objectType"`
	ClusterID    uint           `gorm:"index;comment:集群ID" json:"clusterId"`
	Namespace    string         `gorm:"type:varchar(120);comment:命名空间" json:"namespace"`
	ObjectName   string         `gorm:"type:varchar(200);comment:对象名称" json:"objectName"`
	Container    string         `gorm:"type:varchar(200);comment:容器名称" json:"container"`
	Status       string         `gorm:"type:varchar(30);default:'success';comment:状态" json:"status"`
	Conclusion   string         `gorm:"type:longtext;comment:诊断结论" json:"conclusion"`
	EvidenceJSON string         `gorm:"type:longtext;comment:证据JSON" json:"evidenceJson"`
	Suggestion   string         `gorm:"type:longtext;comment:处理建议" json:"suggestion"`
	Error        string         `gorm:"type:text;comment:错误信息" json:"error,omitempty"`
}

func (DiagnosisTask) TableName() string {
	return "ai_diagnosis_tasks"
}
