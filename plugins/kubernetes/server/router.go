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

package server

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ydcloud-dy/opshub/plugins/kubernetes/service"
)

// RegisterRoutes 注册路由
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
	clusterHandler := NewClusterHandler(db)
	clusterService := service.NewClusterService(db)
	resourceHandler := NewResourceHandler(clusterService, db)
	roleHandler := NewRoleHandler(db)
	roleBindingHandler := NewRoleBindingHandler(db)
	arthasHandler := NewArthasHandler(clusterService, db)
	inspectionHandler := NewInspectionHandler(clusterService, db)

	clusters := router.Group("/kubernetes")
	{
		// 集群管理
		clusters.POST("/clusters", clusterHandler.CreateCluster)
		clusters.GET("/clusters", clusterHandler.ListClusters)
		clusters.GET("/clusters/:id", clusterHandler.GetCluster)
		clusters.PUT("/clusters/:id", clusterHandler.UpdateCluster)
		clusters.DELETE("/clusters/:id", clusterHandler.DeleteCluster)
		clusters.POST("/clusters/:id/test", clusterHandler.TestClusterConnection)
		clusters.GET("/clusters/:id/config", clusterHandler.GetClusterConfig)

		// 集群状态同步
		clusters.POST("/clusters/:id/sync", clusterHandler.SyncClusterStatus)
		clusters.POST("/clusters/sync-all", clusterHandler.SyncAllClustersStatus)

		// KubeConfig管理 (更具体的路由要放在前面)
		clusters.POST("/clusters/kubeconfig/sa", clusterHandler.GetServiceAccountKubeConfig)
		clusters.POST("/clusters/kubeconfig", clusterHandler.GenerateKubeConfig)
		clusters.DELETE("/clusters/kubeconfig", clusterHandler.RevokeKubeConfig)
		clusters.DELETE("/clusters/kubeconfig/revoke", clusterHandler.RevokeCredentialFully)
		clusters.GET("/clusters/kubeconfig/existing", clusterHandler.GetExistingKubeConfig)

		// 资源查询
		clusters.GET("/resources/nodes", resourceHandler.ListNodes)
		clusters.GET("/resources/nodes/:nodeName/yaml", resourceHandler.GetNodeYAML)
		clusters.PUT("/resources/nodes/:nodeName/yaml", resourceHandler.UpdateNodeYAML)
		clusters.POST("/resources/nodes/:nodeName/drain", resourceHandler.DrainNode)
		clusters.POST("/resources/nodes/:nodeName/cordon", resourceHandler.CordonNode)
		clusters.POST("/resources/nodes/:nodeName/uncordon", resourceHandler.UncordonNode)
		clusters.DELETE("/resources/nodes/:nodeName", resourceHandler.DeleteNode)
		clusters.GET("/resources/nodes/:nodeName/metrics", resourceHandler.GetNodeMetrics)

		// 节点批量操作
		clusters.POST("/resources/nodes/batch/drain", resourceHandler.BatchDrainNodes)
		clusters.POST("/resources/nodes/batch/cordon", resourceHandler.BatchCordonNodes)
		clusters.POST("/resources/nodes/batch/uncordon", resourceHandler.BatchUncordonNodes)
		clusters.POST("/resources/nodes/batch/delete", resourceHandler.BatchDeleteNodes)
		clusters.POST("/resources/nodes/batch/labels", resourceHandler.BatchUpdateNodeLabels)
		clusters.POST("/resources/nodes/batch/taints", resourceHandler.BatchUpdateNodeTaints)

		// Shell WebSocket
		clusters.GET("/shell/nodes/:nodeName", resourceHandler.NodeShellWebSocket)
		clusters.GET("/shell/pods", resourceHandler.PodShellWebSocket)

		// CloudTTY 管理
		clusters.GET("/cloudtty/status", resourceHandler.GetCloudTTYStatus)
		clusters.POST("/cloudtty/deploy", resourceHandler.DeployCloudTTY)
		clusters.GET("/cloudtty/service", resourceHandler.GetCloudTTYService)
		clusters.POST("/cloudtty/service", resourceHandler.CreateCloudTTYService)

		clusters.GET("/resources/namespaces", resourceHandler.ListNamespaces)
		clusters.POST("/resources/namespaces", resourceHandler.CreateNamespace)
		clusters.GET("/resources/namespaces/:namespaceName/yaml", resourceHandler.GetNamespaceYAML)
		clusters.PUT("/resources/namespaces/:namespaceName/yaml", resourceHandler.UpdateNamespaceYAML)
		clusters.DELETE("/resources/namespaces/:namespaceName", resourceHandler.DeleteNamespace)

		// 访问控制资源
		clusters.GET("/resources/serviceaccounts", resourceHandler.ListServiceAccounts)
		clusters.POST("/resources/serviceaccounts/:namespace/yaml", resourceHandler.CreateServiceAccountFromYAML)
		clusters.GET("/resources/serviceaccounts/:namespace/:name/yaml", resourceHandler.GetServiceAccountYAML)
		clusters.PUT("/resources/serviceaccounts/:namespace/:name/yaml", resourceHandler.UpdateServiceAccountYAML)
		clusters.DELETE("/resources/serviceaccounts/:namespace/:name", resourceHandler.DeleteServiceAccount)
		clusters.POST("/resources/roles/:namespace/yaml", resourceHandler.CreateRoleFromYAML)
		clusters.GET("/resources/roles/:namespace/:name/yaml", resourceHandler.GetRoleYAML)
		clusters.PUT("/resources/roles/:namespace/:name/yaml", resourceHandler.UpdateRoleYAML)
		clusters.DELETE("/resources/roles/:namespace/:name", resourceHandler.DeleteRole)

		clusters.POST("/resources/rolebindings/:namespace/yaml", resourceHandler.CreateRoleBindingFromYAML)
		clusters.GET("/resources/rolebindings/:namespace/:name/yaml", resourceHandler.GetRoleBindingYAML)
		clusters.PUT("/resources/rolebindings/:namespace/:name/yaml", resourceHandler.UpdateRoleBindingYAML)
		clusters.DELETE("/resources/rolebindings/:namespace/:name", resourceHandler.DeleteRoleBinding)

		clusters.POST("/resources/clusterroles/yaml", resourceHandler.CreateClusterRoleFromYAML)
		clusters.GET("/resources/clusterroles/:name/yaml", resourceHandler.GetClusterRoleYAML)
		clusters.PUT("/resources/clusterroles/:name/yaml", resourceHandler.UpdateClusterRoleYAML)
		clusters.DELETE("/resources/clusterroles/:name", resourceHandler.DeleteClusterRole)

		clusters.POST("/resources/clusterrolebindings/yaml", resourceHandler.CreateClusterRoleBindingFromYAML)
		clusters.GET("/resources/clusterrolebindings/:name/yaml", resourceHandler.GetClusterRoleBindingYAML)
		clusters.PUT("/resources/clusterrolebindings/:name/yaml", resourceHandler.UpdateClusterRoleBindingYAML)
		clusters.DELETE("/resources/clusterrolebindings/:name", resourceHandler.DeleteClusterRoleBinding)

		clusters.GET("/resources/roles", resourceHandler.ListRoles)
		clusters.GET("/resources/rolebindings", resourceHandler.ListRoleBindings)
		clusters.GET("/resources/clusterroles", resourceHandler.ListClusterRoles)
		clusters.GET("/resources/clusterrolebindings", resourceHandler.ListClusterRoleBindings)
		clusters.GET("/resources/podsecuritypolicies", resourceHandler.ListPodSecurityPolicies)

		clusters.GET("/resources/pods", resourceHandler.ListPods)
		clusters.GET("/resources/pods/:namespace/:name", resourceHandler.GetPodDetail)
		clusters.GET("/resources/pods/:namespace/:name/events", resourceHandler.GetPodEvents)
		clusters.GET("/resources/pods/metrics", resourceHandler.GetPodsMetrics)
		clusters.GET("/resources/pods/logs", resourceHandler.GetPodLogs)
		clusters.GET("/pods/files", resourceHandler.ListContainerFiles)
		clusters.GET("/pods/files/download", resourceHandler.DownloadContainerFile)
		clusters.POST("/pods/files/upload", resourceHandler.UploadContainerFile)
		clusters.GET("/resources/deployments", resourceHandler.ListDeployments)
		clusters.GET("/resources/workloads", resourceHandler.GetWorkloads)
		clusters.GET("/resources/workloads/:namespace/:name", resourceHandler.GetWorkloadDetail)
		clusters.GET("/resources/workloads/:namespace/:name/replicasets", resourceHandler.GetWorkloadReplicaSets)
		clusters.GET("/resources/workloads/:namespace/:name/pods", resourceHandler.GetWorkloadPods)
		clusters.GET("/resources/workloads/:namespace/:name/services", resourceHandler.GetWorkloadServices)
		clusters.GET("/resources/workloads/:namespace/:name/ingresses", resourceHandler.GetWorkloadIngresses)
		clusters.GET("/resources/workloads/:namespace/:name/yaml", resourceHandler.GetWorkloadYAML)
		clusters.PUT("/resources/workloads/:namespace/:name/yaml", resourceHandler.UpdateWorkloadYAML)
		clusters.POST("/workloads/update", resourceHandler.UpdateWorkload)
		clusters.POST("/workloads/pause", resourceHandler.PauseWorkload)
		clusters.POST("/workloads/rollback", resourceHandler.RollbackWorkload)
		clusters.POST("/resources/workloads/create", resourceHandler.CreateWorkloadFromYAML)
		clusters.DELETE("/resources/workloads/:namespace/:name", resourceHandler.DeleteWorkload)

		// 工作负载批量操作
		clusters.POST("/resources/workloads/batch/delete", resourceHandler.BatchDeleteWorkloads)
		clusters.POST("/resources/workloads/batch/restart", resourceHandler.BatchRestartWorkloads)
		clusters.POST("/resources/workloads/batch/pause", resourceHandler.BatchPauseWorkloads)
		clusters.POST("/resources/workloads/batch/resume", resourceHandler.BatchResumeWorkloads)

		clusters.GET("/resources/api-groups", resourceHandler.GetAPIGroups)
		clusters.GET("/resources/api-resources", resourceHandler.GetResourcesByAPIGroup)

		// 网络资源管理 - Service
		clusters.GET("/resources/services", resourceHandler.ListServices)
		clusters.GET("/resources/services/:namespace/:name/yaml", resourceHandler.GetServiceYAML)
		clusters.PUT("/resources/services/:namespace/:name/yaml", resourceHandler.UpdateServiceYAML)
		clusters.POST("/resources/services/:namespace/:name", resourceHandler.CreateService)
		clusters.DELETE("/resources/services/:namespace/:name", resourceHandler.DeleteService)

		// 网络资源管理 - Ingress
		clusters.GET("/resources/ingresses", resourceHandler.ListIngresses)
		clusters.GET("/resources/ingresses/:namespace/:name/yaml", resourceHandler.GetIngressYAML)
		clusters.PUT("/resources/ingresses/:namespace/:name/yaml", resourceHandler.UpdateIngressYAML)
		clusters.POST("/resources/ingresses/:namespace/:name", resourceHandler.CreateIngress)
		clusters.DELETE("/resources/ingresses/:namespace/:name", resourceHandler.DeleteIngress)

		// 网络资源管理 - Endpoints
		clusters.GET("/resources/endpoints", resourceHandler.ListEndpoints)
		clusters.POST("/resources/endpoints/:namespace/yaml", resourceHandler.CreateEndpointYAML)
		clusters.GET("/resources/endpoints/:namespace/:name/yaml", resourceHandler.GetEndpointYAML)
		clusters.PUT("/resources/endpoints/:namespace/:name/yaml", resourceHandler.UpdateEndpointYAML)
		clusters.DELETE("/resources/endpoints/:namespace/:name", resourceHandler.DeleteEndpoint)
		clusters.GET("/resources/endpoints/:namespace/:name", resourceHandler.GetEndpointsDetail)

		// 网络资源管理 - NetworkPolicy
		clusters.GET("/resources/networkpolicies", resourceHandler.ListNetworkPolicies)
		clusters.POST("/resources/networkpolicies/:namespace/:name", resourceHandler.CreateNetworkPolicy)
		clusters.GET("/resources/networkpolicies/:namespace/:name/yaml", resourceHandler.GetNetworkPolicyYAML)
		clusters.PUT("/resources/networkpolicies/:namespace/:name/yaml", resourceHandler.UpdateNetworkPolicyYAML)
		clusters.DELETE("/resources/networkpolicies/:namespace/:name", resourceHandler.DeleteNetworkPolicy)

		// 配置管理 - ConfigMap
		clusters.GET("/resources/configmaps", resourceHandler.ListConfigMaps)
		clusters.POST("/resources/configmaps/:namespace/yaml", resourceHandler.CreateConfigMapFromYAML)
		clusters.GET("/resources/configmaps/:namespace/:name/yaml", resourceHandler.GetConfigMapYAML)
		clusters.PUT("/resources/configmaps/:namespace/:name/yaml", resourceHandler.UpdateConfigMapYAML)
		clusters.DELETE("/resources/configmaps/:namespace/:name", resourceHandler.DeleteConfigMap)

		// 配置管理 - Secret
		clusters.GET("/resources/secrets", resourceHandler.ListSecrets)
		clusters.POST("/resources/secrets/:namespace/yaml", resourceHandler.CreateSecretFromYAML)
		clusters.GET("/resources/secrets/:namespace/:name/yaml", resourceHandler.GetSecretYAML)
		clusters.PUT("/resources/secrets/:namespace/:name/yaml", resourceHandler.UpdateSecretYAML)
		clusters.DELETE("/resources/secrets/:namespace/:name", resourceHandler.DeleteSecret)

		// 存储管理 - PersistentVolumeClaim
		clusters.GET("/resources/persistentvolumeclaims", resourceHandler.ListPersistentVolumeClaims)
		clusters.GET("/resources/persistentvolumeclaims/:namespace/:name/yaml", resourceHandler.GetPersistentVolumeClaimYAML)
		clusters.PUT("/resources/persistentvolumeclaims/:namespace/:name/yaml", resourceHandler.UpdatePersistentVolumeClaimYAML)
		clusters.DELETE("/resources/persistentvolumeclaims/:namespace/:name", resourceHandler.DeletePersistentVolumeClaim)
		clusters.POST("/resources/persistentvolumeclaims/:namespace/yaml", resourceHandler.CreatePersistentVolumeClaimYAML)

		// 存储管理 - PersistentVolume
		clusters.GET("/resources/persistentvolumes", resourceHandler.ListPersistentVolumes)
		clusters.POST("/resources/persistentvolumes/yaml", resourceHandler.CreatePersistentVolumeYAML)
		clusters.GET("/resources/persistentvolumes/:name/yaml", resourceHandler.GetPersistentVolumeYAML)
		clusters.PUT("/resources/persistentvolumes/:name/yaml", resourceHandler.UpdatePersistentVolumeYAML)
		clusters.DELETE("/resources/persistentvolumes/:name", resourceHandler.DeletePersistentVolume)

		// 存储管理 - StorageClass
		clusters.GET("/resources/storageclasses", resourceHandler.ListStorageClasses)
		clusters.POST("/resources/storageclasses/yaml", resourceHandler.CreateStorageClassYAML)
		clusters.GET("/resources/storageclasses/:name/yaml", resourceHandler.GetStorageClassYAML)
		clusters.PUT("/resources/storageclasses/:name/yaml", resourceHandler.UpdateStorageClassYAML)
		clusters.DELETE("/resources/storageclasses/:name", resourceHandler.DeleteStorageClass)

		// 配置管理 - ResourceQuota
		clusters.GET("/resources/resourcequotas", resourceHandler.ListResourceQuotas)
		clusters.POST("/resources/resourcequotas/:namespace/yaml", resourceHandler.CreateResourceQuotaFromYAML)
		clusters.GET("/resources/resourcequotas/:namespace/:name/yaml", resourceHandler.GetResourceQuotaYAML)
		clusters.PUT("/resources/resourcequotas/:namespace/:name/yaml", resourceHandler.UpdateResourceQuotaYAML)
		clusters.DELETE("/resources/resourcequotas/:namespace/:name", resourceHandler.DeleteResourceQuota)

		// 配置管理 - LimitRange
		clusters.GET("/resources/limitranges", resourceHandler.ListLimitRanges)
		clusters.POST("/resources/limitranges/:namespace/yaml", resourceHandler.CreateLimitRangeFromYAML)
		clusters.GET("/resources/limitranges/:namespace/:name/yaml", resourceHandler.GetLimitRangeYAML)
		clusters.PUT("/resources/limitranges/:namespace/:name/yaml", resourceHandler.UpdateLimitRangeYAML)
		clusters.DELETE("/resources/limitranges/:namespace/:name", resourceHandler.DeleteLimitRange)

		// 配置管理 - HorizontalPodAutoscaler
		clusters.GET("/resources/horizontalpodautoscalers", resourceHandler.ListHorizontalPodAutoscalers)
		clusters.POST("/resources/horizontalpodautoscalers/:namespace/yaml", resourceHandler.CreateHPAFromYAML)
		clusters.GET("/resources/horizontalpodautoscalers/:namespace/:name/yaml", resourceHandler.GetHorizontalPodAutoscalerYAML)
		clusters.PUT("/resources/horizontalpodautoscalers/:namespace/:name/yaml", resourceHandler.UpdateHorizontalPodAutoscalerYAML)
		clusters.DELETE("/resources/horizontalpodautoscalers/:namespace/:name", resourceHandler.DeleteHorizontalPodAutoscaler)

		// 配置管理 - PodDisruptionBudget
		clusters.GET("/resources/poddisruptionbudgets", resourceHandler.ListPodDisruptionBudgets)
		clusters.POST("/resources/poddisruptionbudgets/:namespace/yaml", resourceHandler.CreatePDBFromYAML)
		clusters.GET("/resources/poddisruptionbudgets/:namespace/:name/yaml", resourceHandler.GetPodDisruptionBudgetYAML)
		clusters.PUT("/resources/poddisruptionbudgets/:namespace/:name/yaml", resourceHandler.UpdatePodDisruptionBudgetYAML)
		clusters.DELETE("/resources/poddisruptionbudgets/:namespace/:name", resourceHandler.DeletePodDisruptionBudget)

		// 终端审计
		clusters.GET("/terminal/sessions", resourceHandler.ListTerminalSessions)
		clusters.GET("/terminal/sessions/:id/play", resourceHandler.PlayTerminalSession)
		clusters.DELETE("/terminal/sessions/:id", resourceHandler.DeleteTerminalSession)

		// 统计信息
		clusters.GET("/resources/stats", resourceHandler.GetClusterStats)
		clusters.GET("/resources/network", resourceHandler.GetClusterNetworkInfo)
		clusters.GET("/resources/components", resourceHandler.GetClusterComponentInfo)
		clusters.GET("/resources/events", resourceHandler.ListEvents)

		// 角色管理
		clusters.GET("/roles/cluster", roleHandler.ListClusterRoles)
		clusters.POST("/roles/create-defaults", roleHandler.CreateDefaultClusterRoles)
		clusters.POST("/roles/create-defaults-namespace", roleHandler.CreateDefaultNamespaceRoles)
		clusters.GET("/roles/namespaces", roleHandler.ListNamespaces)
		clusters.GET("/roles/namespace", roleHandler.ListNamespaceRoles)
		clusters.GET("/roles/:namespace/:name", roleHandler.GetRoleDetail)
		clusters.DELETE("/roles/:namespace/:name", roleHandler.DeleteRole)
		clusters.POST("/clusters/:id/roles", roleHandler.CreateRole)

		// 角色绑定管理
		clusters.POST("/role-bindings/bind", roleBindingHandler.BindUserToRole)
		clusters.DELETE("/role-bindings/unbind", roleBindingHandler.UnbindUserFromRole)
		clusters.GET("/role-bindings/users", roleBindingHandler.GetRoleBoundUsers)
		clusters.GET("/role-bindings/available-users", roleBindingHandler.GetAvailableUsers)
		clusters.GET("/role-bindings/user-roles", roleBindingHandler.GetUserClusterRoles)
		clusters.GET("/role-bindings/user-bindings", roleBindingHandler.GetUserRoleBindings)
		clusters.GET("/role-bindings/credential-users", roleBindingHandler.GetClusterCredentialUsers)

		// Arthas 应用诊断
		arthas := clusters.Group("/arthas")
		{
			// 基础检测
			arthas.GET("/java-processes", arthasHandler.ListJavaProcesses)
			arthas.GET("/check", arthasHandler.CheckArthasInstalled)
			arthas.POST("/install", arthasHandler.InstallArthas)

			// 一次性命令
			arthas.POST("/command", arthasHandler.ExecuteArthasCommand)
			arthas.GET("/dashboard", arthasHandler.GetDashboard)
			arthas.GET("/thread", arthasHandler.GetThreadList)
			arthas.GET("/thread/stack", arthasHandler.GetThreadStack)
			arthas.GET("/jvm", arthasHandler.GetJvmInfo)
			arthas.GET("/sysenv", arthasHandler.GetSysEnv)
			arthas.GET("/sysprop", arthasHandler.GetSysProp)
			arthas.GET("/perfcounter", arthasHandler.GetPerfCounter)
			arthas.GET("/memory", arthasHandler.GetMemory)

			// 类和方法
			arthas.GET("/jad", arthasHandler.DecompileClass)
			arthas.GET("/getstatic", arthasHandler.GetStaticField)
			arthas.GET("/sc", arthasHandler.SearchClass)
			arthas.GET("/sm", arthasHandler.SearchMethod)

			// 火焰图
			arthas.GET("/profiler", arthasHandler.GenerateFlameGraph)

			// WebSocket 实时命令（trace, watch, monitor等）
			arthas.GET("/ws", arthasHandler.ArthasWebSocket)
		}

		// 集群巡检
		inspection := clusters.Group("/inspection")
		{
			inspection.POST("/start", inspectionHandler.StartInspection)
			inspection.GET("/progress/:inspectionId", inspectionHandler.GetInspectionProgress)
			inspection.GET("/result/:inspectionId", inspectionHandler.GetInspectionResult)
			inspection.GET("/history", inspectionHandler.GetInspectionHistory)
			inspection.DELETE("/:inspectionId", inspectionHandler.DeleteInspection)
			inspection.GET("/export/:inspectionId", inspectionHandler.ExportInspection)
		}

	}
}
