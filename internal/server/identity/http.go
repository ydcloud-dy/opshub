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

package identity

import (
	"github.com/gin-gonic/gin"
	bizIdentity "github.com/ydcloud-dy/opshub/internal/biz/identity"
	dataIdentity "github.com/ydcloud-dy/opshub/internal/data/identity"
	svcIdentity "github.com/ydcloud-dy/opshub/internal/service/identity"
	"gorm.io/gorm"
)

// HTTPServer 身份认证HTTP服务
type HTTPServer struct {
	sourceService     *svcIdentity.IdentitySourceService
	appService        *svcIdentity.SSOApplicationService
	portalService     *svcIdentity.PortalService
	credentialService *svcIdentity.CredentialService
	permissionService *svcIdentity.PermissionService
	authLogService    *svcIdentity.AuthLogService
}

// NewIdentityServices 创建身份认证相关服务
func NewIdentityServices(db *gorm.DB) (*HTTPServer, error) {
	// 自动迁移数据库表
	if err := db.AutoMigrate(
		&bizIdentity.IdentitySource{},
		&bizIdentity.SSOApplication{},
		&bizIdentity.UserCredential{},
		&bizIdentity.AppPermission{},
		&bizIdentity.UserOAuthBinding{},
		&bizIdentity.AuthLog{},
		&bizIdentity.UserFavoriteApp{},
	); err != nil {
		return nil, err
	}

	// 创建仓库
	sourceRepo := dataIdentity.NewIdentitySourceRepo(db)
	appRepo := dataIdentity.NewSSOApplicationRepo(db)
	credentialRepo := dataIdentity.NewUserCredentialRepo(db)
	permissionRepo := dataIdentity.NewAppPermissionRepo(db)
	oauthBindingRepo := dataIdentity.NewUserOAuthBindingRepo(db)
	authLogRepo := dataIdentity.NewAuthLogRepo(db)
	favoriteRepo := dataIdentity.NewUserFavoriteAppRepo(db)

	// 创建用例
	sourceUseCase := bizIdentity.NewIdentitySourceUseCase(sourceRepo)
	appUseCase := bizIdentity.NewSSOApplicationUseCase(appRepo)
	credentialUseCase := bizIdentity.NewUserCredentialUseCase(credentialRepo)
	permissionUseCase := bizIdentity.NewAppPermissionUseCase(permissionRepo)
	_ = bizIdentity.NewUserOAuthBindingUseCase(oauthBindingRepo) // 后续OAuth功能使用
	authLogUseCase := bizIdentity.NewAuthLogUseCase(authLogRepo)
	favoriteUseCase := bizIdentity.NewUserFavoriteAppUseCase(favoriteRepo)

	// 创建服务
	sourceService := svcIdentity.NewIdentitySourceService(sourceUseCase)
	appService := svcIdentity.NewSSOApplicationService(appUseCase)
	portalService := svcIdentity.NewPortalService(appUseCase, permissionUseCase, favoriteUseCase, authLogUseCase)
	credentialService := svcIdentity.NewCredentialService(credentialUseCase, appUseCase)
	permissionService := svcIdentity.NewPermissionService(permissionUseCase)
	authLogService := svcIdentity.NewAuthLogService(authLogUseCase)

	return &HTTPServer{
		sourceService:     sourceService,
		appService:        appService,
		portalService:     portalService,
		credentialService: credentialService,
		permissionService: permissionService,
		authLogService:    authLogService,
	}, nil
}

// RegisterRoutes 注册路由
func (s *HTTPServer) RegisterRoutes(router *gin.RouterGroup) {
	identity := router.Group("/identity")
	{
		// 身份源管理
		sources := identity.Group("/sources")
		{
			sources.GET("", s.sourceService.ListSources)
			sources.GET("/enabled", s.sourceService.GetEnabledSources)
			sources.GET("/:id", s.sourceService.GetSource)
			sources.POST("", s.sourceService.CreateSource)
			sources.PUT("/:id", s.sourceService.UpdateSource)
			sources.DELETE("/:id", s.sourceService.DeleteSource)
		}

		// 应用管理
		apps := identity.Group("/apps")
		{
			apps.GET("", s.appService.ListApps)
			apps.GET("/templates", s.appService.GetTemplates)
			apps.GET("/categories", s.appService.GetCategories)
			apps.GET("/:id", s.appService.GetApp)
			apps.POST("", s.appService.CreateApp)
			apps.PUT("/:id", s.appService.UpdateApp)
			apps.DELETE("/:id", s.appService.DeleteApp)
		}

		// 应用门户
		portal := identity.Group("/portal")
		{
			portal.GET("/apps", s.portalService.GetPortalApps)
			portal.GET("/favorites", s.portalService.GetFavoriteApps)
			portal.POST("/access/:id", s.portalService.AccessApp)
			portal.POST("/favorite/:id", s.portalService.FavoriteApp)
		}

		// 凭证管理
		credentials := identity.Group("/credentials")
		{
			credentials.GET("", s.credentialService.ListCredentials)
			credentials.POST("", s.credentialService.CreateCredential)
			credentials.PUT("/:id", s.credentialService.UpdateCredential)
			credentials.DELETE("/:id", s.credentialService.DeleteCredential)
		}

		// 访问策略
		permissions := identity.Group("/permissions")
		{
			permissions.GET("", s.permissionService.ListPermissions)
			permissions.POST("", s.permissionService.CreatePermission)
			permissions.POST("/batch", s.permissionService.BatchCreatePermissions)
			permissions.DELETE("/:id", s.permissionService.DeletePermission)
			permissions.GET("/app/:id", s.permissionService.ListByApp)
		}

		// 认证日志
		logs := identity.Group("/logs")
		{
			logs.GET("", s.authLogService.ListLogs)
			logs.GET("/stats", s.authLogService.GetStats)
			logs.GET("/trend", s.authLogService.GetLoginTrend)
		}
	}
}
