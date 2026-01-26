package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"gorm.io/gorm"

	"github.com/ydcloud-dy/opshub/plugins/kubernetes/service"
)

// RoleHandler 角色处理器
type RoleHandler struct {
	clusterService *service.ClusterService
	db             *gorm.DB
}

// NewRoleHandler 创建角色处理器
func NewRoleHandler(db *gorm.DB) *RoleHandler {
	return &RoleHandler{
		clusterService: service.NewClusterService(db),
		db:             db,
	}
}

// ListClusterRoles 获取集群角色列表
// @Summary 获取集群角色列表
// @Description 获取所有 Kubernetes 集群级别的角色
// @Tags Kubernetes/Role
// @Accept json
// @Produce json
// @Param clusterId query int true "集群ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/roles/cluster [get]
func (h *RoleHandler) ListClusterRoles(c *gin.Context) {
	clusterIdStr := c.Query("clusterId")
	if clusterIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少集群ID参数",
		})
		return
	}

	clusterId, err := strconv.ParseUint(clusterIdStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取当前用户 ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	// 获取集群的 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterId), currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败",
		})
		return
	}

	// 获取所有集群角色
	roles, err := clientset.RbacV1().ClusterRoles().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群角色失败: " + err.Error(),
		})
		return
	}

	// 返回平台管理的所有角色（内置角色 + 用户自建角色）
	roleList := make([]map[string]interface{}, 0)
	for _, role := range roles.Items {
		// 显示带有 opshub 管理标签的角色（包括 default-role=true 和 custom-role=true）
		if role.Labels["opshub.ydcloud-dy.com/managed-by"] == "opshub" {
			convertedRole := convertClusterRole(role)
			roleList = append(roleList, convertedRole)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    roleList,
	})
}

// ListNamespaces 获取命名空间列表
// @Summary 获取命名空间列表
// @Description 获取所有命名空间
// @Tags Kubernetes/Role
// @Accept json
// @Produce json
// @Param clusterId query int true "集群ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/roles/namespaces [get]
func (h *RoleHandler) ListNamespaces(c *gin.Context) {
	clusterIdStr := c.Query("clusterId")
	if clusterIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少集群ID参数",
		})
		return
	}

	clusterId, err := strconv.ParseUint(clusterIdStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取当前用户 ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	// 获取集群的 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterId), currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败",
		})
		return
	}

	// 获取所有命名空间
	namespaces, err := clientset.CoreV1().Namespaces().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取命名空间失败: " + err.Error(),
		})
		return
	}

	// 转换为前端格式
	nsList := make([]map[string]interface{}, 0)
	for _, ns := range namespaces.Items {
		// 获取每个命名空间的 pod 数量（可选）
		pods, _ := clientset.CoreV1().Pods(ns.Name).List(c.Request.Context(), metav1.ListOptions{})
		podCount := 0
		if pods != nil {
			podCount = len(pods.Items)
		}

		nsList = append(nsList, map[string]interface{}{
			"name":      ns.Name,
			"podCount":  podCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nsList,
	})
}

// ListNamespaceRoles 获取命名空间角色列表
// @Summary 获取命名空间角色列表
// @Description 获取指定命名空间的所有角色
// @Tags Kubernetes/Role
// @Accept json
// @Produce json
// @Param clusterId query int true "集群ID"
// @Param namespace query string true "命名空间"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/roles/namespace [get]
func (h *RoleHandler) ListNamespaceRoles(c *gin.Context) {
	clusterIdStr := c.Query("clusterId")
	namespace := c.Query("namespace")

	if clusterIdStr == "" || namespace == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少必需参数",
		})
		return
	}

	clusterId, err := strconv.ParseUint(clusterIdStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取当前用户 ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	// 获取集群的 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterId), currentUserID)
	if err != nil {
		// 检查是否是"用户尚未申请凭据"错误
		if strings.Contains(err.Error(), "尚未申请") || strings.Contains(err.Error(), "凭据") {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "您尚未申请该集群的访问凭据，请在集群管理页面申请 kubeconfig 后再访问",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败",
		})
		return
	}

	// 获取指定命名空间的角色
	roles, err := clientset.RbacV1().Roles(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取命名空间角色失败: " + err.Error(),
		})
		return
	}

	// 返回平台管理的所有角色（内置角色 + 用户自建角色）
	roleList := make([]map[string]interface{}, 0)
	for _, role := range roles.Items {
		// 显示带有 opshub 管理标签的角色（包括 default-role=true 和 custom-role=true）
		if role.Labels["opshub.ydcloud-dy.com/managed-by"] == "opshub" {
			convertedRole := convertNamespaceRole(role, namespace)
			roleList = append(roleList, convertedRole)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    roleList,
	})
}

// GetRoleDetail 获取角色详情
// @Summary 获取角色详情
// @Description 获取角色的详细信息，包括权限规则
// @Tags Kubernetes/Role
// @Accept json
// @Produce json
// @Param clusterId query int true "集群ID"
// @Param namespace path string true "命名空间"
// @param name path string true "角色名"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/roles/{namespace}/{name} [get]
func (h *RoleHandler) GetRoleDetail(c *gin.Context) {
	clusterIdStr := c.Query("clusterId")
	namespace := c.Param("namespace")
	name := c.Param("name")

	if clusterIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少集群ID参数",
		})
		return
	}

	clusterId, err := strconv.ParseUint(clusterIdStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取当前用户 ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	// 获取集群的 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterId), currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败",
		})
		return
	}

	// 获取角色详情
	var detail map[string]interface{}
	if namespace == "" || namespace == "cluster" {
		// 集群角色
		clusterRole, err := clientset.RbacV1().ClusterRoles().Get(c.Request.Context(), name, metav1.GetOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取集群角色失败: " + err.Error(),
			})
			return
		}
		detail = convertClusterRoleDetail(*clusterRole)
	} else {
		// 命名空间角色
		nsRole, err := clientset.RbacV1().Roles(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取命名空间角色失败: " + err.Error(),
			})
			return
		}
		detail = convertNamespaceRoleDetail(*nsRole, namespace)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    detail,
	})
}

// CreateDefaultClusterRoles 创建默认的集群角色
// @Summary 创建默认集群角色
// @Description 创建 OpsHub 平台使用的默认集群角色
// @Tags Kubernetes/Role
// @Accept json
// @Produce json
// @Param clusterId query int true "集群ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/roles/create-defaults [post]
func (h *RoleHandler) CreateDefaultClusterRoles(c *gin.Context) {
	startTime := time.Now()
	clusterIdStr := c.Query("clusterId")
	if clusterIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少集群ID参数",
		})
		return
	}

	clusterId, err := strconv.ParseUint(clusterIdStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取当前用户 ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	// 获取集群的 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterId), currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败",
		})
		return
	}

	// 定义默认角色及其权限
	defaultRoles := getDefaultClusterRoles()

	createStartTime := time.Now()

	// 使用 goroutine 并行创建所有角色
	type result struct {
		name string
		err  error
	}
	resultChan := make(chan result, len(defaultRoles))

	for _, roleDef := range defaultRoles {
		go func(role rbacv1.ClusterRole) {
			roleStart := time.Now()
			// 直接尝试创建，如果已存在会返回错误，我们忽略错误
			_, err := clientset.RbacV1().ClusterRoles().Create(c.Request.Context(), &role, metav1.CreateOptions{})
			if err != nil {
				// 如果是"已存在"错误，不算失败
				if !strings.Contains(err.Error(), "already exists") {
					resultChan <- result{name: role.Name, err: err}
					return
				}
			}
			_ = roleStart // 忽略未使用的变量
			resultChan <- result{name: role.Name, err: nil}
		}(roleDef)
	}

	// 收集结果
	createdRoles := []string{}
	var createErr error
	for i := 0; i < len(defaultRoles); i++ {
		res := <-resultChan
		if res.err != nil {
			createErr = res.err
			break
		}
		if res.err == nil {
			createdRoles = append(createdRoles, res.name)
		}
	}

	_ = createStartTime // 忽略未使用的变量
	_ = startTime // 忽略未使用的变量

	if createErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建角色失败: " + createErr.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "操作完成",
		"data": gin.H{
			"created": createdRoles,
		},
	})
}

// getDefaultClusterRoles 获取默认的集群角色定义
func getDefaultClusterRoles() []rbacv1.ClusterRole {
	return []rbacv1.ClusterRole{
		// cluster-owner - 集群所有者，拥有所有权限
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster-owner",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by": "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"*"},
					Resources: []string{"*"},
					Verbs:     []string{"*"},
				},
				{
					NonResourceURLs: []string{"*"},
					Verbs:           []string{"get", "post", "put", "delete", "options", "patch", "head"},
				},
			},
		},
		// cluster-viewer - 集群只读权限
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster-viewer",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by": "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"*"},
					Resources: []string{"*"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
		// manage-appmarket - 管理应用市场
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "manage-appmarket",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by": "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"appmarket.ydcloud-dy.com", "*"},
					Resources: []string{"*"},
					Verbs:     []string{"*"},
				},
			},
		},
		// manage-cluster-rbac - 管理集群 RBAC
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "manage-cluster-rbac",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by": "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"rbac.authorization.k8s.io"},
					Resources: []string{"clusterroles", "clusterrolebindings", "roles", "rolebindings"},
					Verbs:     []string{"*"},
				},
			},
		},
		// manage-cluster-storage - 管理集群存储
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "manage-cluster-storage",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by": "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"storage.k8s.io"},
					Resources: []string{"storageclasses", "csidrivers", "csinodes"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"persistentvolumes"},
					Verbs:     []string{"*"},
				},
			},
		},
		// manage-crd - 管理自定义资源定义
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "manage-crd",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by": "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"apiextensions.k8s.io"},
					Resources: []string{"customresourcedefinitions"},
					Verbs:     []string{"*"},
				},
			},
		},
		// manage-namespaces - 管理命名空间
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "manage-namespaces",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by": "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"namespaces"},
					Verbs:     []string{"*"},
				},
			},
		},
		// manage-nodes - 管理节点
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "manage-nodes",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by": "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs:     []string{"*"},
				},
			},
		},
		// view-cluster-rbac - 查看集群 RBAC
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "view-cluster-rbac",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by": "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"rbac.authorization.k8s.io"},
					Resources: []string{"clusterroles", "clusterrolebindings", "roles", "rolebindings"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
		// view-cluster-storage - 查看集群存储
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "view-cluster-storage",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by": "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"storage.k8s.io"},
					Resources: []string{"storageclasses", "csidrivers", "csinodes"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"persistentvolumes"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
		// view-crd - 查看 CRD
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "view-crd",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by": "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"apiextensions.k8s.io"},
					Resources: []string{"customresourcedefinitions"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
		// view-events - 查看事件
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "view-events",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by": "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"events"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
		// view-namespaces - 查看命名空间
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "view-namespaces",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by": "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"namespaces"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
		// view-nodes - 查看节点
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "view-nodes",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by": "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
	}
}

// CreateDefaultNamespaceRoles 创建默认的命名空间角色（ClusterRole）
// @Summary 创建默认命名空间角色
// @Description 创建用于命名空间级别授权的 ClusterRole（通过 RoleBinding 在特定命名空间生效）
// @Tags Kubernetes/Role
// @Accept json
// @Produce json
// @Param clusterId query int true "集群ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/roles/create-defaults-namespace [post]
func (h *RoleHandler) CreateDefaultNamespaceRoles(c *gin.Context) {
	clusterIdStr := c.Query("clusterId")

	if clusterIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少集群ID参数",
		})
		return
	}

	clusterId, err := strconv.ParseUint(clusterIdStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取当前用户 ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	// 获取集群的 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterId), currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败",
		})
		return
	}

	// 定义默认的命名空间角色（ClusterRole）
	defaultRoles := getDefaultNamespaceRoles()

	// 使用 goroutine 并行创建所有角色
	type result struct {
		name string
		err  error
	}
	resultChan := make(chan result, len(defaultRoles))

	for _, roleDef := range defaultRoles {
		go func(role rbacv1.ClusterRole) {
			// 尝试创建角色
			_, err := clientset.RbacV1().ClusterRoles().Create(c.Request.Context(), &role, metav1.CreateOptions{})
			if err != nil {
				// 如果已存在，则更新角色（确保标签等元数据正确）
				if strings.Contains(err.Error(), "already exists") {
					_, updateErr := clientset.RbacV1().ClusterRoles().Update(c.Request.Context(), &role, metav1.UpdateOptions{})
					if updateErr != nil {
						resultChan <- result{name: role.Name, err: fmt.Errorf("更新角色失败: %w", updateErr)}
						return
					}
					resultChan <- result{name: role.Name, err: nil}
					return
				}
				// 其他错误
				resultChan <- result{name: role.Name, err: err}
				return
			}
			resultChan <- result{name: role.Name, err: nil}
		}(roleDef)
	}

	// 收集结果
	createdRoles := []string{}
	var createErr error
	for i := 0; i < len(defaultRoles); i++ {
		res := <-resultChan
		if res.err != nil {
			createErr = res.err
			break
		}
		if res.err == nil {
			createdRoles = append(createdRoles, res.name)
		}
	}

	if createErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建角色失败: " + createErr.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "操作完成",
		"data": gin.H{
			"created": createdRoles,
		},
	})
}

// getDefaultNamespaceRoles 获取默认的命名空间角色定义（ClusterRole）
// 这些 ClusterRole 用于命名空间级别的授权，通过 RoleBinding 在特定命名空间生效
func getDefaultNamespaceRoles() []rbacv1.ClusterRole {
	return []rbacv1.ClusterRole{
		// namespace-owner - 命名空间所有者，拥有所有权限
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace-owner",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by":      "opshub",
					"opshub.ydcloud-dy.com/default-role":   "true",
					"opshub.ydcloud-dy.com/namespace-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"*"},
					Resources: []string{"*"},
					Verbs:     []string{"*"},
				},
			},
		},
		// namespace-viewer - 命名空间查看者，只读权限
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace-viewer",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by":      "opshub",
					"opshub.ydcloud-dy.com/default-role":   "true",
					"opshub.ydcloud-dy.com/namespace-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"*"},
					Resources: []string{"*"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
		// manage-workload - 管理工作负载
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "manage-workload",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by":      "opshub",
					"opshub.ydcloud-dy.com/default-role":   "true",
					"opshub.ydcloud-dy.com/namespace-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"apps", "extensions"},
					Resources: []string{"deployments", "statefulsets", "daemonsets", "replicasets"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{"batch"},
					Resources: []string{"jobs", "cronjobs"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"pods", "pods/attach", "pods/exec", "pods/portforward", "pods/log"},
					Verbs:     []string{"*"},
				},
			},
		},
		// manage-config - 管理配置
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "manage-config",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by":    "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
					"opshub.ydcloud-dy.com/namespace-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps", "secrets"},
					Verbs:     []string{"*"},
				},
			},
		},
		// manage-rbac - 管理命名空间 RBAC
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "manage-rbac",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by":    "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
					"opshub.ydcloud-dy.com/namespace-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"rbac.authorization.k8s.io"},
					Resources: []string{"roles", "rolebindings"},
					Verbs:     []string{"*"},
				},
			},
		},
		// manage-service-discovery - 管理服务发现
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "manage-service-discovery",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by":    "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
					"opshub.ydcloud-dy.com/namespace-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"services", "endpoints", "endpointslices"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{"networking.k8s.io"},
					Resources: []string{"ingresses", "ingressclasses"},
					Verbs:     []string{"*"},
				},
			},
		},
		// manage-storage - 管理存储
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "manage-storage",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by":    "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
					"opshub.ydcloud-dy.com/namespace-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"persistentvolumeclaims"},
					Verbs:     []string{"*"},
				},
			},
		},
		// view-workload - 查看工作负载
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "view-workload",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by":    "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
					"opshub.ydcloud-dy.com/namespace-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"apps", "extensions"},
					Resources: []string{"deployments", "statefulsets", "daemonsets", "replicasets"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"batch"},
					Resources: []string{"jobs", "cronjobs"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"pods", "pods/log"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
		// view-config - 查看配置
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "view-config",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by":    "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
					"opshub.ydcloud-dy.com/namespace-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps", "secrets"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
		// view-rbac - 查看 RBAC
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "view-rbac",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by":    "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
					"opshub.ydcloud-dy.com/namespace-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"rbac.authorization.k8s.io"},
					Resources: []string{"roles", "rolebindings"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
		// view-service-discovery - 查看服务发现
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "view-service-discovery",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by":    "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
					"opshub.ydcloud-dy.com/namespace-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"services", "endpoints", "endpointslices"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"networking.k8s.io"},
					Resources: []string{"ingresses", "ingressclasses"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
		// view-storage - 查看存储
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "view-storage",
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by":    "opshub",
					"opshub.ydcloud-dy.com/default-role": "true",
					"opshub.ydcloud-dy.com/namespace-role": "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"persistentvolumeclaims"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
	}
}

// DeleteRole 删除角色
// @Summary 删除角色
// @Description 删除指定的角色
// @Tags Kubernetes/Role
// @Accept json
// @Produce json
// @Param clusterId query int true "集群ID"
// @Param namespace path string true "命名空间"
// @param name path string true "角色名"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/roles/{namespace}/{name} [delete]
func (h *RoleHandler) DeleteRole(c *gin.Context) {
	clusterIdStr := c.Query("clusterId")
	namespace := c.Param("namespace")
	name := c.Param("name")

	if clusterIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少集群ID参数",
		})
		return
	}

	clusterId, err := strconv.ParseUint(clusterIdStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取当前用户 ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	// 获取集群的 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterId), currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败",
		})
		return
	}

	// 检查是否为集群角色或命名空间角色
	// 空字符串或 "cluster" 表示集群角色
	if namespace == "" || namespace == "cluster" {
		// 集群角色 - 先获取角色信息检查是否可删除
		role, err := clientset.RbacV1().ClusterRoles().Get(c.Request.Context(), name, metav1.GetOptions{})
		if err == nil {
			// 检查是否为代码中定义的默认角色
			if role.Labels["opshub.ydcloud-dy.com/default-role"] == "true" {
				c.JSON(http.StatusForbidden, gin.H{
					"code":    403,
					"message": "平台默认角色不能删除",
				})
				return
			}
		}
		// 删除集群角色
		err = clientset.RbacV1().ClusterRoles().Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	} else {
		// 命名空间角色 - 先获取角色信息检查是否可删除
		role, err := clientset.RbacV1().Roles(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
		if err == nil {
			// 检查是否为代码中定义的默认角色
			if role.Labels["opshub.ydcloud-dy.com/default-role"] == "true" {
				c.JSON(http.StatusForbidden, gin.H{
					"code":    403,
					"message": "平台默认角色不能删除",
				})
				return
			}
		}
		// 删除命名空间角色
		err = clientset.RbacV1().Roles(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除角色失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// convertClusterRole 转换集群角色为前端格式
func convertClusterRole(role rbacv1.ClusterRole) map[string]interface{} {
	// 计算 age
	age := role.CreationTimestamp.Format("2006-01-02 15:04:05")

	// 转换 labels
	labels := make(map[string]string)
	for key, value := range role.Labels {
		labels[key] = value
	}

	// 判断是否为自定义角色（可删除）
	// 1. 有 default-role=true 标签的（代码中定义的平台默认角色）不可删除
	// 2. 有 custom-role=true 标签的（用户在平台上创建的角色）可以删除
	isCustom := role.Labels["opshub.ydcloud-dy.com/custom-role"] == "true"

	return map[string]interface{}{
		"name":      role.Name,
		"namespace": "",
		"labels":    labels,
		"age":       age,
		"rules":     role.Rules,
		"isCustom":  isCustom,
	}
}

// convertNamespaceRole 转换命名空间角色为前端格式
func convertNamespaceRole(role rbacv1.Role, namespace string) map[string]interface{} {
	// 计算 age
	age := role.CreationTimestamp.Format("2006-01-02 15:04:05")

	// 转换 labels
	labels := make(map[string]string)
	for key, value := range role.Labels {
		labels[key] = value
	}

	// 判断是否为自定义角色（可删除）
	// 1. 有 default-role=true 标签的（代码中定义的平台默认角色）不可删除
	// 2. 有 custom-role=true 标签的（用户在平台上创建的角色）可以删除
	isCustom := role.Labels["opshub.ydcloud-dy.com/custom-role"] == "true"

	return map[string]interface{}{
		"name":      role.Name,
		"namespace": namespace,
		"labels":    labels,
		"age":       age,
		"rules":     role.Rules,
		"isCustom":  isCustom,
	}
}

// convertClusterRoleDetail 转换集群角色详情
func convertClusterRoleDetail(role rbacv1.ClusterRole) map[string]interface{} {
	detail := convertClusterRole(role)

	// 添加权限规则的详细信息
	rules := make([]map[string]interface{}, 0)
	for _, rule := range role.Rules {
		ruleDetail := map[string]interface{}{
			"apiGroups":        rule.APIGroups,
			"resources":        rule.Resources,
			"verbs":            rule.Verbs,
		}

		if len(rule.ResourceNames) > 0 {
			ruleDetail["resourceNames"] = rule.ResourceNames
		}

		if len(rule.NonResourceURLs) > 0 {
			ruleDetail["nonResourceURLs"] = rule.NonResourceURLs
		}

		rules = append(rules, ruleDetail)
	}

	detail["rules"] = rules

	return detail
}

// convertNamespaceRoleDetail 转换命名空间角色详情
func convertNamespaceRoleDetail(role rbacv1.Role, namespace string) map[string]interface{} {
	detail := convertNamespaceRole(role, namespace)

	// 添加权限规则的详细信息
	rules := make([]map[string]interface{}, 0)
	for _, rule := range role.Rules {
		ruleDetail := map[string]interface{}{
			"apiGroups":        rule.APIGroups,
			"resources":        rule.Resources,
			"verbs":            rule.Verbs,
		}

		if len(rule.ResourceNames) > 0 {
			ruleDetail["resourceNames"] = rule.ResourceNames
		}

		if len(rule.NonResourceURLs) > 0 {
			ruleDetail["nonResourceURLs"] = rule.NonResourceURLs
		}

		rules = append(rules, ruleDetail)
	}

	detail["rules"] = rules

	return detail
}

// CreateRoleRequest 创建角色请求
type CreateRoleRequest struct {
	Namespace string                `json:"namespace"`
	Name      string                `json:"name"`
	Rules     []CreateRoleRule      `json:"rules"`
}

// CreateRoleRule 角色规则
type CreateRoleRule struct {
	APIGroups    []string `json:"apiGroups"`
	Resources    []string `json:"resources"`
	ResourceNames []string `json:"resourceNames"`
	Verbs        []string `json:"verbs"`
}

// CreateRole 创建角色
// @Summary 创建角色
// @Description 创建集群角色或命名空间角色
// @Tags Kubernetes/Role
// @Accept json
// @Produce json
// @Param clusterId path int true "集群ID"
// @Param request body CreateRoleRequest true "角色信息"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /api/v1/plugins/kubernetes/clusters/{clusterId}/roles [post]
func (h *RoleHandler) CreateRole(c *gin.Context) {
	clusterIdStr := c.Param("id")
	if clusterIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少集群ID参数",
		})
		return
	}

	clusterId, err := strconv.ParseUint(clusterIdStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 解析请求
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 验证角色名称
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "角色名称不能为空",
		})
		return
	}

	if len(req.Rules) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "至少需要一条规则",
		})
		return
	}

	// 获取当前用户 ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	// 获取集群的 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterId), currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败",
		})
		return
	}

	// 构建角色规则
	rules := make([]rbacv1.PolicyRule, len(req.Rules))
	for i, rule := range req.Rules {
		rules[i] = rbacv1.PolicyRule{
			APIGroups:    rule.APIGroups,
			Resources:    rule.Resources,
			ResourceNames: rule.ResourceNames,
			Verbs:        rule.Verbs,
		}
	}

	// 判断是集群角色还是命名空间角色
	if req.Namespace == "" {
		// 创建集群角色
		role := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: req.Name,
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by":  "opshub",
					"opshub.ydcloud-dy.com/custom-role": "true",
				},
			},
			Rules: rules,
		}

		_, err = clientset.RbacV1().ClusterRoles().Create(c.Request.Context(), role, metav1.CreateOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "创建集群角色失败: " + err.Error(),
			})
			return
		}
	} else {
		// 创建命名空间角色
		role := &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.Name,
				Namespace: req.Namespace,
				Labels: map[string]string{
					"opshub.ydcloud-dy.com/managed-by":  "opshub",
					"opshub.ydcloud-dy.com/custom-role": "true",
				},
			},
			Rules: rules,
		}

		_, err = clientset.RbacV1().Roles(req.Namespace).Create(c.Request.Context(), role, metav1.CreateOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "创建命名空间角色失败: " + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "创建成功",
	})
}
