package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"sigs.k8s.io/yaml"

	"github.com/ydcloud-dy/opshub/plugins/kubernetes/data/models"
	"github.com/ydcloud-dy/opshub/plugins/kubernetes/model"
	"github.com/ydcloud-dy/opshub/plugins/kubernetes/service"
)

// ResourceHandler Kubernetes资源处理器
type ResourceHandler struct {
	clusterService *service.ClusterService
	db             *gorm.DB
}

// NewResourceHandler 创建资源处理器
func NewResourceHandler(clusterService *service.ClusterService, db *gorm.DB) *ResourceHandler {
	return &ResourceHandler{
		clusterService: clusterService,
		db:             db,
	}
}

// JwtClaims JWT声明结构
type JwtClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// verifyTokenAndGetUserID 验证JWT token并返回用户ID
func (h *ResourceHandler) verifyTokenAndGetUserID(tokenString string) (uint, error) {
	// 从环境变量获取JWT密钥
	secretKey := os.Getenv("OPSHUB_SERVER_JWT_SECRET")
	if secretKey == "" {
		secretKey = "your-secret-key-change-in-production" // 默认值
	}

	token, err := jwt.ParseWithClaims(tokenString, &JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err != nil {
		return 0, fmt.Errorf("token解析失败: %w", err)
	}

	if claims, ok := token.Claims.(*JwtClaims); ok && token.Valid {
		return claims.UserID, nil
	}

	return 0, errors.New("invalid token")
}

// handleGetClientsetError 处理 GetClientsetForUser 的错误
// 返回 true 表示错误已处理（发送了响应），调用者应该 return
// 返回 false 表示不是凭据错误，需要继续处理
func (h *ResourceHandler) handleGetClientsetError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}

	// 检查是否是"用户尚未申请凭据"错误
	if strings.Contains(err.Error(), "尚未申请") || strings.Contains(err.Error(), "凭据") {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "您尚未申请该集群的访问凭据，请在集群管理页面申请 kubeconfig 后再访问",
		})
		return true
	}
	return false
}

// NodeInfo 节点信息
type NodeInfo struct {
	Name             string            `json:"name"`
	Status           string            `json:"status"`
	Roles            string            `json:"roles"`
	Age              string            `json:"age"`
	Version          string            `json:"version"`
	InternalIP       string            `json:"internalIP"`
	ExternalIP       string            `json:"externalIP,omitempty"`
	OSImage          string            `json:"osImage"`
	KernelVersion    string            `json:"kernelVersion"`
	ContainerRuntime string            `json:"containerRuntime"`
	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`
	// 新增字段
	CPUCapacity    string          `json:"cpuCapacity"`    // CPU容量
	MemoryCapacity string          `json:"memoryCapacity"` // 内存容量
	CPUUsed        int64           `json:"cpuUsed"`        // CPU使用量（毫核）
	MemoryUsed     int64           `json:"memoryUsed"`     // 内存使用量（字节）
	PodCount       int             `json:"podCount"`       // Pod数量
	PodCapacity    int             `json:"podCapacity"`    // Pod容量
	Schedulable    bool            `json:"schedulable"`    // 是否可调度
	TaintCount     int             `json:"taintCount"`     // 污点数量
	Taints         []TaintInfo     `json:"taints"`         // 污点详情
	PodCIDR        string          `json:"podCIDR"`        // Pod CIDR
	ProviderID     string          `json:"providerID"`     // Provider ID
	Conditions     []NodeCondition `json:"conditions"`     // 节点条件
}

// TaintInfo 污点信息
type TaintInfo struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Effect string `json:"effect"`
}

// NodeCondition 节点条件
type NodeCondition struct {
	Type               string `json:"type"`
	Status             string `json:"status"`
	LastHeartbeatTime  string `json:"lastHeartbeatTime"`
	LastTransitionTime string `json:"lastTransitionTime"`
	Reason             string `json:"reason"`
	Message            string `json:"message"`
}

// PodInfo Pod信息
type PodInfo struct {
	Name       string          `json:"name"`
	Namespace  string          `json:"namespace"`
	Ready      string          `json:"ready"`
	Status     string          `json:"status"`
	Phase      string          `json:"phase"`
	Restarts   int32           `json:"restarts"`
	Age        string          `json:"age"`
	IP         string          `json:"ip"`
	Node       string          `json:"node"`
	Labels     map[string]string `json:"labels"`
	Containers []ContainerInfo `json:"containers"`
}

// ContainerInfo 容器信息
type ContainerInfo struct {
	Name string `json:"name"`
}

// NamespaceInfo 命名空间信息
type NamespaceInfo struct {
	Name   string            `json:"name"`
	Status string            `json:"status"`
	Age    string            `json:"age"`
	Labels map[string]string `json:"labels"`
}

// DeploymentInfo Deployment信息
type DeploymentInfo struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Ready     string            `json:"ready"`
	UpToDate  int32             `json:"upToDate"`
	Available int32             `json:"available"`
	Age       string            `json:"age"`
	Replicas  int32             `json:"replicas"`
	Selector  map[string]string `json:"selector"`
	Labels    map[string]string `json:"labels"`
}

// DaemonSetInfo DaemonSet信息
type DaemonSetInfo struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Ready     string            `json:"ready"`
	Age       string            `json:"age"`
	Labels    map[string]string `json:"labels"`
}

// StatefulSetInfo StatefulSet信息
type StatefulSetInfo struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Ready     string            `json:"ready"`
	Age       string            `json:"age"`
	Labels    map[string]string `json:"labels"`
}

// JobInfo Job信息
type JobInfo struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Ready     string            `json:"ready"`
	Age       string            `json:"age"`
	Labels    map[string]string `json:"labels"`
}

// ClusterStats 集群统计信息
type ClusterStats struct {
	NodeCount         int     `json:"nodeCount"`
	WorkloadCount     int     `json:"workloadCount"` // Deployment + DaemonSet + StatefulSet + Job
	PodCount          int     `json:"podCount"`
	CPUUsage          float64 `json:"cpuUsage"`          // CPU使用率百分比
	MemoryUsage       float64 `json:"memoryUsage"`       // 内存使用率百分比
	CPUCapacity       float64 `json:"cpuCapacity"`       // CPU总核数
	MemoryCapacity    float64 `json:"memoryCapacity"`    // 内存总容量(字节)
	CPUAllocatable    float64 `json:"cpuAllocatable"`    // CPU可分配量
	MemoryAllocatable float64 `json:"memoryAllocatable"` // 内存可分配量(字节)
	CPUUsed           float64 `json:"cpuUsed"`           // CPU已使用量
	MemoryUsed        float64 `json:"memoryUsed"`        // 内存已使用量(字节)
}

// ClusterNetworkInfo 集群网络信息
type ClusterNetworkInfo struct {
	ServiceCIDR      string `json:"serviceCIDR"`      // Service CIDR
	PodCIDR          string `json:"podCIDR"`          // Pod CIDR
	APIServerAddress string `json:"apiServerAddress"` // API Server 地址
	NetworkPlugin    string `json:"networkPlugin"`    // 网络插件
	ProxyMode        string `json:"proxyMode"`        // 服务转发模式
	DNSService       string `json:"dnsService"`       // DNS 服务
}

// ClusterComponentInfo 集群组件信息
type ClusterComponentInfo struct {
	Components []ComponentInfo `json:"components"` // 控制平面组件
	Runtime    RuntimeInfo     `json:"runtime"`    // 运行时信息
	Storage    []StorageInfo   `json:"storage"`    // 存储信息
}

// ComponentInfo 组件信息
type ComponentInfo struct {
	Name    string `json:"name"`    // 组件名称
	Version string `json:"version"` // 版本
	Status  string `json:"status"`  // 状态
}

// RuntimeInfo 运行时信息
type RuntimeInfo struct {
	ContainerRuntime string `json:"containerRuntime"` // 容器运行时
	Version          string `json:"version"`          // 版本
}

// StorageInfo 存储信息
type StorageInfo struct {
	Name          string `json:"name"`          // 存储名称
	Provisioner   string `json:"provisioner"`   // Provisioner
	ReclaimPolicy string `json:"reclaimPolicy"` // 回收策略
}

// NetworkPolicyInfo 网络策略信息
type NetworkPolicyInfo struct {
	Name      string            `json:"name"`      // 名称
	Namespace string            `json:"namespace"` // 命名空间
	NetworkID string            `json:"networkID"` // 网络ID (UID)
	Type      string            `json:"type"`      // 类型
	Status    string            `json:"status"`    // 状态
	Age       string            `json:"age"`       // 创建时间
	Labels    map[string]string `json:"labels"`    // 标签
}

// EventInfo 事件信息
type EventInfo struct {
	Type           string             `json:"type"`           // 事件类型: Normal, Warning
	Reason         string             `json:"reason"`         // 原因
	Message        string             `json:"message"`        // 消息
	Source         string             `json:"source"`         // 来源
	Count          int32              `json:"count"`          // 次数
	FirstTimestamp string             `json:"firstTimestamp"` // 首次发生时间
	LastTimestamp  string             `json:"lastTimestamp"`  // 最后发生时间
	InvolvedObject InvolvedObjectInfo `json:"involvedObject"` // 关联对象
}

// InvolvedObjectInfo 关联对象信息
type InvolvedObjectInfo struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// ListNodes 获取节点列表
// @Summary 获取节点列表
// @Description 获取 Kubernetes 集群的节点列表及其状态信息
// @Tags Kubernetes/节点管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param clusterId query int true "集群ID"
// @Success 200 {object} map[string]interface{} "节点列表"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/resources/nodes [get]
func (h *ResourceHandler) ListNodes(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取当前登录用户 ID
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权：无法获取用户信息",
		})
		return
	}

	currentUserID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "用户ID类型错误",
		})
		return
	}

	// 使用用户的凭据获取 clientset（实现权限隔离）
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	nodes, err := clientset.CoreV1().Nodes().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "节点")
		return
	}

	// 获取metrics clientset
	metricsClient, err := h.clusterService.GetCachedMetricsClientset(c.Request.Context(), uint(clusterID))
	if err != nil {
		// 继续执行，只是没有metrics数据
		metricsClient = nil
	}

	// 批量获取所有节点的metrics
	nodeMetricsMap := make(map[string]*v1beta1.NodeMetrics)
	if metricsClient != nil {
		allNodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().List(c.Request.Context(), metav1.ListOptions{})
		if err == nil {
			for _, nm := range allNodeMetrics.Items {
				nodeMetricsMap[nm.Name] = &nm
			}
		}
	}

	// 获取所有Pod以计算每个节点的Pod数量
	pods, err := clientset.CoreV1().Pods("").List(c.Request.Context(), metav1.ListOptions{})
	podCountMap := make(map[string]int)
	if err == nil {
		for _, pod := range pods.Items {
			if pod.Spec.NodeName != "" {
				podCountMap[pod.Spec.NodeName]++
			}
		}
	}

	nodeInfos := make([]NodeInfo, 0, len(nodes.Items))
	for _, node := range nodes.Items {
		// 确保 labels 不为 nil
		labels := node.Labels
		if labels == nil {
			labels = make(map[string]string)
		}

		// 确保 annotations 不为 nil
		annotations := node.Annotations
		if annotations == nil {
			annotations = make(map[string]string)
		}

		// 获取 Pod CIDR
		podCIDR := ""
		if len(node.Spec.PodCIDRs) > 0 {
			podCIDR = node.Spec.PodCIDRs[0]
		} else if node.Spec.PodCIDR != "" {
			podCIDR = node.Spec.PodCIDR
		}

		nodeInfo := NodeInfo{
			Name:             node.Name,
			Version:          node.Status.NodeInfo.KubeletVersion,
			OSImage:          node.Status.NodeInfo.OSImage,
			KernelVersion:    node.Status.NodeInfo.KernelVersion,
			ContainerRuntime: node.Status.NodeInfo.ContainerRuntimeVersion,
			Labels:           labels,
			Annotations:      annotations,
			PodCIDR:          podCIDR,
			ProviderID:       node.Spec.ProviderID,
		}

		// 获取节点状态
		for _, condition := range node.Status.Conditions {
			if condition.Type == v1.NodeReady {
				if condition.Status == v1.ConditionTrue {
					nodeInfo.Status = "Ready"
				} else {
					nodeInfo.Status = "NotReady"
				}
				break
			}
		}

		// 获取IP地址（InternalIP 和 ExternalIP）
		for _, addr := range node.Status.Addresses {
			if addr.Type == v1.NodeInternalIP {
				nodeInfo.InternalIP = addr.Address
			} else if addr.Type == v1.NodeExternalIP {
				nodeInfo.ExternalIP = addr.Address
			}
		}

		// 计算节点年龄
		nodeInfo.Age = calculateAge(node.CreationTimestamp.Time)

		// 获取角色（从Label中推断）
		if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok {
			nodeInfo.Roles = "master"
		} else if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
			nodeInfo.Roles = "control-plane"
		} else {
			nodeInfo.Roles = "worker"
		}

		// 获取CPU和内存容量
		cpuCapacity := node.Status.Capacity.Cpu().String()
		memoryCapacity := node.Status.Capacity.Memory().String()
		nodeInfo.CPUCapacity = cpuCapacity
		nodeInfo.MemoryCapacity = memoryCapacity

		// 获取Pod容量（优先使用Allocatable，如果为0则使用Capacity，如果还是0则使用默认值110）
		podCapacity := node.Status.Allocatable.Pods()
		podCapacityValue := int(podCapacity.Value())
		if podCapacityValue == 0 {
			podCapacity = node.Status.Capacity.Pods()
			podCapacityValue = int(podCapacity.Value())
		}
		// 如果还是0，使用默认值110（Kubernetes默认的Pod数量限制）
		if podCapacityValue == 0 {
			podCapacityValue = 110
		}
		nodeInfo.PodCapacity = podCapacityValue

		// 获取Pod数量
		nodeInfo.PodCount = podCountMap[node.Name]

		// 判断是否可调度
		nodeInfo.Schedulable = !node.Spec.Unschedulable

		// 获取污点数量和详情
		nodeInfo.TaintCount = len(node.Spec.Taints)
		nodeInfo.Taints = make([]TaintInfo, 0, len(node.Spec.Taints))
		for _, taint := range node.Spec.Taints {
			nodeInfo.Taints = append(nodeInfo.Taints, TaintInfo{
				Key:    taint.Key,
				Value:  taint.Value,
				Effect: string(taint.Effect),
			})
		}

		// 填充Conditions
		nodeInfo.Conditions = make([]NodeCondition, 0, len(node.Status.Conditions))
		for _, cond := range node.Status.Conditions {
			nodeInfo.Conditions = append(nodeInfo.Conditions, NodeCondition{
				Type:               string(cond.Type),
				Status:             string(cond.Status),
				LastHeartbeatTime:  cond.LastHeartbeatTime.Format("2006-01-02 15:04:05"),
				LastTransitionTime: cond.LastTransitionTime.Format("2006-01-02 15:04:05"),
				Reason:             cond.Reason,
				Message:            cond.Message,
			})
		}

		// 填充CPU和内存使用量
		if nodeMetrics, ok := nodeMetricsMap[node.Name]; ok {
			nodeInfo.CPUUsed = nodeMetrics.Usage.Cpu().MilliValue()
			nodeInfo.MemoryUsed = nodeMetrics.Usage.Memory().Value()
		}

		nodeInfos = append(nodeInfos, nodeInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nodeInfos,
	})
}

// GetNodeMetrics 获取节点指标
// @Summary 获取节点指标
// @Description 获取指定节点的 CPU、内存等资源使用指标
// @Tags Kubernetes/节点管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param clusterId query int true "集群ID"
// @Param nodeName path string true "节点名称"
// @Success 200 {object} map[string]interface{} "节点指标数据"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/resources/nodes/{nodeName}/metrics [get]
func (h *ResourceHandler) GetNodeMetrics(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	nodeName := c.Param("nodeName")

	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
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

	// 获取客户端
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		return
	}

	// 获取 metrics clientset
	metricsClient, err := h.clusterService.GetCachedMetricsClientset(c.Request.Context(), uint(clusterID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取 metrics client 失败: " + err.Error(),
		})
		return
	}

	// 获取节点指标
	nodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().Get(c.Request.Context(), nodeName, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("获取节点指标失败: %v", err),
		})
		return
	}

	// 获取节点信息以获取容量
	node, err := clientset.CoreV1().Nodes().Get(c.Request.Context(), nodeName, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("获取节点信息失败: %v", err),
		})
		return
	}

	// 计算CPU使用率
	cpuUsage := float64(nodeMetrics.Usage.Cpu().MilliValue()) / float64(node.Status.Capacity.Cpu().MilliValue())
	memoryUsage := float64(nodeMetrics.Usage.Memory().Value()) / float64(node.Status.Capacity.Memory().Value())

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"cpuUsage":    cpuUsage,
			"memoryUsage": memoryUsage,
			"cpuUsed":     nodeMetrics.Usage.Cpu().MilliValue(),
			"memoryUsed":  nodeMetrics.Usage.Memory().Value(),
		},
	})
}

// ListNamespaces 获取命名空间列表
// @Summary 获取命名空间列表
// @Description 获取 Kubernetes 集群的命名空间列表
// @Tags Kubernetes/命名空间
// @Accept json
// @Produce json
// @Security Bearer
// @Param clusterId query int true "集群ID"
// @Success 200 {object} map[string]interface{} "命名空间列表"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/resources/namespaces [get]
func (h *ResourceHandler) ListNamespaces(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
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

	// 使用用户凭据获取 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "命名空间")
		return
	}

	namespaceInfos := make([]NamespaceInfo, 0, len(namespaces.Items))
	for _, ns := range namespaces.Items {
		nsInfo := NamespaceInfo{
			Name:   ns.Name,
			Labels: ns.Labels,
			Age:    calculateAge(ns.CreationTimestamp.Time),
		}

		// 获取状态
		if ns.Status.Phase == v1.NamespaceActive {
			nsInfo.Status = "Active"
		} else {
			nsInfo.Status = string(ns.Status.Phase)
		}

		namespaceInfos = append(namespaceInfos, nsInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    namespaceInfos,
	})
}

// ListPods 获取Pod列表
// @Summary 获取Pod列表
// @Description 获取指定命名空间的 Pod 列表
// @Tags Kubernetes/工作负载
// @Accept json
// @Produce json
// @Security Bearer
// @Param clusterId query int true "集群ID"
// @Param namespace query string false "命名空间"
// @Success 200 {object} map[string]interface{} "Pod列表"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/resources/pods [get]
func (h *ResourceHandler) ListPods(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = v1.NamespaceAll
	}

	nodeName := c.Query("nodeName")

	// 获取当前用户 ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	// 使用用户凭据获取 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	listOptions := metav1.ListOptions{}
	if nodeName != "" {
		listOptions.FieldSelector = "spec.nodeName=" + nodeName
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(c.Request.Context(), listOptions)
	if err != nil {
		HandleK8sError(c, err, "Pod")
		return
	}

	podInfos := make([]PodInfo, 0, len(pods.Items))
	for _, pod := range pods.Items {
		podInfo := PodInfo{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Labels:    pod.Labels,
			Age:       calculateAge(pod.CreationTimestamp.Time),
			IP:        pod.Status.PodIP,
			Node:      pod.Spec.NodeName,
		}

		// 计算Ready状态
		readyContainers := 0
		totalContainers := len(pod.Spec.Containers)
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				readyContainers++
			}
			podInfo.Restarts += cs.RestartCount
		}
		podInfo.Ready = strconv.Itoa(readyContainers) + "/" + strconv.Itoa(totalContainers)

		// 获取Pod状态
		podInfo.Status = string(pod.Status.Phase)
		podInfo.Phase = string(pod.Status.Phase)

		// 添加容器信息
		for _, container := range pod.Spec.Containers {
			podInfo.Containers = append(podInfo.Containers, ContainerInfo{
				Name: container.Name,
			})
		}

		podInfos = append(podInfos, podInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    podInfos,
	})
}

// ListDeployments 获取Deployment列表
// @Summary 获取Deployment列表
// @Description 获取指定命名空间的 Deployment 列表
// @Tags Kubernetes/工作负载
// @Accept json
// @Produce json
// @Security Bearer
// @Param clusterId query int true "集群ID"
// @Param namespace query string false "命名空间"
// @Success 200 {object} map[string]interface{} "Deployment列表"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/resources/deployments [get]
func (h *ResourceHandler) ListDeployments(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = v1.NamespaceAll
	}

	// 获取当前用户 ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	// 使用用户凭据获取 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	deployments, err := clientset.AppsV1().Deployments(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "Deployment")
		return
	}

	deploymentInfos := make([]DeploymentInfo, 0, len(deployments.Items))
	for _, deploy := range deployments.Items {
		deployInfo := DeploymentInfo{
			Name:      deploy.Name,
			Namespace: deploy.Namespace,
			UpToDate:  deploy.Status.UpdatedReplicas,
			Available: deploy.Status.AvailableReplicas,
			Age:       calculateAge(deploy.CreationTimestamp.Time),
			Replicas:  *deploy.Spec.Replicas,
			Selector:  deploy.Spec.Selector.MatchLabels,
			Labels:    deploy.Labels,
		}

		// 计算Ready状态
		readyReplicas := deploy.Status.ReadyReplicas
		totalReplicas := *deploy.Spec.Replicas
		deployInfo.Ready = strconv.Itoa(int(readyReplicas)) + "/" + strconv.Itoa(int(totalReplicas))

		deploymentInfos = append(deploymentInfos, deployInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    deploymentInfos,
	})
}

// GetClusterStats 获取集群统计信息
// @Summary 获取集群统计信息
// @Description 获取 Kubernetes 集群的资源统计信息（节点数、Pod数等）
// @Tags Kubernetes/集群
// @Accept json
// @Produce json
// @Security Bearer
// @Param clusterId query int true "集群ID"
// @Success 200 {object} map[string]interface{} "集群统计信息"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/resources/stats [get]
func (h *ResourceHandler) GetClusterStats(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
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

	// 使用用户凭据获取 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	// 获取 metrics clientset
	metricsClient, err := h.clusterService.GetCachedMetricsClientset(c.Request.Context(), uint(clusterID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取 metrics client 失败: " + err.Error(),
		})
		return
	}

	stats := ClusterStats{}

	// 获取节点信息
	nodes, err := clientset.CoreV1().Nodes().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取节点列表失败: " + err.Error(),
		})
		return
	}
	stats.NodeCount = len(nodes.Items)

	// 计算CPU和内存总量及可分配量
	var totalCPUCapacity, totalMemoryCapacity float64
	var totalCPUAllocatable, totalMemoryAllocatable float64

	for _, node := range nodes.Items {
		cpuCapacity := node.Status.Capacity.Cpu().AsApproximateFloat64()
		memoryCapacity := float64(node.Status.Capacity.Memory().Value())
		cpuAllocatable := node.Status.Allocatable.Cpu().AsApproximateFloat64()
		memoryAllocatable := float64(node.Status.Allocatable.Memory().Value())

		totalCPUCapacity += cpuCapacity
		totalMemoryCapacity += memoryCapacity
		totalCPUAllocatable += cpuAllocatable
		totalMemoryAllocatable += memoryAllocatable
	}

	stats.CPUCapacity = totalCPUCapacity
	stats.MemoryCapacity = totalMemoryCapacity
	stats.CPUAllocatable = totalCPUAllocatable
	stats.MemoryAllocatable = totalMemoryAllocatable

	// 获取节点指标（Metrics API）
	nodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取节点指标失败: " + err.Error(),
		})
		return
	}

	// 计算实际使用的CPU和内存
	var totalCPUUsed, totalMemoryUsed float64
	for _, nodeMetric := range nodeMetrics.Items {
		cpuUsed := nodeMetric.Usage.Cpu().AsApproximateFloat64()
		memoryUsed := float64(nodeMetric.Usage.Memory().Value())
		totalCPUUsed += cpuUsed
		totalMemoryUsed += memoryUsed
	}

	// 设置已使用量
	stats.CPUUsed = totalCPUUsed
	stats.MemoryUsed = totalMemoryUsed

	// 计算使用率百分比（基于 Allocatable）
	if totalCPUAllocatable > 0 {
		stats.CPUUsage = (totalCPUUsed / totalCPUAllocatable) * 100
	}
	if totalMemoryAllocatable > 0 {
		stats.MemoryUsage = (totalMemoryUsed / totalMemoryAllocatable) * 100
	}

	// 获取Pod数量
	pods, err := clientset.CoreV1().Pods("").List(c.Request.Context(), metav1.ListOptions{})
	if err == nil {
		stats.PodCount = len(pods.Items)
	}

	// 获取Deployment数量
	deployments, err := clientset.AppsV1().Deployments("").List(c.Request.Context(), metav1.ListOptions{})
	deploymentCount := 0
	if err == nil {
		deploymentCount = len(deployments.Items)
	}

	// 获取DaemonSet数量
	daemonsets, err := clientset.AppsV1().DaemonSets("").List(c.Request.Context(), metav1.ListOptions{})
	daemonsetCount := 0
	if err == nil {
		daemonsetCount = len(daemonsets.Items)
	}

	// 获取StatefulSet数量
	statefulsets, err := clientset.AppsV1().StatefulSets("").List(c.Request.Context(), metav1.ListOptions{})
	statefulsetCount := 0
	if err == nil {
		statefulsetCount = len(statefulsets.Items)
	}

	// 获取Job数量
	jobs, err := clientset.BatchV1().Jobs("").List(c.Request.Context(), metav1.ListOptions{})
	jobCount := 0
	if err == nil {
		jobCount = len(jobs.Items)
	}

	// 工作负载总数 = Deployment + DaemonSet + StatefulSet + Job
	stats.WorkloadCount = deploymentCount + daemonsetCount + statefulsetCount + jobCount

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    stats,
	})
}

// GetClusterNetworkInfo 获取集群网络信息
func (h *ResourceHandler) GetClusterNetworkInfo(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
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

	// 使用用户凭据获取 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	networkInfo := ClusterNetworkInfo{}

	// 获取集群的 API Endpoint
	apiEndpoint, err := h.clusterService.GetClusterAPIEndpoint(c.Request.Context(), uint(clusterID))
	if err == nil && apiEndpoint != "" {
		networkInfo.APIServerAddress = apiEndpoint
	}

	// 获取节点信息来推断网络配置
	nodes, err := clientset.CoreV1().Nodes().List(c.Request.Context(), metav1.ListOptions{})
	if err == nil && len(nodes.Items) > 0 {
		node := nodes.Items[0]

		// 获取 Pod CIDR
		if podCIDR := node.Spec.PodCIDR; podCIDR != "" {
			networkInfo.PodCIDR = podCIDR
		}
	}

	// 获取 CNI 网络插件（从 kube-system 命名空间的 DaemonSet 中检测）
	daemonSets, err := clientset.AppsV1().DaemonSets("kube-system").List(c.Request.Context(), metav1.ListOptions{})
	if err == nil {
		// 常见的 CNI 插件标识
		cniPlugins := map[string]string{
			"calico":          "Calico",
			"flannel":         "Flannel",
			"weave":           "Weave",
			"canal":           "Canal",
			"cilium":          "Cilium",
			"contiv":          "Contiv",
			"kube-router":     "Kube-Router",
			"amazon-vpc-cni":  "AWS VPC CNI",
			"azure-cniplugin": "Azure CNI",
			"vsphere-cni":     "vSphere CNI",
			"tke-cni":         "TKE CNI",
			"tke-bridge":      "TKE Bridge",
			"networkpolicy":   "TKE NetworkPolicy",
		}

		for _, ds := range daemonSets.Items {
			dsName := strings.ToLower(ds.Name)
			for key, name := range cniPlugins {
				if strings.Contains(dsName, key) {
					networkInfo.NetworkPlugin = name
					break
				}
			}
			if networkInfo.NetworkPlugin != "" {
				break
			}
		}
	}

	// 获取 kube-proxy 的 proxy 模式（从 DaemonSet 的命令行参数、环境变量或 ConfigMap 中获取）
	kubeProxyDS, err := clientset.AppsV1().DaemonSets("kube-system").Get(c.Request.Context(), "kube-proxy", metav1.GetOptions{})
	if err == nil && len(kubeProxyDS.Spec.Template.Spec.Containers) > 0 {
		container := kubeProxyDS.Spec.Template.Spec.Containers[0]

		// 1. 从命令行参数中查找（优先级最高）
		for _, arg := range container.Command {
			if strings.Contains(arg, "--proxy-mode=") {
				mode := strings.TrimPrefix(arg, "--proxy-mode=")
				networkInfo.ProxyMode = mode
				break
			}
		}

		// 2. 从命令行参数中查找（空格分隔）
		if networkInfo.ProxyMode == "" && len(container.Command) > 0 {
			for i, arg := range container.Command {
				if arg == "--proxy-mode" && i+1 < len(container.Command) {
					networkInfo.ProxyMode = container.Command[i+1]
					break
				}
			}
		}

		// 3. 从环境变量中查找
		if networkInfo.ProxyMode == "" {
			for _, env := range container.Env {
				if env.Name == "KUBE_PROXY_MODE" {
					networkInfo.ProxyMode = env.Value
					break
				}
			}
		}
	}

	// 如果没找到，从 ConfigMap 中查找
	if networkInfo.ProxyMode == "" {
		kubeProxyCM, err := clientset.CoreV1().ConfigMaps("kube-system").Get(c.Request.Context(), "kube-proxy", metav1.GetOptions{})
		if err == nil {
			// 检查 config.yaml
			if config, ok := kubeProxyCM.Data["config.yaml"]; ok {
				// 查找 proxyMode
				if idx := strings.Index(config, "proxyMode:"); idx >= 0 {
					start := idx + 10 // 跳过 "proxyMode:"
					remaining := config[start:]
					// 提取到行尾或注释
					if end := strings.IndexAny(remaining, "\n#"); end > 0 {
						modeStr := strings.TrimSpace(remaining[:end])
						modeStr = strings.Trim(modeStr, `"`)
						networkInfo.ProxyMode = modeStr
					}
				}
			}
			// 检查 config.conf (Kubernetes 1.10+ 使用这个格式)
			if config, ok := kubeProxyCM.Data["config.conf"]; ok {
				if idx := strings.Index(config, "proxyMode"); idx >= 0 {
					start := idx + 10 // 跳过 "proxyMode" 或 "proxyMode:"
					remaining := config[start:]
					// 跳过可能的冒号和等号
					remaining = strings.TrimLeft(remaining, ":=")
					remaining = strings.TrimSpace(remaining)
					// 提取值到行尾或逗号
					if end := strings.IndexAny(remaining, "\n,"); end > 0 {
						modeStr := strings.TrimSpace(remaining[:end])
						modeStr = strings.Trim(modeStr, `"`)
						networkInfo.ProxyMode = modeStr
					}
				}
			}
			if configJSON, ok := kubeProxyCM.Data["config.json"]; ok {
				// JSON 格式配置
				if idx := strings.Index(configJSON, "proxyMode"); idx > 0 {
					start := idx + 11 // 跳过 "proxyMode:"
					if end := strings.Index(configJSON[start:], ","); end > 0 {
						modeStr := strings.TrimSpace(configJSON[start : start+end])
						modeStr = strings.Trim(modeStr, `"`)
						networkInfo.ProxyMode = modeStr
					}
				}
			}
		}
	}

	// 默认值为 ipvs（现代 Kubernetes 的默认模式）
	if networkInfo.ProxyMode == "" {
		// 尝试从节点信息推断（不是100%准确）
		nodes, err := clientset.CoreV1().Nodes().List(c.Request.Context(), metav1.ListOptions{})
		if err == nil && len(nodes.Items) > 0 {
			// 检查内核模块或系统信息来判断
			// 但这比较复杂，这里简单使用默认值
			networkInfo.ProxyMode = "ipvs"
		}
	}

	// 获取 kube-apiserver 服务
	apiServerSvc, err := clientset.CoreV1().Services("default").Get(c.Request.Context(), "kubernetes", metav1.GetOptions{})
	if err == nil && apiServerSvc != nil {
		// 获取 Service CIDR (从 ClusterIPs 推断)
		if len(apiServerSvc.Spec.ClusterIPs) > 0 {
			// 通常是第一个 IP，但我们可以推断 CIDR
			// 例如：10.0.0.1 可能是 10.0.0.0/24 或 10.0.0.0/16
			ip := apiServerSvc.Spec.ClusterIPs[0]
			// 简化处理，直接显示第一个 ClusterIP
			networkInfo.ServiceCIDR = ip
		}
	}

	// 获取 DNS 服务
	_, err = clientset.CoreV1().Services("kube-system").Get(c.Request.Context(), "kube-dns", metav1.GetOptions{})
	if err == nil {
		networkInfo.DNSService = "CoreDNS"
	} else {
		// 尝试获取其他 DNS 实现
		svcs, _ := clientset.CoreV1().Services("kube-system").List(c.Request.Context(), metav1.ListOptions{})
		for _, svc := range svcs.Items {
			if strings.Contains(svc.Name, "dns") {
				networkInfo.DNSService = svc.Name
				break
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    networkInfo,
	})
}

// GetClusterComponentInfo 获取集群组件信息
func (h *ResourceHandler) GetClusterComponentInfo(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
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

	// 使用用户凭据获取 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	componentInfo := ClusterComponentInfo{
		Components: []ComponentInfo{},
	}

	// 获取节点信息来获取运行时
	nodes, err := clientset.CoreV1().Nodes().List(c.Request.Context(), metav1.ListOptions{})
	if err == nil && len(nodes.Items) > 0 {
		node := nodes.Items[0]
		componentInfo.Runtime = RuntimeInfo{
			ContainerRuntime: node.Status.NodeInfo.ContainerRuntimeVersion,
			Version:          node.Status.NodeInfo.KubeletVersion,
		}
	}

	// 获取控制平面组件 Pod
	pods, err := clientset.CoreV1().Pods("kube-system").List(c.Request.Context(), metav1.ListOptions{})
	if err == nil {
		// 常见的控制平面组件（支持多种命名方式）
		controlPlanePatterns := map[string]string{
			"kube-apiserver":          "API Server",
			"kube-apiserver-":         "API Server",
			"apiserver":               "API Server",
			"kube-controller":         "Controller Manager",
			"kube-controller-":        "Controller Manager",
			"kube-controller-manager": "Controller Manager",
			"cloud-controller":        "Cloud Controller",
			"cloud-controller-":       "Cloud Controller",
			"kube-scheduler":          "Scheduler",
			"kube-scheduler-":         "Scheduler",
			"scheduler":               "Scheduler",
			"etcd":                    "etcd",
			"etcd-":                   "etcd",
			"coredns":                 "CoreDNS",
			"coredns-":                "CoreDNS",
		}

		componentMap := make(map[string]ComponentInfo)

		for _, pod := range pods.Items {
			podName := strings.ToLower(pod.Name)
			var componentName string
			var componentKey string

			// 排除非控制平面组件（CNI、网络插件等）
			if strings.Contains(podName, "calico") ||
				strings.Contains(podName, "flannel") ||
				strings.Contains(podName, "kube-proxy") ||
				strings.Contains(podName, "metrics-server") {
				continue
			}

			// 识别组件（支持前缀匹配和包含匹配）
			for pattern, name := range controlPlanePatterns {
				matched := false
				if strings.HasSuffix(pattern, "-") {
					// 前缀匹配模式
					matched = strings.HasPrefix(podName, pattern)
				} else {
					// 精确匹配或包含匹配
					matched = strings.HasPrefix(podName, pattern) ||
						strings.Contains(podName, pattern)
				}

				if matched {
					// 再次检查，确保不是 CNI 组件
					if strings.Contains(podName, "calico") || strings.Contains(podName, "controllers") {
						if !strings.HasPrefix(podName, "kube-controller") {
							continue
						}
					}

					// 使用更具体的key避免重复
					if pattern == "kube-apiserver" || pattern == "kube-apiserver-" {
						componentKey = "kube-apiserver"
					} else if pattern == "kube-controller" || pattern == "kube-controller-" || pattern == "kube-controller-manager" {
						componentKey = "kube-controller"
					} else if pattern == "kube-scheduler" || pattern == "kube-scheduler-" {
						componentKey = "kube-scheduler"
					} else if strings.HasPrefix(pattern, "etcd") {
						componentKey = "etcd"
					} else if strings.HasPrefix(pattern, "coredns") {
						componentKey = "coredns"
					} else if strings.HasPrefix(pattern, "cloud-controller") {
						componentKey = "cloud-controller"
					}
					componentName = name
					break
				}
			}

			if componentName == "" {
				continue
			}

			// 获取版本
			version := "unknown"
			if len(pod.Spec.Containers) > 0 {
				// 尝试从 Image 中提取版本
				image := pod.Spec.Containers[0].Image
				if idx := strings.LastIndex(image, ":"); idx > 0 {
					version = image[idx+1:]
				} else {
					version = image
				}
			}

			// 获取状态
			status := "Running"
			if pod.Status.Phase != v1.PodRunning {
				status = string(pod.Status.Phase)
			}

			componentMap[componentKey] = ComponentInfo{
				Name:    componentName,
				Version: version,
				Status:  status,
			}
		}

		// 转换为切片
		for _, comp := range componentMap {
			componentInfo.Components = append(componentInfo.Components, comp)
		}
	}

	// 如果没有检测到控制平面组件，可能是二进制部署的集群（systemd 启动）
	// 尝试通过节点标签和版本信息来推断

	// 检查是否已经有控制平面组件（API Server, Scheduler, Controller Manager, etcd）
	hasControlPlanePods := false
	for _, comp := range componentInfo.Components {
		if comp.Name == "API Server" || comp.Name == "Scheduler" ||
			comp.Name == "Controller Manager" || comp.Name == "etcd" {
			hasControlPlanePods = true
			break
		}
	}

	if !hasControlPlanePods {
		// 获取集群版本信息
		serverVersion, err := clientset.Discovery().ServerVersion()
		if err == nil {
			// 获取所有节点
			nodes, err := clientset.CoreV1().Nodes().List(c.Request.Context(), metav1.ListOptions{})
			if err == nil {
				hasControlPlaneNode := false
				for _, node := range nodes.Items {
					nodeName := strings.ToLower(node.Name)

					// 检查节点是否是 master/control-plane 节点
					if _, hasControlPlane := node.Labels["node-role.kubernetes.io/control-plane"]; hasControlPlane {
						hasControlPlaneNode = true
						break
					}
					// 兼容旧的标签
					if _, hasMaster := node.Labels["node-role.kubernetes.io/master"]; hasMaster {
						hasControlPlaneNode = true
						break
					}

					// 如果节点名称包含 master/control-plane/mgr 等关键词，也认为是控制平面节点
					if strings.Contains(nodeName, "master") ||
						strings.Contains(nodeName, "control-plane") ||
						strings.Contains(nodeName, "control") ||
						strings.Contains(nodeName, "mgr") {
						hasControlPlaneNode = true
						break
					}
				}

				// 如果检测到控制平面节点但没有找到控制平面 Pod，说明是二进制部署
				if hasControlPlaneNode {
					// 添加 API Server
					componentInfo.Components = append(componentInfo.Components, ComponentInfo{
						Name:    "API Server",
						Version: serverVersion.GitVersion,
						Status:  "Running",
					})

					// 添加 Scheduler
					componentInfo.Components = append(componentInfo.Components, ComponentInfo{
						Name:    "Scheduler",
						Version: serverVersion.GitVersion,
						Status:  "Running",
					})

					// 添加 Controller Manager
					componentInfo.Components = append(componentInfo.Components, ComponentInfo{
						Name:    "Controller Manager",
						Version: serverVersion.GitVersion,
						Status:  "Running",
					})

					// 添加 etcd（版本未知）
					componentInfo.Components = append(componentInfo.Components, ComponentInfo{
						Name:    "etcd",
						Version: "unknown",
						Status:  "Running",
					})
				}
			}
		}
	}

	// 获取存储类
	storageClasses, err := clientset.StorageV1().StorageClasses().List(c.Request.Context(), metav1.ListOptions{})
	if err == nil {
		for _, sc := range storageClasses.Items {
			componentInfo.Storage = append(componentInfo.Storage, StorageInfo{
				Name:          sc.Name,
				Provisioner:   sc.Provisioner,
				ReclaimPolicy: string(*sc.ReclaimPolicy),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    componentInfo,
	})
}

// ListEvents 获取事件列表
func (h *ResourceHandler) ListEvents(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Query("namespace")
	fieldSelector := c.Query("fieldSelector")

	// 获取当前用户 ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	// 使用用户凭据获取 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	// 构建ListOptions，限制返回50条事件
	listOptions := metav1.ListOptions{
		Limit: 50,
	}

	// 添加 fieldSelector 过滤
	if fieldSelector != "" {
		listOptions.FieldSelector = fieldSelector
	}

	var events *v1.EventList
	if namespace != "" {
		// 获取指定命名空间的事件
		events, err = clientset.CoreV1().Events(namespace).List(c.Request.Context(), listOptions)
	} else {
		// 获取所有命名空间的事件
		events, err = clientset.CoreV1().Events("").List(c.Request.Context(), listOptions)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取事件列表失败: " + err.Error(),
		})
		return
	}

	eventInfos := make([]EventInfo, 0, len(events.Items))
	for _, event := range events.Items {
		// 获取来源信息
		source := event.Source.Component
		if event.Source.Host != "" {
			source = source + " (" + event.Source.Host + ")"
		}

		eventInfo := EventInfo{
			Type:    event.Type,
			Reason:  event.Reason,
			Message: event.Message,
			Source:  source,
			Count:   event.Count,
			InvolvedObject: InvolvedObjectInfo{
				Kind:      event.InvolvedObject.Kind,
				Name:      event.InvolvedObject.Name,
				Namespace: event.InvolvedObject.Namespace,
			},
		}

		// 格式化时间
		if !event.FirstTimestamp.IsZero() {
			eventInfo.FirstTimestamp = event.FirstTimestamp.Format("2006-01-02 15:04:05")
		}
		if !event.LastTimestamp.IsZero() {
			eventInfo.LastTimestamp = event.LastTimestamp.Format("2006-01-02 15:04:05")
		} else if !event.EventTime.IsZero() {
			eventInfo.LastTimestamp = event.EventTime.Format("2006-01-02 15:04:05")
		}

		eventInfos = append(eventInfos, eventInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    eventInfos,
	})
}

// GetAPIGroups 获取集群的API组列表
// @Summary 获取API组列表
// @Description 获取Kubernetes集群所有可用的API组
// @Tags Kubernetes/Resources
// @Accept json
// @Produce json
// @Param clusterId query int true "集群ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/resources/api-groups [get]
func (h *ResourceHandler) GetAPIGroups(c *gin.Context) {
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
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败",
		})
		return
	}

	// 获取所有API组
	discoveryClient := clientset.Discovery()
	apiGroupList, err := discoveryClient.ServerGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取API组失败: " + err.Error(),
		})
		return
	}

	// 收集所有API组名称
	apiGroups := make(map[string]bool)
	apiGroups["core"] = true // core API 用 "core" 表示

	for _, group := range apiGroupList.Groups {
		apiGroups[group.Name] = true
	}

	// 转换为切片并排序（core 放在最前面）
	groupList := make([]string, 0, len(apiGroups))
	// 先添加 core
	groupList = append(groupList, "core")
	// 再添加其他组（按字母排序）
	for group := range apiGroups {
		if group != "core" {
			groupList = append(groupList, group)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    groupList,
	})
}

// GetResourcesByAPIGroup 根据API组获取资源列表
// @Summary 根据API组获取资源列表
// @Description 根据选定的API组列表获取所有这些组下的资源类型
// @Tags Kubernetes/Resources
// @Accept json
// @Produce json
// @Param clusterId query int true "集群ID"
// @Param apiGroups query string true "API组列表（逗号分隔）"
// @Success 200 {object} map[string]interface{} "成功"
// @Router /plugins/kubernetes/resources/api-resources [get]
func (h *ResourceHandler) GetResourcesByAPIGroup(c *gin.Context) {
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

	apiGroupsStr := c.Query("apiGroups")
	if apiGroupsStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少API组参数",
		})
		return
	}

	// 解析API组列表
	apiGroups := strings.Split(apiGroupsStr, ",")
	// 将 "core" 转换为空字符串（Kubernetes core API group 的正确表示）
	for i, group := range apiGroups {
		if strings.TrimSpace(group) == "core" {
			apiGroups[i] = ""
		}
	}

	// 获取当前用户 ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	// 获取集群的 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterId), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败",
		})
		return
	}

	// 获取所有API资源和版本
	discoveryClient := clientset.Discovery()
	_, resourceLists, err := discoveryClient.ServerGroupsAndResources()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取资源列表失败: " + err.Error(),
		})
		return
	}

	// 使用map去重
	resourceMap := make(map[string]bool)

	// 收集所有指定API组的资源
	for _, resourceList := range resourceLists {
		// 提取GroupVersion中的组名
		groupVersion := resourceList.GroupVersion
		groupName := ""
		if strings.Contains(groupVersion, "/") {
			parts := strings.Split(groupVersion, "/")
			if len(parts) == 2 {
				groupName = parts[0]
			}
		}

		// 检查是否在请求的API组列表中
		matched := false
		for _, apiGroup := range apiGroups {
			apiGroup = strings.TrimSpace(apiGroup)
			if apiGroup == "" {
				// 空字符串表示core组，匹配 v1
				if groupVersion == "v1" {
					matched = true
					break
				}
			} else if groupName == apiGroup || groupVersion == apiGroup {
				matched = true
				break
			}
		}

		if matched {
			for _, resource := range resourceList.APIResources {
				// 过滤掉子资源（如 pods/status, pods/log 等）
				if !strings.Contains(resource.Name, "/") {
					resourceMap[resource.Name] = true
				}
			}
		}
	}

	// 转换为切片
	resources := make([]string, 0, len(resourceMap))
	for resource := range resourceMap {
		resources = append(resources, resource)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    resources,
	})
}

// GetNodeYAML 获取节点YAML
// @Summary 获取节点YAML
// @Description 获取指定节点的 YAML 配置
// @Tags Kubernetes/节点管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param clusterId query int true "集群ID"
// @Param nodeName path string true "节点名称"
// @Success 200 {object} map[string]interface{} "节点YAML"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/resources/nodes/{nodeName}/yaml [get]
func (h *ResourceHandler) GetNodeYAML(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	fmt.Printf("🔍 DEBUG [GetNodeYAML]: clusterIDStr=%s\n", clusterIDStr)

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		fmt.Printf("❌ DEBUG [GetNodeYAML]: Invalid clusterID: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	nodeName := c.Param("nodeName")
	if nodeName == "" {
		fmt.Printf("❌ DEBUG [GetNodeYAML]: Empty nodeName\n")
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "节点名称不能为空",
		})
		return
	}

	fmt.Printf("🔍 DEBUG [GetNodeYAML]: nodeName=%s, clusterID=%d\n", nodeName, clusterID)

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		fmt.Printf("❌ DEBUG [GetNodeYAML]: No user_id in context\n")
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	fmt.Printf("✅ DEBUG [GetNodeYAML]: userID=%v\n", currentUserID)

	// 获取clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID.(uint))
	if err != nil {
		fmt.Printf("❌ DEBUG [GetNodeYAML]: GetClientsetForUser failed: %v\n", err)
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	fmt.Printf("✅ DEBUG [GetNodeYAML]: Got clientset\n")

	// 获取节点
	node, err := clientset.CoreV1().Nodes().Get(c.Request.Context(), nodeName, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("❌ DEBUG [GetNodeYAML]: Get node failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取节点失败: " + err.Error(),
		})
		return
	}

	fmt.Printf("✅ DEBUG [GetNodeYAML]: Got node %s\n", node.Name)

	// 清理不需要的字段
	cleanedNode := cleanNodeForYAML(node)

	// 转换为YAML
	yamlBytes, err := yamlMarshal(cleanedNode)
	if err != nil {
		fmt.Printf("❌ DEBUG [GetNodeYAML]: Marshal YAML failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "转换YAML失败: " + err.Error(),
		})
		return
	}

	fmt.Printf("✅ DEBUG [GetNodeYAML]: YAML marshaled successfully, length=%d\n", len(yamlBytes))

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"yaml": string(yamlBytes),
		},
	})
}

// UpdateNodeYAMLRequest 更新节点YAML请求
type UpdateNodeYAMLRequest struct {
	ClusterID int    `json:"clusterId" binding:"required"`
	YAML      string `json:"yaml" binding:"required"`
}

// UpdateNodeYAML 更新节点YAML
func (h *ResourceHandler) UpdateNodeYAML(c *gin.Context) {
	nodeName := c.Param("nodeName")
	if nodeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "节点名称不能为空",
		})
		return
	}

	var req UpdateNodeYAMLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 获取clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID.(uint))
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 解析YAML为map
	var yamlData map[string]interface{}
	if err := yamlUnmarshal([]byte(req.YAML), &yamlData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析YAML失败: " + err.Error(),
		})
		return
	}

	// 验证节点名称
	if metadata, ok := yamlData["metadata"].(map[string]interface{}); ok {
		if name, ok := metadata["name"].(string); ok && name != nodeName {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "YAML中的节点名称与URL中的不一致",
			})
			return
		}
	}

	// 提取新的 labels
	var newLabels map[string]string
	if metadata, ok := yamlData["metadata"].(map[string]interface{}); ok {
		if labels, ok := metadata["labels"].(map[string]interface{}); ok {
			newLabels = make(map[string]string)
			for k, v := range labels {
				if strVal, ok := v.(string); ok {
					newLabels[k] = strVal
				} else {
					// 处理空值的情况
					newLabels[k] = ""
				}
			}
		}
	}

	if newLabels == nil {
		newLabels = make(map[string]string)
	}

	fmt.Printf("🔍 DEBUG [UpdateNodeYAML]: New labels: %+v\n", newLabels)

	// 提取新的 taints
	var newTaints []v1.Taint
	if spec, ok := yamlData["spec"].(map[string]interface{}); ok {
		if taintsData, ok := spec["taints"].([]interface{}); ok {
			newTaints = make([]v1.Taint, 0, len(taintsData))
			for _, taintItem := range taintsData {
				if taintMap, ok := taintItem.(map[string]interface{}); ok {
					taint := v1.Taint{}
					if key, ok := taintMap["key"].(string); ok {
						taint.Key = key
					}
					if value, ok := taintMap["value"].(string); ok {
						taint.Value = value
					}
					if effect, ok := taintMap["effect"].(string); ok {
						taint.Effect = v1.TaintEffect(effect)
					}
					newTaints = append(newTaints, taint)
				}
			}
		}
	}

	fmt.Printf("🔍 DEBUG [UpdateNodeYAML]: New taints: %+v\n", newTaints)

	// 先获取当前节点
	node, err := clientset.CoreV1().Nodes().Get(c.Request.Context(), nodeName, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("❌ DEBUG [UpdateNodeYAML]: Get node failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取节点失败: " + err.Error(),
		})
		return
	}

	// 完全替换 labels
	node.Labels = newLabels

	// 完全替换 taints
	node.Spec.Taints = newTaints

	// 使用 Update 方法更新节点（这样可以确保 labels 和 taints 被完全替换）
	_, err = clientset.CoreV1().Nodes().Update(
		c.Request.Context(),
		node,
		metav1.UpdateOptions{},
	)
	if err != nil {
		fmt.Printf("❌ DEBUG [UpdateNodeYAML]: Update failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新节点失败: " + err.Error(),
		})
		return
	}

	fmt.Printf("✅ DEBUG [UpdateNodeYAML]: Updated node %s successfully with %d labels and %d taints\n", nodeName, len(newLabels), len(newTaints))

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
	})
}

// DrainNodeRequest 排空节点请求
type DrainNodeRequest struct {
	ClusterID int `json:"clusterId" binding:"required"`
}

// DrainNode 排空节点
// @Summary 排空节点
// @Description 排空指定节点上的所有 Pod
// @Tags Kubernetes/节点管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param clusterId query int true "集群ID"
// @Param nodeName path string true "节点名称"
// @Success 200 {object} map[string]interface{} "操作成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/resources/nodes/{nodeName}/drain [post]
func (h *ResourceHandler) DrainNode(c *gin.Context) {
	nodeName := c.Param("nodeName")
	if nodeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "节点名称不能为空",
		})
		return
	}

	var req DrainNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 获取clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID.(uint))
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	fmt.Printf("🔍 DEBUG [DrainNode]: Starting drain for node %s\n", nodeName)

	// 获取节点上的所有Pod
	pods, err := clientset.CoreV1().Pods("").List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("❌ DEBUG [DrainNode]: List pods failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取Pod列表失败: " + err.Error(),
		})
		return
	}

	// 驱逐该节点上的所有Pod（除了DaemonSet的Pod）
	evictedCount := 0
	for _, pod := range pods.Items {
		if pod.Spec.NodeName != nodeName {
			continue
		}

		// 跳过DaemonSet管理的Pod
		if pod.OwnerReferences != nil {
			isDaemonSet := false
			for _, ownerRef := range pod.OwnerReferences {
				if ownerRef.Kind == "DaemonSet" {
					isDaemonSet = true
					break
				}
			}
			if isDaemonSet {
				fmt.Printf("⏭️  DEBUG [DrainNode]: Skipping DaemonSet pod %s\n", pod.Name)
				continue
			}
		}

		// 驱逐Pod
		err = clientset.CoreV1().Pods(pod.Namespace).EvictV1(context.Background(), &policyv1.Eviction{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pod.Name,
				Namespace: pod.Namespace,
			},
		})
		if err != nil {
			fmt.Printf("⚠️  DEBUG [DrainNode]: Failed to evict pod %s/%s: %v\n", pod.Namespace, pod.Name, err)
			// 继续驱逐其他Pod，不中断
			continue
		}
		evictedCount++
		fmt.Printf("✅ DEBUG [DrainNode]: Evicted pod %s/%s\n", pod.Namespace, pod.Name)
	}

	fmt.Printf("✅ DEBUG [DrainNode]: Drain completed for node %s, evicted %d pods\n", nodeName, evictedCount)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "节点排空成功",
		"data": gin.H{
			"evictedPods": evictedCount,
		},
	})
}

// CordonNodeRequest 设为不可调度请求
type CordonNodeRequest struct {
	ClusterID int `json:"clusterId" binding:"required"`
}

// CordonNode 设为不可调度
// @Summary 设置节点为不可调度
// @Description 将指定节点设置为不可调度状态
// @Tags Kubernetes/节点管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param clusterId query int true "集群ID"
// @Param nodeName path string true "节点名称"
// @Success 200 {object} map[string]interface{} "操作成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/resources/nodes/{nodeName}/cordon [post]
func (h *ResourceHandler) CordonNode(c *gin.Context) {
	nodeName := c.Param("nodeName")
	if nodeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "节点名称不能为空",
		})
		return
	}

	var req CordonNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 获取clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID.(uint))
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 获取节点
	node, err := clientset.CoreV1().Nodes().Get(c.Request.Context(), nodeName, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取节点失败: " + err.Error(),
		})
		return
	}

	// 检查是否已经是不可调度状态
	if node.Spec.Unschedulable {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "节点已经是不可调度状态",
		})
		return
	}

	// 设为不可调度
	node.Spec.Unschedulable = true
	_, err = clientset.CoreV1().Nodes().Update(c.Request.Context(), node, metav1.UpdateOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "设为不可调度失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "节点已设为不可调度",
	})
}

// UncordonNode 设为可调度
func (h *ResourceHandler) UncordonNode(c *gin.Context) {
	nodeName := c.Param("nodeName")
	if nodeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "节点名称不能为空",
		})
		return
	}

	var req CordonNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 获取clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID.(uint))
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 获取节点
	node, err := clientset.CoreV1().Nodes().Get(c.Request.Context(), nodeName, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取节点失败: " + err.Error(),
		})
		return
	}

	// 检查是否已经是可调度状态
	if !node.Spec.Unschedulable {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "节点已经是可调度状态",
		})
		return
	}

	// 设为可调度
	node.Spec.Unschedulable = false
	_, err = clientset.CoreV1().Nodes().Update(c.Request.Context(), node, metav1.UpdateOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "设为可调度失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "节点已设为可调度",
	})
}

// DeleteNode 删除节点
func (h *ResourceHandler) DeleteNode(c *gin.Context) {
	nodeName := c.Param("nodeName")
	if nodeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "节点名称不能为空",
		})
		return
	}

	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	fmt.Printf("🔍 DEBUG [DeleteNode]: Deleting node %s, clusterID=%d\n", nodeName, clusterID)

	// 获取clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID.(uint))
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 删除节点
	err = clientset.CoreV1().Nodes().Delete(c.Request.Context(), nodeName, metav1.DeleteOptions{})
	if err != nil {
		fmt.Printf("❌ DEBUG [DeleteNode]: Delete node failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除节点失败: " + err.Error(),
		})
		return
	}

	fmt.Printf("✅ DEBUG [DeleteNode]: Node %s deleted successfully\n", nodeName)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "节点删除成功",
	})
}

// BatchNodesRequest 批量节点操作请求
type BatchNodesRequest struct {
	ClusterID int      `json:"clusterId" binding:"required"`
	NodeNames []string `json:"nodeNames" binding:"required"`
}

// BatchNodeLabelsRequest 批量节点标签操作请求
type BatchNodeLabelsRequest struct {
	ClusterID int                    `json:"clusterId" binding:"required"`
	NodeNames []string              `json:"nodeNames" binding:"required"`
	Labels    map[string]string     `json:"labels" binding:"required"`
	Operation string                `json:"operation" binding:"required"` // add, remove, replace
}

// BatchNodeTaintsRequest 批量节点污点操作请求
type BatchNodeTaintsRequest struct {
	ClusterID int                    `json:"clusterId" binding:"required"`
	NodeNames []string              `json:"nodeNames" binding:"required"`
	Taints    []TaintInfo           `json:"taints" binding:"required"`
	Operation string                `json:"operation" binding:"required"` // add, remove
}

// BatchDrainNodesRequest 批量排空节点请求
type BatchDrainNodesRequest struct {
	ClusterID            int      `json:"clusterId" binding:"required"`
	NodeNames            []string `json:"nodeNames" binding:"required"`
	Force                bool     `json:"force"`
	IgnoreDaemonsets     bool     `json:"ignoreDaemonsets"`
	DeleteLocalData      bool     `json:"deleteLocalData"`
	GracePeriodSeconds   int      `json:"gracePeriodSeconds"`
	Timeout              int      `json:"timeout"`
}

// BatchOperationResult 批量操作结果
type BatchOperationResult struct {
	NodeName  string `json:"nodeName"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
}

// BatchDrainNodes 批量排空节点
func (h *ResourceHandler) BatchDrainNodes(c *gin.Context) {
	var req BatchDrainNodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	if len(req.NodeNames) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "节点名称列表不能为空",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 获取clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID.(uint))
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	results := make([]BatchOperationResult, 0, len(req.NodeNames))

	for _, nodeName := range req.NodeNames {
		// 获取节点上的Pod列表
		pods, err := clientset.CoreV1().Pods("").List(c.Request.Context(), metav1.ListOptions{
			FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
		})
		if err != nil {
			results = append(results, BatchOperationResult{
				NodeName: nodeName,
				Success:  false,
				Message:  "获取Pod列表失败: " + err.Error(),
			})
			continue
		}

		// 删除Pod
		failedPods := []string{}
		for _, pod := range pods.Items {
			// 跳过 DaemonSet 管理的 Pod（除非强制删除）
			if !req.Force && isPodOwnedByDaemonSet(&pod) {
				continue
			}

			// 跳过本地存储的 Pod（除非强制删除）
			if !req.DeleteLocalData && hasLocalStorage(&pod) {
				continue
			}

			// 设置删除宽限期
			gracePeriod := int64(req.GracePeriodSeconds)
			if req.GracePeriodSeconds <= 0 {
				gracePeriod = 30 // 默认30秒
			}

			deleteOptions := metav1.DeleteOptions{
				GracePeriodSeconds: &gracePeriod,
			}

			err := clientset.CoreV1().Pods(pod.Namespace).Delete(c.Request.Context(), pod.Name, deleteOptions)
			if err != nil {
				failedPods = append(failedPods, fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
			}
		}

		if len(failedPods) > 0 {
			results = append(results, BatchOperationResult{
				NodeName: nodeName,
				Success:  false,
				Message:  fmt.Sprintf("部分Pod删除失败: %v", failedPods),
			})
		} else {
			results = append(results, BatchOperationResult{
				NodeName: nodeName,
				Success:  true,
				Message:  "排空成功",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "批量排空操作完成",
		"data": gin.H{
			"results": results,
		},
	})
}

// BatchCordonNodes 批量设为不可调度
func (h *ResourceHandler) BatchCordonNodes(c *gin.Context) {
	var req BatchNodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	if len(req.NodeNames) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "节点名称列表不能为空",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 获取clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID.(uint))
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	results := make([]BatchOperationResult, 0, len(req.NodeNames))

	for _, nodeName := range req.NodeNames {
		err := h.cordonNode(c.Request.Context(), clientset, nodeName, true)
		if err != nil {
			results = append(results, BatchOperationResult{
				NodeName: nodeName,
				Success:  false,
				Message:  err.Error(),
			})
		} else {
			results = append(results, BatchOperationResult{
				NodeName: nodeName,
				Success:  true,
				Message:  "设为不可调度成功",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "批量设为不可调度完成",
		"data": gin.H{
			"results": results,
		},
	})
}

// BatchUncordonNodes 批量设为可调度
func (h *ResourceHandler) BatchUncordonNodes(c *gin.Context) {
	var req BatchNodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	if len(req.NodeNames) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "节点名称列表不能为空",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 获取clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID.(uint))
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	results := make([]BatchOperationResult, 0, len(req.NodeNames))

	for _, nodeName := range req.NodeNames {
		err := h.cordonNode(c.Request.Context(), clientset, nodeName, false)
		if err != nil {
			results = append(results, BatchOperationResult{
				NodeName: nodeName,
				Success:  false,
				Message:  err.Error(),
			})
		} else {
			results = append(results, BatchOperationResult{
				NodeName: nodeName,
				Success:  true,
				Message:  "设为可调度成功",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "批量设为可调度完成",
		"data": gin.H{
			"results": results,
		},
	})
}

// BatchDeleteNodes 批量删除节点
func (h *ResourceHandler) BatchDeleteNodes(c *gin.Context) {
	var req BatchNodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	if len(req.NodeNames) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "节点名称列表不能为空",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 获取clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID.(uint))
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	results := make([]BatchOperationResult, 0, len(req.NodeNames))

	for _, nodeName := range req.NodeNames {
		err := clientset.CoreV1().Nodes().Delete(c.Request.Context(), nodeName, metav1.DeleteOptions{})
		if err != nil {
			results = append(results, BatchOperationResult{
				NodeName: nodeName,
				Success:  false,
				Message:  err.Error(),
			})
		} else {
			results = append(results, BatchOperationResult{
				NodeName: nodeName,
				Success:  true,
				Message:  "删除成功",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "批量删除节点完成",
		"data": gin.H{
			"results": results,
		},
	})
}

// BatchUpdateNodeLabels 批量更新节点标签
func (h *ResourceHandler) BatchUpdateNodeLabels(c *gin.Context) {
	var req BatchNodeLabelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	if len(req.NodeNames) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "节点名称列表不能为空",
		})
		return
	}

	// 验证操作类型
	if req.Operation != "add" && req.Operation != "remove" && req.Operation != "replace" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的操作类型，必须是 add、remove 或 replace",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 获取clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID.(uint))
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	results := make([]BatchOperationResult, 0, len(req.NodeNames))

	for _, nodeName := range req.NodeNames {
		// 获取节点
		node, err := clientset.CoreV1().Nodes().Get(c.Request.Context(), nodeName, metav1.GetOptions{})
		if err != nil {
			results = append(results, BatchOperationResult{
				NodeName: nodeName,
				Success:  false,
				Message:  "获取节点失败: " + err.Error(),
			})
			continue
		}

		// 处理标签操作
		switch req.Operation {
		case "add":
			// 添加标签（不覆盖已有标签）
			for key, value := range req.Labels {
				if node.Labels == nil {
					node.Labels = make(map[string]string)
				}
				if _, exists := node.Labels[key]; !exists {
					node.Labels[key] = value
				}
			}
		case "remove":
			// 删除标签
			for key := range req.Labels {
				delete(node.Labels, key)
			}
		case "replace":
			// 替换标签
			if node.Labels == nil {
				node.Labels = make(map[string]string)
			}
			for key, value := range req.Labels {
				node.Labels[key] = value
			}
		}

		// 更新节点
		_, err = clientset.CoreV1().Nodes().Update(c.Request.Context(), node, metav1.UpdateOptions{})
		if err != nil {
			results = append(results, BatchOperationResult{
				NodeName: nodeName,
				Success:  false,
				Message:  "更新节点失败: " + err.Error(),
			})
		} else {
			results = append(results, BatchOperationResult{
				NodeName: nodeName,
				Success:  true,
				Message:  "标签更新成功",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "批量更新节点标签完成",
		"data": gin.H{
			"results": results,
		},
	})
}

// BatchUpdateNodeTaints 批量更新节点污点
func (h *ResourceHandler) BatchUpdateNodeTaints(c *gin.Context) {
	var req BatchNodeTaintsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	if len(req.NodeNames) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "节点名称列表不能为空",
		})
		return
	}

	// 验证操作类型
	if req.Operation != "add" && req.Operation != "remove" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的操作类型，必须是 add 或 remove",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 获取clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID.(uint))
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	results := make([]BatchOperationResult, 0, len(req.NodeNames))

	for _, nodeName := range req.NodeNames {
		// 获取节点
		node, err := clientset.CoreV1().Nodes().Get(c.Request.Context(), nodeName, metav1.GetOptions{})
		if err != nil {
			results = append(results, BatchOperationResult{
				NodeName: nodeName,
				Success:  false,
				Message:  "获取节点失败: " + err.Error(),
			})
			continue
		}

		// 转换污点格式
		newTaints := make([]v1.Taint, 0, len(req.Taints))
		for _, taint := range req.Taints {
			newTaints = append(newTaints, v1.Taint{
				Key:    taint.Key,
				Value:  taint.Value,
				Effect: v1.TaintEffect(taint.Effect),
			})
		}

		// 处理污点操作
		switch req.Operation {
		case "add":
			// 添加污点
			for _, newTaint := range newTaints {
				found := false
				for i, existingTaint := range node.Spec.Taints {
					if existingTaint.Key == newTaint.Key && existingTaint.Effect == newTaint.Effect {
						// 更新已存在的污点
						node.Spec.Taints[i] = newTaint
						found = true
						break
					}
				}
				if !found {
					node.Spec.Taints = append(node.Spec.Taints, newTaint)
				}
			}
		case "remove":
			// 删除污点
			updatedTaints := make([]v1.Taint, 0, len(node.Spec.Taints))
			for _, existingTaint := range node.Spec.Taints {
				shouldRemove := false
				for _, taintToRemove := range newTaints {
					if existingTaint.Key == taintToRemove.Key && existingTaint.Effect == taintToRemove.Effect {
						shouldRemove = true
						break
					}
				}
				if !shouldRemove {
					updatedTaints = append(updatedTaints, existingTaint)
				}
			}
			node.Spec.Taints = updatedTaints
		}

		// 更新节点
		_, err = clientset.CoreV1().Nodes().Update(c.Request.Context(), node, metav1.UpdateOptions{})
		if err != nil {
			results = append(results, BatchOperationResult{
				NodeName: nodeName,
				Success:  false,
				Message:  "更新节点失败: " + err.Error(),
			})
		} else {
			results = append(results, BatchOperationResult{
				NodeName: nodeName,
				Success:  true,
				Message:  "污点更新成功",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "批量更新节点污点完成",
		"data": gin.H{
			"results": results,
		},
	})
}

// cordonNode 设置节点为可调度/不可调度
func (h *ResourceHandler) cordonNode(ctx context.Context, clientset *kubernetes.Clientset, nodeName string, unschedulable bool) error {
	node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取节点失败: %w", err)
	}

	node.Spec.Unschedulable = unschedulable

	_, err = clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("更新节点失败: %w", err)
	}

	return nil
}

// isPodOwnedByDaemonSet 检查Pod是否由DaemonSet管理
func isPodOwnedByDaemonSet(pod *v1.Pod) bool {
	for _, ownerRef := range pod.OwnerReferences {
		if ownerRef.Kind == "DaemonSet" {
			return true
		}
	}
	return false
}

// hasLocalStorage 检查Pod是否有本地存储
func hasLocalStorage(pod *v1.Pod) bool {
	for _, volume := range pod.Spec.Volumes {
		if volume.EmptyDir != nil || volume.HostPath != nil {
			return true
		}
	}
	return false
}

// yamlMarshal 简单的YAML序列化
func yamlMarshal(obj interface{}) ([]byte, error) {
	return yaml.Marshal(obj)
}

// yamlUnmarshal 简单的YAML反序列化
func yamlUnmarshal(data []byte, obj interface{}) error {
	return yaml.Unmarshal(data, obj)
}

// cleanNodeForYAML 清理Node对象用于YAML输出
func cleanNodeForYAML(node *v1.Node) map[string]interface{} {
	// 创建一个副本，避免修改原始对象
	cleaned := node.DeepCopy()

	// 移除 managedFields
	if cleaned.ObjectMeta.ManagedFields != nil {
		cleaned.ObjectMeta.ManagedFields = nil
	}

	// 转换为 map 以便控制 YAML 序列化顺序
	result := make(map[string]interface{})

	// 确保 apiVersion 和 kind 在最前面
	result["apiVersion"] = "v1"
	result["kind"] = "Node"

	// 添加 metadata
	metadata := make(map[string]interface{})
	if cleaned.Name != "" {
		metadata["name"] = cleaned.Name
	}
	if len(cleaned.Labels) > 0 {
		metadata["labels"] = cleaned.Labels
	}
	if len(cleaned.Annotations) > 0 {
		metadata["annotations"] = cleaned.Annotations
	}
	// 不包含 resourceVersion，使用 PATCH 更新时不需要
	if len(cleaned.Finalizers) > 0 {
		metadata["finalizers"] = cleaned.Finalizers
	}

	result["metadata"] = metadata

	// 添加 spec
	spec := make(map[string]interface{})
	if cleaned.Spec.PodCIDR != "" {
		spec["podCIDR"] = cleaned.Spec.PodCIDR
	}
	if len(cleaned.Spec.PodCIDRs) > 0 {
		spec["podCIDRs"] = cleaned.Spec.PodCIDRs
	}
	if cleaned.Spec.ProviderID != "" {
		spec["providerID"] = cleaned.Spec.ProviderID
	}
	if cleaned.Spec.Unschedulable {
		spec["unschedulable"] = cleaned.Spec.Unschedulable
	}
	if len(cleaned.Spec.Taints) > 0 {
		spec["taints"] = cleaned.Spec.Taints
	}
	if cleaned.Spec.ConfigSource != nil {
		spec["configSource"] = cleaned.Spec.ConfigSource
	}

	result["spec"] = spec

	// 不包含 status，因为 status 是由 Kubernetes 自动管理的

	return result
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境应该更严格
	},
}

// NodeShell WebSocket 处理器 - 使用 debug pod 方式
func (h *ResourceHandler) NodeShellWebSocket(c *gin.Context) {
	nodeName := c.Param("nodeName")
	if nodeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "节点名称不能为空",
		})
		return
	}

	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 升级到 WebSocket 连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	fmt.Printf("🐚 WebSocket shell connected to node %s, clusterID=%d\n", nodeName, clusterID)

	// 获取 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID.(uint))
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("获取集群客户端失败: "+err.Error()+"\r\n"))
		return
	}

	// 获取 REST config
	restConfig, err := h.clusterService.GetRESTConfig(uint(clusterID), currentUserID.(uint))
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("获取集群配置失败: "+err.Error()+"\r\n"))
		return
	}

	// 创建临时 debug pod
	debugPodName := fmt.Sprintf("debug-%s-%d", nodeName, time.Now().Unix())
	debugNamespace := "default"

	conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("正在创建临时 debug pod: %s...\r\n", debugPodName)))

	// 定义 debug pod（使用 node profile 共享节点命名空间）
	debugPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      debugPodName,
			Namespace: debugNamespace,
			Labels: map[string]string{
				"app":        "opshub-debug",
				"node":       nodeName,
				"created-by": "opshub",
			},
		},
		Spec: v1.PodSpec{
			// 使用节点亲和性确保调度到目标节点
			NodeName: nodeName,
			// 使用 hostPID 和 hostNetwork 共享节点的进程和网络命名空间
			HostPID:       true,
			HostNetwork:   true,
			RestartPolicy: v1.RestartPolicyNever,
			// 容器配置
			Containers: []v1.Container{
				{
					Name:    "debug",
					Image:   "swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/nicolaka/netshoot:latest",
					Command: []string{"/bin/bash"},
					Args:    []string{"-c", "sleep 3600"}, // 保持运行
					Stdin:   true,
					TTY:     true,
					// 安全上下文
					SecurityContext: &v1.SecurityContext{
						Privileged: func() *bool { b := true; return &b }(),
					},
				},
			},
			//容忍所有污点，确保可以调度到任何节点
			Tolerations: []v1.Toleration{
				{
					Operator: v1.TolerationOpExists,
				},
			},
		},
	}

	// 创建 debug pod
	createdPod, err := clientset.CoreV1().Pods(debugNamespace).Create(c.Request.Context(), debugPod, metav1.CreateOptions{})
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("创建 debug pod 失败: "+err.Error()+"\r\n"))
		return
	}

	fmt.Printf("🐚 Created debug pod: %s/%s\n", debugNamespace, createdPod.Name)
	conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Debug pod 创建成功，等待启动...\r\n")))

	// 等待 pod 启动（最多等待30秒）
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	err = h.waitForPodReady(ctx, clientset, debugNamespace, debugPodName, conn)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("等待 debug pod 启动失败: "+err.Error()+"\r\n"))
		// 清理 pod
		clientset.CoreV1().Pods(debugNamespace).Delete(ctx, debugPodName, metav1.DeleteOptions{})
		return
	}

	conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("已连接到节点 %s\r\n\r\n", nodeName)))

	// 确保在连接关闭时清理 pod
	defer func() {
		fmt.Printf("🐚 Cleaning up debug pod: %s/%s\n", debugNamespace, debugPodName)
		clientset.CoreV1().Pods(debugNamespace).Delete(context.Background(), debugPodName, metav1.DeleteOptions{})
	}()

	// 构造 exec URL
	serverURL, err := url.Parse(restConfig.Host)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("解析集群 URL 失败: "+err.Error()+"\r\n"))
		return
	}

	// 构造 query 参数
	query := url.Values{}
	query.Set("container", "debug")
	query.Set("stdin", "true")
	query.Set("stdout", "true")
	query.Set("stderr", "true")
	query.Set("tty", "true")

	// 使用 nsenter 进入节点根命名空间
	query.Add("command", "/bin/bash")
	query.Add("command", "-c")
	query.Add("command", "nsenter -t 1 -m -u -i -n -p -- /bin/bash || /bin/bash")

	execURL := &url.URL{
		Scheme:   serverURL.Scheme,
		Host:     serverURL.Host,
		Path:     fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/exec", debugNamespace, debugPodName),
		RawQuery: query.Encode(),
	}

	fmt.Printf("🐚 Exec URL: %s\n", execURL.String())

	// 创建 SPDY executor
	executor, err := remotecommand.NewSPDYExecutor(restConfig, "POST", execURL)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("创建 executor 失败: "+err.Error()+"\r\n"))
		return
	}

	// 创建 WebSocket 读写器
	wsReader := &WebSocketReader{
		conn: conn,
		data: make(chan []byte, 256),
	}
	wsWriter := &WebSocketWriter{conn: conn}

	// 处理 WebSocket 消息
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("🐚 WebSocket read error: %v\n", err)
				return
			}
			wsReader.data <- data
		}
	}()

	// 发送初始消息
	conn.WriteMessage(websocket.TextMessage, []byte("连接成功，正在初始化 shell...\r\n"))

	// 启动 exec 会话，使用 chroot 或 nsenter 进入节点 shell
	// 注意：这里需要容器有足够权限（特权容器），通常使用 kube-system 的 Pod
	streamOptions := remotecommand.StreamOptions{
		Stdin:  wsReader,
		Stdout: wsWriter,
		Stderr: wsWriter,
		Tty:    true,
	}

	// 执行远程命令（命令已在 URL query 参数中指定）
	err = executor.Stream(streamOptions)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Shell 执行失败: "+err.Error()+"\r\n"))
		fmt.Printf("🐚 Shell execution error: %v\n", err)
	}

	<-done
	fmt.Printf("🐚 WebSocket shell disconnected from node %s\n", nodeName)
}

// WebSocketReader 实现 io.Reader 接口
type WebSocketReader struct {
	conn *websocket.Conn
	data chan []byte
}

func (r *WebSocketReader) Read(p []byte) (int, error) {
	data, ok := <-r.data
	if !ok {
		return 0, io.EOF
	}
	n := copy(p, data)
	return n, nil
}

// WebSocketWriter 实现 io.Writer 接口
type WebSocketWriter struct {
	conn *websocket.Conn
}

func (w *WebSocketWriter) Write(p []byte) (int, error) {
	err := w.conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// waitForPodReady 等待 Pod 准备就绪
func (h *ResourceHandler) waitForPodReady(ctx context.Context, clientset *kubernetes.Clientset, namespace, podName string, conn *websocket.Conn) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("获取 pod 状态失败: %w", err)
			}

			switch pod.Status.Phase {
			case v1.PodRunning:
				// 检查容器是否就绪
				for _, cs := range pod.Status.ContainerStatuses {
					if !cs.Ready {
						// 容器还未就绪，继续等待
						goto continueWait
					}
				}
				fmt.Printf("🐚 Pod %s/%s is ready\n", namespace, podName)
				return nil
			case v1.PodFailed, v1.PodSucceeded:
				return fmt.Errorf("pod %s/%s 处于 %s 状态", namespace, podName, pod.Status.Phase)
			}
		}
	continueWait:
	}
}

// GetCloudTTYStatus 检查 CloudTTY 是否已安装
func (h *ResourceHandler) GetCloudTTYStatus(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 获取 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID.(uint))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"data":    gin.H{"installed": false},
			"message": "获取集群客户端失败",
		})
		return
	}

	// 检查 CloudTTY deployment 是否存在
	// 尝试多个可能的deployment名称
	deployments := []string{
		"cloudtty-operator-controller-manager",
		"cloudtty-controller-manager",
	}
	installed := false
	for _, deployName := range deployments {
		_, err = clientset.AppsV1().Deployments("cloudtty-system").Get(c.Request.Context(), deployName, metav1.GetOptions{})
		if err == nil {
			installed = true
			fmt.Printf("✅ [GetCloudTTYStatus] CloudTTY 已安装，找到 Deployment: %s\n", deployName)
			break
		}
	}

	if !installed {
		fmt.Printf("❌ [GetCloudTTYStatus] CloudTTY 未安装，尝试的 Deployment: %v\n", deployments)
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{"installed": false},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{"installed": true},
	})
}

// DeployCloudTTY 部署 CloudTTY
func (h *ResourceHandler) DeployCloudTTY(c *gin.Context) {
	var req struct {
		ClusterID int `json:"clusterId" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 获取 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败",
		})
		return
	}

	// 创建 cloudtty-system 命名空间
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cloudtty-system",
			Labels: map[string]string{
				"name": "cloudtty-system",
			},
		},
	}

	_, err = clientset.CoreV1().Namespaces().Create(c.Request.Context(), ns, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建命名空间失败: " + err.Error(),
		})
		return
	}

	// CloudTTY CRD 定义（如果需要）
	// 注意：实际部署 CloudTTY 需要使用 kubectl apply 或者 helm
	// 这里提供一个简化版本，实际应该使用 CloudTTY 官方安装方式

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "CloudTTY 部署功能需要使用官方 helm chart 或 kubectl manifest",
		"data": gin.H{
			"note": "请使用以下命令部署 CloudTTY:",
			"commands": []string{
				"helm repo add cloudtty https://cloudtty.github.io/cloudtty",
				"helm repo update",
				"helm install cloudtty cloudtty/cloudtty -n cloudtty-system --create-namespace",
			},
		},
	})
}

// GetCloudTTYService 获取 CloudTTY 服务信息
func (h *ResourceHandler) GetCloudTTYService(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 获取 clientset（用于获取节点信息）
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败",
		})
		return
	}

	// 获取集群的 kubeconfig 用于 kubectl 命令
	kubeConfig, err := h.clusterService.GetClusterKubeConfig(c.Request.Context(), uint(clusterID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群 kubeconfig 失败",
		})
		return
	}

	// 将 kubeconfig 写入临时文件
	tmpFile, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建临时文件失败",
		})
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(kubeConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "写入 kubeconfig 失败",
		})
		return
	}

	// 使用 kubectl 获取 cloudshell CR
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kubectl", "get", "cloudshell", "-n", "cloudtty-system", "-o", "json")
	cmd.Env = append(os.Environ(), "KUBECONFIG="+tmpFile.Name())

	output, err := cmd.CombinedOutput()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"data":    nil,
			"message": fmt.Sprintf("CloudTTY cloudshell未找到: %v, output: %s", err, string(output)),
		})
		return
	}

	// 解析 JSON 输出
	var result struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
			Status struct {
				AccessURL string `json:"accessURL"`
				Phase     string `json:"phase"`
			} `json:"status"`
		} `json:"items"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("解析 cloudshell 数据失败: %v", err),
		})
		return
	}

	if len(result.Items) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"data":    nil,
			"message": "CloudTTY cloudshell 实例未找到",
		})
		return
	}

	// 获取第一个 cloudshell 实例
	cloudshell := result.Items[0]

	// 检查 cloudshell 状态是否就绪（Ready 或 Complete 都表示可用）
	if cloudshell.Status.Phase != "Ready" && cloudshell.Status.Phase != "Complete" {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"data":    nil,
			"message": fmt.Sprintf("CloudTTY cloudshell 状态未就绪: %s", cloudshell.Status.Phase),
		})
		return
	}

	if cloudshell.Status.AccessURL == "" {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"data":    nil,
			"message": "CloudTTY cloudshell AccessURL 为空",
		})
		return
	}

	// 解析 AccessURL 提取端口号（格式: "IP:PORT"）
	// 注意：IP 可能是 Service ClusterIP，我们需要使用节点 IP
	accessURL := cloudshell.Status.AccessURL
	parts := strings.Split(accessURL, ":")
	if len(parts) != 2 {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"data":    nil,
			"message": fmt.Sprintf("CloudTTY AccessURL 格式错误: %s", accessURL),
		})
		return
	}

	nodePort := parts[1]

	// 获取集群节点列表，选择一个节点的 IP
	nodes, err := clientset.CoreV1().Nodes().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取节点列表失败",
		})
		return
	}

	if len(nodes.Items) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"data":    nil,
			"message": "集群中没有可用节点",
		})
		return
	}

	// 获取第一个节点的 IP（优先使用 InternalIP）
	var nodeIP string
	for _, addr := range nodes.Items[0].Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			nodeIP = addr.Address
			break
		}
	}

	if nodeIP == "" {
		// 如果没有 InternalIP，使用第一个地址
		nodeIP = nodes.Items[0].Status.Addresses[0].Address
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"nodeIP":    nodeIP,
			"port":      nodePort,
			"type":      "NodePort",
			"path":      "/",
			"installed": true,
			"ready":     true,
		},
	})
}

// CreateCloudTTYService 创建 CloudTTY Service
func (h *ResourceHandler) CreateCloudTTYService(c *gin.Context) {
	var req struct {
		ClusterID int `json:"clusterId" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 获取 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败",
		})
		return
	}

	// 获取一个节点IP
	nodes, err := clientset.CoreV1().Nodes().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil || len(nodes.Items) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取节点列表失败",
		})
		return
	}
	nodeIP := nodes.Items[0].Status.Addresses[0].Address

	// 创建CloudTTY Service
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cloudtty",
			Namespace: "cloudtty-system",
			Labels: map[string]string{
				"app": "cloudtty",
			},
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeNodePort,
			Ports: []v1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.IntOrString{IntVal: 30000},
					NodePort:   30000,
				},
			},
			Selector: map[string]string{
				"app": "cloudtty",
			},
		},
	}

	_, err = clientset.CoreV1().Services("cloudtty-system").Create(c.Request.Context(), svc, metav1.CreateOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusOK, gin.H{
				"code":    0,
				"message": "CloudTTY Service已存在",
				"data": gin.H{
					"nodeIP": nodeIP,
					"port":   30000,
					"path":   "/cloudtty",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建Service失败: " + err.Error(),
		})
		return
	}

	fmt.Printf("✅ Created CloudTTY Service: %s:%d\n", nodeIP, 30000)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "CloudTTY Service创建成功",
		"data": gin.H{
			"nodeIP": nodeIP,
			"port":   30000,
			"path":   "/cloudtty",
		},
	})
}

// WorkloadInfo 工作负载信息
type WorkloadInfo struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Type        string            `json:"type"`
	Labels      map[string]string `json:"labels"`
	ReadyPods   int32             `json:"readyPods"`
	DesiredPods int32             `json:"desiredPods"`
	Requests    *ResourceInfo     `json:"requests,omitempty"`
	Limits      *ResourceInfo     `json:"limits,omitempty"`
	Images      []string          `json:"images,omitempty"`
	CreatedAt   string            `json:"createdAt"`
	UpdatedAt   string            `json:"updatedAt"`
	// DaemonSet 专用字段
	CurrentScheduled *int32 `json:"currentScheduled,omitempty"`
	DesiredScheduled *int32 `json:"desiredScheduled,omitempty"`
	// Job 专用字段
	Status   *string `json:"status,omitempty"`
	Duration *string `json:"duration,omitempty"`
	// CronJob 专用字段
	Schedule         *string `json:"schedule,omitempty"`
	LastScheduleTime *string `json:"lastScheduleTime,omitempty"`
	Suspended        *bool   `json:"suspended,omitempty"`
	// Pod 专用字段
	Containers   *string `json:"containers,omitempty"`
	CPU          *string `json:"cpu,omitempty"`
	Memory       *string `json:"memory,omitempty"`
	PodStatus    *string `json:"podStatus,omitempty"`
	RestartCount *int32  `json:"restartCount,omitempty"`
	PodIP        *string `json:"podIP,omitempty"`
	Node         *string `json:"node,omitempty"`
}

// ResourceInfo 资源信息
type ResourceInfo struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

// GetWorkloads 获取工作负载列表
// @Summary 获取工作负载列表
// @Description 获取所有类型的工作负载（Deployment、StatefulSet、DaemonSet、Job、CronJob、Pod）
// @Tags Kubernetes/工作负载
// @Accept json
// @Produce json
// @Security Bearer
// @Param clusterId query int true "集群ID"
// @Param namespace query string false "命名空间"
// @Param kind query string false "工作负载类型"
// @Success 200 {object} map[string]interface{} "工作负载列表"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/resources/workloads [get]
func (h *ResourceHandler) GetWorkloads(c *gin.Context) {
	// 获取参数
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	workloadType := c.Query("type")   // Deployment, StatefulSet, DaemonSet, Job, CronJob
	namespace := c.Query("namespace") // 命名空间过滤

	// 获取当前用户 ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	// 获取 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	fmt.Printf("📊 [GetWorkloads] 用户 %d 查询集群 %d 的工作负载列表, 类型: %s, 命名空间: %s\n",
		currentUserID, clusterID, workloadType, namespace)

	var workloads []WorkloadInfo
	ctx := c.Request.Context()

	// 根据类型查询不同的工作负载
	if workloadType == "" || workloadType == "Deployment" {
		// 获取 Deployments
		deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, deploy := range deployments.Items {
				workload := h.convertDeploymentToWorkload(&deploy)
				workloads = append(workloads, workload)
			}
		}
	}

	if workloadType == "" || workloadType == "StatefulSet" {
		// 获取 StatefulSets
		stsList, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, sts := range stsList.Items {
				workload := h.convertStatefulSetToWorkload(&sts)
				workloads = append(workloads, workload)
			}
		}
	}

	if workloadType == "" || workloadType == "DaemonSet" {
		// 获取 DaemonSets
		dsList, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, ds := range dsList.Items {
				workload := h.convertDaemonSetToWorkload(&ds)
				workloads = append(workloads, workload)
			}
		}
	}

	if workloadType == "" || workloadType == "Job" {
		// 获取 Jobs
		jobList, err := clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, job := range jobList.Items {
				workload := h.convertJobToWorkload(&job)
				workloads = append(workloads, workload)
			}
		}
	}

	if workloadType == "" || workloadType == "CronJob" {
		// 获取 CronJobs
		cronJobList, err := clientset.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, cronJob := range cronJobList.Items {
				workload := h.convertCronJobToWorkload(&cronJob)
				workloads = append(workloads, workload)
			}
		}
	}

	if workloadType == "" || workloadType == "Pod" {
		// 获取 Pods
		podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, pod := range podList.Items {
				workload := h.convertPodToWorkload(&pod)
				workloads = append(workloads, workload)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    workloads,
	})
}

// convertDeploymentToWorkload 将 Deployment 转换为 WorkloadInfo
func (h *ResourceHandler) convertDeploymentToWorkload(deploy *appsv1.Deployment) WorkloadInfo {
	// 计算 Pod 数量
	readyPods := deploy.Status.ReadyReplicas
	desiredPods := int32(0)
	if deploy.Spec.Replicas != nil {
		desiredPods = *deploy.Spec.Replicas
	}

	// 获取镜像和资源信息
	var images []string
	var requests, limits *ResourceInfo

	if len(deploy.Spec.Template.Spec.Containers) > 0 {
		for _, container := range deploy.Spec.Template.Spec.Containers {
			images = append(images, container.Image)
		}
		requests, limits = h.getResourceInfo(deploy.Spec.Template.Spec.Containers)
	}

	return WorkloadInfo{
		Name:        deploy.Name,
		Namespace:   deploy.Namespace,
		Type:        "Deployment",
		Labels:      deploy.Labels,
		ReadyPods:   readyPods,
		DesiredPods: desiredPods,
		Requests:    requests,
		Limits:      limits,
		Images:      images,
		CreatedAt:   deploy.CreationTimestamp.Format("2006-01-02 15:04:05"),
		UpdatedAt:   deploy.CreationTimestamp.Format("2006-01-02 15:04:05"),
	}
}

// convertStatefulSetToWorkload 将 StatefulSet 转换为 WorkloadInfo
func (h *ResourceHandler) convertStatefulSetToWorkload(sts *appsv1.StatefulSet) WorkloadInfo {
	readyPods := sts.Status.ReadyReplicas
	desiredPods := int32(0)
	if sts.Spec.Replicas != nil {
		desiredPods = *sts.Spec.Replicas
	}

	var images []string
	var requests, limits *ResourceInfo

	if len(sts.Spec.Template.Spec.Containers) > 0 {
		for _, container := range sts.Spec.Template.Spec.Containers {
			images = append(images, container.Image)
		}
		requests, limits = h.getResourceInfo(sts.Spec.Template.Spec.Containers)
	}

	return WorkloadInfo{
		Name:        sts.Name,
		Namespace:   sts.Namespace,
		Type:        "StatefulSet",
		Labels:      sts.Labels,
		ReadyPods:   readyPods,
		DesiredPods: desiredPods,
		Requests:    requests,
		Limits:      limits,
		Images:      images,
		CreatedAt:   sts.CreationTimestamp.Format("2006-01-02 15:04:05"),
		UpdatedAt:   sts.CreationTimestamp.Format("2006-01-02 15:04:05"),
	}
}

// convertDaemonSetToWorkload 将 DaemonSet 转换为 WorkloadInfo
func (h *ResourceHandler) convertDaemonSetToWorkload(ds *appsv1.DaemonSet) WorkloadInfo {
	readyPods := ds.Status.NumberReady
	desiredPods := ds.Status.DesiredNumberScheduled
	currentScheduled := ds.Status.CurrentNumberScheduled
	desiredScheduled := ds.Status.DesiredNumberScheduled

	var images []string
	var requests, limits *ResourceInfo

	if len(ds.Spec.Template.Spec.Containers) > 0 {
		for _, container := range ds.Spec.Template.Spec.Containers {
			images = append(images, container.Image)
		}
		requests, limits = h.getResourceInfo(ds.Spec.Template.Spec.Containers)
	}

	return WorkloadInfo{
		Name:             ds.Name,
		Namespace:        ds.Namespace,
		Type:             "DaemonSet",
		Labels:           ds.Labels,
		ReadyPods:        readyPods,
		DesiredPods:      desiredPods,
		Requests:         requests,
		Limits:           limits,
		Images:           images,
		CreatedAt:        ds.CreationTimestamp.Format("2006-01-02 15:04:05"),
		UpdatedAt:        ds.CreationTimestamp.Format("2006-01-02 15:04:05"),
		CurrentScheduled: &currentScheduled,
		DesiredScheduled: &desiredScheduled,
	}
}

// convertJobToWorkload 将 Job 转换为 WorkloadInfo
func (h *ResourceHandler) convertJobToWorkload(job *batchv1.Job) WorkloadInfo {
	readyPods := job.Status.Succeeded
	desiredPods := *job.Spec.Parallelism

	// 计算Job状态
	status := "Running"
	if job.Status.Succeeded > 0 && job.Status.Succeeded >= *job.Spec.Completions {
		status = "Succeeded"
	} else if job.Status.Failed > 0 {
		status = "Failed"
	} else if job.Status.Active > 0 {
		status = "Running"
	}

	// 计算耗时
	var duration *string
	if job.Status.StartTime != nil {
		if job.Status.CompletionTime != nil {
			// 已完成
			dur := job.Status.CompletionTime.Sub(job.Status.StartTime.Time)
			durStr := formatDuration(dur)
			duration = &durStr
		} else {
			// 进行中
			dur := time.Since(job.Status.StartTime.Time)
			durStr := formatDuration(dur)
			duration = &durStr
		}
	}

	var images []string
	var requests, limits *ResourceInfo

	if len(job.Spec.Template.Spec.Containers) > 0 {
		for _, container := range job.Spec.Template.Spec.Containers {
			images = append(images, container.Image)
		}
		requests, limits = h.getResourceInfo(job.Spec.Template.Spec.Containers)
	}

	return WorkloadInfo{
		Name:        job.Name,
		Namespace:   job.Namespace,
		Type:        "Job",
		Labels:      job.Labels,
		ReadyPods:   readyPods,
		DesiredPods: desiredPods,
		Requests:    requests,
		Limits:      limits,
		Images:      images,
		CreatedAt:   job.CreationTimestamp.Format("2006-01-02 15:04:05"),
		UpdatedAt:   job.CreationTimestamp.Format("2006-01-02 15:04:05"),
		Status:      &status,
		Duration:    duration,
	}
}

// convertCronJobToWorkload 将 CronJob 转换为 WorkloadInfo
func (h *ResourceHandler) convertCronJobToWorkload(cronJob *batchv1.CronJob) WorkloadInfo {
	var images []string
	var requests, limits *ResourceInfo

	if len(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers) > 0 {
		for _, container := range cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers {
			images = append(images, container.Image)
		}
		requests, limits = h.getResourceInfo(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers)
	}

	// 调度表达式
	schedule := cronJob.Spec.Schedule

	// 最后调度时间
	var lastScheduleTime *string
	if cronJob.Status.LastScheduleTime != nil {
		t := cronJob.Status.LastScheduleTime.Format("2006-01-02 15:04:05")
		lastScheduleTime = &t
	}

	// 暂停状态
	suspended := cronJob.Spec.Suspend != nil && *cronJob.Spec.Suspend

	return WorkloadInfo{
		Name:             cronJob.Name,
		Namespace:        cronJob.Namespace,
		Type:             "CronJob",
		Labels:           cronJob.Labels,
		ReadyPods:        0,
		DesiredPods:      0,
		Requests:         requests,
		Limits:           limits,
		Images:           images,
		CreatedAt:        cronJob.CreationTimestamp.Format("2006-01-02 15:04:05"),
		UpdatedAt:        cronJob.CreationTimestamp.Format("2006-01-02 15:04:05"),
		Schedule:         &schedule,
		LastScheduleTime: lastScheduleTime,
		Suspended:        &suspended,
	}
}

// convertPodToWorkload 将 Pod 转换为 WorkloadInfo
func (h *ResourceHandler) convertPodToWorkload(pod *v1.Pod) WorkloadInfo {
	// 计算 Pod 就绪状态
	readyPods := int32(0)
	if pod.Status.Phase == "Running" {
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				readyPods++
			}
		}
	}

	var images []string
	var requests, limits *ResourceInfo

	// 获取容器名称
	containerNames := make([]string, 0, len(pod.Spec.Containers))
	if len(pod.Spec.Containers) > 0 {
		for _, container := range pod.Spec.Containers {
			images = append(images, container.Image)
			containerNames = append(containerNames, container.Name)
		}
		requests, limits = h.getResourceInfo(pod.Spec.Containers)
	}

	// 容器信息
	containers := ""
	if len(containerNames) > 0 {
		containers = fmt.Sprintf("%d/%d", len(containerNames), len(pod.Spec.Containers))
	}

	// CPU/内存
	var cpu, memory *string
	if requests != nil {
		cpu = &requests.CPU
		memory = &requests.Memory
	} else if limits != nil {
		cpu = &limits.CPU
		memory = &limits.Memory
	}

	// Pod 状态 - 检查容器状态获取更详细的信息
	podStatus := string(pod.Status.Phase)

	// 如果Pod不是Running或Succeeded状态，检查容器状态获取详细原因
	if pod.Status.Phase != v1.PodRunning && pod.Status.Phase != v1.PodSucceeded {
		for _, cs := range pod.Status.ContainerStatuses {
			// 检查等待状态
			if cs.State.Waiting != nil {
				// 如果有具体的错误原因，使用该原因作为状态
				if cs.State.Waiting.Reason != "" {
					podStatus = cs.State.Waiting.Reason
					break
				}
			}
			// 检查终止状态
			if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
				podStatus = cs.State.Terminated.Reason
				break
			}
		}
	}

	// 重启次数
	var restartCount int32
	for _, cs := range pod.Status.ContainerStatuses {
		restartCount += cs.RestartCount
	}

	// Pod IP
	podIP := pod.Status.PodIP

	// 调度节点
	node := pod.Spec.NodeName

	return WorkloadInfo{
		Name:         pod.Name,
		Namespace:    pod.Namespace,
		Type:         "Pod",
		Labels:       pod.Labels,
		ReadyPods:    readyPods,
		DesiredPods:  1, // Pod 始终期望 1 个副本（自身）
		Requests:     requests,
		Limits:       limits,
		Images:       images,
		CreatedAt:    pod.CreationTimestamp.Format("2006-01-02 15:04:05"),
		UpdatedAt:    pod.CreationTimestamp.Format("2006-01-02 15:04:05"),
		Containers:   &containers,
		CPU:          cpu,
		Memory:       memory,
		PodStatus:    &podStatus,
		RestartCount: &restartCount,
		PodIP:        &podIP,
		Node:         &node,
	}
}

// getResourceInfo 获取容器的资源信息
func (h *ResourceHandler) getResourceInfo(containers []v1.Container) (requests, limits *ResourceInfo) {
	var totalCPUReq, totalMemReq int64
	var totalCPULim, totalMemLim int64

	for _, container := range containers {
		if container.Resources.Requests != nil {
			totalCPUReq += container.Resources.Requests.Cpu().MilliValue()
			totalMemReq += container.Resources.Requests.Memory().Value()
		}
		if container.Resources.Limits != nil {
			totalCPULim += container.Resources.Limits.Cpu().MilliValue()
			totalMemLim += container.Resources.Limits.Memory().Value()
		}
	}

	if totalCPUReq > 0 || totalMemReq > 0 {
		requests = &ResourceInfo{
			CPU:    formatCPU(totalCPUReq),
			Memory: formatMemory(totalMemReq),
		}
	}

	if totalCPULim > 0 || totalMemLim > 0 {
		limits = &ResourceInfo{
			CPU:    formatCPU(totalCPULim),
			Memory: formatMemory(totalMemLim),
		}
	}

	return requests, limits
}

// formatCPU 格式化 CPU
func formatCPU(milliValue int64) string {
	if milliValue == 0 {
		return ""
	}
	if milliValue < 1000 {
		return fmt.Sprintf("%dm", milliValue)
	}
	return fmt.Sprintf("%.2f", float64(milliValue)/1000)
}

// formatMemory 格式化内存
func formatMemory(bytes int64) string {
	if bytes == 0 {
		return ""
	}

	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2fTi", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2fGi", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2fMi", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2fKi", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d", bytes)
	}
}

// formatDuration 格式化时间间隔
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	} else {
		return fmt.Sprintf("%dd%dh", int(d.Hours())/24, int(d.Hours())%24)
	}
}

// GetWorkloadYAMLRequest 获取工作负载YAML请求
type GetWorkloadYAMLRequest struct {
	ClusterID int    `form:"clusterId" binding:"required"`
	Type      string `form:"type" binding:"required"` // Deployment, StatefulSet, DaemonSet, Job, CronJob
}

// GetWorkloadYAML 获取工作负载YAML
func (h *ResourceHandler) GetWorkloadYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	// 从 query 参数获取集群ID和类型
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	workloadType := c.Query("type")
	if workloadType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少工作负载类型参数",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	fmt.Printf("🔍 DEBUG [GetWorkloadYAML]: namespace=%s, name=%s, clusterID=%d, userID=%d, type=%s\n",
		namespace, name, clusterID, currentUserID, workloadType)

	// 获取clientset（修复参数顺序：clusterID 在前，userID 在后）
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 根据类型获取资源
	var obj interface{}
	switch workloadType {
	case "Deployment":
		deployment, err := clientset.AppsV1().Deployments(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取Deployment失败: " + err.Error(),
			})
			return
		}
		obj = deployment
	case "StatefulSet":
		statefulset, err := clientset.AppsV1().StatefulSets(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取StatefulSet失败: " + err.Error(),
			})
			return
		}
		obj = statefulset
	case "DaemonSet":
		daemonset, err := clientset.AppsV1().DaemonSets(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取DaemonSet失败: " + err.Error(),
			})
			return
		}
		obj = daemonset
	case "Job":
		job, err := clientset.BatchV1().Jobs(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取Job失败: " + err.Error(),
			})
			return
		}
		obj = job
	case "CronJob":
		cronjob, err := clientset.BatchV1().CronJobs(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取CronJob失败: " + err.Error(),
			})
			return
		}
		obj = cronjob
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "不支持的工作负载类型: " + workloadType,
		})
		return
	}

	// 清理对象（移除 managedFields 和 status 等不需要的字段）
	cleanedObj := cleanWorkloadForYAML(obj, workloadType)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"items": cleanedObj,
		},
	})
}

// GetWorkloadDetail 获取工作负载详情
func (h *ResourceHandler) GetWorkloadDetail(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	workloadType := c.Query("type")
	if workloadType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少工作负载类型参数",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败",
		})
		return
	}

	var workload interface{}
	switch workloadType {
	case "Deployment":
		deployment, err := clientset.AppsV1().Deployments(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
		if err != nil {
			HandleK8sError(c, err, "Deployment")
			return
		}
		workload = deployment
	case "StatefulSet":
		sts, err := clientset.AppsV1().StatefulSets(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
		if err != nil {
			HandleK8sError(c, err, "StatefulSet")
			return
		}
		workload = sts
	case "DaemonSet":
		ds, err := clientset.AppsV1().DaemonSets(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
		if err != nil {
			HandleK8sError(c, err, "DaemonSet")
			return
		}
		workload = ds
	case "Job":
		job, err := clientset.BatchV1().Jobs(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
		if err != nil {
			HandleK8sError(c, err, "Job")
			return
		}
		workload = job
	case "CronJob":
		cronjob, err := clientset.BatchV1().CronJobs(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
		if err != nil {
			HandleK8sError(c, err, "CronJob")
			return
		}
		workload = cronjob
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "不支持的工作负载类型",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"items": []interface{}{workload},
		},
	})
}

// GetWorkloadReplicaSets 获取工作负载的ReplicaSet列表
func (h *ResourceHandler) GetWorkloadReplicaSets(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败",
		})
		return
	}

	// 获取该工作负载的所有ReplicaSet
	labelSelector := fmt.Sprintf("app=%s", name)
	replicaSets, err := clientset.AppsV1().ReplicaSets(namespace).List(c.Request.Context(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		HandleK8sError(c, err, "ReplicaSet")
		return
	}

	// 转换为通用格式
	var items []interface{}
	for _, rs := range replicaSets.Items {
		items = append(items, rs)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"items": items,
		},
	})
}

// GetWorkloadPods 获取工作负载的Pod列表
func (h *ResourceHandler) GetWorkloadPods(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	workloadType := c.Query("type")

	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败",
		})
		return
	}

	// 获取工作负载的标签选择器
	var labelSelector string

	switch workloadType {
	case "Deployment":
		deployment, err := clientset.AppsV1().Deployments(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
		if err != nil {
			HandleK8sError(c, err, "Deployment")
			return
		}
		// 构建 label selector
		var selectors []string
		for k, v := range deployment.Spec.Selector.MatchLabels {
			selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
		}
		labelSelector = strings.Join(selectors, ",")

	case "StatefulSet":
		sts, err := clientset.AppsV1().StatefulSets(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
		if err != nil {
			HandleK8sError(c, err, "StatefulSet")
			return
		}
		var selectors []string
		for k, v := range sts.Spec.Selector.MatchLabels {
			selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
		}
		labelSelector = strings.Join(selectors, ",")

	case "DaemonSet":
		ds, err := clientset.AppsV1().DaemonSets(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
		if err != nil {
			HandleK8sError(c, err, "DaemonSet")
			return
		}
		var selectors []string
		for k, v := range ds.Spec.Selector.MatchLabels {
			selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
		}
		labelSelector = strings.Join(selectors, ",")

	default:
		// 默认使用 app=<name>
		labelSelector = fmt.Sprintf("app=%s", name)
	}

	// 获取该工作负载的所有Pod
	pods, err := clientset.CoreV1().Pods(namespace).List(c.Request.Context(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		HandleK8sError(c, err, "Pod")
		return
	}

	// 转换为通用格式
	var items []interface{}
	for _, pod := range pods.Items {
		items = append(items, pod)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"items": items,
		},
	})
}

// GetWorkloadServices 获取工作负载关联的Service列表
func (h *ResourceHandler) GetWorkloadServices(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败",
		})
		return
	}

	// 获取所有Service
	services, err := clientset.CoreV1().Services(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "Service")
		return
	}

	// 过滤出与该工作负载关联的Service（通过selector匹配）
	var items []interface{}
	for _, svc := range services.Items {
		if svc.Spec.Selector != nil {
			if appName, exists := svc.Spec.Selector["app"]; exists && appName == name {
				items = append(items, svc)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"items": items,
		},
	})
}

// GetWorkloadIngresses 获取工作负载关联的Ingress列表
func (h *ResourceHandler) GetWorkloadIngresses(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败",
		})
		return
	}

	// 获取所有Ingress
	ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "Ingress")
		return
	}

	// 过滤出与该工作负载关联的Ingress（通过service.name匹配）
	var items []interface{}
	for _, ing := range ingresses.Items {
		for _, rule := range ing.Spec.Rules {
			if rule.HTTP != nil {
				for _, path := range rule.HTTP.Paths {
					serviceName := path.Backend.Service.Name
					if serviceName == name {
						items = append(items, ing)
						break
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"items": items,
		},
	})
}

// UpdateWorkloadYAMLRequest 更新工作负载YAML请求
type UpdateWorkloadYAMLRequest struct {
	ClusterID int    `json:"clusterId" binding:"required"`
	Type      string `json:"type" binding:"required"`
	YAML      string `json:"yaml" binding:"required"`
}

// UpdateWorkloadYAML 更新工作负载YAML
func (h *ResourceHandler) UpdateWorkloadYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	var req UpdateWorkloadYAMLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	fmt.Printf("🔍 DEBUG [UpdateWorkloadYAML]: namespace=%s, name=%s, clusterID=%d, userID=%d, type=%s\n",
		namespace, name, req.ClusterID, currentUserID, req.Type)

	// 获取clientset（修复参数顺序：clusterID 在前，userID 在后）
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 解析YAML
	var yamlData map[string]interface{}
	if err := yamlUnmarshal([]byte(req.YAML), &yamlData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析YAML失败: " + err.Error(),
		})
		return
	}

	// 验证资源名称
	if metadata, ok := yamlData["metadata"].(map[string]interface{}); ok {
		if yamlName, ok := metadata["name"].(string); ok && yamlName != name {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "YAML中的资源名称与URL中的不一致",
			})
			return
		}
		if yamlNamespace, ok := metadata["namespace"].(string); ok && yamlNamespace != namespace {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "YAML中的命名空间与URL中的不一致",
			})
			return
		}
		// 清除不可变字段，避免更新冲突
		delete(metadata, "uid")
		delete(metadata, "selfLink")
		delete(metadata, "creationTimestamp")
		delete(metadata, "deletionTimestamp")
		delete(metadata, "deletionGracePeriodSeconds")
		delete(metadata, "generation")
		delete(metadata, "resourceVersion")
	}

	// 转换为JSON用于PATCH
	patchData, err := json.Marshal(yamlData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "序列化Patch数据失败: " + err.Error(),
		})
		return
	}

	// 根据类型更新资源
	switch req.Type {
	case "Deployment":
		_, err := clientset.AppsV1().Deployments(namespace).Patch(c.Request.Context(), name, types.MergePatchType, patchData, metav1.PatchOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新Deployment失败: " + err.Error(),
			})
			return
		}
	case "StatefulSet":
		_, err := clientset.AppsV1().StatefulSets(namespace).Patch(c.Request.Context(), name, types.MergePatchType, patchData, metav1.PatchOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新StatefulSet失败: " + err.Error(),
			})
			return
		}
	case "DaemonSet":
		_, err := clientset.AppsV1().DaemonSets(namespace).Patch(c.Request.Context(), name, types.MergePatchType, patchData, metav1.PatchOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新DaemonSet失败: " + err.Error(),
			})
			return
		}
	case "Job":
		_, err := clientset.BatchV1().Jobs(namespace).Patch(c.Request.Context(), name, types.MergePatchType, patchData, metav1.PatchOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新Job失败: " + err.Error(),
			})
			return
		}
	case "CronJob":
		_, err := clientset.BatchV1().CronJobs(namespace).Patch(c.Request.Context(), name, types.MergePatchType, patchData, metav1.PatchOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新CronJob失败: " + err.Error(),
			})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "不支持的工作负载类型: " + req.Type,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data": gin.H{
			"needRefresh": true, // 告诉前端需要刷新列表
		},
	})
}

// UpdateWorkloadRequest 更新工作负载请求（所有参数在请求体中）
// UpdateWorkloadRequest 直接接收 Kubernetes 对象
// 从 URL 参数获取 cluster、namespace、type、name，请求体直接是 Kubernetes 对象
type UpdateWorkloadRequest struct {
	// 这些字段从 URL 参数获取，不在请求体中
	Cluster   string
	Namespace string
	Type      string
	Name      string
	// WorkloadData 是请求体，直接是 Kubernetes 对象
	WorkloadData map[string]interface{}
}

// UpdateWorkload 更新工作负载
// URL 参数: cluster, namespace, type, name
// 请求体: 直接是 Kubernetes 对象（Deployment、StatefulSet 等）
func (h *ResourceHandler) UpdateWorkload(c *gin.Context) {
	// 从 URL 参数获取基本信息
	cluster := c.Query("cluster")
	namespace := c.Query("namespace")
	workloadType := c.Query("type")
	name := c.Query("name")

	if cluster == "" || namespace == "" || workloadType == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少必要参数: cluster, namespace, type, name",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	fmt.Printf("🔍 DEBUG [UpdateWorkload]: cluster=%s, namespace=%s, name=%s, type=%s, userID=%d\n",
		cluster, namespace, name, workloadType, currentUserID)

	// 根据集群名称获取集群ID（查询数据库）
	var clusterID int
	err := h.db.Raw("SELECT id FROM k8s_clusters WHERE name = ? AND created_by = ? LIMIT 1", cluster, currentUserID).Scan(&clusterID).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群信息失败: " + err.Error(),
		})
		return
	}

	// 获取clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 直接从请求体读取 Kubernetes 对象
	var yamlData map[string]interface{}
	if err := c.ShouldBindJSON(&yamlData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析请求体失败: " + err.Error(),
		})
		return
	}

	// 验证资源名称
	if metadata, ok := yamlData["metadata"].(map[string]interface{}); ok {
		if yamlName, ok := metadata["name"].(string); ok && yamlName != name {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "YAML中的资源名称与URL参数中的不一致",
			})
			return
		}
		if yamlNamespace, ok := metadata["namespace"].(string); ok && yamlNamespace != namespace {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "YAML中的命名空间与请求中的不一致",
			})
			return
		}
	}

	// 转换为JSON用于PATCH
	patchData, err := json.Marshal(yamlData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "序列化Patch数据失败: " + err.Error(),
		})
		return
	}

	// 根据类型更新资源
	// 使用 MergePatchType 而不是 MergePatchType，以支持通过 null 删除字段
	switch workloadType {
	case "Deployment":
		_, err := clientset.AppsV1().Deployments(namespace).Patch(c.Request.Context(), name, types.MergePatchType, patchData, metav1.PatchOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新Deployment失败: " + err.Error(),
			})
			return
		}
	case "StatefulSet":
		_, err := clientset.AppsV1().StatefulSets(namespace).Patch(c.Request.Context(), name, types.MergePatchType, patchData, metav1.PatchOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新StatefulSet失败: " + err.Error(),
			})
			return
		}
	case "DaemonSet":
		_, err := clientset.AppsV1().DaemonSets(namespace).Patch(c.Request.Context(), name, types.MergePatchType, patchData, metav1.PatchOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新DaemonSet失败: " + err.Error(),
			})
			return
		}
	case "Job":
		_, err := clientset.BatchV1().Jobs(namespace).Patch(c.Request.Context(), name, types.MergePatchType, patchData, metav1.PatchOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新Job失败: " + err.Error(),
			})
			return
		}
	case "CronJob":
		_, err := clientset.BatchV1().CronJobs(namespace).Patch(c.Request.Context(), name, types.MergePatchType, patchData, metav1.PatchOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新CronJob失败: " + err.Error(),
			})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "不支持的工作负载类型: " + workloadType,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data": gin.H{
			"needRefresh": true, // 告诉前端需要刷新列表
		},
	})
}

// CreateWorkloadFromYAML 从 YAML 创建工作负载
func (h *ResourceHandler) CreateWorkloadFromYAML(c *gin.Context) {
	var req struct {
		ClusterID uint   `json:"clusterId" binding:"required"`
		YAML      string `json:"yaml" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	fmt.Printf("🎯 CreateWorkloadFromYAML: clusterID=%d, userID=%d\n", req.ClusterID, currentUserID)

	// 解析 YAML
	yamlDecoder := k8syaml.NewYAMLOrJSONDecoder(strings.NewReader(req.YAML), 4096)
	var yamlObj map[string]interface{}
	if err := yamlDecoder.Decode(&yamlObj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "解析 YAML 失败: " + err.Error()})
		return
	}

	// 获取资源类型和元数据
	kind, _ := yamlObj["kind"].(string)
	metadata, _ := yamlObj["metadata"].(map[string]interface{})
	namespace, _ := metadata["namespace"].(string)
	name, _ := metadata["name"].(string)

	if kind == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "YAML 缺少必要字段 kind 或 metadata.name"})
		return
	}

	// 默认命名空间为 default
	if namespace == "" {
		namespace = "default"
		metadata["namespace"] = namespace
	}

	// 获取 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), req.ClusterID, currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取集群客户端失败: " + err.Error()})
		return
	}

	// 转换为 JSON
	jsonData, err := json.Marshal(yamlObj)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "序列化数据失败: " + err.Error()})
		return
	}

	// 根据类型创建资源
	switch kind {
	case "Deployment":
		var deployment appsv1.Deployment
		if err := json.Unmarshal(jsonData, &deployment); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "解析 Deployment 失败: " + err.Error()})
			return
		}
		_, err = clientset.AppsV1().Deployments(namespace).Create(c.Request.Context(), &deployment, metav1.CreateOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建 Deployment 失败: " + err.Error()})
			return
		}

	case "StatefulSet":
		var sts appsv1.StatefulSet
		if err := json.Unmarshal(jsonData, &sts); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "解析 StatefulSet 失败: " + err.Error()})
			return
		}
		_, err = clientset.AppsV1().StatefulSets(namespace).Create(c.Request.Context(), &sts, metav1.CreateOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建 StatefulSet 失败: " + err.Error()})
			return
		}

	case "DaemonSet":
		var ds appsv1.DaemonSet
		if err := json.Unmarshal(jsonData, &ds); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "解析 DaemonSet 失败: " + err.Error()})
			return
		}
		_, err = clientset.AppsV1().DaemonSets(namespace).Create(c.Request.Context(), &ds, metav1.CreateOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建 DaemonSet 失败: " + err.Error()})
			return
		}

	case "Job":
		var job batchv1.Job
		if err := json.Unmarshal(jsonData, &job); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "解析 Job 失败: " + err.Error()})
			return
		}
		_, err = clientset.BatchV1().Jobs(namespace).Create(c.Request.Context(), &job, metav1.CreateOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建 Job 失败: " + err.Error()})
			return
		}

	case "CronJob":
		var cronJob batchv1.CronJob
		if err := json.Unmarshal(jsonData, &cronJob); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "解析 CronJob 失败: " + err.Error()})
			return
		}
		_, err = clientset.BatchV1().CronJobs(namespace).Create(c.Request.Context(), &cronJob, metav1.CreateOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建 CronJob 失败: " + err.Error()})
			return
		}

	case "Pod":
		var pod v1.Pod
		if err := json.Unmarshal(jsonData, &pod); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "解析 Pod 失败: " + err.Error()})
			return
		}
		_, err = clientset.CoreV1().Pods(namespace).Create(c.Request.Context(), &pod, metav1.CreateOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建 Pod 失败: " + err.Error()})
			return
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不支持的工作负载类型: " + kind})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"kind":      kind,
			"namespace": namespace,
			"name":      name,
		},
	})
}

// DeleteWorkload 删除工作负载
func (h *ResourceHandler) DeleteWorkload(c *gin.Context) {
	// 从URL参数获取基本信息
	namespace := c.Param("namespace")
	name := c.Param("name")
	clusterID := c.Query("clusterId")
	workloadType := c.Query("type")

	if namespace == "" || name == "" || clusterID == "" || workloadType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少必要参数: namespace, name, clusterId, type",
		})
		return
	}

	// 转换clusterID
	clusterIDUint, err := strconv.ParseUint(clusterID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的clusterId",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	fmt.Printf("🗑️ DeleteWorkload: namespace=%s, name=%s, type=%s, clusterID=%s, userID=%d\n",
		namespace, name, workloadType, clusterID, currentUserID)

	// 获取clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterIDUint), currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 根据类型删除资源
	switch workloadType {
	case "Deployment":
		err = clientset.AppsV1().Deployments(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "删除Deployment失败: " + err.Error(),
			})
			return
		}

	case "StatefulSet":
		err = clientset.AppsV1().StatefulSets(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "删除StatefulSet失败: " + err.Error(),
			})
			return
		}

	case "DaemonSet":
		err = clientset.AppsV1().DaemonSets(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "删除DaemonSet失败: " + err.Error(),
			})
			return
		}

	case "Job":
		err = clientset.BatchV1().Jobs(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "删除Job失败: " + err.Error(),
			})
			return
		}

	case "CronJob":
		err = clientset.BatchV1().CronJobs(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "删除CronJob失败: " + err.Error(),
			})
			return
		}

	case "Pod":
		err = clientset.CoreV1().Pods(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "删除Pod失败: " + err.Error(),
			})
			return
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "不支持的工作负载类型: " + workloadType,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data": gin.H{
			"namespace": namespace,
			"name":      name,
			"type":      workloadType,
		},
	})
}

// cleanWorkloadForYAML 清理工作负载对象用于YAML输出
func cleanWorkloadForYAML(obj interface{}, workloadType string) map[string]interface{} {
	// 转换为 map 以便控制 YAML 序列化
	result := make(map[string]interface{})

	// 根据不同的工作负载类型设置 apiVersion 和 kind
	switch workloadType {
	case "Deployment":
		result["apiVersion"] = "apps/v1"
		result["kind"] = "Deployment"
		if deploy, ok := obj.(*appsv1.Deployment); ok {
			result["metadata"] = cleanMetadata(deploy.ObjectMeta)
			result["spec"] = deploy.Spec
		}
	case "StatefulSet":
		result["apiVersion"] = "apps/v1"
		result["kind"] = "StatefulSet"
		if sts, ok := obj.(*appsv1.StatefulSet); ok {
			result["metadata"] = cleanMetadata(sts.ObjectMeta)
			result["spec"] = sts.Spec
		}
	case "DaemonSet":
		result["apiVersion"] = "apps/v1"
		result["kind"] = "DaemonSet"
		if ds, ok := obj.(*appsv1.DaemonSet); ok {
			result["metadata"] = cleanMetadata(ds.ObjectMeta)
			result["spec"] = ds.Spec
		}
	case "Job":
		result["apiVersion"] = "batch/v1"
		result["kind"] = "Job"
		if job, ok := obj.(*batchv1.Job); ok {
			result["metadata"] = cleanMetadata(job.ObjectMeta)
			result["spec"] = job.Spec
		}
	case "CronJob":
		result["apiVersion"] = "batch/v1"
		result["kind"] = "CronJob"
		if cronJob, ok := obj.(*batchv1.CronJob); ok {
			result["metadata"] = cleanMetadata(cronJob.ObjectMeta)
			result["spec"] = cronJob.Spec
		}
	}

	// 不包含 status 字段

	return result
}

// cleanMetadata 清理 metadata 字段
func cleanMetadata(meta metav1.ObjectMeta) map[string]interface{} {
	metadata := make(map[string]interface{})

	if meta.Name != "" {
		metadata["name"] = meta.Name
	}
	if meta.Namespace != "" {
		metadata["namespace"] = meta.Namespace
	}
	if len(meta.Labels) > 0 {
		metadata["labels"] = meta.Labels
	}
	if len(meta.Annotations) > 0 {
		metadata["annotations"] = meta.Annotations
	}
	// 不包含 managedFields、resourceVersion、uid、generation 等字段

	return metadata
}

// ==================== Service 相关 ====================

// ServiceInfo 服务信息
type ServiceInfo struct {
	Name            string            `json:"name"`            // 服务名称
	Namespace       string            `json:"namespace"`       // 命名空间
	Type            string            `json:"type"`            // 服务类型: ClusterIP, NodePort, LoadBalancer, ExternalName
	ClusterIP       string            `json:"clusterIP"`       // Cluster IP 地址
	ExternalIP      string            `json:"externalIP"`      // External IP 地址
	Ports           []ServicePortInfo `json:"ports"`           // 端口列表
	Selector        map[string]string `json:"selector"`        // Pod 选择器
	SessionAffinity string            `json:"sessionAffinity"` // 会话亲和性
	Age             string            `json:"age"`             // 创建时间
	Labels          map[string]string `json:"labels"`          // 标签
	Endpoints       int               `json:"endpoints"`       // 端点数量
}

// ServicePortInfo 服务端口信息
type ServicePortInfo struct {
	Name       string `json:"name"`       // 端口名称
	Protocol   string `json:"protocol"`   // 协议: TCP, UDP, SCTP
	Port       int32  `json:"port"`       // 服务端口
	TargetPort string `json:"targetPort"` // 目标端口
	NodePort   int32  `json:"nodePort"`   // NodePort (仅 NodePort 类型)
}

// ListServices 获取服务列表
// @Summary 获取Service列表
// @Description 获取指定命名空间的 Service 列表
// @Tags Kubernetes/网络
// @Accept json
// @Produce json
// @Security Bearer
// @Param clusterId query int true "集群ID"
// @Param namespace query string false "命名空间"
// @Success 200 {object} map[string]interface{} "Service列表"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/resources/services [get]
func (h *ResourceHandler) ListServices(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = v1.NamespaceAll
	}

	// 获取当前用户 ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	// 使用用户凭据获取 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	services, err := clientset.CoreV1().Services(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "服务")
		return
	}

	// 获取 Endpoints 用于统计端点数量
	endpointsMap := make(map[string]int)
	endpoints, err := clientset.CoreV1().Endpoints(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err == nil {
		for _, ep := range endpoints.Items {
			readyCount := 0
			for _, subset := range ep.Subsets {
				readyCount += len(subset.Addresses)
			}
			endpointsMap[ep.Name] = readyCount
		}
	}

	serviceInfos := make([]ServiceInfo, 0, len(services.Items))
	for _, svc := range services.Items {
		// 确保 labels 不为 nil
		labels := svc.Labels
		if labels == nil {
			labels = make(map[string]string)
		}

		// 确保 selector 不为 nil
		selector := svc.Spec.Selector
		if selector == nil {
			selector = make(map[string]string)
		}

		// 获取 External IP
		externalIP := ""
		if len(svc.Spec.ExternalIPs) > 0 {
			externalIP = svc.Spec.ExternalIPs[0]
		} else if svc.Spec.Type == v1.ServiceTypeLoadBalancer && len(svc.Status.LoadBalancer.Ingress) > 0 {
			if svc.Status.LoadBalancer.Ingress[0].IP != "" {
				externalIP = svc.Status.LoadBalancer.Ingress[0].IP
			} else if svc.Status.LoadBalancer.Ingress[0].Hostname != "" {
				externalIP = svc.Status.LoadBalancer.Ingress[0].Hostname
			}
		}

		// 转换端口信息
		portInfos := make([]ServicePortInfo, 0, len(svc.Spec.Ports))
		for _, port := range svc.Spec.Ports {
			targetPort := ""
			if port.TargetPort.Type == intstr.Int {
				targetPort = strconv.Itoa(int(port.TargetPort.IntVal))
			} else {
				targetPort = port.TargetPort.StrVal
			}

			portInfo := ServicePortInfo{
				Name:       port.Name,
				Protocol:   string(port.Protocol),
				Port:       port.Port,
				TargetPort: targetPort,
				NodePort:   port.NodePort,
			}
			portInfos = append(portInfos, portInfo)
		}

		serviceInfo := ServiceInfo{
			Name:            svc.Name,
			Namespace:       svc.Namespace,
			Type:            string(svc.Spec.Type),
			ClusterIP:       svc.Spec.ClusterIP,
			ExternalIP:      externalIP,
			Ports:           portInfos,
			Selector:        selector,
			SessionAffinity: string(svc.Spec.SessionAffinity),
			Age:             calculateAge(svc.CreationTimestamp.Time),
			Labels:          labels,
			Endpoints:       endpointsMap[svc.Name],
		}

		serviceInfos = append(serviceInfos, serviceInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    serviceInfos,
	})
}

// GetServiceYAML 获取服务 YAML
func (h *ResourceHandler) GetServiceYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	svc, err := clientset.CoreV1().Services(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "服务")
		return
	}

	// 清理对象用于 YAML 输出
	cleanedSvc := cleanServiceForYAML(svc)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"items": cleanedSvc,
		},
	})
}

// UpdateServiceYAMLRequest 更新服务 YAML 请求
// UpdateServiceYAMLRequest 更新服务请求
type UpdateServiceYAMLRequest struct {
	ClusterID int                    `json:"clusterId" binding:"required"`
	Data      map[string]interface{} `json:"-" binding:"required"`
}

// UpdateServiceYAML 更新服务 YAML
func (h *ResourceHandler) UpdateServiceYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	// 直接绑定请求体到 map[string]interface{}
	var jsonData map[string]interface{}
	if err := c.ShouldBindJSON(&jsonData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	fmt.Printf("🔍 [UpdateServiceYAML] 收到请求 - namespace=%s, name=%s\n", namespace, name)
	fmt.Printf("🔍 [UpdateServiceYAML] 原始请求数据: %+v\n", jsonData)

	// 提取 clusterId
	clusterIDFloat, ok := jsonData["clusterId"].(float64)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少 clusterId 字段",
		})
		return
	}
	clusterID := int(clusterIDFloat)

	// 删除 clusterId 字段，剩余的就是 Kubernetes 资源数据
	delete(jsonData, "clusterId")

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 获取现有 Service 以比对
	existingService, err := clientset.CoreV1().Services(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("❌ [UpdateServiceYAML] 获取现有 Service 失败: %v\n", err)
		HandleK8sError(c, err, "服务")
		return
	}

	fmt.Printf("🔍 [UpdateServiceYAML] 现有 Service 类型: %s, ClusterIP: %s\n", existingService.Spec.Type, existingService.Spec.ClusterIP)

	// 验证资源名称
	if metadata, ok := jsonData["metadata"].(map[string]interface{}); ok {
		if jsonName, ok := metadata["name"].(string); ok && jsonName != name {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "资源名称与URL中的不一致",
			})
			return
		}
		if jsonNamespace, ok := metadata["namespace"].(string); ok && jsonNamespace != namespace {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "命名空间与URL中的不一致",
			})
			return
		}
		// 清除不可变字段，避免更新冲突
		delete(metadata, "uid")
		delete(metadata, "selfLink")
		delete(metadata, "creationTimestamp")
		delete(metadata, "deletionTimestamp")
		delete(metadata, "deletionGracePeriodSeconds")
		delete(metadata, "generation")
		delete(metadata, "resourceVersion")
		delete(metadata, "managedFields")
	}

	// 清理 spec 中的字段
	if spec, ok := jsonData["spec"].(map[string]interface{}); ok {
		// 获取新的 Service 类型
		newType := spec["type"]
		if newType == nil {
			newType = string(existingService.Spec.Type)
		}

		fmt.Printf("🔍 [UpdateServiceYAML] 新的 Service 类型: %v\n", newType)

		// Service 类型特定的字段清理
		switch newType {
		case "ClusterIP":
			// ClusterIP类型：删除NodePort/LoadBalancer特有字段
			if ports, ok := spec["ports"].([]interface{}); ok {
				for _, port := range ports {
					if portMap, ok := port.(map[string]interface{}); ok {
						delete(portMap, "nodePort")
					}
				}
			}
			delete(spec, "externalTrafficPolicy")
			delete(spec, "healthCheckNodePort")
			delete(spec, "allocateLoadBalancerNodePorts")
			delete(spec, "loadBalancerSourceRanges")
			delete(spec, "loadBalancerIP")
			delete(spec, "loadBalancerClass")
		case "NodePort":
			// NodePort类型：删除LoadBalancer特有字段
			delete(spec, "healthCheckNodePort")
			delete(spec, "allocateLoadBalancerNodePorts")
			delete(spec, "loadBalancerSourceRanges")
			delete(spec, "loadBalancerIP")
			delete(spec, "loadBalancerClass")
		case "LoadBalancer":
			// LoadBalancer 类型保留大部分字段
		case "ExternalName":
			// ExternalName类型：删除ClusterIP相关字段
			delete(spec, "clusterIP")
			delete(spec, "clusterIPs")
			delete(spec, "ports")
			delete(spec, "selector")
			delete(spec, "externalTrafficPolicy")
			delete(spec, "healthCheckNodePort")
			delete(spec, "allocateLoadBalancerNodePorts")
			delete(spec, "loadBalancerSourceRanges")
		}

		// 清除状态相关的只读字段
		delete(spec, "loadBalancerIP") // 已废弃
		delete(spec, "sessionAffinityConfig") // 如果为空则删除
	}

	// 删除 status 字段（只读）
	delete(jsonData, "status")

	fmt.Printf("🔍 [UpdateServiceYAML] 清理后的数据: %+v\n", jsonData)

	// 转换为 JSON 用于 PATCH
	patchData, err := json.Marshal(jsonData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "序列化Patch数据失败: " + err.Error(),
		})
		return
	}

	fmt.Printf("🔍 [UpdateServiceYAML] Patch 数据: %s\n", string(patchData))

	// 使用 Patch 方法更新，避免不可变字段冲突
	updatedService, err := clientset.CoreV1().Services(namespace).Patch(c.Request.Context(), name, types.MergePatchType, patchData, metav1.PatchOptions{})
	if err != nil {
		fmt.Printf("❌ [UpdateServiceYAML] Patch 失败: %v\n", err)
		HandleK8sError(c, err, "服务")
		return
	}

	fmt.Printf("✅ [UpdateServiceYAML] 更新成功 - 新类型: %s, ClusterIP: %s\n", updatedService.Spec.Type, updatedService.Spec.ClusterIP)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// CreateServiceRequest 创建服务请求
type CreateServiceRequest struct {
	ClusterID       int                     `json:"clusterId" binding:"required"`
	Namespace       string                  `json:"namespace" binding:"required"`
	Name            string                  `json:"name" binding:"required"`
	Type            string                  `json:"type" binding:"required"`
	ClusterIP       string                  `json:"clusterIP"`
	Ports           []ServicePortCreateInfo `json:"ports" binding:"required"`
	Selector        map[string]string       `json:"selector"`
	SessionAffinity string                  `json:"sessionAffinity"`
}

// ServicePortCreateInfo 端口创建信息
type ServicePortCreateInfo struct {
	Name       string `json:"name"`
	Protocol   string `json:"protocol" binding:"required"`
	Port       int32  `json:"port" binding:"required"`
	TargetPort string `json:"targetPort"`
	NodePort   int32  `json:"nodePort"`
}

// CreateService 创建服务
func (h *ResourceHandler) CreateService(c *gin.Context) {
	namespace := c.Param("namespace")

	var req CreateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 构建 Service 对象
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: namespace,
		},
		Spec: v1.ServiceSpec{
			Type:            v1.ServiceType(req.Type),
			ClusterIP:       req.ClusterIP,
			Selector:        req.Selector,
			SessionAffinity: v1.ServiceAffinity(req.SessionAffinity),
		},
	}

	// 如果 ClusterIP 为 "None"，设置为 Headless Service
	if req.Type == "ClusterIP" && req.ClusterIP == "None" {
		svc.Spec.ClusterIP = "None"
	}

	// 转换端口信息
	for _, port := range req.Ports {
		servicePort := v1.ServicePort{
			Name:     port.Name,
			Protocol: v1.Protocol(port.Protocol),
			Port:     port.Port,
		}

		// 解析 TargetPort
		if port.TargetPort != "" {
			// 尝试解析为数字
			if portNum, err := strconv.Atoi(port.TargetPort); err == nil {
				servicePort.TargetPort = intstr.FromInt(portNum)
			} else {
				servicePort.TargetPort = intstr.FromString(port.TargetPort)
			}
		} else {
			// 默认使用 Port
			servicePort.TargetPort = intstr.FromInt(int(port.Port))
		}

		// NodePort 只有 NodePort 类型才需要设置
		if req.Type == "NodePort" && port.NodePort > 0 {
			servicePort.NodePort = port.NodePort
		}

		svc.Spec.Ports = append(svc.Spec.Ports, servicePort)
	}

	// 创建 Service
	_, err = clientset.CoreV1().Services(namespace).Create(c.Request.Context(), svc, metav1.CreateOptions{})
	if err != nil {
		HandleK8sError(c, err, "服务")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// DeleteServiceRequest 删除服务请求
type DeleteServiceRequest struct {
	ClusterID int `json:"clusterId" binding:"required"`
}

// CreateEndpointRequest 创建端点请求
type CreateEndpointRequest struct {
	ClusterID int                    `json:"clusterId" binding:"required"`
	Data      map[string]interface{} `json:"-" binding:"required"`
}

// UpdateEndpointRequest 更新端点请求
type UpdateEndpointRequest struct {
	ClusterID int                    `json:"clusterId" binding:"required"`
	Data      map[string]interface{} `json:"-" binding:"required"`
}

// DeleteService 删除服务
func (h *ResourceHandler) DeleteService(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	var req DeleteServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	err = clientset.CoreV1().Services(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		HandleK8sError(c, err, "服务")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// cleanServiceForYAML 清理 Service 对象用于 YAML 输出
func cleanServiceForYAML(svc *v1.Service) map[string]interface{} {
	result := make(map[string]interface{})
	result["apiVersion"] = "v1"
	result["kind"] = "Service"
	result["metadata"] = cleanMetadata(svc.ObjectMeta)
	result["spec"] = svc.Spec
	return result
}

// ==================== Ingress 相关 ====================

// IngressInfo Ingress 信息
type IngressInfo struct {
	Name         string            `json:"name"`         // Ingress 名称
	Namespace    string            `json:"namespace"`    // 命名空间
	Hosts        []string          `json:"hosts"`        // 主机名列表
	Paths        []IngressPathInfo `json:"paths"`        // 路径列表
	TLS          []IngressTLSInfo  `json:"tls"`          // TLS 配置
	IngressClass string            `json:"ingressClass"` // Ingress Class
	Age          string            `json:"age"`          // 创建时间
	Labels       map[string]string `json:"labels"`       // 标签
	Addresses    []string          `json:"addresses"`    // IP 地址列表
}

// IngressPathInfo Ingress 路径信息
type IngressPathInfo struct {
	Host     string `json:"host"`     // 主机名
	Path     string `json:"path"`     // 路径
	PathType string `json:"pathType"` // 路径类型
	Service  string `json:"service"`  // 服务名称
	Port     int32  `json:"port"`     // 服务端口
}

// IngressTLSInfo TLS 配置
type IngressTLSInfo struct {
	Hosts      []string `json:"hosts"`      // 主机名列表
	SecretName string   `json:"secretName"` // Secret 名称
}

// ListIngresses 获取 Ingress 列表
// @Summary 获取Ingress列表
// @Description 获取指定命名空间的 Ingress 列表
// @Tags Kubernetes/网络
// @Accept json
// @Produce json
// @Security Bearer
// @Param clusterId query int true "集群ID"
// @Param namespace query string false "命名空间"
// @Success 200 {object} map[string]interface{} "Ingress列表"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/resources/ingresses [get]
func (h *ResourceHandler) ListIngresses(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = v1.NamespaceAll
	}

	// 获取当前用户 ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	// 使用用户凭据获取 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	// 获取 Ingress（使用 networking.k8s.io/v1）
	ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "Ingress")
		return
	}

	ingressInfos := make([]IngressInfo, 0, len(ingresses.Items))
	for _, ing := range ingresses.Items {
		// 确保 labels 不为 nil
		labels := ing.Labels
		if labels == nil {
			labels = make(map[string]string)
		}

		// 获取主机名列表
		hosts := make([]string, 0)
		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}
		}

		// 获取路径信息
		paths := make([]IngressPathInfo, 0)
		for _, rule := range ing.Spec.Rules {
			if rule.HTTP != nil {
				for _, path := range rule.HTTP.Paths {
					serviceName := ""
					servicePort := int32(0)
					if path.Backend.Service != nil {
						serviceName = path.Backend.Service.Name
						if path.Backend.Service.Port.Number > 0 {
							servicePort = path.Backend.Service.Port.Number
						}
					}

					pathInfo := IngressPathInfo{
						Host:     rule.Host,
						Path:     path.Path,
						PathType: string(*path.PathType),
						Service:  serviceName,
						Port:     servicePort,
					}
					paths = append(paths, pathInfo)
				}
			}
		}

		// 获取 TLS 配置
		tlsInfos := make([]IngressTLSInfo, 0, len(ing.Spec.TLS))
		for _, tls := range ing.Spec.TLS {
			tlsInfo := IngressTLSInfo{
				Hosts:      tls.Hosts,
				SecretName: tls.SecretName,
			}
			tlsInfos = append(tlsInfos, tlsInfo)
		}

		// 获取 IngressClass
		ingressClass := ""
		if ing.Spec.IngressClassName != nil {
			ingressClass = *ing.Spec.IngressClassName
		}

		// 获取地址
		addresses := make([]string, 0)
		for _, addr := range ing.Status.LoadBalancer.Ingress {
			if addr.IP != "" {
				addresses = append(addresses, addr.IP)
			} else if addr.Hostname != "" {
				addresses = append(addresses, addr.Hostname)
			}
		}

		ingressInfo := IngressInfo{
			Name:         ing.Name,
			Namespace:    ing.Namespace,
			Hosts:        hosts,
			Paths:        paths,
			TLS:          tlsInfos,
			IngressClass: ingressClass,
			Age:          calculateAge(ing.CreationTimestamp.Time),
			Labels:       labels,
			Addresses:    addresses,
		}

		ingressInfos = append(ingressInfos, ingressInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    ingressInfos,
	})
}

// GetIngressYAML 获取 Ingress YAML
func (h *ResourceHandler) GetIngressYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	ing, err := clientset.NetworkingV1().Ingresses(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "Ingress")
		return
	}

	// 清理对象用于 YAML 输出
	cleanedIngs := cleanIngressForYAML(ing)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"items": cleanedIngs,
		},
	})
}

// UpdateIngressYAML 更新 Ingress YAML
func (h *ResourceHandler) UpdateIngressYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	// 直接绑定请求体到 map[string]interface{}
	var jsonData map[string]interface{}
	if err := c.ShouldBindJSON(&jsonData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	fmt.Printf("🔍 [UpdateIngressYAML] 收到请求 - namespace=%s, name=%s\n", namespace, name)
	fmt.Printf("🔍 [UpdateIngressYAML] 原始请求数据: %+v\n", jsonData)

	// 提取 clusterId
	clusterIDFloat, ok := jsonData["clusterId"].(float64)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少 clusterId 字段",
		})
		return
	}
	clusterID := int(clusterIDFloat)

	// 删除 clusterId 字段，剩余的就是 Kubernetes 资源数据
	delete(jsonData, "clusterId")

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 验证资源名称
	if metadata, ok := jsonData["metadata"].(map[string]interface{}); ok {
		if jsonName, ok := metadata["name"].(string); ok && jsonName != name {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "资源名称与URL中的不一致",
			})
			return
		}
		if jsonNamespace, ok := metadata["namespace"].(string); ok && jsonNamespace != namespace {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "命名空间与URL中的不一致",
			})
			return
		}
		// 清除不可变字段，避免更新冲突
		delete(metadata, "uid")
		delete(metadata, "selfLink")
		delete(metadata, "creationTimestamp")
		delete(metadata, "deletionTimestamp")
		delete(metadata, "deletionGracePeriodSeconds")
		delete(metadata, "generation")
		delete(metadata, "resourceVersion")
		delete(metadata, "managedFields")
	}

	// 删除 status 字段（只读）
	delete(jsonData, "status")

	fmt.Printf("🔍 [UpdateIngressYAML] 清理后的数据: %+v\n", jsonData)

	// 转换为 JSON 用于 PATCH
	patchData, err := json.Marshal(jsonData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "序列化Patch数据失败: " + err.Error(),
		})
		return
	}

	fmt.Printf("🔍 [UpdateIngressYAML] Patch 数据: %s\n", string(patchData))

	updatedIngress, err := clientset.NetworkingV1().Ingresses(namespace).Patch(c.Request.Context(), name, types.MergePatchType, patchData, metav1.PatchOptions{})
	if err != nil {
		fmt.Printf("❌ [UpdateIngressYAML] Patch 失败: %v\n", err)
		HandleK8sError(c, err, "Ingress")
		return
	}

	fmt.Printf("✅ [UpdateIngressYAML] 更新成功 - Ingress: %s/%s\n", updatedIngress.Namespace, updatedIngress.Name)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// CreateIngressRequest 创建 Ingress 请求
type CreateIngressRequest struct {
	ClusterID    int                 `json:"clusterId" binding:"required"`
	Namespace    string              `json:"namespace" binding:"required"`
	Name         string              `json:"name" binding:"required"`
	IngressClass string              `json:"ingressClass"`
	Rules        []IngressRuleCreate `json:"rules" binding:"required"`
	TLS          []IngressTLSCreate  `json:"tls"`
}

// IngressRuleCreate Ingress 规则创建信息
type IngressRuleCreate struct {
	Host  string              `json:"host"`
	Paths []IngressPathCreate `json:"paths" binding:"required"`
}

// IngressPathCreate Ingress 路径创建信息
type IngressPathCreate struct {
	Path     string `json:"path" binding:"required"`
	PathType string `json:"pathType" binding:"required"`
	Service  string `json:"service" binding:"required"`
	Port     int32  `json:"port" binding:"required"`
}

// IngressTLSCreate TLS 创建信息
type IngressTLSCreate struct {
	Hosts      []string `json:"hosts"`
	SecretName string   `json:"secretName"`
}

// CreateIngress 创建 Ingress
func (h *ResourceHandler) CreateIngress(c *gin.Context) {
	namespace := c.Param("namespace")

	var req CreateIngressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 构建 Ingress 对象
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: namespace,
		},
	}

	// 设置 IngressClass
	if req.IngressClass != "" {
		ing.Spec.IngressClassName = &req.IngressClass
	}

	// 转换规则
	for _, rule := range req.Rules {
		ingressRule := networkingv1.IngressRule{
			Host: rule.Host,
		}

		if len(rule.Paths) > 0 {
			httpRule := &networkingv1.HTTPIngressRuleValue{
				Paths: make([]networkingv1.HTTPIngressPath, 0, len(rule.Paths)),
			}

			for _, path := range rule.Paths {
				pathType := networkingv1.PathType(path.PathType)
				ingressPath := networkingv1.HTTPIngressPath{
					Path:     path.Path,
					PathType: &pathType,
					Backend: networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{
							Name: path.Service,
							Port: networkingv1.ServiceBackendPort{
								Number: path.Port,
							},
						},
					},
				}
				httpRule.Paths = append(httpRule.Paths, ingressPath)
			}

			ingressRule.HTTP = httpRule
		}

		ing.Spec.Rules = append(ing.Spec.Rules, ingressRule)
	}

	// 转换 TLS
	for _, tls := range req.TLS {
		ingressTLS := networkingv1.IngressTLS{
			Hosts:      tls.Hosts,
			SecretName: tls.SecretName,
		}
		ing.Spec.TLS = append(ing.Spec.TLS, ingressTLS)
	}

	// 创建 Ingress
	_, err = clientset.NetworkingV1().Ingresses(namespace).Create(c.Request.Context(), ing, metav1.CreateOptions{})
	if err != nil {
		HandleK8sError(c, err, "Ingress")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// DeleteIngress 删除 Ingress
func (h *ResourceHandler) DeleteIngress(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	var req DeleteServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	err = clientset.NetworkingV1().Ingresses(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		HandleK8sError(c, err, "Ingress")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// cleanIngressForYAML 清理 Ingress 对象用于 YAML 输出
func cleanIngressForYAML(ing *networkingv1.Ingress) map[string]interface{} {
	result := make(map[string]interface{})
	result["apiVersion"] = "networking.k8s.io/v1"
	result["kind"] = "Ingress"
	result["metadata"] = cleanMetadata(ing.ObjectMeta)
	result["spec"] = ing.Spec
	return result
}

// ==================== Endpoints 相关 ====================

// EndpointsInfo 端点信息
type EndpointsInfo struct {
	Name      string               `json:"name"`      // 端点名称（与 Service 同名）
	Namespace string               `json:"namespace"` // 命名空间
	Subsets   []EndpointSubsetInfo `json:"subsets"`   // 端点子集
	Age       string               `json:"age"`       // 创建时间
	Labels    map[string]string    `json:"labels"`    // 标签
}

// EndpointSubsetInfo 端点子集信息
type EndpointSubsetInfo struct {
	Addresses         []EndpointAddressInfo `json:"addresses"`         // 地址列表
	NotReadyAddresses []EndpointAddressInfo `json:"notReadyAddresses"` // 未就绪地址
	Ports             []EndpointPortInfo    `json:"ports"`             // 端口列表
}

// EndpointAddressInfo 端点地址信息
type EndpointAddressInfo struct {
	IP        string `json:"ip"`        // IP 地址
	Hostname  string `json:"hostname"`  // 主机名
	NodeName  string `json:"nodeName"`  // 节点名称
	TargetRef string `json:"targetRef"` // 关联的 Pod 名称
	Ready     bool   `json:"ready"`     // 是否就绪
}

// EndpointPortInfo 端点端口信息
type EndpointPortInfo struct {
	Name     string `json:"name"`     // 端口名称
	Protocol string `json:"protocol"` // 协议
	Port     int32  `json:"port"`     // 端口号
}

// ListEndpoints 获取端点列表
func (h *ResourceHandler) ListEndpoints(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = v1.NamespaceAll
	}

	// 获取当前用户 ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	// 使用用户凭据获取 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	endpoints, err := clientset.CoreV1().Endpoints(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "端点")
		return
	}

	endpointsInfos := make([]EndpointsInfo, 0, len(endpoints.Items))
	for _, ep := range endpoints.Items {
		// 确保 labels 不为 nil
		labels := ep.Labels
		if labels == nil {
			labels = make(map[string]string)
		}

		// 转换子集信息
		subsets := make([]EndpointSubsetInfo, 0, len(ep.Subsets))
		for _, subset := range ep.Subsets {
			subsetInfo := EndpointSubsetInfo{
				Addresses:         make([]EndpointAddressInfo, 0, len(subset.Addresses)),
				NotReadyAddresses: make([]EndpointAddressInfo, 0, len(subset.NotReadyAddresses)),
				Ports:             make([]EndpointPortInfo, 0, len(subset.Ports)),
			}

			// 转换就绪地址
			for _, addr := range subset.Addresses {
				nodeName := ""
				if addr.NodeName != nil {
					nodeName = *addr.NodeName
				}
				addrInfo := EndpointAddressInfo{
					IP:       addr.IP,
					Hostname: addr.Hostname,
					NodeName: nodeName,
					Ready:    true,
				}
				if addr.TargetRef != nil {
					addrInfo.TargetRef = addr.TargetRef.Name
				}
				subsetInfo.Addresses = append(subsetInfo.Addresses, addrInfo)
			}

			// 转换未就绪地址
			for _, addr := range subset.NotReadyAddresses {
				nodeName := ""
				if addr.NodeName != nil {
					nodeName = *addr.NodeName
				}
				addrInfo := EndpointAddressInfo{
					IP:       addr.IP,
					Hostname: addr.Hostname,
					NodeName: nodeName,
					Ready:    false,
				}
				if addr.TargetRef != nil {
					addrInfo.TargetRef = addr.TargetRef.Name
				}
				subsetInfo.NotReadyAddresses = append(subsetInfo.NotReadyAddresses, addrInfo)
			}

			// 转换端口
			for _, port := range subset.Ports {
				portInfo := EndpointPortInfo{
					Name:     port.Name,
					Protocol: string(port.Protocol),
					Port:     port.Port,
				}
				subsetInfo.Ports = append(subsetInfo.Ports, portInfo)
			}

			subsets = append(subsets, subsetInfo)
		}

		endpointsInfo := EndpointsInfo{
			Name:      ep.Name,
			Namespace: ep.Namespace,
			Subsets:   subsets,
			Age:       calculateAge(ep.CreationTimestamp.Time),
			Labels:    labels,
		}

		endpointsInfos = append(endpointsInfos, endpointsInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    endpointsInfos,
	})
}

// GetEndpointsDetail 获取端点详情
func (h *ResourceHandler) GetEndpointsDetail(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	ep, err := clientset.CoreV1().Endpoints(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "端点")
		return
	}

	// 使用清理函数移除不需要的字段
	cleaned := cleanEndpointsForYAML(ep)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    cleaned,
	})
}

// CreateEndpointYAML 创建端点 (从 YAML)
func (h *ResourceHandler) CreateEndpointYAML(c *gin.Context) {
	namespace := c.Param("namespace")

	// 直接绑定请求体到 map[string]interface{}
	var jsonData map[string]interface{}
	if err := c.ShouldBindJSON(&jsonData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 提取 clusterId
	clusterIDFloat, ok := jsonData["clusterId"].(float64)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少 clusterId 字段",
		})
		return
	}
	clusterID := int(clusterIDFloat)

	// 删除 clusterId 字段，剩余的就是 Kubernetes 资源数据
	delete(jsonData, "clusterId")

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 将 jsonData 转换为 Endpoints 对象
	var endpoint v1.Endpoints
	endpointData, err := json.Marshal(jsonData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "序列化数据失败: " + err.Error(),
		})
		return
	}

	err = json.Unmarshal(endpointData, &endpoint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "解析 Endpoints 数据失败: " + err.Error(),
		})
		return
	}

	// 确保命名空间正确
	endpoint.Namespace = namespace

	// 创建 Endpoint
	_, err = clientset.CoreV1().Endpoints(namespace).Create(c.Request.Context(), &endpoint, metav1.CreateOptions{})
	if err != nil {
		HandleK8sError(c, err, "端点")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// GetEndpointYAML 获取端点 YAML
func (h *ResourceHandler) GetEndpointYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	ep, err := clientset.CoreV1().Endpoints(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "端点")
		return
	}

	// 使用清理函数移除不需要的字段
	cleaned := cleanEndpointsForYAML(ep)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    cleaned,
	})
}

// UpdateEndpointYAML 更新端点 YAML
func (h *ResourceHandler) UpdateEndpointYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	// 直接绑定请求体到 map[string]interface{}
	var jsonData map[string]interface{}
	if err := c.ShouldBindJSON(&jsonData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	fmt.Printf("🔍 [UpdateEndpointYAML] 收到请求 - namespace=%s, name=%s\n", namespace, name)
	fmt.Printf("🔍 [UpdateEndpointYAML] 原始请求数据: %+v\n", jsonData)

	// 提取 clusterId
	clusterIDFloat, ok := jsonData["clusterId"].(float64)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少 clusterId 字段",
		})
		return
	}
	clusterID := int(clusterIDFloat)

	// 删除 clusterId 字段，剩余的就是 Kubernetes 资源数据
	delete(jsonData, "clusterId")

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 清理 metadata 中的不可变字段
	if metadata, ok := jsonData["metadata"].(map[string]interface{}); ok {
		delete(metadata, "uid")
		delete(metadata, "selfLink")
		delete(metadata, "creationTimestamp")
		delete(metadata, "deletionTimestamp")
		delete(metadata, "deletionGracePeriodSeconds")
		delete(metadata, "generation")
		delete(metadata, "managedFields")
		// 注意：Endpoints 需要保留 resourceVersion，会在后面设置
	}

	// 删除 status 字段（只读）
	delete(jsonData, "status")

	fmt.Printf("🔍 [UpdateEndpointYAML] 清理后的数据: %+v\n", jsonData)

	// 获取现有 Endpoint 以保留资源版本等信息
	existingEp, err := clientset.CoreV1().Endpoints(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("❌ [UpdateEndpointYAML] 获取现有 Endpoint 失败: %v\n", err)
		HandleK8sError(c, err, "端点")
		return
	}

	// 将 jsonData 转换为 Endpoints 对象
	var endpoint v1.Endpoints
	endpointData, err := json.Marshal(jsonData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "序列化数据失败: " + err.Error(),
		})
		return
	}

	fmt.Printf("🔍 [UpdateEndpointYAML] 序列化的 Endpoint 数据: %s\n", string(endpointData))

	err = json.Unmarshal(endpointData, &endpoint)
	if err != nil {
		fmt.Printf("❌ [UpdateEndpointYAML] 解析 Endpoints 数据失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "解析 Endpoints 数据失败: " + err.Error(),
		})
		return
	}

	// 保留资源版本，确保更新成功
	endpoint.ResourceVersion = existingEp.ResourceVersion
	endpoint.UID = existingEp.UID

	fmt.Printf("🔍 [UpdateEndpointYAML] 准备更新 - ResourceVersion=%s, Subsets=%+v\n", endpoint.ResourceVersion, endpoint.Subsets)

	// 更新 Endpoint
	updatedEp, err := clientset.CoreV1().Endpoints(namespace).Update(c.Request.Context(), &endpoint, metav1.UpdateOptions{})
	if err != nil {
		fmt.Printf("❌ [UpdateEndpointYAML] Update 失败: %v\n", err)
		HandleK8sError(c, err, "端点")
		return
	}

	fmt.Printf("✅ [UpdateEndpointYAML] 更新成功 - Endpoint: %s/%s, Subsets count=%d\n", updatedEp.Namespace, updatedEp.Name, len(updatedEp.Subsets))

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// DeleteEndpoint 删除端点
func (h *ResourceHandler) DeleteEndpoint(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	var req DeleteServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	err = clientset.CoreV1().Endpoints(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		HandleK8sError(c, err, "端点")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// ==================== NetworkPolicy 相关 ====================

// NetworkPolicyDetailInfo 网络策略详情
type NetworkPolicyDetailInfo struct {
	Name        string            `json:"name"`        // 策略名称
	Namespace   string            `json:"namespace"`   // 命名空间
	PodSelector map[string]string `json:"podSelector"` // Pod 选择器
	Ingress     []PolicyRuleInfo  `json:"ingress"`     // 入站规则
	Egress      []PolicyRuleInfo  `json:"egress"`      // 出站规则
	Age         string            `json:"age"`         // 创建时间
	Labels      map[string]string `json:"labels"`      // 标签
}

// PolicyRuleInfo 策略规则
type PolicyRuleInfo struct {
	Ports []PolicyPortInfo `json:"ports"` // 端口
	From  []PolicyPeerInfo `json:"from"`  // 来源 (入站)
	To    []PolicyPeerInfo `json:"to"`    // 目标 (出站)
}

// PolicyPortInfo 策略端口
type PolicyPortInfo struct {
	Protocol string `json:"protocol"` // 协议
	Port     string `json:"port"`     // 端口号/范围
}

// PolicyPeerInfo 策略对端
type PolicyPeerInfo struct {
	PodSelector       map[string]string `json:"podSelector"`       // Pod 选择器
	NamespaceSelector map[string]string `json:"namespaceSelector"` // 命名空间选择器
	IPBlock           *IPBlockInfo      `json:"ipBlock"`           // IP 块
}

// IPBlockInfo IP 块
type IPBlockInfo struct {
	CIDR   string   `json:"cidr"`   // CIDR 表示
	Except []string `json:"except"` // 排除的 IP
}

// ListNetworkPolicies 获取网络策略列表
func (h *ResourceHandler) ListNetworkPolicies(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = v1.NamespaceAll
	}

	// 获取当前用户 ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	// 使用用户凭据获取 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	policies, err := clientset.NetworkingV1().NetworkPolicies(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "网络策略")
		return
	}

	policyInfos := make([]NetworkPolicyDetailInfo, 0, len(policies.Items))
	for _, np := range policies.Items {
		// 确保 labels 不为 nil
		labels := np.Labels
		if labels == nil {
			labels = make(map[string]string)
		}

		// 转换 Pod 选择器
		podSelector := make(map[string]string)
		if np.Spec.PodSelector.MatchLabels != nil {
			podSelector = np.Spec.PodSelector.MatchLabels
		}

		// 转换入站规则
		ingressRules := make([]PolicyRuleInfo, 0, len(np.Spec.Ingress))
		for _, rule := range np.Spec.Ingress {
			ruleInfo := PolicyRuleInfo{}

			// 转换端口
			for _, port := range rule.Ports {
				portInfo := PolicyPortInfo{}
				if port.Protocol != nil {
					portInfo.Protocol = string(*port.Protocol)
				}
				if port.Port != nil {
					if port.Port.Type == intstr.Int {
						portInfo.Port = strconv.Itoa(int(port.Port.IntVal))
					} else {
						portInfo.Port = port.Port.StrVal
					}
				}
				ruleInfo.Ports = append(ruleInfo.Ports, portInfo)
			}

			// 转换来源
			for _, from := range rule.From {
				peerInfo := PolicyPeerInfo{}
				if from.PodSelector != nil {
					peerInfo.PodSelector = from.PodSelector.MatchLabels
				}
				if from.NamespaceSelector != nil {
					peerInfo.NamespaceSelector = from.NamespaceSelector.MatchLabels
				}
				if from.IPBlock != nil {
					peerInfo.IPBlock = &IPBlockInfo{
						CIDR:   from.IPBlock.CIDR,
						Except: from.IPBlock.Except,
					}
				}
				ruleInfo.From = append(ruleInfo.From, peerInfo)
			}

			ingressRules = append(ingressRules, ruleInfo)
		}

		// 转换出站规则
		egressRules := make([]PolicyRuleInfo, 0, len(np.Spec.Egress))
		for _, rule := range np.Spec.Egress {
			ruleInfo := PolicyRuleInfo{}

			// 转换端口
			for _, port := range rule.Ports {
				portInfo := PolicyPortInfo{}
				if port.Protocol != nil {
					portInfo.Protocol = string(*port.Protocol)
				}
				if port.Port != nil {
					if port.Port.Type == intstr.Int {
						portInfo.Port = strconv.Itoa(int(port.Port.IntVal))
					} else {
						portInfo.Port = port.Port.StrVal
					}
				}
				ruleInfo.Ports = append(ruleInfo.Ports, portInfo)
			}

			// 转换目标
			for _, to := range rule.To {
				peerInfo := PolicyPeerInfo{}
				if to.PodSelector != nil {
					peerInfo.PodSelector = to.PodSelector.MatchLabels
				}
				if to.NamespaceSelector != nil {
					peerInfo.NamespaceSelector = to.NamespaceSelector.MatchLabels
				}
				if to.IPBlock != nil {
					peerInfo.IPBlock = &IPBlockInfo{
						CIDR:   to.IPBlock.CIDR,
						Except: to.IPBlock.Except,
					}
				}
				ruleInfo.To = append(ruleInfo.To, peerInfo)
			}

			egressRules = append(egressRules, ruleInfo)
		}

		policyInfo := NetworkPolicyDetailInfo{
			Name:        np.Name,
			Namespace:   np.Namespace,
			PodSelector: podSelector,
			Ingress:     ingressRules,
			Egress:      egressRules,
			Age:         calculateAge(np.CreationTimestamp.Time),
			Labels:      labels,
		}

		policyInfos = append(policyInfos, policyInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    policyInfos,
	})
}

// GetNetworkPolicyYAML 获取网络策略 YAML
func (h *ResourceHandler) GetNetworkPolicyYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	np, err := clientset.NetworkingV1().NetworkPolicies(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "网络策略")
		return
	}

	// 清理对象用于 YAML 输出
	cleanedNp := cleanNetworkPolicyForYAML(np)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"items": cleanedNp,
		},
	})
}

// UpdateNetworkPolicyYAML 更新网络策略 YAML
func (h *ResourceHandler) UpdateNetworkPolicyYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	// 直接绑定请求体到 map[string]interface{}
	var jsonData map[string]interface{}
	if err := c.ShouldBindJSON(&jsonData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 提取 clusterId
	clusterIDFloat, ok := jsonData["clusterId"].(float64)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少 clusterId 字段",
		})
		return
	}
	clusterID := int(clusterIDFloat)

	// 删除 clusterId 字段，剩余的就是 Kubernetes 资源数据
	delete(jsonData, "clusterId")

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 验证资源名称
	if metadata, ok := jsonData["metadata"].(map[string]interface{}); ok {
		if jsonName, ok := metadata["name"].(string); ok && jsonName != name {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "资源名称与URL中的不一致",
			})
			return
		}
		if jsonNamespace, ok := metadata["namespace"].(string); ok && jsonNamespace != namespace {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "命名空间与URL中的不一致",
			})
			return
		}
		// 清除不可变字段，避免更新冲突
		delete(metadata, "uid")
		delete(metadata, "selfLink")
		delete(metadata, "creationTimestamp")
		delete(metadata, "deletionTimestamp")
		delete(metadata, "deletionGracePeriodSeconds")
		delete(metadata, "generation")
		delete(metadata, "resourceVersion")
	}

	// 转换为 JSON 用于 PATCH
	patchData, err := json.Marshal(jsonData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "序列化Patch数据失败: " + err.Error(),
		})
		return
	}

	_, err = clientset.NetworkingV1().NetworkPolicies(namespace).Patch(c.Request.Context(), name, types.MergePatchType, patchData, metav1.PatchOptions{})
	if err != nil {
		HandleK8sError(c, err, "网络策略")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// ==================== NetworkPolicy 创建相关 ====================

// CreateNetworkPolicyRequest 创建 NetworkPolicy 请求
type CreateNetworkPolicyRequest struct {
	ClusterID   int                        `json:"clusterId" binding:"required"`
	PodSelector map[string]string          `json:"podSelector"`
	PolicyTypes []string                   `json:"policyTypes"` // Ingress, Egress
	Ingress     []NetworkPolicyIngressRule `json:"ingress"`
	Egress      []NetworkPolicyEgressRule  `json:"egress"`
}

// NetworkPolicyIngressRule 入站规则
type NetworkPolicyIngressRule struct {
	Ports []NetworkPolicyPort `json:"ports"`
	From  []NetworkPolicyPeer `json:"from"`
}

// NetworkPolicyEgressRule 出站规则
type NetworkPolicyEgressRule struct {
	Ports []NetworkPolicyPort `json:"ports"`
	To    []NetworkPolicyPeer `json:"to"`
}

// NetworkPolicyPort 端口
type NetworkPolicyPort struct {
	Protocol  string `json:"protocol"`  // TCP, UDP, SCTP
	Port      *int32 `json:"port"`      // 端口编号
	EndPort   *int32 `json:"endPort"`   // 结束端口（范围）
	NamedPort string `json:"namedPort"` // 命名端口
}

// NetworkPolicyPeer 对端
type NetworkPolicyPeer struct {
	PodSelector       map[string]string     `json:"podSelector"`
	NamespaceSelector map[string]string     `json:"namespaceSelector"`
	IPBlock           *NetworkPolicyIPBlock `json:"ipBlock"`
}

// NetworkPolicyIPBlock IP 段
type NetworkPolicyIPBlock struct {
	CIDR   string   `json:"cidr"`   // IP 地址段
	Except []string `json:"except"` // 排除的 IP
}

// CreateNetworkPolicy 创建网络策略
func (h *ResourceHandler) CreateNetworkPolicy(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	var req CreateNetworkPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 构建 NetworkPolicy 对象
	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: req.PodSelector,
			},
		},
	}

	// 设置 PolicyTypes
	for _, pt := range req.PolicyTypes {
		np.Spec.PolicyTypes = append(np.Spec.PolicyTypes, networkingv1.PolicyType(pt))
	}

	// 转换 Ingress 规则
	for _, rule := range req.Ingress {
		npIngressRule := networkingv1.NetworkPolicyIngressRule{}

		// 转换 Ports
		for _, port := range rule.Ports {
			npPort := networkingv1.NetworkPolicyPort{}
			if port.Protocol != "" {
				protocol := v1.Protocol(port.Protocol)
				npPort.Protocol = &protocol
			}

			if port.Port != nil {
				intOrStr := intstr.FromInt(int(*port.Port))
				npPort.Port = &intOrStr
			}
			if port.EndPort != nil {
				npPort.EndPort = port.EndPort
			}
			if port.NamedPort != "" {
				// 使用 intstr.String
				strVal := intstr.FromString(port.NamedPort)
				npPort.Port = &strVal
			}

			npIngressRule.Ports = append(npIngressRule.Ports, npPort)
		}

		// 转换 From
		for _, peer := range rule.From {
			npPeer := networkingv1.NetworkPolicyPeer{}

			if peer.PodSelector != nil {
				npPeer.PodSelector = &metav1.LabelSelector{
					MatchLabels: peer.PodSelector,
				}
			}

			if peer.NamespaceSelector != nil {
				npPeer.NamespaceSelector = &metav1.LabelSelector{
					MatchLabels: peer.NamespaceSelector,
				}
			}

			if peer.IPBlock != nil {
				npPeer.IPBlock = &networkingv1.IPBlock{
					CIDR:   peer.IPBlock.CIDR,
					Except: peer.IPBlock.Except,
				}
			}

			npIngressRule.From = append(npIngressRule.From, npPeer)
		}

		np.Spec.Ingress = append(np.Spec.Ingress, npIngressRule)
	}

	// 转换 Egress 规则
	for _, rule := range req.Egress {
		npEgressRule := networkingv1.NetworkPolicyEgressRule{}

		// 转换 Ports
		for _, port := range rule.Ports {
			npPort := networkingv1.NetworkPolicyPort{}
			if port.Protocol != "" {
				protocol := v1.Protocol(port.Protocol)
				npPort.Protocol = &protocol
			}

			if port.Port != nil {
				intOrStr := intstr.FromInt(int(*port.Port))
				npPort.Port = &intOrStr
			}
			if port.EndPort != nil {
				npPort.EndPort = port.EndPort
			}
			if port.NamedPort != "" {
				strVal := intstr.FromString(port.NamedPort)
				npPort.Port = &strVal
			}

			npEgressRule.Ports = append(npEgressRule.Ports, npPort)
		}

		// 转换 To
		for _, peer := range rule.To {
			npPeer := networkingv1.NetworkPolicyPeer{}

			if peer.PodSelector != nil {
				npPeer.PodSelector = &metav1.LabelSelector{
					MatchLabels: peer.PodSelector,
				}
			}

			if peer.NamespaceSelector != nil {
				npPeer.NamespaceSelector = &metav1.LabelSelector{
					MatchLabels: peer.NamespaceSelector,
				}
			}

			if peer.IPBlock != nil {
				npPeer.IPBlock = &networkingv1.IPBlock{
					CIDR:   peer.IPBlock.CIDR,
					Except: peer.IPBlock.Except,
				}
			}

			npEgressRule.To = append(npEgressRule.To, npPeer)
		}

		np.Spec.Egress = append(np.Spec.Egress, npEgressRule)
	}

	// 创建 NetworkPolicy
	_, err = clientset.NetworkingV1().NetworkPolicies(namespace).Create(c.Request.Context(), np, metav1.CreateOptions{})
	if err != nil {
		HandleK8sError(c, err, "网络策略")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// DeleteNetworkPolicy 删除网络策略
func (h *ResourceHandler) DeleteNetworkPolicy(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	var req DeleteServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	err = clientset.NetworkingV1().NetworkPolicies(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		HandleK8sError(c, err, "网络策略")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// cleanNetworkPolicyForYAML 清理 NetworkPolicy 对象用于 YAML 输出
func cleanNetworkPolicyForYAML(np *networkingv1.NetworkPolicy) map[string]interface{} {
	result := make(map[string]interface{})
	result["apiVersion"] = "networking.k8s.io/v1"
	result["kind"] = "NetworkPolicy"
	result["metadata"] = cleanMetadata(np.ObjectMeta)
	result["spec"] = np.Spec
	return result
}

// CreateNamespaceRequest 创建命名空间请求
type CreateNamespaceRequest struct {
	YAML string `json:"yaml" binding:"required"`
}

// UpdateNamespaceYAMLRequest 更新命名空间YAML请求
type UpdateNamespaceYAMLRequest struct {
	ClusterID int    `json:"clusterId" binding:"required"`
	YAML      string `json:"yaml" binding:"required"`
}

// CreateNamespace 创建命名空间
// @Summary 创建命名空间
// @Description 创建新的 Kubernetes 命名空间
// @Tags Kubernetes/命名空间
// @Accept json
// @Produce json
// @Security Bearer
// @Param clusterId query int true "集群ID"
// @Param body body object true "命名空间配置"
// @Success 200 {object} map[string]interface{} "创建成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/resources/namespaces [post]
func (h *ResourceHandler) CreateNamespace(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	var req CreateNamespaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	// 解析 YAML
	var namespace v1.Namespace
	if err := yaml.Unmarshal([]byte(req.YAML), &namespace); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析 YAML 失败: " + err.Error(),
		})
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 创建命名空间
	_, err = clientset.CoreV1().Namespaces().Create(c.Request.Context(), &namespace, metav1.CreateOptions{})
	if err != nil {
		HandleK8sError(c, err, "命名空间")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// GetNamespaceYAML 获取命名空间YAML
func (h *ResourceHandler) GetNamespaceYAML(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespaceName := c.Param("namespaceName")
	if namespaceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "命名空间名称不能为空",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	namespace, err := clientset.CoreV1().Namespaces().Get(c.Request.Context(), namespaceName, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "命名空间")
		return
	}

	// 清理对象用于 YAML 输出
	cleaned := cleanNamespaceForYAML(namespace)
	yamlBytes, err := yaml.Marshal(cleaned)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "序列化 YAML 失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"yaml": string(yamlBytes),
		},
	})
}

// UpdateNamespaceYAML 更新命名空间YAML
func (h *ResourceHandler) UpdateNamespaceYAML(c *gin.Context) {
	namespaceName := c.Param("namespaceName")
	if namespaceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "命名空间名称不能为空",
		})
		return
	}

	var req UpdateNamespaceYAMLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 解析 YAML
	var namespace v1.Namespace
	if err := yaml.Unmarshal([]byte(req.YAML), &namespace); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析 YAML 失败: " + err.Error(),
		})
		return
	}

	// 确保名称一致
	namespace.Name = namespaceName

	// 更新命名空间
	_, err = clientset.CoreV1().Namespaces().Update(c.Request.Context(), &namespace, metav1.UpdateOptions{})
	if err != nil {
		HandleK8sError(c, err, "命名空间")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// DeleteNamespace 删除命名空间
func (h *ResourceHandler) DeleteNamespace(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespaceName := c.Param("namespaceName")
	if namespaceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "命名空间名称不能为空",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	err = clientset.CoreV1().Namespaces().Delete(c.Request.Context(), namespaceName, metav1.DeleteOptions{})
	if err != nil {
		HandleK8sError(c, err, "命名空间")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// cleanNamespaceForYAML 清理 Namespace 对象用于 YAML 输出
func cleanNamespaceForYAML(ns *v1.Namespace) map[string]interface{} {
	result := make(map[string]interface{})
	result["apiVersion"] = "v1"
	result["kind"] = "Namespace"
	result["metadata"] = cleanMetadata(ns.ObjectMeta)
	return result
}

// ==================== ConfigMap 相关 ====================

// ConfigMapInfo ConfigMap 信息
type ConfigMapInfo struct {
	Name      string            `json:"name"`      // ConfigMap 名称
	Namespace string            `json:"namespace"` // 命名空间
	DataCount int               `json:"dataCount"` // 数据项数量
	Age       string            `json:"age"`       // 创建时间
	CreatedAt string            `json:"createdAt"` // 创建时间（完整格式）
	Labels    map[string]string `json:"labels"`    // 标签
}

// ListConfigMaps 获取 ConfigMap 列表
// @Summary 获取ConfigMap列表
// @Description 获取指定命名空间的 ConfigMap 列表
// @Tags Kubernetes/配置
// @Accept json
// @Produce json
// @Security Bearer
// @Param clusterId query int true "集群ID"
// @Param namespace query string false "命名空间"
// @Success 200 {object} map[string]interface{} "ConfigMap列表"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/resources/configmaps [get]
func (h *ResourceHandler) ListConfigMaps(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = v1.NamespaceAll
	}

	// 获取当前用户 ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	// 使用用户凭据获取 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	configMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "ConfigMap")
		return
	}

	configMapInfos := make([]ConfigMapInfo, 0, len(configMaps.Items))
	for _, cm := range configMaps.Items {
		// 确保 labels 不为 nil
		labels := cm.Labels
		if labels == nil {
			labels = make(map[string]string)
		}

		configMapInfo := ConfigMapInfo{
			Name:      cm.Name,
			Namespace: cm.Namespace,
			DataCount: len(cm.Data),
			Age:       calculateAge(cm.CreationTimestamp.Time),
			CreatedAt: cm.CreationTimestamp.Format("2006-01-02 15:04:05"),
			Labels:    labels,
		}

		configMapInfos = append(configMapInfos, configMapInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    configMapInfos,
	})
}

// GetConfigMapYAML 获取 ConfigMap YAML
func (h *ResourceHandler) GetConfigMapYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "ConfigMap")
		return
	}

	// 使用清理函数移除不需要的字段
	cleaned := cleanConfigMapForYAML(configMap)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items":   cleaned,
			"total":   0,
			"page":    0,
			"pageSize": 0,
		},
		"msg": "获取成功",
	})
}

// UpdateConfigMapYAML 更新 ConfigMap YAML
func (h *ResourceHandler) UpdateConfigMapYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	// 从 query 参数获取 clusterId
	clusterIDStr := c.Query("clusterId")
	if clusterIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少 clusterId 参数",
		})
		return
	}

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "clusterId 参数格式错误",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 解析请求体为 map，以便清除不可变字段
	var jsonData map[string]interface{}
	if err := c.ShouldBindJSON(&jsonData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析请求体失败: " + err.Error(),
		})
		return
	}

	// 验证并清除metadata中的不可变字段
	if metadata, ok := jsonData["metadata"].(map[string]interface{}); ok {
		if jsonName, ok := metadata["name"].(string); ok && jsonName != name {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "资源名称与URL中的不一致",
			})
			return
		}
		if jsonNamespace, ok := metadata["namespace"].(string); ok && jsonNamespace != namespace {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "命名空间与URL中的不一致",
			})
			return
		}
		// 清除不可变字段
		delete(metadata, "uid")
		delete(metadata, "selfLink")
		delete(metadata, "creationTimestamp")
		delete(metadata, "deletionTimestamp")
		delete(metadata, "deletionGracePeriodSeconds")
		delete(metadata, "generation")
		delete(metadata, "resourceVersion")
	}

	// 转换为 JSON 用于 PATCH
	patchData, err := json.Marshal(jsonData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "序列化Patch数据失败: " + err.Error(),
		})
		return
	}

	// 使用 Patch 方法更新，避免不可变字段冲突
	_, err = clientset.CoreV1().ConfigMaps(namespace).Patch(c.Request.Context(), name, types.MergePatchType, patchData, metav1.PatchOptions{})
	if err != nil {
		HandleK8sError(c, err, "ConfigMap")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// DeleteConfigMap 删除 ConfigMap
func (h *ResourceHandler) DeleteConfigMap(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	err = clientset.CoreV1().ConfigMaps(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		HandleK8sError(c, err, "ConfigMap")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// ==================== Secret 相关 ====================

// SecretInfo Secret 信息
type SecretInfo struct {
	Name      string            `json:"name"`      // Secret 名称
	Namespace string            `json:"namespace"` // 命名空间
	Type      string            `json:"type"`      // Secret 类型
	DataCount int               `json:"dataCount"` // 数据项数量
	Age       string            `json:"age"`       // 创建时间
	Labels    map[string]string `json:"labels"`    // 标签
}

// ListSecrets 获取 Secret 列表
// @Summary 获取Secret列表
// @Description 获取指定命名空间的 Secret 列表
// @Tags Kubernetes/配置
// @Accept json
// @Produce json
// @Security Bearer
// @Param clusterId query int true "集群ID"
// @Param namespace query string false "命名空间"
// @Success 200 {object} map[string]interface{} "Secret列表"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/resources/secrets [get]
func (h *ResourceHandler) ListSecrets(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = v1.NamespaceAll
	}

	// 获取当前用户 ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	// 使用用户凭据获取 clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	secrets, err := clientset.CoreV1().Secrets(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "Secret")
		return
	}

	secretInfos := make([]SecretInfo, 0, len(secrets.Items))
	for _, s := range secrets.Items {
		// 确保 labels 不为 nil
		labels := s.Labels
		if labels == nil {
			labels = make(map[string]string)
		}

		secretInfo := SecretInfo{
			Name:      s.Name,
			Namespace: s.Namespace,
			Type:      string(s.Type),
			DataCount: len(s.Data),
			Age:       calculateAge(s.CreationTimestamp.Time),
			Labels:    labels,
		}

		secretInfos = append(secretInfos, secretInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    secretInfos,
	})
}

// GetSecretYAML 获取 Secret YAML
func (h *ResourceHandler) GetSecretYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "Secret")
		return
	}

	// 使用清理函数移除不需要的字段
	cleaned := cleanSecretForYAML(secret)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items":   cleaned,
			"total":   0,
			"page":    0,
			"pageSize": 0,
		},
		"msg": "获取成功",
	})
}

// UpdateSecretYAML 更新 Secret YAML
func (h *ResourceHandler) UpdateSecretYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	// 从 query 参数获取 clusterId
	clusterIDStr := c.Query("clusterId")
	if clusterIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少 clusterId 参数",
		})
		return
	}

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "clusterId 参数格式错误",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 解析请求体为 map，以便清除不可变字段
	var jsonData map[string]interface{}
	if err := c.ShouldBindJSON(&jsonData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析请求体失败: " + err.Error(),
		})
		return
	}

	// 验证并清除metadata中的不可变字段
	if metadata, ok := jsonData["metadata"].(map[string]interface{}); ok {
		if jsonName, ok := metadata["name"].(string); ok && jsonName != name {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "资源名称与URL中的不一致",
			})
			return
		}
		if jsonNamespace, ok := metadata["namespace"].(string); ok && jsonNamespace != namespace {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "命名空间与URL中的不一致",
			})
			return
		}
		// 清除不可变字段
		delete(metadata, "uid")
		delete(metadata, "selfLink")
		delete(metadata, "creationTimestamp")
		delete(metadata, "deletionTimestamp")
		delete(metadata, "deletionGracePeriodSeconds")
		delete(metadata, "generation")
		delete(metadata, "resourceVersion")
	}

	// 转换为 JSON 用于 PATCH
	patchData, err := json.Marshal(jsonData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "序列化Patch数据失败: " + err.Error(),
		})
		return
	}

	// 使用 Patch 方法更新，避免不可变字段冲突
	_, err = clientset.CoreV1().Secrets(namespace).Patch(c.Request.Context(), name, types.MergePatchType, patchData, metav1.PatchOptions{})
	if err != nil {
		HandleK8sError(c, err, "Secret")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// DeleteSecret 删除 Secret
func (h *ResourceHandler) DeleteSecret(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	err = clientset.CoreV1().Secrets(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		HandleK8sError(c, err, "Secret")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// GetPodLogs 获取Pod日志
// @Summary 获取Pod日志
// @Description 获取指定 Pod 容器的日志
// @Tags Kubernetes/工作负载
// @Accept json
// @Produce json
// @Security Bearer
// @Param clusterId query int true "集群ID"
// @Param namespace query string true "命名空间"
// @Param name query string true "Pod名称"
// @Param container query string false "容器名称"
// @Param tailLines query int false "返回日志行数"
// @Param previous query bool false "是否获取上一个容器的日志"
// @Success 200 {object} map[string]interface{} "Pod日志"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/resources/pods/logs [get]
func (h *ResourceHandler) GetPodLogs(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	namespace := c.Query("namespace")
	podName := c.Query("podName")
	container := c.Query("container")
	tailLinesStr := c.DefaultQuery("tailLines", "100")

	if clusterIDStr == "" || namespace == "" || podName == "" || container == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少必要参数: clusterId, namespace, podName, container",
		})
		return
	}

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	tailLines, err := strconv.ParseInt(tailLinesStr, 10, 64)
	if err != nil {
		tailLines = 100
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 获取Pod日志请求
	podLogOptions := &v1.PodLogOptions{
		Container:  container,
		Timestamps: true,
	}

	// 只有当 tailLines > 0 时才设置 TailLines 参数
	// tailLines = 0 表示获取全部日志（不传 TailLines 参数）
	if tailLines > 0 {
		podLogOptions.TailLines = &tailLines
	}

	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, podLogOptions)

	logStream, err := req.Stream(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取日志失败: " + err.Error(),
		})
		return
	}
	defer logStream.Close()

	// 读取日志内容
	logs, err := io.ReadAll(logStream)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "读取日志失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data": gin.H{
			"logs": string(logs),
		},
	})
}

// GetPodsMetrics 获取Pod的实际使用指标（CPU和内存）
func (h *ResourceHandler) GetPodsMetrics(c *gin.Context) {
	namespace := c.Query("namespace")
	clusterIDStr := c.Query("clusterId")

	if clusterIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少必要参数: clusterId",
		})
		return
	}

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取 metrics clientset
	metricsClient, err := h.clusterService.GetCachedMetricsClientset(c.Request.Context(), uint(clusterID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取 metrics client 失败: " + err.Error(),
		})
		return
	}

	// 获取所有 Pod metrics
	allPodMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取 Pod metrics 失败: " + err.Error(),
		})
		return
	}

	// 构建返回数据：map[podName] -> metrics
	metricsMap := make(map[string]interface{})
	for _, pm := range allPodMetrics.Items {
		podName := pm.Name

		// CPU 使用量（从 MilliCPU 转换）
		cpuUsage := int64(0)
		if pm.Containers != nil {
			for _, c := range pm.Containers {
				if c.Usage != nil {
					cpuUsage += c.Usage.Cpu().MilliValue()
				}
			}
		}

		// 内存使用量（从字节转换）
		memoryUsage := int64(0)
		if pm.Containers != nil {
			for _, c := range pm.Containers {
				if c.Usage != nil {
					memoryUsage += c.Usage.Memory().Value()
				}
			}
		}

		metricsMap[podName] = map[string]interface{}{
			"cpu":       cpuUsage,    // 毫核
			"memory":    memoryUsage, // 字节
			"cpuStr":    formatCPUMetrics(cpuUsage),
			"memoryStr": formatMemoryMetrics(memoryUsage),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data": gin.H{
			"metrics": metricsMap,
		},
	})
}

// formatCPUMetrics 格式化 CPU 使用量
func formatCPUMetrics(milliValue int64) string {
	if milliValue == 0 {
		return "-"
	}
	if milliValue >= 1000 {
		return fmt.Sprintf("%.1f Core", float64(milliValue)/1000)
	}
	return fmt.Sprintf("%dm", milliValue)
}

// formatMemoryMetrics 格式化内存使用量
func formatMemoryMetrics(bytes int64) string {
	if bytes == 0 {
		return "-"
	}
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	if bytes >= GB {
		return fmt.Sprintf("%.1f Gi", float64(bytes)/float64(GB))
	}
	if bytes >= MB {
		return fmt.Sprintf("%.0f Mi", float64(bytes)/float64(MB))
	}
	if bytes >= KB {
		return fmt.Sprintf("%.0f Ki", float64(bytes)/float64(KB))
	}
	return fmt.Sprintf("%d B", bytes)
}

// PodShellWebSocket Pod容器Shell WebSocket连接
// @Summary Pod终端WebSocket
// @Description 通过 WebSocket 连接到 Pod 容器的终端
// @Tags Kubernetes/终端
// @Accept json
// @Produce json
// @Security Bearer
// @Param clusterId query int true "集群ID"
// @Param namespace query string true "命名空间"
// @Param pod query string true "Pod名称"
// @Param container query string false "容器名称"
// @Success 101 {string} string "WebSocket连接升级成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Router /plugins/kubernetes/shell/pods [get]
func (h *ResourceHandler) PodShellWebSocket(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	namespace := c.Query("namespace")
	podName := c.Query("podName")
	containerName := c.Query("container")

	if clusterIDStr == "" || namespace == "" || podName == "" || containerName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少必要参数",
		})
		return
	}

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 获取当前用户ID（从认证中间件设置的 context 中获取）
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 获取用户名（从认证中间件设置的 context 中获取）
	username := ""
	if usernameVal, exists := c.Get("username"); exists {
		username = usernameVal.(string)
	}

	// 升级到 WebSocket 连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	fmt.Printf("🐚 WebSocket shell connected to pod %s/%s, container %s, clusterID=%d\n", namespace, podName, containerName, clusterID)

	// 获取 REST config
	restConfig, err := h.clusterService.GetRESTConfig(uint(clusterID), currentUserID.(uint))
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("获取集群配置失败: "+err.Error()+"\r\n"))
		return
	}

	// 构造 exec URL
	serverURL, err := url.Parse(restConfig.Host)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("解析集群 URL 失败: "+err.Error()+"\r\n"))
		return
	}

	// 构造 query 参数
	query := url.Values{}
	query.Set("container", containerName)
	query.Set("stdin", "true")
	query.Set("stdout", "true")
	query.Set("stderr", "true")
	query.Set("tty", "true")

	// 添加要执行的命令
	query.Add("command", "/bin/sh")
	query.Add("command", "-c")
	query.Add("command", "command -v bash >/dev/null 2>&1 && exec bash || exec sh")

	execURL := &url.URL{
		Scheme:   serverURL.Scheme,
		Host:     serverURL.Host,
		Path:     fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/exec", namespace, podName),
		RawQuery: query.Encode(),
	}

	fmt.Printf("🐚 Exec URL: %s\n", execURL.String())

	// 创建 SPDY executor
	executor, err := remotecommand.NewSPDYExecutor(restConfig, "POST", execURL)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("创建 executor 失败: "+err.Error()+"\r\n"))
		return
	}

	// 创建录制器（录制目录）
	recordingDir := "./data/terminal-recordings"
	recorder, err := NewAsciinemaRecorder(recordingDir, 120, 30)
	if err != nil {
		// 录制失败不影响终端使用，只是不录制
		recorder = nil
	}

	// 创建 WebSocket 读写器（带录制功能）
	wsReader := &RecordingWebSocketReader{
		conn:      conn,
		data:      make(chan []byte, 256),
		recorder:  recorder,
		startTime: time.Now(),
	}
	wsWriter := &RecordingWebSocketWriter{
		conn:      conn,
		recorder:  recorder,
		startTime: time.Now(),
	}

	// 处理 WebSocket 消息
	done := make(chan struct{})
	// 创建可取消的 context
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer close(done)
		defer cancel() // 当 goroutine 结束时取消 context
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			wsReader.data <- message
		}
	}()

	// 流式处理
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  wsReader,
		Stdout: wsWriter,
		Stderr: wsWriter,
		Tty:    true,
	})

	// 等待读取 goroutine 结束
	<-done

	// 关闭录制器并保存会话记录
	if recorder != nil {
		duration := recorder.GetDuration()
		fileSize := recorder.GetFileSize()
		recordingPath := recorder.GetRecordingPath()

		recorder.Close()

		// 获取集群名称
		var cluster models.Cluster
		clusterName := ""
		if err := h.db.First(&cluster, clusterID).Error; err == nil {
			clusterName = cluster.Alias
			if clusterName == "" {
				clusterName = cluster.Name
			}
		} else {
			clusterName = fmt.Sprintf("Cluster-%d", clusterID)
		}

		// 保存会话记录到数据库（所有会话都记录）
		session := model.TerminalSession{
			ClusterID:     uint(clusterID),
			ClusterName:   clusterName,
			Namespace:     namespace,
			PodName:       podName,
			ContainerName: containerName,
			UserID:        currentUserID.(uint),
			Username:      username,
			RecordingPath: recordingPath,
			Duration:      duration,
			FileSize:      fileSize,
			Status:        model.SessionStatusCompleted,
		}

		h.db.Create(&session)
	}
}

// PauseWorkload 暂停/恢复工作负载
func (h *ResourceHandler) PauseWorkload(c *gin.Context) {
	fmt.Printf("🎯 PauseWorkload called\n")

	var req struct {
		ClusterID uint   `json:"clusterId" binding:"required"`
		Namespace string `json:"namespace" binding:"required"`
		Name      string `json:"name" binding:"required"`
		Type      string `json:"type" binding:"required"`
		Paused    bool   `json:"paused"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("❌ Bind error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	fmt.Printf("✅ Parsed: ClusterID=%d, Namespace=%s, Name=%s, Type=%s, Paused=%v\n",
		req.ClusterID, req.Namespace, req.Name, req.Type, req.Paused)

	// 获取 clientset
	clientset, err := h.clusterService.GetCachedClientset(c.Request.Context(), req.ClusterID)
	if err != nil {
		fmt.Printf("❌ GetCachedClientset error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群 clientset 失败: " + err.Error(),
		})
		return
	}
	fmt.Printf("✅ Got clientset\n")

	switch req.Type {
	case "Deployment":
		fmt.Printf("📦 Processing Deployment...\n")
		deployment, err := clientset.AppsV1().Deployments(req.Namespace).Get(c.Request.Context(), req.Name, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("❌ Get Deployment error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取 Deployment 失败: " + err.Error(),
			})
			return
		}
		fmt.Printf("✅ Got Deployment, current paused=%v\n", deployment.Spec.Paused)

		// 更新暂停状态
		deployment.Spec.Paused = req.Paused
		fmt.Printf("📝 Setting paused to %v\n", req.Paused)

		updatedDeployment, err := clientset.AppsV1().Deployments(req.Namespace).Update(c.Request.Context(), deployment, metav1.UpdateOptions{})
		if err != nil {
			fmt.Printf("❌ Update Deployment error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新 Deployment 失败: " + err.Error(),
			})
			return
		}
		fmt.Printf("✅ Deployment updated, new paused=%v\n", updatedDeployment.Spec.Paused)

	default:
		fmt.Printf("❌ Unsupported workload type: %s\n", req.Type)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "不支持的工作负载类型: " + req.Type,
		})
		return
	}

	fmt.Printf("✅ Sending success response\n")
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "操作成功",
		"data": gin.H{
			"paused": req.Paused,
		},
	})
}

// RollbackWorkload 回滚工作负载到指定版本
func (h *ResourceHandler) RollbackWorkload(c *gin.Context) {
	fmt.Printf("🔄 RollbackWorkload called\n")

	var req struct {
		ClusterID uint   `json:"clusterId" binding:"required"`
		Namespace string `json:"namespace" binding:"required"`
		Name      string `json:"name" binding:"required"`
		Type      string `json:"type" binding:"required"`
		Revision  string `json:"revision" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("❌ Bind error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	fmt.Printf("✅ Parsed: ClusterID=%d, Namespace=%s, Name=%s, Type=%s, Revision=%s\n",
		req.ClusterID, req.Namespace, req.Name, req.Type, req.Revision)

	// 获取 clientset
	clientset, err := h.clusterService.GetCachedClientset(c.Request.Context(), req.ClusterID)
	if err != nil {
		fmt.Printf("❌ GetCachedClientset error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群 clientset 失败: " + err.Error(),
		})
		return
	}
	fmt.Printf("✅ Got clientset\n")

	switch req.Type {
	case "Deployment":
		fmt.Printf("📦 Processing Deployment rollback...\n")

		// 使用 Kubernetes Rollback API (如果可用) 或者手动回滚
		// 注意：Kubernetes 1.15+ 移除了 rollout undo 命令，需要使用其他方式
		// 这里我们使用创建新的 ReplicaSet 的方式来回滚

		// 首先找到对应版本的 ReplicaSet
		replicaSets, err := clientset.AppsV1().ReplicaSets(req.Namespace).List(c.Request.Context(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("app=%s", req.Name), // 假设有 app 标签
		})
		if err != nil {
			fmt.Printf("❌ List ReplicaSets error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取 ReplicaSet 列表失败: " + err.Error(),
			})
			return
		}

		// 找到匹配 revision 的 ReplicaSet
		var targetReplicaSet *appsv1.ReplicaSet
		for i := range replicaSets.Items {
			rs := &replicaSets.Items[i]
			revision := rs.Annotations["deployment.kubernetes.io/revision"]
			if revision == req.Revision {
				targetReplicaSet = rs
				break
			}
		}

		if targetReplicaSet == nil {
			fmt.Printf("❌ Target ReplicaSet not found for revision %s\n", req.Revision)
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "未找到指定版本的 ReplicaSet",
			})
			return
		}

		fmt.Printf("✅ Found target ReplicaSet: %s\n", targetReplicaSet.Name)

		// 获取当前 Deployment
		deployment, err := clientset.AppsV1().Deployments(req.Namespace).Get(c.Request.Context(), req.Name, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("❌ Get Deployment error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取 Deployment 失败: " + err.Error(),
			})
			return
		}

		// 更新 Deployment 的 template 来匹配目标 ReplicaSet
		deployment.Spec.Template = targetReplicaSet.Spec.Template

		// 更新 Deployment
		_, err = clientset.AppsV1().Deployments(req.Namespace).Update(c.Request.Context(), deployment, metav1.UpdateOptions{})
		if err != nil {
			fmt.Printf("❌ Update Deployment error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "回滚 Deployment 失败: " + err.Error(),
			})
			return
		}

		fmt.Printf("✅ Deployment rolled back successfully\n")

	default:
		fmt.Printf("❌ Unsupported workload type: %s\n", req.Type)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "不支持的工作负载类型: " + req.Type,
		})
		return
	}

	fmt.Printf("✅ Sending success response\n")
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "回滚成功",
		"data": gin.H{
			"revision": req.Revision,
		},
	})
}

// BatchWorkloadsRequest 批量工作负载操作请求
type BatchWorkloadsRequest struct {
	ClusterID  uint                  `json:"clusterId" binding:"required"`
	Workloads []WorkloadItem        `json:"workloads" binding:"required"`
}

// WorkloadItem 工作负载项
type WorkloadItem struct {
	Namespace string `json:"namespace" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Type      string `json:"type" binding:"required"` // Deployment, StatefulSet, DaemonSet, Job, Pod
}

// BatchWorkloadResult 批量工作负载操作结果
type BatchWorkloadResult struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
}

// BatchDeleteWorkloads 批量删除工作负载
func (h *ResourceHandler) BatchDeleteWorkloads(c *gin.Context) {
	var req BatchWorkloadsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	if len(req.Workloads) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "工作负载列表不能为空",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 获取clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), req.ClusterID, currentUserID.(uint))
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	results := make([]BatchWorkloadResult, 0, len(req.Workloads))

	for _, item := range req.Workloads {
		var err error

		switch item.Type {
		case "Deployment":
			err = clientset.AppsV1().Deployments(item.Namespace).Delete(c.Request.Context(), item.Name, metav1.DeleteOptions{})
		case "StatefulSet":
			err = clientset.AppsV1().StatefulSets(item.Namespace).Delete(c.Request.Context(), item.Name, metav1.DeleteOptions{})
		case "DaemonSet":
			err = clientset.AppsV1().DaemonSets(item.Namespace).Delete(c.Request.Context(), item.Name, metav1.DeleteOptions{})
		case "Job":
			err = clientset.BatchV1().Jobs(item.Namespace).Delete(c.Request.Context(), item.Name, metav1.DeleteOptions{})
		case "Pod":
			err = clientset.CoreV1().Pods(item.Namespace).Delete(c.Request.Context(), item.Name, metav1.DeleteOptions{})
		default:
			err = fmt.Errorf("不支持的工作负载类型: %s", item.Type)
		}

		if err != nil {
			results = append(results, BatchWorkloadResult{
				Namespace: item.Namespace,
				Name:      item.Name,
				Type:      item.Type,
				Success:   false,
				Message:   err.Error(),
			})
		} else {
			results = append(results, BatchWorkloadResult{
				Namespace: item.Namespace,
				Name:      item.Name,
				Type:      item.Type,
				Success:   true,
				Message:   "删除成功",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "批量删除操作完成",
		"data": gin.H{
			"results": results,
		},
	})
}

// BatchRestartWorkloads 批量重启工作负载
func (h *ResourceHandler) BatchRestartWorkloads(c *gin.Context) {
	var req BatchWorkloadsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	if len(req.Workloads) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "工作负载列表不能为空",
		})
		return
	}

	// 添加调试日志
	fmt.Printf("🔄 [BatchRestartWorkloads] 收到重启请求，集群ID: %d，工作负载数量: %d\n", req.ClusterID, len(req.Workloads))
	for i, w := range req.Workloads {
		fmt.Printf("  [%d] Namespace: %s, Name: %s, Type: %s\n", i+1, w.Namespace, w.Name, w.Type)
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 获取clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), req.ClusterID, currentUserID.(uint))
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	results := make([]BatchWorkloadResult, 0, len(req.Workloads))

	for _, item := range req.Workloads {
		var err error

		// 重启策略：对于支持重启的工作负载，通过触发滚动更新来重启
		// 对于 Pod，直接删除并让它重新创建

		switch item.Type {
		case "Deployment":
			// 通过更新 annotations 触发滚动更新
			fmt.Printf("  🔄 正在重启 Deployment: %s/%s\n", item.Namespace, item.Name)
			deployment, getErr := clientset.AppsV1().Deployments(item.Namespace).Get(c.Request.Context(), item.Name, metav1.GetOptions{})
			if getErr == nil {
				if deployment.Spec.Template.Annotations == nil {
					deployment.Spec.Template.Annotations = make(map[string]string)
				}
				// 添加时间戳触发更新
				deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)
				_, err = clientset.AppsV1().Deployments(item.Namespace).Update(c.Request.Context(), deployment, metav1.UpdateOptions{})
				fmt.Printf("  ✅ Deployment %s/%s 重启%v\n", item.Namespace, item.Name, map[bool]string{true: "成功", false: "失败"}[err == nil])
			} else {
				err = getErr
				fmt.Printf("  ❌ Deployment %s/%s 获取失败: %v\n", item.Namespace, item.Name, getErr)
			}

		case "StatefulSet":
			// StatefulSet 也支持同样的重启方式
			statefulSet, getErr := clientset.AppsV1().StatefulSets(item.Namespace).Get(c.Request.Context(), item.Name, metav1.GetOptions{})
			if getErr == nil {
				if statefulSet.Spec.Template.Annotations == nil {
					statefulSet.Spec.Template.Annotations = make(map[string]string)
				}
				statefulSet.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)
				_, err = clientset.AppsV1().StatefulSets(item.Namespace).Update(c.Request.Context(), statefulSet, metav1.UpdateOptions{})
			} else {
				err = getErr
			}

		case "DaemonSet":
			// DaemonSet 也支持同样的重启方式
			daemonSet, getErr := clientset.AppsV1().DaemonSets(item.Namespace).Get(c.Request.Context(), item.Name, metav1.GetOptions{})
			if getErr == nil {
				if daemonSet.Spec.Template.Annotations == nil {
					daemonSet.Spec.Template.Annotations = make(map[string]string)
				}
				daemonSet.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)
				_, err = clientset.AppsV1().DaemonSets(item.Namespace).Update(c.Request.Context(), daemonSet, metav1.UpdateOptions{})
			} else {
				err = getErr
			}

		case "Pod":
			// Pod 通过删除让它重新创建（如果由控制器管理）
			// 或者记录不支持单独重启
			err = fmt.Errorf("Pod 不支持直接重启，请删除后由控制器重建")

		case "Job":
			// Job 不支持重启，只能重新创建
			err = fmt.Errorf("Job 不支持重启，请删除后重新创建")

		default:
			err = fmt.Errorf("不支持的工作负载类型: %s", item.Type)
		}

		if err != nil {
			results = append(results, BatchWorkloadResult{
				Namespace: item.Namespace,
				Name:      item.Name,
				Type:      item.Type,
				Success:   false,
				Message:   err.Error(),
			})
		} else {
			results = append(results, BatchWorkloadResult{
				Namespace: item.Namespace,
				Name:      item.Name,
				Type:      item.Type,
				Success:   true,
				Message:   "重启成功",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "批量重启操作完成",
		"data": gin.H{
			"results": results,
		},
	})
}

// BatchPauseWorkloads 批量停止工作负载
func (h *ResourceHandler) BatchPauseWorkloads(c *gin.Context) {
	var req BatchWorkloadsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	if len(req.Workloads) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "工作负载列表不能为空",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 获取clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), req.ClusterID, currentUserID.(uint))
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	results := make([]BatchWorkloadResult, 0, len(req.Workloads))

	for _, item := range req.Workloads {
		var err error

		switch item.Type {
		case "Deployment":
			deployment, err := clientset.AppsV1().Deployments(item.Namespace).Get(c.Request.Context(), item.Name, metav1.GetOptions{})
			if err == nil {
				deployment.Spec.Paused = true
				_, err = clientset.AppsV1().Deployments(item.Namespace).Update(c.Request.Context(), deployment, metav1.UpdateOptions{})
			}

		case "StatefulSet":
			// StatefulSet 不支持 paused，需要通过将副本数设为0来实现"停止"
			statefulSet, err := clientset.AppsV1().StatefulSets(item.Namespace).Get(c.Request.Context(), item.Name, metav1.GetOptions{})
			if err == nil {
				// 保存原始副本数
				if statefulSet.Annotations == nil {
					statefulSet.Annotations = make(map[string]string)
				}
				statefulSet.Annotations["opshub/original-replicas"] = fmt.Sprintf("%d", *statefulSet.Spec.Replicas)
				zero := int32(0)
				statefulSet.Spec.Replicas = &zero
				_, err = clientset.AppsV1().StatefulSets(item.Namespace).Update(c.Request.Context(), statefulSet, metav1.UpdateOptions{})
			}

		case "DaemonSet":
			// DaemonSet 不支持停止
			err = fmt.Errorf("DaemonSet 不支持停止操作")

		case "Job", "Pod":
			err = fmt.Errorf("%s 不支持停止操作", item.Type)

		default:
			err = fmt.Errorf("不支持的工作负载类型: %s", item.Type)
		}

		if err != nil {
			results = append(results, BatchWorkloadResult{
				Namespace: item.Namespace,
				Name:      item.Name,
				Type:      item.Type,
				Success:   false,
				Message:   err.Error(),
			})
		} else {
			results = append(results, BatchWorkloadResult{
				Namespace: item.Namespace,
				Name:      item.Name,
				Type:      item.Type,
				Success:   true,
				Message:   "停止成功",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "批量停止操作完成",
		"data": gin.H{
			"results": results,
		},
	})
}

// BatchResumeWorkloads 批量启动工作负载
func (h *ResourceHandler) BatchResumeWorkloads(c *gin.Context) {
	var req BatchWorkloadsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	if len(req.Workloads) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "工作负载列表不能为空",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	// 获取clientset
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), req.ClusterID, currentUserID.(uint))
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	results := make([]BatchWorkloadResult, 0, len(req.Workloads))

	for _, item := range req.Workloads {
		var err error

		switch item.Type {
		case "Deployment":
			deployment, err := clientset.AppsV1().Deployments(item.Namespace).Get(c.Request.Context(), item.Name, metav1.GetOptions{})
			if err == nil {
				deployment.Spec.Paused = false
				_, err = clientset.AppsV1().Deployments(item.Namespace).Update(c.Request.Context(), deployment, metav1.UpdateOptions{})
			}

		case "StatefulSet":
			// StatefulSet 恢复副本数
			statefulSet, err := clientset.AppsV1().StatefulSets(item.Namespace).Get(c.Request.Context(), item.Name, metav1.GetOptions{})
			if err == nil {
				// 尝试从 annotations 获取原始副本数
				if originalReplicasStr, ok := statefulSet.Annotations["opshub/original-replicas"]; ok {
					originalReplicas, err := strconv.ParseInt(originalReplicasStr, 10, 32)
					if err == nil && originalReplicas > 0 {
						replicas := int32(originalReplicas)
						statefulSet.Spec.Replicas = &replicas
					}
				} else {
					// 如果没有保存原始副本数，默认设为1
					one := int32(1)
					statefulSet.Spec.Replicas = &one
				}
				delete(statefulSet.Annotations, "opshub/original-replicas")
				_, err = clientset.AppsV1().StatefulSets(item.Namespace).Update(c.Request.Context(), statefulSet, metav1.UpdateOptions{})
			}

		case "DaemonSet", "Job", "Pod":
			err = fmt.Errorf("%s 不支持启动操作", item.Type)

		default:
			err = fmt.Errorf("不支持的工作负载类型: %s", item.Type)
		}

		if err != nil {
			results = append(results, BatchWorkloadResult{
				Namespace: item.Namespace,
				Name:      item.Name,
				Type:      item.Type,
				Success:   false,
				Message:   err.Error(),
			})
		} else {
			results = append(results, BatchWorkloadResult{
				Namespace: item.Namespace,
				Name:      item.Name,
				Type:      item.Type,
				Success:   true,
				Message:   "启动成功",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "批量启动操作完成",
		"data": gin.H{
			"results": results,
		},
	})
}

// FileInfo 文件信息
type FileInfo struct {
	Name    string `json:"name"`
	Size    string `json:"size"`
	Mode    string `json:"mode"`
	IsDir   bool   `json:"isDir"`
	ModTime string `json:"modTime"`
	User    string `json:"user"`
	Group   string `json:"group"`
	Link    string `json:"link"`
	Path    string `json:"path"`
}

// GetPodDetail 获取 Pod 详情
func (h *ResourceHandler) GetPodDetail(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	namespace := c.Param("namespace")
	podName := c.Param("name")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	pod, err := clientset.CoreV1().Pods(namespace).Get(c.Request.Context(), podName, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取 Pod 失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    pod,
	})
}

// GetPodEvents 获取 Pod 事件
func (h *ResourceHandler) GetPodEvents(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	namespace := c.Param("namespace")
	podName := c.Param("name")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未授权",
		})
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	events, err := clientset.CoreV1().Events(namespace).List(c.Request.Context(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", podName),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取事件失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events.Items,
	})
}

// ListContainerFiles 列出容器文件
func (h *ResourceHandler) ListContainerFiles(c *gin.Context) {
	clusterIDStr := c.Query("cluster_id")
	if clusterIDStr == "" {
		clusterIDStr = c.Query("clusterId")
	}
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"msg":  "无效的集群ID",
		})
		return
	}

	namespace := c.Query("namespace")
	podName := c.Query("podName")
	containerName := c.Query("containerName")
	path := c.Query("path")

	if namespace == "" || podName == "" || containerName == "" {
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"msg":  "缺少必要参数",
		})
		return
	}

	if path == "" {
		path = "/"
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	restConfig, err := h.clusterService.GetRESTConfig(uint(clusterID), currentUserID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "获取集群配置失败: " + err.Error(),
		})
		return
	}

	serverURL, err := url.Parse(restConfig.Host)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "解析集群 URL 失败: " + err.Error(),
		})
		return
	}

	query := url.Values{}
	query.Set("container", containerName)
	query.Set("stdin", "true")
	query.Set("stdout", "true")
	query.Set("stderr", "true")
	query.Set("tty", "false")

	cmdStr := fmt.Sprintf("ls -la '%s'", path)
	query.Add("command", "sh")
	query.Add("command", "-c")
	query.Add("command", cmdStr)

	execURL := &url.URL{
		Scheme:   serverURL.Scheme,
		Host:     serverURL.Host,
		Path:     fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/exec", namespace, podName),
		RawQuery: query.Encode(),
	}

	executor, err := remotecommand.NewSPDYExecutor(restConfig, "POST", execURL)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "创建 executor 失败: " + err.Error(),
		})
		return
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()

	go func() {
		pw.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  pr,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "执行命令失败: " + err.Error(),
		})
		return
	}

	output := stdout.String()
	files := parseLsOutput(output)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"files": files,
		},
		"msg": "获取成功",
	})
}

// DownloadContainerFile 下载容器文件
func (h *ResourceHandler) DownloadContainerFile(c *gin.Context) {
	clusterIDStr := c.Query("cluster_id")
	if clusterIDStr == "" {
		clusterIDStr = c.Query("clusterId")
	}
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"msg":  "无效的集群ID",
		})
		return
	}

	namespace := c.Query("namespace")
	podName := c.Query("podName")
	containerName := c.Query("containerName")
	path := c.Query("path")

	if namespace == "" || podName == "" || containerName == "" || path == "" {
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"msg":  "缺少必要参数",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	restConfig, err := h.clusterService.GetRESTConfig(uint(clusterID), currentUserID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "获取集群配置失败: " + err.Error(),
		})
		return
	}

	serverURL, err := url.Parse(restConfig.Host)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "解析集群 URL 失败: " + err.Error(),
		})
		return
	}

	query := url.Values{}
	query.Set("container", containerName)
	query.Set("stdin", "true")
	query.Set("stdout", "true")
	query.Set("stderr", "true")
	query.Set("tty", "false")

	cmdStr := fmt.Sprintf("cat '%s' | base64", path)
	query.Add("command", "sh")
	query.Add("command", "-c")
	query.Add("command", cmdStr)

	execURL := &url.URL{
		Scheme:   serverURL.Scheme,
		Host:     serverURL.Host,
		Path:     fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/exec", namespace, podName),
		RawQuery: query.Encode(),
	}

	executor, err := remotecommand.NewSPDYExecutor(restConfig, "POST", execURL)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "创建 executor 失败: " + err.Error(),
		})
		return
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()

	go func() {
		pw.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  pr,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "读取文件失败: " + err.Error(),
		})
		return
	}

	decodedData, err := base64.StdEncoding.DecodeString(stdout.String())
	if err != nil {
		decodedData = stdout.Bytes()
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(path)))
	c.Data(http.StatusOK, "application/octet-stream", decodedData)
}

// UploadContainerFile 上传文件到容器
func (h *ResourceHandler) UploadContainerFile(c *gin.Context) {
	clusterIDStr := c.PostForm("cluster_id")
	if clusterIDStr == "" {
		clusterIDStr = c.PostForm("clusterId")
	}
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"msg":  "无效的集群ID",
		})
		return
	}

	namespace := c.PostForm("namespace")
	podName := c.PostForm("podName")
	containerName := c.PostForm("containerName")
	path := c.PostForm("path")

	if namespace == "" || podName == "" || containerName == "" || path == "" {
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"msg":  "缺少必要参数",
		})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"msg":  "获取上传文件失败: " + err.Error(),
		})
		return
	}

	// 构造完整的文件路径（目录 + 文件名）
	fileName := filepath.Base(file.Filename)
	targetPath := path
	if !strings.HasSuffix(targetPath, "/") {
		targetPath += "/"
	}
	targetPath += fileName

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "打开文件失败: " + err.Error(),
		})
		return
	}
	defer src.Close()

	fileData, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "读取文件失败: " + err.Error(),
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	restConfig, err := h.clusterService.GetRESTConfig(uint(clusterID), currentUserID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "获取集群配置失败: " + err.Error(),
		})
		return
	}

	serverURL, err := url.Parse(restConfig.Host)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "解析集群 URL 失败: " + err.Error(),
		})
		return
	}

	query := url.Values{}
	query.Set("container", containerName)
	query.Set("stdin", "true")
	query.Set("stdout", "true")
	query.Set("stderr", "true")
	query.Set("tty", "false")

	// 使用 base64 -d 从 stdin 读取并解码到文件
	cmdStr := fmt.Sprintf("base64 -d > '%s'", targetPath)
	query.Add("command", "sh")
	query.Add("command", "-c")
	query.Add("command", cmdStr)

	execURL := &url.URL{
		Scheme:   serverURL.Scheme,
		Host:     serverURL.Host,
		Path:     fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/exec", namespace, podName),
		RawQuery: query.Encode(),
	}

	executor, err := remotecommand.NewSPDYExecutor(restConfig, "POST", execURL)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "创建 executor 失败: " + err.Error(),
		})
		return
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// 创建 pipe 用于 stdin
	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()

	// 在 goroutine 中写入 base64 编码的文件内容到 stdin
	go func() {
		encodedContent := base64.StdEncoding.EncodeToString(fileData)
		pw.Write([]byte(encodedContent))
		pw.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  pr,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "上传文件失败: " + err.Error() + ", stderr: " + stderr.String(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "文件上传成功",
	})
}

// parseLsOutput 解析 ls -la 命令的输出
func parseLsOutput(output string) []FileInfo {
	lines := strings.Split(output, "\n")
	var files []FileInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "total") || strings.HasPrefix(line, "ERROR:") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		fileInfo := FileInfo{
			Mode:  fields[0],
			User:  fields[2],
			Group: fields[3],
			Size:  fields[4],
		}

		fileInfo.IsDir = strings.HasPrefix(fileInfo.Mode, "d")

		nameAndLink := strings.Join(fields[8:], " ")

		if idx := strings.Index(nameAndLink, " -> "); idx > 0 {
			fileInfo.Name = nameAndLink[:idx]
			fileInfo.Link = nameAndLink[idx+4:]
		} else {
			fileInfo.Name = nameAndLink
		}

		// 跳过 . 和 .. 目录
		if fileInfo.Name == "." || fileInfo.Name == ".." {
			continue
		}

		if len(fields) >= 8 {
			timeStr := strings.Join(fields[5:8], " ")
			fileInfo.ModTime = timeStr
		}

		files = append(files, fileInfo)
	}

	return files
}

// ==================== Storage 相关 ====================

// PVCInfo PersistentVolumeClaim 信息
type PVCInfo struct {
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace"`
	Status       string            `json:"status"`
	Capacity     string            `json:"capacity"`
	AccessModes  []string          `json:"accessModes"`
	StorageClass string            `json:"storageClass"`
	VolumeName   string            `json:"volumeName"`
	Age          string            `json:"age"`
	Labels       map[string]string `json:"labels"`
}

// PVInfo PersistentVolume 信息
type PVInfo struct {
	Name          string            `json:"name"`
	Capacity      string            `json:"capacity"`
	AccessModes   []string          `json:"accessModes"`
	ReclaimPolicy string            `json:"reclaimPolicy"`
	Status        string            `json:"status"`
	Claim         string            `json:"claim"`
	StorageClass  string            `json:"storageClass"`
	Reason        string            `json:"reason"`
	Age           string            `json:"age"`
	Labels        map[string]string `json:"labels"`
}

// StorageClassInfo StorageClass 信息
type StorageClassInfo struct {
	Name                 string            `json:"name"`
	Provisioner          string            `json:"provisioner"`
	ReclaimPolicy        string            `json:"reclaimPolicy"`
	VolumeBindingMode    string            `json:"volumeBindingMode"`
	AllowVolumeExpansion bool              `json:"allowVolumeExpansion"`
	Age                  string            `json:"age"`
	Labels               map[string]string `json:"labels"`
}

// ListPersistentVolumeClaims 获取 PVC 列表
// @Summary 获取PVC列表
// @Description 获取指定命名空间的持久卷声明列表
// @Tags Kubernetes/存储
// @Accept json
// @Produce json
// @Security Bearer
// @Param clusterId query int true "集群ID"
// @Param namespace query string false "命名空间"
// @Success 200 {object} map[string]interface{} "PVC列表"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/resources/persistentvolumeclaims [get]
func (h *ResourceHandler) ListPersistentVolumeClaims(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = v1.NamespaceAll
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "PVC")
		return
	}

	result := make([]PVCInfo, 0, len(pvcs.Items))
	for _, pvc := range pvcs.Items {
		accessModes := make([]string, len(pvc.Spec.AccessModes))
		for i, am := range pvc.Spec.AccessModes {
			accessModes[i] = string(am)
		}

		capacity := ""
		if pvc.Spec.Resources.Requests != nil {
			if storage, ok := pvc.Spec.Resources.Requests[v1.ResourceStorage]; ok {
				capacity = storage.String()
			}
		}

		storageClass := ""
		if pvc.Spec.StorageClassName != nil {
			storageClass = *pvc.Spec.StorageClassName
		}

		volumeName := ""
		if pvc.Spec.VolumeName != "" {
			volumeName = pvc.Spec.VolumeName
		}

		claim := PVCInfo{
			Name:         pvc.Name,
			Namespace:    pvc.Namespace,
			Status:       string(pvc.Status.Phase),
			Capacity:     capacity,
			AccessModes:  accessModes,
			StorageClass: storageClass,
			VolumeName:   volumeName,
			Age:          calculateAge(pvc.CreationTimestamp.Time),
			Labels:       pvc.Labels,
		}
		result = append(result, claim)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data":    result,
	})
}

// GetPersistentVolumeClaimYAML 获取 PVC YAML
func (h *ResourceHandler) GetPersistentVolumeClaimYAML(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	pvc, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "PVC")
		return
	}

	// 使用清理函数移除不需要的字段
	cleaned := cleanPersistentVolumeClaimForYAML(pvc)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data":    cleaned,
	})
}

// UpdatePersistentVolumeClaimYAML 更新 PVC YAML
func (h *ResourceHandler) UpdatePersistentVolumeClaimYAML(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	var jsonData map[string]interface{}
	if err := c.ShouldBindJSON(&jsonData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求数据: " + err.Error(),
		})
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	existingPVC, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "PVC")
		return
	}

	var pvc v1.PersistentVolumeClaim
	pvcData, err := json.Marshal(jsonData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "序列化数据失败: " + err.Error(),
		})
		return
	}

	err = json.Unmarshal(pvcData, &pvc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "解析 PVC 数据失败: " + err.Error(),
		})
		return
	}

	pvc.ResourceVersion = existingPVC.ResourceVersion

	_, err = clientset.CoreV1().PersistentVolumeClaims(namespace).Update(c.Request.Context(), &pvc, metav1.UpdateOptions{})
	if err != nil {
		HandleK8sError(c, err, "PVC")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
	})
}

// DeletePersistentVolumeClaim 删除 PVC
func (h *ResourceHandler) DeletePersistentVolumeClaim(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	err = clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		HandleK8sError(c, err, "PVC")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// CreatePersistentVolumeClaimYAML 通过 YAML 创建 PVC
func (h *ResourceHandler) CreatePersistentVolumeClaimYAML(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Param("namespace")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	var jsonData map[string]interface{}
	if err := c.ShouldBindJSON(&jsonData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求数据: " + err.Error(),
		})
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	var pvc v1.PersistentVolumeClaim
	pvcData, err := json.Marshal(jsonData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "序列化数据失败: " + err.Error(),
		})
		return
	}

	err = json.Unmarshal(pvcData, &pvc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "解析 PVC 数据失败: " + err.Error(),
		})
		return
	}

	_, err = clientset.CoreV1().PersistentVolumeClaims(namespace).Create(c.Request.Context(), &pvc, metav1.CreateOptions{})
	if err != nil {
		HandleK8sError(c, err, "PVC")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
	})
}

// ListPersistentVolumes 获取 PV 列表
// @Summary 获取PV列表
// @Description 获取集群的持久卷列表
// @Tags Kubernetes/存储
// @Accept json
// @Produce json
// @Security Bearer
// @Param clusterId query int true "集群ID"
// @Success 200 {object} map[string]interface{} "PV列表"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/resources/persistentvolumes [get]
func (h *ResourceHandler) ListPersistentVolumes(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	pvs, err := clientset.CoreV1().PersistentVolumes().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "PV")
		return
	}

	result := make([]PVInfo, 0, len(pvs.Items))
	for _, pv := range pvs.Items {
		accessModes := make([]string, len(pv.Spec.AccessModes))
		for i, am := range pv.Spec.AccessModes {
			accessModes[i] = string(am)
		}

		capacity := ""
		if pv.Spec.Capacity != nil {
			if storage, ok := pv.Spec.Capacity[v1.ResourceStorage]; ok {
				capacity = storage.String()
			}
		}

		reclaimPolicy := string(pv.Spec.PersistentVolumeReclaimPolicy)

		claim := ""
		if pv.Spec.ClaimRef != nil {
			claim = pv.Spec.ClaimRef.Namespace + "/" + pv.Spec.ClaimRef.Name
		}

		storageClass := pv.Spec.StorageClassName

		reason := ""
		if pv.Status.Reason != "" {
			reason = pv.Status.Reason
		}

		volume := PVInfo{
			Name:          pv.Name,
			Capacity:      capacity,
			AccessModes:   accessModes,
			ReclaimPolicy: reclaimPolicy,
			Status:        string(pv.Status.Phase),
			Claim:         claim,
			StorageClass:  storageClass,
			Reason:        reason,
			Age:           calculateAge(pv.CreationTimestamp.Time),
			Labels:        pv.Labels,
		}
		result = append(result, volume)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data":    result,
	})
}

// GetPersistentVolumeYAML 获取 PV YAML
func (h *ResourceHandler) GetPersistentVolumeYAML(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	name := c.Param("name")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	pv, err := clientset.CoreV1().PersistentVolumes().Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "PV")
		return
	}

	// 使用清理函数移除不需要的字段
	cleaned := cleanPersistentVolumeForYAML(pv)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data":    cleaned,
	})
}

// UpdatePersistentVolumeYAML 更新 PV YAML
func (h *ResourceHandler) UpdatePersistentVolumeYAML(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	name := c.Param("name")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	var jsonData map[string]interface{}
	if err := c.ShouldBindJSON(&jsonData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求数据: " + err.Error(),
		})
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	existingPV, err := clientset.CoreV1().PersistentVolumes().Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "PV")
		return
	}

	var pv v1.PersistentVolume
	pvData, err := json.Marshal(jsonData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "序列化数据失败: " + err.Error(),
		})
		return
	}

	err = json.Unmarshal(pvData, &pv)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "解析 PV 数据失败: " + err.Error(),
		})
		return
	}

	pv.ResourceVersion = existingPV.ResourceVersion

	_, err = clientset.CoreV1().PersistentVolumes().Update(c.Request.Context(), &pv, metav1.UpdateOptions{})
	if err != nil {
		HandleK8sError(c, err, "PV")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
	})
}

// DeletePersistentVolume 删除 PV
func (h *ResourceHandler) DeletePersistentVolume(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	name := c.Param("name")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	err = clientset.CoreV1().PersistentVolumes().Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		HandleK8sError(c, err, "PV")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// CreatePersistentVolumeYAML 通过 YAML 创建 PV
func (h *ResourceHandler) CreatePersistentVolumeYAML(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	var jsonData map[string]interface{}
	if err := c.ShouldBindJSON(&jsonData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求数据: " + err.Error(),
		})
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	var pv v1.PersistentVolume
	pvData, err := json.Marshal(jsonData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "序列化数据失败: " + err.Error(),
		})
		return
	}

	err = json.Unmarshal(pvData, &pv)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "解析 PV 数据失败: " + err.Error(),
		})
		return
	}

	_, err = clientset.CoreV1().PersistentVolumes().Create(c.Request.Context(), &pv, metav1.CreateOptions{})
	if err != nil {
		HandleK8sError(c, err, "PV")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
	})
}

// ListStorageClasses 获取 StorageClass 列表
// @Summary 获取StorageClass列表
// @Description 获取集群的存储类列表
// @Tags Kubernetes/存储
// @Accept json
// @Produce json
// @Security Bearer
// @Param clusterId query int true "集群ID"
// @Success 200 {object} map[string]interface{} "StorageClass列表"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /plugins/kubernetes/resources/storageclasses [get]
func (h *ResourceHandler) ListStorageClasses(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	scs, err := clientset.StorageV1().StorageClasses().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "StorageClass")
		return
	}

	result := make([]StorageClassInfo, 0, len(scs.Items))
	for _, sc := range scs.Items {
		reclaimPolicy := ""
		if sc.ReclaimPolicy != nil {
			reclaimPolicy = string(*sc.ReclaimPolicy)
		}

		volumeBindingMode := ""
		if sc.VolumeBindingMode != nil {
			volumeBindingMode = string(*sc.VolumeBindingMode)
		}

		allowVolumeExpansion := false
		if sc.AllowVolumeExpansion != nil {
			allowVolumeExpansion = *sc.AllowVolumeExpansion
		}

		classInfo := StorageClassInfo{
			Name:                 sc.Name,
			Provisioner:          sc.Provisioner,
			ReclaimPolicy:        reclaimPolicy,
			VolumeBindingMode:    volumeBindingMode,
			AllowVolumeExpansion: allowVolumeExpansion,
			Age:                  calculateAge(sc.CreationTimestamp.Time),
			Labels:               sc.Labels,
		}
		result = append(result, classInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data":    result,
	})
}

// GetStorageClassYAML 获取 StorageClass YAML
func (h *ResourceHandler) GetStorageClassYAML(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	name := c.Param("name")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	sc, err := clientset.StorageV1().StorageClasses().Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "StorageClass")
		return
	}

	// 使用清理函数移除不需要的字段
	cleaned := cleanStorageClassForYAML(sc)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data":    cleaned,
	})
}

// UpdateStorageClassYAML 更新 StorageClass YAML
func (h *ResourceHandler) UpdateStorageClassYAML(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	name := c.Param("name")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	var jsonData map[string]interface{}
	if err := c.ShouldBindJSON(&jsonData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求数据: " + err.Error(),
		})
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	existingSC, err := clientset.StorageV1().StorageClasses().Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "StorageClass")
		return
	}

	var sc storagev1.StorageClass
	scData, err := json.Marshal(jsonData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "序列化数据失败: " + err.Error(),
		})
		return
	}

	err = json.Unmarshal(scData, &sc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "解析 StorageClass 数据失败: " + err.Error(),
		})
		return
	}

	sc.ResourceVersion = existingSC.ResourceVersion

	_, err = clientset.StorageV1().StorageClasses().Update(c.Request.Context(), &sc, metav1.UpdateOptions{})
	if err != nil {
		HandleK8sError(c, err, "StorageClass")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
	})
}

// DeleteStorageClass 删除 StorageClass
func (h *ResourceHandler) DeleteStorageClass(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	name := c.Param("name")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	err = clientset.StorageV1().StorageClasses().Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		HandleK8sError(c, err, "StorageClass")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// CreateStorageClassYAML 通过 YAML 创建 StorageClass
func (h *ResourceHandler) CreateStorageClassYAML(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	var jsonData map[string]interface{}
	if err := c.ShouldBindJSON(&jsonData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求数据: " + err.Error(),
		})
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	var sc storagev1.StorageClass
	scData, err := json.Marshal(jsonData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "序列化数据失败: " + err.Error(),
		})
		return
	}

	err = json.Unmarshal(scData, &sc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "解析 StorageClass 数据失败: " + err.Error(),
		})
		return
	}

	_, err = clientset.StorageV1().StorageClasses().Create(c.Request.Context(), &sc, metav1.CreateOptions{})
	if err != nil {
		HandleK8sError(c, err, "StorageClass")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
	})
}

// ==================== ResourceQuota 相关 ====================

// ResourceQuotaInfo ResourceQuota 信息
type ResourceQuotaInfo struct {
	Name           string `json:"name"`
	Namespace      string `json:"namespace"`
	RequestsCpu    string `json:"requestsCpu"`
	RequestsMemory string `json:"requestsMemory"`
	LimitsCpu      string `json:"limitsCpu"`
	LimitsMemory   string `json:"limitsMemory"`
	Age            string `json:"age"`
	CreatedAt      string `json:"createdAt"`
}

// ListResourceQuotas 获取 ResourceQuota 列表
func (h *ResourceHandler) ListResourceQuotas(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = v1.NamespaceAll
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	quotas, err := clientset.CoreV1().ResourceQuotas(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "ResourceQuota")
		return
	}

	quotaInfos := make([]ResourceQuotaInfo, 0, len(quotas.Items))
	for _, q := range quotas.Items {
		info := ResourceQuotaInfo{
			Name:      q.Name,
			Namespace: q.Namespace,
			Age:       calculateAge(q.CreationTimestamp.Time),
			CreatedAt: q.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}

		if q.Spec.Hard != nil {
			if cpu, ok := q.Spec.Hard[v1.ResourceCPU]; ok {
				if req, ok := q.Status.Used[v1.ResourceCPU]; ok {
					info.RequestsCpu = fmt.Sprintf("%s/%s", req.String(), cpu.String())
				} else {
					info.RequestsCpu = cpu.String()
				}
			}
			if mem, ok := q.Spec.Hard[v1.ResourceMemory]; ok {
				if req, ok := q.Status.Used[v1.ResourceMemory]; ok {
					info.RequestsMemory = fmt.Sprintf("%s/%s", req.String(), mem.String())
				} else {
					info.RequestsMemory = mem.String()
				}
			}
			if limitsCpu, ok := q.Spec.Hard[v1.ResourceLimitsCPU]; ok {
				if used, ok := q.Status.Used[v1.ResourceLimitsCPU]; ok {
					info.LimitsCpu = fmt.Sprintf("%s/%s", used.String(), limitsCpu.String())
				} else {
					info.LimitsCpu = limitsCpu.String()
				}
			}
			if limitsMem, ok := q.Spec.Hard[v1.ResourceLimitsMemory]; ok {
				if used, ok := q.Status.Used[v1.ResourceLimitsMemory]; ok {
					info.LimitsMemory = fmt.Sprintf("%s/%s", used.String(), limitsMem.String())
				} else {
					info.LimitsMemory = limitsMem.String()
				}
			}
		}

		quotaInfos = append(quotaInfos, info)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    quotaInfos,
	})
}

// GetResourceQuotaYAML 获取 ResourceQuota YAML
func (h *ResourceHandler) GetResourceQuotaYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	quota, err := clientset.CoreV1().ResourceQuotas(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "ResourceQuota")
		return
	}

	yamlData, err := yaml.Marshal(quota)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "YAML 转换失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"yaml": string(yamlData),
		},
	})
}

// UpdateResourceQuotaYAML 更新 ResourceQuota YAML
func (h *ResourceHandler) UpdateResourceQuotaYAML(c *gin.Context) {
	var req struct {
		ClusterID int    `json:"clusterId"`
		YAML      string `json:"yaml"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	var quota v1.ResourceQuota
	if err := yaml.Unmarshal([]byte(req.YAML), &quota); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析 YAML 失败: " + err.Error(),
		})
		return
	}

	quota.Name = name
	quota.Namespace = namespace

	_, err = clientset.CoreV1().ResourceQuotas(namespace).Update(c.Request.Context(), &quota, metav1.UpdateOptions{})
	if err != nil {
		HandleK8sError(c, err, "ResourceQuota")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// DeleteResourceQuota 删除 ResourceQuota
func (h *ResourceHandler) DeleteResourceQuota(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	err = clientset.CoreV1().ResourceQuotas(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		HandleK8sError(c, err, "ResourceQuota")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// CreateResourceQuotaFromYAML 从 YAML 创建 ResourceQuota
func (h *ResourceHandler) CreateResourceQuotaFromYAML(c *gin.Context) {
	var req struct {
		ClusterID int    `json:"clusterId"`
		YAML      string `json:"yaml"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	namespace := c.Param("namespace")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	var quota v1.ResourceQuota
	if err := yaml.Unmarshal([]byte(req.YAML), &quota); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析 YAML 失败: " + err.Error(),
		})
		return
	}

	quota.Namespace = namespace

	_, err = clientset.CoreV1().ResourceQuotas(namespace).Create(c.Request.Context(), &quota, metav1.CreateOptions{})
	if err != nil {
		HandleK8sError(c, err, "ResourceQuota")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// ==================== LimitRange 相关 ====================

// LimitRangeInfo LimitRange 信息
type LimitRangeInfo struct {
	Name                 string `json:"name"`
	Namespace            string `json:"namespace"`
	Type                 string `json:"type"`
	Resource             string `json:"resource"`
	Min                  string `json:"min"`
	Max                  string `json:"max"`
	DefaultLimit         string `json:"defaultLimit"`
	DefaultRequest       string `json:"defaultRequest"`
	MaxLimitRequestRatio string `json:"maxLimitRequestRatio"`
	Age                  string `json:"age"`
}

// ListLimitRanges 获取 LimitRange 列表
func (h *ResourceHandler) ListLimitRanges(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = v1.NamespaceAll
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	limitRanges, err := clientset.CoreV1().LimitRanges(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "LimitRange")
		return
	}

	limitRangeInfos := make([]LimitRangeInfo, 0)
	for _, lr := range limitRanges.Items {
		info := LimitRangeInfo{
			Name:      lr.Name,
			Namespace: lr.Namespace,
			Age:       calculateAge(lr.CreationTimestamp.Time),
		}

		for _, limit := range lr.Spec.Limits {
			info.Type = string(limit.Type)

			if limit.Min != nil {
				for res, qty := range limit.Min {
					info.Resource = string(res)
					info.Min = qty.String()
					break
				}
			}

			if limit.Max != nil {
				for res, qty := range limit.Max {
					info.Resource = string(res)
					info.Max = qty.String()
					break
				}
			}

			if limit.Default != nil {
				for res, qty := range limit.Default {
					info.Resource = string(res)
					info.DefaultLimit = qty.String()
					break
				}
			}

			if limit.DefaultRequest != nil {
				for res, qty := range limit.DefaultRequest {
					info.Resource = string(res)
					info.DefaultRequest = qty.String()
					break
				}
			}

			if limit.MaxLimitRequestRatio != nil {
				for res, qty := range limit.MaxLimitRequestRatio {
					info.Resource = string(res)
					info.MaxLimitRequestRatio = qty.String()
					break
				}
			}

			limitRangeInfos = append(limitRangeInfos, info)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    limitRangeInfos,
	})
}

// GetLimitRangeYAML 获取 LimitRange YAML
func (h *ResourceHandler) GetLimitRangeYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	limitRange, err := clientset.CoreV1().LimitRanges(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "LimitRange")
		return
	}

	yamlData, err := yaml.Marshal(limitRange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "YAML 转换失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"yaml": string(yamlData),
		},
	})
}

// UpdateLimitRangeYAML 更新 LimitRange YAML
func (h *ResourceHandler) UpdateLimitRangeYAML(c *gin.Context) {
	var req struct {
		ClusterID int    `json:"clusterId"`
		YAML      string `json:"yaml"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	var limitRange v1.LimitRange
	if err := yaml.Unmarshal([]byte(req.YAML), &limitRange); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析 YAML 失败: " + err.Error(),
		})
		return
	}

	limitRange.Name = name
	limitRange.Namespace = namespace

	_, err = clientset.CoreV1().LimitRanges(namespace).Update(c.Request.Context(), &limitRange, metav1.UpdateOptions{})
	if err != nil {
		HandleK8sError(c, err, "LimitRange")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// DeleteLimitRange 删除 LimitRange
func (h *ResourceHandler) DeleteLimitRange(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	err = clientset.CoreV1().LimitRanges(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		HandleK8sError(c, err, "LimitRange")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// CreateLimitRangeFromYAML 从 YAML 创建 LimitRange
func (h *ResourceHandler) CreateLimitRangeFromYAML(c *gin.Context) {
	var req struct {
		ClusterID int    `json:"clusterId"`
		YAML      string `json:"yaml"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	namespace := c.Param("namespace")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	var limitRange v1.LimitRange
	if err := yaml.Unmarshal([]byte(req.YAML), &limitRange); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析 YAML 失败: " + err.Error(),
		})
		return
	}

	limitRange.Namespace = namespace

	_, err = clientset.CoreV1().LimitRanges(namespace).Create(c.Request.Context(), &limitRange, metav1.CreateOptions{})
	if err != nil {
		HandleK8sError(c, err, "LimitRange")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// ==================== HPA 相关 ====================

// HPAInfo HPA 信息
type HPAInfo struct {
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	ReferenceTarget string `json:"referenceTarget"`
	MinReplicas     *int32 `json:"minReplicas"`
	MaxReplicas     int32  `json:"maxReplicas"`
	CurrentReplicas int32  `json:"currentReplicas"`
	TargetCPU       string `json:"targetCPU"`
	TargetMemory    string `json:"targetMemory"`
	Age             string `json:"age"`
	CreatedAt       string `json:"createdAt"`
}

// ListHorizontalPodAutoscalers 获取 HPA 列表
func (h *ResourceHandler) ListHorizontalPodAutoscalers(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = v1.NamespaceAll
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	hpas, err := clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "HPA")
		return
	}

	hpaInfos := make([]HPAInfo, 0, len(hpas.Items))
	for _, hpa := range hpas.Items {
		info := HPAInfo{
			Name:            hpa.Name,
			Namespace:       hpa.Namespace,
			MaxReplicas:     hpa.Spec.MaxReplicas,
			CurrentReplicas: hpa.Status.CurrentReplicas,
			Age:             calculateAge(hpa.CreationTimestamp.Time),
			CreatedAt:       hpa.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}

		// 获取引用目标
		if hpa.Spec.ScaleTargetRef.Name != "" {
			info.ReferenceTarget = fmt.Sprintf("%s/%s", hpa.Spec.ScaleTargetRef.Kind, hpa.Spec.ScaleTargetRef.Name)
		}

		info.MinReplicas = hpa.Spec.MinReplicas

		// v1 API 只支持 CPU 目标
		if hpa.Spec.TargetCPUUtilizationPercentage != nil {
			info.TargetCPU = fmt.Sprintf("%d%%", *hpa.Spec.TargetCPUUtilizationPercentage)
		}

		hpaInfos = append(hpaInfos, info)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    hpaInfos,
	})
}

// GetHorizontalPodAutoscalerYAML 获取 HPA YAML
func (h *ResourceHandler) GetHorizontalPodAutoscalerYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	hpa, err := clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "HPA")
		return
	}

	yamlData, err := yaml.Marshal(hpa)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "YAML 转换失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"yaml": string(yamlData),
		},
	})
}

// UpdateHorizontalPodAutoscalerYAML 更新 HPA YAML
func (h *ResourceHandler) UpdateHorizontalPodAutoscalerYAML(c *gin.Context) {
	var req struct {
		ClusterID int    `json:"clusterId"`
		YAML      string `json:"yaml"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	var hpa autoscalingv1.HorizontalPodAutoscaler
	if err := yaml.Unmarshal([]byte(req.YAML), &hpa); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析 YAML 失败: " + err.Error(),
		})
		return
	}

	hpa.Name = name
	hpa.Namespace = namespace

	_, err = clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Update(c.Request.Context(), &hpa, metav1.UpdateOptions{})
	if err != nil {
		HandleK8sError(c, err, "HPA")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// DeleteHorizontalPodAutoscaler 删除 HPA
func (h *ResourceHandler) DeleteHorizontalPodAutoscaler(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	err = clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		HandleK8sError(c, err, "HPA")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// CreateHPAFromYAML 从 YAML 创建 HPA
func (h *ResourceHandler) CreateHPAFromYAML(c *gin.Context) {
	var req struct {
		ClusterID int    `json:"clusterId"`
		YAML      string `json:"yaml"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	namespace := c.Param("namespace")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	var hpa autoscalingv1.HorizontalPodAutoscaler
	if err := yaml.Unmarshal([]byte(req.YAML), &hpa); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析 YAML 失败: " + err.Error(),
		})
		return
	}

	hpa.Namespace = namespace

	_, err = clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Create(c.Request.Context(), &hpa, metav1.CreateOptions{})
	if err != nil {
		HandleK8sError(c, err, "HPA")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// ==================== PodDisruptionBudget 相关 ====================

// PDBInfo PodDisruptionBudget 信息
type PDBInfo struct {
	Name               string `json:"name"`
	Namespace          string `json:"namespace"`
	MinAvailable       string `json:"minAvailable"`
	MaxUnavailable     string `json:"maxUnavailable"`
	AllowedDisruptions int32  `json:"allowedDisruptions"`
	CurrentHealthy     int32  `json:"currentHealthy"`
	DesiredHealthy     int32  `json:"desiredHealthy"`
	Age                string `json:"age"`
	CreatedAt          string `json:"createdAt"`
}

// ListPodDisruptionBudgets 获取 PodDisruptionBudget 列表
func (h *ResourceHandler) ListPodDisruptionBudgets(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = v1.NamespaceAll
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	pdbs, err := clientset.PolicyV1().PodDisruptionBudgets(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "PodDisruptionBudget")
		return
	}

	pdbInfos := make([]PDBInfo, 0, len(pdbs.Items))
	for _, pdb := range pdbs.Items {
		info := PDBInfo{
			Name:               pdb.Name,
			Namespace:          pdb.Namespace,
			AllowedDisruptions: pdb.Status.DisruptionsAllowed,
			CurrentHealthy:     pdb.Status.CurrentHealthy,
			DesiredHealthy:     pdb.Status.DesiredHealthy,
			Age:                calculateAge(pdb.CreationTimestamp.Time),
			CreatedAt:          pdb.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}

		if pdb.Spec.MinAvailable != nil {
			info.MinAvailable = pdb.Spec.MinAvailable.String()
		}

		if pdb.Spec.MaxUnavailable != nil {
			info.MaxUnavailable = pdb.Spec.MaxUnavailable.String()
		}

		pdbInfos = append(pdbInfos, info)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    pdbInfos,
	})
}

// GetPodDisruptionBudgetYAML 获取 PodDisruptionBudget YAML
func (h *ResourceHandler) GetPodDisruptionBudgetYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	pdb, err := clientset.PolicyV1().PodDisruptionBudgets(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "PodDisruptionBudget")
		return
	}

	yamlData, err := yaml.Marshal(pdb)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "YAML 转换失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"yaml": string(yamlData),
		},
	})
}

// UpdatePodDisruptionBudgetYAML 更新 PodDisruptionBudget YAML
func (h *ResourceHandler) UpdatePodDisruptionBudgetYAML(c *gin.Context) {
	var req struct {
		ClusterID int    `json:"clusterId"`
		YAML      string `json:"yaml"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	var pdb policyv1.PodDisruptionBudget
	if err := yaml.Unmarshal([]byte(req.YAML), &pdb); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析 YAML 失败: " + err.Error(),
		})
		return
	}

	pdb.Name = name
	pdb.Namespace = namespace

	_, err = clientset.PolicyV1().PodDisruptionBudgets(namespace).Update(c.Request.Context(), &pdb, metav1.UpdateOptions{})
	if err != nil {
		HandleK8sError(c, err, "PodDisruptionBudget")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// DeletePodDisruptionBudget 删除 PodDisruptionBudget
func (h *ResourceHandler) DeletePodDisruptionBudget(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	err = clientset.PolicyV1().PodDisruptionBudgets(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		HandleK8sError(c, err, "PodDisruptionBudget")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// CreatePDBFromYAML 从 YAML 创建 PodDisruptionBudget
func (h *ResourceHandler) CreatePDBFromYAML(c *gin.Context) {
	var req struct {
		ClusterID int    `json:"clusterId"`
		YAML      string `json:"yaml"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	namespace := c.Param("namespace")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	var pdb policyv1.PodDisruptionBudget
	if err := yaml.Unmarshal([]byte(req.YAML), &pdb); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析 YAML 失败: " + err.Error(),
		})
		return
	}

	pdb.Namespace = namespace

	_, err = clientset.PolicyV1().PodDisruptionBudgets(namespace).Create(c.Request.Context(), &pdb, metav1.CreateOptions{})
	if err != nil {
		HandleK8sError(c, err, "PodDisruptionBudget")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// ==================== 终端审计相关 ====================

// TerminalSessionInfo 终端会话信息
type TerminalSessionInfo struct {
	ID            uint   `json:"id"`
	ClusterID     uint   `json:"clusterId"`
	ClusterName   string `json:"clusterName"`
	Namespace     string `json:"namespace"`
	PodName       string `json:"podName"`
	ContainerName string `json:"containerName"`
	UserID        uint   `json:"userId"`
	Username      string `json:"username"`
	Duration      int    `json:"duration"`
	FileSize      int64  `json:"fileSize"`
	CreatedAt     string `json:"createdAt"`
}

// ListTerminalSessions 获取终端会话列表
func (h *ResourceHandler) ListTerminalSessions(c *gin.Context) {
	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	var sessions []model.TerminalSession
	err := h.db.Where("user_id = ?", currentUserID).Order("created_at DESC").Find(&sessions).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取会话列表失败: " + err.Error(),
		})
		return
	}

	// 转换为前端格式
	result := make([]TerminalSessionInfo, 0, len(sessions))
	for _, session := range sessions {
		// 获取集群名称
		var cluster models.Cluster
		clusterName := ""
		if session.ClusterID > 0 {
			h.db.First(&cluster, session.ClusterID)
			clusterName = cluster.Alias
			if clusterName == "" {
				clusterName = cluster.Name
			}
		}

		result = append(result, TerminalSessionInfo{
			ID:            session.ID,
			ClusterID:     session.ClusterID,
			ClusterName:   clusterName,
			Namespace:     session.Namespace,
			PodName:       session.PodName,
			ContainerName: session.ContainerName,
			UserID:        session.UserID,
			Username:      session.Username,
			Duration:      session.Duration,
			FileSize:      session.FileSize,
			CreatedAt:     session.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// PlayTerminalSession 播放终端会话
func (h *ResourceHandler) PlayTerminalSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "会话ID不能为空",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	var session model.TerminalSession
	err := h.db.Where("id = ? AND user_id = ?", sessionID, currentUserID).First(&session).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "会话不存在",
		})
		return
	}

	// 读取录制文件
	data, err := os.ReadFile(session.RecordingPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "读取录制文件失败: " + err.Error(),
		})
		return
	}

	// 设置响应头
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=terminal-%d.cast", session.ID))

	c.Data(http.StatusOK, "application/json", data)
}

// DeleteTerminalSession 删除终端会话
func (h *ResourceHandler) DeleteTerminalSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "会话ID不能为空",
		})
		return
	}

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	var session model.TerminalSession
	err := h.db.Where("id = ? AND user_id = ?", sessionID, currentUserID).First(&session).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "会话不存在",
		})
		return
	}

	// 删除录制文件
	if session.RecordingPath != "" {
		os.Remove(session.RecordingPath)
	}

	// 删除数据库记录
	err = h.db.Delete(&session).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除会话失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// ==================== 访问控制资源 ====================

// ListServiceAccounts 获取ServiceAccount列表
func (h *ResourceHandler) ListServiceAccounts(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	namespace := c.Query("namespace")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的集群ID"})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未授权：无法获取用户信息"})
		return
	}
	currentUserID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "用户ID类型错误"})
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取集群连接失败: " + err.Error()})
		return
	}

	list, err := clientset.CoreV1().ServiceAccounts(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "ServiceAccount")
		return
	}

	result := make([]gin.H, 0, len(list.Items))
	for _, item := range list.Items {
		secretNames := make([]string, 0, len(item.Secrets))
		for _, s := range item.Secrets {
			secretNames = append(secretNames, s.Name)
		}
		result = append(result, gin.H{
			"name":      item.Name,
			"namespace": item.Namespace,
			"secrets":   secretNames,
			"age":       calculateAge(item.CreationTimestamp.Time),
			"labels":    item.Labels,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// GetServiceAccountYAML 获取 ServiceAccount YAML
func (h *ResourceHandler) GetServiceAccountYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	serviceAccount, err := clientset.CoreV1().ServiceAccounts(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "ServiceAccount")
		return
	}

	// 构造返回数据，添加 apiVersion 和 kind
	result := gin.H{
		"apiVersion": serviceAccount.APIVersion,
		"kind":       "ServiceAccount",
		"metadata":   serviceAccount.ObjectMeta,
	}
	if len(serviceAccount.Secrets) > 0 {
		result["secrets"] = serviceAccount.Secrets
	}
	if serviceAccount.AutomountServiceAccountToken != nil {
		result["automountServiceAccountToken"] = *serviceAccount.AutomountServiceAccountToken
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// UpdateServiceAccountYAML 更新 ServiceAccount YAML
func (h *ResourceHandler) UpdateServiceAccountYAML(c *gin.Context) {
	var req struct {
		ClusterID int    `json:"clusterId"`
		YAML      string `json:"yaml"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(req.ClusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 解析 YAML
	var serviceAccount v1.ServiceAccount
	if err := yaml.Unmarshal([]byte(req.YAML), &serviceAccount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析 YAML 失败: " + err.Error(),
		})
		return
	}

	namespace := c.Param("namespace")

	// 更新 ServiceAccount
	_, err = clientset.CoreV1().ServiceAccounts(namespace).Update(c.Request.Context(), &serviceAccount, metav1.UpdateOptions{})
	if err != nil {
		HandleK8sError(c, err, "ServiceAccount")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
	})
}

// CreateServiceAccountFromYAML 从 YAML 创建 ServiceAccount
func (h *ResourceHandler) CreateServiceAccountFromYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 读取 YAML 内容
	yamlContent, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "读取请求体失败: " + err.Error(),
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 解析 YAML
	var serviceAccount v1.ServiceAccount
	if err := yaml.Unmarshal(yamlContent, &serviceAccount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析 YAML 失败: " + err.Error(),
		})
		return
	}

	// 确保命名空间正确
	serviceAccount.Namespace = namespace

	// 创建 ServiceAccount
	_, err = clientset.CoreV1().ServiceAccounts(namespace).Create(c.Request.Context(), &serviceAccount, metav1.CreateOptions{})
	if err != nil {
		HandleK8sError(c, err, "ServiceAccount")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
	})
}

// DeleteServiceAccount 删除ServiceAccount
func (h *ResourceHandler) DeleteServiceAccount(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	err = clientset.CoreV1().ServiceAccounts(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		HandleK8sError(c, err, "ServiceAccount")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// GetRoleYAML 获取Role YAML
func (h *ResourceHandler) GetRoleYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	role, err := clientset.RbacV1().Roles(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "Role")
		return
	}

	// 构造返回数据，添加 apiVersion 和 kind
	result := gin.H{
		"apiVersion": role.APIVersion,
		"kind":       "Role",
		"metadata":   role.ObjectMeta,
		"rules":      role.Rules,
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data":    result,
	})
}

// CreateRoleFromYAML 从YAML创建Role
func (h *ResourceHandler) CreateRoleFromYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 读取 YAML 内容
	yamlContent, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "读取请求体失败: " + err.Error(),
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 解析 YAML
	var role rbacv1.Role
	if err := yaml.Unmarshal(yamlContent, &role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析 YAML 失败: " + err.Error(),
		})
		return
	}

	// 确保命名空间正确
	role.Namespace = namespace

	// 创建 Role
	_, err = clientset.RbacV1().Roles(namespace).Create(c.Request.Context(), &role, metav1.CreateOptions{})
	if err != nil {
		HandleK8sError(c, err, "Role")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// UpdateRoleYAML 更新Role YAML
func (h *ResourceHandler) UpdateRoleYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	// 从 query 参数获取 clusterId
	clusterIDStr := c.Query("clusterId")
	if clusterIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少 clusterId 参数",
		})
		return
	}

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "clusterId 参数格式错误",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 直接解析请求体为 Role 对象
	var role rbacv1.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析请求体失败: " + err.Error(),
		})
		return
	}

	// 确保名称一致
	role.Name = name
	role.Namespace = namespace

	// 更新 Role
	_, err = clientset.RbacV1().Roles(namespace).Update(c.Request.Context(), &role, metav1.UpdateOptions{})
	if err != nil {
		HandleK8sError(c, err, "Role")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// DeleteRole 删除Role
func (h *ResourceHandler) DeleteRole(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	err = clientset.RbacV1().Roles(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		HandleK8sError(c, err, "Role")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// GetRoleBindingYAML 获取RoleBinding YAML
func (h *ResourceHandler) GetRoleBindingYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	roleBinding, err := clientset.RbacV1().RoleBindings(namespace).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "RoleBinding")
		return
	}

	// 构造返回数据，添加 apiVersion 和 kind
	result := gin.H{
		"apiVersion": roleBinding.APIVersion,
		"kind":       "RoleBinding",
		"metadata":   roleBinding.ObjectMeta,
		"subjects":   roleBinding.Subjects,
		"roleRef":    roleBinding.RoleRef,
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data":    result,
	})
}

// CreateRoleBindingFromYAML 从YAML创建RoleBinding
func (h *ResourceHandler) CreateRoleBindingFromYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 读取 YAML 内容
	yamlContent, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "读取请求体失败: " + err.Error(),
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 解析 YAML
	var roleBinding rbacv1.RoleBinding
	if err := yaml.Unmarshal(yamlContent, &roleBinding); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析 YAML 失败: " + err.Error(),
		})
		return
	}

	// 确保命名空间正确
	roleBinding.Namespace = namespace

	// 创建 RoleBinding
	_, err = clientset.RbacV1().RoleBindings(namespace).Create(c.Request.Context(), &roleBinding, metav1.CreateOptions{})
	if err != nil {
		HandleK8sError(c, err, "RoleBinding")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// UpdateRoleBindingYAML 更新RoleBinding YAML
func (h *ResourceHandler) UpdateRoleBindingYAML(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	// 从 query 参数获取 clusterId
	clusterIDStr := c.Query("clusterId")
	if clusterIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少 clusterId 参数",
		})
		return
	}

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "clusterId 参数格式错误",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 直接解析请求体为 RoleBinding 对象
	var roleBinding rbacv1.RoleBinding
	if err := c.ShouldBindJSON(&roleBinding); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析请求体失败: " + err.Error(),
		})
		return
	}

	// 确保名称一致
	roleBinding.Name = name
	roleBinding.Namespace = namespace

	// 更新 RoleBinding
	_, err = clientset.RbacV1().RoleBindings(namespace).Update(c.Request.Context(), &roleBinding, metav1.UpdateOptions{})
	if err != nil {
		HandleK8sError(c, err, "RoleBinding")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// DeleteRoleBinding 删除RoleBinding
func (h *ResourceHandler) DeleteRoleBinding(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	err = clientset.RbacV1().RoleBindings(namespace).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		HandleK8sError(c, err, "RoleBinding")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// GetClusterRoleYAML 获取ClusterRole YAML
func (h *ResourceHandler) GetClusterRoleYAML(c *gin.Context) {
	name := c.Param("name")
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	clusterRole, err := clientset.RbacV1().ClusterRoles().Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "ClusterRole")
		return
	}

	// 构造返回数据，添加 apiVersion 和 kind
	result := gin.H{
		"apiVersion": clusterRole.APIVersion,
		"kind":       "ClusterRole",
		"metadata":   clusterRole.ObjectMeta,
		"rules":      clusterRole.Rules,
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data":    result,
	})
}

// CreateClusterRoleFromYAML 从YAML创建ClusterRole
func (h *ResourceHandler) CreateClusterRoleFromYAML(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 读取 YAML 内容
	yamlContent, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "读取请求体失败: " + err.Error(),
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 解析 YAML
	var clusterRole rbacv1.ClusterRole
	if err := yaml.Unmarshal(yamlContent, &clusterRole); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析 YAML 失败: " + err.Error(),
		})
		return
	}

	// 创建 ClusterRole
	_, err = clientset.RbacV1().ClusterRoles().Create(c.Request.Context(), &clusterRole, metav1.CreateOptions{})
	if err != nil {
		HandleK8sError(c, err, "ClusterRole")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// UpdateClusterRoleYAML 更新ClusterRole YAML
func (h *ResourceHandler) UpdateClusterRoleYAML(c *gin.Context) {
	name := c.Param("name")

	// 从 query 参数获取 clusterId
	clusterIDStr := c.Query("clusterId")
	if clusterIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少 clusterId 参数",
		})
		return
	}

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "clusterId 参数格式错误",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 直接解析请求体为 ClusterRole 对象
	var clusterRole rbacv1.ClusterRole
	if err := c.ShouldBindJSON(&clusterRole); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析请求体失败: " + err.Error(),
		})
		return
	}

	// 确保名称一致
	clusterRole.Name = name

	// 更新 ClusterRole
	_, err = clientset.RbacV1().ClusterRoles().Update(c.Request.Context(), &clusterRole, metav1.UpdateOptions{})
	if err != nil {
		HandleK8sError(c, err, "ClusterRole")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// DeleteClusterRole 删除ClusterRole
func (h *ResourceHandler) DeleteClusterRole(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	name := c.Param("name")

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	err = clientset.RbacV1().ClusterRoles().Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		HandleK8sError(c, err, "ClusterRole")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// GetClusterRoleBindingYAML 获取ClusterRoleBinding YAML
func (h *ResourceHandler) GetClusterRoleBindingYAML(c *gin.Context) {
	name := c.Param("name")
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	clusterRoleBinding, err := clientset.RbacV1().ClusterRoleBindings().Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		HandleK8sError(c, err, "ClusterRoleBinding")
		return
	}

	// 构造返回数据，添加 apiVersion 和 kind
	result := gin.H{
		"apiVersion": clusterRoleBinding.APIVersion,
		"kind":       "ClusterRoleBinding",
		"metadata":   clusterRoleBinding.ObjectMeta,
		"subjects":   clusterRoleBinding.Subjects,
		"roleRef":    clusterRoleBinding.RoleRef,
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data":    result,
	})
}

// CreateClusterRoleBindingFromYAML 从YAML创建ClusterRoleBinding
func (h *ResourceHandler) CreateClusterRoleBindingFromYAML(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	// 读取 YAML 内容
	yamlContent, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "读取请求体失败: " + err.Error(),
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 解析 YAML
	var clusterRoleBinding rbacv1.ClusterRoleBinding
	if err := yaml.Unmarshal(yamlContent, &clusterRoleBinding); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析 YAML 失败: " + err.Error(),
		})
		return
	}

	// 创建 ClusterRoleBinding
	_, err = clientset.RbacV1().ClusterRoleBindings().Create(c.Request.Context(), &clusterRoleBinding, metav1.CreateOptions{})
	if err != nil {
		HandleK8sError(c, err, "ClusterRoleBinding")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// UpdateClusterRoleBindingYAML 更新ClusterRoleBinding YAML
func (h *ResourceHandler) UpdateClusterRoleBindingYAML(c *gin.Context) {
	name := c.Param("name")

	// 从 query 参数获取 clusterId
	clusterIDStr := c.Query("clusterId")
	if clusterIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少 clusterId 参数",
		})
		return
	}

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "clusterId 参数格式错误",
		})
		return
	}

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 直接解析请求体为 ClusterRoleBinding 对象
	var clusterRoleBinding rbacv1.ClusterRoleBinding
	if err := c.ShouldBindJSON(&clusterRoleBinding); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析请求体失败: " + err.Error(),
		})
		return
	}

	// 确保名称一致
	clusterRoleBinding.Name = name

	// 更新 ClusterRoleBinding
	_, err = clientset.RbacV1().ClusterRoleBindings().Update(c.Request.Context(), &clusterRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		HandleK8sError(c, err, "ClusterRoleBinding")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// DeleteClusterRoleBinding 删除ClusterRoleBinding
func (h *ResourceHandler) DeleteClusterRoleBinding(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	name := c.Param("name")

	// 获取当前用户ID
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	err = clientset.RbacV1().ClusterRoleBindings().Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		HandleK8sError(c, err, "ClusterRoleBinding")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// ListRoles 获取Role列表
func (h *ResourceHandler) ListRoles(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	namespace := c.Query("namespace")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的集群ID"})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未授权：无法获取用户信息"})
		return
	}
	currentUserID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "用户ID类型错误"})
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取集群连接失败: " + err.Error()})
		return
	}

	result := make([]gin.H, 0)

	if namespace == "" {
		// 获取所有命名空间的 Roles
		namespaceList, err := clientset.CoreV1().Namespaces().List(c.Request.Context(), metav1.ListOptions{})
		if err != nil {
			HandleK8sError(c, err, "Namespace")
			return
		}

		for _, ns := range namespaceList.Items {
			list, err := clientset.RbacV1().Roles(ns.Name).List(c.Request.Context(), metav1.ListOptions{})
			if err != nil {
				continue // 跳过有错误的命名空间
			}
			for _, item := range list.Items {
				result = append(result, gin.H{
					"name":      item.Name,
					"namespace": item.Namespace,
					"age":       calculateAge(item.CreationTimestamp.Time),
					"labels":    item.Labels,
				})
			}
		}
	} else {
		// 获取指定命名空间的 Roles
		list, err := clientset.RbacV1().Roles(namespace).List(c.Request.Context(), metav1.ListOptions{})
		if err != nil {
			HandleK8sError(c, err, "Role")
			return
		}
		for _, item := range list.Items {
			result = append(result, gin.H{
				"name":      item.Name,
				"namespace": item.Namespace,
				"age":       calculateAge(item.CreationTimestamp.Time),
				"labels":    item.Labels,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// ListRoleBindings 获取RoleBinding列表
func (h *ResourceHandler) ListRoleBindings(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	namespace := c.Query("namespace")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的集群ID"})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未授权：无法获取用户信息"})
		return
	}
	currentUserID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "用户ID类型错误"})
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取集群连接失败: " + err.Error()})
		return
	}

	result := make([]gin.H, 0)

	if namespace == "" {
		// 获取所有命名空间的 RoleBindings
		namespaceList, err := clientset.CoreV1().Namespaces().List(c.Request.Context(), metav1.ListOptions{})
		if err != nil {
			HandleK8sError(c, err, "Namespace")
			return
		}

		for _, ns := range namespaceList.Items {
			list, err := clientset.RbacV1().RoleBindings(ns.Name).List(c.Request.Context(), metav1.ListOptions{})
			if err != nil {
				continue // 跳过有错误的命名空间
			}
			for _, item := range list.Items {
				roleKind := "Role"
				roleName := ""
				if item.RoleRef.Kind == "ClusterRole" {
					roleKind = "ClusterRole"
				}
				roleName = item.RoleRef.Name

				subjects := make([]gin.H, 0, len(item.Subjects))
				for _, subject := range item.Subjects {
					subjects = append(subjects, gin.H{
						"kind":      subject.Kind,
						"name":      subject.Name,
						"namespace": subject.Namespace,
					})
				}

				result = append(result, gin.H{
					"name":      item.Name,
					"namespace": item.Namespace,
					"roleKind":  roleKind,
					"roleName":  roleName,
					"subjects":  subjects,
					"age":       calculateAge(item.CreationTimestamp.Time),
					"labels":    item.Labels,
				})
			}
		}
	} else {
		// 获取指定命名空间的 RoleBindings
		list, err := clientset.RbacV1().RoleBindings(namespace).List(c.Request.Context(), metav1.ListOptions{})
		if err != nil {
			HandleK8sError(c, err, "RoleBinding")
			return
		}
		for _, item := range list.Items {
			roleKind := "Role"
			roleName := ""
			if item.RoleRef.Kind == "ClusterRole" {
				roleKind = "ClusterRole"
			}
			roleName = item.RoleRef.Name

			subjects := make([]gin.H, 0, len(item.Subjects))
			for _, subject := range item.Subjects {
				subjects = append(subjects, gin.H{
					"kind":      subject.Kind,
					"name":      subject.Name,
					"namespace": subject.Namespace,
				})
			}

			result = append(result, gin.H{
				"name":      item.Name,
				"namespace": item.Namespace,
				"roleKind":  roleKind,
				"roleName":  roleName,
				"subjects":  subjects,
				"age":       calculateAge(item.CreationTimestamp.Time),
				"labels":    item.Labels,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// ListClusterRoles 获取ClusterRole列表
func (h *ResourceHandler) ListClusterRoles(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的集群ID"})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未授权：无法获取用户信息"})
		return
	}
	currentUserID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "用户ID类型错误"})
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取集群连接失败: " + err.Error()})
		return
	}

	list, err := clientset.RbacV1().ClusterRoles().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "ClusterRole")
		return
	}

	result := make([]gin.H, 0, len(list.Items))
	for _, item := range list.Items {
		result = append(result, gin.H{
			"name":   item.Name,
			"age":    calculateAge(item.CreationTimestamp.Time),
			"labels": item.Labels,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// ListClusterRoleBindings 获取ClusterRoleBinding列表
func (h *ResourceHandler) ListClusterRoleBindings(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的集群ID"})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未授权：无法获取用户信息"})
		return
	}
	currentUserID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "用户ID类型错误"})
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取集群连接失败: " + err.Error()})
		return
	}

	list, err := clientset.RbacV1().ClusterRoleBindings().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "ClusterRoleBinding")
		return
	}

	result := make([]gin.H, 0, len(list.Items))
	for _, item := range list.Items {
		roleName := item.RoleRef.Name

		subjects := make([]gin.H, 0, len(item.Subjects))
		for _, subject := range item.Subjects {
			subjects = append(subjects, gin.H{
				"kind":      subject.Kind,
				"name":      subject.Name,
				"namespace": subject.Namespace,
			})
		}

		result = append(result, gin.H{
			"name":     item.Name,
			"roleName": roleName,
			"subjects": subjects,
			"age":      calculateAge(item.CreationTimestamp.Time),
			"labels":   item.Labels,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// ListPodSecurityPolicies 获取PodSecurityPolicy列表
func (h *ResourceHandler) ListPodSecurityPolicies(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的集群ID"})
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未授权：无法获取用户信息"})
		return
	}
	currentUserID, ok := userIDVal.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "用户ID类型错误"})
		return
	}

	// 验证集群连接
	_, err = h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取集群连接失败: " + err.Error()})
		return
	}

	// PodSecurityPolicy 已在 Kubernetes 1.25+ 中废弃，API不存在
	// 直接返回空列表和提示信息
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "PodSecurityPolicy API已废弃，Kubernetes 1.25+不再支持",
		"data":    []gin.H{},
	})
}

// ==================== ConfigMap 创建 ====================

// CreateConfigMapFromYAML 从 YAML/JSON 创建 ConfigMap
func (h *ResourceHandler) CreateConfigMapFromYAML(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Param("namespace")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 直接绑定 JSON 对象
	var configMap v1.ConfigMap
	if err := c.ShouldBindJSON(&configMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析请求失败: " + err.Error(),
		})
		return
	}

	// 确保命名空间正确
	configMap.Namespace = namespace

	_, err = clientset.CoreV1().ConfigMaps(namespace).Create(c.Request.Context(), &configMap, metav1.CreateOptions{})
	if err != nil {
		HandleK8sError(c, err, "ConfigMap")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}

// ==================== Secret 创建 ====================

// CreateSecretFromYAML 从 YAML/JSON 创建 Secret
func (h *ResourceHandler) CreateSecretFromYAML(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的集群ID",
		})
		return
	}

	namespace := c.Param("namespace")

	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return
	}

	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群客户端失败: " + err.Error(),
		})
		return
	}

	// 直接绑定 JSON 对象
	var secret v1.Secret
	if err := c.ShouldBindJSON(&secret); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "解析请求失败: " + err.Error(),
		})
		return
	}

	// 确保命名空间正确
	secret.Namespace = namespace

	_, err = clientset.CoreV1().Secrets(namespace).Create(c.Request.Context(), &secret, metav1.CreateOptions{})
	if err != nil {
		HandleK8sError(c, err, "Secret")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"needRefresh": true,
		},
	})
}
