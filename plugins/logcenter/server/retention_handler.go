package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	rbacsvc "github.com/ydcloud-dy/opshub/internal/service/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
	logmodel "github.com/ydcloud-dy/opshub/plugins/logcenter/model"
	"gorm.io/gorm"
)

type retentionPolicyPayload struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	StorageID   uint           `json:"storageId"`
	DefaultDays int            `json:"defaultDays"`
	LevelDays   map[string]int `json:"levelDays"`
	Priority    int            `json:"priority"`
	Enabled     bool           `json:"enabled"`
}

type retentionPolicyView struct {
	ID                 uint                   `json:"id"`
	CreatedBy          uint                   `json:"createdBy"`
	UpdatedBy          uint                   `json:"updatedBy"`
	CreatedAt          time.Time              `json:"createdAt"`
	UpdatedAt          time.Time              `json:"updatedAt"`
	Payload            retentionPolicyPayload `json:"payload"`
	BoundPolicyCount   int64                  `json:"boundPolicyCount"`
	UpdatedPolicyCount int                    `json:"updatedPolicyCount,omitempty"`
}

func (h *Handler) ListRetentionPolicies(c *gin.Context) {
	if !h.requireRetentionPolicyAdmin(c) {
		return
	}
	var policies []logmodel.RetentionPolicy
	query := h.db.WithContext(c.Request.Context()).Order("updated_at DESC")
	if storageID := parseUint(c.Query("storageId")); storageID > 0 {
		query = query.Where("storage_id IN ?", []uint{0, storageID})
	}
	if err := query.Find(&policies).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取日志保留策略失败: "+err.Error())
		return
	}
	bindingCounts, err := retentionPolicyBindingCounts(h.db.WithContext(c.Request.Context()))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取保留策略绑定关系失败: "+err.Error())
		return
	}
	views := make([]retentionPolicyView, 0, len(policies))
	for _, policy := range policies {
		views = append(views, retentionPolicyToView(policy, bindingCounts[policy.ID], 0))
	}
	response.Success(c, views)
}

func (h *Handler) CreateRetentionPolicy(c *gin.Context) {
	if !h.requireRetentionPolicyAdmin(c) {
		return
	}
	var payload retentionPolicyPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	policy, err := retentionPolicyFromPayload(payload)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	policy.CreatedBy = rbacsvc.GetUserID(c)
	policy.UpdatedBy = policy.CreatedBy
	if err := h.db.WithContext(c.Request.Context()).Create(&policy).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建日志保留策略失败: "+err.Error())
		return
	}
	response.Success(c, retentionPolicyToView(policy, 0, 0))
}

func (h *Handler) UpdateRetentionPolicy(c *gin.Context) {
	if !h.requireRetentionPolicyAdmin(c) {
		return
	}
	var existing logmodel.RetentionPolicy
	if err := h.db.WithContext(c.Request.Context()).First(&existing, parseUint(c.Param("id"))).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "日志保留策略不存在")
		return
	}
	var payload retentionPolicyPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	policy, err := retentionPolicyFromPayload(payload)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	policy.ID = existing.ID
	policy.CreatedAt = existing.CreatedAt
	policy.CreatedBy = existing.CreatedBy
	policy.UpdatedBy = rbacsvc.GetUserID(c)
	updatedPolicyCount := 0
	if err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&policy).Error; err != nil {
			return err
		}
		if !policy.Enabled {
			return nil
		}
		count, err := syncRetentionPolicyDrafts(tx, policy, policy.UpdatedBy)
		updatedPolicyCount = count
		return err
	}); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新日志保留策略失败: "+err.Error())
		return
	}
	bindingCounts, err := retentionPolicyBindingCounts(h.db.WithContext(c.Request.Context()))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取保留策略绑定关系失败: "+err.Error())
		return
	}
	response.Success(c, retentionPolicyToView(policy, bindingCounts[policy.ID], updatedPolicyCount))
}

func (h *Handler) DeleteRetentionPolicy(c *gin.Context) {
	if !h.requireRetentionPolicyAdmin(c) {
		return
	}
	id := parseUint(c.Param("id"))
	if id == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "保留策略 ID 无效")
		return
	}
	bindingCounts, err := retentionPolicyBindingCounts(h.db.WithContext(c.Request.Context()))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取保留策略绑定关系失败: "+err.Error())
		return
	}
	if bindingCounts[id] > 0 {
		response.ErrorCode(c, http.StatusConflict, fmt.Sprintf("该保留策略仍被 %d 个采集策略引用，请先解除绑定", bindingCounts[id]))
		return
	}
	if err := h.db.WithContext(c.Request.Context()).Delete(&logmodel.RetentionPolicy{}, id).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除日志保留策略失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) GetStorageCapacity(c *gin.Context) {
	if !h.requireLogAdmin(c, "只有管理员可以查看日志容量预测") {
		return
	}
	storageID := parseUint(c.Query("storageId"))
	cluster, password, ok := h.internalStorage(c, storageID)
	if !ok {
		return
	}
	retentionDays := parseInt(c.Query("retentionDays"), 0)
	result, err := h.clickhouse.EstimateCapacity(c.Request.Context(), cluster, password, retentionDays)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "读取 ClickHouse 容量失败: "+err.Error())
		return
	}
	response.Success(c, result)
}

func retentionPolicyFromPayload(payload retentionPolicyPayload) (logmodel.RetentionPolicy, error) {
	payload.Name = strings.TrimSpace(payload.Name)
	if payload.Name == "" {
		return logmodel.RetentionPolicy{}, fmt.Errorf("保留策略名称不能为空")
	}
	if payload.DefaultDays <= 0 || payload.DefaultDays > 3650 {
		return logmodel.RetentionPolicy{}, fmt.Errorf("默认保留天数必须在 1 到 3650 天之间")
	}
	levelDays := make(map[string]int, len(payload.LevelDays))
	for level, days := range payload.LevelDays {
		level = strings.ToUpper(strings.TrimSpace(level))
		if level == "WARNING" {
			level = "WARN"
		}
		if days <= 0 || days > 3650 {
			return logmodel.RetentionPolicy{}, fmt.Errorf("%s 级别保留天数必须在 1 到 3650 天之间", level)
		}
		levelDays[level] = days
	}
	raw, _ := json.Marshal(levelDays)
	if payload.Priority <= 0 {
		payload.Priority = 100
	}
	return logmodel.RetentionPolicy{
		Name: payload.Name, Description: strings.TrimSpace(payload.Description), StorageID: payload.StorageID,
		DefaultDays: payload.DefaultDays, LevelDays: string(raw), Priority: payload.Priority, Enabled: payload.Enabled,
	}, nil
}

func retentionPolicyToView(policy logmodel.RetentionPolicy, boundPolicyCount int64, updatedPolicyCount int) retentionPolicyView {
	levelDays := make(map[string]int)
	_ = json.Unmarshal([]byte(policy.LevelDays), &levelDays)
	return retentionPolicyView{
		ID: policy.ID, CreatedBy: policy.CreatedBy, UpdatedBy: policy.UpdatedBy,
		CreatedAt: policy.CreatedAt, UpdatedAt: policy.UpdatedAt,
		Payload: retentionPolicyPayload{
			Name: policy.Name, Description: policy.Description, StorageID: policy.StorageID,
			DefaultDays: policy.DefaultDays, LevelDays: levelDays, Priority: policy.Priority, Enabled: policy.Enabled,
		},
		BoundPolicyCount: boundPolicyCount, UpdatedPolicyCount: updatedPolicyCount,
	}
}

func retentionPolicyBindingCounts(tx *gorm.DB) (map[uint]int64, error) {
	references := make(map[uint]map[uint]struct{})
	addReference := func(retentionPolicyID, collectionPolicyID uint) {
		if retentionPolicyID == 0 || collectionPolicyID == 0 {
			return
		}
		if references[retentionPolicyID] == nil {
			references[retentionPolicyID] = make(map[uint]struct{})
		}
		references[retentionPolicyID][collectionPolicyID] = struct{}{}
	}
	var policies []logmodel.CollectionPolicy
	if err := tx.Select("id", "retention_policy_id").Where("status <> ?", collectionPolicyStatusArchived).Find(&policies).Error; err != nil {
		return nil, err
	}
	activePolicyIDs := make(map[uint]struct{}, len(policies))
	for _, policy := range policies {
		activePolicyIDs[policy.ID] = struct{}{}
		addReference(policy.RetentionPolicyID, policy.ID)
	}
	var drafts []logmodel.PolicyRevision
	if err := tx.Select("policy_id", "content").Where("version = ?", 0).Find(&drafts).Error; err != nil {
		return nil, err
	}
	for _, draft := range drafts {
		if _, active := activePolicyIDs[draft.PolicyID]; !active {
			continue
		}
		var payload policyPayload
		if err := json.Unmarshal([]byte(draft.Content), &payload); err == nil {
			addReference(payload.RetentionPolicyID, draft.PolicyID)
		}
	}
	counts := make(map[uint]int64, len(references))
	for retentionPolicyID, collectionPolicies := range references {
		counts[retentionPolicyID] = int64(len(collectionPolicies))
	}
	return counts, nil
}

func syncRetentionPolicyDrafts(tx *gorm.DB, retentionPolicy logmodel.RetentionPolicy, userID uint) (int, error) {
	var policies []logmodel.CollectionPolicy
	if err := tx.Where("status <> ?", collectionPolicyStatusArchived).Find(&policies).Error; err != nil {
		return 0, err
	}
	updated := 0
	for index := range policies {
		policy := &policies[index]
		payload, hasDraft, err := loadPendingPolicyDraft(tx, policy.ID)
		if err != nil {
			return updated, err
		}
		if !hasDraft {
			payload, err = loadPolicyPayload(tx, *policy)
			if err != nil {
				return updated, err
			}
		}
		if payload.RetentionPolicyID != retentionPolicy.ID {
			continue
		}
		if err := applyRetentionPolicyValues(&payload, retentionPolicy); err != nil {
			return updated, err
		}
		payload.normalize()
		if hasDraft || policy.Version > 0 {
			if err := savePendingPolicyDraft(tx, policy.ID, payload, userID); err != nil {
				return updated, err
			}
			if err := tx.Model(policy).Updates(map[string]interface{}{"updated_by": userID}).Error; err != nil {
				return updated, err
			}
		} else {
			payload.applyToModel(policy)
			policy.UpdatedBy = userID
			if err := tx.Save(policy).Error; err != nil {
				return updated, err
			}
		}
		updated++
	}
	return updated, nil
}
