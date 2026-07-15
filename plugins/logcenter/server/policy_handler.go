package server

import (
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
	ID              uint                  `json:"id"`
	Status          string                `json:"status"`
	Version         uint64                `json:"version"`
	CreatedBy       uint                  `json:"createdBy"`
	UpdatedBy       uint                  `json:"updatedBy"`
	CreatedAt       time.Time             `json:"createdAt"`
	UpdatedAt       time.Time             `json:"updatedAt"`
	Payload         policyPayload         `json:"payload"`
	TargetCount     int                   `json:"targetCount"`
<<<<<<< HEAD
	InstanceTotal   int64                 `json:"instanceTotal"`
	InstanceApplied int64                 `json:"instanceApplied"`
=======
	TargetExpected  int                   `json:"targetExpected"`
	InstanceTotal   int64                 `json:"instanceTotal"`
	InstanceOnline  int64                 `json:"instanceOnline"`
	InstanceApplied int64                 `json:"instanceApplied"`
	InstancePending int64                 `json:"instancePending"`
>>>>>>> feat: update log
	ErrorInstances  int64                 `json:"errorInstances"`
	TargetHosts     []policyTargetHost    `json:"targetHosts"`
	TargetClusters  []policyTargetCluster `json:"targetClusters"`
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

func (h *Handler) ListCollectionPolicies(c *gin.Context) {
	query := h.db.WithContext(c.Request.Context()).Model(&logmodel.CollectionPolicy{}).Order("updated_at DESC")
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
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
	if err := validatePolicyPayload(payload); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	payload.applyToModel(&policy)
	policy.Status = "draft"
	policy.UpdatedBy = rbacsvc.GetUserID(c)
	if err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&policy).Error; err != nil {
			return err
		}
		return replacePolicyTargets(tx, policy.ID, payload.Targets)
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
	policy, ok := h.findCollectionPolicy(c)
	if !ok {
		return
	}
	if policy.Version > 0 || policy.Status != "draft" {
		response.ErrorCode(c, http.StatusConflict, "只有从未发布的草稿策略可以删除")
		return
	}
	var assignmentCount int64
	h.db.Model(&logmodel.CollectorAssignment{}).Where("policy_id = ?", policy.ID).Count(&assignmentCount)
	if assignmentCount > 0 {
		response.ErrorCode(c, http.StatusConflict, "策略仍有关联实例，无法删除")
		return
	}
	if err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("policy_id = ?", policy.ID).Delete(&logmodel.PolicyTarget{}).Error; err != nil {
			return err
		}
		return tx.Delete(&policy).Error
	}); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除采集策略失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"id": policy.ID})
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
		result, err := publishPolicy(tx, policy, rbacsvc.GetUserID(c), strings.TrimSpace(req.ChangeSummary))
		published = result
		return err
	})
	if err != nil {
		status := http.StatusInternalServerError
		if err == gorm.ErrRecordNotFound {
			status = http.StatusNotFound
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
	policyID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var revisions []logmodel.PolicyRevision
	if err := h.db.WithContext(c.Request.Context()).Where("policy_id = ?", policyID).Order("version DESC").Find(&revisions).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取发布记录失败: "+err.Error())
		return
	}
	for index := range revisions {
		revisions[index].Content = ""
	}
	response.Success(c, revisions)
}

func (h *Handler) ListAllPolicyRevisions(c *gin.Context) {
	var revisions []logmodel.PolicyRevision
	query := h.db.WithContext(c.Request.Context()).Order("created_at DESC")
	if policyID := parseUint(c.Query("policyId")); policyID > 0 {
		query = query.Where("policy_id = ?", policyID)
	}
	if err := query.Limit(500).Find(&revisions).Error; err != nil {
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
	response.Success(c, result)
}

func (h *Handler) PreviewPolicyTargets(c *gin.Context) {
	policy, ok := h.findCollectionPolicy(c)
	if !ok {
		return
	}
	if policy.SourceMode == "kubernetes" {
		clusters, err := resolvePolicyClusters(h.db.WithContext(c.Request.Context()), policy.ID)
		if err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "预览目标集群失败: "+err.Error())
			return
		}
		response.Success(c, clustersToTargetView(clusters))
		return
	}
	hosts, err := resolvePolicyHosts(h.db.WithContext(c.Request.Context()), policy.ID)
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

func (h *Handler) ListCollectorInstances(c *gin.Context) {
	var instances []logmodel.CollectorInstance
	if err := h.db.WithContext(c.Request.Context()).Order("updated_at DESC").Find(&instances).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取采集实例失败: "+err.Error())
		return
	}
	now := time.Now()
	result := make([]gin.H, 0, len(instances))
	for _, instance := range instances {
		status := instance.Status
		if instance.LastHeartbeatAt == nil || now.Sub(*instance.LastHeartbeatAt) > 90*time.Second {
			status = "offline"
		}
		var assignments []logmodel.CollectorAssignment
		h.db.Where("instance_id = ?", instance.InstanceID).Order("policy_id ASC").Find(&assignments)
		result = append(result, gin.H{"instance": instance, "status": status, "assignments": assignments})
	}
	response.Success(c, result)
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
	if err := validatePolicyPayload(payload); err != nil {
		return policy, err
	}
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
	payload, err := loadPolicyPayload(h.db.WithContext(c.Request.Context()), policy)
	if err != nil {
		return policyView{}, err
	}
<<<<<<< HEAD
	var targetCount int
=======
	var targetCount, targetExpected int
>>>>>>> feat: update log
	var targetHosts []policyTargetHost
	var targetClusters []policyTargetCluster
	if policy.SourceMode == "kubernetes" {
		clusters, err := resolvePolicyClusters(h.db.WithContext(c.Request.Context()), policy.ID)
		if err != nil {
			return policyView{}, err
		}
		targetClusters = clustersToTargetView(clusters)
		targetCount = len(targetClusters)
<<<<<<< HEAD
=======
		for _, cluster := range targetClusters {
			targetExpected += cluster.NodeCount
		}
>>>>>>> feat: update log
	} else {
		hosts, err := resolvePolicyHosts(h.db.WithContext(c.Request.Context()), policy.ID)
		if err != nil {
			return policyView{}, err
		}
		targetHosts = hostsToTargetView(hosts)
		targetCount = len(targetHosts)
<<<<<<< HEAD
	}
	var total, applied, failed int64
	h.db.Model(&logmodel.CollectorAssignment{}).Where("policy_id = ?", policy.ID).Count(&total)
	h.db.Model(&logmodel.CollectorAssignment{}).Where("policy_id = ? AND apply_status = ?", policy.ID, "applied").Count(&applied)
	h.db.Model(&logmodel.CollectorAssignment{}).Where("policy_id = ? AND apply_status = ?", policy.ID, "failed").Count(&failed)
	return policyView{
		ID: policy.ID, Status: policy.Status, Version: policy.Version, CreatedBy: policy.CreatedBy, UpdatedBy: policy.UpdatedBy,
		CreatedAt: policy.CreatedAt, UpdatedAt: policy.UpdatedAt, Payload: payload,
		TargetCount: targetCount, InstanceTotal: total, InstanceApplied: applied, ErrorInstances: failed,
=======
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
		ID: policy.ID, Status: policy.Status, Version: policy.Version, CreatedBy: policy.CreatedBy, UpdatedBy: policy.UpdatedBy,
		CreatedAt: policy.CreatedAt, UpdatedAt: policy.UpdatedAt, Payload: payload,
		TargetCount: targetCount, TargetExpected: targetExpected,
		InstanceTotal: total, InstanceOnline: online, InstanceApplied: applied, InstancePending: pending, ErrorInstances: failed,
>>>>>>> feat: update log
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

func (h *Handler) requirePolicyAdmin(c *gin.Context) bool {
	_, isAdmin, err := h.userAccessibleHostIDs(c)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取管理员权限失败: "+err.Error())
		return false
	}
	if !isAdmin {
		response.ErrorCode(c, http.StatusForbidden, "只有管理员可以管理日志采集策略")
		return false
	}
	return true
}

func (payload *policyPayload) normalize() {
	payload.Name = strings.TrimSpace(payload.Name)
<<<<<<< HEAD
=======
	payload.Environment = strings.TrimSpace(payload.Environment)
	payload.Service = strings.TrimSpace(payload.Service)
>>>>>>> feat: update log
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

func validatePolicyPayload(payload policyPayload) error {
	if payload.Name == "" {
		return fmt.Errorf("策略名称不能为空")
	}
<<<<<<< HEAD
=======
	if payload.Environment == "" {
		return fmt.Errorf("运行环境不能为空")
	}
	if payload.Service == "" {
		return fmt.Errorf("服务名称不能为空")
	}
>>>>>>> feat: update log
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
	var clusterIDs []uint
	if err := db.Model(&logmodel.PolicyTarget{}).Where("policy_id = ? AND target_type = ?", policyID, "cluster").Distinct("target_id").Pluck("target_id", &clusterIDs).Error; err != nil {
		return nil, err
	}
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
