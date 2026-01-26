package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ydcloud-dy/opshub/plugins/kubernetes/data/models"
	"github.com/ydcloud-dy/opshub/plugins/kubernetes/service"
)

// ClusterHandler 集群 HTTP 处理器
type ClusterHandler struct {
	clusterService *service.ClusterService
	db             *gorm.DB
}

// NewClusterHandler 创建集群处理器
func NewClusterHandler(db *gorm.DB) *ClusterHandler {
	return &ClusterHandler{
		clusterService: service.NewClusterService(db),
		db:             db,
	}
}

// CreateCluster 创建集群
// @Summary 创建集群
// @Description 创建新的 Kubernetes 集群
// @Tags Kubernetes/集群管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body service.CreateClusterRequest true "集群信息"
// @Success 200 {object} map[string]interface{} "创建成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/clusters [post]
func (h *ClusterHandler) CreateCluster(c *gin.Context) {
	// 检查是否为管理员
	if !RequireAdmin(c, h.db) {
		return
	}

	var req service.CreateClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 从上下文获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}
	req.UserID = userID.(uint)

	cluster, err := h.clusterService.CreateCluster(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    cluster,
	})
}

// UpdateCluster 更新集群
// @Summary 更新集群
// @Description 更新 Kubernetes 集群信息
// @Tags Kubernetes/集群管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "集群ID"
// @Param body body service.UpdateClusterRequest true "集群信息"
// @Success 200 {object} map[string]interface{} "更新成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/clusters/{id} [put]
func (h *ClusterHandler) UpdateCluster(c *gin.Context) {
	// 检查是否为管理员
	if !RequireAdmin(c, h.db) {
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	var req service.UpdateClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	cluster, err := h.clusterService.UpdateCluster(c.Request.Context(), uint(id), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    cluster,
	})
}

// DeleteCluster 删除集群
// @Summary 删除集群
// @Description 删除 Kubernetes 集群
// @Tags Kubernetes/集群管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "集群ID"
// @Success 200 {object} map[string]interface{} "删除成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/clusters/{id} [delete]
func (h *ClusterHandler) DeleteCluster(c *gin.Context) {
	// 检查是否为管理员
	if !RequireAdmin(c, h.db) {
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	if err := h.clusterService.DeleteCluster(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// GetCluster 获取集群详情
// @Summary 获取集群详情
// @Description 获取 Kubernetes 集群详细信息
// @Tags Kubernetes/Cluster
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/clusters/{id} [get]
func (h *ClusterHandler) GetCluster(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	cluster, err := h.clusterService.GetCluster(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    cluster,
	})
}

// ListClusters 获取集群列表
// @Summary 获取集群列表
// @Description 获取所有 Kubernetes 集群
// @Tags Kubernetes/Cluster
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/clusters [get]
func (h *ClusterHandler) ListClusters(c *gin.Context) {
	clusters, err := h.clusterService.ListClusters(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    clusters,
	})
}

// TestClusterConnection 测试集群连接
// @Summary 测试集群连接
// @Description 测试 Kubernetes 集群连接状态
// @Tags Kubernetes/Cluster
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/clusters/{id}/test [post]
func (h *ClusterHandler) TestClusterConnection(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	version, err := h.clusterService.TestClusterConnection(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": err.Error(),
			"data": gin.H{
				"status":  "failed",
				"version": "",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "连接成功",
		"data": gin.H{
			"status":  "success",
			"version": version,
		},
	})
}

// GetClusterConfig 获取集群凭证（解密后的 KubeConfig）
// @Summary 获取集群凭证
// @Description 获取集群的 KubeConfig 配置（已解密）
// @Tags Kubernetes/Cluster
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/clusters/{id}/config [get]
func (h *ClusterHandler) GetClusterConfig(c *gin.Context) {
	// 检查是否为管理员
	if !RequireAdmin(c, h.db) {
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	kubeConfig, err := h.clusterService.GetClusterKubeConfig(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    kubeConfig,
	})
}

// GenerateKubeConfig 生成用户的 KubeConfig 凭据
// @Summary 生成 KubeConfig
// @Description 为当前用户生成集群的 KubeConfig 凭据
// @Tags Kubernetes/Cluster
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/clusters/kubeconfig [post]
func (h *ClusterHandler) GenerateKubeConfig(c *gin.Context) {
	var req service.GenerateKubeConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 从上下文获取用户名和用户ID（从 JWT token 中提取）
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 生成 KubeConfig
	kubeConfig, uniqueUsername, err := h.clusterService.GenerateUserKubeConfig(
		c.Request.Context(),
		req.ClusterID,
		username.(string),
		userID.(uint),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "生成 KubeConfig 失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"kubeconfig": kubeConfig,
			"username":   uniqueUsername,
		},
	})
}

// RevokeKubeConfig 吊销用户的 KubeConfig 凭据
// @Summary 吊销 KubeConfig
// @Description 吊销用户的集群 KubeConfig 凭据
// @Tags Kubernetes/Cluster
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/clusters/kubeconfig [delete]
func (h *ClusterHandler) RevokeKubeConfig(c *gin.Context) {
	var req service.GenerateKubeConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 验证权限：从上下文获取用户名（从 JWT token 中提取）
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 确保请求的用户名与当前登录用户匹配
	// req.Username 应该是完整的 ServiceAccount 名称（如 opshub-dujie-45h2d）
	// 我们需要验证这个 ServiceAccount 是否属于当前用户
	expectedPrefix := "opshub-" + username.(string)
	if !strings.HasPrefix(req.Username, expectedPrefix) {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "无权吊销其他用户的凭据",
		})
		return
	}

	// 吊销 KubeConfig - 使用完整的 ServiceAccount 名称
	err := h.clusterService.RevokeUserKubeConfig(
		c.Request.Context(),
		req.ClusterID,
		req.Username,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "吊销 KubeConfig 失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "吊销成功",
	})
}

// RevokeCredentialFully 完全吊销用户凭据（删除 SA、RoleBinding 和数据库记录）
// @Summary 完全吊销凭据
// @Description 删除 ServiceAccount、所有相关 RoleBinding 和数据库记录
// @Tags Kubernetes/Cluster
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/clusters/kubeconfig/revoke [delete]
func (h *ClusterHandler) RevokeCredentialFully(c *gin.Context) {
	var req struct {
		ClusterID      uint   `json:"clusterId" binding:"required"`
		ServiceAccount string `json:"serviceAccount" binding:"required"`
		Username       string `json:"username" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	err := h.clusterService.RevokeCredentialFully(
		c.Request.Context(),
		req.ClusterID,
		req.ServiceAccount,
		req.Username,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "吊销凭据失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "吊销成功",
	})
}

// GetServiceAccountKubeConfig 根据ServiceAccount名称获取KubeConfig
// @Summary 根据ServiceAccount获取KubeConfig
// @Description 为指定的ServiceAccount生成KubeConfig凭据
// @Tags Kubernetes/Cluster
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /plugins/kubernetes/clusters/kubeconfig/sa [post]
func (h *ClusterHandler) GetServiceAccountKubeConfig(c *gin.Context) {
	var req struct {
		ClusterID      uint   `json:"clusterId" binding:"required"`
		ServiceAccount string `json:"serviceAccount" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 获取clientset
	clientset, err := h.clusterService.GetCachedClientset(c.Request.Context(), req.ClusterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取K8s客户端失败: " + err.Error(),
		})
		return
	}

	// 获取API Endpoint
	apiEndpoint, err := h.clusterService.GetClusterAPIEndpoint(c.Request.Context(), req.ClusterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群地址失败: " + err.Error(),
		})
		return
	}

	// 为ServiceAccount生成kubeconfig
	kubeConfig, err := h.clusterService.GenerateKubeConfigForSA(
		clientset,
		&models.Cluster{
			ID:          req.ClusterID,
			Name:        fmt.Sprintf("cluster-%d", req.ClusterID),
			APIEndpoint: apiEndpoint,
		},
		req.ServiceAccount,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "生成KubeConfig失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"kubeconfig": kubeConfig,
		},
	})
}

// GetExistingKubeConfig 获取用户现有的KubeConfig
// @Summary 获取用户现有的KubeConfig
// @Description 获取当前用户在指定集群的最新KubeConfig
// @Tags Kubernetes/Cluster
// @Accept json
// @Produce json
// @Param clusterId query int true "集群ID"
// @Success 200 {object} map[string]interface{}
// @Router /plugins/kubernetes/clusters/kubeconfig/existing [get]
func (h *ClusterHandler) GetExistingKubeConfig(c *gin.Context) {
	clusterIdStr := c.Query("clusterId")
	if clusterIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少集群ID参数",
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

	// 从上下文获取用户名和用户ID（从 JWT token 中提取）
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 获取现有的KubeConfig
	kubeConfig, saName, err := h.clusterService.GetUserExistingKubeConfig(
		c.Request.Context(),
		uint(clusterID),
		username.(string),
		userID.(uint),
	)

	if err != nil {
		// 如果是"用户尚未申请凭据"错误，返回404
		if err.Error() == "用户尚未申请凭据" {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "用户尚未申请凭据",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取KubeConfig失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"kubeconfig": kubeConfig,
			"username":   saName,
		},
	})
}

// SyncClusterStatus 同步集群状态
// @Summary 同步集群状态
// @Description 同步指定集群的状态信息（节点数、Pod数等）
// @Tags Kubernetes/Cluster
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/clusters/{id}/sync [post]
func (h *ClusterHandler) SyncClusterStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 异步同步状态
	go func() {
		ctx := context.Background()
		_ = h.clusterService.SyncClusterStatus(ctx, uint(id))
	}()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "同步任务已启动",
	})
}

// SyncAllClustersStatus 同步所有集群状态
// @Summary 同步所有集群状态
// @Description 同步所有集群的状态信息（用于后台定时任务）
// @Tags Kubernetes/Cluster
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/clusters/sync-all [post]
func (h *ClusterHandler) SyncAllClustersStatus(c *gin.Context) {
	// 异步同步所有集群状态
	go func() {
		ctx := context.Background()
		_ = h.clusterService.SyncAllClustersStatus(ctx)
	}()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "批量同步任务已启动",
	})
}
