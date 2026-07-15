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

// RootCauseAnalysis 告警根因分析记录。
type RootCauseAnalysis struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	UserID       uint           `gorm:"index;comment:用户ID" json:"userId"`
	Username     string         `gorm:"type:varchar(100);comment:用户名" json:"username"`
	AlertEventID uint           `gorm:"index;comment:告警事件ID" json:"alertEventId"`
	RuleID       uint           `gorm:"index;comment:告警规则ID" json:"ruleId"`
	RuleName     string         `gorm:"type:varchar(160);comment:规则名称" json:"ruleName"`
	Severity     string         `gorm:"type:varchar(30);index;comment:严重级别" json:"severity"`
	State        string         `gorm:"type:varchar(30);index;comment:告警状态" json:"state"`
	Summary      string         `gorm:"type:longtext;comment:摘要" json:"summary"`
	RootCause    string         `gorm:"type:longtext;comment:根因分析" json:"rootCause"`
	EvidenceJSON string         `gorm:"type:longtext;comment:证据JSON" json:"evidenceJson"`
	Suggestion   string         `gorm:"type:longtext;comment:处理建议" json:"suggestion"`
	Model        string         `gorm:"type:varchar(100);comment:模型" json:"model"`
	Fallback     bool           `gorm:"default:false;comment:是否本地兜底" json:"fallback"`
	Status       string         `gorm:"type:varchar(30);default:'success';index;comment:状态" json:"status"`
}

func (RootCauseAnalysis) TableName() string {
	return "ai_root_cause_analyses"
}
