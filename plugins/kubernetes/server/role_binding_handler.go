package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ydcloud-dy/opshub/plugins/kubernetes/service"
)

// RoleBindingHandler 角色绑定处理器
type RoleBindingHandler struct {
	roleBindingService *service.RoleBindingService
	db                 *gorm.DB
}

// NewRoleBindingHandler 创建角色绑定处理器
func NewRoleBindingHandler(db *gorm.DB) *RoleBindingHandler {
	return &RoleBindingHandler{
		roleBindingService: service.NewRoleBindingService(db),
		db:                 db,
	}
}

// BindUserToRoleRequest 绑定用户到角色请求
type BindUserToRoleRequest struct {
	ClusterID     uint64 `json:"clusterId" binding:"required"`
	UserID        uint64 `json:"userId" binding:"required"`
	RoleName      string `json:"roleName" binding:"required"`
	RoleNamespace string `json:"roleNamespace"`
	RoleType      string `json:"roleType" binding:"required"` // ClusterRole 或 Role
}

// BindUserToRole 绑定用户到K8s角色
// @Summary 绑定用户到K8s角色
// @Description 将平台用户绑定到指定的K8s角色
// @Tags Kubernetes/RoleBinding
// @Accept json
// @Produce json
// @Param body body BindUserToRoleRequest true "绑定信息"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/role-bindings/bind [post]
func (h *RoleBindingHandler) BindUserToRole(c *gin.Context) {
	// 检查是否为管理员
	if !RequireAdmin(c, h.db) {
		return
	}

	var req BindUserToRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 从上下文获取当前用户ID
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	currentUserID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "用户信息错误",
		})
		return
	}

	// 绑定角色
	err := h.roleBindingService.BindUserRole(
		c.Request.Context(),
		req.ClusterID,
		req.UserID,
		req.RoleName,
		req.RoleNamespace,
		req.RoleType,
		uint64(currentUserID),
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "绑定失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "绑定成功",
	})
}

// UnbindUserFromRoleRequest 解绑用户角色请求
type UnbindUserFromRoleRequest struct {
	ClusterID     uint64 `json:"clusterId" binding:"required"`
	UserID        uint64 `json:"userId" binding:"required"`
	RoleName      string `json:"roleName" binding:"required"`
	RoleNamespace string `json:"roleNamespace"`
}

// UnbindUserFromRole 解绑用户K8s角色
// @Summary 解绑用户K8s角色
// @Description 解除用户与K8s角色的绑定
// @Tags Kubernetes/RoleBinding
// @Accept json
// @Produce json
// @Param body body UnbindUserFromRoleRequest true "解绑信息"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/role-bindings/unbind [delete]
func (h *RoleBindingHandler) UnbindUserFromRole(c *gin.Context) {
	// 检查是否为管理员
	if !RequireAdmin(c, h.db) {
		return
	}

	var req UnbindUserFromRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 解绑角色
	err := h.roleBindingService.UnbindUserRole(
		c.Request.Context(),
		req.ClusterID,
		req.UserID,
		req.RoleName,
		req.RoleNamespace,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "解绑失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "解绑成功",
	})
}

// GetRoleBoundUsers 获取角色已绑定的用户列表
// @Summary 获取角色已绑定的用户列表
// @Description 获取指定角色的所有绑定用户
// @Tags Kubernetes/RoleBinding
// @Accept json
// @Produce json
// @Param clusterId query int true "集群ID"
// @Param roleName query string true "角色名称"
// @Param roleNamespace query string false "角色命名空间"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/role-bindings/users [get]
func (h *RoleBindingHandler) GetRoleBoundUsers(c *gin.Context) {
	clusterIdStr := c.Query("clusterId")
	roleName := c.Query("roleName")
	roleNamespace := c.Query("roleNamespace")

	if clusterIdStr == "" || roleName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少必需参数",
		})
		return
	}

	clusterID, err := strconv.ParseUint(clusterIdStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	users, err := h.roleBindingService.GetRoleBoundUsers(
		c.Request.Context(),
		clusterID,
		roleName,
		roleNamespace,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取用户列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    users,
	})
}

// GetAvailableUsersRequest 获取可用用户列表请求
type GetAvailableUsersRequest struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page" binding:"required,min=1"`
	PageSize int    `form:"pageSize" binding:"required,min=1,max=100"`
}

// GetAvailableUsers 获取可绑定的用户列表
// @Summary 获取可绑定的用户列表
// @Description 获取系统中可以绑定的用户列表
// @Tags Kubernetes/RoleBinding
// @Accept json
// @Produce json
// @Param keyword query string false "搜索关键词"
// @Param page query int true "页码"
// @Param pageSize query int true "每页数量"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/role-bindings/available-users [get]
func (h *RoleBindingHandler) GetAvailableUsers(c *gin.Context) {
	var req GetAvailableUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req.Page = 1
		req.PageSize = 20
	}

	users, total, err := h.roleBindingService.GetAvailableUsers(
		c.Request.Context(),
		req.Keyword,
		req.Page,
		req.PageSize,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取用户列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":  users,
			"total": total,
			"page":  req.Page,
			"pageSize": req.PageSize,
		},
	})
}

// GetUserClusterRoles 获取用户在指定集群的角色列表
// @Summary 获取用户在指定集群的角色列表
// @Description 获取指定用户在指定集群的所有角色
// @Tags Kubernetes/RoleBinding
// @Accept json
// @Produce json
// @Param clusterId query int true "集群ID"
// @Param userId query int true "用户ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/role-bindings/user-roles [get]
func (h *RoleBindingHandler) GetUserClusterRoles(c *gin.Context) {
	clusterIdStr := c.Query("clusterId")
	userIdStr := c.Query("userId")

	if clusterIdStr == "" || userIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少必需参数",
		})
		return
	}

	clusterID, err := strconv.ParseUint(clusterIdStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	userID, err := strconv.ParseUint(userIdStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的用户ID",
		})
		return
	}

	bindings, err := h.roleBindingService.GetUserClusterRoles(
		c.Request.Context(),
		clusterID,
		userID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取用户角色失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    bindings,
	})
}

// GetClusterCredentialUsers 获取集群的凭据用户列表
// @Summary 获取集群的凭据用户列表
// @Description 获取当前用户在指定集群的凭据用户（ServiceAccount）
// @Tags Kubernetes/RoleBinding
// @Accept json
// @Produce json
// @Param clusterId query int true "集群ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/role-bindings/credential-users [get]
func (h *RoleBindingHandler) GetClusterCredentialUsers(c *gin.Context) {
	clusterIdStr := c.Query("clusterId")

	if clusterIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少必需参数",
		})
		return
	}

	clusterID, err := strconv.ParseUint(clusterIdStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 从上下文获取当前用户ID
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	currentUserID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "用户信息错误",
		})
		return
	}

	users, err := h.roleBindingService.GetClusterCredentialUsers(
		c.Request.Context(),
		clusterID,
		uint64(currentUserID),
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取凭据用户失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    users,
	})
}

// GetUserRoleBindings 获取用户的所有K8s角色绑定
// @Summary 获取用户的所有K8s角色绑定
// @Description 获取指定集群中所有用户（或指定用户）的K8s角色绑定列表
// @Tags Kubernetes/RoleBinding
// @Accept json
// @Produce json
// @Param clusterId query int true "集群ID"
// @Param userId query int false "用户ID（可选，不传则返回所有用户）"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/role-bindings/user-bindings [get]
func (h *RoleBindingHandler) GetUserRoleBindings(c *gin.Context) {
	clusterIdStr := c.Query("clusterId")
	userIdStr := c.Query("userId")

	if clusterIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少必需参数 clusterId",
		})
		return
	}

	clusterID, err := strconv.ParseUint(clusterIdStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	var userID *uint64
	if userIdStr != "" {
		parsedUserID, err := strconv.ParseUint(userIdStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "无效的用户ID",
			})
			return
		}
		userID = &parsedUserID
	}

	bindings, err := h.roleBindingService.GetUserRoleBindings(
		c.Request.Context(),
		clusterID,
		userID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取角色绑定失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    bindings,
	})
}
