package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/plugins/monitor/model"
	"gorm.io/gorm"
)

func (h *DataSourceHandler) ListNoticeTemplates(c *gin.Context) {
	var templates []model.NoticeTemplate
	query := h.db.Model(&model.NoticeTemplate{})
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", like, like)
	}
	if noticeType := strings.TrimSpace(c.Query("noticeType")); noticeType != "" {
		query = query.Where("notice_type = ?", noticeType)
	}
	if err := query.Order("notice_type ASC, id DESC").Find(&templates).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "获取通知模板失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": templates})
}

func (h *DataSourceHandler) GetNoticeTemplate(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var template model.NoticeTemplate
	if err := h.db.First(&template, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "通知模板不存在"})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": template})
}

func (h *DataSourceHandler) CreateNoticeTemplate(c *gin.Context) {
	var req model.NoticeTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	normalizeNoticeTemplate(&req)
	if err := validateNoticeTemplate(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "创建通知模板失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "创建成功", "data": req})
}

func (h *DataSourceHandler) UpdateNoticeTemplate(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var template model.NoticeTemplate
	if err := h.db.First(&template, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "通知模板不存在"})
		return
	}
	var req model.NoticeTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	normalizeNoticeTemplate(&req)
	if err := validateNoticeTemplate(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}
	template.Name = req.Name
	template.NoticeType = req.NoticeType
	template.Description = req.Description
	template.Template = req.Template
	template.TemplateFiring = req.TemplateFiring
	template.TemplateRecover = req.TemplateRecover
	template.EnableFeiShuJSONCard = req.EnableFeiShuJSONCard
	template.Enabled = req.Enabled
	if err := h.db.Save(&template).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "更新通知模板失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "更新成功", "data": template})
}

func (h *DataSourceHandler) DeleteNoticeTemplate(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var objects []model.NoticeObject
	if err := h.db.Find(&objects).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "检查模板引用失败", "error": err.Error()})
		return
	}
	idText := fmt.Sprintf(`"noticeTemplateId":"%d"`, id)
	for _, object := range objects {
		if strings.Contains(object.Routes, idText) {
			c.JSON(400, gin.H{"code": 400, "message": "该通知模板已被通知对象引用，请先调整通知对象"})
			return
		}
	}
	if err := h.db.Delete(&model.NoticeTemplate{}, id).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "删除通知模板失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "删除成功"})
}

func (h *DataSourceHandler) ListNoticeObjects(c *gin.Context) {
	var objects []model.NoticeObject
	query := h.db.Model(&model.NoticeObject{})
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR description LIKE ? OR uuid LIKE ?", like, like, like)
	}
	if enabled := strings.TrimSpace(c.Query("enabled")); enabled != "" {
		query = query.Where("enabled = ?", enabled == "true" || enabled == "1")
	}
	if err := query.Order("id DESC").Find(&objects).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "获取通知对象失败", "error": err.Error()})
		return
	}
	for i := range objects {
		h.fillNoticeObjectRuntime(&objects[i])
	}
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": objects})
}

func (h *DataSourceHandler) GetNoticeObject(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var object model.NoticeObject
	if err := h.db.First(&object, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "通知对象不存在"})
		return
	}
	h.fillNoticeObjectRuntime(&object)
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": object})
}

func (h *DataSourceHandler) CreateNoticeObject(c *gin.Context) {
	var req model.NoticeObject
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	normalizeNoticeObject(&req)
	if err := validateNoticeObject(h.db, &req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if req.UUID == "" {
		req.UUID = generateMonitorUUID("notice")
	}
	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "创建通知对象失败", "error": err.Error()})
		return
	}
	h.fillNoticeObjectRuntime(&req)
	c.JSON(200, gin.H{"code": 0, "message": "创建成功", "data": req})
}

func (h *DataSourceHandler) UpdateNoticeObject(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var object model.NoticeObject
	if err := h.db.First(&object, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "通知对象不存在"})
		return
	}
	var req model.NoticeObject
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	normalizeNoticeObject(&req)
	if err := validateNoticeObject(h.db, &req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}
	object.Name = req.Name
	object.Description = req.Description
	object.DutyTableID = req.DutyTableID
	object.Routes = req.Routes
	object.Enabled = req.Enabled
	object.LastStatus = req.LastStatus
	if object.LastStatus == "" {
		object.LastStatus = "ready"
	}
	if err := h.db.Save(&object).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "更新通知对象失败", "error": err.Error()})
		return
	}
	h.fillNoticeObjectRuntime(&object)
	c.JSON(200, gin.H{"code": 0, "message": "更新成功", "data": object})
}

func (h *DataSourceHandler) TestNoticeObject(c *gin.Context) {
	var req struct {
		NoticeObject model.NoticeObject `json:"noticeObject"`
		RouteIndex   int                `json:"routeIndex"`
		Severity     string             `json:"severity"`
		State        string             `json:"state"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	object := req.NoticeObject
	normalizeNoticeObject(&object)
	if object.Name == "" {
		object.Name = "通知对象测试"
	}
	if !json.Valid([]byte(object.Routes)) {
		c.JSON(400, gin.H{"code": 400, "message": "通知路由必须是合法 JSON"})
		return
	}
	routes := parseNoticeObjectRoutes(object.Routes)
	if len(routes) == 0 {
		c.JSON(400, gin.H{"code": 400, "message": "请至少配置一条通知路由"})
		return
	}
	if req.RouteIndex < 0 || req.RouteIndex >= len(routes) {
		c.JSON(400, gin.H{"code": 400, "message": "通知策略不存在"})
		return
	}
	route := routes[req.RouteIndex]
	if !noticeRouteEnabled(route) {
		c.JSON(400, gin.H{"code": 400, "message": "当前通知策略已停用"})
		return
	}
	if object.DutyTableID > 0 {
		h.fillNoticeObjectRuntime(&object)
	}
	severity := normalizeSeverityLevel(firstNonEmpty(req.Severity, "P1"))
	state := strings.TrimSpace(req.State)
	if state == "" {
		state = "firing"
	}
	now := time.Now()
	payload := ruleNotificationPayload{
		RuleID:         0,
		RuleName:       "OpsHub 通知测试",
		FaultCenterID:  0,
		DataSourceID:   0,
		DataSourceName: "OpsHub 测试数据源",
		DataSourceType: "prometheus",
		Severity:       severity,
		State:          state,
		Value:          95.2,
		Condition:      "gt",
		Threshold:      90,
		Message:        "这是一条 OpsHub 通知对象测试消息，用于验证 Hook、模板和值班表 @ 能力。",
		Labels:         `{"instance":"opshub-test","service":"monitor","env":"test"}`,
		Annotations:    `{"summary":"通知对象测试","description":"用于验证通知策略是否可达"}`,
		Fingerprint:    generateMonitorUUID("notice-test"),
		StartedAt:      now.Add(-5 * time.Minute),
		Time:           now,
	}
	if state == "recovered" {
		payload.EndedAt = &now
	}
	if err := h.sendRuleNotificationToNoticeRoute(c.Request.Context(), object, route, payload); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "通知测试失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "通知测试已发送"})
}

func (h *DataSourceHandler) DeleteNoticeObject(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var centers []model.FaultCenter
	if err := h.db.Find(&centers).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "检查通知对象引用失败", "error": err.Error()})
		return
	}
	idText := fmt.Sprintf("%d", id)
	for _, center := range centers {
		if jsonArrayContainsID(center.NoticeObjectIDs, idText) || strings.Contains(center.NoticeRoutes, idText) || strings.Contains(center.UpgradeStrategy, idText) {
			c.JSON(400, gin.H{"code": 400, "message": "该通知对象已被故障中心引用，请先调整故障中心配置"})
			return
		}
	}
	if err := h.db.Delete(&model.NoticeObject{}, id).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "删除通知对象失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "删除成功"})
}

func (h *DataSourceHandler) ListDutyTables(c *gin.Context) {
	var tables []model.DutyTable
	query := h.db.Model(&model.DutyTable{})
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR description LIKE ? OR manager_username LIKE ?", like, like, like)
	}
	if err := query.Order("id DESC").Find(&tables).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "获取值班表失败", "error": err.Error()})
		return
	}
	for i := range tables {
		h.fillDutyTableRuntime(&tables[i])
	}
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": tables})
}

func (h *DataSourceHandler) GetDutyTable(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var table model.DutyTable
	if err := h.db.First(&table, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "值班表不存在"})
		return
	}
	h.fillDutyTableRuntime(&table)
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": table})
}

func (h *DataSourceHandler) CreateDutyTable(c *gin.Context) {
	var req model.DutyTable
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	normalizeDutyTable(&req)
	if err := validateDutyTable(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "创建值班表失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "创建成功", "data": req})
}

func (h *DataSourceHandler) UpdateDutyTable(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var table model.DutyTable
	if err := h.db.First(&table, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "值班表不存在"})
		return
	}
	var req model.DutyTable
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	normalizeDutyTable(&req)
	if err := validateDutyTable(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}
	table.Name = req.Name
	table.Description = req.Description
	table.ManagerUserID = req.ManagerUserID
	table.ManagerUsername = req.ManagerUsername
	table.Enabled = req.Enabled
	if err := h.db.Save(&table).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "更新值班表失败", "error": err.Error()})
		return
	}
	h.fillDutyTableRuntime(&table)
	c.JSON(200, gin.H{"code": 0, "message": "更新成功", "data": table})
}

func (h *DataSourceHandler) DeleteDutyTable(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var refs int64
	if err := h.db.Model(&model.NoticeObject{}).Where("duty_table_id = ?", id).Count(&refs).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "检查值班表引用失败", "error": err.Error()})
		return
	}
	if refs > 0 {
		c.JSON(400, gin.H{"code": 400, "message": "该值班表已被通知对象引用，请先调整通知对象"})
		return
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("duty_table_id = ?", id).Delete(&model.DutySchedule{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.DutyTable{}, id).Error
	}); err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "删除值班表失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "删除成功"})
}

func (h *DataSourceHandler) ListDutySchedules(c *gin.Context) {
	var schedules []model.DutySchedule
	query := h.db.Model(&model.DutySchedule{})
	if dutyTableID := strings.TrimSpace(c.Query("dutyTableId")); dutyTableID != "" {
		query = query.Where("duty_table_id = ?", dutyTableID)
	}
	if month := strings.TrimSpace(c.Query("month")); month != "" {
		query = query.Where("duty_date LIKE ?", month+"%")
	}
	if err := query.Order("duty_date ASC, id ASC").Find(&schedules).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "获取值班日程失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": schedules})
}

func (h *DataSourceHandler) UpsertDutySchedule(c *gin.Context) {
	var req model.DutySchedule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	normalizeDutySchedule(&req)
	if err := validateDutySchedule(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}
	var schedule model.DutySchedule
	err := h.db.Where("duty_table_id = ? AND duty_date = ?", req.DutyTableID, req.DutyDate).First(&schedule).Error
	if err == nil {
		schedule.Users = req.Users
		schedule.Status = req.Status
		if err := h.db.Save(&schedule).Error; err != nil {
			c.JSON(500, gin.H{"code": 500, "message": "更新值班日程失败", "error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"code": 0, "message": "更新成功", "data": schedule})
		return
	}
	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "创建值班日程失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "创建成功", "data": req})
}

func (h *DataSourceHandler) UpdateDutySchedule(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var schedule model.DutySchedule
	if err := h.db.First(&schedule, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "值班日程不存在"})
		return
	}
	var req model.DutySchedule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数错误", "error": err.Error()})
		return
	}
	normalizeDutySchedule(&req)
	if err := validateDutySchedule(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}
	schedule.DutyTableID = req.DutyTableID
	schedule.DutyDate = req.DutyDate
	schedule.Users = req.Users
	schedule.Status = req.Status
	if err := h.db.Save(&schedule).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "更新值班日程失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "更新成功", "data": schedule})
}

func (h *DataSourceHandler) DeleteDutySchedule(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := h.db.Delete(&model.DutySchedule{}, id).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "删除值班日程失败", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "删除成功"})
}

func normalizeNoticeTemplate(template *model.NoticeTemplate) {
	template.Name = strings.TrimSpace(template.Name)
	template.NoticeType = normalizeNoticeType(template.NoticeType)
	template.Description = strings.TrimSpace(template.Description)
	template.Template = ""
	template.TemplateFiring = strings.TrimSpace(template.TemplateFiring)
	template.TemplateRecover = strings.TrimSpace(template.TemplateRecover)
	if template.NoticeType == "" {
		template.NoticeType = "FeiShu"
	}
}

func validateNoticeTemplate(template *model.NoticeTemplate) error {
	if template.Name == "" {
		return fmt.Errorf("请输入通知模板名称")
	}
	if template.NoticeType == "" {
		return fmt.Errorf("请选择通知模板类型")
	}
	if !isSupportedNoticeType(template.NoticeType) {
		return fmt.Errorf("暂不支持的通知模板类型：%s", template.NoticeType)
	}
	if template.TemplateFiring == "" {
		return fmt.Errorf("请填写触发模板")
	}
	if template.TemplateRecover == "" {
		return fmt.Errorf("请填写恢复模板")
	}
	return nil
}

func normalizeNoticeObject(object *model.NoticeObject) {
	object.Name = strings.TrimSpace(object.Name)
	object.UUID = strings.TrimSpace(object.UUID)
	object.Description = strings.TrimSpace(object.Description)
	object.Routes = normalizeJSONText(object.Routes, "[]")
	object.LastStatus = strings.TrimSpace(object.LastStatus)
	if object.LastStatus == "" {
		object.LastStatus = "ready"
	}
}

func validateNoticeObject(db *gorm.DB, object *model.NoticeObject) error {
	if object.Name == "" {
		return fmt.Errorf("请输入通知对象名称")
	}
	if !json.Valid([]byte(object.Routes)) {
		return fmt.Errorf("通知路由必须是合法 JSON")
	}
	var routes []noticeObjectRouteConfig
	if err := json.Unmarshal([]byte(object.Routes), &routes); err != nil {
		return fmt.Errorf("通知路由必须是合法 JSON")
	}
	if len(routes) == 0 {
		return fmt.Errorf("请至少配置一条通知路由")
	}
	for index, route := range routes {
		noticeType := normalizeNoticeType(route.NoticeType)
		if !isSupportedNoticeType(noticeType) {
			return fmt.Errorf("暂不支持的通知类型：%s", noticeType)
		}
		templateID := noticeTemplateID(route.NoticeTemplateID)
		if templateID == 0 {
			return fmt.Errorf("通知策略 %d 请选择通知模板", index+1)
		}
		var template model.NoticeTemplate
		if err := db.First(&template, templateID).Error; err != nil {
			return fmt.Errorf("通知策略 %d 的通知模板不存在", index+1)
		}
		if normalizeNoticeType(template.NoticeType) != noticeType {
			return fmt.Errorf("通知策略 %d 的通知模板类型与通知类型不一致", index+1)
		}
		if len(route.Severitys) == 0 {
			return fmt.Errorf("通知策略 %d 请选择告警级别", index+1)
		}
		if noticeType == "Email" {
			if strings.TrimSpace(route.Subject) == "" {
				return fmt.Errorf("通知策略 %d 请填写邮件主题", index+1)
			}
			if !noticeRouteHasSMTPConfig(db, route) {
				return fmt.Errorf("通知策略 %d 请填写 SMTP 服务器和发件邮箱", index+1)
			}
			if strings.TrimSpace(route.SMTPUser) != "" && strings.TrimSpace(route.SMTPPassword) == "" {
				return fmt.Errorf("通知策略 %d 请填写 SMTP 密码", index+1)
			}
			if len(normalizeEmailRecipients(route.To)) == 0 && object.DutyTableID == 0 {
				return fmt.Errorf("通知策略 %d 请填写固定收件人或选择值班表", index+1)
			}
		} else if strings.TrimSpace(route.Hook) == "" {
			return fmt.Errorf("通知策略 %d 请填写 Hook 地址", index+1)
		}
		start := effectiveClockText(route.EffectiveTime.StartTime)
		end := effectiveClockText(route.EffectiveTime.EndTime)
		if (start == "") != (end == "") {
			return fmt.Errorf("通知策略 %d 请同时填写生效开始时间和结束时间", index+1)
		}
	}
	return nil
}

func noticeRouteHasSMTPConfig(db *gorm.DB, route noticeObjectRouteConfig) bool {
	if strings.TrimSpace(route.SMTPHost) != "" && strings.TrimSpace(firstNonEmpty(route.FromEmail, route.SMTPUser)) != "" {
		return true
	}
	var count int64
	if err := db.Model(&model.AlertChannel{}).Where("enabled = ? AND channel_type = ?", true, "email").Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func normalizeDutyTable(table *model.DutyTable) {
	table.Name = strings.TrimSpace(table.Name)
	table.Description = strings.TrimSpace(table.Description)
	table.ManagerUsername = strings.TrimSpace(table.ManagerUsername)
}

func validateDutyTable(table *model.DutyTable) error {
	if table.Name == "" {
		return fmt.Errorf("请输入值班表名称")
	}
	return nil
}

func normalizeDutySchedule(schedule *model.DutySchedule) {
	schedule.DutyDate = strings.TrimSpace(schedule.DutyDate)
	schedule.Users = normalizeJSONText(schedule.Users, "[]")
	schedule.Status = strings.TrimSpace(schedule.Status)
	if schedule.DutyDate == "" {
		schedule.DutyDate = time.Now().Format("2006-01-02")
	}
	if schedule.Status == "" {
		schedule.Status = "active"
	}
}

func validateDutySchedule(schedule *model.DutySchedule) error {
	if schedule.DutyTableID == 0 {
		return fmt.Errorf("请选择值班表")
	}
	if _, err := time.Parse("2006-01-02", schedule.DutyDate); err != nil {
		return fmt.Errorf("值班日期格式必须是 YYYY-MM-DD")
	}
	if !json.Valid([]byte(schedule.Users)) {
		return fmt.Errorf("值班人员必须是合法 JSON")
	}
	return nil
}

func normalizeNoticeType(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	mapping := map[string]string{
		"feishu":   "FeiShu",
		"lark":     "FeiShu",
		"dingtalk": "DingDing",
		"dingding": "DingDing",
		"wechat":   "WeChat",
		"wecom":    "WeChat",
		"email":    "Email",
		"mail":     "Email",
		"slack":    "Slack",
		"webhook":  "WebHook",
		"phone":    "Phone",
		"sms":      "SMS",
		"sreflow":  "SREFlow",
	}
	if normalized, ok := mapping[value]; ok {
		return normalized
	}
	return strings.TrimSpace(raw)
}

func isSupportedNoticeType(noticeType string) bool {
	switch normalizeNoticeType(noticeType) {
	case "FeiShu", "Email", "DingDing", "WeChat", "WebHook":
		return true
	default:
		return false
	}
}

func (h *DataSourceHandler) fillNoticeObjectRuntime(object *model.NoticeObject) {
	if object.DutyTableID == 0 {
		return
	}
	var table model.DutyTable
	if err := h.db.First(&table, object.DutyTableID).Error; err != nil {
		return
	}
	h.fillDutyTableRuntime(&table)
	object.DutyTableName = table.Name
	object.CurrentDutyUsers = table.CurrentDutyUsers
}

func (h *DataSourceHandler) fillDutyTableRuntime(table *model.DutyTable) {
	var schedule model.DutySchedule
	today := time.Now().Format("2006-01-02")
	if err := h.db.Where("duty_table_id = ? AND duty_date = ? AND status = ?", table.ID, today, "active").First(&schedule).Error; err != nil {
		table.CurrentDutyUsers = []model.DutyUser{}
		return
	}
	table.CurrentDutyUsers = h.enrichDutyUsersWithSystemProfiles(parseDutyUsers(schedule.Users))
}

func parseDutyUsers(raw string) []model.DutyUser {
	var users []model.DutyUser
	if err := json.Unmarshal([]byte(raw), &users); err != nil {
		return []model.DutyUser{}
	}
	if users == nil {
		return []model.DutyUser{}
	}
	return users
}

func (h *DataSourceHandler) enrichDutyUsersWithSystemProfiles(users []model.DutyUser) []model.DutyUser {
	if len(users) == 0 {
		return users
	}
	ids := make([]uint, 0, len(users))
	usernames := make([]string, 0, len(users))
	for _, user := range users {
		if user.ID > 0 {
			ids = append(ids, user.ID)
		}
		if username := strings.TrimSpace(user.Username); username != "" {
			usernames = append(usernames, username)
		}
	}
	profiles := make([]model.DutyUser, 0)
	query := h.db.Table("sys_user").
		Select("id, username, real_name, email, phone, notify_user_id, feishu_user_id, feishu_open_id, dingtalk_user_id, wecom_user_id")
	switch {
	case len(ids) > 0 && len(usernames) > 0:
		query = query.Where("id IN ? OR username IN ?", uniqueUintIDs(ids), uniqueStrings(usernames))
	case len(ids) > 0:
		query = query.Where("id IN ?", uniqueUintIDs(ids))
	case len(usernames) > 0:
		query = query.Where("username IN ?", uniqueStrings(usernames))
	default:
		return users
	}
	if err := query.Find(&profiles).Error; err != nil {
		return users
	}
	byID := map[uint]model.DutyUser{}
	byUsername := map[string]model.DutyUser{}
	for _, profile := range profiles {
		normalizeDutyUserIdentifier(&profile)
		if profile.ID > 0 {
			byID[profile.ID] = profile
		}
		if username := strings.TrimSpace(profile.Username); username != "" {
			byUsername[username] = profile
		}
	}
	for i := range users {
		profile, ok := byID[users[i].ID]
		if !ok {
			profile, ok = byUsername[strings.TrimSpace(users[i].Username)]
		}
		if !ok {
			normalizeDutyUserIdentifier(&users[i])
			continue
		}
		mergeDutyUserProfile(&users[i], profile)
	}
	return users
}

func mergeDutyUserProfile(user *model.DutyUser, profile model.DutyUser) {
	if profile.ID > 0 {
		user.ID = profile.ID
	}
	user.Username = firstNonEmpty(user.Username, profile.Username)
	user.RealName = firstNonEmpty(user.RealName, profile.RealName)
	user.Email = firstNonEmpty(user.Email, profile.Email)
	user.Phone = firstNonEmpty(user.Phone, profile.Phone)
	user.NotifyUserID = firstNonEmpty(profile.NotifyUserID, user.NotifyUserID)
	user.FeishuUserID = firstNonEmpty(profile.FeishuUserID, user.FeishuUserID)
	user.FeishuOpenID = firstNonEmpty(profile.FeishuOpenID, user.FeishuOpenID)
	user.DingTalkUserID = firstNonEmpty(profile.DingTalkUserID, user.DingTalkUserID)
	user.WeComUserID = firstNonEmpty(profile.WeComUserID, user.WeComUserID)
	normalizeDutyUserIdentifier(user)
}

func normalizeDutyUserIdentifier(user *model.DutyUser) {
	user.NotifyUserID = firstNonEmpty(user.NotifyUserID, user.FeishuOpenID, user.FeishuUserID, user.DingTalkUserID, user.WeComUserID)
	if user.NotifyUserID == "" {
		return
	}
	user.FeishuUserID = firstNonEmpty(user.FeishuUserID, user.NotifyUserID)
	user.FeishuOpenID = firstNonEmpty(user.FeishuOpenID, user.NotifyUserID)
	user.DingTalkUserID = firstNonEmpty(user.DingTalkUserID, user.NotifyUserID)
	user.WeComUserID = firstNonEmpty(user.WeComUserID, user.NotifyUserID)
}

func jsonArrayContainsID(raw string, id string) bool {
	var ids []interface{}
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return false
	}
	for _, item := range ids {
		switch value := item.(type) {
		case float64:
			if fmt.Sprintf("%.0f", value) == id {
				return true
			}
		case string:
			if value == id {
				return true
			}
		}
	}
	return false
}

func generateMonitorUUID(prefix string) string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(buf)
}
