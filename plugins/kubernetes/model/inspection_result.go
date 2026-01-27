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

package model

// InspectionResult 巡检结果
type InspectionResult struct {
	ClusterID   uint64            `json:"clusterId"`
	ClusterName string            `json:"clusterName"`
	Score       int               `json:"score"`
	Summary     InspectionSummary `json:"summary"`
	ClusterInfo ClusterInfoResult `json:"clusterInfo"`
	NodeHealth  NodeHealthResult  `json:"nodeHealth"`
	Components  ComponentsResult  `json:"components"`
	Workloads   WorkloadsResult   `json:"workloads"`
	Network     NetworkResult     `json:"network"`
	Storage     StorageResult     `json:"storage"`
	Security    SecurityResult    `json:"security"`
	Config      ConfigResult      `json:"config"`
	Capacity    CapacityResult    `json:"capacity"`
	Events      EventsResult      `json:"events"`
}

// InspectionSummary 巡检摘要
type InspectionSummary struct {
	TotalChecks   int `json:"totalChecks"`
	PassedChecks  int `json:"passedChecks"`
	WarningChecks int `json:"warningChecks"`
	FailedChecks  int `json:"failedChecks"`
	Duration      int `json:"duration"` // 秒
}

// CheckItem 检查项
type CheckItem struct {
	Category   string `json:"category"`   // 类别
	Name       string `json:"name"`       // 检查项名称
	Status     string `json:"status"`     // success/warning/error
	Value      string `json:"value"`      // 检查值
	Expected   string `json:"expected"`   // 期望值
	Detail     string `json:"detail"`     // 详细信息
	Suggestion string `json:"suggestion"` // 优化建议
}

// ClusterInfoResult 集群信息检查结果
type ClusterInfoResult struct {
	Version         string      `json:"version"`
	Platform        string      `json:"platform"`
	GitVersion      string      `json:"gitVersion"`
	GoVersion       string      `json:"goVersion"`
	BuildDate       string      `json:"buildDate"`
	ConnectionState string      `json:"connectionState"` // connected/failed
	ConnectionDelay int64       `json:"connectionDelay"` // 连接延迟(毫秒)
	Items           []CheckItem `json:"items"`
}

// NodeHealthResult 节点健康检查结果
type NodeHealthResult struct {
	TotalNodes       int            `json:"totalNodes"`
	ReadyNodes       int            `json:"readyNodes"`
	NotReadyNodes    int            `json:"notReadyNodes"`
	PressureNodes    int            `json:"pressureNodes"`  // 有资源压力的节点数
	TaintedNodes     int            `json:"taintedNodes"`   // 有污点的节点数
	NodeUtilization  []NodeResource `json:"nodeUtilization"`
	Items            []CheckItem    `json:"items"`
}

// NodeResource 节点资源使用情况
type NodeResource struct {
	Name            string  `json:"name"`
	CPUCapacity     string  `json:"cpuCapacity"`
	CPUUsed         string  `json:"cpuUsed"`
	CPUUsagePercent float64 `json:"cpuUsagePercent"`
	MemoryCapacity  string  `json:"memoryCapacity"`
	MemoryUsed      string  `json:"memoryUsed"`
	MemoryPercent   float64 `json:"memoryPercent"`
	PodCount        int     `json:"podCount"`
	PodCapacity     int     `json:"podCapacity"`
	Status          string  `json:"status"` // Ready/NotReady
}

// ComponentsResult 组件检查结果
type ComponentsResult struct {
	ControlPlane []ComponentStatus `json:"controlPlane"`
	Addons       []ComponentStatus `json:"addons"`
	Items        []CheckItem       `json:"items"`
}

// ComponentStatus 组件状态
type ComponentStatus struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"` // Healthy/Unhealthy
	Ready     string `json:"ready"`
	Restarts  int    `json:"restarts"`
	Age       string `json:"age"`
}

// WorkloadsResult 工作负载检查结果
type WorkloadsResult struct {
	TotalDeployments  int             `json:"totalDeployments"`
	HealthyDeployments int            `json:"healthyDeployments"`
	TotalDaemonSets   int             `json:"totalDaemonSets"`
	HealthyDaemonSets int             `json:"healthyDaemonSets"`
	TotalStatefulSets int             `json:"totalStatefulSets"`
	HealthyStatefulSets int           `json:"healthyStatefulSets"`
	TotalPods         int             `json:"totalPods"`
	RunningPods       int             `json:"runningPods"`
	PendingPods       int             `json:"pendingPods"`
	FailedPods        int             `json:"failedPods"`
	HighRestartPods   int             `json:"highRestartPods"` // 重启次数>5的Pod数
	ImagePullErrors   int             `json:"imagePullErrors"` // 镜像拉取错误数
	PodsByPhase       map[string]int  `json:"podsByPhase"`
	Items             []CheckItem     `json:"items"`
	UnhealthyWorkloads []WorkloadInfo `json:"unhealthyWorkloads"`
}

// WorkloadInfo 工作负载信息
type WorkloadInfo struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Ready     string `json:"ready"`
	Status    string `json:"status"`
	Reason    string `json:"reason"`
}

// NetworkResult 网络检查结果
type NetworkResult struct {
	TotalServices        int         `json:"totalServices"`
	ClusterIPServices    int         `json:"clusterIPServices"`
	NodePortServices     int         `json:"nodePortServices"`
	LoadBalancerServices int         `json:"loadBalancerServices"`
	NoEndpointServices   int         `json:"noEndpointServices"`
	TotalIngresses       int         `json:"totalIngresses"`
	NetworkPolicies      int         `json:"networkPolicies"`
	Items                []CheckItem `json:"items"`
}

// StorageResult 存储检查结果
type StorageResult struct {
	TotalPVs       int         `json:"totalPVs"`
	AvailablePVs   int         `json:"availablePVs"`
	BoundPVs       int         `json:"boundPVs"`
	ReleasedPVs    int         `json:"releasedPVs"`
	FailedPVs      int         `json:"failedPVs"`
	TotalPVCs      int         `json:"totalPVCs"`
	BoundPVCs      int         `json:"boundPVCs"`
	PendingPVCs    int         `json:"pendingPVCs"`
	StorageClasses int         `json:"storageClasses"`
	DefaultSC      string      `json:"defaultSC"` // 默认StorageClass名称
	Items          []CheckItem `json:"items"`
}

// SecurityResult 安全检查结果
type SecurityResult struct {
	ServiceAccounts     int         `json:"serviceAccounts"`
	Roles               int         `json:"roles"`
	ClusterRoles        int         `json:"clusterRoles"`
	RoleBindings        int         `json:"roleBindings"`
	ClusterRoleBindings int         `json:"clusterRoleBindings"`
	PrivilegedPods      int         `json:"privilegedPods"`   // 特权Pod数
	HostNetworkPods     int         `json:"hostNetworkPods"`  // 使用hostNetwork的Pod数
	RootUserContainers  int         `json:"rootUserContainers"` // 以root用户运行的容器数
	ClusterAdminBindings int        `json:"clusterAdminBindings"` // cluster-admin绑定数
	Items               []CheckItem `json:"items"`
}

// ConfigResult 配置检查结果
type ConfigResult struct {
	TotalConfigMaps      int         `json:"totalConfigMaps"`
	TotalSecrets         int         `json:"totalSecrets"`
	NamespaceCount       int         `json:"namespaceCount"`
	ResourceQuotaCount   int         `json:"resourceQuotaCount"`
	LimitRangeCount      int         `json:"limitRangeCount"`
	NoRequestLimitPods   int         `json:"noRequestLimitPods"`   // 未设置资源限制的Pod数
	Items                []CheckItem `json:"items"`
}

// CapacityResult 容量检查结果
type CapacityResult struct {
	TotalCPU          string  `json:"totalCPU"`          // 总CPU
	AllocatedCPU      string  `json:"allocatedCPU"`      // 已分配CPU
	CPUAllocatePercent float64 `json:"cpuAllocatePercent"` // CPU分配比例
	TotalMemory       string  `json:"totalMemory"`       // 总内存
	AllocatedMemory   string  `json:"allocatedMemory"`   // 已分配内存
	MemoryAllocatePercent float64 `json:"memoryAllocatePercent"` // 内存分配比例
	TotalPodCapacity  int     `json:"totalPodCapacity"`  // 总Pod容量
	CurrentPodCount   int     `json:"currentPodCount"`   // 当前Pod数
	PodDensityPercent float64 `json:"podDensityPercent"` // Pod密度
	Items             []CheckItem `json:"items"`
}

// EventsResult 事件检查结果
type EventsResult struct {
	WarningEvents    int           `json:"warningEvents"`
	ErrorEvents      int           `json:"errorEvents"`
	RecentEvents     []EventInfo   `json:"recentEvents"`
	HighFreqEvents   []EventInfo   `json:"highFreqEvents"` // 高频事件
	Items            []CheckItem   `json:"items"`
}

// EventInfo 事件信息
type EventInfo struct {
	Type      string `json:"type"`
	Reason    string `json:"reason"`
	Message   string `json:"message"`
	Object    string `json:"object"`
	Namespace string `json:"namespace"`
	Count     int    `json:"count"`
	LastSeen  string `json:"lastSeen"`
}

// StartInspectionRequest 开始巡检请求
type StartInspectionRequest struct {
	ClusterIDs []uint64          `json:"clusterIds"`
	Options    InspectionOptions `json:"options"`
	UserID     uint64            `json:"userId"`
}

// InspectionOptions 巡检选项
type InspectionOptions struct {
	CheckCluster   bool `json:"checkCluster"`
	CheckNodes     bool `json:"checkNodes"`
	CheckWorkloads bool `json:"checkWorkloads"`
	CheckNetwork   bool `json:"checkNetwork"`
	CheckStorage   bool `json:"checkStorage"`
	CheckSecurity  bool `json:"checkSecurity"`
	CheckConfig    bool `json:"checkConfig"`
	CheckCapacity  bool `json:"checkCapacity"`
	CheckEvents    bool `json:"checkEvents"`
}

// InspectionProgress 巡检进度
type InspectionProgress struct {
	InspectionID      uint64 `json:"inspectionId"`
	Status            string `json:"status"`
	Progress          int    `json:"progress"` // 0-100
	CurrentStep       string `json:"currentStep"`
	CompletedClusters int    `json:"completedClusters"`
	TotalClusters     int    `json:"totalClusters"`
}

// InspectionHistoryItem 巡检历史记录项
type InspectionHistoryItem struct {
	ID          uint64 `json:"id"`
	ClusterID   uint64 `json:"clusterId"`
	ClusterName string `json:"clusterName"`
	Score       int    `json:"score"`
	Status      string `json:"status"`
	CheckCount  int    `json:"checkCount"`
	PassCount   int    `json:"passCount"`
	WarningCount int   `json:"warningCount"`
	FailCount   int    `json:"failCount"`
	Duration    int    `json:"duration"`
	CreatedAt   string `json:"createdAt"`
}
