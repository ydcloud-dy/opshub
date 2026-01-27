package test

import (
	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/plugin"
	"gorm.io/gorm"
)

// TestPlugin 测试插件
type TestPlugin struct{}

// New 创建测试插件实例
func New() plugin.Plugin {
	return &TestPlugin{}
}

// Name 插件名称
func (p *TestPlugin) Name() string {
	return "test"
}

// Description 插件描述
func (p *TestPlugin) Description() string {
	return "这是一个简单的测试插件，用于测试插件安装功能"
}

// Version 插件版本
func (p *TestPlugin) Version() string {
	return "1.0.0"
}

// Author 插件作者
func (p *TestPlugin) Author() string {
	return "Test Team"
}

// Enable 启用插件
func (p *TestPlugin) Enable(db *gorm.DB) error {
	// 可以在这里初始化数据库表、配置等
	// 例如：db.AutoMigrate(&TestModel{})
	return nil
}

// Disable 禁用插件
func (p *TestPlugin) Disable(db *gorm.DB) error {
	// 清理资源
	return nil
}

// RegisterRoutes 注册路由
func (p *TestPlugin) RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
	// 创建测试路由组
	testGroup := router.Group("/test")
	{
		// 测试接口
		testGroup.GET("/hello", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"code":    0,
				"message": "Hello from Test Plugin!",
				"data": gin.H{
					"plugin":  "test",
					"version": "1.0.0",
					"status":  "running",
				},
			})
		})

		// 获取插件信息
		testGroup.GET("/info", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"code":    0,
				"message": "success",
				"data": gin.H{
					"name":        p.Name(),
					"description": p.Description(),
					"version":     p.Version(),
					"author":      p.Author(),
				},
			})
		})
	}
}

// GetMenus 获取菜单配置
func (p *TestPlugin) GetMenus() []plugin.MenuConfig {
	return []plugin.MenuConfig{
		{
			Name:       "测试插件",
			Path:       "/test",
			Icon:       "Grape",
			Sort:       95,
			Hidden:     false,
			ParentPath: "",
		},
		{
			Name:       "测试首页",
			Path:       "/test/home",
			Icon:       "House",
			Sort:       1,
			Hidden:     false,
			ParentPath: "/test",
		},
	}
}
