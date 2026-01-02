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
	resourceHandler := NewResourceHandler(clusterService)
	roleHandler := NewRoleHandler(db)
	roleBindingHandler := NewRoleBindingHandler(db)

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
		clusters.GET("/resources/nodes/:nodeName/metrics", resourceHandler.GetNodeMetrics)
		clusters.GET("/resources/namespaces", resourceHandler.ListNamespaces)
		clusters.GET("/resources/pods", resourceHandler.ListPods)
		clusters.GET("/resources/deployments", resourceHandler.ListDeployments)
		clusters.GET("/resources/api-groups", resourceHandler.GetAPIGroups)
		clusters.GET("/resources/api-resources", resourceHandler.GetResourcesByAPIGroup)

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
	}
}
