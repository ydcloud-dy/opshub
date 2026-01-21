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
