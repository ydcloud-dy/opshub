package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"sigs.k8s.io/yaml"

	"github.com/ydcloud-dy/opshub/plugins/kubernetes/service"
)

// ResourceHandler Kubernetes资源处理器
type ResourceHandler struct {
	clusterService *service.ClusterService
}

// NewResourceHandler 创建资源处理器
func NewResourceHandler(clusterService *service.ClusterService) *ResourceHandler {
	return &ResourceHandler{
		clusterService: clusterService,
	}
}

// handleGetClientsetError 处理 GetClientsetForUser 的错误
// 返回 true 表示错误已处理（发送了响应），调用者应该 return
// 返回 false 表示不是凭据错误，需要继续处理
func (h *ResourceHandler) handleGetClientsetError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	// 打印错误信息用于调试
	fmt.Printf("🔍 [handleGetClientsetError] 错误信息: %s\n", err.Error())

	// 检查是否是"用户尚未申请凭据"错误
	if strings.Contains(err.Error(), "尚未申请") || strings.Contains(err.Error(), "凭据") {
		fmt.Printf("❌ [handleGetClientsetError] 返回 403\n")
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "您尚未申请该集群的访问凭据，请在集群管理页面申请 kubeconfig 后再访问",
		})
		return true
	}
	fmt.Printf("⚠️ [handleGetClientsetError] 不是凭据错误，返回 false\n")
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
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Ready     string            `json:"ready"`
	Status    string            `json:"status"`
	Restarts  int32             `json:"restarts"`
	Age       string            `json:"age"`
	IP        string            `json:"ip"`
	Node      string            `json:"node"`
	Labels    map[string]string `json:"labels"`
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

	// 调试日志
	fmt.Printf("🔍 DEBUG [ListNodes]: clusterID=%d, currentUserID=%d\n", clusterID, currentUserID)

	// 使用用户的凭据获取 clientset（实现权限隔离）
	clientset, err := h.clusterService.GetClientsetForUser(c.Request.Context(), uint(clusterID), currentUserID)
	if err != nil {
		fmt.Printf("❌ DEBUG [ListNodes]: GetClientsetForUser failed for userID=%d: %v\n", currentUserID, err)
		if h.handleGetClientsetError(c, err) {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取集群连接失败: " + err.Error(),
		})
		return
	}

	fmt.Printf("✅ DEBUG [ListNodes]: Successfully got clientset for userID=%d\n", currentUserID)

	nodes, err := clientset.CoreV1().Nodes().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		HandleK8sError(c, err, "节点")
		return
	}

	// 获取metrics clientset
	metricsClient, err := h.clusterService.GetCachedMetricsClientset(c.Request.Context(), uint(clusterID))
	if err != nil {
		fmt.Printf("❌ DEBUG [ListNodes]: GetCachedMetricsClientset failed: %v\n", err)
		// 继续执行，只是没有metrics数据
		metricsClient = nil
	}

	// 批量获取所有节点的metrics
	nodeMetricsMap := make(map[string]*v1beta1.NodeMetrics)
	if metricsClient != nil {
		allNodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().List(c.Request.Context(), metav1.ListOptions{})
		if err == nil {
			fmt.Printf("✅ DEBUG [ListNodes]: Successfully got %d node metrics\n", len(allNodeMetrics.Items))
			for _, nm := range allNodeMetrics.Items {
				nodeMetricsMap[nm.Name] = &nm
			}
		} else {
			fmt.Printf("❌ DEBUG [ListNodes]: Failed to get node metrics: %v\n", err)
		}
	} else {
		fmt.Printf("⚠️  DEBUG [ListNodes]: metricsClient is nil\n")
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
			fmt.Printf("📊 DEBUG [ListNodes]: Node %s - CPUUsed: %d millicores, MemoryUsed: %d bytes\n",
				node.Name, nodeInfo.CPUUsed, nodeInfo.MemoryUsed)
		} else {
			fmt.Printf("⚠️  DEBUG [ListNodes]: No metrics found for node %s\n", node.Name)
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
		fmt.Printf("❌ DEBUG [GetNodeMetrics]: GetClientsetForUser failed for userID=%d: %v\n", currentUserID, err)
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

		podInfos = append(podInfos, podInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    podInfos,
	})
}

// ListDeployments 获取Deployment列表
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

		// 调试日志：打印 kube-system 命名空间下的所有 Pod
		log.Printf("[GetComponentInfo] kube-system namespace has %d pods", len(pods.Items))
		for _, pod := range pods.Items {
			log.Printf("[GetComponentInfo] Found pod: %s (OwnerReferences: %d)",
				pod.Name, len(pod.OwnerReferences))
		}

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
					log.Printf("[GetComponentInfo] Matched pod %s to component %s (pattern: %s)",
						pod.Name, componentName, pattern)
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
			log.Printf("[GetComponentInfo] Added component: %s (version: %s, status: %s)",
				componentName, version, status)
		}

		// 转换为切片
		for _, comp := range componentMap {
			componentInfo.Components = append(componentInfo.Components, comp)
		}
		log.Printf("[GetComponentInfo] Total components found from pods: %d", len(componentInfo.Components))
	} else {
		log.Printf("[GetComponentInfo] Failed to list pods in kube-system: %v", err)
	}

	// 如果没有检测到控制平面组件，可能是二进制部署的集群（systemd 启动）
	// 尝试通过节点标签和版本信息来推断
	log.Printf("[GetComponentInfo] Checking for binary deployment cluster...")

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
		log.Printf("[GetComponentInfo] No control plane pods found, checking for binary deployment...")

		// 获取集群版本信息
		serverVersion, err := clientset.Discovery().ServerVersion()
		if err == nil {
			k8sVersion := serverVersion.GitVersion
			log.Printf("[GetComponentInfo] Kubernetes version: %s", k8sVersion)

			// 获取所有节点
			nodes, err := clientset.CoreV1().Nodes().List(c.Request.Context(), metav1.ListOptions{})
			if err == nil {
				hasControlPlaneNode := false
				for _, node := range nodes.Items {
					nodeName := strings.ToLower(node.Name)
					log.Printf("[GetComponentInfo] Checking node: %s", node.Name)

					// 检查节点是否是 master/control-plane 节点
					if _, hasControlPlane := node.Labels["node-role.kubernetes.io/control-plane"]; hasControlPlane {
						log.Printf("[GetComponentInfo] Found control-plane node by label: %s", node.Name)
						hasControlPlaneNode = true
						break
					}
					// 兼容旧的标签
					if _, hasMaster := node.Labels["node-role.kubernetes.io/master"]; hasMaster {
						log.Printf("[GetComponentInfo] Found master node by label: %s", node.Name)
						hasControlPlaneNode = true
						break
					}

					// 如果节点名称包含 master/control-plane/mgr 等关键词，也认为是控制平面节点
					if strings.Contains(nodeName, "master") ||
						strings.Contains(nodeName, "control-plane") ||
						strings.Contains(nodeName, "control") ||
						strings.Contains(nodeName, "mgr") {
						log.Printf("[GetComponentInfo] Found control-plane node by name pattern: %s", node.Name)
						hasControlPlaneNode = true
						break
					}
				}

				// 如果检测到控制平面节点但没有找到控制平面 Pod，说明是二进制部署
				if hasControlPlaneNode {
					log.Printf("[GetComponentInfo] Detected binary deployment cluster, adding components...")

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

					log.Printf("[GetComponentInfo] Added 4 control plane components for binary deployment")
				} else {
					log.Printf("[GetComponentInfo] No control-plane node found, skipping binary deployment detection")
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
// @Success 200 {object} Response
// @Router /api/v1/plugins/kubernetes/resources/api-groups [get]
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
// @Success 200 {object} Response
// @Router /api/v1/plugins/kubernetes/resources/api-resources [get]
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

	// 使用 Update 方法更新节点（这样可以确保 labels 被完全替换）
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

	fmt.Printf("✅ DEBUG [UpdateNodeYAML]: Updated node %s successfully with %d labels\n", nodeName, len(newLabels))

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
		log.Printf("WebSocket upgrade failed: %v", err)
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
	_, err = clientset.AppsV1().Deployments("cloudtty-system").Get(c.Request.Context(), "cloudtty-controller-manager", metav1.GetOptions{})
	if err != nil {
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
}

// ResourceInfo 资源信息
type ResourceInfo struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

// GetWorkloads 获取工作负载列表
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
	desiredPods := deploy.Status.Replicas

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
	desiredPods := sts.Status.Replicas

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

	var images []string
	var requests, limits *ResourceInfo

	if len(ds.Spec.Template.Spec.Containers) > 0 {
		for _, container := range ds.Spec.Template.Spec.Containers {
			images = append(images, container.Image)
		}
		requests, limits = h.getResourceInfo(ds.Spec.Template.Spec.Containers)
	}

	return WorkloadInfo{
		Name:        ds.Name,
		Namespace:   ds.Namespace,
		Type:        "DaemonSet",
		Labels:      ds.Labels,
		ReadyPods:   readyPods,
		DesiredPods: desiredPods,
		Requests:    requests,
		Limits:      limits,
		Images:      images,
		CreatedAt:   ds.CreationTimestamp.Format("2006-01-02 15:04:05"),
		UpdatedAt:   ds.CreationTimestamp.Format("2006-01-02 15:04:05"),
	}
}

// convertJobToWorkload 将 Job 转换为 WorkloadInfo
func (h *ResourceHandler) convertJobToWorkload(job *batchv1.Job) WorkloadInfo {
	readyPods := job.Status.Succeeded
	desiredPods := *job.Spec.Parallelism

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

	return WorkloadInfo{
		Name:        cronJob.Name,
		Namespace:   cronJob.Namespace,
		Type:        "CronJob",
		Labels:      cronJob.Labels,
		ReadyPods:   0,
		DesiredPods: 0,
		Requests:    requests,
		Limits:      limits,
		Images:      images,
		CreatedAt:   cronJob.CreationTimestamp.Format("2006-01-02 15:04:05"),
		UpdatedAt:   cronJob.CreationTimestamp.Format("2006-01-02 15:04:05"),
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
		_, err := clientset.AppsV1().Deployments(namespace).Patch(c.Request.Context(), name, types.StrategicMergePatchType, patchData, metav1.PatchOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新Deployment失败: " + err.Error(),
			})
			return
		}
	case "StatefulSet":
		_, err := clientset.AppsV1().StatefulSets(namespace).Patch(c.Request.Context(), name, types.StrategicMergePatchType, patchData, metav1.PatchOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新StatefulSet失败: " + err.Error(),
			})
			return
		}
	case "DaemonSet":
		_, err := clientset.AppsV1().DaemonSets(namespace).Patch(c.Request.Context(), name, types.StrategicMergePatchType, patchData, metav1.PatchOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新DaemonSet失败: " + err.Error(),
			})
			return
		}
	case "Job":
		_, err := clientset.BatchV1().Jobs(namespace).Patch(c.Request.Context(), name, types.StrategicMergePatchType, patchData, metav1.PatchOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新Job失败: " + err.Error(),
			})
			return
		}
	case "CronJob":
		_, err := clientset.BatchV1().CronJobs(namespace).Patch(c.Request.Context(), name, types.StrategicMergePatchType, patchData, metav1.PatchOptions{})
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
