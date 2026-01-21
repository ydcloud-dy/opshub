package asset

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	"github.com/ydcloud-dy/opshub/pkg/response"
	"gorm.io/gorm"
)

// TerminalAuditHandler 终端审计处理器
type TerminalAuditHandler struct {
	db *gorm.DB
}

// NewTerminalAuditHandler 创建终端审计处理器
func NewTerminalAuditHandler(db *gorm.DB) *TerminalAuditHandler {
	return &TerminalAuditHandler{db: db}
}

// ListTerminalSessions 获取终端会话列表
func (h *TerminalAuditHandler) ListTerminalSessions(c *gin.Context) {
	// 获取分页参数
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("pageSize", "10")
	keyword := c.Query("keyword")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	// 构建查询
	query := h.db.Model(&assetbiz.TerminalSession{})

	// 搜索关键词
	if keyword != "" {
		query = query.Where("host_name LIKE ? OR host_ip LIKE ? OR username LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		return
	}

	// 分页查询
	var sessions []*assetbiz.TerminalSession
	if err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&sessions).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		return
	}

	// 转换为VO
	list := make([]*assetbiz.TerminalSessionInfo, 0, len(sessions))
	for _, session := range sessions {
		info := &assetbiz.TerminalSessionInfo{
			ID:            session.ID,
			HostID:        session.HostID,
			HostName:      session.HostName,
			HostIP:        session.HostIP,
			UserID:        session.UserID,
			Username:      session.Username,
			Duration:      session.Duration,
			DurationText:  formatDuration(session.Duration),
			FileSize:      session.FileSize,
			FileSizeText:  formatFileSize(session.FileSize),
			Status:        session.Status,
			StatusText:    getStatusText(session.Status),
			CreatedAt:     session.CreatedAt,
			CreatedAtText: session.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		list = append(list, info)
	}

	response.Success(c, gin.H{
		"total": total,
		"list":  list,
	})
}

// PlayTerminalSession 播放终端会话录制
func (h *TerminalAuditHandler) PlayTerminalSession(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的会话ID")
		return
	}

	// 查询会话
	var session assetbiz.TerminalSession
	if err := h.db.First(&session, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.ErrorCode(c, http.StatusNotFound, "会话不存在")
		} else {
			response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		}
		return
	}

	// 读取录制文件
	content, err := os.ReadFile(session.RecordingPath)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取录制文件失败")
		return
	}

	// 返回录制文件内容
	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, string(content))
}

// DeleteTerminalSession 删除终端会话
func (h *TerminalAuditHandler) DeleteTerminalSession(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的会话ID")
		return
	}

	// 查询会话
	var session assetbiz.TerminalSession
	if err := h.db.First(&session, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.ErrorCode(c, http.StatusNotFound, "会话不存在")
		} else {
			response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		}
		return
	}

	// 删除录制文件
	if err := os.Remove(session.RecordingPath); err != nil {
		fmt.Printf("删除录制文件失败: %v\n", err)
		// 即使文件删除失败，仍然继续删除数据库记录
	}

	// 删除数据库记录
	if err := h.db.Delete(&session).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败")
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// formatDuration 格式化时长
func formatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	} else if seconds < 3600 {
		minutes := seconds / 60
		secs := seconds % 60
		return fmt.Sprintf("%dm %ds", minutes, secs)
	} else {
		hours := seconds / 3600
		minutes := (seconds % 3600) / 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
}

// formatFileSize 格式化文件大小
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// getStatusText 获取状态文本
func getStatusText(status string) string {
	statusMap := map[string]string{
		"recording":  "录制中",
		"completed":  "已完成",
		"failed":     "失败",
	}
	if text, ok := statusMap[status]; ok {
		return text
	}
	return status
}
