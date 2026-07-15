package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	"github.com/ydcloud-dy/opshub/pkg/response"
	k8smodel "github.com/ydcloud-dy/opshub/plugins/kubernetes/data/models"
)

func (h *Handler) ListInternalLogAssets(c *gin.Context) {
	decision, ok := h.authorizeInternalAction(c, "query", parseUint(c.Query("storageId")))
	if !ok {
		return
	}
	hostQuery := h.db.WithContext(c.Request.Context()).Order("name ASC")
	if !decision.IsAdmin {
		if len(decision.AllowedHostIDs) == 0 {
			hostQuery = hostQuery.Where("1 = 0")
		} else {
			hostQuery = hostQuery.Where("id IN ?", decision.AllowedHostIDs)
		}
	}
	var hosts []assetbiz.Host
	if err := hostQuery.Find(&hosts).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取主机列表失败: "+err.Error())
		return
	}
	clusterQuery := h.db.WithContext(c.Request.Context()).Select("id", "name", "alias", "version", "node_count", "status").Order("name ASC")
	if !decision.IsAdmin {
		clusterIDs := make([]uint64, 0, len(decision.AllowedKubernetesScopes))
		for clusterID := range decision.AllowedKubernetesScopes {
			clusterIDs = append(clusterIDs, clusterID)
		}
		if len(clusterIDs) == 0 {
			clusterQuery = clusterQuery.Where("1 = 0")
		} else {
			clusterQuery = clusterQuery.Where("id IN ?", clusterIDs)
		}
	}
	var clusters []k8smodel.Cluster
	if err := clusterQuery.Find(&clusters).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取 Kubernetes 集群失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"hosts": hostsToTargetView(hosts), "clusters": clustersToTargetView(clusters)})
}
