package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	"github.com/ydcloud-dy/opshub/pkg/response"
	k8smodel "github.com/ydcloud-dy/opshub/plugins/kubernetes/data/models"
	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
)

func (h *Handler) ListInternalLogAssets(c *gin.Context) {
	decision, ok := h.authorizeInternalAction(c, "query", parseUint(c.Query("storageId")))
	if !ok {
		return
	}
	hostQuery := h.db.WithContext(c.Request.Context()).Order("name ASC")
	policyHostIDs := make([]uint64, 0)
	policyHostGroupIDs := make([]uint64, 0)
	policyClusterIDs := make([]uint64, 0)
	if !decision.IsAdmin && decision.AllowedPolicyIDs != nil && len(decision.AllowedPolicyIDs) > 0 {
		var targets []logmodel.PolicyTarget
		if err := h.db.WithContext(c.Request.Context()).Where("policy_id IN ?", decision.AllowedPolicyIDs).Find(&targets).Error; err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "读取采集策略目标失败: "+err.Error())
			return
		}
		for _, target := range targets {
			switch target.TargetType {
			case "host":
				policyHostIDs = append(policyHostIDs, uint64(target.TargetID))
			case "host_group":
				policyHostGroupIDs = append(policyHostGroupIDs, uint64(target.TargetID))
			case "cluster":
				policyClusterIDs = append(policyClusterIDs, uint64(target.TargetID))
			}
		}
	}
	if !decision.IsAdmin {
		if len(decision.AllowedHostIDs) == 0 {
			hostQuery = hostQuery.Where("1 = 0")
		} else {
			hostQuery = hostQuery.Where("id IN ?", decision.AllowedHostIDs)
		}
		if decision.AllowedPolicyIDs != nil {
			switch {
			case len(policyHostIDs) > 0 && len(policyHostGroupIDs) > 0:
				hostQuery = hostQuery.Where("(id IN ? OR group_id IN ?)", uniqueUint64s(policyHostIDs), uniqueUint64s(policyHostGroupIDs))
			case len(policyHostIDs) > 0:
				hostQuery = hostQuery.Where("id IN ?", uniqueUint64s(policyHostIDs))
			case len(policyHostGroupIDs) > 0:
				hostQuery = hostQuery.Where("group_id IN ?", uniqueUint64s(policyHostGroupIDs))
			default:
				hostQuery = hostQuery.Where("1 = 0")
			}
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
		if decision.AllowedPolicyIDs != nil {
			if len(policyClusterIDs) == 0 {
				clusterQuery = clusterQuery.Where("1 = 0")
			} else {
				clusterQuery = clusterQuery.Where("id IN ?", uniqueUint64s(policyClusterIDs))
			}
		}
	}
	var clusters []k8smodel.Cluster
	if err := clusterQuery.Find(&clusters).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取 Kubernetes 集群失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"hosts": hostsToTargetView(hosts), "clusters": clustersToTargetView(clusters)})
}
