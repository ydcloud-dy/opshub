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
	authService        *AuthService
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
