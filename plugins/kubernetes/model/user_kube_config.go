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

// UserKubeConfig 用户K8s凭据表
type UserKubeConfig struct {
	ID             uint64    `gorm:"primaryKey" json:"id"`
	ClusterID      uint64    `gorm:"not null;index" json:"clusterId"`
	UserID         uint64    `gorm:"not null;index" json:"userId"`
	ServiceAccount string    `gorm:"size:255;not null;index" json:"serviceAccount"`
	Namespace      string    `gorm:"size:255;default:'default'" json:"namespace"`
	IsActive       bool      `gorm:"default:1" json:"isActive"`
	CreatedBy      uint64    `gorm:"not null" json:"createdBy"`
	CreatedAt      time.Time `gorm:"type:datetime" json:"createdAt"`
	RevokedAt      *time.Time `gorm:"type:datetime" json:"revokedAt"`
}

// TableName 指定表名
func (UserKubeConfig) TableName() string {
	return "k8s_user_kube_configs"
}
