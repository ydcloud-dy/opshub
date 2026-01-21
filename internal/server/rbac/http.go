package rbac

import (
	"github.com/gin-gonic/gin"
	auditbiz "github.com/ydcloud-dy/opshub/internal/biz/audit"
	auditdata "github.com/ydcloud-dy/opshub/internal/data/audit"
	rbacService "github.com/ydcloud-dy/opshub/internal/service/rbac"
	rbacdata "github.com/ydcloud-dy/opshub/internal/data/rbac"
	rbacbiz "github.com/ydcloud-dy/opshub/internal/biz/rbac"
	"gorm.io/gorm"
)

type HTTPServer struct {
	userService            *rbacService.UserService
	roleService            *rbacService.RoleService
	departmentService      *rbacService.DepartmentService
	menuService            *rbacService.MenuService
	positionService        *rbacService.PositionService
	captchaService         *rbacService.CaptchaService
	assetPermissionService *rbacService.AssetPermissionService
	authMiddleware         *rbacService.AuthMiddleware
}

func NewHTTPServer(
	userService *rbacService.UserService,
	roleService *rbacService.RoleService,
	departmentService *rbacService.DepartmentService,
	menuService *rbacService.MenuService,
	positionService *rbacService.PositionService,
	captchaService *rbacService.CaptchaService,
	assetPermissionService *rbacService.AssetPermissionService,
	authMiddleware *rbacService.AuthMiddleware,
) *HTTPServer {
	return &HTTPServer{
		userService:            userService,
		roleService:            roleService,
		departmentService:      departmentService,
		menuService:            menuService,
		positionService:        positionService,
		captchaService:         captchaService,
		assetPermissionService: assetPermissionService,
		authMiddleware:         authMiddleware,
	}
}

func (s *HTTPServer) RegisterRoutes(r *gin.Engine) {
	// 公开路由
	public := r.Group("/api/v1/public")
	{
		public.POST("/login", s.userService.Login)
	}

	// 验证码路由（无需认证）
	captcha := r.Group("/api/v1/captcha")
	{
		captcha.GET("", s.captchaService.GetCaptcha)
		captcha.POST("/verify", s.captchaService.VerifyCaptcha)
	}

	// 需要认证的路由
	auth := r.Group("/api/v1")
	auth.Use(s.authMiddleware.AuthRequired())
	{
		// 用户相关
		auth.GET("/profile", s.userService.GetProfile)
		auth.PUT("/profile/password", s.userService.ChangePassword)

		// 用户管理
		users := auth.Group("/users")
		{
			users.GET("", s.userService.ListUsers)
			users.GET("/:id", s.userService.GetUser)
			users.POST("", s.userService.CreateUser)
			users.PUT("/:id", s.userService.UpdateUser)
			users.DELETE("/:id", s.userService.DeleteUser)
			users.POST("/:id/roles", s.userService.AssignUserRoles)
			users.POST("/:id/positions", s.userService.AssignUserPositions)
			users.PUT("/:id/reset-password", s.userService.ResetPassword)
		}

		// 角色管理
		roles := auth.Group("/roles")
		{
			roles.GET("", s.roleService.ListRoles)
			roles.GET("/all", s.roleService.GetAllRoles)
			roles.GET("/:id", s.roleService.GetRole)
			roles.POST("", s.roleService.CreateRole)
			roles.PUT("/:id", s.roleService.UpdateRole)
			roles.DELETE("/:id", s.roleService.DeleteRole)
			roles.POST("/:id/menus", s.roleService.AssignRoleMenus)
		}

		// 部门管理
		departments := auth.Group("/departments")
		{
			departments.GET("/tree", s.departmentService.GetDepartmentTree)
			departments.GET("/parent-options", s.departmentService.GetParentOptions)
			departments.POST("", s.departmentService.CreateDepartment)
			departments.GET("/:id", s.departmentService.GetDepartment)
			departments.PUT("/:id", s.departmentService.UpdateDepartment)
			departments.DELETE("/:id", s.departmentService.DeleteDepartment)
		}

		// 菜单管理
		menus := auth.Group("/menus")
		{
			menus.GET("/tree", s.menuService.GetMenuTree)
			menus.GET("/user", s.menuService.GetUserMenu)
			menus.GET("/:id", s.menuService.GetMenu)
			menus.POST("", s.menuService.CreateMenu)
			menus.PUT("/:id", s.menuService.UpdateMenu)
			menus.DELETE("/:id", s.menuService.DeleteMenu)
		}

		// 岗位管理
		positions := auth.Group("/positions")
		{
			positions.GET("", s.positionService.ListPositions)
			positions.GET("/:id", s.positionService.GetPosition)
			positions.POST("", s.positionService.CreatePosition)
			positions.PUT("/:id", s.positionService.UpdatePosition)
			positions.DELETE("/:id", s.positionService.DeletePosition)
			positions.GET("/:id/users", s.positionService.GetPositionUsers)
			positions.POST("/:id/users", s.positionService.AssignUsersToPosition)
			positions.DELETE("/:id/users/:userId", s.positionService.RemoveUserFromPosition)
		}

		// 资产权限管理
		assetPermissions := auth.Group("/asset-permissions")
		{
			assetPermissions.GET("", s.assetPermissionService.ListAssetPermissions)
			assetPermissions.POST("", s.assetPermissionService.CreateAssetPermission)
			// 具体路由必须放在通用 /:id 路由之前
			assetPermissions.GET("/role/:roleId", s.assetPermissionService.GetAssetPermissionsByRole)
			assetPermissions.GET("/group/:assetGroupId", s.assetPermissionService.GetAssetPermissionsByGroup)
			assetPermissions.GET("/user/host", s.assetPermissionService.GetUserHostPermissions)
			// 通用 /:id 路由必须放在最后
			assetPermissions.GET("/:id", s.assetPermissionService.GetAssetPermissionDetail)
			assetPermissions.PUT("/:id", s.assetPermissionService.UpdateAssetPermission)
			assetPermissions.DELETE("/:id", s.assetPermissionService.DeleteAssetPermission)
			// 删除分组权限用空路径（没有 :id）
			assetPermissions.DELETE("", s.assetPermissionService.DeleteAssetPermissionByRoleAndGroup)
		}
	}
}

// 依赖注入函数
func NewRBACServices(db *gorm.DB, jwtSecret string) (
	*rbacService.UserService,
	*rbacService.RoleService,
	*rbacService.DepartmentService,
	*rbacService.MenuService,
	*rbacService.PositionService,
	*rbacService.CaptchaService,
	*rbacService.AssetPermissionService,
	*rbacService.AuthMiddleware,
) {
	// 初始化Repository
	userRepo := rbacdata.NewUserRepo(db)
	roleRepo := rbacdata.NewRoleRepo(db)
	deptRepo := rbacdata.NewDepartmentRepo(db)
	menuRepo := rbacdata.NewMenuRepo(db)
	positionRepo := rbacdata.NewPositionRepo(db)
	assetPermissionRepo := rbacdata.NewAssetPermissionRepo(db)

	// 初始化Audit Repository
	loginLogRepo := auditdata.NewLoginLogRepo(db)

	// 初始化UseCase
	userUseCase := rbacbiz.NewUserUseCase(userRepo)
	roleUseCase := rbacbiz.NewRoleUseCase(roleRepo)
	deptUseCase := rbacbiz.NewDepartmentUseCase(deptRepo)
	menuUseCase := rbacbiz.NewMenuUseCase(menuRepo)
	positionUseCase := rbacbiz.NewPositionUseCase(positionRepo)
	assetPermissionUseCase := rbacbiz.NewAssetPermissionUseCase(assetPermissionRepo)

	// 初始化Audit UseCase
	loginLogUseCase := auditbiz.NewLoginLogUseCase(loginLogRepo)

	// 初始化Service
	authService := rbacService.NewAuthService(jwtSecret, roleUseCase)
	userService := rbacService.NewUserService(userUseCase, authService)
	roleService := rbacService.NewRoleService(roleUseCase)
	departmentService := rbacService.NewDepartmentService(deptUseCase)
	menuService := rbacService.NewMenuService(menuUseCase, roleUseCase)
	positionService := rbacService.NewPositionService(positionUseCase)
	captchaService := rbacService.NewCaptchaService()
	assetPermissionService := rbacService.NewAssetPermissionService(assetPermissionUseCase)
	authMiddleware := rbacService.NewAuthMiddleware(authService)

	// 设置验证码服务到用户服务
	userService.SetCaptchaService(captchaService)

	// 设置登录日志用例到用户服务
	userService.SetLoginLogUseCase(loginLogUseCase)

	return userService, roleService, departmentService, menuService, positionService, captchaService, assetPermissionService, authMiddleware
}
