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
	ID        uint                   `json:"id"`
	CreatedBy uint                   `json:"createdBy"`
	UpdatedBy uint                   `json:"updatedBy"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
	Payload   retentionPolicyPayload `json:"payload"`
}

func (h *Handler) ListRetentionPolicies(c *gin.Context) {
	var policies []logmodel.RetentionPolicy
	query := h.db.WithContext(c.Request.Context()).Order("priority ASC, updated_at DESC")
	if storageID := parseUint(c.Query("storageId")); storageID > 0 {
		query = query.Where("storage_id IN ?", []uint{0, storageID})
	}
	if err := query.Find(&policies).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取日志保留策略失败: "+err.Error())
		return
	}
	views := make([]retentionPolicyView, 0, len(policies))
	for _, policy := range policies {
		views = append(views, retentionPolicyToView(policy))
	}
	response.Success(c, views)
}

func (h *Handler) CreateRetentionPolicy(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
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
	response.Success(c, retentionPolicyToView(policy))
}

func (h *Handler) UpdateRetentionPolicy(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
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
	if err := h.db.WithContext(c.Request.Context()).Save(&policy).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新日志保留策略失败: "+err.Error())
		return
	}
	response.Success(c, retentionPolicyToView(policy))
}

func (h *Handler) DeleteRetentionPolicy(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
		return
	}
	id := parseUint(c.Param("id"))
	if id == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "保留策略 ID 无效")
		return
	}
	if err := h.db.WithContext(c.Request.Context()).Delete(&logmodel.RetentionPolicy{}, id).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除日志保留策略失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *Handler) GetStorageCapacity(c *gin.Context) {
	if !h.requirePolicyAdmin(c) {
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

func retentionPolicyToView(policy logmodel.RetentionPolicy) retentionPolicyView {
	levelDays := make(map[string]int)
	_ = json.Unmarshal([]byte(policy.LevelDays), &levelDays)
	return retentionPolicyView{
		ID: policy.ID, CreatedBy: policy.CreatedBy, UpdatedBy: policy.UpdatedBy,
		CreatedAt: policy.CreatedAt, UpdatedAt: policy.UpdatedAt,
		Payload: retentionPolicyPayload{
			Name: policy.Name, Description: policy.Description, StorageID: policy.StorageID,
			DefaultDays: policy.DefaultDays, LevelDays: levelDays, Priority: policy.Priority, Enabled: policy.Enabled,
		},
	}
}
