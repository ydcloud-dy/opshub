package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	"github.com/ydcloud-dy/opshub/internal/logagent"
	rbacsvc "github.com/ydcloud-dy/opshub/internal/service/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
	k8smodel "github.com/ydcloud-dy/opshub/plugins/kubernetes/data/models"
	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"k8s.io/apimachinery/pkg/labels"
)

type policyTargetInput struct {
	TargetType       string   `json:"targetType"`
	TargetID         uint     `json:"targetId"`
	Namespace        string   `json:"namespace"`
	WorkloadKind     string   `json:"workloadKind"`
	WorkloadName     string   `json:"workloadName"`
	LabelSelector    string   `json:"labelSelector"`
	ContainerInclude []string `json:"containerInclude"`
	ContainerExclude []string `json:"containerExclude"`
}

type policyPayload struct {
	Name              string                   `json:"name"`
	SourceMode        string                   `json:"sourceMode"`
	Description       string                   `json:"description"`
	Paths             []string                 `json:"paths"`
	ExcludePaths      []string                 `json:"excludePaths"`
	ReadFrom          string                   `json:"readFrom"`
	Encoding          string                   `json:"encoding"`
	Environment       string                   `json:"environment"`
	Service           string                   `json:"service"`
	Stream            string                   `json:"stream"`
	MaxLineBytes      int                      `json:"maxLineBytes"`
	Parser            logagent.ParserConfig    `json:"parser"`
	Multiline         logagent.MultilineConfig `json:"multiline"`
	Redaction         logagent.RedactionConfig `json:"redaction"`
	Retention         logagent.RetentionConfig `json:"retention"`
	RetentionPolicyID uint                     `json:"retentionPolicyId"`
	RetentionDays     int                      `json:"retentionDays"`
	WALMaxBytes       int64                    `json:"walMaxBytes"`
	Targets           []policyTargetInput      `json:"targets"`
}

type policyView struct {
	ID                uint                                `json:"id"`
	Status            string                              `json:"status"`
	Version           uint64                              `json:"version"`
	HasDraft          bool                                `json:"hasUnpublishedChanges"`
	CreatedBy         uint                                `json:"createdBy"`
	UpdatedBy         uint                                `json:"updatedBy"`
	CreatedAt         time.Time                           `json:"createdAt"`
	UpdatedAt         time.Time                           `json:"updatedAt"`
	Payload           policyPayload                       `json:"payload"`
	DraftPayload      *policyPayload                      `json:"draftPayload,omitempty"`
	TargetCount       int                                 `json:"targetCount"`
	TargetExpected    int                                 `json:"targetExpected"`
	InstanceTotal     int64                               `json:"instanceTotal"`
	InstanceOnline    int64                               `json:"instanceOnline"`
	InstanceApplied   int64                               `json:"instanceApplied"`
	InstancePending   int64                               `json:"instancePending"`
	ErrorInstances    int64                               `json:"errorInstances"`
	TargetHosts       []policyTargetHost                  `json:"targetHosts"`
	TargetClusters    []policyTargetCluster               `json:"targetClusters"`
	CollectorShutdown []kubernetesCollectorShutdownResult `json:"collectorShutdown,omitempty"`
}

type policyTargetHost struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	IP           string `json:"ip"`
	GroupID      uint   `json:"groupId"`
	AgentID      string `json:"agentId"`
	AgentVersion string `json:"agentVersion"`
	AgentStatus  string `json:"agentStatus"`
}

type policyTargetGroup struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	ParentID  uint   `json:"parentId"`
	HostCount int64  `json:"hostCount"`
}

type policyTargetCluster struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Alias     string `json:"alias"`
	Version   string `json:"version"`
	NodeCount int    `json:"nodeCount"`
	Status    int    `json:"status"`
}

type policyRevisionView struct {
	logmodel.PolicyRevision
	PolicyName string `json:"policyName"`
}

const (
	collectionPolicyStatusDraft     = "draft"
	collectionPolicyStatusPublished = "published"
	collectionPolicyStatusDisabled  = "disabled"
	collectionPolicyStatusArchived  = "archived"
)

func (h *Handler) ListCollectionPolicies(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
		return
	}
	query := h.db.WithContext(c.Request.Context()).Model(&logmodel.CollectionPolicy{}).Order("updated_at DESC")
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	} else {
		query = query.Where("status <> ?", collectionPolicyStatusArchived)
	}
	var policies []logmodel.CollectionPolicy
	if err := query.Find(&policies).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取采集策略失败: "+err.Error())
		return
	}
	views := make([]policyView, 0, len(policies))
	for _, policy := range policies {
		view, err := h.buildPolicyView(c, policy)
		if err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "读取采集策略详情失败: "+err.Error())
			return
		}
		views = append(views, view)
	}
	response.Success(c, views)
}

func (h *Handler) GetCollectionPolicy(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
		return
	}
	policy, ok := h.findCollectionPolicy(c)
	if !ok {
		return
	}
	view, err := h.buildPolicyView(c, policy)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取采集策略详情失败: "+err.Error())
		return
	}
	response.Success(c, view)
}

func (h *Handler) CreateCollectionPolicy(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
		return
	}
	var payload policyPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	payload.normalize()
	if err := applyRetentionPolicySnapshot(h.db.WithContext(c.Request.Context()), &payload); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validatePolicyPayload(payload); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	userID := rbacsvc.GetUserID(c)
	policy := payload.toModel()
	policy.Status = "draft"
	policy.CreatedBy = userID
	policy.UpdatedBy = userID
	if err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&policy).Error; err != nil {
			return err
		}
		return replacePolicyTargets(tx, policy.ID, payload.Targets)
	}); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建采集策略失败: "+err.Error())
		return
	}
	view, _ := h.buildPolicyView(c, policy)
	response.Success(c, view)
}

func (h *Handler) UpdateCollectionPolicy(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
		return
	}
	policy, ok := h.findCollectionPolicy(c)
	if !ok {
		return
	}
	var payload policyPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	payload.normalize()
	if err := applyRetentionPolicySnapshot(h.db.WithContext(c.Request.Context()), &payload); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validatePolicyPayload(payload); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	if policy.Status == collectionPolicyStatusArchived {
		response.ErrorCode(c, http.StatusConflict, "已归档策略不可编辑，请先恢复策略")
		return
	}
	userID := rbacsvc.GetUserID(c)
	if policy.Status == collectionPolicyStatusPublished && policy.Version > 0 {
		if err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			if err := savePendingPolicyDraft(tx, policy.ID, payload, userID); err != nil {
				return err
			}
			return tx.Model(&policy).Updates(map[string]interface{}{"updated_by": userID}).Error
		}); err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "保存待发布策略失败: "+err.Error())
			return
		}
		_ = h.db.WithContext(c.Request.Context()).First(&policy, policy.ID).Error
		view, _ := h.buildPolicyView(c, policy)
		response.Success(c, view)
		return
	}
	payload.applyToModel(&policy)
	policy.Status = collectionPolicyStatusDraft
	policy.UpdatedBy = userID
	if err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&policy).Error; err != nil {
			return err
		}
		if err := replacePolicyTargets(tx, policy.ID, payload.Targets); err != nil {
			return err
		}
		return tx.Where("policy_id = ? AND version = ?", policy.ID, 0).Delete(&logmodel.PolicyRevision{}).Error
	}); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新采集策略失败: "+err.Error())
		return
	}
	view, _ := h.buildPolicyView(c, policy)
	response.Success(c, view)
}

func (h *Handler) DeleteCollectionPolicy(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
		return
	}
	policyID := parseUint(c.Param("id"))
	if policyID == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "策略 ID 无效")
		return
	}
	var action string
	var shutdownClusters []k8smodel.Cluster
	err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var policy logmodel.CollectionPolicy
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&policy, policyID).Error; err != nil {
			return err
		}
		policyAction, decisionErr := collectionPolicyDeleteAction(policy)
		if decisionErr != nil {
			return decisionErr
		}
		switch policyAction {
		case "deleted":
			if err := deleteCollectionPolicyData(tx, policy.ID); err != nil {
				return err
			}
			if err := tx.Delete(&policy).Error; err != nil {
				return err
			}
			action = "deleted"
		case "archived":
			var activeAssignments int64
			if err := tx.Model(&logmodel.CollectorAssignment{}).
				Where("policy_id = ? AND desired_state = ?", policy.ID, "active").Count(&activeAssignments).Error; err != nil {
				return err
			}
			if activeAssignments > 0 {
				return &policyLifecycleConflict{message: "策略仍有活动采集实例，请先完成停用下发后再归档"}
			}
			assignmentUpdates := map[string]interface{}{"desired_state": "disabled", "apply_status": "pending", "last_error": ""}
			if policy.SourceMode == "kubernetes" {
				now := time.Now()
				assignmentUpdates["apply_status"] = "disabled"
				assignmentUpdates["applied_at"] = &now
				clusters, err := resolvePolicyClusters(tx, policy.ID)
				if err != nil {
					return err
				}
				shutdownClusters = clusters
			}
			if err := tx.Model(&logmodel.CollectorAssignment{}).Where("policy_id = ?", policy.ID).Updates(assignmentUpdates).Error; err != nil {
				return err
			}
			policy.Status = collectionPolicyStatusArchived
			policy.UpdatedBy = rbacsvc.GetUserID(c)
			if err := tx.Save(&policy).Error; err != nil {
				return err
			}
			action = "archived"
		}
		return nil
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.ErrorCode(c, http.StatusNotFound, "采集策略不存在")
			return
		}
		if conflict, ok := err.(*policyLifecycleConflict); ok {
			response.ErrorCode(c, http.StatusConflict, conflict.Error())
			return
		}
		response.ErrorCode(c, http.StatusInternalServerError, "删除或归档采集策略失败: "+err.Error())
		return
	}
	result := gin.H{"id": policyID, "action": action}
	if action == "archived" && len(shutdownClusters) > 0 {
		result["collectorShutdown"] = shutdownUnusedKubernetesCollectors(
			c.Request.Context(), uint(policyID), shutdownClusters,
			h.countOtherPublishedKubernetesPolicies, h.uninstallKubernetesCollector,
		)
	}
	response.Success(c, result)
}

func (h *Handler) RestoreCollectionPolicy(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
		return
	}
	policyID := parseUint(c.Param("id"))
	if policyID == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "策略 ID 无效")
		return
	}
	var policy logmodel.CollectionPolicy
	err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&policy, policyID).Error; err != nil {
			return err
		}
		if policy.Status != collectionPolicyStatusArchived {
			return &policyLifecycleConflict{message: "只有已归档策略可以恢复"}
		}
		policy.Status = collectionPolicyStatusDraft
		policy.UpdatedBy = rbacsvc.GetUserID(c)
		return tx.Save(&policy).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.ErrorCode(c, http.StatusNotFound, "采集策略不存在")
			return
		}
		if conflict, ok := err.(*policyLifecycleConflict); ok {
			response.ErrorCode(c, http.StatusConflict, conflict.Error())
			return
		}
		response.ErrorCode(c, http.StatusInternalServerError, "恢复采集策略失败: "+err.Error())
		return
	}
	view, viewErr := h.buildPolicyView(c, policy)
	if viewErr != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取恢复后的策略失败: "+viewErr.Error())
		return
	}
	response.Success(c, view)
}

func (h *Handler) PublishCollectionPolicy(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
		return
	}
	policyID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		ChangeSummary string `json:"changeSummary"`
	}
	_ = c.ShouldBindJSON(&req)
	var published logmodel.CollectionPolicy
	err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var policy logmodel.CollectionPolicy
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&policy, uint(policyID)).Error; err != nil {
			return err
		}
		if policy.Status == collectionPolicyStatusArchived {
			return &policyLifecycleConflict{message: "已归档策略不能直接发布，请先恢复策略"}
		}
		if err := applyPendingPolicyDraft(tx, &policy); err != nil {
			return err
		}
		result, err := publishPolicy(tx, policy, rbacsvc.GetUserID(c), strings.TrimSpace(req.ChangeSummary))
		published = result
		return err
	})
	if err != nil {
		status := http.StatusInternalServerError
		if err == gorm.ErrRecordNotFound {
			status = http.StatusNotFound
		} else if conflict, ok := err.(*policyLifecycleConflict); ok {
			status = http.StatusConflict
			response.ErrorCode(c, status, "发布采集策略失败: "+conflict.Error())
			return
		}
		response.ErrorCode(c, status, "发布采集策略失败: "+err.Error())
		return
	}
	view, _ := h.buildPolicyView(c, published)
	response.Success(c, view)
}

func (h *Handler) DisableCollectionPolicy(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
		return
	}
	policy, ok := h.findCollectionPolicy(c)
	if !ok {
		return
	}
	if policy.Status == collectionPolicyStatusArchived {
		response.ErrorCode(c, http.StatusConflict, "已归档策略不能停用")
		return
	}
	var req struct {
		UninstallCollectors *bool `json:"uninstallCollectors"`
	}
	_ = c.ShouldBindJSON(&req)
	var targetClusters []k8smodel.Cluster
	uninstallCollectors := policy.SourceMode == "kubernetes"
	if req.UninstallCollectors != nil {
		uninstallCollectors = *req.UninstallCollectors
	}
	if uninstallCollectors && policy.SourceMode == "kubernetes" {
		var err error
		targetClusters, err = resolvePolicyClusters(h.db.WithContext(c.Request.Context()), policy.ID)
		if err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "读取策略目标集群失败: "+err.Error())
			return
		}
	}
	policy.Status = "disabled"
	policy.UpdatedBy = rbacsvc.GetUserID(c)
	err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&policy).Error; err != nil {
			return err
		}
		return tx.Model(&logmodel.CollectorAssignment{}).Where("policy_id = ?", policy.ID).Updates(map[string]interface{}{
			"desired_state": "disabled", "apply_status": "pending", "last_error": "", "applied_at": nil,
		}).Error
	})
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "停用采集策略失败: "+err.Error())
		return
	}
	view, _ := h.buildPolicyView(c, policy)
	if uninstallCollectors && len(targetClusters) > 0 {
		view.CollectorShutdown = shutdownUnusedKubernetesCollectors(
			c.Request.Context(), policy.ID, targetClusters,
			h.countOtherPublishedKubernetesPolicies, h.uninstallKubernetesCollector,
		)
	}
	response.Success(c, view)
}

func (h *Handler) RollbackCollectionPolicy(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
		return
	}
	policy, ok := h.findCollectionPolicy(c)
	if !ok {
		return
	}
	if policy.Status == collectionPolicyStatusArchived {
		response.ErrorCode(c, http.StatusConflict, "已归档策略不能直接回滚，请先恢复策略")
		return
	}
	version, _ := strconv.ParseUint(c.Param("version"), 10, 64)
	var revision logmodel.PolicyRevision
	if err := h.db.WithContext(c.Request.Context()).Where("policy_id = ? AND version = ?", policy.ID, version).First(&revision).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "目标版本不存在")
		return
	}
	var payload policyPayload
	if err := json.Unmarshal([]byte(revision.Content), &payload); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "解析历史版本失败: "+err.Error())
		return
	}
	payload.normalize()
	var published logmodel.CollectionPolicy
	err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		payload.applyToModel(&policy)
		policy.UpdatedBy = rbacsvc.GetUserID(c)
		if err := tx.Save(&policy).Error; err != nil {
			return err
		}
		if err := replacePolicyTargets(tx, policy.ID, payload.Targets); err != nil {
			return err
		}
		if err := tx.Where("policy_id = ? AND version = ?", policy.ID, 0).Delete(&logmodel.PolicyRevision{}).Error; err != nil {
			return err
		}
		result, err := publishPolicy(tx, policy, rbacsvc.GetUserID(c), fmt.Sprintf("回滚到 v%d", version))
		published = result
		return err
	})
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "回滚采集策略失败: "+err.Error())
		return
	}
	view, _ := h.buildPolicyView(c, published)
	response.Success(c, view)
}

func (h *Handler) ListPolicyRevisions(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
		return
	}
	policyID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var revisions []logmodel.PolicyRevision
	if err := h.db.WithContext(c.Request.Context()).Where("policy_id = ? AND version > ?", policyID, 0).Order("version DESC").Find(&revisions).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取发布记录失败: "+err.Error())
		return
	}
	for index := range revisions {
		revisions[index].Content = ""
	}
	response.Success(c, revisions)
}

func (h *Handler) ListAllPolicyRevisions(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
		return
	}
	page, pageSize := parsePage(c)
	var revisions []logmodel.PolicyRevision
	query := h.db.WithContext(c.Request.Context()).Model(&logmodel.PolicyRevision{}).Where("version > ?", 0)
	if policyID := parseUint(c.Query("policyId")); policyID > 0 {
		query = query.Where("policy_id = ?", policyID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "统计发布记录失败: "+err.Error())
		return
	}
	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&revisions).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取发布记录失败: "+err.Error())
		return
	}
	policyIDs := make([]uint, 0, len(revisions))
	for _, revision := range revisions {
		policyIDs = append(policyIDs, revision.PolicyID)
	}
	policyNames := make(map[uint]string)
	if len(policyIDs) > 0 {
		var policies []logmodel.CollectionPolicy
		if err := h.db.Select("id", "name").Where("id IN ?", policyIDs).Find(&policies).Error; err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "读取策略名称失败: "+err.Error())
			return
		}
		for _, policy := range policies {
			policyNames[policy.ID] = policy.Name
		}
	}
	result := make([]policyRevisionView, 0, len(revisions))
	for _, revision := range revisions {
		revision.Content = ""
		result = append(result, policyRevisionView{PolicyRevision: revision, PolicyName: policyNames[revision.PolicyID]})
	}
	response.Success(c, gin.H{"total": total, "page": page, "pageSize": pageSize, "data": result})
}

func (h *Handler) PreviewPolicyTargets(c *gin.Context) {
	policy, ok := h.findCollectionPolicy(c)
	if !ok {
		return
	}
	payload, _, err := loadPolicyViewPayload(h.db.WithContext(c.Request.Context()), policy)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取策略目标失败: "+err.Error())
		return
	}
	if payload.SourceMode == "kubernetes" {
		clusters, err := resolvePolicyClustersFromInputs(h.db.WithContext(c.Request.Context()), payload.Targets)
		if err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "预览目标集群失败: "+err.Error())
			return
		}
		response.Success(c, clustersToTargetView(clusters))
		return
	}
	hosts, err := resolvePolicyHostsFromInputs(h.db.WithContext(c.Request.Context()), payload.Targets)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "预览目标主机失败: "+err.Error())
		return
	}
	response.Success(c, hostsToTargetView(hosts))
}

func (h *Handler) ListPolicyHostOptions(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
		return
	}
	var hosts []assetbiz.Host
	if err := h.db.WithContext(c.Request.Context()).Order("name ASC").Find(&hosts).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取主机列表失败: "+err.Error())
		return
	}
	response.Success(c, hostsToTargetView(hosts))
}

func (h *Handler) ListPolicyTargetOptions(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
		return
	}
	var hosts []assetbiz.Host
	if err := h.db.WithContext(c.Request.Context()).Order("name ASC").Find(&hosts).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取主机列表失败: "+err.Error())
		return
	}
	var groups []assetbiz.AssetGroup
	if err := h.db.WithContext(c.Request.Context()).Where("status = ?", 1).Order("sort ASC, name ASC").Find(&groups).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取主机分组失败: "+err.Error())
		return
	}
	groupViews := make([]policyTargetGroup, 0, len(groups))
	for _, group := range groups {
		var count int64
		h.db.Model(&assetbiz.Host{}).Where("group_id = ?", group.ID).Count(&count)
		groupViews = append(groupViews, policyTargetGroup{ID: group.ID, Name: group.Name, ParentID: group.ParentID, HostCount: count})
	}
	var clusters []k8smodel.Cluster
	if err := h.db.WithContext(c.Request.Context()).Select("id", "name", "alias", "version", "node_count", "status").Order("name ASC").Find(&clusters).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取 Kubernetes 集群失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{
		"hosts": hostsToTargetView(hosts), "groups": groupViews, "clusters": clustersToTargetView(clusters),
	})
}

type collectorInstanceView struct {
	Status                    string                         `json:"status"`
	RuntimeStatus             string                         `json:"runtimeStatus"`
	LifecycleStatus           string                         `json:"lifecycleStatus"`
	CollectorCredentialStatus string                         `json:"collectorCredentialStatus,omitempty"`
	ActivePolicyCount         int64                          `json:"activePolicyCount"`
	Instance                  logmodel.CollectorInstance     `json:"instance"`
	Assignments               []logmodel.CollectorAssignment `json:"assignments"`
}

func (h *Handler) ListCollectorInstances(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
		return
	}
	ctx := c.Request.Context()
	var instances []logmodel.CollectorInstance
	if err := h.db.WithContext(ctx).Order("updated_at DESC").Find(&instances).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取采集实例失败: "+err.Error())
		return
	}
	credentialStatuses, err := h.clusterCollectorCredentialStatuses(ctx)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取集群采集凭据失败: "+err.Error())
		return
	}
	activePolicyCounts, err := h.activeKubernetesPolicyCounts(ctx)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "统计 Kubernetes 策略失败: "+err.Error())
		return
	}
	now := time.Now()
	result := make([]collectorInstanceView, 0, len(instances))
	for _, instance := range instances {
		runtimeStatus := collectorRuntimeStatus(instance, now)
		instance.Status = runtimeStatus
		credentialStatus := credentialStatuses[instance.ClusterID]
		activePolicyCount := activePolicyCounts[instance.ClusterID]
		lifecycleStatus := collectorLifecycleStatus(instance, runtimeStatus, credentialStatus, activePolicyCount)
		var assignments []logmodel.CollectorAssignment
		if err := h.db.WithContext(ctx).Where("instance_id = ?", instance.InstanceID).Order("policy_id ASC").Find(&assignments).Error; err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "读取采集实例下发状态失败: "+err.Error())
			return
		}
		result = append(result, collectorInstanceView{
			Status: runtimeStatus, RuntimeStatus: runtimeStatus, LifecycleStatus: lifecycleStatus,
			CollectorCredentialStatus: credentialStatus, ActivePolicyCount: activePolicyCount,
			Instance: instance, Assignments: assignments,
		})
	}
	response.Success(c, result)
}

func (h *Handler) clusterCollectorCredentialStatuses(ctx context.Context) (map[uint]string, error) {
	var credentials []logmodel.ClusterCollectorCredential
	if err := h.db.WithContext(ctx).Select("cluster_id", "status").Find(&credentials).Error; err != nil {
		return nil, err
	}
	result := make(map[uint]string, len(credentials))
	for _, credential := range credentials {
		result[credential.ClusterID] = strings.ToLower(strings.TrimSpace(credential.Status))
	}
	return result, nil
}

func (h *Handler) activeKubernetesPolicyCounts(ctx context.Context) (map[uint]int64, error) {
	type clusterPolicyCount struct {
		ClusterID uint
		Count     int64
	}
	var rows []clusterPolicyCount
	err := h.db.WithContext(ctx).Table("log_policy_targets AS targets").
		Select("targets.target_id AS cluster_id, COUNT(DISTINCT policies.id) AS count").
		Joins("JOIN log_collection_policies AS policies ON policies.id = targets.policy_id").
		Where("targets.target_type = ? AND policies.source_mode = ? AND policies.status = ?", "cluster", "kubernetes", collectionPolicyStatusPublished).
		Group("targets.target_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[uint]int64, len(rows))
	for _, row := range rows {
		result[row.ClusterID] = row.Count
	}
	return result, nil
}

func collectorRuntimeStatus(instance logmodel.CollectorInstance, now time.Time) string {
	status := strings.ToLower(strings.TrimSpace(instance.Status))
	if status == "" {
		status = "offline"
	}
	if instance.LastHeartbeatAt == nil || now.Sub(*instance.LastHeartbeatAt) > 90*time.Second {
		return "offline"
	}
	if status == "offline" {
		return "online"
	}
	return status
}

func collectorLifecycleStatus(instance logmodel.CollectorInstance, runtimeStatus, credentialStatus string, activePolicyCount int64) string {
	if instance.Mode != "kubernetes-node" {
		return "active"
	}
	credentialStatus = strings.ToLower(strings.TrimSpace(credentialStatus))
	if credentialStatus == "revoked" {
		return "retired"
	}
	if activePolicyCount == 0 {
		if credentialStatus == "active" {
			return "idle"
		}
		return "retired"
	}
	return "active"
}

func (h *Handler) DeleteCollectorInstance(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
		return
	}
	instanceID := c.Param("id")
	if strings.TrimSpace(instanceID) == "" {
		response.ErrorCode(c, http.StatusBadRequest, "采集实例 ID 不能为空")
		return
	}
	var instance logmodel.CollectorInstance
	if err := h.db.WithContext(c.Request.Context()).Where("instance_id = ?", instanceID).First(&instance).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.ErrorCode(c, http.StatusNotFound, "采集实例不存在")
			return
		}
		response.ErrorCode(c, http.StatusInternalServerError, "读取采集实例失败: "+err.Error())
		return
	}
	if collectorRuntimeStatus(instance, time.Now()) == "online" {
		response.ErrorCode(c, http.StatusConflict, "采集实例仍在线，不能清理")
		return
	}
	var activeAssignments int64
	if err := h.db.WithContext(c.Request.Context()).Model(&logmodel.CollectorAssignment{}).
		Where("instance_id = ? AND desired_state = ?", instance.InstanceID, "active").Count(&activeAssignments).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "检查采集实例策略失败: "+err.Error())
		return
	}
	if activeAssignments > 0 {
		response.ErrorCode(c, http.StatusConflict, "采集实例仍有关联的活动策略，请先停用或调整策略")
		return
	}
	if err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("instance_id = ?", instance.InstanceID).Delete(&logmodel.CollectorAssignment{}).Error; err != nil {
			return err
		}
		return tx.Delete(&instance).Error
	}); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "清理采集实例失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"instanceId": instance.InstanceID})
}

func (h *Handler) RestartCollectorInstance(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
		return
	}
	instanceID := c.Param("id")
	result := h.db.WithContext(c.Request.Context()).Model(&logmodel.CollectorInstance{}).Where("instance_id = ?", instanceID).
		UpdateColumn("reload_generation", gorm.Expr("reload_generation + 1"))
	if result.Error != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "请求重载失败: "+result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		response.ErrorCode(c, http.StatusNotFound, "采集实例不存在")
		return
	}
	h.db.Model(&logmodel.CollectorAssignment{}).Where("instance_id = ?", instanceID).Update("apply_status", "pending")
	response.Success(c, gin.H{"instanceId": instanceID})
}

func publishPolicy(tx *gorm.DB, policy logmodel.CollectionPolicy, userID uint, summary string) (logmodel.CollectionPolicy, error) {
	payload, err := loadPolicyPayload(tx, policy)
	if err != nil {
		return policy, err
	}
	if err := applyRetentionPolicySnapshot(tx, &payload); err != nil {
		return policy, err
	}
	payload.normalize()
	if err := validatePolicyPayload(payload); err != nil {
		return policy, err
	}
	payload.applyToModel(&policy)
	instanceIDs := make(map[string]struct{})
	if payload.SourceMode == "kubernetes" {
		clusters, err := resolvePolicyClusters(tx, policy.ID)
		if err != nil {
			return policy, err
		}
		if len(clusters) == 0 {
			return policy, fmt.Errorf("策略没有匹配到 Kubernetes 集群")
		}
		clusterIDs := make([]uint, 0, len(clusters))
		for _, cluster := range clusters {
			clusterIDs = append(clusterIDs, cluster.ID)
		}
		var instances []logmodel.CollectorInstance
		if err := tx.Where("mode = ? AND cluster_id IN ?", "kubernetes-node", clusterIDs).Find(&instances).Error; err != nil {
			return policy, err
		}
		for _, instance := range instances {
			instanceIDs[instance.InstanceID] = struct{}{}
		}
	} else {
		hosts, err := resolvePolicyHosts(tx, policy.ID)
		if err != nil {
			return policy, err
		}
		if len(hosts) == 0 {
			return policy, fmt.Errorf("策略没有匹配到主机")
		}
		for _, host := range hosts {
			if strings.TrimSpace(host.AgentID) == "" {
				continue
			}
			instance := logmodel.CollectorInstance{
				InstanceID: host.AgentID, AgentID: host.AgentID, Mode: "host", HostID: host.ID,
				Hostname: firstNonEmpty(host.Hostname, host.Name), CollectorType: "opshub-agent",
				Version: host.AgentVersion, Status: firstNonEmpty(host.AgentStatus, "offline"),
			}
			if err := tx.Where("instance_id = ?", host.AgentID).Assign(instance).FirstOrCreate(&instance).Error; err != nil {
				return policy, err
			}
			instanceIDs[host.AgentID] = struct{}{}
		}
	}
	content, _ := json.Marshal(payload)
	checksumRaw := sha256.Sum256(content)
	policy.Version++
	policy.Status = "published"
	policy.UpdatedBy = userID
	revision := logmodel.PolicyRevision{
		PolicyID: policy.ID, Version: policy.Version, Content: string(content),
		Checksum: hex.EncodeToString(checksumRaw[:]), ChangeSummary: firstNonEmpty(summary, fmt.Sprintf("发布 v%d", policy.Version)), CreatedBy: userID,
	}
	if err := tx.Create(&revision).Error; err != nil {
		return policy, err
	}
	if err := tx.Save(&policy).Error; err != nil {
		return policy, err
	}
	if err := tx.Model(&logmodel.CollectorAssignment{}).Where("policy_id = ?", policy.ID).Updates(map[string]interface{}{
		"desired_state": "disabled", "apply_status": "pending", "applied_at": nil,
	}).Error; err != nil {
		return policy, err
	}
	for instanceID := range instanceIDs {
		assignment := logmodel.CollectorAssignment{
			InstanceID: instanceID, PolicyID: policy.ID, PolicyVersion: policy.Version,
			DesiredState: "active", ApplyStatus: "pending",
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "instance_id"}, {Name: "policy_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"policy_version": policy.Version, "desired_state": "active", "apply_status": "pending", "applied_at": nil, "last_error": "",
			}),
		}).Create(&assignment).Error; err != nil {
			return policy, err
		}
	}
	return policy, nil
}

func (h *Handler) buildPolicyView(c *gin.Context, policy logmodel.CollectionPolicy) (policyView, error) {
	db := h.db.WithContext(c.Request.Context())
	payload, err := loadPolicyPayload(db, policy)
	if err != nil {
		return policyView{}, err
	}
	draftPayload, hasDraft, err := loadPendingPolicyDraft(db, policy.ID)
	if err != nil {
		return policyView{}, err
	}
	var draftView *policyPayload
	if hasDraft {
		draftView = &draftPayload
	}
	var targetCount, targetExpected int
	var targetHosts []policyTargetHost
	var targetClusters []policyTargetCluster
	if payload.SourceMode == "kubernetes" {
		clusters, err := resolvePolicyClustersFromInputs(h.db.WithContext(c.Request.Context()), payload.Targets)
		if err != nil {
			return policyView{}, err
		}
		targetClusters = clustersToTargetView(clusters)
		targetCount = len(targetClusters)
		for _, cluster := range targetClusters {
			targetExpected += cluster.NodeCount
		}
	} else {
		hosts, err := resolvePolicyHostsFromInputs(h.db.WithContext(c.Request.Context()), payload.Targets)
		if err != nil {
			return policyView{}, err
		}
		targetHosts = hostsToTargetView(hosts)
		targetCount = len(targetHosts)
		targetExpected = len(targetHosts)
	}
	var total, online, applied, failed int64
	activeAssignment := "policy_id = ? AND desired_state = ?"
	h.db.Model(&logmodel.CollectorAssignment{}).Where(activeAssignment, policy.ID, "active").Count(&total)
	h.db.Model(&logmodel.CollectorAssignment{}).Where(activeAssignment+" AND apply_status = ?", policy.ID, "active", "applied").Count(&applied)
	h.db.Model(&logmodel.CollectorAssignment{}).Where(activeAssignment+" AND apply_status = ?", policy.ID, "active", "failed").Count(&failed)
	h.db.Table("log_collector_assignments AS assignment").
		Joins("JOIN log_collector_instances AS instance ON instance.instance_id = assignment.instance_id").
		Where("assignment.policy_id = ? AND assignment.desired_state = ? AND instance.last_heartbeat_at >= ?", policy.ID, "active", time.Now().Add(-90*time.Second)).
		Count(&online)
	pending := total - applied - failed
	if pending < 0 {
		pending = 0
	}
	return policyView{
		ID: policy.ID, Status: policy.Status, Version: policy.Version, HasDraft: hasDraft, CreatedBy: policy.CreatedBy, UpdatedBy: policy.UpdatedBy,
		CreatedAt: policy.CreatedAt, UpdatedAt: policy.UpdatedAt, Payload: payload, DraftPayload: draftView,
		TargetCount: targetCount, TargetExpected: targetExpected,
		InstanceTotal: total, InstanceOnline: online, InstanceApplied: applied, InstancePending: pending, ErrorInstances: failed,
		TargetHosts: targetHosts, TargetClusters: targetClusters,
	}, nil
}

func (h *Handler) findCollectionPolicy(c *gin.Context) (logmodel.CollectionPolicy, bool) {
	var policy logmodel.CollectionPolicy
	if err := h.db.WithContext(c.Request.Context()).First(&policy, c.Param("id")).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "采集策略不存在")
		return policy, false
	}
	return policy, true
}

type policyLifecycleConflict struct {
	message string
}

func (e *policyLifecycleConflict) Error() string { return e.message }

func collectionPolicyDeleteAction(policy logmodel.CollectionPolicy) (string, error) {
	switch {
	case policy.Status == collectionPolicyStatusDraft && policy.Version == 0:
		return "deleted", nil
	case policy.Status == collectionPolicyStatusDisabled:
		return "archived", nil
	case policy.Status == collectionPolicyStatusPublished:
		return "", &policyLifecycleConflict{message: "已发布策略不能直接删除，请先停用策略"}
	case policy.Status == collectionPolicyStatusArchived:
		return "", &policyLifecycleConflict{message: "策略已经归档"}
	default:
		return "", &policyLifecycleConflict{message: "当前策略状态不允许删除"}
	}
}

func deleteCollectionPolicyData(tx *gorm.DB, policyID uint) error {
	for _, item := range []interface{}{&logmodel.PolicyTarget{}, &logmodel.PolicyRevision{}, &logmodel.CollectorAssignment{}} {
		if err := tx.Where("policy_id = ?", policyID).Delete(item).Error; err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) requirePolicyAdmin(c *gin.Context) bool {
	return h.requireLogAdmin(c, "只有管理员可以管理日志采集策略")
}

func (h *Handler) requireAccessPolicyAdmin(c *gin.Context) bool {
	return h.requireLogAdmin(c, "只有管理员可以管理日志访问策略")
}

func (h *Handler) requireRetentionPolicyAdmin(c *gin.Context) bool {
	return h.requireLogAdmin(c, "只有管理员可以管理日志保留策略")
}

func (h *Handler) requireLogAdmin(c *gin.Context, message string) bool {
	_, isAdmin, err := h.userAccessibleHostIDs(c)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取管理员权限失败: "+err.Error())
		return false
	}
	if !isAdmin {
		response.ErrorCode(c, http.StatusForbidden, message)
		return false
	}
	return true
}

func (h *Handler) logAdminStatus(c *gin.Context) (bool, bool) {
	_, isAdmin, err := h.userAccessibleHostIDs(c)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取管理员权限失败: "+err.Error())
		return false, false
	}
	return isAdmin, true
}

func (payload *policyPayload) normalize() {
	payload.Name = strings.TrimSpace(payload.Name)
	payload.Environment = strings.TrimSpace(payload.Environment)
	payload.Service = strings.TrimSpace(payload.Service)
	payload.SourceMode = firstNonEmpty(strings.TrimSpace(payload.SourceMode), "host")
	payload.ReadFrom = firstNonEmpty(strings.TrimSpace(payload.ReadFrom), "latest")
	payload.Encoding = firstNonEmpty(strings.TrimSpace(payload.Encoding), "utf-8")
	payload.Stream = firstNonEmpty(strings.TrimSpace(payload.Stream), "stdout")
	if payload.MaxLineBytes <= 0 {
		payload.MaxLineBytes = 256 * 1024
	}
	if payload.RetentionDays <= 0 {
		payload.RetentionDays = payload.Retention.DefaultDays
	}
	if payload.RetentionDays <= 0 {
		payload.RetentionDays = 30
	}
	if payload.Retention.DefaultDays <= 0 {
		payload.Retention.DefaultDays = payload.RetentionDays
	}
	if !payload.Redaction.Configured {
		payload.Redaction = logagent.DefaultRedactionConfig()
	}
	if payload.WALMaxBytes <= 0 {
		payload.WALMaxBytes = 2 * 1024 * 1024 * 1024
	}
	if payload.Parser.Type == "" {
		payload.Parser.Type = "raw"
	}
	if payload.SourceMode == "kubernetes" && len(payload.Paths) == 0 {
		payload.Paths = []string{"/var/log/containers/*.log"}
	}
	for index := range payload.Paths {
		payload.Paths[index] = strings.TrimSpace(payload.Paths[index])
	}
	for index := range payload.ExcludePaths {
		payload.ExcludePaths[index] = strings.TrimSpace(payload.ExcludePaths[index])
	}
}

func applyRetentionPolicyValues(payload *policyPayload, policy logmodel.RetentionPolicy) error {
	if policy.DefaultDays <= 0 || policy.DefaultDays > 3650 {
		return fmt.Errorf("保留策略 %s 的默认保留天数无效", policy.Name)
	}
	levelDays := make(map[string]int)
	if strings.TrimSpace(policy.LevelDays) != "" {
		if err := json.Unmarshal([]byte(policy.LevelDays), &levelDays); err != nil {
			return fmt.Errorf("解析保留策略 %s 失败: %w", policy.Name, err)
		}
	}
	payload.Retention = logagent.RetentionConfig{DefaultDays: policy.DefaultDays, LevelDays: levelDays}
	payload.RetentionDays = policy.DefaultDays
	return nil
}

func applyRetentionPolicySnapshot(tx *gorm.DB, payload *policyPayload) error {
	if payload.RetentionPolicyID == 0 {
		return nil
	}
	var policy logmodel.RetentionPolicy
	if err := tx.First(&policy, payload.RetentionPolicyID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("绑定的保留策略不存在，请重新选择")
		}
		return fmt.Errorf("读取保留策略失败: %w", err)
	}
	if !policy.Enabled {
		return fmt.Errorf("保留策略 %s 已停用，请启用或更换策略", policy.Name)
	}
	return applyRetentionPolicyValues(payload, policy)
}

func validatePolicyPayload(payload policyPayload) error {
	if payload.Name == "" {
		return fmt.Errorf("策略名称不能为空")
	}
	if payload.Environment == "" {
		return fmt.Errorf("运行环境不能为空")
	}
	if payload.Service == "" {
		return fmt.Errorf("服务名称不能为空")
	}
	if payload.SourceMode != "host" && payload.SourceMode != "kubernetes" {
		return fmt.Errorf("不支持的采集模式 %s", payload.SourceMode)
	}
	if len(payload.Targets) == 0 {
		return fmt.Errorf("至少选择一个采集目标")
	}
	for _, target := range payload.Targets {
		if payload.SourceMode == "host" && (target.TargetType != "host" && target.TargetType != "host_group") {
			return fmt.Errorf("主机策略只能绑定主机或主机分组")
		}
		if payload.SourceMode == "kubernetes" && target.TargetType != "cluster" {
			return fmt.Errorf("Kubernetes 策略只能绑定集群")
		}
		if target.TargetID == 0 {
			return fmt.Errorf("采集目标 ID 无效")
		}
		if target.LabelSelector != "" {
			if _, err := labels.Parse(target.LabelSelector); err != nil {
				return fmt.Errorf("Pod 标签选择器无效: %w", err)
			}
		}
	}
	var kubernetesConfig *logagent.KubernetesSourceConfig
	format := "plain"
	if payload.SourceMode == "kubernetes" {
		format = "cri"
		kubernetesConfig = &logagent.KubernetesSourceConfig{ClusterID: uint64(payload.Targets[0].TargetID), Selectors: policyTargetsToKubernetesSelectors(payload.Targets)}
	}
	config := logagent.Config{
		Enabled: true, GatewayURL: "http://validation.local", GatewayToken: "validation", Sources: []logagent.SourceConfig{{
			ID: "validation", Format: format, Paths: payload.Paths, ExcludePaths: payload.ExcludePaths, ReadFrom: payload.ReadFrom,
			Encoding: payload.Encoding, Environment: payload.Environment, Service: payload.Service, Stream: payload.Stream,
			MaxLineBytes: payload.MaxLineBytes, Parser: payload.Parser, Multiline: payload.Multiline,
			Redaction: payload.Redaction, Retention: payload.Retention, Kubernetes: kubernetesConfig,
		}},
	}
	config.Normalize()
	return config.Validate()
}

func (payload policyPayload) toModel() logmodel.CollectionPolicy {
	policy := logmodel.CollectionPolicy{}
	payload.applyToModel(&policy)
	return policy
}

func (payload policyPayload) applyToModel(policy *logmodel.CollectionPolicy) {
	pathsJSON, _ := json.Marshal(gin.H{"include": payload.Paths, "exclude": payload.ExcludePaths})
	optionsJSON, _ := json.Marshal(gin.H{
		"readFrom": payload.ReadFrom, "encoding": payload.Encoding, "environment": payload.Environment,
		"service": payload.Service, "stream": payload.Stream, "maxLineBytes": payload.MaxLineBytes,
	})
	parserJSON, _ := json.Marshal(payload.Parser)
	multilineJSON, _ := json.Marshal(payload.Multiline)
	redactionJSON, _ := json.Marshal(payload.Redaction)
	retentionJSON, _ := json.Marshal(payload.Retention)
	walJSON, _ := json.Marshal(gin.H{"maxBytes": payload.WALMaxBytes})
	policy.Name = payload.Name
	policy.SourceMode = payload.SourceMode
	policy.Description = payload.Description
	policy.Paths = string(pathsJSON)
	policy.SourceOptions = string(optionsJSON)
	policy.ParserType = payload.Parser.Type
	policy.ParserConfig = string(parserJSON)
	policy.MultilineConfig = string(multilineJSON)
	policy.MaskConfig = string(redactionJSON)
	policy.RetentionPolicyID = payload.RetentionPolicyID
	policy.RetentionDays = payload.RetentionDays
	policy.RetentionConfig = string(retentionJSON)
	policy.WALConfig = string(walJSON)
}

func loadPolicyPayload(db *gorm.DB, policy logmodel.CollectionPolicy) (policyPayload, error) {
	payload := policyPayload{
		Name: policy.Name, SourceMode: policy.SourceMode, Description: policy.Description,
		RetentionPolicyID: policy.RetentionPolicyID, RetentionDays: policy.RetentionDays,
	}
	var paths struct {
		Include []string `json:"include"`
		Exclude []string `json:"exclude"`
	}
	_ = json.Unmarshal([]byte(policy.Paths), &paths)
	payload.Paths = paths.Include
	payload.ExcludePaths = paths.Exclude
	var options struct {
		ReadFrom     string `json:"readFrom"`
		Encoding     string `json:"encoding"`
		Environment  string `json:"environment"`
		Service      string `json:"service"`
		Stream       string `json:"stream"`
		MaxLineBytes int    `json:"maxLineBytes"`
	}
	_ = json.Unmarshal([]byte(policy.SourceOptions), &options)
	payload.ReadFrom, payload.Encoding, payload.Environment = options.ReadFrom, options.Encoding, options.Environment
	payload.Service, payload.Stream, payload.MaxLineBytes = options.Service, options.Stream, options.MaxLineBytes
	_ = json.Unmarshal([]byte(policy.ParserConfig), &payload.Parser)
	_ = json.Unmarshal([]byte(policy.MultilineConfig), &payload.Multiline)
	_ = json.Unmarshal([]byte(policy.MaskConfig), &payload.Redaction)
	_ = json.Unmarshal([]byte(policy.RetentionConfig), &payload.Retention)
	var wal struct {
		MaxBytes int64 `json:"maxBytes"`
	}
	_ = json.Unmarshal([]byte(policy.WALConfig), &wal)
	payload.WALMaxBytes = wal.MaxBytes
	var targets []logmodel.PolicyTarget
	if err := db.Where("policy_id = ?", policy.ID).Order("id ASC").Find(&targets).Error; err != nil {
		return payload, err
	}
	for _, target := range targets {
		var include, exclude []string
		_ = json.Unmarshal([]byte(target.ContainerInclude), &include)
		_ = json.Unmarshal([]byte(target.ContainerExclude), &exclude)
		payload.Targets = append(payload.Targets, policyTargetInput{
			TargetType: target.TargetType, TargetID: target.TargetID, Namespace: target.Namespace,
			WorkloadKind: target.WorkloadKind, WorkloadName: target.WorkloadName, LabelSelector: target.LabelSelector,
			ContainerInclude: include, ContainerExclude: exclude,
		})
	}
	payload.normalize()
	return payload, nil
}

func loadPolicyViewPayload(db *gorm.DB, policy logmodel.CollectionPolicy) (policyPayload, bool, error) {
	draft, found, err := loadPendingPolicyDraft(db, policy.ID)
	if err != nil {
		return policyPayload{}, false, err
	}
	if found {
		return draft, true, nil
	}
	payload, loadErr := loadPolicyPayload(db, policy)
	return payload, false, loadErr
}

func loadPendingPolicyDraft(db *gorm.DB, policyID uint) (policyPayload, bool, error) {
	var draft logmodel.PolicyRevision
	result := db.Where("policy_id = ? AND version = ?", policyID, 0).Limit(1).Find(&draft)
	if result.Error != nil {
		return policyPayload{}, false, result.Error
	}
	if result.RowsAffected == 0 {
		return policyPayload{}, false, nil
	}
	var payload policyPayload
	if unmarshalErr := json.Unmarshal([]byte(draft.Content), &payload); unmarshalErr != nil {
		return policyPayload{}, false, fmt.Errorf("解析待发布策略失败: %w", unmarshalErr)
	}
	payload.normalize()
	return payload, true, nil
}

func savePendingPolicyDraft(tx *gorm.DB, policyID uint, payload policyPayload, userID uint) error {
	content, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	checksumRaw := sha256.Sum256(content)
	draft := logmodel.PolicyRevision{
		PolicyID: policyID, Version: 0, Content: string(content), Checksum: hex.EncodeToString(checksumRaw[:]),
		ChangeSummary: "待发布草稿", CreatedBy: userID,
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "policy_id"}, {Name: "version"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"content": draft.Content, "checksum": draft.Checksum, "change_summary": draft.ChangeSummary,
			"created_by": userID, "created_at": time.Now(),
		}),
	}).Create(&draft).Error
}

func applyPendingPolicyDraft(tx *gorm.DB, policy *logmodel.CollectionPolicy) error {
	var draft logmodel.PolicyRevision
	err := tx.Where("policy_id = ? AND version = ?", policy.ID, 0).First(&draft).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	var payload policyPayload
	if err := json.Unmarshal([]byte(draft.Content), &payload); err != nil {
		return fmt.Errorf("解析待发布策略失败: %w", err)
	}
	payload.normalize()
	if err := validatePolicyPayload(payload); err != nil {
		return err
	}
	payload.applyToModel(policy)
	if err := tx.Save(policy).Error; err != nil {
		return err
	}
	if err := replacePolicyTargets(tx, policy.ID, payload.Targets); err != nil {
		return err
	}
	return tx.Delete(&draft).Error
}

func replacePolicyTargets(tx *gorm.DB, policyID uint, targets []policyTargetInput) error {
	if err := tx.Where("policy_id = ?", policyID).Delete(&logmodel.PolicyTarget{}).Error; err != nil {
		return err
	}
	seen := make(map[string]struct{})
	for _, input := range targets {
		key := fmt.Sprintf("%s:%d:%s:%s:%s", input.TargetType, input.TargetID, input.Namespace, input.WorkloadKind, input.WorkloadName)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		include, _ := json.Marshal(input.ContainerInclude)
		exclude, _ := json.Marshal(input.ContainerExclude)
		target := logmodel.PolicyTarget{
			PolicyID: policyID, TargetType: input.TargetType, TargetID: input.TargetID,
			Namespace: input.Namespace, WorkloadKind: input.WorkloadKind, WorkloadName: input.WorkloadName,
			LabelSelector: input.LabelSelector, ContainerInclude: string(include), ContainerExclude: string(exclude),
		}
		if err := tx.Create(&target).Error; err != nil {
			return err
		}
	}
	return nil
}

func resolvePolicyHosts(db *gorm.DB, policyID uint) ([]assetbiz.Host, error) {
	var targets []logmodel.PolicyTarget
	if err := db.Where("policy_id = ?", policyID).Find(&targets).Error; err != nil {
		return nil, err
	}
	inputs := make([]policyTargetInput, 0, len(targets))
	for _, target := range targets {
		inputs = append(inputs, policyTargetInput{TargetType: target.TargetType, TargetID: target.TargetID})
	}
	return resolvePolicyHostsFromInputs(db, inputs)
}

func resolvePolicyHostsFromInputs(db *gorm.DB, targets []policyTargetInput) ([]assetbiz.Host, error) {
	hostIDs := make(map[uint]struct{})
	groupIDs := make([]uint, 0)
	for _, target := range targets {
		switch target.TargetType {
		case "host":
			hostIDs[target.TargetID] = struct{}{}
		case "host_group":
			groupIDs = append(groupIDs, target.TargetID)
		}
	}
	if len(groupIDs) > 0 {
		var grouped []uint
		if err := db.Model(&assetbiz.Host{}).Where("group_id IN ?", groupIDs).Pluck("id", &grouped).Error; err != nil {
			return nil, err
		}
		for _, id := range grouped {
			hostIDs[id] = struct{}{}
		}
	}
	ids := make([]uint, 0, len(hostIDs))
	for id := range hostIDs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(left, right int) bool { return ids[left] < ids[right] })
	if len(ids) == 0 {
		return []assetbiz.Host{}, nil
	}
	var hosts []assetbiz.Host
	if err := db.Where("id IN ?", ids).Order("name ASC").Find(&hosts).Error; err != nil {
		return nil, err
	}
	return hosts, nil
}

func hostsToTargetView(hosts []assetbiz.Host) []policyTargetHost {
	result := make([]policyTargetHost, 0, len(hosts))
	for _, host := range hosts {
		result = append(result, policyTargetHost{
			ID: host.ID, Name: host.Name, IP: host.IP, GroupID: host.GroupID,
			AgentID: host.AgentID, AgentVersion: host.AgentVersion, AgentStatus: host.AgentStatus,
		})
	}
	return result
}

func resolvePolicyClusters(db *gorm.DB, policyID uint) ([]k8smodel.Cluster, error) {
	var targets []logmodel.PolicyTarget
	if err := db.Where("policy_id = ? AND target_type = ?", policyID, "cluster").Find(&targets).Error; err != nil {
		return nil, err
	}
	inputs := make([]policyTargetInput, 0, len(targets))
	for _, target := range targets {
		inputs = append(inputs, policyTargetInput{TargetType: target.TargetType, TargetID: target.TargetID})
	}
	return resolvePolicyClustersFromInputs(db, inputs)
}

func resolvePolicyClustersFromInputs(db *gorm.DB, targets []policyTargetInput) ([]k8smodel.Cluster, error) {
	clusterSet := make(map[uint]struct{})
	for _, target := range targets {
		if target.TargetType == "cluster" && target.TargetID > 0 {
			clusterSet[target.TargetID] = struct{}{}
		}
	}
	clusterIDs := make([]uint, 0, len(clusterSet))
	for clusterID := range clusterSet {
		clusterIDs = append(clusterIDs, clusterID)
	}
	sort.Slice(clusterIDs, func(left, right int) bool { return clusterIDs[left] < clusterIDs[right] })
	if len(clusterIDs) == 0 {
		return []k8smodel.Cluster{}, nil
	}
	var clusters []k8smodel.Cluster
	if err := db.Select("id", "name", "alias", "version", "node_count", "status").Where("id IN ?", clusterIDs).Order("name ASC").Find(&clusters).Error; err != nil {
		return nil, err
	}
	return clusters, nil
}

func clustersToTargetView(clusters []k8smodel.Cluster) []policyTargetCluster {
	result := make([]policyTargetCluster, 0, len(clusters))
	for _, cluster := range clusters {
		result = append(result, policyTargetCluster{
			ID: cluster.ID, Name: cluster.Name, Alias: cluster.Alias, Version: cluster.Version,
			NodeCount: cluster.NodeCount, Status: cluster.Status,
		})
	}
	return result
}

func policyTargetsToKubernetesSelectors(targets []policyTargetInput) []logagent.KubernetesSelector {
	selectors := make([]logagent.KubernetesSelector, 0, len(targets))
	for _, target := range targets {
		selectors = append(selectors, logagent.KubernetesSelector{
			Namespace: target.Namespace, WorkloadKind: target.WorkloadKind, WorkloadName: target.WorkloadName,
			LabelSelector: target.LabelSelector, ContainerInclude: target.ContainerInclude, ContainerExclude: target.ContainerExclude,
		})
	}
	return selectors
}
