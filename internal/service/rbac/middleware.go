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

package rbac

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

const (
	UserIdKey   = "user_id"
	UsernameKey = "username"
)

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) uint {
	if userID, exists := c.Get(UserIdKey); exists {
		if id, ok := userID.(uint); ok {
			return id
		}
	}
	return 0
}

// GetUsername 从上下文获取用户名
func GetUsername(c *gin.Context) string {
	if username, exists := c.Get(UsernameKey); exists {
		if name, ok := username.(string); ok {
			return name
		}
	}
	return ""
}

// AuthMiddleware JWT认证中间件
type AuthMiddleware struct {
	authService         *AuthService
	assetPermissionRepo rbac.AssetPermissionRepo
}

func NewAuthMiddleware(authService *AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// SetAssetPermissionRepo 设置资产权限仓储
func (m *AuthMiddleware) SetAssetPermissionRepo(repo rbac.AssetPermissionRepo) {
	m.assetPermissionRepo = repo
}

// AuthRequired JWT认证
func (m *AuthMiddleware) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		// 优先从 Authorization header 获取 token
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			}
		}

		// 如果 header 中没有，尝试从 query 参数获取（用于 WebSocket 连接）
		if token == "" {
			token = c.Query("token")
		}

		// 从 session cookie 获取（用于 OAuth2 SSO 流程）
		if token == "" {
			if cookieToken, err := c.Cookie("opshub_session"); err == nil && cookieToken != "" {
				token = cookieToken
			}
		}

		// 如果都没有，返回未授权
		if token == "" {
			response.ErrorCode(c, http.StatusUnauthorized, "未登录")
			c.Abort()
			return
		}

		claims, err := m.authService.ParseToken(token)
		if err != nil {
			response.ErrorCode(c, http.StatusUnauthorized, "token无效或已过期")
			c.Abort()
			return
		}

		c.Set(UserIdKey, claims.UserID)
		c.Set(UsernameKey, claims.Username)
		c.Set("userID", claims.UserID) // 兼容 OAuth2 使用的 key
		c.Next()
	}
}

// ReadonlyUserRequired 阻止 test 演示账号执行写操作或访问敏感查询接口。
func (m *AuthMiddleware) ReadonlyUserRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !IsReadonlyUser(c) {
			c.Next()
			return
		}

		c.Set("readonly_user", true)

		if isReadonlySafeRequest(c.Request.Method, c.Request.URL.Path) {
			c.Next()
			return
		}

		response.ErrorCode(c, http.StatusForbidden, "test账号仅允许查看，不能执行修改、测试、下载、终端或敏感配置操作")
		c.Abort()
	}
}

func IsReadonlyUser(c *gin.Context) bool {
	return strings.EqualFold(strings.TrimSpace(GetUsername(c)), "test")
}

func isReadonlySafeRequest(method, path string) bool {
	if method == http.MethodOptions || method == http.MethodHead {
		return true
	}
	if method != http.MethodGet {
		return false
	}

	allowedExactPaths := map[string]struct{}{
		"/api/v1/menus/user":             {},
		"/api/v1/profile":                {},
		"/api/v1/mfa/status":             {},
		"/api/v1/system/config/basic":    {},
		"/api/v1/system/config/security": {},
	}
	if _, ok := allowedExactPaths[path]; ok {
		return true
	}

	blockedExactPaths := map[string]struct{}{
		"/api/v1/system/config/ldap": {},
		"/api/v1/credentials/all":    {},
		"/api/v1/cloud-accounts/all": {},
	}
	if _, ok := blockedExactPaths[path]; ok {
		return false
	}

	blockedContains := []string{
		"/api/v1/identity",
		"/asset/terminal",
		"/dns-providers",
		"/deploy-configs",
		"/files/download",
		"/template/download",
		"/terminal-sessions/",
		"/terminal/sessions/",
		"/shell/",
		"/arthas/ws",
		"/pods/files/download",
		"/certificates/",
		"/config",
		"/kubeconfig",
		"/secrets",
		"/yaml",
		"/cloudtty",
		"/pods/files",
		"/credentials",
		"/cloud-accounts",
		"/private-key",
		"/secret",
		"/token",
		"/password",
	}
	for _, fragment := range blockedContains {
		if strings.Contains(path, fragment) {
			return false
		}
	}

	return true
}

// OptionalAuth 可选认证中间件（不强制要求登录，用于OAuth2授权端点）
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		// 从 Authorization header 获取 token
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			}
		}

		// 从 query 参数获取
		if token == "" {
			token = c.Query("token")
		}

		// 从 session cookie 获取（用于 OAuth2 SSO 流程，浏览器重定向时会自动携带）
		if token == "" {
			if cookieToken, err := c.Cookie("opshub_session"); err == nil && cookieToken != "" {
				token = cookieToken
			}
		}

		// 如果有 token，尝试解析
		if token != "" {
			claims, err := m.authService.ParseToken(token)
			if err == nil {
				c.Set(UserIdKey, claims.UserID)
				c.Set(UsernameKey, claims.Username)
				c.Set("userID", claims.UserID)
			}
		}

		c.Next()
	}
}

// RequireAdmin 检查是否为管理员
func (m *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == 0 {
			response.ErrorCode(c, http.StatusUnauthorized, "未登录")
			c.Abort()
			return
		}

		// 获取用户角色
		roles, err := m.authService.roleUseCase.GetByUserID(c.Request.Context(), userID)
		if err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "获取用户角色失败")
			c.Abort()
			return
		}

		// 检查是否有admin角色
		hasAdminRole := false
		for _, role := range roles {
			if role.Code == "admin" {
				hasAdminRole = true
				break
			}
		}

		if !hasAdminRole {
			response.ErrorCode(c, http.StatusForbidden, "权限不足：此操作仅限管理员执行")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireHostPermission 检查主机操作权限的中间件
func (m *AuthMiddleware) RequireHostPermission(operation uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.assetPermissionRepo == nil {
			response.ErrorCode(c, http.StatusInternalServerError, "权限检查未初始化")
			c.Abort()
			return
		}

		// 获取用户ID
		userID := GetUserID(c)
		if userID == 0 {
			response.ErrorCode(c, http.StatusUnauthorized, "未登录")
			c.Abort()
			return
		}

		// 获取主机ID
		hostIDStr := c.Param("id")
		if hostIDStr == "" {
			// 对于创建操作，暂不检查具体权限（创建权限通过分组权限检查）
			c.Next()
			return
		}

		hostID, err := strconv.ParseUint(hostIDStr, 10, 32)
		if err != nil {
			response.ErrorCode(c, http.StatusBadRequest, "无效的主机ID")
			c.Abort()
			return
		}

		// 检查权限
		hasPermission, err := m.assetPermissionRepo.CheckHostOperationPermission(
			c.Request.Context(),
			userID,
			uint(hostID),
			operation,
		)

		if err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "权限检查失败")
			c.Abort()
			return
		}

		if !hasPermission {
			response.ErrorCode(c, http.StatusForbidden, "权限不足")
			c.Abort()
			return
		}

		c.Next()
	}
}
