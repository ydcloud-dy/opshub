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

// K8sUserRoleBinding 用户K8s角色绑定
type K8sUserRoleBinding struct {
	ID            uint64    `gorm:"primaryKey" json:"id"`
	ClusterID     uint64    `gorm:"not null;index:idx_cluster_id;index:idx_cluster_user_role" json:"clusterId"`
	UserID        uint64    `gorm:"not null;index:idx_user_id;index:idx_cluster_user_role" json:"userId"`
	RoleName      string    `gorm:"size:255;not null;index:idx_cluster_user_role" json:"roleName"`
	RoleNamespace string    `gorm:"size:255;default:'';index:idx_cluster_user_role" json:"roleNamespace"`
	RoleType      string    `gorm:"size:50;not null" json:"roleType"` // ClusterRole 或 Role
	BoundBy       uint64    `gorm:"not null" json:"boundBy"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// TableName 指定表名
func (K8sUserRoleBinding) TableName() string {
	return "k8s_user_role_bindings"
}
