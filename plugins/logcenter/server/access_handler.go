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
)

type logAccessDecision struct {
	IsAdmin                 bool
	AllowedHostIDs          []uint64
	AllowedKubernetesScopes map[uint64][]string
	DeniedFields            []string
	MaskFields              []string
}

type accessPolicyPayload struct {
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	SubjectType        string   `json:"subjectType"`
	SubjectID          uint     `json:"subjectId"`
	StorageID          uint     `json:"storageId"`
	LibraryItemPattern string   `json:"libraryItemPattern"`
	AllowedActions     []string `json:"allowedActions"`
	DeniedFields       []string `json:"deniedFields"`
	MaskFields         []string `json:"maskFields"`
	Enabled            bool     `json:"enabled"`
}

type accessPolicyView struct {
	ID        uint                `json:"id"`
	CreatedBy uint                `json:"createdBy"`
	UpdatedBy uint                `json:"updatedBy"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
	Payload   accessPolicyPayload `json:"payload"`
}

func (h *Handler) authorizeInternalAction(c *gin.Context, action string, storageID uint) (logAccessDecision, bool) {
	allowedHostIDs, isAdmin, err := h.userAccessibleHostIDs(c)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取主机日志权限失败: "+err.Error())
		return logAccessDecision{}, false
	}
	decision := logAccessDecision{IsAdmin: isAdmin, AllowedHostIDs: allowedHostIDs}
	if isAdmin {
		return decision, true
	}
	kubernetesScopes, err := h.userAccessibleKubernetesScopes(c)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取 Kubernetes 日志权限失败: "+err.Error())
		return logAccessDecision{}, false
	}
	decision.AllowedKubernetesScopes = kubernetesScopes

	effectiveStorageID, err := h.effectiveStorageID(c, storageID)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取日志存储权限失败: "+err.Error())
		return logAccessDecision{}, false
	}
	policies, err := h.userLogAccessPolicies(c, effectiveStorageID)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取日志访问策略失败: "+err.Error())
		return logAccessDecision{}, false
	}
	if len(policies) == 0 {
		return decision, true
	}
	actionAllowed := false
	for _, policy := range policies {
		if stringListContains(decodeStringList(policy.AllowedActions), action) {
			actionAllowed = true
		}
		decision.DeniedFields = append(decision.DeniedFields, decodeStringList(policy.DeniedFields)...)
		decision.MaskFields = append(decision.MaskFields, decodeStringList(policy.MaskFields)...)
	}
	decision.DeniedFields = uniqueNormalizedStrings(decision.DeniedFields)
	decision.MaskFields = uniqueNormalizedStrings(decision.MaskFields)
	if !actionAllowed {
		response.ErrorCode(c, http.StatusForbidden, fmt.Sprintf("当前日志访问策略不允许执行 %s 操作", action))
		return logAccessDecision{}, false
	}
	return decision, true
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
	req.AllowedKubernetesScopes = decision.AllowedKubernetesScopes
	req.DeniedFields = decision.DeniedFields
	req.MaskFields = decision.MaskFields
	return true
}

func validateQueryFieldAccess(c *gin.Context, req logsvc.InternalQueryRequest, deniedFields []string) bool {
	if strings.TrimSpace(req.Query) != "" && strings.TrimSpace(req.Query) != "*" &&
		(logsvc.FieldAccessDenied("body", deniedFields) || logsvc.FieldAccessDenied("message", deniedFields)) {
		response.ErrorCode(c, http.StatusForbidden, "当前日志访问策略不允许检索日志正文")
		return false
	}
	for _, filter := range req.Filters {
		if logsvc.FieldAccessDenied(filter.Field, deniedFields) {
			response.ErrorCode(c, http.StatusForbidden, "当前日志访问策略不允许筛选字段: "+filter.Field)
			return false
		}
	}
	return true
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
	userID := rbacsvc.GetUserID(c)
	var roleIDs []uint
	if err := h.db.WithContext(c.Request.Context()).Table("sys_user_role").Where("user_id = ?", userID).Pluck("role_id", &roleIDs).Error; err != nil {
		return nil, err
	}
	query := h.db.WithContext(c.Request.Context()).Where("enabled = ? AND data_source_id IN ?", true, []uint{0, storageID})
	if len(roleIDs) == 0 {
		query = query.Where("subject_type = ? AND subject_id = ?", "user", userID)
	} else {
		query = query.Where("(subject_type = ? AND subject_id = ?) OR (subject_type = ? AND subject_id IN ?)", "user", userID, "role", roleIDs)
	}
	var policies []logmodel.AccessPolicy
	return policies, query.Order("id ASC").Find(&policies).Error
}

func (h *Handler) ListAccessPolicies(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
		return
	}
	var policies []logmodel.AccessPolicy
	if err := h.db.WithContext(c.Request.Context()).Order("updated_at DESC").Find(&policies).Error; err != nil {
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
	if !h.requirePolicyAdmin(c) {
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
	if err := h.db.WithContext(c.Request.Context()).Create(&policy).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建日志访问策略失败: "+err.Error())
		return
	}
	response.Success(c, accessPolicyToView(policy))
}

func (h *Handler) UpdateAccessPolicy(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
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
	if err := h.db.WithContext(c.Request.Context()).Save(&policy).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新日志访问策略失败: "+err.Error())
		return
	}
	response.Success(c, accessPolicyToView(policy))
}

func (h *Handler) DeleteAccessPolicy(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
		return
	}
	id := parseUint(c.Param("id"))
	if id == 0 || h.db.WithContext(c.Request.Context()).Delete(&logmodel.AccessPolicy{}, id).Error != nil {
		response.ErrorCode(c, http.StatusBadRequest, "删除日志访问策略失败")
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) GetAccessPolicyOptions(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
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
	if err := h.db.WithContext(c.Request.Context()).Table("sys_user").Select("id, username, real_name").Where("deleted_at IS NULL").Order("username ASC").Scan(&users).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取用户选项失败: "+err.Error())
		return
	}
	if err := h.db.WithContext(c.Request.Context()).Table("sys_role").Select("id, name, code").Where("deleted_at IS NULL").Order("sort ASC, id ASC").Scan(&roles).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取角色选项失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"users": users, "roles": roles})
}

func accessPolicyFromPayload(payload accessPolicyPayload) (logmodel.AccessPolicy, error) {
	payload.Name = strings.TrimSpace(payload.Name)
	payload.SubjectType = strings.ToLower(strings.TrimSpace(payload.SubjectType))
	if payload.Name == "" || payload.SubjectID == 0 {
		return logmodel.AccessPolicy{}, fmt.Errorf("策略名称和授权对象不能为空")
	}
	if payload.SubjectType != "user" && payload.SubjectType != "role" {
		return logmodel.AccessPolicy{}, fmt.Errorf("授权对象类型仅支持用户或角色")
	}
	actions := uniqueNormalizedStrings(payload.AllowedActions)
	for _, action := range actions {
		if action != "query" && action != "tail" && action != "export" {
			return logmodel.AccessPolicy{}, fmt.Errorf("不支持的日志操作: %s", action)
		}
	}
	return logmodel.AccessPolicy{
		Name: payload.Name, Description: strings.TrimSpace(payload.Description), SubjectType: payload.SubjectType,
		SubjectID: payload.SubjectID, DataSourceID: payload.StorageID, LibraryItemPattern: strings.TrimSpace(payload.LibraryItemPattern),
		AllowedActions: encodeStringList(actions), DeniedFields: encodeStringList(uniqueNormalizedStrings(payload.DeniedFields)),
		MaskFields: encodeStringList(uniqueNormalizedStrings(payload.MaskFields)), Enabled: payload.Enabled,
	}, nil
}

func accessPolicyToView(policy logmodel.AccessPolicy) accessPolicyView {
	return accessPolicyView{
		ID: policy.ID, CreatedBy: policy.CreatedBy, UpdatedBy: policy.UpdatedBy,
		CreatedAt: policy.CreatedAt, UpdatedAt: policy.UpdatedAt,
		Payload: accessPolicyPayload{
			Name: policy.Name, Description: policy.Description, SubjectType: policy.SubjectType, SubjectID: policy.SubjectID,
			StorageID: policy.DataSourceID, LibraryItemPattern: policy.LibraryItemPattern,
			AllowedActions: decodeStringList(policy.AllowedActions), DeniedFields: decodeStringList(policy.DeniedFields),
			MaskFields: decodeStringList(policy.MaskFields), Enabled: policy.Enabled,
		},
	}
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
