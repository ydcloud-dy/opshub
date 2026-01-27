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

package monitor

import (
	"context"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"time"

	"github.com/ydcloud-dy/opshub/internal/plugin"
	"github.com/ydcloud-dy/opshub/plugins/monitor/model"
	"github.com/ydcloud-dy/opshub/plugins/monitor/server"
)

// Plugin 监控中心插件实现
type Plugin struct {
	db        *gorm.DB
	name      string
	ctx       context.Context
	cancelCtx context.CancelFunc
}

// New 创建插件实例
func New() *Plugin {
	return &Plugin{
		name: "monitor",
	}
}

// Name 返回插件名称
func (p *Plugin) Name() string {
	return "monitor"
}

// Description 返回插件描述
func (p *Plugin) Description() string {
	return "监控中心插件 - 支持域名监控等功能"
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
		&model.DomainMonitor{},
		&model.DomainCheckHistory{},
		&model.AlertConfig{},
		&model.AlertChannel{},
		&model.AlertReceiver{},
		&model.AlertReceiverChannel{},
		&model.AlertLog{},
	}

	// 自动迁移所有插件相关的表
	// GORM 的 AutoMigrate 会自动添加缺失的列，不会删除已有数据
	for _, m := range models {
		if err := db.AutoMigrate(m); err != nil {
			return err
		}
	}

	// 启动定时检查任务
	p.ctx, p.cancelCtx = context.WithCancel(context.Background())
	go p.startMonitorScheduler()

	return nil
}

// Disable 禁用插件
func (p *Plugin) Disable(db *gorm.DB) error {
	// 停止定时任务
	if p.cancelCtx != nil {
		p.cancelCtx()
	}
	return nil
}

// startMonitorScheduler 启动监控调度器
func (p *Plugin) startMonitorScheduler() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	handler := server.NewHandler(p.db)

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.checkDueDomains(handler)
		}
	}
}

// checkDueDomains 检查到期需要检查的域名
func (p *Plugin) checkDueDomains(handler *server.Handler) {
	var monitors []model.DomainMonitor
	now := time.Now()

	// 查找需要检查的域名：状态为正常或异常，且下次检查时间已过
	p.db.Where("status IN ? AND next_check <= ?", []string{"normal", "abnormal"}, now).
		Find(&monitors)

	for _, monitor := range monitors {
		// 在后台执行检查，避免阻塞
		go handler.CheckDomainByID(monitor.ID)
	}
}

// RegisterRoutes 注册路由
func (p *Plugin) RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
	server.RegisterRoutes(router, db)
}

// GetMenus 获取插件菜单配置
func (p *Plugin) GetMenus() []plugin.MenuConfig {
	return []plugin.MenuConfig{
		{
			Name: "域名监控",
			Path: "/monitor/domains",
			Icon: "Monitor",
			Sort: 40,
		},
	}
}
