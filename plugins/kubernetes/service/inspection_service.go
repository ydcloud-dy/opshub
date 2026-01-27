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

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/ydcloud-dy/opshub/plugins/kubernetes/model"
)

// InspectionService 巡检服务
type InspectionService struct {
	db             *gorm.DB
	clusterService *ClusterService
}

// NewInspectionService 创建巡检服务
func NewInspectionService(db *gorm.DB, clusterService *ClusterService) *InspectionService {
	return &InspectionService{
		db:             db,
		clusterService: clusterService,
	}
}

// StartInspection 开始巡检
func (s *InspectionService) StartInspection(ctx context.Context, req *model.StartInspectionRequest) (*model.ClusterInspection, error) {
	startTime := time.Now()

	// 获取第一个集群信息（目前单集群巡检）
	if len(req.ClusterIDs) == 0 {
		return nil, fmt.Errorf("未选择集群")
	}

	clusterID := req.ClusterIDs[0]
	cluster, err := s.clusterService.GetCluster(ctx, uint(clusterID))
	if err != nil {
		return nil, fmt.Errorf("获取集群信息失败: %w", err)
	}

	// 创建巡检记录
	inspection := &model.ClusterInspection{
		ClusterID:   clusterID,
		ClusterName: cluster.Name,
		Status:      model.InspectionStatusRunning,
		UserID:      req.UserID,
		StartTime:   startTime,
	}
	if err := s.db.Create(inspection).Error; err != nil {
		return nil, fmt.Errorf("创建巡检记录失败: %w", err)
	}

	// 异步执行巡检
	go s.runInspection(context.Background(), inspection, req.Options)

	return inspection, nil
}

// runInspection 执行巡检
func (s *InspectionService) runInspection(ctx context.Context, inspection *model.ClusterInspection, options model.InspectionOptions) {
	startTime := time.Now()

	// 获取clientset
	clientset, err := s.clusterService.GetCachedClientset(ctx, uint(inspection.ClusterID))
	if err != nil {
		s.updateInspectionFailed(inspection, err)
		return
	}

	result := &model.InspectionResult{
		ClusterID:   inspection.ClusterID,
		ClusterName: inspection.ClusterName,
	}

	// 设置默认选项
	if !options.CheckCluster && !options.CheckNodes && !options.CheckWorkloads {
		options = model.InspectionOptions{
			CheckCluster:   true,
			CheckNodes:     true,
			CheckWorkloads: true,
			CheckNetwork:   true,
			CheckStorage:   true,
			CheckSecurity:  true,
			CheckConfig:    true,
			CheckCapacity:  true,
			CheckEvents:    true,
		}
	}

	// 并发执行各项检查
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 集群信息检查
	if options.CheckCluster {
		wg.Add(1)
		go func() {
			defer wg.Done()
			clusterInfo := s.checkClusterInfo(ctx, clientset)
			mu.Lock()
			result.ClusterInfo = clusterInfo
			mu.Unlock()
		}()
	}

	// 节点健康检查
	if options.CheckNodes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			nodeHealth := s.checkNodeHealth(ctx, clientset)
			mu.Lock()
			result.NodeHealth = nodeHealth
			mu.Unlock()
		}()
	}

	// 组件检查
	if options.CheckCluster {
		wg.Add(1)
		go func() {
			defer wg.Done()
			components := s.checkComponents(ctx, clientset)
			mu.Lock()
			result.Components = components
			mu.Unlock()
		}()
	}

	// 工作负载检查
	if options.CheckWorkloads {
		wg.Add(1)
		go func() {
			defer wg.Done()
			workloads := s.checkWorkloads(ctx, clientset)
			mu.Lock()
			result.Workloads = workloads
			mu.Unlock()
		}()
	}

	// 网络检查
	if options.CheckNetwork {
		wg.Add(1)
		go func() {
			defer wg.Done()
			network := s.checkNetwork(ctx, clientset)
			mu.Lock()
			result.Network = network
			mu.Unlock()
		}()
	}

	// 存储检查
	if options.CheckStorage {
		wg.Add(1)
		go func() {
			defer wg.Done()
			storage := s.checkStorage(ctx, clientset)
			mu.Lock()
			result.Storage = storage
			mu.Unlock()
		}()
	}

	// 安全检查
	if options.CheckSecurity {
		wg.Add(1)
		go func() {
			defer wg.Done()
			security := s.checkSecurity(ctx, clientset)
			mu.Lock()
			result.Security = security
			mu.Unlock()
		}()
	}

	// 配置检查
	if options.CheckConfig {
		wg.Add(1)
		go func() {
			defer wg.Done()
			config := s.checkConfig(ctx, clientset)
			mu.Lock()
			result.Config = config
			mu.Unlock()
		}()
	}

	// 容量检查
	if options.CheckCapacity {
		wg.Add(1)
		go func() {
			defer wg.Done()
			capacity := s.checkCapacity(ctx, clientset)
			mu.Lock()
			result.Capacity = capacity
			mu.Unlock()
		}()
	}

	// 事件检查
	if options.CheckEvents {
		wg.Add(1)
		go func() {
			defer wg.Done()
			events := s.checkEvents(ctx, clientset)
			mu.Lock()
			result.Events = events
			mu.Unlock()
		}()
	}

	wg.Wait()

	// 汇总结果
	result.Summary = s.summarizeResults(result)
	result.Score = s.calculateScore(result)

	// 保存结果
	s.saveInspectionResult(inspection, result, startTime)
}

// checkClusterInfo 检查集群信息
func (s *InspectionService) checkClusterInfo(ctx context.Context, clientset *kubernetes.Clientset) model.ClusterInfoResult {
	result := model.ClusterInfoResult{
		Items: []model.CheckItem{},
	}

	startTime := time.Now()

	// 获取版本信息
	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		result.ConnectionState = "failed"
		result.Items = append(result.Items, model.CheckItem{
			Category:   "集群连接",
			Name:       "API Server连接",
			Status:     model.CheckStatusError,
			Detail:     fmt.Sprintf("连接失败: %v", err),
			Suggestion: "请检查集群凭证是否有效，网络是否可达",
		})
		return result
	}

	result.ConnectionDelay = time.Since(startTime).Milliseconds()
	result.ConnectionState = "connected"
	result.Version = version.GitVersion
	result.Platform = version.Platform
	result.GitVersion = version.GitVersion
	result.GoVersion = version.GoVersion
	result.BuildDate = version.BuildDate

	// 连接状态检查
	if result.ConnectionDelay > 2000 {
		result.Items = append(result.Items, model.CheckItem{
			Category:   "集群连接",
			Name:       "API Server连接延迟",
			Status:     model.CheckStatusWarning,
			Value:      fmt.Sprintf("%dms", result.ConnectionDelay),
			Expected:   "<2000ms",
			Detail:     "连接延迟较高",
			Suggestion: "检查网络状况，考虑使用更近的节点",
		})
	} else {
		result.Items = append(result.Items, model.CheckItem{
			Category: "集群连接",
			Name:     "API Server连接",
			Status:   model.CheckStatusSuccess,
			Value:    fmt.Sprintf("%dms", result.ConnectionDelay),
			Detail:   "连接正常",
		})
	}

	// 版本检查
	result.Items = append(result.Items, model.CheckItem{
		Category: "版本信息",
		Name:     "Kubernetes版本",
		Status:   model.CheckStatusSuccess,
		Value:    result.Version,
		Detail:   fmt.Sprintf("平台: %s, Go版本: %s", result.Platform, result.GoVersion),
	})

	return result
}

// checkNodeHealth 检查节点健康
func (s *InspectionService) checkNodeHealth(ctx context.Context, clientset *kubernetes.Clientset) model.NodeHealthResult {
	result := model.NodeHealthResult{
		Items:           []model.CheckItem{},
		NodeUtilization: []model.NodeResource{},
	}

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		result.Items = append(result.Items, model.CheckItem{
			Category: "节点",
			Name:     "节点列表获取",
			Status:   model.CheckStatusError,
			Detail:   fmt.Sprintf("获取节点列表失败: %v", err),
		})
		return result
	}

	result.TotalNodes = len(nodes.Items)

	for _, node := range nodes.Items {
		nodeResource := model.NodeResource{
			Name:   node.Name,
			Status: "NotReady",
		}

		// 检查Ready状态
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady {
				if cond.Status == corev1.ConditionTrue {
					result.ReadyNodes++
					nodeResource.Status = "Ready"
				} else {
					result.NotReadyNodes++
					result.Items = append(result.Items, model.CheckItem{
						Category:   "节点状态",
						Name:       node.Name,
						Status:     model.CheckStatusError,
						Detail:     fmt.Sprintf("节点NotReady: %s", cond.Message),
						Suggestion: "检查节点kubelet日志，确认网络和资源状态",
					})
				}
			}

			// 检查资源压力
			if cond.Type == corev1.NodeMemoryPressure && cond.Status == corev1.ConditionTrue {
				result.PressureNodes++
				result.Items = append(result.Items, model.CheckItem{
					Category:   "资源压力",
					Name:       node.Name,
					Status:     model.CheckStatusWarning,
					Detail:     "节点存在内存压力",
					Suggestion: "清理未使用的容器或增加节点内存",
				})
			}
			if cond.Type == corev1.NodeDiskPressure && cond.Status == corev1.ConditionTrue {
				result.PressureNodes++
				result.Items = append(result.Items, model.CheckItem{
					Category:   "资源压力",
					Name:       node.Name,
					Status:     model.CheckStatusWarning,
					Detail:     "节点存在磁盘压力",
					Suggestion: "清理磁盘空间或扩容",
				})
			}
			if cond.Type == corev1.NodePIDPressure && cond.Status == corev1.ConditionTrue {
				result.PressureNodes++
				result.Items = append(result.Items, model.CheckItem{
					Category:   "资源压力",
					Name:       node.Name,
					Status:     model.CheckStatusWarning,
					Detail:     "节点存在PID压力",
					Suggestion: "检查是否有进程泄漏",
				})
			}
		}

		// 检查污点
		if len(node.Spec.Taints) > 0 {
			result.TaintedNodes++
		}

		// 资源容量
		cpuCap := node.Status.Capacity.Cpu()
		memCap := node.Status.Capacity.Memory()
		podCap := node.Status.Capacity.Pods()

		nodeResource.CPUCapacity = cpuCap.String()
		nodeResource.MemoryCapacity = formatMemory(memCap.Value())
		nodeResource.PodCapacity = int(podCap.Value())

		result.NodeUtilization = append(result.NodeUtilization, nodeResource)
	}

	// 总体评估
	if result.NotReadyNodes == 0 {
		result.Items = append(result.Items, model.CheckItem{
			Category: "节点状态",
			Name:     "节点健康检查",
			Status:   model.CheckStatusSuccess,
			Value:    fmt.Sprintf("%d/%d Ready", result.ReadyNodes, result.TotalNodes),
			Detail:   "所有节点运行正常",
		})
	}

	return result
}

// checkComponents 检查控制平面组件
func (s *InspectionService) checkComponents(ctx context.Context, clientset *kubernetes.Clientset) model.ComponentsResult {
	result := model.ComponentsResult{
		ControlPlane: []model.ComponentStatus{},
		Addons:       []model.ComponentStatus{},
		Items:        []model.CheckItem{},
	}

	// 检查kube-system命名空间中的核心组件
	pods, err := clientset.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{})
	if err != nil {
		result.Items = append(result.Items, model.CheckItem{
			Category: "组件",
			Name:     "组件检查",
			Status:   model.CheckStatusError,
			Detail:   fmt.Sprintf("获取kube-system Pod失败: %v", err),
		})
		return result
	}

	controlPlaneComponents := []string{"kube-apiserver", "kube-controller-manager", "kube-scheduler", "etcd"}
	addonComponents := []string{"coredns", "kube-dns", "kube-proxy", "calico", "flannel", "cilium"}

	for _, pod := range pods.Items {
		componentName := ""
		isControlPlane := false
		isAddon := false

		for _, cp := range controlPlaneComponents {
			if strings.Contains(pod.Name, cp) {
				componentName = cp
				isControlPlane = true
				break
			}
		}
		if !isControlPlane {
			for _, addon := range addonComponents {
				if strings.Contains(pod.Name, addon) {
					componentName = addon
					isAddon = true
					break
				}
			}
		}

		if componentName == "" {
			continue
		}

		status := "Healthy"
		if pod.Status.Phase != corev1.PodRunning {
			status = "Unhealthy"
		}

		restarts := 0
		ready := "0/0"
		if len(pod.Status.ContainerStatuses) > 0 {
			readyCount := 0
			for _, cs := range pod.Status.ContainerStatuses {
				restarts += int(cs.RestartCount)
				if cs.Ready {
					readyCount++
				}
			}
			ready = fmt.Sprintf("%d/%d", readyCount, len(pod.Status.ContainerStatuses))
		}

		compStatus := model.ComponentStatus{
			Name:      componentName,
			Namespace: pod.Namespace,
			Status:    status,
			Ready:     ready,
			Restarts:  restarts,
			Age:       formatAge(pod.CreationTimestamp.Time),
		}

		if isControlPlane {
			result.ControlPlane = append(result.ControlPlane, compStatus)
		} else if isAddon {
			result.Addons = append(result.Addons, compStatus)
		}
	}

	// 检查控制平面组件状态
	unhealthyControlPlane := 0
	for _, comp := range result.ControlPlane {
		if comp.Status != "Healthy" {
			unhealthyControlPlane++
			result.Items = append(result.Items, model.CheckItem{
				Category:   "控制平面",
				Name:       comp.Name,
				Status:     model.CheckStatusError,
				Detail:     fmt.Sprintf("组件状态异常: %s", comp.Status),
				Suggestion: "检查组件日志，确认组件是否正常运行",
			})
		}
	}

	if unhealthyControlPlane == 0 && len(result.ControlPlane) > 0 {
		result.Items = append(result.Items, model.CheckItem{
			Category: "控制平面",
			Name:     "核心组件检查",
			Status:   model.CheckStatusSuccess,
			Value:    fmt.Sprintf("%d组件正常", len(result.ControlPlane)),
			Detail:   "所有控制平面组件运行正常",
		})
	}

	// 检查插件组件
	unhealthyAddons := 0
	for _, addon := range result.Addons {
		if addon.Status != "Healthy" {
			unhealthyAddons++
		}
		if addon.Restarts > 5 {
			result.Items = append(result.Items, model.CheckItem{
				Category:   "插件组件",
				Name:       addon.Name,
				Status:     model.CheckStatusWarning,
				Value:      fmt.Sprintf("重启%d次", addon.Restarts),
				Detail:     "组件重启次数较多",
				Suggestion: "检查组件日志，排查重启原因",
			})
		}
	}

	if unhealthyAddons == 0 && len(result.Addons) > 0 {
		result.Items = append(result.Items, model.CheckItem{
			Category: "插件组件",
			Name:     "插件检查",
			Status:   model.CheckStatusSuccess,
			Value:    fmt.Sprintf("%d插件正常", len(result.Addons)),
			Detail:   "所有插件组件运行正常",
		})
	}

	return result
}

// checkWorkloads 检查工作负载
func (s *InspectionService) checkWorkloads(ctx context.Context, clientset *kubernetes.Clientset) model.WorkloadsResult {
	result := model.WorkloadsResult{
		Items:              []model.CheckItem{},
		PodsByPhase:        make(map[string]int),
		UnhealthyWorkloads: []model.WorkloadInfo{},
	}

	// 检查Deployments
	deployments, err := clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err == nil {
		result.TotalDeployments = len(deployments.Items)
		for _, dep := range deployments.Items {
			if dep.Status.ReadyReplicas == dep.Status.Replicas && dep.Status.Replicas > 0 {
				result.HealthyDeployments++
			} else if dep.Status.Replicas > 0 {
				result.UnhealthyWorkloads = append(result.UnhealthyWorkloads, model.WorkloadInfo{
					Kind:      "Deployment",
					Namespace: dep.Namespace,
					Name:      dep.Name,
					Ready:     fmt.Sprintf("%d/%d", dep.Status.ReadyReplicas, dep.Status.Replicas),
					Status:    "NotReady",
					Reason:    "副本数不符合预期",
				})
			}
		}
	}

	// 检查DaemonSets
	daemonsets, err := clientset.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err == nil {
		result.TotalDaemonSets = len(daemonsets.Items)
		for _, ds := range daemonsets.Items {
			if ds.Status.NumberReady == ds.Status.DesiredNumberScheduled {
				result.HealthyDaemonSets++
			} else {
				result.UnhealthyWorkloads = append(result.UnhealthyWorkloads, model.WorkloadInfo{
					Kind:      "DaemonSet",
					Namespace: ds.Namespace,
					Name:      ds.Name,
					Ready:     fmt.Sprintf("%d/%d", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled),
					Status:    "NotReady",
					Reason:    "节点数不符合预期",
				})
			}
		}
	}

	// 检查StatefulSets
	statefulsets, err := clientset.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err == nil {
		result.TotalStatefulSets = len(statefulsets.Items)
		for _, sts := range statefulsets.Items {
			if sts.Status.ReadyReplicas == *sts.Spec.Replicas && *sts.Spec.Replicas > 0 {
				result.HealthyStatefulSets++
			} else if *sts.Spec.Replicas > 0 {
				result.UnhealthyWorkloads = append(result.UnhealthyWorkloads, model.WorkloadInfo{
					Kind:      "StatefulSet",
					Namespace: sts.Namespace,
					Name:      sts.Name,
					Ready:     fmt.Sprintf("%d/%d", sts.Status.ReadyReplicas, *sts.Spec.Replicas),
					Status:    "NotReady",
					Reason:    "副本数不符合预期",
				})
			}
		}
	}

	// 检查Pods
	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err == nil {
		result.TotalPods = len(pods.Items)
		for _, pod := range pods.Items {
			result.PodsByPhase[string(pod.Status.Phase)]++

			switch pod.Status.Phase {
			case corev1.PodRunning:
				result.RunningPods++
			case corev1.PodPending:
				result.PendingPods++
			case corev1.PodFailed:
				result.FailedPods++
			}

			// 检查重启次数
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.RestartCount > 5 {
					result.HighRestartPods++
					break
				}
			}

			// 检查镜像拉取错误
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.State.Waiting != nil {
					reason := cs.State.Waiting.Reason
					if reason == "ImagePullBackOff" || reason == "ErrImagePull" {
						result.ImagePullErrors++
						break
					}
				}
			}
		}
	}

	// 生成检查项
	unhealthyWorkloads := len(result.UnhealthyWorkloads)
	if unhealthyWorkloads > 0 {
		result.Items = append(result.Items, model.CheckItem{
			Category:   "工作负载",
			Name:       "工作负载健康检查",
			Status:     model.CheckStatusWarning,
			Value:      fmt.Sprintf("%d个异常", unhealthyWorkloads),
			Detail:     fmt.Sprintf("存在%d个工作负载副本数不符合预期", unhealthyWorkloads),
			Suggestion: "检查异常工作负载的事件和日志",
		})
	} else {
		result.Items = append(result.Items, model.CheckItem{
			Category: "工作负载",
			Name:     "工作负载健康检查",
			Status:   model.CheckStatusSuccess,
			Value:    "全部正常",
			Detail:   "所有工作负载运行正常",
		})
	}

	if result.PendingPods > 0 {
		result.Items = append(result.Items, model.CheckItem{
			Category:   "Pod状态",
			Name:       "Pending Pod检查",
			Status:     model.CheckStatusWarning,
			Value:      fmt.Sprintf("%d个Pending", result.PendingPods),
			Detail:     fmt.Sprintf("存在%d个Pod处于Pending状态", result.PendingPods),
			Suggestion: "检查资源是否充足，节点是否可调度",
		})
	}

	if result.FailedPods > 0 {
		result.Items = append(result.Items, model.CheckItem{
			Category:   "Pod状态",
			Name:       "Failed Pod检查",
			Status:     model.CheckStatusError,
			Value:      fmt.Sprintf("%d个Failed", result.FailedPods),
			Detail:     fmt.Sprintf("存在%d个Pod处于Failed状态", result.FailedPods),
			Suggestion: "检查Pod事件和容器日志",
		})
	}

	if result.HighRestartPods > 0 {
		result.Items = append(result.Items, model.CheckItem{
			Category:   "Pod状态",
			Name:       "高重启Pod检查",
			Status:     model.CheckStatusWarning,
			Value:      fmt.Sprintf("%d个高重启", result.HighRestartPods),
			Detail:     fmt.Sprintf("存在%d个Pod重启次数超过5次", result.HighRestartPods),
			Suggestion: "检查容器日志，排查重启原因",
		})
	}

	if result.ImagePullErrors > 0 {
		result.Items = append(result.Items, model.CheckItem{
			Category:   "Pod状态",
			Name:       "镜像拉取检查",
			Status:     model.CheckStatusError,
			Value:      fmt.Sprintf("%d个错误", result.ImagePullErrors),
			Detail:     fmt.Sprintf("存在%d个Pod镜像拉取失败", result.ImagePullErrors),
			Suggestion: "检查镜像名称是否正确，镜像仓库是否可达",
		})
	}

	return result
}

// checkNetwork 检查网络
func (s *InspectionService) checkNetwork(ctx context.Context, clientset *kubernetes.Clientset) model.NetworkResult {
	result := model.NetworkResult{
		Items: []model.CheckItem{},
	}

	// 获取Services
	services, err := clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err == nil {
		result.TotalServices = len(services.Items)
		for _, svc := range services.Items {
			switch svc.Spec.Type {
			case corev1.ServiceTypeClusterIP:
				result.ClusterIPServices++
			case corev1.ServiceTypeNodePort:
				result.NodePortServices++
			case corev1.ServiceTypeLoadBalancer:
				result.LoadBalancerServices++
			}
		}

		// 检查无Endpoint的Service
		endpoints, _ := clientset.CoreV1().Endpoints("").List(ctx, metav1.ListOptions{})
		endpointNames := make(map[string]bool)
		if endpoints != nil {
			for _, ep := range endpoints.Items {
				hasAddresses := false
				for _, subset := range ep.Subsets {
					if len(subset.Addresses) > 0 {
						hasAddresses = true
						break
					}
				}
				if hasAddresses {
					endpointNames[fmt.Sprintf("%s/%s", ep.Namespace, ep.Name)] = true
				}
			}
		}

		for _, svc := range services.Items {
			if svc.Spec.Type != corev1.ServiceTypeExternalName {
				key := fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)
				if !endpointNames[key] && len(svc.Spec.Selector) > 0 {
					result.NoEndpointServices++
				}
			}
		}
	}

	// 获取Ingresses
	ingresses, err := clientset.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	if err == nil {
		result.TotalIngresses = len(ingresses.Items)
	}

	// 获取NetworkPolicies
	networkPolicies, err := clientset.NetworkingV1().NetworkPolicies("").List(ctx, metav1.ListOptions{})
	if err == nil {
		result.NetworkPolicies = len(networkPolicies.Items)
	}

	// 生成检查项
	result.Items = append(result.Items, model.CheckItem{
		Category: "网络",
		Name:     "Service统计",
		Status:   model.CheckStatusSuccess,
		Value:    fmt.Sprintf("共%d个", result.TotalServices),
		Detail:   fmt.Sprintf("ClusterIP: %d, NodePort: %d, LoadBalancer: %d", result.ClusterIPServices, result.NodePortServices, result.LoadBalancerServices),
	})

	if result.NoEndpointServices > 0 {
		result.Items = append(result.Items, model.CheckItem{
			Category:   "网络",
			Name:       "无Endpoint Service检查",
			Status:     model.CheckStatusWarning,
			Value:      fmt.Sprintf("%d个", result.NoEndpointServices),
			Detail:     fmt.Sprintf("存在%d个Service没有对应的Endpoint", result.NoEndpointServices),
			Suggestion: "检查Service选择器是否匹配Pod标签",
		})
	}

	return result
}

// checkStorage 检查存储
func (s *InspectionService) checkStorage(ctx context.Context, clientset *kubernetes.Clientset) model.StorageResult {
	result := model.StorageResult{
		Items: []model.CheckItem{},
	}

	// 获取PVs
	pvs, err := clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err == nil {
		result.TotalPVs = len(pvs.Items)
		for _, pv := range pvs.Items {
			switch pv.Status.Phase {
			case corev1.VolumeAvailable:
				result.AvailablePVs++
			case corev1.VolumeBound:
				result.BoundPVs++
			case corev1.VolumeReleased:
				result.ReleasedPVs++
			case corev1.VolumeFailed:
				result.FailedPVs++
			}
		}
	}

	// 获取PVCs
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
	if err == nil {
		result.TotalPVCs = len(pvcs.Items)
		for _, pvc := range pvcs.Items {
			switch pvc.Status.Phase {
			case corev1.ClaimBound:
				result.BoundPVCs++
			case corev1.ClaimPending:
				result.PendingPVCs++
			}
		}
	}

	// 获取StorageClasses
	scs, err := clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err == nil {
		result.StorageClasses = len(scs.Items)
		for _, sc := range scs.Items {
			if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
				result.DefaultSC = sc.Name
				break
			}
		}
	}

	// 生成检查项
	result.Items = append(result.Items, model.CheckItem{
		Category: "存储",
		Name:     "PV统计",
		Status:   model.CheckStatusSuccess,
		Value:    fmt.Sprintf("共%d个", result.TotalPVs),
		Detail:   fmt.Sprintf("可用: %d, 已绑定: %d, 已释放: %d, 失败: %d", result.AvailablePVs, result.BoundPVs, result.ReleasedPVs, result.FailedPVs),
	})

	if result.FailedPVs > 0 {
		result.Items = append(result.Items, model.CheckItem{
			Category:   "存储",
			Name:       "PV状态检查",
			Status:     model.CheckStatusError,
			Value:      fmt.Sprintf("%d个失败", result.FailedPVs),
			Detail:     fmt.Sprintf("存在%d个PV处于Failed状态", result.FailedPVs),
			Suggestion: "检查PV配置和后端存储状态",
		})
	}

	if result.PendingPVCs > 0 {
		result.Items = append(result.Items, model.CheckItem{
			Category:   "存储",
			Name:       "PVC状态检查",
			Status:     model.CheckStatusWarning,
			Value:      fmt.Sprintf("%d个Pending", result.PendingPVCs),
			Detail:     fmt.Sprintf("存在%d个PVC处于Pending状态", result.PendingPVCs),
			Suggestion: "检查是否有可用的PV或StorageClass是否配置正确",
		})
	}

	if result.DefaultSC == "" && result.StorageClasses > 0 {
		result.Items = append(result.Items, model.CheckItem{
			Category:   "存储",
			Name:       "默认StorageClass检查",
			Status:     model.CheckStatusWarning,
			Detail:     "未设置默认StorageClass",
			Suggestion: "建议设置一个默认的StorageClass",
		})
	}

	return result
}

// checkSecurity 检查安全配置
func (s *InspectionService) checkSecurity(ctx context.Context, clientset *kubernetes.Clientset) model.SecurityResult {
	result := model.SecurityResult{
		Items: []model.CheckItem{},
	}

	// 获取ServiceAccounts
	sas, err := clientset.CoreV1().ServiceAccounts("").List(ctx, metav1.ListOptions{})
	if err == nil {
		result.ServiceAccounts = len(sas.Items)
	}

	// 获取Roles
	roles, err := clientset.RbacV1().Roles("").List(ctx, metav1.ListOptions{})
	if err == nil {
		result.Roles = len(roles.Items)
	}

	// 获取ClusterRoles
	clusterRoles, err := clientset.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err == nil {
		result.ClusterRoles = len(clusterRoles.Items)
	}

	// 获取RoleBindings
	rbs, err := clientset.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{})
	if err == nil {
		result.RoleBindings = len(rbs.Items)
	}

	// 获取ClusterRoleBindings
	crbs, err := clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err == nil {
		result.ClusterRoleBindings = len(crbs.Items)
		// 检查cluster-admin绑定
		for _, crb := range crbs.Items {
			if crb.RoleRef.Name == "cluster-admin" {
				result.ClusterAdminBindings++
			}
		}
	}

	// 检查Pod安全配置
	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, pod := range pods.Items {
			if pod.Spec.HostNetwork {
				result.HostNetworkPods++
			}
			for _, container := range pod.Spec.Containers {
				if container.SecurityContext != nil {
					if container.SecurityContext.Privileged != nil && *container.SecurityContext.Privileged {
						result.PrivilegedPods++
						break
					}
					if container.SecurityContext.RunAsUser != nil && *container.SecurityContext.RunAsUser == 0 {
						result.RootUserContainers++
					}
				}
			}
		}
	}

	// 生成检查项
	result.Items = append(result.Items, model.CheckItem{
		Category: "安全",
		Name:     "RBAC统计",
		Status:   model.CheckStatusSuccess,
		Value:    fmt.Sprintf("SA: %d, Role: %d, CRB: %d", result.ServiceAccounts, result.Roles, result.ClusterRoleBindings),
		Detail:   "RBAC资源统计",
	})

	if result.ClusterAdminBindings > 3 {
		result.Items = append(result.Items, model.CheckItem{
			Category:   "安全",
			Name:       "cluster-admin绑定检查",
			Status:     model.CheckStatusWarning,
			Value:      fmt.Sprintf("%d个", result.ClusterAdminBindings),
			Detail:     fmt.Sprintf("存在%d个cluster-admin绑定", result.ClusterAdminBindings),
			Suggestion: "建议最小化cluster-admin权限的使用",
		})
	}

	if result.PrivilegedPods > 0 {
		result.Items = append(result.Items, model.CheckItem{
			Category:   "安全",
			Name:       "特权容器检查",
			Status:     model.CheckStatusWarning,
			Value:      fmt.Sprintf("%d个", result.PrivilegedPods),
			Detail:     fmt.Sprintf("存在%d个Pod使用特权模式", result.PrivilegedPods),
			Suggestion: "建议避免使用特权模式，除非绝对必要",
		})
	}

	if result.HostNetworkPods > 0 {
		result.Items = append(result.Items, model.CheckItem{
			Category:   "安全",
			Name:       "hostNetwork检查",
			Status:     model.CheckStatusWarning,
			Value:      fmt.Sprintf("%d个", result.HostNetworkPods),
			Detail:     fmt.Sprintf("存在%d个Pod使用hostNetwork", result.HostNetworkPods),
			Suggestion: "建议避免使用hostNetwork，除非网络插件需要",
		})
	}

	return result
}

// checkConfig 检查配置
func (s *InspectionService) checkConfig(ctx context.Context, clientset *kubernetes.Clientset) model.ConfigResult {
	result := model.ConfigResult{
		Items: []model.CheckItem{},
	}

	// 获取ConfigMaps
	cms, err := clientset.CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{})
	if err == nil {
		result.TotalConfigMaps = len(cms.Items)
	}

	// 获取Secrets
	secrets, err := clientset.CoreV1().Secrets("").List(ctx, metav1.ListOptions{})
	if err == nil {
		result.TotalSecrets = len(secrets.Items)
	}

	// 获取命名空间
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err == nil {
		result.NamespaceCount = len(namespaces.Items)
	}

	// 获取ResourceQuotas
	quotas, err := clientset.CoreV1().ResourceQuotas("").List(ctx, metav1.ListOptions{})
	if err == nil {
		result.ResourceQuotaCount = len(quotas.Items)
	}

	// 获取LimitRanges
	limitRanges, err := clientset.CoreV1().LimitRanges("").List(ctx, metav1.ListOptions{})
	if err == nil {
		result.LimitRangeCount = len(limitRanges.Items)
	}

	// 检查未设置资源限制的Pod
	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, pod := range pods.Items {
			hasLimits := true
			for _, container := range pod.Spec.Containers {
				if container.Resources.Limits.Cpu().IsZero() && container.Resources.Limits.Memory().IsZero() {
					hasLimits = false
					break
				}
			}
			if !hasLimits {
				result.NoRequestLimitPods++
			}
		}
	}

	// 生成检查项
	result.Items = append(result.Items, model.CheckItem{
		Category: "配置",
		Name:     "配置资源统计",
		Status:   model.CheckStatusSuccess,
		Value:    fmt.Sprintf("ConfigMap: %d, Secret: %d", result.TotalConfigMaps, result.TotalSecrets),
		Detail:   "配置资源数量统计",
	})

	if result.NoRequestLimitPods > 0 {
		totalPods := 0
		if pods != nil {
			totalPods = len(pods.Items)
		}
		percentage := 0.0
		if totalPods > 0 {
			percentage = float64(result.NoRequestLimitPods) / float64(totalPods) * 100
		}

		status := model.CheckStatusSuccess
		if percentage > 50 {
			status = model.CheckStatusError
		} else if percentage > 20 {
			status = model.CheckStatusWarning
		}

		result.Items = append(result.Items, model.CheckItem{
			Category:   "配置",
			Name:       "资源限制检查",
			Status:     status,
			Value:      fmt.Sprintf("%.1f%%未设置", percentage),
			Detail:     fmt.Sprintf("%d/%d个Pod未设置资源限制", result.NoRequestLimitPods, totalPods),
			Suggestion: "建议为所有容器设置资源请求和限制",
		})
	}

	return result
}

// checkCapacity 检查容量
func (s *InspectionService) checkCapacity(ctx context.Context, clientset *kubernetes.Clientset) model.CapacityResult {
	result := model.CapacityResult{
		Items: []model.CheckItem{},
	}

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return result
	}

	var totalCPU, allocatedCPU int64
	var totalMemory, allocatedMemory int64
	var totalPodCapacity, currentPodCount int

	for _, node := range nodes.Items {
		cpuCap := node.Status.Capacity.Cpu()
		memCap := node.Status.Capacity.Memory()
		podCap := node.Status.Capacity.Pods()

		totalCPU += cpuCap.MilliValue()
		totalMemory += memCap.Value()
		totalPodCapacity += int(podCap.Value())
	}

	// 获取所有Pod的资源请求
	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err == nil {
		currentPodCount = len(pods.Items)
		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodPending {
				for _, container := range pod.Spec.Containers {
					allocatedCPU += container.Resources.Requests.Cpu().MilliValue()
					allocatedMemory += container.Resources.Requests.Memory().Value()
				}
			}
		}
	}

	result.TotalCPU = fmt.Sprintf("%.2f核", float64(totalCPU)/1000)
	result.AllocatedCPU = fmt.Sprintf("%.2f核", float64(allocatedCPU)/1000)
	if totalCPU > 0 {
		result.CPUAllocatePercent = float64(allocatedCPU) / float64(totalCPU) * 100
	}

	result.TotalMemory = formatMemory(totalMemory)
	result.AllocatedMemory = formatMemory(allocatedMemory)
	if totalMemory > 0 {
		result.MemoryAllocatePercent = float64(allocatedMemory) / float64(totalMemory) * 100
	}

	result.TotalPodCapacity = totalPodCapacity
	result.CurrentPodCount = currentPodCount
	if totalPodCapacity > 0 {
		result.PodDensityPercent = float64(currentPodCount) / float64(totalPodCapacity) * 100
	}

	// 生成检查项
	cpuStatus := model.CheckStatusSuccess
	if result.CPUAllocatePercent > 85 {
		cpuStatus = model.CheckStatusError
	} else if result.CPUAllocatePercent > 70 {
		cpuStatus = model.CheckStatusWarning
	}

	result.Items = append(result.Items, model.CheckItem{
		Category:   "容量",
		Name:       "CPU容量",
		Status:     cpuStatus,
		Value:      fmt.Sprintf("%.1f%%", result.CPUAllocatePercent),
		Detail:     fmt.Sprintf("已分配: %s / 总量: %s", result.AllocatedCPU, result.TotalCPU),
		Suggestion: getCpuSuggestion(result.CPUAllocatePercent),
	})

	memStatus := model.CheckStatusSuccess
	if result.MemoryAllocatePercent > 85 {
		memStatus = model.CheckStatusError
	} else if result.MemoryAllocatePercent > 70 {
		memStatus = model.CheckStatusWarning
	}

	result.Items = append(result.Items, model.CheckItem{
		Category:   "容量",
		Name:       "内存容量",
		Status:     memStatus,
		Value:      fmt.Sprintf("%.1f%%", result.MemoryAllocatePercent),
		Detail:     fmt.Sprintf("已分配: %s / 总量: %s", result.AllocatedMemory, result.TotalMemory),
		Suggestion: getMemorySuggestion(result.MemoryAllocatePercent),
	})

	podStatus := model.CheckStatusSuccess
	if result.PodDensityPercent > 85 {
		podStatus = model.CheckStatusError
	} else if result.PodDensityPercent > 70 {
		podStatus = model.CheckStatusWarning
	}

	result.Items = append(result.Items, model.CheckItem{
		Category:   "容量",
		Name:       "Pod密度",
		Status:     podStatus,
		Value:      fmt.Sprintf("%.1f%%", result.PodDensityPercent),
		Detail:     fmt.Sprintf("当前: %d / 最大: %d", result.CurrentPodCount, result.TotalPodCapacity),
		Suggestion: getPodSuggestion(result.PodDensityPercent),
	})

	return result
}

// checkEvents 检查事件
func (s *InspectionService) checkEvents(ctx context.Context, clientset *kubernetes.Clientset) model.EventsResult {
	result := model.EventsResult{
		Items:          []model.CheckItem{},
		RecentEvents:   []model.EventInfo{},
		HighFreqEvents: []model.EventInfo{},
	}

	// 获取最近1小时的事件
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	events, err := clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		result.Items = append(result.Items, model.CheckItem{
			Category: "事件",
			Name:     "事件检查",
			Status:   model.CheckStatusError,
			Detail:   fmt.Sprintf("获取事件失败: %v", err),
		})
		return result
	}

	eventCounts := make(map[string]int)
	for _, event := range events.Items {
		if event.LastTimestamp.Time.After(oneHourAgo) {
			if event.Type == "Warning" {
				result.WarningEvents++
			}
			if event.Type == "Normal" && strings.Contains(strings.ToLower(event.Reason), "error") {
				result.ErrorEvents++
			}

			// 统计高频事件
			key := fmt.Sprintf("%s/%s/%s", event.InvolvedObject.Kind, event.InvolvedObject.Name, event.Reason)
			eventCounts[key] += int(event.Count)
		}

		// 收集最近事件
		if len(result.RecentEvents) < 10 && event.Type == "Warning" {
			result.RecentEvents = append(result.RecentEvents, model.EventInfo{
				Type:      event.Type,
				Reason:    event.Reason,
				Message:   event.Message,
				Object:    fmt.Sprintf("%s/%s", event.InvolvedObject.Kind, event.InvolvedObject.Name),
				Namespace: event.Namespace,
				Count:     int(event.Count),
				LastSeen:  event.LastTimestamp.Format("2006-01-02 15:04:05"),
			})
		}
	}

	// 找出高频事件
	type eventCount struct {
		key   string
		count int
	}
	var sortedEvents []eventCount
	for k, v := range eventCounts {
		if v > 10 {
			sortedEvents = append(sortedEvents, eventCount{k, v})
		}
	}
	sort.Slice(sortedEvents, func(i, j int) bool {
		return sortedEvents[i].count > sortedEvents[j].count
	})

	for i, ec := range sortedEvents {
		if i >= 5 {
			break
		}
		parts := strings.Split(ec.key, "/")
		result.HighFreqEvents = append(result.HighFreqEvents, model.EventInfo{
			Object: ec.key,
			Count:  ec.count,
			Reason: parts[len(parts)-1],
		})
	}

	// 生成检查项
	status := model.CheckStatusSuccess
	if result.ErrorEvents > 5 || result.WarningEvents > 50 {
		status = model.CheckStatusError
	} else if result.ErrorEvents > 0 || result.WarningEvents > 10 {
		status = model.CheckStatusWarning
	}

	result.Items = append(result.Items, model.CheckItem{
		Category:   "事件",
		Name:       "最近1小时事件",
		Status:     status,
		Value:      fmt.Sprintf("Warning: %d, Error: %d", result.WarningEvents, result.ErrorEvents),
		Detail:     "最近1小时内的告警和错误事件统计",
		Suggestion: getEventSuggestion(result.WarningEvents, result.ErrorEvents),
	})

	if len(result.HighFreqEvents) > 0 {
		result.Items = append(result.Items, model.CheckItem{
			Category:   "事件",
			Name:       "高频事件检查",
			Status:     model.CheckStatusWarning,
			Value:      fmt.Sprintf("%d个高频事件", len(result.HighFreqEvents)),
			Detail:     "存在重复次数超过10次的事件",
			Suggestion: "检查高频事件的原因，排查是否存在持续性问题",
		})
	}

	return result
}

// summarizeResults 汇总检查结果
func (s *InspectionService) summarizeResults(result *model.InspectionResult) model.InspectionSummary {
	summary := model.InspectionSummary{}

	// 收集所有检查项
	allItems := []model.CheckItem{}
	allItems = append(allItems, result.ClusterInfo.Items...)
	allItems = append(allItems, result.NodeHealth.Items...)
	allItems = append(allItems, result.Components.Items...)
	allItems = append(allItems, result.Workloads.Items...)
	allItems = append(allItems, result.Network.Items...)
	allItems = append(allItems, result.Storage.Items...)
	allItems = append(allItems, result.Security.Items...)
	allItems = append(allItems, result.Config.Items...)
	allItems = append(allItems, result.Capacity.Items...)
	allItems = append(allItems, result.Events.Items...)

	summary.TotalChecks = len(allItems)
	for _, item := range allItems {
		switch item.Status {
		case model.CheckStatusSuccess:
			summary.PassedChecks++
		case model.CheckStatusWarning:
			summary.WarningChecks++
		case model.CheckStatusError:
			summary.FailedChecks++
		}
	}

	return summary
}

// calculateScore 计算健康评分
func (s *InspectionService) calculateScore(result *model.InspectionResult) int {
	summary := result.Summary
	if summary.TotalChecks == 0 {
		return 100
	}

	// 基础分100分
	// 每个警告扣5分，每个错误扣15分
	score := 100 - (summary.WarningChecks * 5) - (summary.FailedChecks * 15)

	if score < 0 {
		score = 0
	}

	return score
}

// saveInspectionResult 保存巡检结果
func (s *InspectionService) saveInspectionResult(inspection *model.ClusterInspection, result *model.InspectionResult, startTime time.Time) {
	endTime := time.Now()
	duration := int(endTime.Sub(startTime).Seconds())

	reportData, _ := json.Marshal(result)

	inspection.Status = model.InspectionStatusCompleted
	inspection.Score = result.Score
	inspection.CheckCount = result.Summary.TotalChecks
	inspection.PassCount = result.Summary.PassedChecks
	inspection.WarningCount = result.Summary.WarningChecks
	inspection.FailCount = result.Summary.FailedChecks
	inspection.Duration = duration
	inspection.ReportData = string(reportData)
	inspection.EndTime = &endTime

	s.db.Save(inspection)
}

// updateInspectionFailed 更新巡检失败状态
func (s *InspectionService) updateInspectionFailed(inspection *model.ClusterInspection, err error) {
	endTime := time.Now()
	inspection.Status = model.InspectionStatusFailed
	inspection.EndTime = &endTime
	inspection.ReportData = fmt.Sprintf(`{"error": "%s"}`, err.Error())
	s.db.Save(inspection)
}

// GetInspectionProgress 获取巡检进度
func (s *InspectionService) GetInspectionProgress(ctx context.Context, inspectionID uint64) (*model.InspectionProgress, error) {
	var inspection model.ClusterInspection
	if err := s.db.First(&inspection, inspectionID).Error; err != nil {
		return nil, err
	}

	progress := &model.InspectionProgress{
		InspectionID: inspectionID,
		Status:       inspection.Status,
	}

	if inspection.Status == model.InspectionStatusCompleted {
		progress.Progress = 100
		progress.CurrentStep = "巡检完成"
	} else if inspection.Status == model.InspectionStatusFailed {
		progress.Progress = 0
		progress.CurrentStep = "巡检失败"
	} else {
		// 根据已有检查项估算进度
		progress.Progress = 50 // 简化处理
		progress.CurrentStep = "正在执行巡检..."
	}

	return progress, nil
}

// GetInspectionResult 获取巡检结果
func (s *InspectionService) GetInspectionResult(ctx context.Context, inspectionID uint64) (*model.ClusterInspection, *model.InspectionResult, error) {
	var inspection model.ClusterInspection
	if err := s.db.First(&inspection, inspectionID).Error; err != nil {
		return nil, nil, err
	}

	var result model.InspectionResult
	if inspection.ReportData != "" {
		if err := json.Unmarshal([]byte(inspection.ReportData), &result); err != nil {
			return &inspection, nil, nil
		}
	}

	return &inspection, &result, nil
}

// GetInspectionHistory 获取巡检历史
func (s *InspectionService) GetInspectionHistory(ctx context.Context, clusterID uint64, page, pageSize int) ([]model.InspectionHistoryItem, int64, error) {
	var total int64
	query := s.db.Model(&model.ClusterInspection{})
	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}
	query.Count(&total)

	var inspections []model.ClusterInspection
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&inspections).Error; err != nil {
		return nil, 0, err
	}

	items := make([]model.InspectionHistoryItem, len(inspections))
	for i, insp := range inspections {
		items[i] = model.InspectionHistoryItem{
			ID:           insp.ID,
			ClusterID:    insp.ClusterID,
			ClusterName:  insp.ClusterName,
			Score:        insp.Score,
			Status:       insp.Status,
			CheckCount:   insp.CheckCount,
			PassCount:    insp.PassCount,
			WarningCount: insp.WarningCount,
			FailCount:    insp.FailCount,
			Duration:     insp.Duration,
			CreatedAt:    insp.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return items, total, nil
}

// DeleteInspection 删除巡检记录
func (s *InspectionService) DeleteInspection(ctx context.Context, inspectionID uint64) error {
	return s.db.Delete(&model.ClusterInspection{}, inspectionID).Error
}

// 辅助函数
func formatAge(t time.Time) string {
	duration := time.Since(t)
	if duration.Hours() >= 24 {
		return fmt.Sprintf("%dd", int(duration.Hours()/24))
	}
	if duration.Hours() >= 1 {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	}
	return fmt.Sprintf("%dm", int(duration.Minutes()))
}

func formatMemory(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2fTi", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2fGi", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2fMi", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2fKi", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

func getCpuSuggestion(percent float64) string {
	if percent > 85 {
		return "CPU资源即将耗尽，需立即扩容"
	} else if percent > 70 {
		return "建议规划CPU扩容"
	}
	return "CPU容量充足"
}

func getMemorySuggestion(percent float64) string {
	if percent > 85 {
		return "内存资源即将耗尽，需立即扩容"
	} else if percent > 70 {
		return "建议规划内存扩容"
	}
	return "内存容量充足"
}

func getPodSuggestion(percent float64) string {
	if percent > 85 {
		return "Pod容量即将耗尽，需立即添加节点"
	} else if percent > 70 {
		return "建议规划节点扩容"
	}
	return "Pod容量充足"
}

func getEventSuggestion(warnings, errors int) string {
	if errors > 5 || warnings > 50 {
		return "存在大量告警和错误事件，需要立即排查"
	} else if errors > 0 || warnings > 10 {
		return "存在告警事件，建议关注"
	}
	return "事件状态正常"
}
