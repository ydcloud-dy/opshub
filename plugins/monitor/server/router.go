package server

import (
	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/plugins/monitor/model"
	"gorm.io/gorm"
)

// RegisterRoutes 注册路由
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
	handler := NewHandler(db)
	alertHandler := NewAlertHandler(db)

	// 监控插件路由组 - 使用 /monitor 前缀
	monitorGroup := router.Group("/monitor")
	{
		// 域名监控路由组
		domains := monitorGroup.Group("/domains")
		{
			domains.GET("", handler.ListDomains)               // 获取域名监控列表
			domains.GET("/stats", handler.GetStats)            // 获取统计数据
			domains.GET("/:id", handler.GetDomain)             // 获取域名监控详情
			domains.POST("", handler.CreateDomain)             // 创建域名监控
			domains.PUT("/:id", handler.UpdateDomain)          // 更新域名监控
			domains.DELETE("/:id", handler.DeleteDomain)       // 删除域名监控
			domains.POST("/:id/check", handler.CheckDomain)    // 立即检查域名
		}

		// 告警配置路由组
		alerts := monitorGroup.Group("/alerts")
		{
			// 告警通道管理
			channels := alerts.Group("/channels")
			{
				channels.GET("", alertHandler.ListAlertChannels)           // 获取告警通道列表
				channels.GET("/:id", alertHandler.GetAlertChannel)          // 获取告警通道详情
				channels.POST("", alertHandler.CreateAlertChannel)          // 创建告警通道
				channels.PUT("/:id", alertHandler.UpdateAlertChannel)       // 更新告警通道
				channels.DELETE("/:id", alertHandler.DeleteAlertChannel)    // 删除告警通道
			}

			// 告警接收人管理
			receivers := alerts.Group("/receivers")
			{
				receivers.GET("", alertHandler.ListAlertReceivers)          // 获取告警接收人列表
				receivers.GET("/:id", alertHandler.GetAlertReceiver)        // 获取告警接收人详情
				receivers.POST("", alertHandler.CreateAlertReceiver)        // 创建告警接收人
				receivers.PUT("/:id", alertHandler.UpdateAlertReceiver)     // 更新告警接收人
				receivers.DELETE("/:id", alertHandler.DeleteAlertReceiver)  // 删除告警接收人
			}

			// 告警日志管理
			logs := alerts.Group("/logs")
			{
				logs.GET("", alertHandler.ListAlertLogs)                    // 获取告警日志列表
				logs.GET("/stats", alertHandler.GetAlertStats)              // 获取告警统计信息
			}
		}
	}
}

// AutoMigrate 自动迁移表结构
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.DomainMonitor{},
	)
}
