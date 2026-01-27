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

package models

import "time"

// Cluster 集群模型
type Cluster struct {
	ID             uint       `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time  `gorm:"type:datetime" json:"createdAt"`
	UpdatedAt      time.Time  `gorm:"type:datetime" json:"updatedAt"`
	Name           string     `gorm:"size:100;not null;uniqueIndex" json:"name"`          // 集群名称
	Alias          string     `gorm:"size:100" json:"alias"`                              // 集群别名
	APIEndpoint    string     `gorm:"size:500;not null" json:"apiEndpoint"`                // API Server 地址
	KubeConfig     string     `gorm:"type:text;not null" json:"kubeConfig"`                // KubeConfig 内容（加密存储）
	Version        string     `gorm:"size:50" json:"version"`                             // Kubernetes 版本
	Status         int        `gorm:"default:1" json:"status"`                            // 状态: 1-正常 2-连接失败 3-不可用
	Region         string     `gorm:"size:100" json:"region"`                             // 区域
	Provider       string     `gorm:"size:50" json:"provider"`                            // 云服务商: aws/aliyun/tencent/native
	Description    string     `gorm:"size:500" json:"description"`                        // 描述
	CreatedBy      uint       `json:"createdBy"`                                         // 创建人ID
	NodeCount      int        `gorm:"default:0" json:"nodeCount"`                         // 节点数量（缓存）
	PodCount       int        `gorm:"default:0" json:"podCount"`                          // Pod数量（缓存）
	StatusSyncedAt *time.Time `gorm:"type:datetime" json:"statusSyncedAt"`               // 状态最后同步时间
}

// TableName 指定表名
func (Cluster) TableName() string {
	return "k8s_clusters"
}

// ClusterStatus 集群状态常量
const (
	ClusterStatusNormal   = 1 // 正常
	ClusterStatusFailed   = 2 // 连接失败
	ClusterStatusDisabled = 3 // 不可用
)

// ClusterProvider 云服务商常量
const (
	ProviderNative  = "native"  // 自建集群
	ProviderAWS     = "aws"     // AWS
	ProviderAliyun  = "aliyun"  // 阿里云
	ProviderTencent = "tencent" // 腾讯云
)
