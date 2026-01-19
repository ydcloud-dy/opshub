package server

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/plugins/monitor/model"
	"github.com/ydcloud-dy/opshub/plugins/monitor/repository"
	"github.com/ydcloud-dy/opshub/plugins/monitor/service"
	"gorm.io/gorm"
)

// Handler 域名监控处理器
type Handler struct {
	db           *gorm.DB
	repo         *repository.DomainMonitorRepository
	alertService *service.AlertService
}

// NewHandler 创建处理器
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{
		db:           db,
		repo:         repository.NewDomainMonitorRepository(db),
		alertService: service.NewAlertService(),
	}
}

// ListDomains 获取域名监控列表
// @Summary 获取域名监控列表
// @Tags DomainMonitor
// @Accept json
// @Produce json
// @Success 200 {array} model.DomainMonitor
// @Router /monitor/domains [get]
func (h *Handler) ListDomains(c *gin.Context) {
	monitors, err := h.repo.GetAll()
	if err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "获取域名列表失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data":    monitors,
	})
}

// GetDomain 获取域名监控详情
// @Summary 获取域名监控详情
// @Tags DomainMonitor
// @Accept json
// @Produce json
// @Param id path int true "域名ID"
// @Success 200 {object} model.DomainMonitor
// @Router /monitor/domains/{id} [get]
func (h *Handler) GetDomain(c *gin.Context) {
	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的ID",
		})
		return
	}

	monitor, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(404, gin.H{
			"code":    404,
			"message": "域名监控不存在",
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data":    monitor,
	})
}

// CreateDomain 创建域名监控
// @Summary 创建域名监控
// @Tags DomainMonitor
// @Accept json
// @Produce json
// @Param monitor body model.DomainMonitor true "域名监控信息"
// @Success 200 {object} model.DomainMonitor
// @Router /monitor/domains [post]
func (h *Handler) CreateDomain(c *gin.Context) {
	var req model.DomainMonitor
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 检查域名是否已存在
	existing, err := h.repo.GetByDomain(req.Domain)
	if err == nil && existing != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "该域名已在监控中",
		})
		return
	}

	// 设置初始状态
	req.Status = "unknown"
	req.ResponseTime = 0
	now := time.Now()
	req.LastCheck = &now

	// 创建监控记录
	if err := h.repo.Create(&req); err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "创建域名监控失败",
			"error":   err.Error(),
		})
		return
	}

	// 创建成功后立即执行一次域名检查
	result, err := h.performCheck(&req)
	if err != nil {
		// 即使检查失败，记录也已创建，只是保持unknown状态
		// 可以选择记录日志或直接忽略
	} else {
		// 更新检查结果
		req.Status = result.Status
		req.ResponseTime = result.ResponseTime
		req.SSLValid = result.SSLValid
		req.SSLExpiry = result.SSLExpiry
		checkTime := time.Now()
		req.LastCheck = &checkTime

		// 计算下次检查时间
		nextCheck := checkTime.Add(time.Duration(req.CheckInterval) * time.Second)
		req.NextCheck = &nextCheck

		// 保存更新后的状态
		h.repo.Update(&req)
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "创建成功",
		"data":    req,
	})
}

// UpdateDomain 更新域名监控
// @Summary 更新域名监控
// @Tags DomainMonitor
// @Accept json
// @Produce json
// @Param id path int true "域名ID"
// @Param monitor body model.DomainMonitor true "域名监控信息"
// @Success 200 {object} model.DomainMonitor
// @Router /monitor/domains/{id} [put]
func (h *Handler) UpdateDomain(c *gin.Context) {
	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的ID",
		})
		return
	}

	// 获取现有记录
	monitor, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(404, gin.H{
			"code":    404,
			"message": "域名监控不存在",
		})
		return
	}

	var req model.DomainMonitor
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 更新字段
	monitor.CheckInterval = req.CheckInterval
	monitor.EnableSSL = req.EnableSSL
	monitor.EnableAlert = req.EnableAlert

	// 如果域名改变，检查新域名是否已存在
	if req.Domain != monitor.Domain {
		existing, err := h.repo.GetByDomain(req.Domain)
		if err == nil && existing != nil && existing.ID != id {
			c.JSON(400, gin.H{
				"code":    400,
				"message": "该域名已在监控中",
			})
			return
		}
		monitor.Domain = req.Domain
	}

	// 保存更新
	if err := h.repo.Update(monitor); err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "更新域名监控失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "更新成功",
		"data":    monitor,
	})
}

// DeleteDomain 删除域名监控
// @Summary 删除域名监控
// @Tags DomainMonitor
// @Accept json
// @Produce json
// @Param id path int true "域名ID"
// @Success 200 {string} string "success"
// @Router /monitor/domains/{id} [delete]
func (h *Handler) DeleteDomain(c *gin.Context) {
	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的ID",
		})
		return
	}

	if err := h.repo.Delete(id); err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "删除域名监控失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// CheckDomain 立即检查域名
// @Summary 立即检查域名
// @Tags DomainMonitor
// @Accept json
// @Produce json
// @Param id path int true "域名ID"
// @Success 200 {object} model.DomainMonitor
// @Router /monitor/domains/{id}/check [post]
func (h *Handler) CheckDomain(c *gin.Context) {
	var id uint
	if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
		c.JSON(400, gin.H{
			"code":    400,
			"message": "无效的ID",
		})
		return
	}

	// 获取监控记录
	monitor, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(404, gin.H{
			"code":    404,
			"message": "域名监控不存在",
		})
		return
	}

	// 执行域名检查
	result, err := h.performCheck(monitor)
	if err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "域名检查失败",
			"error":   err.Error(),
		})
		return
	}

	// 更新监控记录
	monitor.Status = result.Status
	monitor.ResponseTime = result.ResponseTime
	monitor.SSLValid = result.SSLValid
	monitor.SSLExpiry = result.SSLExpiry
	now := time.Now()
	monitor.LastCheck = &now

	// 计算下次检查时间
	nextCheck := now.Add(time.Duration(monitor.CheckInterval) * time.Second)
	monitor.NextCheck = &nextCheck

	if err := h.repo.Update(monitor); err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "更新域名监控失败",
			"error":   err.Error(),
		})
		return
	}

	// 检查并发送告警
	go h.checkAndSendAlert(monitor, result)

	c.JSON(200, gin.H{
		"code":    0,
		"message": "检查完成",
		"data":    monitor,
	})
}

// GetStats 获取统计数据
// @Summary 获取域名监控统计数据
// @Tags DomainMonitor
// @Accept json
// @Produce json
// @Success 200 {object} map[string]int64
// @Router /monitor/domains/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.repo.GetStats()
	if err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "获取统计数据失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data":    stats,
	})
}

// CheckResult 检查结果
type CheckResult struct {
	Status       string      `json:"status"`        // normal, abnormal
	ResponseTime int         `json:"responseTime"`  // 响应时间(ms)
	SSLValid     bool        `json:"sslValid"`      // SSL是否有效
	SSLExpiry    *time.Time  `json:"sslExpiry"`     // SSL过期时间
}

// performCheck 执行域名检查
func (h *Handler) performCheck(monitor *model.DomainMonitor) (*CheckResult, error) {
	result := &CheckResult{
		Status:       "abnormal",
		ResponseTime: 0,
		SSLValid:     false,
		SSLExpiry:    nil,
	}

	// 构造目标URL
	url := fmt.Sprintf("https://%s", monitor.Domain)

	// 记录开始时间
	start := time.Now()

	// 创建HTTP客户端，跳过SSL验证(用于测试连接)
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // 跳过证书验证，只测试连接
			},
		},
	}

	// 发送请求
	resp, err := client.Get(url)
	if err != nil {
		// 检查是否是网络错误
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			result.ResponseTime = int(10 * time.Second / time.Millisecond)
		}
		return result, nil
	}
	defer resp.Body.Close()

	// 计算响应时间
	result.ResponseTime = int(time.Since(start).Milliseconds())

	// 检查响应状态码
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Status = "normal"
	} else {
		result.Status = "abnormal"
	}

	// 如果启用SSL检查，验证SSL证书
	if monitor.EnableSSL {
		// 重新发送请求获取证书信息
		conn, err := tls.Dial("tcp", fmt.Sprintf("%s:443", monitor.Domain), &tls.Config{
			InsecureSkipVerify: false,
		})
		if err != nil {
			result.SSLValid = false
		} else {
			state := conn.ConnectionState()
			if len(state.PeerCertificates) > 0 {
				cert := state.PeerCertificates[0]
				result.SSLValid = true
				result.SSLExpiry = &cert.NotAfter

				// 检查证书是否即将过期(30天内)
				if time.Until(cert.NotAfter) < 30*24*time.Hour {
					result.SSLValid = false
				}
			}
			conn.Close()
		}
	}

	return result, nil
}

// checkAndSendAlert 检查并发送告警
func (h *Handler) checkAndSendAlert(monitor *model.DomainMonitor, result *CheckResult) {
	// 如果未启用告警，直接返回
	if !monitor.EnableAlert {
		return
	}

	// 检查告警频率控制
	if h.shouldSuppressAlert(monitor.ID, result.Status) {
		return
	}

	// 检查各种告警条件
	alerts := h.checkAlertConditions(monitor, result)

	// 如果有告警，发送通知
	if len(alerts) > 0 {
		for _, alert := range alerts {
			h.sendAlert(monitor, alert)
		}
	}
}

// checkAlertConditions 检查告警条件
func (h *Handler) checkAlertConditions(monitor *model.DomainMonitor, result *CheckResult) []service.AlertMessage {
	var alerts []service.AlertMessage
	now := time.Now()

	// 1. 域名无法访问告警
	if result.Status == "abnormal" && result.ResponseTime == 0 {
		alerts = append(alerts, service.AlertMessage{
			AlertType: "domain_down",
			Domain:    monitor.Domain,
			Status:    result.Status,
			Message:   "域名无法访问或响应超时",
			Timestamp: now.Format("2006-01-02 15:04:05"),
		})
	}

	// 2. 响应时间过高告警
	if monitor.ResponseThreshold != nil && result.ResponseTime > *monitor.ResponseThreshold {
		alerts = append(alerts, service.AlertMessage{
			AlertType:    "high_response_time",
			Domain:       monitor.Domain,
			Status:       result.Status,
			Message:      fmt.Sprintf("响应时间 %dms 超过阈值 %dms", result.ResponseTime, *monitor.ResponseThreshold),
			ResponseTime: result.ResponseTime,
			Timestamp:    now.Format("2006-01-02 15:04:05"),
		})
	}

	// 3. SSL证书即将过期告警
	if monitor.EnableSSL && result.SSLExpiry != nil {
		daysUntilExpiry := int(time.Until(*result.SSLExpiry).Hours() / 24)
		if monitor.SSLExpiryDays != nil && daysUntilExpiry <= *monitor.SSLExpiryDays && daysUntilExpiry > 0 {
			alerts = append(alerts, service.AlertMessage{
				AlertType: "ssl_expiring",
				Domain:    monitor.Domain,
				Status:    result.Status,
				Message:   fmt.Sprintf("SSL证书将在 %d 天后过期", daysUntilExpiry),
				SSLExpiry: result.SSLExpiry.Format("2006-01-02 15:04:05"),
				Timestamp: now.Format("2006-01-02 15:04:05"),
			})
		}
	}

	// 4. SSL证书已过期告警
	if monitor.EnableSSL && result.SSLExpiry != nil && result.SSLExpiry.Before(now) {
		alerts = append(alerts, service.AlertMessage{
			AlertType: "ssl_expired",
			Domain:    monitor.Domain,
			Status:    result.Status,
			Message:   "SSL证书已过期",
			SSLExpiry: result.SSLExpiry.Format("2006-01-02 15:04:05"),
			Timestamp: now.Format("2006-01-02 15:04:05"),
		})
	}

	// 5. SSL证书无效告警
	if monitor.EnableSSL && !result.SSLValid {
		alerts = append(alerts, service.AlertMessage{
			AlertType: "ssl_invalid",
			Domain:    monitor.Domain,
			Status:    result.Status,
			Message:   "SSL证书无效或无法验证",
			Timestamp: now.Format("2006-01-02 15:04:05"),
		})
	}

	return alerts
}

// sendAlert 发送告警
func (h *Handler) sendAlert(monitor *model.DomainMonitor, alert service.AlertMessage) {
	// TODO: 从数据库获取告警通道配置和接收人
	// 目前使用模拟数据

	// 记录告警日志
	h.logAlert(monitor.ID, alert, "pending", "", "")

	// 发送告警（这里暂时注释掉，等配置功能完成后再启用）
	// config := service.AlertChannelConfig{
	// 	// 从数据库或配置文件读取
	// }
	// receivers := []string{"admin@example.com"}
	// err := h.alertService.SendAlert(alert, config, receivers)
	// if err != nil {
	// 	h.logAlert(monitor.ID, alert, "failed", "", err.Error())
	// } else {
	// 	h.logAlert(monitor.ID, alert, "success", "", "")
	// }
}

// logAlert 记录告警日志
func (h *Handler) logAlert(domainMonitorID uint, alert service.AlertMessage, status, channel, errorMsg string) {
	alertLog := &model.AlertLog{
		DomainMonitorID: domainMonitorID,
		Domain:          alert.Domain,
		AlertType:       alert.AlertType,
		Status:          status,
		Message:         alert.Message,
		ChannelType:     channel,
		ErrorMsg:        errorMsg,
		SentAt:          time.Now(),
	}

	h.db.Create(alertLog)
}

// shouldSuppressAlert 检查是否应该抑制告警（告警频率控制）
func (h *Handler) shouldSuppressAlert(domainMonitorID uint, status string) bool {
	// 查询最近的告警日志
	var recentAlertCount int64
	h.db.Model(&model.AlertLog{}).
		Where("domain_monitor_id = ? AND alert_type = ? AND status = ? AND sent_at > ?",
			domainMonitorID, "domain_down", status, time.Now().Add(-10*time.Minute)).
		Count(&recentAlertCount)

	return recentAlertCount > 0
}
