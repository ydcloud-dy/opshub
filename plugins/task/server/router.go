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
	"github.com/ydcloud-dy/opshub/plugins/task/model"
	"gorm.io/gorm"
)

func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
	handler := NewHandler(db)

	// 任务插件路由组 - 使用 /task 前缀
	taskGroup := router.Group("/task")
	{
		// 任务执行
		taskGroup.POST("/execute", handler.ExecuteTask)

		// 文件分发
		taskGroup.POST("/distribute", handler.DistributeFiles)

		// 任务作业
		jobs := taskGroup.Group("/jobs")
		{
			jobs.GET("", handler.ListJobTasks)
			jobs.GET("/:id", handler.GetJobTask)
			jobs.POST("", handler.CreateJobTask)
			jobs.PUT("/:id", handler.UpdateJobTask)
			jobs.DELETE("/:id", handler.DeleteJobTask)
		}

		// 任务模板
		templates := taskGroup.Group("/templates")
		{
			templates.GET("", handler.ListJobTemplates)
			templates.GET("/all", handler.GetAllJobTemplates)
			templates.GET("/:id", handler.GetJobTemplate)
			templates.POST("", handler.CreateJobTemplate)
			templates.PUT("/:id", handler.UpdateJobTemplate)
			templates.DELETE("/:id", handler.DeleteJobTemplate)
		}

		// Ansible任务
		ansible := taskGroup.Group("/ansible")
		{
			ansible.GET("", handler.ListAnsibleTasks)
			ansible.GET("/:id", handler.GetAnsibleTask)
			ansible.POST("", handler.CreateAnsibleTask)
			ansible.PUT("/:id", handler.UpdateAnsibleTask)
			ansible.DELETE("/:id", handler.DeleteAnsibleTask)
		}

		// 执行记录
		executionHistory := taskGroup.Group("/execution-history")
		{
			executionHistory.GET("", handler.ListExecutionHistory)
			executionHistory.GET("/:id", handler.GetExecutionHistory)
			executionHistory.DELETE("/:id", handler.DeleteExecutionHistory)
			executionHistory.POST("/batch-delete", handler.BatchDeleteExecutionHistory)
			executionHistory.POST("/export", handler.ExportExecutionHistory)
		}
	}
}

// 自动注册表模型
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.JobTask{},
		&model.JobTemplate{},
		&model.AnsibleTask{},
	)
}
