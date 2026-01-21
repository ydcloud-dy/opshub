package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ydcloud-dy/opshub/cmd/root"
	"github.com/ydcloud-dy/opshub/internal/biz"
	"github.com/ydcloud-dy/opshub/internal/conf"
	dataPkg "github.com/ydcloud-dy/opshub/internal/data"
	"github.com/ydcloud-dy/opshub/internal/server"
	"github.com/ydcloud-dy/opshub/internal/service"
	rbacmodel "github.com/ydcloud-dy/opshub/internal/biz/rbac"
	auditmodel "github.com/ydcloud-dy/opshub/internal/biz/audit"
	"github.com/ydcloud-dy/opshub/plugins/kubernetes/data/models"
	k8smodel "github.com/ydcloud-dy/opshub/plugins/kubernetes/model"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"golang.org/x/crypto/bcrypt"
)

// 全局变量，用于在服务器生命周期内保持连接
var (
	globalData  *dataPkg.Data
	globalRedis *dataPkg.Redis
	globalHTTPServer *server.HTTPServer
)

var Cmd = &cobra.Command{
	Use:   "server",
	Short: "启动服务",
	Long:  `启动 OpsHub HTTP 服务器`,
	PreRun: func(cmd *cobra.Command, args []string) {
		// 从命令行参数覆盖配置
		if mode := viper.GetString("mode"); mode != "" {
			viper.Set("server.mode", mode)
		}
		if logLevel := viper.GetString("log-level"); logLevel != "" {
			viper.Set("log.level", logLevel)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// 加载配置
		cfg, err := runServer()
		if err != nil {
			fmt.Printf("启动服务失败: %v\n", err)
			os.Exit(1)
		}

		// 等待中断信号
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		fmt.Println("\n正在关闭服务...")
		ctx := context.Background()
		if err := stopServer(ctx, cfg); err != nil {
			fmt.Printf("关闭服务失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("服务已关闭")
	},
}

func init() {
	root.Cmd.AddCommand(Cmd)
}

func runServer() (*conf.Config, error) {
	// 加载配置
	cfg, err := conf.Load(root.GetConfigFile())
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 初始化日志
	logCfg := &appLogger.Config{
		Level:      cfg.Log.Level,
		Filename:   cfg.Log.Filename,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
		Console:    cfg.Log.Console,
	}
	if err := appLogger.Init(logCfg); err != nil {
		return nil, fmt.Errorf("初始化日志失败: %w", err)
	}
	defer appLogger.Sync()

	appLogger.Info("服务启动中...",
		zap.String("version", "1.0.0"),
		zap.String("mode", cfg.Server.Mode),
	)

	// 初始化数据层
	data, err := dataPkg.NewData(cfg)
	if err != nil {
		return nil, fmt.Errorf("初始化数据层失败: %w", err)
	}
	globalData = data // 保存到全局变量，防止被垃圾回收

	// 初始化Redis
	redis, err := dataPkg.NewRedis(cfg)
	if err != nil {
		return nil, fmt.Errorf("初始化Redis失败: %w", err)
	}
	globalRedis = redis // 保存到全局变量

	// 初始化业务层
	biz := biz.NewBiz(data, redis)

	// 初始化服务层
	svc := service.NewService(biz)

	// 自动迁移数据库表
	if err := autoMigrate(data.DB()); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	// 初始化默认数据
	if err := initDefaultData(data.DB()); err != nil {
		return nil, fmt.Errorf("初始化默认数据失败: %w", err)
	}

	// 初始化HTTP服务器
	httpServer := server.NewHTTPServer(cfg, svc, data.DB())
	globalHTTPServer = httpServer // 保存到全局变量

	// 启动服务器
	go func() {
		if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("HTTP服务器启动失败", zap.Error(err))
		}
	}()

	// 打印启动信息
	printStartupInfo(cfg)

	return cfg, nil
}

// autoMigrate 自动迁移数据库表
func autoMigrate(db *gorm.DB) error {
	// 自动迁移表结构
	if err := db.AutoMigrate(
		&rbacmodel.SysUser{},
		&rbacmodel.SysRole{},
		&rbacmodel.SysDepartment{},
		&rbacmodel.SysMenu{},
		&rbacmodel.SysUserRole{},
		&rbacmodel.SysRoleMenu{},
		&rbacmodel.SysPosition{},
		&rbacmodel.SysUserPosition{},
		&rbacmodel.SysRoleAssetPermission{},
		// Kubernetes 集群相关表
		&models.Cluster{},
		&k8smodel.UserKubeConfig{},
		&k8smodel.K8sUserRoleBinding{},
		// 审计日志相关表
		&auditmodel.SysOperationLog{},
		&auditmodel.SysLoginLog{},
		&auditmodel.SysDataLog{},
	); err != nil {
		return err
	}

	// 为用户表创建虚拟列和唯一索引
	// 问题：MySQL 唯一索引中多个 NULL 值被认为是不同的，无法正确约束
	// 解决：使用虚拟列 is_deleted (0=未删除, 1=已删除) 来创建唯一索引

	// 1. 检查并添加虚拟列 is_deleted
	var columnExists bool
	db.Raw("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'sys_user' AND COLUMN_NAME = 'is_deleted'").Scan(&columnExists)

	if !columnExists {
		// 虚拟列不存在，添加它
		if err := db.Exec("ALTER TABLE sys_user ADD COLUMN is_deleted TINYINT(1) GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN 0 ELSE 1 END) STORED").Error; err != nil {
			appLogger.Warn("添加虚拟列失败", zap.Error(err))
		} else {
			appLogger.Info("成功添加虚拟列 is_deleted")
		}
	}

	// 2. 删除旧的索引
	db.Exec("DROP INDEX idx_username_deleted_at ON sys_user")
	db.Exec("DROP INDEX idx_email_deleted_at ON sys_user")
	db.Exec("DROP INDEX idx_username_email_deleted_at ON sys_user")

	// 3. 创建新的唯一索引：用户名 + 邮箱 + is_deleted
	// 这样未删除的记录 (is_deleted=0) 中，username + email 的组合必须唯一
	// 已删除的记录 (is_deleted=1) 不会阻止新记录创建
	if err := db.Exec("CREATE UNIQUE INDEX idx_username_email_is_deleted ON sys_user(username, email, is_deleted)").Error; err != nil {
		appLogger.Warn("创建用户名邮箱唯一索引失败", zap.Error(err))
	} else {
		appLogger.Info("成功创建用户名邮箱联合唯一索引")
	}

	return nil
}

// initDefaultData 初始化默认数据
func initDefaultData(db *gorm.DB) error {
	// 检查是否已有管理员用户
	var count int64
	db.Model(&rbacmodel.SysUser{}).Where("username = ?", "admin").Count(&count)
	if count > 0 {
		return nil // 已存在管理员，无需初始化
	}

	appLogger.Info("开始初始化默认数据...")

	// 创建默认部门
	dept := &rbacmodel.SysDepartment{
		Name:     "总公司",
		Code:     "HQ",
		ParentID: 0,
		Sort:     0,
		Status:   1,
	}
	if err := db.Create(dept).Error; err != nil {
		return fmt.Errorf("创建默认部门失败: %w", err)
	}

	// 创建管理员角色
	adminRole := &rbacmodel.SysRole{
		Name:        "超级管理员",
		Code:        "admin",
		Description: "系统超级管理员，拥有所有权限",
		Sort:        0,
		Status:      1,
	}
	if err := db.Create(adminRole).Error; err != nil {
		return fmt.Errorf("创建管理员角色失败: %w", err)
	}

	// 创建管理员用户（密码：123456）
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("加密密码失败: %w", err)
	}

	adminUser := &rbacmodel.SysUser{
		Username:    "admin",
		Password:    string(hashedPassword),
		RealName:    "系统管理员",
		Email:       "admin@opshub.com",
		Status:      1,
		DepartmentID: dept.ID,
	}
	if err := db.Create(adminUser).Error; err != nil {
		return fmt.Errorf("创建管理员用户失败: %w", err)
	}

	// 为管理员分配角色
	if err := db.Exec("INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)",
		adminUser.ID, adminRole.ID).Error; err != nil {
		return fmt.Errorf("分配管理员角色失败: %w", err)
	}

	// 创建默认菜单
	menus := []*rbacmodel.SysMenu{
		{Name: "首页", Code: "dashboard", Type: 2, ParentID: 0, Path: "/dashboard", Component: "Dashboard", Icon: "HomeFilled", Sort: 0, Visible: 1, Status: 1},
		{Name: "系统管理", Code: "system", Type: 1, ParentID: 0, Path: "/system", Icon: "Setting", Sort: 100, Visible: 1, Status: 1},
		{Name: "用户管理", Code: "users", Type: 2, ParentID: 0, Path: "/users", Component: "system/Users", Icon: "User", Sort: 1, Visible: 1, Status: 1},
		{Name: "角色管理", Code: "roles", Type: 2, ParentID: 0, Path: "/roles", Component: "system/Roles", Icon: "UserFilled", Sort: 2, Visible: 1, Status: 1},
		{Name: "部门管理", Code: "departments", Type: 2, ParentID: 0, Path: "/departments", Component: "system/Departments", Icon: "OfficeBuilding", Sort: 3, Visible: 1, Status: 1},
		{Name: "菜单管理", Code: "menus", Type: 2, ParentID: 0, Path: "/menus", Component: "system/Menus", Icon: "Menu", Sort: 4, Visible: 1, Status: 1},
	}

	// 存储菜单ID映射，用于设置父子关系
	menuIDMap := make(map[string]uint)

	for _, menu := range menus {
		if err := db.Create(menu).Error; err != nil {
			return fmt.Errorf("创建默认菜单失败: %w", err)
		}
		// 为管理员角色分配所有菜单权限
		db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, menu.ID)
		menuIDMap[menu.Code] = menu.ID
	}

	// 创建操作审计菜单（如果系统管理菜单已存在，使用其ID作为父菜单）
	var systemMenuID uint
	db.Model(&rbacmodel.SysMenu{}).Where("code = ? AND name = ?", "system", "系统管理").Pluck("id", &systemMenuID)

	if systemMenuID == 0 {
		// 如果系统管理菜单不存在，创建操作审计为顶级菜单
		systemMenuID = 0
	}

	auditMenus := []*rbacmodel.SysMenu{
		{Name: "岗位信息", Code: "position-info", Type: 2, ParentID: systemMenuID, Path: "/position-info", Component: "system/PositionInfo", Icon: "Avatar", Sort: 5, Visible: 1, Status: 1},
		{Name: "操作审计", Code: "audit", Type: 1, ParentID: 0, Path: "/audit", Icon: "Document", Sort: 50, Visible: 1, Status: 1},
	}

	for _, menu := range auditMenus {
		var existingCount int64
		db.Model(&rbacmodel.SysMenu{}).Where("code = ?", menu.Code).Count(&existingCount)
		if existingCount == 0 {
			if err := db.Create(menu).Error; err != nil {
				return fmt.Errorf("创建审计菜单失败: %w", err)
			}
			// 为管理员角色分配菜单权限
			db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, menu.ID)
			menuIDMap[menu.Code] = menu.ID
		}
	}

	// 获取操作审计菜单ID
	var auditMenuID uint
	db.Model(&rbacmodel.SysMenu{}).Where("code = ?", "audit").Pluck("id", &auditMenuID)

	if auditMenuID > 0 {
		// 创建操作审计子菜单
		auditSubMenus := []*rbacmodel.SysMenu{
			{Name: "操作日志", Code: "operation-logs", Type: 2, ParentID: auditMenuID, Path: "/audit/operation-logs", Component: "audit/OperationLogs", Icon: "Document", Sort: 1, Visible: 1, Status: 1},
			{Name: "登录日志", Code: "login-logs", Type: 2, ParentID: auditMenuID, Path: "/audit/login-logs", Component: "audit/LoginLogs", Icon: "CircleCheck", Sort: 2, Visible: 1, Status: 1},
			{Name: "数据日志", Code: "data-logs", Type: 2, ParentID: auditMenuID, Path: "/audit/data-logs", Component: "audit/DataLogs", Icon: "DataLine", Sort: 3, Visible: 1, Status: 1},
		}

		for _, menu := range auditSubMenus {
			var existingCount int64
			db.Model(&rbacmodel.SysMenu{}).Where("code = ?", menu.Code).Count(&existingCount)
			if existingCount == 0 {
				if err := db.Create(menu).Error; err != nil {
					return fmt.Errorf("创建审计子菜单失败: %w", err)
				}
				// 为管理员角色分配菜单权限
				db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, menu.ID)
			}
		}
	}

	appLogger.Info("默认数据初始化完成")
	appLogger.Info("默认管理员账号: admin")
	appLogger.Info("默认管理员密码: 123456")

	return nil
}

func stopServer(ctx context.Context, cfg *conf.Config) error {
	appLogger.Info("服务正在关闭...")

	// 停止HTTP服务器
	if globalHTTPServer != nil {
		if err := globalHTTPServer.Stop(ctx); err != nil {
			appLogger.Error("停止HTTP服务器失败", zap.Error(err))
		}
	}

	// 关闭数据库连接
	if globalData != nil {
		if err := globalData.Close(); err != nil {
			appLogger.Error("关闭数据库连接失败", zap.Error(err))
		}
	}

	// 关闭Redis连接
	if globalRedis != nil {
		if err := globalRedis.Close(); err != nil {
			appLogger.Error("关闭Redis连接失败", zap.Error(err))
		}
	}

	return nil
}

func printStartupInfo(cfg *conf.Config) {
	addr := fmt.Sprintf("%s:%d", "0.0.0.0", cfg.Server.HttpPort)

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("       OpsHub 运维管理平台启动成功")
	fmt.Println("========================================")
	fmt.Printf("版本:     1.0.0\n")
	fmt.Printf("模式:     %s\n", cfg.Server.Mode)
	fmt.Printf("监听地址: http://%s\n", addr)
	fmt.Printf("健康检查: http://%s/health\n", addr)
	fmt.Printf("API文档:  http://%s/swagger/index.html\n", addr)
	fmt.Println("========================================")
	fmt.Println()
}
