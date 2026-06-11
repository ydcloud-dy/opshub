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
	"github.com/ydcloud-dy/opshub/plugins/monitor/model"
	"gorm.io/gorm"
)

// RegisterRoutes 注册路由
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
	handler := NewHandler(db)
	alertHandler := NewAlertHandler(db)
	dataSourceHandler := NewDataSourceHandler(db)

	// 监控插件路由组 - 使用 /monitor 前缀
	monitorGroup := router.Group("/monitor")
	{
		// 域名监控路由组
		domains := monitorGroup.Group("/domains")
		{
			domains.GET("", handler.ListDomains)                 // 获取域名监控列表
			domains.GET("/stats", handler.GetStats)              // 获取统计数据
			domains.GET("/:id", handler.GetDomain)               // 获取域名监控详情
			domains.GET("/:id/history", handler.GetCheckHistory) // 获取检查历史
			domains.POST("", handler.CreateDomain)               // 创建域名监控
			domains.PUT("/:id", handler.UpdateDomain)            // 更新域名监控
			domains.DELETE("/:id", handler.DeleteDomain)         // 删除域名监控
			domains.POST("/:id/check", handler.CheckDomain)      // 立即检查域名
		}

		// 多数据源管理路由组
		monitorGroup.POST("/datasource-remote-write-test", dataSourceHandler.TestRemoteWriteConfig)
		dataSources := monitorGroup.Group("/datasources")
		{
			dataSources.GET("", dataSourceHandler.ListDataSources)          // 获取数据源列表
			dataSources.GET("/:id", dataSourceHandler.GetDataSource)        // 获取数据源详情
			dataSources.POST("", dataSourceHandler.CreateDataSource)        // 创建数据源
			dataSources.PUT("/:id", dataSourceHandler.UpdateDataSource)     // 更新数据源
			dataSources.DELETE("/:id", dataSourceHandler.DeleteDataSource)  // 删除数据源
			dataSources.POST("/:id/test", dataSourceHandler.TestDataSource) // 测试数据源连通性
			dataSources.POST("/:id/remote-write-test", dataSourceHandler.TestDataSourceRemoteWrite)
			dataSources.POST("/:id/query", dataSourceHandler.QueryDataSource) // 查询测试
			dataSources.GET("/:id/indices", dataSourceHandler.ListDataSourceIndices)
			dataSources.GET("/:id/suggestions", dataSourceHandler.SuggestDataSource)
		}

		// WatchAlert 风格拨测任务路由组
		probeTasks := monitorGroup.Group("/probe-tasks")
		{
			probeTasks.GET("", dataSourceHandler.ListProbeTasks)
			probeTasks.GET("/:id", dataSourceHandler.GetProbeTask)
			probeTasks.POST("", dataSourceHandler.CreateProbeTask)
			probeTasks.PUT("/:id", dataSourceHandler.UpdateProbeTask)
			probeTasks.DELETE("/:id", dataSourceHandler.DeleteProbeTask)
			probeTasks.POST("/:id/run", dataSourceHandler.RunProbeTask)
		}

		monitorGroup.POST("/instant-probe", dataSourceHandler.InstantProbe)

		// 故障中心路由组
		faultCenters := monitorGroup.Group("/fault-centers")
		{
			faultCenters.GET("", dataSourceHandler.ListFaultCenters)
			faultCenters.GET("/:id", dataSourceHandler.GetFaultCenter)
			faultCenters.GET("/:id/slo", dataSourceHandler.GetFaultCenterSLO)
			faultCenters.POST("", dataSourceHandler.CreateFaultCenter)
			faultCenters.PUT("/:id", dataSourceHandler.UpdateFaultCenter)
			faultCenters.DELETE("/:id", dataSourceHandler.DeleteFaultCenter)
		}

		// WatchAlert 风格通知对象路由组
		noticeObjects := monitorGroup.Group("/notice-objects")
		{
			noticeObjects.GET("", dataSourceHandler.ListNoticeObjects)
			noticeObjects.POST("/test", dataSourceHandler.TestNoticeObject)
			noticeObjects.GET("/:id", dataSourceHandler.GetNoticeObject)
			noticeObjects.POST("", dataSourceHandler.CreateNoticeObject)
			noticeObjects.PUT("/:id", dataSourceHandler.UpdateNoticeObject)
			noticeObjects.DELETE("/:id", dataSourceHandler.DeleteNoticeObject)
		}

		// 通知模板路由组
		noticeTemplates := monitorGroup.Group("/notice-templates")
		{
			noticeTemplates.GET("", dataSourceHandler.ListNoticeTemplates)
			noticeTemplates.GET("/:id", dataSourceHandler.GetNoticeTemplate)
			noticeTemplates.POST("", dataSourceHandler.CreateNoticeTemplate)
			noticeTemplates.PUT("/:id", dataSourceHandler.UpdateNoticeTemplate)
			noticeTemplates.DELETE("/:id", dataSourceHandler.DeleteNoticeTemplate)
		}

		// 值班表路由组
		dutyTables := monitorGroup.Group("/duty-tables")
		{
			dutyTables.GET("", dataSourceHandler.ListDutyTables)
			dutyTables.GET("/:id", dataSourceHandler.GetDutyTable)
			dutyTables.POST("", dataSourceHandler.CreateDutyTable)
			dutyTables.PUT("/:id", dataSourceHandler.UpdateDutyTable)
			dutyTables.DELETE("/:id", dataSourceHandler.DeleteDutyTable)
		}

		// 值班日程路由组
		dutySchedules := monitorGroup.Group("/duty-schedules")
		{
			dutySchedules.GET("", dataSourceHandler.ListDutySchedules)
			dutySchedules.POST("", dataSourceHandler.UpsertDutySchedule)
			dutySchedules.PUT("/:id", dataSourceHandler.UpdateDutySchedule)
			dutySchedules.DELETE("/:id", dataSourceHandler.DeleteDutySchedule)
		}

		// 告警规则组路由组
		ruleGroups := monitorGroup.Group("/rule-groups")
		{
			ruleGroups.GET("", dataSourceHandler.ListRuleGroups)
			ruleGroups.POST("", dataSourceHandler.CreateRuleGroup)
			ruleGroups.PUT("/:id", dataSourceHandler.UpdateRuleGroup)
			ruleGroups.DELETE("/:id", dataSourceHandler.DeleteRuleGroup)
		}

		// 数据源告警规则路由组
		rules := monitorGroup.Group("/rules")
		{
			rules.GET("", dataSourceHandler.ListAlertRules) // 获取告警规则列表
			rules.POST("/batch-update", dataSourceHandler.BatchUpdateAlertRules)
			rules.POST("/batch-delete", dataSourceHandler.BatchDeleteAlertRules)
			rules.POST("/export", dataSourceHandler.ExportAlertRules)
			rules.POST("/import", dataSourceHandler.ImportAlertRules)
			rules.POST("/import-prometheus-yaml", dataSourceHandler.ImportPrometheusRuleYAML)
			rules.GET("/:id", dataSourceHandler.GetAlertRule)       // 获取告警规则详情
			rules.POST("", dataSourceHandler.CreateAlertRule)       // 创建告警规则
			rules.PUT("/:id", dataSourceHandler.UpdateAlertRule)    // 更新告警规则
			rules.DELETE("/:id", dataSourceHandler.DeleteAlertRule) // 删除告警规则
			rules.POST("/:id/evaluate", dataSourceHandler.EvaluateAlertRule)
		}

		// 数据源告警事件路由组
		events := monitorGroup.Group("/alert-events")
		{
			events.GET("", dataSourceHandler.ListAlertEvents)
			events.GET("/stats", dataSourceHandler.GetAlertEventStats)
			events.POST("/batch-ack", dataSourceHandler.BatchAcknowledgeAlertEvents)
			events.POST("/batch-delete", dataSourceHandler.BatchDeleteAlertEvents)
			events.GET("/:id", dataSourceHandler.GetAlertEvent)
			events.GET("/:id/callback-queries", dataSourceHandler.GetAlertEventCallbackQueries)
			events.POST("/:id/ack", dataSourceHandler.AcknowledgeAlertEvent)
			events.POST("/:id/silence", dataSourceHandler.SilenceAlertEvent)
			events.DELETE("/:id", dataSourceHandler.DeleteAlertEvent)
		}

		// 证书管理路由组
		certificates := monitorGroup.Group("/certificates")
		{
			certificates.POST("/upload", handler.UploadCertificate)     // 上传证书文件
			certificates.POST("/validate", handler.ValidateCertificate) // 验证证书内容
		}

		// 告警配置路由组
		alerts := monitorGroup.Group("/alerts")
		{
			// 告警通道管理
			channels := alerts.Group("/channels")
			{
				channels.GET("", alertHandler.ListAlertChannels)         // 获取告警通道列表
				channels.GET("/:id", alertHandler.GetAlertChannel)       // 获取告警通道详情
				channels.POST("", alertHandler.CreateAlertChannel)       // 创建告警通道
				channels.PUT("/:id", alertHandler.UpdateAlertChannel)    // 更新告警通道
				channels.DELETE("/:id", alertHandler.DeleteAlertChannel) // 删除告警通道
			}

			// 告警接收人管理
			receivers := alerts.Group("/receivers")
			{
				receivers.GET("", alertHandler.ListAlertReceivers)         // 获取告警接收人列表
				receivers.GET("/:id", alertHandler.GetAlertReceiver)       // 获取告警接收人详情
				receivers.POST("", alertHandler.CreateAlertReceiver)       // 创建告警接收人
				receivers.PUT("/:id", alertHandler.UpdateAlertReceiver)    // 更新告警接收人
				receivers.DELETE("/:id", alertHandler.DeleteAlertReceiver) // 删除告警接收人
			}

			// 告警接收人与通道关联管理
			receiverChannels := alerts.Group("/receiver-channels")
			{
				receiverChannels.GET("/:receiverId", alertHandler.ListReceiverChannels)                   // 获取接收人的通道关联列表
				receiverChannels.POST("/:receiverId", alertHandler.AddReceiverChannel)                    // 添加接收人通道关联
				receiverChannels.DELETE("/:receiverId/:channelId", alertHandler.RemoveReceiverChannel)    // 删除接收人通道关联
				receiverChannels.PUT("/:receiverId/:channelId", alertHandler.UpdateReceiverChannelConfig) // 更新接收人通道关联配置
			}

			// 告警日志管理
			logs := alerts.Group("/logs")
			{
				logs.GET("", alertHandler.ListAlertLogs)       // 获取告警日志列表
				logs.GET("/stats", alertHandler.GetAlertStats) // 获取告警日志统计
			}

			// 告警统计
			alerts.GET("/stats", alertHandler.GetAlertStats) // 获取告警统计信息
		}
	}
}

// AutoMigrate 自动迁移表结构
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.DomainMonitor{},
		&model.DomainCheckHistory{},
		&model.AlertConfig{},
		&model.AlertChannel{},
		&model.AlertReceiver{},
		&model.AlertReceiverChannel{},
		&model.AlertLog{},
		&model.DataSource{},
		&model.FaultCenter{},
		&model.AlertRuleGroup{},
		&model.AlertRule{},
		&model.AlertEvent{},
		&model.ProbeTask{},
		&model.ProbeHistory{},
		&model.NoticeTemplate{},
		&model.NoticeObject{},
		&model.DutyTable{},
		&model.DutySchedule{},
	)
}
