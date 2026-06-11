package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/plugins/monitor/model"
	"gorm.io/gorm"
)

type faultCenterSLOPoint struct {
	Date string  `json:"date"`
	MTTA float64 `json:"mtta"`
	MTTR float64 `json:"mttr"`
	MTBF float64 `json:"mtbf"`
}

func (h *DataSourceHandler) ListFaultCenters(c *gin.Context) {
	var centers []model.FaultCenter
	query := h.db.Model(&model.FaultCenter{})
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", like, like)
	}
	if err := query.Order("id DESC").Find(&centers).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "获取故障中心列表失败", "error": err.Error()})
		return
	}
	for i := range centers {
		h.fillFaultCenterCounters(&centers[i])
	}
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": centers})
}

func (h *DataSourceHandler) GetFaultCenter(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var center model.FaultCenter
	if err := h.db.First(&center, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "故障中心不存在"})
		return
	}
	h.fillFaultCenterCounters(&center)
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": center})
}

func (h *DataSourceHandler) CreateFaultCenter(c *gin.Context) {
	var req model.FaultCenter
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	normalizeFaultCenter(&req)
	if err := validateFaultCenter(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "创建故障中心失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "创建成功", "data": req})
}

func (h *DataSourceHandler) UpdateFaultCenter(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var center model.FaultCenter
	if err := h.db.First(&center, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "故障中心不存在"})
		return
	}
	var req model.FaultCenter
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	req.ID = center.ID
	normalizeFaultCenter(&req)
	if err := validateFaultCenter(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}

	center.Name = req.Name
	center.Description = req.Description
	center.NoticeObjectIDs = req.NoticeObjectIDs
	center.NoticeChannelIDs = req.NoticeChannelIDs
	center.NoticeRoutes = req.NoticeRoutes
	center.RepeatNoticeInterval = req.RepeatNoticeInterval
	center.RecoverNotify = req.RecoverNotify
	center.AggregationType = req.AggregationType
	center.SilenceEnabled = req.SilenceEnabled
	center.SilenceRules = req.SilenceRules
	center.RecoverWaitSeconds = req.RecoverWaitSeconds
	center.UpgradeEnabled = req.UpgradeEnabled
	center.UpgradableSeverities = req.UpgradableSeverities
	center.UpgradeStrategy = req.UpgradeStrategy

	if err := h.db.Save(&center).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "更新故障中心失败", "error": err.Error()})
		return
	}
	if err := h.resyncFaultCenterSilenceEvents(center.ID); err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "同步静默事件状态失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "更新成功", "data": center})
}

func (h *DataSourceHandler) DeleteFaultCenter(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var ruleCount int64
	if err := h.db.Model(&model.AlertRule{}).Where("fault_center_id = ?", id).Count(&ruleCount).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "检查故障中心引用失败", "error": err.Error()})
		return
	}
	if ruleCount > 0 {
		c.JSON(400, gin.H{"code": 400, "message": "该故障中心已有告警规则引用，请先调整规则路由"})
		return
	}
	if err := h.db.Delete(&model.FaultCenter{}, id).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "删除故障中心失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "删除成功"})
}

func (h *DataSourceHandler) GetFaultCenterSLO(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	start := startOfDay(time.Now()).AddDate(0, 0, -6)
	var events []model.AlertEvent
	if err := h.db.Where("fault_center_id = ? AND started_at >= ?", id, start).
		Order("started_at ASC").
		Find(&events).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "获取 SLO 数据失败", "error": err.Error()})
		return
	}

	points := make([]faultCenterSLOPoint, 0, 7)
	for i := 0; i < 7; i++ {
		day := start.AddDate(0, 0, i)
		next := day.AddDate(0, 0, 1)
		var mttaTotal, mttrTotal, mtbfTotal float64
		var mttaCount, mttrCount, mtbfCount int
		var lastFailureStart *time.Time
		for _, event := range events {
			if event.AcknowledgedAt != nil && !event.AcknowledgedAt.Before(day) && event.AcknowledgedAt.Before(next) {
				mttaTotal += event.AcknowledgedAt.Sub(event.StartedAt).Seconds()
				mttaCount++
			}
			if event.EndedAt != nil && !event.EndedAt.Before(day) && event.EndedAt.Before(next) {
				mttrTotal += event.EndedAt.Sub(event.StartedAt).Seconds()
				mttrCount++
			}
			if (event.State == "firing" || event.State == "error") && !event.StartedAt.Before(day) && event.StartedAt.Before(next) {
				if lastFailureStart != nil {
					mtbfTotal += event.StartedAt.Sub(*lastFailureStart).Seconds()
					mtbfCount++
				}
				startedAt := event.StartedAt
				lastFailureStart = &startedAt
			}
		}
		point := faultCenterSLOPoint{Date: day.Format("01-02")}
		if mttaCount > 0 {
			point.MTTA = mttaTotal / float64(mttaCount)
		}
		if mttrCount > 0 {
			point.MTTR = mttrTotal / float64(mttrCount)
		}
		if mtbfCount > 0 {
			point.MTBF = mtbfTotal / float64(mtbfCount)
		}
		points = append(points, point)
	}
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": points})
}

func (h *DataSourceHandler) AcknowledgeAlertEvent(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req struct {
		Username string `json:"username"`
	}
	_ = c.ShouldBindJSON(&req)
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		req.Username = "admin"
	}
	now := time.Now()
	updates := map[string]interface{}{
		"acknowledged":    true,
		"acknowledged_by": req.Username,
		"acknowledged_at": now,
		"state":           "processing",
		"last_eval_at":    now,
	}
	if err := h.db.Model(&model.AlertEvent{}).
		Where("id = ? AND ended_at IS NULL", id).
		Updates(updates).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "认领告警事件失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "认领成功"})
}

func (h *DataSourceHandler) BatchAcknowledgeAlertEvents(c *gin.Context) {
	var req struct {
		IDs      []uint `json:"ids"`
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	ids := uniqueUintIDs(req.IDs)
	if len(ids) == 0 {
		c.JSON(400, gin.H{"code": 400, "message": "请选择需要认领的告警事件"})
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		req.Username = "admin"
	}
	now := time.Now()
	updates := map[string]interface{}{
		"acknowledged":    true,
		"acknowledged_by": req.Username,
		"acknowledged_at": now,
		"state":           "processing",
		"last_eval_at":    now,
	}
	result := h.db.Model(&model.AlertEvent{}).
		Where("id IN ? AND state IN ? AND ended_at IS NULL", ids, activeAlertEventStates()).
		Updates(updates)
	if result.Error != nil {
		c.JSON(500, gin.H{"code": 500, "message": "批量认领告警事件失败", "error": result.Error.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "批量认领成功", "data": gin.H{"updated": result.RowsAffected}})
}

func (h *DataSourceHandler) SilenceAlertEvent(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req struct {
		Username string `json:"username"`
	}
	_ = c.ShouldBindJSON(&req)
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		req.Username = "admin"
	}
	now := time.Now()
	updates := map[string]interface{}{
		"state":           "silenced",
		"acknowledged":    true,
		"acknowledged_by": req.Username,
		"acknowledged_at": now,
		"last_eval_at":    now,
	}
	if err := h.db.Model(&model.AlertEvent{}).
		Where("id = ? AND ended_at IS NULL", id).
		Updates(updates).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "静默告警事件失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "静默成功"})
}

func (h *DataSourceHandler) DeleteAlertEvent(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := h.db.Delete(&model.AlertEvent{}, id).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "删除告警事件失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "删除成功"})
}

func (h *DataSourceHandler) BatchDeleteAlertEvents(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	ids := uniqueUintIDs(req.IDs)
	if len(ids) == 0 {
		c.JSON(400, gin.H{"code": 400, "message": "请选择需要删除的告警事件"})
		return
	}
	result := h.db.Where("id IN ?", ids).Delete(&model.AlertEvent{})
	if result.Error != nil {
		c.JSON(500, gin.H{"code": 500, "message": "批量删除告警事件失败", "error": result.Error.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "批量删除成功", "data": gin.H{"deleted": result.RowsAffected}})
}

func (h *DataSourceHandler) ListRuleGroups(c *gin.Context) {
	var groups []model.AlertRuleGroup
	if err := h.db.Order("sort ASC, id ASC").Find(&groups).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "获取规则组失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": groups})
}

func (h *DataSourceHandler) CreateRuleGroup(c *gin.Context) {
	var req model.AlertRuleGroup
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	normalizeRuleGroup(&req)
	if req.Name == "" {
		c.JSON(400, gin.H{"code": 400, "message": "请输入规则组名称"})
		return
	}
	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "创建规则组失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "创建成功", "data": req})
}

func (h *DataSourceHandler) UpdateRuleGroup(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var group model.AlertRuleGroup
	if err := h.db.First(&group, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "规则组不存在"})
		return
	}
	var req model.AlertRuleGroup
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	normalizeRuleGroup(&req)
	if req.Name == "" {
		c.JSON(400, gin.H{"code": 400, "message": "请输入规则组名称"})
		return
	}
	group.Name = req.Name
	group.Description = req.Description
	group.Sort = req.Sort
	if err := h.db.Save(&group).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "更新规则组失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "更新成功", "data": group})
}

func (h *DataSourceHandler) DeleteRuleGroup(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var ruleCount int64
	if err := h.db.Model(&model.AlertRule{}).Where("rule_group_id = ?", id).Count(&ruleCount).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "检查规则组引用失败", "error": err.Error()})
		return
	}
	if ruleCount > 0 {
		c.JSON(400, gin.H{"code": 400, "message": "该规则组已有告警规则引用，请先移动规则"})
		return
	}
	if err := h.db.Delete(&model.AlertRuleGroup{}, id).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "删除规则组失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "删除成功"})
}

func (h *DataSourceHandler) fillFaultCenterCounters(center *model.FaultCenter) {
	_ = h.ensurePendingAlertEvents(context.Background(), strconv.FormatUint(uint64(center.ID), 10))
	_ = h.db.Model(&model.AlertEvent{}).
		Distinct("rule_id").
		Where("fault_center_id = ? AND state = ? AND ended_at IS NULL", center.ID, "pending").
		Count(&center.CurrentPreAlertNumber).Error
	_ = h.db.Model(&model.AlertEvent{}).
		Distinct("rule_id").
		Where("fault_center_id = ? AND state IN ? AND ended_at IS NULL", center.ID, activeAlertEventStates()).
		Count(&center.CurrentAlertNumber).Error

	wait := center.RecoverWaitSeconds
	if wait <= 0 {
		wait = 30
	}
	since := time.Now().Add(-time.Duration(wait) * time.Second)
	_ = h.db.Model(&model.AlertEvent{}).
		Where("fault_center_id = ? AND state = ? AND last_eval_at >= ?", center.ID, "recovered", since).
		Count(&center.CurrentRecoverNumber).Error
}

func normalizeFaultCenter(center *model.FaultCenter) {
	center.Name = strings.TrimSpace(center.Name)
	center.Description = strings.TrimSpace(center.Description)
	center.AggregationType = strings.TrimSpace(center.AggregationType)
	if center.AggregationType == "" {
		center.AggregationType = "Rule"
	}
	if center.RecoverWaitSeconds <= 0 {
		center.RecoverWaitSeconds = 30
	}
	center.NoticeObjectIDs = normalizeJSONText(center.NoticeObjectIDs, "[]")
	center.NoticeChannelIDs = normalizeJSONText(center.NoticeChannelIDs, "[]")
	center.NoticeRoutes = normalizeJSONText(center.NoticeRoutes, "[]")
	center.SilenceRules = normalizeJSONText(center.SilenceRules, "[]")
	center.RepeatNoticeInterval = normalizeJSONText(center.RepeatNoticeInterval, `{"p0":30,"p1":60,"p2":120}`)
	center.UpgradableSeverities = normalizeJSONText(center.UpgradableSeverities, `["p0","p1"]`)
	center.UpgradeStrategy = normalizeJSONText(center.UpgradeStrategy, `{"enabled":false,"timeout":30,"repeatInterval":60,"noticeObjectIds":[]}`)
}

func validateFaultCenter(center *model.FaultCenter) error {
	if center.Name == "" {
		return fmt.Errorf("请输入故障中心名称")
	}
	for name, raw := range map[string]string{
		"通知对象": center.NoticeObjectIDs,
		"通知通道": center.NoticeChannelIDs,
		"通知路由": center.NoticeRoutes,
		"静默规则": center.SilenceRules,
		"重复通知": center.RepeatNoticeInterval,
		"升级等级": center.UpgradableSeverities,
		"升级策略": center.UpgradeStrategy,
	} {
		if !json.Valid([]byte(raw)) {
			return fmt.Errorf("%s配置必须是合法 JSON", name)
		}
	}
	return nil
}

func normalizeRuleGroup(group *model.AlertRuleGroup) {
	group.Name = strings.TrimSpace(group.Name)
	group.Description = strings.TrimSpace(group.Description)
}

func normalizeJSONText(raw, fallback string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	return raw
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func findDefaultFaultCenter(db *gorm.DB) uint {
	var center model.FaultCenter
	if err := db.Order("id ASC").First(&center).Error; err != nil {
		return 0
	}
	return center.ID
}

func findDefaultRuleGroup(db *gorm.DB) uint {
	var group model.AlertRuleGroup
	if err := db.Order("sort ASC, id ASC").First(&group).Error; err != nil {
		return 0
	}
	return group.ID
}
