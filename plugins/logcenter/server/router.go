package server

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
	handler := NewHandler(db)

	group := router.Group("/logcenter")
	{
		group.GET("/overview", handler.GetOverview)

		storages := group.Group("/storages")
		{
			storages.GET("", handler.ListStorageClusters)
			storages.POST("", handler.CreateStorageCluster)
			storages.GET("/:id", handler.GetStorageCluster)
			storages.PUT("/:id", handler.UpdateStorageCluster)
			storages.DELETE("/:id", handler.DeleteStorageCluster)
			storages.POST("/:id/test", handler.TestStorageCluster)
			storages.POST("/:id/initialize", handler.InitializeStorageCluster)
		}

		internal := group.Group("/internal")
		{
			internal.POST("/query", handler.QueryInternalLogs)
			internal.POST("/query/histogram", handler.QueryInternalHistogram)
			internal.POST("/query/context", handler.QueryInternalContext)
			internal.POST("/query/options", handler.QueryInternalResourceOptions)
			internal.GET("/assets", handler.ListInternalLogAssets)
			internal.POST("/tail", handler.TailInternalLogs)
			internal.POST("/exports", handler.CreateInternalLogExport)
			internal.GET("/exports", handler.ListInternalLogExports)
			internal.GET("/exports/:id", handler.GetInternalLogExport)
			internal.GET("/exports/:id/download", handler.DownloadInternalLogExport)
			internal.GET("/fields", handler.ListInternalFields)
		}

		ingest := group.Group("/ingest")
		{
			ingest.GET("/status", handler.GetIngestStatus)
			ingest.POST("/test", handler.TestIngest)
		}

		policies := group.Group("/policies")
		{
			policies.GET("", handler.ListCollectionPolicies)
			policies.POST("", handler.CreateCollectionPolicy)
			policies.GET("/host-options", handler.ListPolicyHostOptions)
			policies.GET("/target-options", handler.ListPolicyTargetOptions)
			policies.GET("/revisions", handler.ListAllPolicyRevisions)
			policies.GET("/:id", handler.GetCollectionPolicy)
			policies.PUT("/:id", handler.UpdateCollectionPolicy)
			policies.DELETE("/:id", handler.DeleteCollectionPolicy)
			policies.POST("/:id/preview-targets", handler.PreviewPolicyTargets)
			policies.POST("/:id/publish", handler.PublishCollectionPolicy)
			policies.POST("/:id/disable", handler.DisableCollectionPolicy)
			policies.POST("/:id/rollback/:version", handler.RollbackCollectionPolicy)
			policies.GET("/:id/revisions", handler.ListPolicyRevisions)
		}

		collectors := group.Group("/collectors")
		{
			collectors.GET("/instances", handler.ListCollectorInstances)
			collectors.POST("/instances/:id/restart", handler.RestartCollectorInstance)
		}

		kubernetes := group.Group("/kubernetes/clusters")
		{
			kubernetes.GET("/:id/options", handler.GetKubernetesPolicyOptions)
			kubernetes.GET("/:id/collector/status", handler.GetKubernetesCollectorStatus)
			kubernetes.POST("/:id/collector/manifest", handler.GenerateKubernetesCollectorManifest)
			kubernetes.POST("/:id/collector/install", handler.InstallKubernetesCollector)
			kubernetes.DELETE("/:id/collector", handler.UninstallKubernetesCollector)
		}

		group.GET("/histories", handler.ListHistories)
		group.DELETE("/histories/:id", handler.DeleteHistory)
		group.POST("/histories/batch-delete", handler.BatchDeleteHistories)

		views := group.Group("/views")
		{
			views.GET("", handler.ListSavedViews)
			views.GET("/:id", handler.GetSavedView)
			views.POST("", handler.CreateSavedView)
			views.PUT("/:id", handler.UpdateSavedView)
			views.DELETE("/:id", handler.DeleteSavedView)
		}

		alerts := group.Group("/alerts")
		{
			alerts.GET("/:ruleId/context", handler.GetAlertContext)
			alerts.PUT("/:ruleId/context", handler.UpdateAlertContext)
		}

		access := group.Group("/access-policies")
		{
			access.GET("", handler.ListAccessPolicies)
			access.GET("/options", handler.GetAccessPolicyOptions)
			access.POST("", handler.CreateAccessPolicy)
			access.PUT("/:id", handler.UpdateAccessPolicy)
			access.DELETE("/:id", handler.DeleteAccessPolicy)
		}

		retention := group.Group("/retention-policies")
		{
			retention.GET("", handler.ListRetentionPolicies)
			retention.POST("", handler.CreateRetentionPolicy)
			retention.PUT("/:id", handler.UpdateRetentionPolicy)
			retention.DELETE("/:id", handler.DeleteRetentionPolicy)
		}
		group.GET("/capacity", handler.GetStorageCapacity)

		templates := group.Group("/templates")
		{
			templates.GET("", handler.ListTemplates)
			templates.GET("/:id", handler.GetTemplate)
			templates.POST("", handler.CreateTemplate)
			templates.PUT("/:id", handler.UpdateTemplate)
			templates.DELETE("/:id", handler.DeleteTemplate)
			templates.POST("/:id/clone", handler.CloneTemplate)
		}

		library := group.Group("/library")
		{
			library.GET("", handler.ListLibrary)
			library.GET("/:id", handler.GetLibraryItem)
			library.PUT("/:id", handler.UpdateLibraryItem)
			library.DELETE("/:id", handler.DeleteLibraryItem)
			library.GET("/:id/fields", handler.ListLibraryFields)
			library.PUT("/:id/fields/:fieldId", handler.UpdateLibraryField)
		}

	}
}
