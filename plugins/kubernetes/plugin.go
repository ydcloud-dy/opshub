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

package kubernetes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ydcloud-dy/opshub/internal/plugin"
	"github.com/ydcloud-dy/opshub/plugins/kubernetes/model"
	"github.com/ydcloud-dy/opshub/plugins/kubernetes/server"
)

// Plugin Kubernetes 插件实现
type Plugin struct {
	db   *gorm.DB
	name string
}

// New 创建插件实例
func New() *Plugin {
	return &Plugin{
		name: "kubernetes",
	}
}

// Name 返回插件名称
func (p *Plugin) Name() string {
	return "kubernetes"
}

// Description 返回插件描述
func (p *Plugin) Description() string {
	return "Kubernetes 集群管理插件"
}

// Version 返回插件版本
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Author 返回插件作者
func (p *Plugin) Author() string {
	return "J"
}

// Enable 启用插件
func (p *Plugin) Enable(db *gorm.DB) error {
	p.db = db

	// 自动迁移所有插件相关的表
	models := []interface{}{
		&Cluster{},
		&model.K8sUserRoleBinding{},
		&model.UserKubeConfig{},
		&model.TerminalSession{},
		&model.ClusterInspection{},
	}

	for _, m := range models {
		if !db.Migrator().HasTable(m) {
			if err := db.AutoMigrate(m); err != nil {
				return err
			}
		}
	}

	return nil
}

// Disable 禁用插件
func (p *Plugin) Disable(db *gorm.DB) error {
	// 清理插件资源（如果需要）
	return nil
}

// RegisterRoutes 注册路由
func (p *Plugin) RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
	server.RegisterRoutes(router, db)
}

// GetMenus 获取菜单配置
func (p *Plugin) GetMenus() []plugin.MenuConfig {
	return []plugin.MenuConfig{
		{
			Name:      "容器管理",
			Path:      "/kubernetes",
			Icon:      "Platform",
			Sort:      100,
			Hidden:    false,
			ParentPath: "",
		},
		{
			Name:      "集群管理",
			Path:      "/kubernetes/clusters",
			Icon:      "Connection",
			Sort:      101,
			Hidden:    false,
			ParentPath: "/kubernetes",
		},
		{
			Name:      "应用诊断",
			Path:      "/kubernetes/application-diagnosis",
			Icon:      "Grid",
			Sort:      102,
			Hidden:    false,
			ParentPath: "/kubernetes",
		},
		{
			Name:      "集群巡检",
			Path:      "/kubernetes/cluster-inspection",
			Icon:      "Grid",
			Sort:      103,
			Hidden:    false,
			ParentPath: "/kubernetes",
		},
	}
}

// GetClusterClientset 获取指定集群的 clientset（供插件内部其他模块使用）
func (p *Plugin) GetClusterClientset(clusterID uint, kubeConfig string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfig))
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

// Cluster 集群模型（用于 AutoMigrate）
type Cluster struct {
	ID          uint   `gorm:"primarykey"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	Name        string `gorm:"size:100;not null;uniqueIndex"`
	Alias       string `gorm:"size:100"`
	APIEndpoint string `gorm:"size:500;not null"`
	KubeConfig  string `gorm:"type:text;not null"`
	Version     string `gorm:"size:50"`
	Status      int    `gorm:"default:1"`
	Region      string `gorm:"size:100"`
	Provider    string `gorm:"size:50"`
	Description string `gorm:"size:500"`
	CreatedBy   uint
	IsDeleted   bool `gorm:"default:false;index"`
}

// TableName 指定表名
func (Cluster) TableName() string {
	return "k8s_clusters"
}
