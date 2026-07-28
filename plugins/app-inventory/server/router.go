package server

import (
	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/plugins/app-inventory/service"
)

func RegisterRoutes(router *gin.RouterGroup, svc *service.Service) {
	h := NewHandler(svc)
	group := router.Group("/app-inventory")
	{
		group.GET("/overview", h.Overview)
		group.GET("/topology", h.Topology)
		group.GET("/references", h.References)

		apps := group.Group("/apps")
		{
			apps.GET("", h.ListApplications)
			apps.POST("", h.CreateApplication)
			apps.GET("/:id", h.GetApplication)
			apps.PUT("/:id", h.UpdateApplication)
			apps.DELETE("/:id", h.DeleteApplication)
			apps.POST("/:id/probe", h.ProbeApplication)
			apps.GET("/:id/topology", h.Topology)
		}

		environments := group.Group("/environments")
		{
			environments.GET("", h.ListEnvironments)
			environments.POST("", h.CreateEnvironment)
			environments.PUT("/:id", h.UpdateEnvironment)
			environments.DELETE("/:id", h.DeleteEnvironment)
		}

		domains := group.Group("/domains")
		{
			domains.GET("", h.ListDomains)
			domains.POST("", h.CreateDomain)
			domains.PUT("/:id", h.UpdateDomain)
			domains.DELETE("/:id", h.DeleteDomain)
			domains.POST("/:id/probe", h.ProbeDomain)
		}

		resources := group.Group("/resources")
		{
			resources.GET("", h.ListResources)
			resources.POST("", h.CreateResource)
			resources.PUT("/:id", h.UpdateResource)
			resources.DELETE("/:id", h.DeleteResource)
			resources.POST("/:id/probe", h.ProbeResource)
		}

		components := group.Group("/components")
		{
			components.GET("", h.ListComponents)
			components.POST("", h.CreateComponent)
			components.PUT("/:id", h.UpdateComponent)
			components.DELETE("/:id", h.DeleteComponent)
			components.POST("/:id/probe", h.ProbeComponent)
		}

		dependencies := group.Group("/dependencies")
		{
			dependencies.GET("", h.ListDependencies)
			dependencies.POST("", h.CreateDependency)
			dependencies.PUT("/:id", h.UpdateDependency)
			dependencies.DELETE("/:id", h.DeleteDependency)
		}

		credentials := group.Group("/credentials")
		{
			credentials.GET("", h.ListCredentials)
			credentials.POST("", h.CreateCredential)
			credentials.PUT("/:id", h.UpdateCredential)
			credentials.DELETE("/:id", h.DeleteCredential)
			credentials.POST("/:id/reveal", h.RevealCredential)
			credentials.GET("/:id/grants", h.ListGrants)
			credentials.POST("/:id/grants", h.UpsertGrant)
			credentials.DELETE("/grants/:id", h.DeleteGrant)
			credentials.GET("/audits", h.ListSecretAudits)
		}

		discovery := group.Group("/discovery")
		{
			discovery.GET("/kubernetes/namespaces", h.ListKubernetesNamespaces)
			discovery.POST("/kubernetes/preview", h.DiscoverKubernetes)
			discovery.POST("/kubernetes/import", h.ImportKubernetes)
			discovery.GET("/runs", h.ListDiscoveryRuns)
		}
	}
}
