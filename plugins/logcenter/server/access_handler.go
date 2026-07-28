package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	rbacsvc "github.com/ydcloud-dy/opshub/internal/service/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
	logsvc "github.com/ydcloud-dy/opshub/plugins/logcenter/service"
	"gorm.io/gorm"
)

const (
	logAccessScopeAll              = "all"
	logAccessScopeCollectionPolicy = "collection_policy"
)

type logAccessDecision struct {
	IsAdmin                 bool
	AllowedPolicyIDs        []uint64
	AllowedHostIDs          []uint64
	AllowedKubernetesScopes map[uint64][]string
	DeniedFields            []string
	MaskFields              []string
}

type logAccessFailure struct {
	Status  int
	Message string
}

type accessPolicyPayload struct {
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	SubjectType         string   `json:"subjectType"`
	SubjectID           uint     `json:"subjectId"`
	StorageID           uint     `json:"storageId"`
	LibraryItemPattern  string   `json:"libraryItemPattern"`
	ScopeMode           string   `json:"scopeMode"`
	CollectionPolicyIDs []uint   `json:"collectionPolicyIds"`
	AllowedActions      []string `json:"allowedActions"`
	DeniedFields        []string `json:"deniedFields"`
	MaskFields          []string `json:"maskFields"`
	Enabled             bool     `json:"enabled"`
}

type accessPolicyView struct {
	ID        uint                `json:"id"`
	CreatedBy uint                `json:"createdBy"`
	UpdatedBy uint                `json:"updatedBy"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
	Payload   accessPolicyPayload `json:"payload"`
}

type accessCollectionPolicyOption struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	SourceMode  string `json:"sourceMode"`
	Status      string `json:"status"`
	Environment string `json:"environment"`
}

type internalLogAccessCapabilities struct {
	IsAdmin   bool `json:"isAdmin"`
	CanQuery  bool `json:"canQuery"`
	CanTail   bool `json:"canTail"`
	CanExport bool `json:"canExport"`
}

func (h *Handler) authorizeInternalAction(c *gin.Context, action string, storageID uint) (logAccessDecision, bool) {
	decision, failure := h.resolveInternalAction(c, action, storageID)
	if failure != nil {
		response.ErrorCode(c, failure.Status, failure.Message)
		return logAccessDecision{}, false
	}
	return decision, true
}

func (h *Handler) resolveInternalAction(c *gin.Context, action string, storageID uint) (logAccessDecision, *logAccessFailure) {
	allowedHostIDs, isAdmin, err := h.userAccessibleHostIDs(c)
	if err != nil {
		return logAccessDecision{}, &logAccessFailure{Status: http.StatusInternalServerError, Message: "读取主机日志权限失败: " + err.Error()}
	}
	decision := logAccessDecision{IsAdmin: isAdmin, AllowedHostIDs: allowedHostIDs}
	if isAdmin {
		return decision, nil
	}
	kubernetesScopes, err := h.userAccessibleKubernetesScopes(c)
	if err != nil {
		return logAccessDecision{}, &logAccessFailure{Status: http.StatusInternalServerError, Message: "读取 Kubernetes 日志权限失败: " + err.Error()}
	}
	decision.AllowedKubernetesScopes = kubernetesScopes

	effectiveStorageID, err := h.effectiveStorageID(c, storageID)
	if err != nil {
		return logAccessDecision{}, &logAccessFailure{Status: http.StatusInternalServerError, Message: "读取日志存储权限失败: " + err.Error()}
	}
	policies, err := h.userLogAccessPolicies(c, effectiveStorageID)
	if err != nil {
		return logAccessDecision{}, &logAccessFailure{Status: http.StatusInternalServerError, Message: "读取日志访问策略失败: " + err.Error()}
	}
	evaluated, actionAllowed := evaluateLogAccessPolicies(policies, action)
	if !actionAllowed {
		return logAccessDecision{}, &logAccessFailure{Status: http.StatusForbidden, Message: fmt.Sprintf("当前账号未配置允许 %s 的有效日志访问策略", action)}
	}
	decision.AllowedPolicyIDs = evaluated.AllowedPolicyIDs
	decision.DeniedFields = evaluated.DeniedFields
	decision.MaskFields = evaluated.MaskFields
	return decision, nil
}

func evaluateLogAccessPolicies(policies []logmodel.AccessPolicy, action string) (logAccessDecision, bool) {
	decision := logAccessDecision{}
	actionAllowed := false
	unrestricted := false
	policyIDs := make([]uint64, 0)
	for _, policy := range policies {
		decision.DeniedFields = append(decision.DeniedFields, decodeStringList(policy.DeniedFields)...)
		decision.MaskFields = append(decision.MaskFields, decodeStringList(policy.MaskFields)...)
		if !stringListContains(decodeStringList(policy.AllowedActions), action) {
			continue
		}
		actionAllowed = true
		if normalizeStoredAccessScopeMode(policy.ScopeMode) == logAccessScopeAll {
			unrestricted = true
			continue
		}
		for _, scope := range policy.Scopes {
			if scope.CollectionPolicyID > 0 {
				policyIDs = append(policyIDs, uint64(scope.CollectionPolicyID))
			}
		}
	}
	decision.DeniedFields = uniqueNormalizedStrings(decision.DeniedFields)
	decision.MaskFields = uniqueNormalizedStrings(decision.MaskFields)
	if !unrestricted {
		decision.AllowedPolicyIDs = uniqueUint64s(policyIDs)
	}
	return decision, actionAllowed
}

func (h *Handler) GetInternalAccessCapabilities(c *gin.Context) {
	_, isAdmin, err := h.userAccessibleHostIDs(c)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取日志访问能力失败: "+err.Error())
		return
	}
	var policies []logmodel.AccessPolicy
	if !isAdmin {
		policies, err = h.userAllLogAccessPolicies(c)
		if err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "读取日志访问策略失败: "+err.Error())
			return
		}
	}
	response.Success(c, logAccessCapabilitiesForPolicies(policies, isAdmin))
}

func logAccessCapabilitiesForPolicies(policies []logmodel.AccessPolicy, isAdmin bool) internalLogAccessCapabilities {
	capabilities := internalLogAccessCapabilities{IsAdmin: isAdmin}
	if isAdmin {
		capabilities.CanQuery = true
		capabilities.CanTail = true
		capabilities.CanExport = true
		return capabilities
	}
	_, capabilities.CanQuery = evaluateLogAccessPolicies(policies, "query")
	_, capabilities.CanTail = evaluateLogAccessPolicies(policies, "tail")
	_, capabilities.CanExport = evaluateLogAccessPolicies(policies, "export")
	return capabilities
}

func (h *Handler) applyInternalPermissions(c *gin.Context, req *logsvc.InternalQueryRequest, action string) bool {
	decision, ok := h.authorizeInternalAction(c, action, req.StorageID)
	if !ok {
		return false
	}
	if decision.IsAdmin {
		return true
	}
	req.AllowedHostIDs = decision.AllowedHostIDs
	req.AllowedPolicyIDs = decision.AllowedPolicyIDs
	req.AllowedKubernetesScopes = decision.AllowedKubernetesScopes
	req.DeniedFields = decision.DeniedFields
	req.MaskFields = decision.MaskFields
	if !validateQueryFieldAccess(c, *req, decision.DeniedFields) {
		return false
	}

	if len(req.Scope.HostIDs) > 0 {
		filtered := filterAllowedIDs(req.Scope.HostIDs, uint64SliceSet(decision.AllowedHostIDs))
		if len(filtered) == 0 {
			req.DenyAll = true
		}
		req.Scope.HostIDs = filtered
	}
	if len(req.Scope.ClusterIDs) > 0 {
		allowedClusters := make(map[uint64]struct{}, len(decision.AllowedKubernetesScopes))
		for clusterID := range decision.AllowedKubernetesScopes {
			allowedClusters[clusterID] = struct{}{}
		}
		filtered := filterAllowedIDs(req.Scope.ClusterIDs, allowedClusters)
		if len(filtered) == 0 {
			req.DenyAll = true
		}
		req.Scope.ClusterIDs = filtered
	}
	return true
}

func (h *Handler) applyInternalContextPermissions(c *gin.Context, req *logsvc.InternalContextRequest) bool {
	decision, ok := h.authorizeInternalAction(c, "query", req.StorageID)
	if !ok {
		return false
	}
	if decision.IsAdmin {
		return true
	}
	req.AllowedHostIDs = decision.AllowedHostIDs
	req.AllowedPolicyIDs = decision.AllowedPolicyIDs
	req.AllowedKubernetesScopes = decision.AllowedKubernetesScopes
	req.DeniedFields = decision.DeniedFields
	req.MaskFields = decision.MaskFields
	return true
}

func validateQueryFieldAccess(c *gin.Context, req logsvc.InternalQueryRequest, deniedFields []string) bool {
	if message := queryFieldAccessError(req, deniedFields); message != "" {
		response.ErrorCode(c, http.StatusForbidden, message)
		return false
	}
	return true
}

func queryFieldAccessError(req logsvc.InternalQueryRequest, deniedFields []string) string {
	if strings.TrimSpace(req.Query) != "" && strings.TrimSpace(req.Query) != "*" &&
		(logsvc.FieldAccessDenied("body", deniedFields) || logsvc.FieldAccessDenied("message", deniedFields)) {
		return "当前日志访问策略不允许检索日志正文"
	}
	for _, filter := range req.Filters {
		if logsvc.FieldAccessDenied(filter.Field, deniedFields) {
			return "当前日志访问策略不允许筛选字段: " + filter.Field
		}
	}
	return ""
}

func (h *Handler) userAccessibleKubernetesScopes(c *gin.Context) (map[uint64][]string, error) {
	userID := uint64(rbacsvc.GetUserID(c))
	result := make(map[uint64][]string)
	if userID == 0 {
		return result, nil
	}
	fullAccess := make(map[uint64]struct{})
	var ownedClusterIDs []uint64
	if h.db.Migrator().HasTable("k8s_clusters") {
		if err := h.db.WithContext(c.Request.Context()).Table("k8s_clusters").Where("created_by = ?", userID).Pluck("id", &ownedClusterIDs).Error; err != nil {
			return nil, err
		}
	}
	for _, clusterID := range ownedClusterIDs {
		fullAccess[clusterID] = struct{}{}
	}
	if h.db.Migrator().HasTable("k8s_user_role_bindings") {
		var bindings []struct {
			ClusterID     uint64
			RoleNamespace string
		}
		if err := h.db.WithContext(c.Request.Context()).Table("k8s_user_role_bindings").
			Select("cluster_id, role_namespace").Where("user_id = ?", userID).Scan(&bindings).Error; err != nil {
			return nil, err
		}
		for _, binding := range bindings {
			if strings.TrimSpace(binding.RoleNamespace) == "" {
				fullAccess[binding.ClusterID] = struct{}{}
				continue
			}
			result[binding.ClusterID] = append(result[binding.ClusterID], strings.TrimSpace(binding.RoleNamespace))
		}
	}
	if h.db.Migrator().HasTable("k8s_user_kube_configs") {
		var configs []struct {
			ClusterID uint64
			Namespace string
		}
		if err := h.db.WithContext(c.Request.Context()).Table("k8s_user_kube_configs").
			Select("cluster_id, namespace").Where("user_id = ? AND is_active = ?", userID, true).Scan(&configs).Error; err != nil {
			return nil, err
		}
		for _, config := range configs {
			if strings.TrimSpace(config.Namespace) == "" {
				fullAccess[config.ClusterID] = struct{}{}
				continue
			}
			result[config.ClusterID] = append(result[config.ClusterID], strings.TrimSpace(config.Namespace))
		}
	}
	for clusterID := range fullAccess {
		result[clusterID] = nil
	}
	for clusterID, namespaces := range result {
		if _, full := fullAccess[clusterID]; full {
			continue
		}
		result[clusterID] = uniqueNormalizedStrings(namespaces)
	}
	return result, nil
}

func (h *Handler) effectiveStorageID(c *gin.Context, storageID uint) (uint, error) {
	if storageID > 0 {
		return storageID, nil
	}
	var id uint
	err := h.db.WithContext(c.Request.Context()).Model(&logmodel.StorageCluster{}).
		Where("enabled = ?", true).Order("is_primary DESC, id ASC").Limit(1).Pluck("id", &id).Error
	return id, err
}

func (h *Handler) userLogAccessPolicies(c *gin.Context, storageID uint) ([]logmodel.AccessPolicy, error) {
	query, err := h.userLogAccessPolicyQuery(c)
	if err != nil {
		return nil, err
	}
	var policies []logmodel.AccessPolicy
	return policies, query.Where("data_source_id IN ?", []uint{0, storageID}).Preload("Scopes").Order("id ASC").Find(&policies).Error
}

func (h *Handler) userAllLogAccessPolicies(c *gin.Context) ([]logmodel.AccessPolicy, error) {
	query, err := h.userLogAccessPolicyQuery(c)
	if err != nil {
		return nil, err
	}
	var policies []logmodel.AccessPolicy
	return policies, query.Preload("Scopes").Order("id ASC").Find(&policies).Error
}

func (h *Handler) userLogAccessPolicyQuery(c *gin.Context) (*gorm.DB, error) {
	userID := rbacsvc.GetUserID(c)
	var roleIDs []uint
	if err := h.db.WithContext(c.Request.Context()).Table("sys_user_role").Where("user_id = ?", userID).Pluck("role_id", &roleIDs).Error; err != nil {
		return nil, err
	}
	query := h.db.WithContext(c.Request.Context()).Model(&logmodel.AccessPolicy{}).Where("enabled = ?", true)
	if len(roleIDs) == 0 {
		query = query.Where("subject_type = ? AND subject_id = ?", "user", userID)
	} else {
		query = query.Where("(subject_type = ? AND subject_id = ?) OR (subject_type = ? AND subject_id IN ?)", "user", userID, "role", roleIDs)
	}
	return query, nil
}

func (h *Handler) ListAccessPolicies(c *gin.Context) {
	if !h.requireAccessPolicyAdmin(c) {
		return
	}
	var policies []logmodel.AccessPolicy
	if err := h.db.WithContext(c.Request.Context()).Preload("Scopes").Order("updated_at DESC").Find(&policies).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取日志访问策略失败: "+err.Error())
		return
	}
	views := make([]accessPolicyView, 0, len(policies))
	for _, policy := range policies {
		views = append(views, accessPolicyToView(policy))
	}
	response.Success(c, views)
}

func (h *Handler) CreateAccessPolicy(c *gin.Context) {
	if !h.requireAccessPolicyAdmin(c) {
		return
	}
	var payload accessPolicyPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	policy, err := accessPolicyFromPayload(payload)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	policy.CreatedBy = rbacsvc.GetUserID(c)
	policy.UpdatedBy = policy.CreatedBy
	if err := h.saveAccessPolicy(c, &policy, false); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建日志访问策略失败: "+err.Error())
		return
	}
	response.Success(c, accessPolicyToView(policy))
}

func (h *Handler) UpdateAccessPolicy(c *gin.Context) {
	if !h.requireAccessPolicyAdmin(c) {
		return
	}
	var existing logmodel.AccessPolicy
	if err := h.db.WithContext(c.Request.Context()).First(&existing, parseUint(c.Param("id"))).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "日志访问策略不存在")
		return
	}
	var payload accessPolicyPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	policy, err := accessPolicyFromPayload(payload)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	policy.ID = existing.ID
	policy.CreatedAt = existing.CreatedAt
	policy.CreatedBy = existing.CreatedBy
	policy.UpdatedBy = rbacsvc.GetUserID(c)
	if err := h.saveAccessPolicy(c, &policy, true); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新日志访问策略失败: "+err.Error())
		return
	}
	response.Success(c, accessPolicyToView(policy))
}

func (h *Handler) DeleteAccessPolicy(c *gin.Context) {
	if !h.requireAccessPolicyAdmin(c) {
		return
	}
	id := parseUint(c.Param("id"))
	if id == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "删除日志访问策略失败")
		return
	}
	if err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("access_policy_id = ?", id).Delete(&logmodel.AccessPolicyScope{}).Error; err != nil {
			return err
		}
		result := tx.Delete(&logmodel.AccessPolicy{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	}); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "删除日志访问策略失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) GetAccessPolicyOptions(c *gin.Context) {
	if !h.requireAccessPolicyAdmin(c) {
		return
	}
	var users []struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
		RealName string `json:"realName"`
	}
	var roles []struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
		Code string `json:"code"`
	}
	var collectionPolicies []logmodel.CollectionPolicy
	if err := h.db.WithContext(c.Request.Context()).Table("sys_user").Select("id, username, real_name").Where("deleted_at IS NULL").Order("username ASC").Scan(&users).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取用户选项失败: "+err.Error())
		return
	}
	if err := h.db.WithContext(c.Request.Context()).Table("sys_role").Select("id, name, code").Where("deleted_at IS NULL").Order("sort ASC, id ASC").Scan(&roles).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取角色选项失败: "+err.Error())
		return
	}
	if err := h.db.WithContext(c.Request.Context()).Order("name ASC, id ASC").Find(&collectionPolicies).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取采集策略选项失败: "+err.Error())
		return
	}
	policyOptions := make([]accessCollectionPolicyOption, 0, len(collectionPolicies))
	for _, policy := range collectionPolicies {
		var sourceOptions struct {
			Environment string `json:"environment"`
		}
		_ = json.Unmarshal([]byte(policy.SourceOptions), &sourceOptions)
		policyOptions = append(policyOptions, accessCollectionPolicyOption{
			ID: policy.ID, Name: policy.Name, SourceMode: policy.SourceMode,
			Status: policy.Status, Environment: strings.TrimSpace(sourceOptions.Environment),
		})
	}
	response.Success(c, gin.H{"users": users, "roles": roles, "collectionPolicies": policyOptions})
}

func accessPolicyFromPayload(payload accessPolicyPayload) (logmodel.AccessPolicy, error) {
	payload.Name = strings.TrimSpace(payload.Name)
	payload.SubjectType = strings.ToLower(strings.TrimSpace(payload.SubjectType))
	payload.ScopeMode = strings.ToLower(strings.TrimSpace(payload.ScopeMode))
	if payload.ScopeMode == "" {
		payload.ScopeMode = logAccessScopeCollectionPolicy
	}
	if payload.Name == "" || payload.SubjectID == 0 {
		return logmodel.AccessPolicy{}, fmt.Errorf("策略名称和授权对象不能为空")
	}
	if payload.SubjectType != "user" && payload.SubjectType != "role" {
		return logmodel.AccessPolicy{}, fmt.Errorf("授权对象类型仅支持用户或角色")
	}
	if payload.ScopeMode != logAccessScopeAll && payload.ScopeMode != logAccessScopeCollectionPolicy {
		return logmodel.AccessPolicy{}, fmt.Errorf("日志范围仅支持全部采集策略或指定采集策略")
	}
	collectionPolicyIDs := uniqueUintValues(payload.CollectionPolicyIDs)
	if payload.ScopeMode == logAccessScopeCollectionPolicy && len(collectionPolicyIDs) == 0 {
		return logmodel.AccessPolicy{}, fmt.Errorf("指定采集策略时至少选择一个采集策略")
	}
	if payload.ScopeMode == logAccessScopeAll {
		collectionPolicyIDs = nil
	}
	actions := uniqueNormalizedStrings(payload.AllowedActions)
	if len(actions) == 0 {
		return logmodel.AccessPolicy{}, fmt.Errorf("至少允许一种日志操作")
	}
	for _, action := range actions {
		if action != "query" && action != "tail" && action != "export" {
			return logmodel.AccessPolicy{}, fmt.Errorf("不支持的日志操作: %s", action)
		}
	}
	return logmodel.AccessPolicy{
		Name: payload.Name, Description: strings.TrimSpace(payload.Description), SubjectType: payload.SubjectType,
		SubjectID: payload.SubjectID, DataSourceID: payload.StorageID, LibraryItemPattern: strings.TrimSpace(payload.LibraryItemPattern), ScopeMode: payload.ScopeMode,
		AllowedActions: encodeStringList(actions), DeniedFields: encodeStringList(uniqueNormalizedStrings(payload.DeniedFields)),
		MaskFields: encodeStringList(uniqueNormalizedStrings(payload.MaskFields)), Enabled: payload.Enabled,
		Scopes: accessPolicyScopes(collectionPolicyIDs),
	}, nil
}

func accessPolicyToView(policy logmodel.AccessPolicy) accessPolicyView {
	return accessPolicyView{
		ID: policy.ID, CreatedBy: policy.CreatedBy, UpdatedBy: policy.UpdatedBy,
		CreatedAt: policy.CreatedAt, UpdatedAt: policy.UpdatedAt,
		Payload: accessPolicyPayload{
			Name: policy.Name, Description: policy.Description, SubjectType: policy.SubjectType, SubjectID: policy.SubjectID,
			StorageID: policy.DataSourceID, LibraryItemPattern: policy.LibraryItemPattern,
			ScopeMode: normalizeStoredAccessScopeMode(policy.ScopeMode), CollectionPolicyIDs: accessPolicyCollectionPolicyIDs(policy.Scopes),
			AllowedActions: decodeStringList(policy.AllowedActions), DeniedFields: decodeStringList(policy.DeniedFields),
			MaskFields: decodeStringList(policy.MaskFields), Enabled: policy.Enabled,
		},
	}
}

func (h *Handler) saveAccessPolicy(c *gin.Context, policy *logmodel.AccessPolicy, update bool) error {
	scopes := append([]logmodel.AccessPolicyScope(nil), policy.Scopes...)
	return h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := validateAccessPolicyScopes(tx, policy.ScopeMode, scopes); err != nil {
			return err
		}
		if update {
			if err := tx.Omit("Scopes").Save(policy).Error; err != nil {
				return err
			}
			if err := tx.Where("access_policy_id = ?", policy.ID).Delete(&logmodel.AccessPolicyScope{}).Error; err != nil {
				return err
			}
		} else if err := tx.Omit("Scopes").Create(policy).Error; err != nil {
			return err
		}
		for index := range scopes {
			scopes[index].ID = 0
			scopes[index].AccessPolicyID = policy.ID
		}
		if len(scopes) > 0 {
			if err := tx.Create(&scopes).Error; err != nil {
				return err
			}
		}
		policy.Scopes = scopes
		return nil
	})
}

func validateAccessPolicyScopes(tx *gorm.DB, scopeMode string, scopes []logmodel.AccessPolicyScope) error {
	if normalizeStoredAccessScopeMode(scopeMode) == logAccessScopeAll {
		return nil
	}
	if len(scopes) == 0 {
		return fmt.Errorf("指定采集策略时至少选择一个采集策略")
	}
	ids := make([]uint, 0, len(scopes))
	for _, scope := range scopes {
		ids = append(ids, scope.CollectionPolicyID)
	}
	var count int64
	if err := tx.Model(&logmodel.CollectionPolicy{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return fmt.Errorf("选择的采集策略不存在或已被删除，请刷新后重新选择")
	}
	return nil
}

func normalizeStoredAccessScopeMode(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), logAccessScopeCollectionPolicy) {
		return logAccessScopeCollectionPolicy
	}
	return logAccessScopeAll
}

func accessPolicyScopes(ids []uint) []logmodel.AccessPolicyScope {
	result := make([]logmodel.AccessPolicyScope, 0, len(ids))
	for _, id := range ids {
		result = append(result, logmodel.AccessPolicyScope{CollectionPolicyID: id})
	}
	return result
}

func accessPolicyCollectionPolicyIDs(scopes []logmodel.AccessPolicyScope) []uint {
	values := make([]uint, 0, len(scopes))
	for _, scope := range scopes {
		values = append(values, scope.CollectionPolicyID)
	}
	return uniqueUintValues(values)
}

func uniqueUintValues(values []uint) []uint {
	seen := make(map[uint]struct{}, len(values))
	result := make([]uint, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(left, right int) bool { return result[left] < result[right] })
	return result
}

func uniqueUint64s(values []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(values))
	result := make([]uint64, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(left, right int) bool { return result[left] < result[right] })
	return result
}

func decodeStringList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	var values []string
	if strings.HasPrefix(raw, "[") && json.Unmarshal([]byte(raw), &values) == nil {
		return uniqueNormalizedStrings(values)
	}
	return uniqueNormalizedStrings(strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == '\n' }))
}

func encodeStringList(values []string) string {
	raw, _ := json.Marshal(uniqueNormalizedStrings(values))
	return string(raw)
}

func uniqueNormalizedStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func stringListContains(values []string, expected string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(expected)) {
			return true
		}
	}
	return false
}

func uint64SliceSet(values []uint64) map[uint64]struct{} {
	result := make(map[uint64]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func filterAllowedIDs(values []uint64, allowed map[uint64]struct{}) []uint64 {
	result := make([]uint64, 0, len(values))
	for _, value := range values {
		if _, ok := allowed[value]; ok {
			result = append(result, value)
		}
	}
	return result
}
