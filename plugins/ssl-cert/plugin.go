package sslcert

import (
	"context"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ydcloud-dy/opshub/internal/plugin"
	"github.com/ydcloud-dy/opshub/plugins/ssl-cert/deployer"
	"github.com/ydcloud-dy/opshub/plugins/ssl-cert/model"
	"github.com/ydcloud-dy/opshub/plugins/ssl-cert/server"
	"github.com/ydcloud-dy/opshub/plugins/ssl-cert/service"
)

// Plugin SSL证书管理插件实现
type Plugin struct {
	db        *gorm.DB
	scheduler *service.Scheduler

	// 配置
	acmeEmail   string
	acmeStaging bool

	ctx       context.Context
	cancelCtx context.CancelFunc
}

// New 创建插件实例
func New() *Plugin {
	// 从环境变量读取ACME邮箱，如果未设置则使用空字符串（将在申请时报错提示用户配置）
	acmeEmail := os.Getenv("OPSHUB_ACME_EMAIL")
	if acmeEmail == "" {
		// 尝试其他常见的环境变量名
		acmeEmail = os.Getenv("ACME_EMAIL")
	}
	if acmeEmail == "" {
		acmeEmail = os.Getenv("LETSENCRYPT_EMAIL")
	}

	// 是否使用Let's Encrypt测试环境
	acmeStaging := os.Getenv("OPSHUB_ACME_STAGING") == "true"

	return &Plugin{
		acmeEmail:   acmeEmail,
		acmeStaging: acmeStaging,
	}
}

// NewWithConfig 创建带配置的插件实例
func NewWithConfig(acmeEmail string, acmeStaging bool) *Plugin {
	return &Plugin{
		acmeEmail:   acmeEmail,
		acmeStaging: acmeStaging,
	}
}

// Name 返回插件名称
func (p *Plugin) Name() string {
	return "ssl-cert"
}

// Description 返回插件描述
func (p *Plugin) Description() string {
	return "SSL证书自动续期插件 - 支持Let's Encrypt和云厂商证书服务,自动部署到Nginx和K8s"
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
		&model.SSLCertificate{},
		&model.DNSProvider{},
		&model.DeployConfig{},
		&model.RenewTask{},
	}

	for _, m := range models {
		if err := db.AutoMigrate(m); err != nil {
			return err
		}
	}

	// 创建上下文
	p.ctx, p.cancelCtx = context.WithCancel(context.Background())

	// 创建部署器依赖
	deployerDeps := &deployer.Dependencies{
		HostGetter:    NewHostGetter(db),
		ClusterGetter: NewClusterGetter(db),
	}

	// 创建并启动调度器
	p.scheduler = service.NewScheduler(db, deployerDeps, p.acmeEmail, p.acmeStaging, time.Hour)
	p.scheduler.Start()

	return nil
}

// Disable 禁用插件
func (p *Plugin) Disable(db *gorm.DB) error {
	// 停止调度器
	if p.scheduler != nil {
		p.scheduler.Stop()
	}

	// 取消上下文
	if p.cancelCtx != nil {
		p.cancelCtx()
	}

	return nil
}

// RegisterRoutes 注册路由
func (p *Plugin) RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
	// 创建部署器依赖
	deployerDeps := &deployer.Dependencies{
		HostGetter:    NewHostGetter(db),
		ClusterGetter: NewClusterGetter(db),
	}

	// 创建服务
	certSvc := service.NewCertificateService(db, deployerDeps, p.acmeEmail, p.acmeStaging)
	dnsSvc := service.NewDNSProviderService(db)
	deploySvc := service.NewDeployService(db, deployerDeps)
	taskSvc := service.NewTaskService(db, certSvc)

	// 注册路由
	sslCertGroup := router.Group("/ssl-cert")
	server.RegisterRoutes(sslCertGroup, certSvc, dnsSvc, deploySvc, taskSvc)
}

// GetMenus 获取插件菜单配置
func (p *Plugin) GetMenus() []plugin.MenuConfig {
	return []plugin.MenuConfig{
		{
			Name:       "SSL证书",
			Path:       "/ssl-cert",
			Icon:       "Key",
			Sort:       50,
			ParentPath: "",
		},
		{
			Name:       "证书管理",
			Path:       "/ssl-cert/certificates",
			Icon:       "Document",
			Sort:       1,
			ParentPath: "/ssl-cert",
		},
		{
			Name:       "DNS配置",
			Path:       "/ssl-cert/dns-providers",
			Icon:       "Connection",
			Sort:       2,
			ParentPath: "/ssl-cert",
		},
		{
			Name:       "部署配置",
			Path:       "/ssl-cert/deploy-configs",
			Icon:       "Upload",
			Sort:       3,
			ParentPath: "/ssl-cert",
		},
		{
			Name:       "任务记录",
			Path:       "/ssl-cert/tasks",
			Icon:       "List",
			Sort:       4,
			ParentPath: "/ssl-cert",
		},
	}
}
