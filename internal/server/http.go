package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/ydcloud-dy/opshub/internal/conf"
	"github.com/ydcloud-dy/opshub/internal/plugin"
	assetserver "github.com/ydcloud-dy/opshub/internal/server/asset"
	auditserver "github.com/ydcloud-dy/opshub/internal/server/audit"
	"github.com/ydcloud-dy/opshub/internal/server/rbac"
	rbacdata "github.com/ydcloud-dy/opshub/internal/data/rbac"
	"github.com/ydcloud-dy/opshub/internal/service"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"github.com/ydcloud-dy/opshub/pkg/middleware"
	k8splugin "github.com/ydcloud-dy/opshub/plugins/kubernetes"
	monitorplugin "github.com/ydcloud-dy/opshub/plugins/monitor"
	taskplugin "github.com/ydcloud-dy/opshub/plugins/task"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// HTTPServer HTTP服务器
type HTTPServer struct {
	server    *http.Server
	conf      *conf.Config
	svc       *service.Service
	db        *gorm.DB
	pluginMgr *plugin.Manager
	uploadSrv *UploadServer
}

// NewHTTPServer 创建HTTP服务器
func NewHTTPServer(conf *conf.Config, svc *service.Service, db *gorm.DB) *HTTPServer {
	// 设置Gin模式
	gin.SetMode(conf.Server.Mode)

	// 创建路由
	router := gin.New()

	// 使用中间件
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.AuditLogOperation(db))

	// 创建插件管理器
	pluginMgr := plugin.NewManager(db)

	// 创建上传服务
	uploadDir := "./web/public/uploads"
	uploadURL := "/uploads"
	uploadSrv := NewUploadServer(db, uploadDir, uploadURL)

	// 注册 Kubernetes 插件
	if err := pluginMgr.Register(k8splugin.New()); err != nil {
		appLogger.Error("注册Kubernetes插件失败", zap.Error(err))
	}

	// 注册 Task 插件
	if err := pluginMgr.Register(taskplugin.New()); err != nil {
		appLogger.Error("注册Task插件失败", zap.Error(err))
	}

	// 注册 Monitor 插件
	if err := pluginMgr.Register(monitorplugin.New()); err != nil {
		appLogger.Error("注册Monitor插件失败", zap.Error(err))
	}

	// 注册路由
	s := &HTTPServer{
		conf:      conf,
		svc:       svc,
		db:        db,
		pluginMgr: pluginMgr,
		uploadSrv: uploadSrv,
	}

	// 先启用所有插件（在注册路由之前）
	s.enablePlugins()

	// 注册路由（插件启用后才能注册路由）
	s.registerRoutes(router, conf.Server.JWTSecret)

	// 创建HTTP服务器
	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", conf.Server.HttpPort),
		Handler:      router,
		ReadTimeout:  time.Duration(conf.Server.ReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(conf.Server.WriteTimeout) * time.Millisecond,
	}

	return s
}

// registerRoutes 注册路由
func (s *HTTPServer) registerRoutes(router *gin.Engine, jwtSecret string) {
	// Swagger 文档
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 健康检查
	router.GET("/health", s.svc.Health)

	// 静态文件服务 - 上传的文件
	router.Static("/uploads", "./web/public/uploads")

	// 创建 RBAC 服务
	userService, roleService, departmentService, menuService, positionService, captchaService, assetPermissionService, authMiddleware := rbac.NewRBACServices(s.db, jwtSecret)

	// RBAC 路由
	rbacServer := rbac.NewHTTPServer(userService, roleService, departmentService, menuService, positionService, captchaService, assetPermissionService, authMiddleware)
	rbacServer.RegisterRoutes(router)

	// 创建 Audit 服务
	operationLogService, loginLogService, dataLogService := auditserver.NewAuditServices(s.db)

	// 创建 Asset 服务
	assetGroupService, hostService, terminalManager := assetserver.NewAssetServices(s.db)

	// 设置authMiddleware的assetPermissionRepo
	assetPermissionRepo := rbacdata.NewAssetPermissionRepo(s.db)
	authMiddleware.SetAssetPermissionRepo(assetPermissionRepo)

	// Asset 路由
	assetServer := assetserver.NewHTTPServer(assetGroupService, hostService, terminalManager, s.db, authMiddleware)

	// API v1 - 需要认证的接口
	v1 := router.Group("/api/v1")
	v1.Use(authMiddleware.AuthRequired())
	{
		// Audit 路由
		auditHTTPServer := auditserver.NewHTTPService(operationLogService, loginLogService, dataLogService)
		auditHTTPServer.RegisterRoutes(v1)

		// 注册 Asset 路由
		assetServer.RegisterRoutes(v1)

		// 上传接口
		v1.POST("/upload/avatar", s.uploadSrv.UploadAvatar)
		v1.PUT("/profile/avatar", s.uploadSrv.UpdateUserAvatar)
	}

	// API v1 - 公开接口(不需要认证)
	public := router.Group("/api/v1/public")
	{
		public.GET("/example", s.svc.Example)
	}

	// 插件路由
	pluginsGroup := router.Group("/api/v1/plugins")
	pluginsGroup.Use(authMiddleware.AuthRequired())
	s.pluginMgr.RegisterAllRoutes(pluginsGroup)

	// 插件管理接口
	pluginInfoGroup := router.Group("/api/v1/plugins")
	pluginInfoGroup.Use(authMiddleware.AuthRequired())
	{
		pluginInfoGroup.GET("", s.listPlugins)
		pluginInfoGroup.GET("/:name", s.getPlugin)
		pluginInfoGroup.GET("/:name/menus", s.getPluginMenus)
		pluginInfoGroup.POST("/:name/enable", s.enablePlugin)
		pluginInfoGroup.POST("/:name/disable", s.disablePlugin)
		pluginInfoGroup.POST("/upload", s.uploadSrv.UploadPlugin)
		pluginInfoGroup.DELETE("/:name/uninstall", s.uploadSrv.UninstallPlugin)
	}

	// 前端静态文件服务（后面会用到）
	// router.Static("/assets", "./web/dist/assets")
	// router.NoRoute(func(c *gin.Context) {
	//     c.File("./web/dist/index.html")
	// })
}

// enablePlugins 启用所有已注册的插件
func (s *HTTPServer) enablePlugins() {
	for _, p := range s.pluginMgr.GetAllPlugins() {
		if err := s.pluginMgr.Enable(p.Name()); err != nil {
			appLogger.Error("启用插件失败",
				zap.String("plugin", p.Name()),
				zap.Error(err),
			)
		} else {
			appLogger.Info("插件启用成功",
				zap.String("plugin", p.Name()),
				zap.String("version", p.Version()),
			)
		}
	}
}

// listPlugins 获取所有插件列表
func (s *HTTPServer) listPlugins(c *gin.Context) {
	plugins := s.pluginMgr.GetAllPlugins()
	result := make([]map[string]interface{}, 0, len(plugins))

	for _, p := range plugins {
		enabled := s.pluginMgr.IsEnabled(p.Name())
		appLogger.Info("获取插件状态",
			zap.String("plugin", p.Name()),
			zap.Bool("enabled", enabled),
		)
		result = append(result, map[string]interface{}{
			"name":        p.Name(),
			"description": p.Description(),
			"version":     p.Version(),
			"author":      p.Author(),
			"enabled":     enabled,
		})
	}

	appLogger.Info("返回插件列表", zap.Int("count", len(result)))
	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// getPlugin 获取插件详情
func (s *HTTPServer) getPlugin(c *gin.Context) {
	name := c.Param("name")
	plugin, exists := s.pluginMgr.GetPlugin(name)
	if !exists {
		c.JSON(404, gin.H{
			"code":    404,
			"message": "plugin not found",
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": map[string]interface{}{
			"name":        plugin.Name(),
			"description": plugin.Description(),
			"version":     plugin.Version(),
			"author":      plugin.Author(),
			"enabled":     s.pluginMgr.IsEnabled(name),
		},
	})
}

// getPluginMenus 获取插件的菜单配置
func (s *HTTPServer) getPluginMenus(c *gin.Context) {
	name := c.Param("name")
	plugin, exists := s.pluginMgr.GetPlugin(name)
	if !exists {
		c.JSON(404, gin.H{
			"code":    404,
			"message": "plugin not found",
		})
		return
	}

	menus := plugin.GetMenus()
	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data":    menus,
	})
}

// enablePlugin 启用插件
func (s *HTTPServer) enablePlugin(c *gin.Context) {
	name := c.Param("name")

	if err := s.pluginMgr.Enable(name); err != nil {
		appLogger.Error("启用插件失败",
			zap.String("plugin", name),
			zap.Error(err),
		)
		c.JSON(500, gin.H{
			"code":    500,
			"message": fmt.Sprintf("启用插件失败: %v", err),
		})
		return
	}

	appLogger.Info("插件启用成功", zap.String("plugin", name))
	c.JSON(200, gin.H{
		"code":    0,
		"message": "插件启用成功，请刷新页面以生效",
	})
}

// disablePlugin 禁用插件
func (s *HTTPServer) disablePlugin(c *gin.Context) {
	name := c.Param("name")

	if err := s.pluginMgr.Disable(name); err != nil {
		appLogger.Error("禁用插件失败",
			zap.String("plugin", name),
			zap.Error(err),
		)
		c.JSON(500, gin.H{
			"code":    500,
			"message": fmt.Sprintf("禁用插件失败: %v", err),
		})
		return
	}

	appLogger.Info("插件禁用成功", zap.String("plugin", name))
	c.JSON(200, gin.H{
		"code":    0,
		"message": "插件禁用成功，请刷新页面以生效",
	})
}

// Start 启动服务器
func (s *HTTPServer) Start() error {
	appLogger.Info("HTTP服务器启动",
		zap.String("addr", s.server.Addr),
		zap.String("mode", s.conf.Server.Mode),
	)

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP服务器启动失败: %w", err)
	}

	return nil
}

// Stop 停止服务器
func (s *HTTPServer) Stop(ctx context.Context) error {
	appLogger.Info("HTTP服务器停止中...")
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("HTTP服务器停止失败: %w", err)
	}
	appLogger.Info("HTTP服务器已停止")
	return nil
}
