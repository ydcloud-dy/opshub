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

package server

import (
	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/plugins/nginx/model"
	"gorm.io/gorm"
)

// RegisterRoutes 注册路由
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
	handler := NewHandler(db)

	// Nginx 统计插件路由组 - 使用 /nginx 前缀
	nginxGroup := router.Group("/nginx")
	{
		// 数据源管理
		sources := nginxGroup.Group("/sources")
		{
			sources.GET("", handler.ListSources)         // 获取数据源列表
			sources.GET("/:id", handler.GetSource)       // 获取数据源详情
			sources.POST("", handler.CreateSource)       // 创建数据源
			sources.PUT("/:id", handler.UpdateSource)    // 更新数据源
			sources.DELETE("/:id", handler.DeleteSource) // 删除数据源
		}

		// 概况统计
		nginxGroup.GET("/overview", handler.GetOverview)            // 获取概况
		nginxGroup.GET("/overview/trend", handler.GetRequestsTrend) // 获取请求趋势

		// 概况页面新增接口
		overview := nginxGroup.Group("/overview")
		{
			overview.GET("/active-visitors", handler.GetActiveVisitors)  // 活跃访客
			overview.GET("/core-metrics", handler.GetCoreMetrics)        // 核心指标
			overview.GET("/overview-trend", handler.GetOverviewTrend)    // UV+PV趋势
			overview.GET("/new-vs-returning", handler.GetNewVsReturning) // 新老访客
			overview.GET("/top-referers", handler.GetTopReferers)        // 来路排行
			overview.GET("/top-pages", handler.GetTopPages)              // 受访页面
			overview.GET("/top-entry-pages", handler.GetTopEntryPages)   // 入口页面
			overview.GET("/geo", handler.GetOverviewGeo)                 // 地域分布
			overview.GET("/devices", handler.GetOverviewDevices)         // 终端设备
		}

		// 数据日报
		nginxGroup.GET("/daily-report", handler.GetDailyReport) // 获取日报数据

		// 日志采集
		nginxGroup.POST("/collect", handler.CollectLogs) // 手动触发日志采集

		// 回填地理位置数据
		nginxGroup.POST("/backfill-geo", handler.BackfillGeoData) // 回填地理位置数据

		// 访问明细 (旧接口保持兼容)
		accessLogs := nginxGroup.Group("/access-logs")
		{
			accessLogs.GET("", handler.ListAccessLogs)      // 获取访问日志列表
			accessLogs.GET("/top-uris", handler.GetTopURIs) // 获取 Top URI
			accessLogs.GET("/top-ips", handler.GetTopIPs)   // 获取 Top IP
		}

		// 新版日志查询 (带维度信息)
		nginxGroup.GET("/logs", handler.ListAccessLogsWithDimensions) // 获取带维度信息的访问日志

		// 统计分析接口
		stats := nginxGroup.Group("/stats")
		{
			stats.GET("/timeseries", handler.GetTimeSeries)        // 时间序列数据
			stats.GET("/geo", handler.GetGeoDistribution)          // 地理分布
			stats.GET("/browsers", handler.GetBrowserDistribution) // 浏览器分布
			stats.GET("/devices", handler.GetDeviceDistribution)   // 设备分布
			stats.GET("/top-urls", handler.GetTopURIs)             // Top URLs (新路径)
			stats.GET("/top-ips", handler.GetTopIPsWithGeo)        // Top IPs (带地理信息)
			stats.GET("/status-codes", handler.GetOverview)        // 状态码分布 (复用概况)
			stats.GET("/response-time", handler.GetTimeSeries)     // 响应时间 (复用时间序列)
		}
	}
}

// AutoMigrate 自动迁移表结构
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		// 数据源
		&model.NginxSource{},

		// 维度表
		&model.NginxDimIP{},
		&model.NginxDimURL{},
		&model.NginxDimReferer{},
		&model.NginxDimUserAgent{},

		// 事实表
		&model.NginxFactAccessLog{},

		// 新版聚合表
		&model.NginxAggHourly{},
		&model.NginxAggDaily{},

		// 旧版表 (保持兼容)
		&model.NginxAccessLog{},
		&model.NginxDailyStats{},
		&model.NginxHourlyStats{},
	)
}
