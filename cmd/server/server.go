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
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ydcloud-dy/opshub/cmd/root"
	"github.com/ydcloud-dy/opshub/internal/biz"
	assetmodel "github.com/ydcloud-dy/opshub/internal/biz/asset"
	auditmodel "github.com/ydcloud-dy/opshub/internal/biz/audit"
	mfamodel "github.com/ydcloud-dy/opshub/internal/biz/mfa"
	rbacmodel "github.com/ydcloud-dy/opshub/internal/biz/rbac"
	systemmodel "github.com/ydcloud-dy/opshub/internal/biz/system"
	"github.com/ydcloud-dy/opshub/internal/conf"
	dataPkg "github.com/ydcloud-dy/opshub/internal/data"
	systemdata "github.com/ydcloud-dy/opshub/internal/data/system"
	"github.com/ydcloud-dy/opshub/internal/server"
	"github.com/ydcloud-dy/opshub/internal/service"
	rbacservice "github.com/ydcloud-dy/opshub/internal/service/rbac"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"github.com/ydcloud-dy/opshub/plugins/kubernetes/data/models"
	k8smodel "github.com/ydcloud-dy/opshub/plugins/kubernetes/model"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 全局变量，用于在服务器生命周期内保持连接
var (
	globalData       *dataPkg.Data
	globalRedis      *dataPkg.Redis
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

	// 初始化验证码存储（使用 Redis）
	rbacservice.InitCaptchaStore(redis.Get())

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
	if err := ensureAssetAgentMenu(data.DB()); err != nil {
		appLogger.Warn("补充资产Agent管理菜单失败", zap.Error(err))
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
		// 系统配置相关表
		&systemmodel.SysConfig{},
		&systemmodel.SysUserLoginAttempt{},
		// 资产管理相关表
		&assetmodel.Host{},
		&assetmodel.Credential{},
		&assetmodel.AssetGroup{},
		&assetmodel.CloudAccount{},
		&assetmodel.TerminalSession{},
		// Kubernetes 集群相关表
		&models.Cluster{},
		&k8smodel.UserKubeConfig{},
		&k8smodel.K8sUserRoleBinding{},
		// 审计日志相关表
		&auditmodel.SysOperationLog{},
		&auditmodel.SysLoginLog{},
		&auditmodel.SysDataLog{},
		// MFA相关表
		&mfamodel.UserMFA{},
		&mfamodel.MFALog{},
		&mfamodel.TrustedDevice{},
	); err != nil {
		return err
	}
	if err := ensureSysUserNotifyColumns(db); err != nil {
		appLogger.Warn("补充用户通知标识字段失败", zap.Error(err))
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

func ensureSysUserNotifyColumns(db *gorm.DB) error {
	columns := []struct {
		name string
		ddl  string
	}{
		{"notify_user_id", "ALTER TABLE sys_user ADD COLUMN notify_user_id VARCHAR(150) NULL COMMENT '第三方通知用户标识' AFTER phone"},
		{"feishu_user_id", "ALTER TABLE sys_user ADD COLUMN feishu_user_id VARCHAR(100) NULL COMMENT '飞书用户ID' AFTER notify_user_id"},
		{"feishu_open_id", "ALTER TABLE sys_user ADD COLUMN feishu_open_id VARCHAR(100) NULL COMMENT '飞书OpenID' AFTER feishu_user_id"},
		{"dingtalk_user_id", "ALTER TABLE sys_user ADD COLUMN dingtalk_user_id VARCHAR(100) NULL COMMENT '钉钉用户ID' AFTER feishu_open_id"},
		{"wecom_user_id", "ALTER TABLE sys_user ADD COLUMN wecom_user_id VARCHAR(100) NULL COMMENT '企业微信用户ID' AFTER dingtalk_user_id"},
	}
	for _, column := range columns {
		var count int64
		if err := db.Raw("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'sys_user' AND COLUMN_NAME = ?", column.name).Scan(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if err := db.Exec(column.ddl).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureAssetAgentMenu(db *gorm.DB) error {
	var assetMenu rbacmodel.SysMenu
	if err := db.Where("code = ?", "asset-management").First(&assetMenu).Error; err != nil {
		return err
	}

	agentMenu := rbacmodel.SysMenu{
		Name:      "Agent管理",
		Code:      "asset-agent-management",
		Type:      2,
		ParentID:  assetMenu.ID,
		Path:      "/asset/agents",
		Component: "asset/Agents",
		Icon:      "Connection",
		Sort:      2,
		Visible:   1,
		Status:    1,
	}

	var existing rbacmodel.SysMenu
	err := db.Where("code = ?", agentMenu.Code).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := db.Create(&agentMenu).Error; err != nil {
			return err
		}
		existing = agentMenu
	} else if err != nil {
		return err
	} else {
		updates := map[string]interface{}{
			"name":      agentMenu.Name,
			"type":      agentMenu.Type,
			"parent_id": agentMenu.ParentID,
			"path":      agentMenu.Path,
			"component": agentMenu.Component,
			"icon":      agentMenu.Icon,
			"sort":      agentMenu.Sort,
			"visible":   agentMenu.Visible,
			"status":    agentMenu.Status,
		}
		if err := db.Model(&existing).Updates(updates).Error; err != nil {
			return err
		}
	}

	var roles []rbacmodel.SysRole
	if err := db.Where("code IN ?", []string{"admin", "user"}).Find(&roles).Error; err != nil {
		return err
	}
	for _, role := range roles {
		if err := db.Exec("INSERT IGNORE INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", role.ID, existing.ID).Error; err != nil {
			return err
		}
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
		Username:     "admin",
		Password:     string(hashedPassword),
		RealName:     "系统管理员",
		Email:        "admin@opshub.com",
		Status:       1,
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

	// 辅助函数：创建菜单并分配权限
	createMenuWithPermission := func(menu *rbacmodel.SysMenu) error {
		if err := db.Create(menu).Error; err != nil {
			return err
		}
		db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, menu.ID)
		return nil
	}

	// ========== 1. 仪表盘 ==========
	dashboardMenu := &rbacmodel.SysMenu{Name: "仪表盘", Code: "dashboard", Type: 1, ParentID: 0, Path: "/dashboard", Icon: "HomeFilled", Sort: 0, Visible: 1, Status: 1}
	if err := createMenuWithPermission(dashboardMenu); err != nil {
		return fmt.Errorf("创建仪表盘菜单失败: %w", err)
	}

	// ========== 2. 资产管理 ==========
	assetMenu := &rbacmodel.SysMenu{Name: "资产管理", Code: "asset-management", Type: 1, ParentID: 0, Path: "/asset", Icon: "Coin", Sort: 1, Visible: 1, Status: 1}
	if err := createMenuWithPermission(assetMenu); err != nil {
		return fmt.Errorf("创建资产管理菜单失败: %w", err)
	}

	assetSubMenus := []*rbacmodel.SysMenu{
		{Name: "主机管理", Code: "host-management", Type: 2, ParentID: assetMenu.ID, Path: "/asset/hosts", Component: "asset/Hosts", Icon: "Monitor", Sort: 1, Visible: 1, Status: 1},
		{Name: "凭据管理", Code: "asset:credentials", Type: 3, ParentID: assetMenu.ID, Path: "/asset/credentials", Component: "asset/Credentials", Icon: "Lock", Sort: 2, Visible: 1, Status: 1},
		{Name: "业务分组", Code: "business-group", Type: 2, ParentID: assetMenu.ID, Path: "/asset/groups", Component: "asset/Groups", Icon: "Collection", Sort: 3, Visible: 1, Status: 1},
		{Name: "云账号管理", Code: "cloud-accounts", Type: 2, ParentID: assetMenu.ID, Path: "/asset/cloud-accounts", Component: "asset/CloudAccounts", Icon: "Cloudy", Sort: 4, Visible: 1, Status: 1},
		{Name: "终端审计", Code: "asset_terminal_audit", Type: 2, ParentID: assetMenu.ID, Path: "/asset/terminal-audit", Icon: "View", Sort: 5, Visible: 1, Status: 1},
		{Name: "权限配置", Code: "asset_permission", Type: 2, ParentID: assetMenu.ID, Path: "/asset/permissions", Component: "views/asset/AssetPermission.vue", Icon: "Lock", Sort: 6, Visible: 1, Status: 1},
	}
	for _, menu := range assetSubMenus {
		if err := createMenuWithPermission(menu); err != nil {
			return fmt.Errorf("创建资产管理子菜单失败: %w", err)
		}
	}

	// ========== 3. 身份认证（暂不开放，如需启用请取消注释） ==========
	// identityMenu := &rbacmodel.SysMenu{Name: "身份认证", Code: "identity", Type: 1, ParentID: 0, Path: "/identity", Icon: "Key", Sort: 2, Visible: 1, Status: 1}
	// if err := createMenuWithPermission(identityMenu); err != nil {
	// 	return fmt.Errorf("创建身份认证菜单失败: %w", err)
	// }
	// identitySubMenus := []*rbacmodel.SysMenu{
	// 	{Name: "身份源管理", Code: "identity_sources", Type: 2, ParentID: identityMenu.ID, Path: "/identity/sources", Component: "identity/IdentitySources", Icon: "User", Sort: 1, Visible: 1, Status: 1},
	// 	{Name: "应用管理", Code: "identity_apps", Type: 2, ParentID: identityMenu.ID, Path: "/identity/apps", Component: "identity/SSOApplications", Icon: "Grid", Sort: 2, Visible: 1, Status: 1},
	// 	{Name: "凭证管理", Code: "identity_credentials", Type: 2, ParentID: identityMenu.ID, Path: "/identity/credentials", Component: "identity/Credentials", Icon: "Lock", Sort: 3, Visible: 1, Status: 1},
	// 	{Name: "访问策略", Code: "identity_permissions", Type: 2, ParentID: identityMenu.ID, Path: "/identity/permissions", Component: "identity/Permissions", Icon: "Key", Sort: 4, Visible: 1, Status: 1},
	// 	{Name: "认证日志", Code: "identity_logs", Type: 2, ParentID: identityMenu.ID, Path: "/identity/logs", Component: "identity/AuthLogs", Icon: "Document", Sort: 5, Visible: 1, Status: 1},
	// 	{Name: "应用门户", Code: "identity_portal", Type: 2, ParentID: identityMenu.ID, Path: "/identity/portal", Component: "identity/Portal", Icon: "Menu", Sort: 6, Visible: 1, Status: 1},
	// }
	// for _, menu := range identitySubMenus {
	// 	if err := createMenuWithPermission(menu); err != nil {
	// 		return fmt.Errorf("创建身份认证子菜单失败: %w", err)
	// 	}
	// }

	// ========== 4. 操作审计 ==========
	auditMenu := &rbacmodel.SysMenu{Name: "操作审计", Code: "audit", Type: 1, ParentID: 0, Path: "/audit", Icon: "Document", Sort: 50, Visible: 1, Status: 1}
	if err := createMenuWithPermission(auditMenu); err != nil {
		return fmt.Errorf("创建操作审计菜单失败: %w", err)
	}

	auditSubMenus := []*rbacmodel.SysMenu{
		{Name: "操作日志", Code: "operation-logs", Type: 2, ParentID: auditMenu.ID, Path: "/audit/operation-logs", Component: "audit/OperationLogs", Icon: "Document", Sort: 1, Visible: 1, Status: 1},
		{Name: "登录日志", Code: "login-logs", Type: 2, ParentID: auditMenu.ID, Path: "/audit/login-logs", Component: "audit/LoginLogs", Icon: "CircleCheck", Sort: 2, Visible: 1, Status: 1},
	}
	for _, menu := range auditSubMenus {
		if err := createMenuWithPermission(menu); err != nil {
			return fmt.Errorf("创建操作审计子菜单失败: %w", err)
		}
	}

	// ========== 5. 插件管理 ==========
	pluginMenu := &rbacmodel.SysMenu{Name: "插件管理", Code: "plugin", Type: 1, ParentID: 0, Path: "/plugin", Icon: "Grid", Sort: 80, Visible: 1, Status: 1}
	if err := createMenuWithPermission(pluginMenu); err != nil {
		return fmt.Errorf("创建插件管理菜单失败: %w", err)
	}

	pluginSubMenus := []*rbacmodel.SysMenu{
		{Name: "插件列表", Code: "plugin-list", Type: 2, ParentID: pluginMenu.ID, Path: "/plugin/list", Component: "plugin/PluginList", Icon: "Grid", Sort: 1, Visible: 1, Status: 1},
		{Name: "插件安装", Code: "plugin-install", Type: 2, ParentID: pluginMenu.ID, Path: "/plugin/install", Component: "plugin/PluginInstall", Icon: "Upload", Sort: 2, Visible: 1, Status: 1},
	}
	for _, menu := range pluginSubMenus {
		if err := createMenuWithPermission(menu); err != nil {
			return fmt.Errorf("创建插件管理子菜单失败: %w", err)
		}
	}

	// ========== 6. 系统管理 ==========
	systemMenu := &rbacmodel.SysMenu{Name: "系统管理", Code: "system", Type: 1, ParentID: 0, Path: "/system", Icon: "Setting", Sort: 100, Visible: 1, Status: 1}
	if err := createMenuWithPermission(systemMenu); err != nil {
		return fmt.Errorf("创建系统管理菜单失败: %w", err)
	}

	systemSubMenus := []*rbacmodel.SysMenu{
		{Name: "用户管理", Code: "users", Type: 2, ParentID: systemMenu.ID, Path: "/users", Component: "system/Users", Icon: "User", Sort: 1, Visible: 1, Status: 1},
		{Name: "角色管理", Code: "roles", Type: 2, ParentID: systemMenu.ID, Path: "/roles", Component: "system/Roles", Icon: "UserFilled", Sort: 2, Visible: 1, Status: 1},
		{Name: "菜单管理", Code: "menus", Type: 2, ParentID: systemMenu.ID, Path: "/menus", Component: "system/Menus", Icon: "Menu", Sort: 3, Visible: 1, Status: 1},
		{Name: "部门信息", Code: "dept-info", Type: 2, ParentID: systemMenu.ID, Path: "/dept-info", Component: "system/DeptInfo", Icon: "OfficeBuilding", Sort: 4, Visible: 1, Status: 1},
		{Name: "岗位信息", Code: "position-info", Type: 2, ParentID: systemMenu.ID, Path: "/position-info", Component: "system/PositionInfo", Icon: "Avatar", Sort: 5, Visible: 1, Status: 1},
		{Name: "系统配置", Code: "system-config", Type: 2, ParentID: systemMenu.ID, Path: "/system-config", Component: "system/SystemConfig", Icon: "Setting", Sort: 6, Visible: 1, Status: 1},
	}
	for _, menu := range systemSubMenus {
		if err := createMenuWithPermission(menu); err != nil {
			return fmt.Errorf("创建系统管理子菜单失败: %w", err)
		}
	}

	appLogger.Info("默认数据初始化完成")
	appLogger.Info("默认管理员账号: admin")
	appLogger.Info("默认管理员密码: 123456")

	// 初始化系统默认配置
	configRepo := systemdata.NewConfigRepo(db)
	if err := configRepo.InitDefaultConfigs(context.Background()); err != nil {
		appLogger.Warn("初始化系统配置失败", zap.Error(err))
	} else {
		appLogger.Info("系统默认配置初始化完成")
	}

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
