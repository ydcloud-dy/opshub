// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package server

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/plugins/monitor/model"
	"gorm.io/gorm"
)

// AlertHandler 告警配置处理器
type AlertHandler struct {
	db *gorm.DB
}

// NewAlertHandler 创建告警配置处理器
func NewAlertHandler(db *gorm.DB) *AlertHandler {
	return &AlertHandler{db: db}
}

// ========== 告警通道配置管理 ==========

// ListAlertChannels 获取告警通道列表
// @Summary 获取告警通道列表
// @Tags AlertChannel
// @Accept json
// @Produce json
// @Success 200 {array} model.AlertChannel
// @Router /monitor/alerts/channels [get]
func (h *AlertHandler) ListAlertChannels(c *gin.Context) {
	var channels []model.AlertChannel
	if err := h.db.Find(&channels).Error; err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "获取告警通道列表失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data":    channels,
	})
}

// GetAlertChannel 获取告警通道详情
// @Summary 获取告警通道详情
// @Tags AlertChannel
// @Accept json
// @Produce json
// @Param id path int true "通道ID"
// @Success 200 {object} model.AlertChannel
// @Router /monitor/alerts/channels/{id} [get]
func (h *AlertHandler) GetAlertChannel(c *gin.Context) {
	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的ID",
		})
		return
	}

	var channel model.AlertChannel
	if err := h.db.First(&channel, id).Error; err != nil {
		c.JSON(404, gin.H{
			"code":    404,
			"message": "告警通道不存在",
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data":    channel,
	})
}

// CreateAlertChannel 创建告警通道
// @Summary 创建告警通道
// @Tags AlertChannel
// @Accept json
// @Produce json
// @Param channel body model.AlertChannel true "告警通道信息"
// @Success 200 {object} model.AlertChannel
// @Router /monitor/alerts/channels [post]
func (h *AlertHandler) CreateAlertChannel(c *gin.Context) {
	var req model.AlertChannel
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "创建告警通道失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "创建成功",
		"data":    req,
	})
}

// UpdateAlertChannel 更新告警通道
// @Summary 更新告警通道
// @Tags AlertChannel
// @Accept json
// @Produce json
// @Param id path int true "通道ID"
// @Param channel body model.AlertChannel true "告警通道信息"
// @Success 200 {object} model.AlertChannel
// @Router /monitor/alerts/channels/{id} [put]
func (h *AlertHandler) UpdateAlertChannel(c *gin.Context) {
	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的ID",
		})
		return
	}

	// 获取现有记录
	var channel model.AlertChannel
	if err := h.db.First(&channel, id).Error; err != nil {
		c.JSON(404, gin.H{
			"code":    404,
			"message": "告警通道不存在",
		})
		return
	}

	var req model.AlertChannel
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 更新字段
	channel.Name = req.Name
	channel.ChannelType = req.ChannelType
	channel.Enabled = req.Enabled
	channel.Config = req.Config

	// 保存更新
	if err := h.db.Save(&channel).Error; err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "更新告警通道失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "更新成功",
		"data":    channel,
	})
}

// DeleteAlertChannel 删除告警通道
// @Summary 删除告警通道
// @Tags AlertChannel
// @Accept json
// @Produce json
// @Param id path int true "通道ID"
// @Success 200 {string} string "success"
// @Router /monitor/alerts/channels/{id} [delete]
func (h *AlertHandler) DeleteAlertChannel(c *gin.Context) {
	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的ID",
		})
		return
	}

	if err := h.db.Delete(&model.AlertChannel{}, id).Error; err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "删除告警通道失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// ========== 告警接收人管理 ==========

// ListAlertReceivers 获取告警接收人列表
// @Summary 获取告警接收人列表
// @Tags AlertReceiver
// @Accept json
// @Produce json
// @Success 200 {array} model.AlertReceiver
// @Router /monitor/alerts/receivers [get]
func (h *AlertHandler) ListAlertReceivers(c *gin.Context) {
	var receivers []model.AlertReceiver
	if err := h.db.Find(&receivers).Error; err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "获取告警接收人列表失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data":    receivers,
	})
}

// GetAlertReceiver 获取告警接收人详情
// @Summary 获取告警接收人详情
// @Tags AlertReceiver
// @Accept json
// @Produce json
// @Param id path int true "接收人ID"
// @Success 200 {object} model.AlertReceiver
// @Router /monitor/alerts/receivers/{id} [get]
func (h *AlertHandler) GetAlertReceiver(c *gin.Context) {
	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的ID",
		})
		return
	}

	var receiver model.AlertReceiver
	if err := h.db.First(&receiver, id).Error; err != nil {
		c.JSON(404, gin.H{
			"code":    404,
			"message": "告警接收人不存在",
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data":    receiver,
	})
}

// CreateAlertReceiver 创建告警接收人
// @Summary 创建告警接收人
// @Tags AlertReceiver
// @Accept json
// @Produce json
// @Param receiver body model.AlertReceiver true "告警接收人信息"
// @Success 200 {object} model.AlertReceiver
// @Router /monitor/alerts/receivers [post]
func (h *AlertHandler) CreateAlertReceiver(c *gin.Context) {
	var req model.AlertReceiver
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "创建告警接收人失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "创建成功",
		"data":    req,
	})
}

// UpdateAlertReceiver 更新告警接收人
// @Summary 更新告警接收人
// @Tags AlertReceiver
// @Accept json
// @Produce json
// @Param id path int true "接收人ID"
// @Param receiver body model.AlertReceiver true "告警接收人信息"
// @Success 200 {object} model.AlertReceiver
// @Router /monitor/alerts/receivers/{id} [put]
func (h *AlertHandler) UpdateAlertReceiver(c *gin.Context) {
	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的ID",
		})
		return
	}

	// 获取现有记录
	var receiver model.AlertReceiver
	if err := h.db.First(&receiver, id).Error; err != nil {
		c.JSON(404, gin.H{
			"code":    404,
			"message": "告警接收人不存在",
		})
		return
	}

	var req model.AlertReceiver
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 更新字段
	receiver.Name = req.Name
	receiver.Email = req.Email
	receiver.Phone = req.Phone
	receiver.WeChatID = req.WeChatID
	receiver.DingTalkID = req.DingTalkID
	receiver.FeishuID = req.FeishuID
	receiver.UserID = req.UserID
	receiver.EnableEmail = req.EnableEmail
	receiver.EnableWebhook = req.EnableWebhook
	receiver.EnableWeChat = req.EnableWeChat
	receiver.EnableDingTalk = req.EnableDingTalk
	receiver.EnableFeishu = req.EnableFeishu
	receiver.EnableSystemMsg = req.EnableSystemMsg

	// 保存更新
	if err := h.db.Save(&receiver).Error; err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "更新告警接收人失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "更新成功",
		"data":    receiver,
	})
}

// DeleteAlertReceiver 删除告警接收人
// @Summary 删除告警接收人
// @Tags AlertReceiver
// @Accept json
// @Produce json
// @Param id path int true "接收人ID"
// @Success 200 {string} string "success"
// @Router /monitor/alerts/receivers/{id} [delete]
func (h *AlertHandler) DeleteAlertReceiver(c *gin.Context) {
	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的ID",
		})
		return
	}

	if err := h.db.Delete(&model.AlertReceiver{}, id).Error; err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "删除告警接收人失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// ========== 告警接收人与通道关联管理 ==========

// ListReceiverChannels 获取接收人的通道关联列表
// @Summary 获取接收人的通道关联列表
// @Tags AlertReceiverChannel
// @Accept json
// @Produce json
// @Param receiverId path int true "接收人ID"
// @Success 200 {array} model.AlertReceiverChannel
// @Router /monitor/alerts/receiver-channels/:receiverId [get]
func (h *AlertHandler) ListReceiverChannels(c *gin.Context) {
	var receiverID uint
	if _, err := fmt.Sscanf(c.Param("receiverId"), "%d", &receiverID); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的接收人ID",
		})
		return
	}

	var channels []model.AlertReceiverChannel
	if err := h.db.Where("receiver_id = ?", receiverID).Find(&channels).Error; err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "获取通道关联失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data":    channels,
	})
}

// AddReceiverChannel 添加接收人通道关联
// @Summary 添加接收人通道关联
// @Tags AlertReceiverChannel
// @Accept json
// @Produce json
// @Param receiverId path int true "接收人ID"
// @Param req body model.AlertReceiverChannel true "关联信息"
// @Success 200 {object} model.AlertReceiverChannel
// @Router /monitor/alerts/receiver-channels/:receiverId [post]
func (h *AlertHandler) AddReceiverChannel(c *gin.Context) {
	var receiverID uint
	if _, err := fmt.Sscanf(c.Param("receiverId"), "%d", &receiverID); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的接收人ID",
		})
		return
	}

	var req struct {
		ChannelID uint   `json:"channelId" binding:"required"`
		Config    string `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 检查接收人和通道是否存在
	var receiver model.AlertReceiver
	if err := h.db.First(&receiver, receiverID).Error; err != nil {
		c.JSON(404, gin.H{
			"code":    404,
			"message": "接收人不存在",
		})
		return
	}

	var channel model.AlertChannel
	if err := h.db.First(&channel, req.ChannelID).Error; err != nil {
		c.JSON(404, gin.H{
			"code":    404,
			"message": "通道不存在",
		})
		return
	}

	// 检查关联是否已存在
	var existing model.AlertReceiverChannel
	result := h.db.Where("receiver_id = ? AND channel_id = ?", receiverID, req.ChannelID).First(&existing)
	if result.RowsAffected > 0 {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "该关联已存在",
		})
		return
	}

	// 创建新关联
	rc := model.AlertReceiverChannel{
		ReceiverID: receiverID,
		ChannelID:  req.ChannelID,
		Config:     req.Config,
	}

	if err := h.db.Create(&rc).Error; err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "添加通道关联失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "添加成功",
		"data":    rc,
	})
}

// RemoveReceiverChannel 删除接收人通道关联
// @Summary 删除接收人通道关联
// @Tags AlertReceiverChannel
// @Accept json
// @Produce json
// @Param receiverId path int true "接收人ID"
// @Param channelId path int true "通道ID"
// @Success 200 {string} string "success"
// @Router /monitor/alerts/receiver-channels/:receiverId/:channelId [delete]
func (h *AlertHandler) RemoveReceiverChannel(c *gin.Context) {
	var receiverID, channelID uint
	if _, err := fmt.Sscanf(c.Param("receiverId"), "%d", &receiverID); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的接收人ID",
		})
		return
	}
	if _, err := fmt.Sscanf(c.Param("channelId"), "%d", &channelID); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的通道ID",
		})
		return
	}

	if err := h.db.Where("receiver_id = ? AND channel_id = ?", receiverID, channelID).Delete(&model.AlertReceiverChannel{}).Error; err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "删除通道关联失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// UpdateReceiverChannelConfig 更新接收人通道关联配置
// @Summary 更新接收人通道关联配置
// @Tags AlertReceiverChannel
// @Accept json
// @Produce json
// @Param receiverId path int true "接收人ID"
// @Param channelId path int true "通道ID"
// @Param req body map[string]string true "配置"
// @Success 200 {object} model.AlertReceiverChannel
// @Router /monitor/alerts/receiver-channels/:receiverId/:channelId [put]
func (h *AlertHandler) UpdateReceiverChannelConfig(c *gin.Context) {
	var receiverID, channelID uint
	if _, err := fmt.Sscanf(c.Param("receiverId"), "%d", &receiverID); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的接收人ID",
		})
		return
	}
	if _, err := fmt.Sscanf(c.Param("channelId"), "%d", &channelID); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的通道ID",
		})
		return
	}

	var req struct {
		Config string `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	var rc model.AlertReceiverChannel
	if err := h.db.Where("receiver_id = ? AND channel_id = ?", receiverID, channelID).First(&rc).Error; err != nil {
		c.JSON(404, gin.H{
			"code":    404,
			"message": "通道关联不存在",
		})
		return
	}

	rc.Config = req.Config
	if err := h.db.Save(&rc).Error; err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "更新配置失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "更新成功",
		"data":    rc,
	})
}

// ========== 告警日志管理 ==========

// ListAlertLogs 获取告警日志列表
// @Summary 获取告警日志列表
// @Tags AlertLog
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Param domainMonitorId query int false "域名监控ID"
// @Param alertType query string false "告警类型"
// @Success 200 {array} model.AlertLog
// @Router /monitor/alerts/logs [get]
func (h *AlertHandler) ListAlertLogs(c *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 构建查询
	query := h.db.Model(&model.AlertLog{})

	// 过滤条件
	if domainMonitorID := c.Query("domainMonitorId"); domainMonitorID != "" {
		query = query.Where("domain_monitor_id = ?", domainMonitorID)
	}
	if alertType := c.Query("alertType"); alertType != "" {
		query = query.Where("alert_type = ?", alertType)
	}

	// 获取总数
	var total int64
	query.Count(&total)

	// 分页查询
	var logs []model.AlertLog
	offset := (page - 1) * pageSize
	if err := query.Order("sent_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "获取告警日志失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":     logs,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}

// GetAlertStats 获取告警统计信息
// @Summary 获取告警统计信息
// @Tags AlertLog
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /monitor/alerts/stats [get]
func (h *AlertHandler) GetAlertStats(c *gin.Context) {
	var stats struct {
		TotalAlerts int64 `json:"totalAlerts"`
		SuccessSent int64 `json:"successSent"`
		FailedSent  int64 `json:"failedSent"`
		TodayAlerts int64 `json:"todayAlerts"`
	}

	// 总告警数
	h.db.Model(&model.AlertLog{}).Count(&stats.TotalAlerts)

	// 发送成功数
	h.db.Model(&model.AlertLog{}).Where("status = ?", "success").Count(&stats.SuccessSent)

	// 发送失败数
	h.db.Model(&model.AlertLog{}).Where("status = ?", "failed").Count(&stats.FailedSent)

	// 今日告警数
	h.db.Model(&model.AlertLog{}).Where("DATE(sent_at) = CURDATE()").Count(&stats.TodayAlerts)

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data":    stats,
	})
}
