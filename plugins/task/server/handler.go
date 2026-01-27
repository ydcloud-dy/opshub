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
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	"github.com/ydcloud-dy/opshub/pkg/response"
	"github.com/ydcloud-dy/opshub/plugins/task/model"
	"gorm.io/gorm"
)

type Handler struct {
	db            *gorm.DB
	encryptionKey []byte
}

func NewHandler(db *gorm.DB) *Handler {
	// 使用与凭证仓库相同的加密密钥
	encryptionKey := []byte("opshub-enc-key-32-bytes-long!!!!")
	return &Handler{
		db:            db,
		encryptionKey: encryptionKey,
	}
}

// ==================== 任务作业 ====================

// ListJobTasks 获取任务作业列表
// @Summary 获取任务作业列表
// @Description 分页获取任务作业列表，支持按关键词、任务类型、状态筛选
// @Tags 任务管理-任务作业
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键词"
// @Param taskType query string false "任务类型"
// @Param status query string false "任务状态"
// @Success 200 {object} response.Response "获取成功"
// @Router /task/job-tasks [get]
func (h *Handler) ListJobTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")
	taskType := c.Query("taskType")
	status := c.Query("status")

	var jobTasks []*model.JobTask
	var total int64

	query := h.db.Model(&model.JobTask{}).Where("deleted_at IS NULL")

	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	if taskType != "" {
		query = query.Where("task_type = ?", taskType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	query.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&jobTasks)

	response.Success(c, gin.H{
		"list":     jobTasks,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetJobTask 获取任务作业详情
// @Summary 获取任务作业详情
// @Description 获取指定任务作业的详细信息
// @Tags 任务管理-任务作业
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "任务ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 404 {object} response.Response "任务不存在"
// @Router /task/job-tasks/{id} [get]
func (h *Handler) GetJobTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var jobTask model.JobTask
	if err := h.db.Where("id = ? AND deleted_at IS NULL", id).First(&jobTask).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "任务不存在")
		return
	}
	response.Success(c, jobTask)
}

// CreateJobTask 创建任务作业
// @Summary 创建任务作业
// @Description 创建新的任务作业
// @Tags 任务管理-任务作业
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body model.JobTask true "任务信息"
// @Success 200 {object} response.Response "创建成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /task/job-tasks [post]
func (h *Handler) CreateJobTask(c *gin.Context) {
	var jobTask model.JobTask
	if err := c.ShouldBindJSON(&jobTask); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误")
		return
	}
	jobTask.Status = "pending"
	jobTask.CreatedBy = 1 // TODO: 从JWT获取
	if err := h.db.Create(&jobTask).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败")
		return
	}
	response.Success(c, jobTask)
}

// UpdateJobTask 更新任务作业
// @Summary 更新任务作业
// @Description 更新指定任务作业的信息
// @Tags 任务管理-任务作业
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "任务ID"
// @Param body body model.JobTask true "任务信息"
// @Success 200 {object} response.Response "更新成功"
// @Failure 404 {object} response.Response "任务不存在"
// @Router /task/job-tasks/{id} [put]
func (h *Handler) UpdateJobTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var jobTask model.JobTask
	if err := h.db.Where("id = ? AND deleted_at IS NULL", id).First(&jobTask).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "任务不存在")
		return
	}
	if err := c.ShouldBindJSON(&jobTask); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误")
		return
	}
	h.db.Save(&jobTask)
	response.Success(c, jobTask)
}

// DeleteJobTask 删除任务作业
// @Summary 删除任务作业
// @Description 删除指定的任务作业记录（已禁用）
// @Tags 任务管理-任务作业
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "任务ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 403 {object} response.Response "操作已禁用"
// @Router /task/job-tasks/{id} [delete]
func (h *Handler) DeleteJobTask(c *gin.Context) {
	response.ErrorCode(c, http.StatusForbidden, "删除任务记录功能已被禁用，如需删除请联系系统管理员")
}

// ==================== 任务模板 ====================

// ListJobTemplates 获取任务模板列表
// @Summary 获取任务模板列表
// @Description 分页获取任务模板列表，支持按关键词、分类筛选
// @Tags 任务管理-任务模板
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键词"
// @Param category query string false "模板分类"
// @Success 200 {object} response.Response "获取成功"
// @Router /task/job-templates [get]
func (h *Handler) ListJobTemplates(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")
	category := c.Query("category")

	var templates []*model.JobTemplate
	var total int64

	query := h.db.Model(&model.JobTemplate{}).Where("deleted_at IS NULL")

	if keyword != "" {
		query = query.Where("name LIKE ? OR code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	query.Order("sort ASC, created_at DESC").Limit(pageSize).Offset(offset).Find(&templates)

	response.Success(c, gin.H{
		"list":     templates,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetAllJobTemplates 获取所有启用的任务模板
// @Summary 获取所有任务模板
// @Description 获取所有启用状态的任务模板，用于下拉选择
// @Tags 任务管理-任务模板
// @Accept json
// @Produce json
// @Security Bearer
// @Param category query string false "模板分类"
// @Success 200 {object} response.Response "获取成功"
// @Router /task/job-templates/all [get]
func (h *Handler) GetAllJobTemplates(c *gin.Context) {
	category := c.Query("category")
	var templates []*model.JobTemplate
	query := h.db.Where("deleted_at IS NULL AND status = ?", 1)
	if category != "" {
		query = query.Where("category = ?", category)
	}
	query.Order("sort ASC").Find(&templates)
	response.Success(c, templates)
}

// GetJobTemplate 获取任务模板详情
// @Summary 获取任务模板详情
// @Description 获取指定任务模板的详细信息
// @Tags 任务管理-任务模板
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "模板ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 404 {object} response.Response "模板不存在"
// @Router /task/job-templates/{id} [get]
func (h *Handler) GetJobTemplate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var template model.JobTemplate
	if err := h.db.Where("id = ? AND deleted_at IS NULL", id).First(&template).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "模板不存在")
		return
	}
	response.Success(c, template)
}

// CreateJobTemplate 创建任务模板
// @Summary 创建任务模板
// @Description 创建新的任务模板
// @Tags 任务管理-任务模板
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body model.JobTemplate true "模板信息"
// @Success 200 {object} response.Response "创建成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /task/job-templates [post]
func (h *Handler) CreateJobTemplate(c *gin.Context) {
	var template model.JobTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	template.Status = 1
	template.CreatedBy = 1
	// 处理空的 variables 字段，MySQL JSON 字段不能为空字符串
	if template.Variables == "" {
		template.Variables = "[]"
	}
	if err := h.db.Create(&template).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}
	response.Success(c, template)
}

// UpdateJobTemplate 更新任务模板
// @Summary 更新任务模板
// @Description 更新指定任务模板的信息
// @Tags 任务管理-任务模板
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "模板ID"
// @Param body body model.JobTemplate true "模板信息"
// @Success 200 {object} response.Response "更新成功"
// @Failure 404 {object} response.Response "模板不存在"
// @Router /task/job-templates/{id} [put]
func (h *Handler) UpdateJobTemplate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var template model.JobTemplate
	if err := h.db.Where("id = ? AND deleted_at IS NULL", id).First(&template).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "模板不存在")
		return
	}
	if err := c.ShouldBindJSON(&template); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误")
		return
	}
	h.db.Save(&template)
	response.Success(c, template)
}

// DeleteJobTemplate 删除任务模板
// @Summary 删除任务模板
// @Description 删除指定的任务模板（已禁用）
// @Tags 任务管理-任务模板
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "模板ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 403 {object} response.Response "操作已禁用"
// @Router /task/job-templates/{id} [delete]
func (h *Handler) DeleteJobTemplate(c *gin.Context) {
	response.ErrorCode(c, http.StatusForbidden, "删除模板功能已被禁用，如需删除请联系系统管理员")
}

// ==================== Ansible任务 ====================

// ListAnsibleTasks 获取Ansible任务列表
// @Summary 获取Ansible任务列表
// @Description 分页获取Ansible任务列表，支持按关键词、状态筛选
// @Tags 任务管理-Ansible任务
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键词"
// @Param status query string false "任务状态"
// @Success 200 {object} response.Response "获取成功"
// @Router /task/ansible-tasks [get]
func (h *Handler) ListAnsibleTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")
	status := c.Query("status")

	var tasks []*model.AnsibleTask
	var total int64

	query := h.db.Model(&model.AnsibleTask{}).Where("deleted_at IS NULL")

	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	query.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&tasks)

	response.Success(c, gin.H{
		"list":     tasks,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetAnsibleTask 获取Ansible任务详情
// @Summary 获取Ansible任务详情
// @Description 获取指定Ansible任务的详细信息
// @Tags 任务管理-Ansible任务
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "任务ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 404 {object} response.Response "任务不存在"
// @Router /task/ansible-tasks/{id} [get]
func (h *Handler) GetAnsibleTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var task model.AnsibleTask
	if err := h.db.Where("id = ? AND deleted_at IS NULL", id).First(&task).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "任务不存在")
		return
	}
	response.Success(c, task)
}

// CreateAnsibleTask 创建Ansible任务
// @Summary 创建Ansible任务
// @Description 创建新的Ansible任务
// @Tags 任务管理-Ansible任务
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body model.AnsibleTask true "任务信息"
// @Success 200 {object} response.Response "创建成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /task/ansible-tasks [post]
func (h *Handler) CreateAnsibleTask(c *gin.Context) {
	var ansibleTask model.AnsibleTask
	if err := c.ShouldBindJSON(&ansibleTask); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误")
		return
	}
	ansibleTask.Status = "pending"
	if ansibleTask.Fork == 0 {
		ansibleTask.Fork = 5
	}
	if ansibleTask.Timeout == 0 {
		ansibleTask.Timeout = 600
	}
	if ansibleTask.Verbose == "" {
		ansibleTask.Verbose = "v"
	}
	ansibleTask.CreatedBy = 1
	if err := h.db.Create(&ansibleTask).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败")
		return
	}
	response.Success(c, ansibleTask)
}

// UpdateAnsibleTask 更新Ansible任务
// @Summary 更新Ansible任务
// @Description 更新指定Ansible任务的信息
// @Tags 任务管理-Ansible任务
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "任务ID"
// @Param body body model.AnsibleTask true "任务信息"
// @Success 200 {object} response.Response "更新成功"
// @Failure 404 {object} response.Response "任务不存在"
// @Router /task/ansible-tasks/{id} [put]
func (h *Handler) UpdateAnsibleTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var ansibleTask model.AnsibleTask
	if err := h.db.Where("id = ? AND deleted_at IS NULL", id).First(&ansibleTask).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "任务不存在")
		return
	}
	if err := c.ShouldBindJSON(&ansibleTask); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误")
		return
	}
	h.db.Save(&ansibleTask)
	response.Success(c, ansibleTask)
}

// DeleteAnsibleTask 删除Ansible任务
// @Summary 删除Ansible任务
// @Description 删除指定的Ansible任务（已禁用）
// @Tags 任务管理-Ansible任务
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "任务ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 403 {object} response.Response "操作已禁用"
// @Router /task/ansible-tasks/{id} [delete]
func (h *Handler) DeleteAnsibleTask(c *gin.Context) {
	response.ErrorCode(c, http.StatusForbidden, "删除Ansible任务功能已被禁用，如需删除请联系系统管理员")
}

// ==================== 任务执行 ====================

// ExecuteTaskRequest 执行任务请求
type ExecuteTaskRequest struct {
	HostIDs     []uint `json:"hostIds" binding:"required"`
	ScriptType  string `json:"scriptType" binding:"required"` // Shell, Python
	Content     string `json:"content" binding:"required"`
	Name        string `json:"name"`
}

// ExecuteTaskResponse 执行任务响应
type ExecuteTaskResponse struct {
	TaskID  uint                    `json:"taskId"`
	Results []HostExecutionResult   `json:"results"`
}

// HostExecutionResult 主机执行结果
type HostExecutionResult struct {
	HostID   uint   `json:"hostId"`
	HostName string `json:"hostName"`
	HostIP   string `json:"hostIp"`
	Status   string `json:"status"` // success, failed
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
}

// ExecuteTask 执行任务
// @Summary 执行任务
// @Description 在指定主机上执行Shell或Python脚本
// @Tags 任务管理-任务执行
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body ExecuteTaskRequest true "执行参数"
// @Success 200 {object} response.Response "执行成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 403 {object} response.Response "命令被拦截"
// @Router /task/execute [post]
func (h *Handler) ExecuteTask(c *gin.Context) {
	var req ExecuteTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 安全检查：检查命令是否包含危险操作
	if err := h.checkCommandSafety(req.Content); err != nil {
		response.ErrorCode(c, http.StatusForbidden, err.Error())
		return
	}

	ctx := c.Request.Context()

	// 创建任务记录
	taskName := req.Name
	if taskName == "" {
		taskName = fmt.Sprintf("手动执行任务 - %s", time.Now().Format("2006-01-02 15:04:05"))
	}

	// 从context获取当前用户ID
	var createdBy uint = 1
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uint); ok {
			createdBy = uid
		}
	}

	hostIDsJSON, _ := json.Marshal(req.HostIDs)
	jobTask := model.JobTask{
		Name:        taskName,
		TaskType:    "manual",
		Status:      "running",
		TargetHosts: string(hostIDsJSON),
		Parameters:  "", // 空字符串而不是nil
		CreatedBy:   createdBy,
		ExecuteTime: ptrTime(time.Now()),
	}

	if err := h.db.Create(&jobTask).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, fmt.Sprintf("创建任务记录失败: %v", err))
		return
	}

	// 执行任务
	results := make([]HostExecutionResult, 0, len(req.HostIDs))
	allSuccess := true

	for _, hostID := range req.HostIDs {
		result := h.executeOnHost(ctx, hostID, req.ScriptType, req.Content)
		results = append(results, result)
		if result.Status != "success" {
			allSuccess = false
		}
	}

	// 更新任务状态
	resultJSON, _ := json.Marshal(results)
	if allSuccess {
		jobTask.Status = "success"
	} else {
		jobTask.Status = "failed"
	}
	jobTask.Result = string(resultJSON)
	h.db.Save(&jobTask)

	response.Success(c, ExecuteTaskResponse{
		TaskID:  jobTask.ID,
		Results: results,
	})
}

// checkCommandSafety 检查命令安全性
func (h *Handler) checkCommandSafety(content string) error {
	// 转换为小写以便不区分大小写检查
	contentLower := strings.ToLower(content)

	// 移除空格和特殊字符，防止绕过检测
	contentCompact := strings.ReplaceAll(contentLower, " ", "")
	contentCompact = strings.ReplaceAll(contentCompact, "\t", "")
	contentCompact = strings.ReplaceAll(contentCompact, "\n", "")

	// ============ 完全禁止的命令（一刀切） ============
	absoluteBannedCommands := []string{
		"rm", "unlink", "shred",  // 任何形式的删除
	}

	for _, cmd := range absoluteBannedCommands {
		// 检查命令是否作为独立词出现
		if strings.Contains(contentLower, cmd+" ") ||
		   strings.Contains(contentLower, cmd+"\t") ||
		   strings.Contains(contentLower, cmd+"\n") ||
		   strings.HasPrefix(contentLower, cmd+" ") ||
		   strings.HasSuffix(contentLower, " "+cmd) ||
		   contentLower == cmd {
			return fmt.Errorf("命令【%s】已被完全禁用，系统不允许执行任何删除操作", cmd)
		}
	}

	// ============ 危险命令黑名单 ============
	dangerousPatterns := []struct {
		pattern string
		desc    string
	}{
		// === 磁盘和文件系统操作 ===
		{"dd", "磁盘数据复制命令"},
		{"mkfs", "磁盘格式化命令"},
		{"fdisk", "磁盘分区命令"},
		{"parted", "磁盘分区命令"},
		{"gdisk", "GPT磁盘分区"},
		{"cfdisk", "磁盘分区工具"},
		{"sfdisk", "脚本化分区工具"},
		{"mkswap", "创建交换空间"},
		{"swapoff", "关闭交换空间"},
		{"fsck", "文件系统检查"},
		{"e2fsck", "ext文件系统检查"},
		{"xfs_repair", "XFS文件系统修复"},
		{"tune2fs", "调整文件系统参数"},
		{"resize2fs", "调整文件系统大小"},

		// === 系统控制命令 ===
		{"shutdown", "关机命令"},
		{"reboot", "重启命令"},
		{"halt", "停机命令"},
		{"poweroff", "关机命令"},
		{"init 0", "关机命令"},
		{"init 6", "重启命令"},
		{"telinit", "改变运行级别"},
		{"systemctl reboot", "系统重启"},
		{"systemctl poweroff", "系统关机"},
		{"systemctl halt", "系统停机"},

		// === 进程管理 ===
		{"kill -9", "强制终止进程"},
		{"kill -kill", "强制终止进程"},
		{"killall", "批量终止进程"},
		{"pkill", "批量终止进程"},
		{"killall5", "终止所有进程"},
		{"skill", "发送信号到进程"},

		// === 权限和所有权修改 ===
		{"chmod 777", "最宽松权限设置"},
		{"chmod 666", "危险权限设置"},
		{"chmod 000", "拒绝所有权限"},
		{"chmod -r 777", "递归最宽松权限"},
		{"chmod -r 666", "递归危险权限"},
		{"chmod a+w", "添加写权限"},
		{"chmod u+s", "设置SUID"},
		{"chmod g+s", "设置SGID"},
		{"chown -r", "递归修改所有者"},
		{"chgrp -r", "递归修改组"},
		{"chattr", "修改文件属性"},
		{"setfacl", "设置ACL权限"},

		// === 服务管理 ===
		{"systemctl stop", "停止服务"},
		{"systemctl disable", "禁用服务"},
		{"systemctl mask", "屏蔽服务"},
		{"systemctl kill", "终止服务"},
		{"service stop", "停止服务"},
		{"service restart", "重启服务"},
		{"systemctl restart", "重启服务"},

		// === 网络和防火墙 ===
		{"iptables -f", "清空防火墙规则"},
		{"iptables -x", "删除防火墙链"},
		{"iptables -d", "删除防火墙规则"},
		{"ip6tables -f", "清空IPv6防火墙"},
		{"firewall-cmd --reload", "重载防火墙"},
		{"ufw disable", "禁用防火墙"},
		{"ufw reset", "重置防火墙"},
		{"ifconfig down", "关闭网络接口"},
		{"ip link set down", "关闭网络链路"},
		{"ip addr del", "删除IP地址"},
		{"ip route del", "删除路由"},

		// === 用户和组管理 ===
		{"userdel", "删除用户"},
		{"groupdel", "删除用户组"},
		{"passwd -d", "删除用户密码"},
		{"passwd -l", "锁定用户"},
		{"usermod -l", "修改用户"},
		{"usermod -u 0", "修改用户UID为root"},
		{"visudo", "编辑sudo配置"},

		// === 定时任务 ===
		{"crontab -r", "删除定时任务"},
		{"crontab -e", "编辑定时任务"},
		{"at -r", "删除at任务"},

		// === 系统配置文件 ===
		{"/etc/passwd", "系统密码文件"},
		{"/etc/shadow", "系统影子文件"},
		{"/etc/sudoers", "sudo配置文件"},
		{"/etc/ssh/sshd_config", "SSH配置文件"},
		{"/etc/fstab", "文件系统挂载表"},
		{"/etc/hosts", "主机名解析文件"},
		{"/etc/resolv.conf", "DNS解析配置"},
		{"/etc/profile", "系统环境变量"},
		{"/etc/bashrc", "bash配置"},
		{"/root/.ssh", "root SSH配置"},
		{"/root/.bashrc", "root bash配置"},
		{"> /etc/", "重定向到系统配置目录"},
		{">> /etc/", "追加到系统配置目录"},

		// === 远程脚本执行 ===
		{"curl | bash", "通过curl执行远程脚本"},
		{"curl | sh", "通过curl执行远程脚本"},
		{"curl|bash", "通过curl执行远程脚本（无空格）"},
		{"curl|sh", "通过curl执行远程脚本（无空格）"},
		{"wget | bash", "通过wget执行远程脚本"},
		{"wget | sh", "通过wget执行远程脚本"},
		{"wget|bash", "通过wget执行远程脚本（无空格）"},
		{"wget|sh", "通过wget执行远程脚本（无空格）"},
		{"wget -o- | sh", "通过wget执行远程脚本"},
		{"curl -s | bash", "通过curl静默执行脚本"},
		{"curl -fssl | sh", "通过curl执行脚本"},

		// === 恶意代码模式 ===
		{":(){ :|:& };:", "fork炸弹"},
		{"fork", "fork炸弹相关"},
		{".() { .", "bash函数炸弹"},
		{"while true", "无限循环"},
		{"for((;;))", "无限循环"},

		// === 设备文件操作 ===
		{"> /dev/sd", "写入磁盘设备"},
		{"> /dev/hd", "写入磁盘设备"},
		{"> /dev/vd", "写入虚拟磁盘"},
		{"> /dev/nvme", "写入NVMe设备"},
		{"> /dev/xvd", "写入Xen虚拟磁盘"},
		{"> /dev/null", "重定向到空设备"},
		{"< /dev/zero", "从zero设备读取"},
		{"< /dev/random", "从随机设备读取"},

		// === 内核和模块操作 ===
		{"insmod", "加载内核模块"},
		{"rmmod", "卸载内核模块"},
		{"modprobe -r", "移除内核模块"},
		{"depmod", "生成模块依赖"},
		{"sysctl -w", "修改内核参数"},
		{"echo > /proc/", "修改proc文件系统"},
		{"echo > /sys/", "修改sys文件系统"},

		// === 挂载操作 ===
		{"mount -o remount", "重新挂载"},
		{"mount --bind", "绑定挂载"},
		{"umount -f", "强制卸载"},
		{"umount -l", "懒卸载"},

		// === 历史和日志清理 ===
		{"history -c", "清除命令历史"},
		{"history -w", "写入历史文件"},
		{"> ~/.bash_history", "清空bash历史"},
		{"> /var/log/", "清空日志文件"},
		{"truncate", "截断文件"},
		{"echo > /var/log", "清空日志"},
		{"cat /dev/null >", "清空文件"},

		// === 包管理器删除操作 ===
		{"apt remove", "APT删除软件包"},
		{"apt purge", "APT清除软件包"},
		{"apt autoremove", "APT自动删除"},
		{"yum remove", "YUM删除软件包"},
		{"yum erase", "YUM删除软件包"},
		{"dnf remove", "DNF删除软件包"},
		{"rpm -e", "RPM删除软件包"},
		{"dpkg -r", "DPKG删除软件包"},
		{"dpkg --purge", "DPKG清除软件包"},
		{"pip uninstall", "PIP卸载包"},
		{"npm uninstall", "NPM卸载包"},
		{"gem uninstall", "GEM卸载包"},

		// === 容器和虚拟化 ===
		{"docker rm", "删除Docker容器"},
		{"docker rmi", "删除Docker镜像"},
		{"docker system prune", "清理Docker系统"},
		{"docker volume rm", "删除Docker卷"},
		{"docker network rm", "删除Docker网络"},
		{"kubectl delete", "删除Kubernetes资源"},
		{"kubectl drain", "驱逐节点"},
		{"virsh destroy", "销毁虚拟机"},
		{"virsh undefine", "取消定义虚拟机"},
		{"lxc delete", "删除LXC容器"},

		// === 数据库危险操作 ===
		{"drop database", "删除数据库"},
		{"drop table", "删除数据表"},
		{"truncate table", "清空数据表"},
		{"delete from", "删除数据"},
		{"drop user", "删除数据库用户"},

		// === 编译和代码执行 ===
		{"gcc -o", "编译C代码"},
		{"g++ -o", "编译C++代码"},
		{"python -c", "执行Python代码"},
		{"python3 -c", "执行Python3代码"},
		{"perl -e", "执行Perl代码"},
		{"ruby -e", "执行Ruby代码"},
		{"php -r", "执行PHP代码"},
		{"node -e", "执行Node.js代码"},
		{"eval", "执行动态代码"},
		{"exec", "执行命令"},
		{"source /dev/", "source设备文件"},
		{". /dev/", "点命令执行设备文件"},

		// === SELinux ===
		{"setenforce 0", "禁用SELinux"},
		{"setenforce permissive", "SELinux宽容模式"},
		{"semanage", "SELinux策略管理"},

		// === 加密和压缩 ===
		{"openssl enc", "OpenSSL加密"},
		{"gpg --encrypt", "GPG加密"},
		{"7z a -p", "7z密码压缩"},
		{"zip -P", "ZIP密码压缩"},

		// === 网络扫描和攻击工具 ===
		{"nmap", "网络扫描"},
		{"masscan", "大规模扫描"},
		{"nc -l", "监听端口"},
		{"netcat -l", "监听端口"},
		{"tcpdump", "抓包工具"},
		{"wireshark", "抓包工具"},
		{"tshark", "命令行抓包"},
		{"arpspoof", "ARP欺骗"},
		{"ettercap", "中间人攻击"},
		{"hydra", "暴力破解"},
		{"john", "密码破解"},
		{"hashcat", "哈希破解"},
		{"metasploit", "渗透测试框架"},
		{"sqlmap", "SQL注入工具"},

		// === 其他危险操作 ===
		{"wget --no-check-certificate", "忽略证书验证"},
		{"curl -k", "忽略SSL证书"},
		{"ldconfig", "更新动态链接库缓存"},
		{"updatedb", "更新文件数据库"},
		{"grub-install", "安装GRUB"},
		{"lilo", "LILO引导加载器"},
		{"dd bs=", "指定块大小的dd"},
		{"shred -n", "多次覆盖删除"},
		{"wipe", "安全擦除"},
		{"srm", "安全删除"},
		{"dban", "磁盘擦除"},
		{"badblocks -w", "写入测试坏块"},
		{"hdparm", "硬盘参数设置"},
		{"sdparm", "SCSI磁盘参数"},
	}

	// 检查是否包含危险命令
	for _, dp := range dangerousPatterns {
		if strings.Contains(contentLower, dp.pattern) {
			return fmt.Errorf("命令包含危险操作【%s】，已被系统拦截", dp.desc)
		}
	}

	// ============ 检查危险路径模式 ============
	dangerousPaths := []string{
		"/boot", "/bin", "/sbin", "/lib", "/lib64",
		"/usr/bin", "/usr/sbin", "/usr/lib",
		"/var/lib", "/var/run", "/var/lock",
		"/dev/sd", "/dev/hd", "/dev/vd", "/dev/nvme",
		"/proc/sys", "/sys/class", "/sys/module",
	}

	for _, path := range dangerousPaths {
		if strings.Contains(contentLower, path) {
			return fmt.Errorf("命令涉及系统关键目录【%s】，已被系统拦截", path)
		}
	}

	// ============ 检查根目录操作 ============
	rootPatterns := []string{
		"/*", "/ ", "/\t", "/\n", "/;", "/|", "/&",
	}

	for _, pattern := range rootPatterns {
		if strings.Contains(contentLower, pattern) {
			return fmt.Errorf("命令涉及根目录操作，已被系统拦截")
		}
	}

	// ============ 检查管道和重定向到系统目录 ============
	if strings.Contains(contentCompact, ">/etc") ||
	   strings.Contains(contentCompact, ">>/etc") ||
	   strings.Contains(contentCompact, ">/usr") ||
	   strings.Contains(contentCompact, ">/var") ||
	   strings.Contains(contentCompact, ">/boot") {
		return fmt.Errorf("命令包含重定向到系统目录的操作，已被系统拦截")
	}

	// ============ 检查别名劫持 ============
	if strings.Contains(contentLower, "alias ") {
		return fmt.Errorf("命令包含别名定义，可能导致命令劫持，已被系统拦截")
	}

	// ============ 检查环境变量污染 ============
	dangerousEnvPatterns := []string{
		"export path=",
		"export ld_library_path=",
		"export ld_preload=",
	}

	for _, pattern := range dangerousEnvPatterns {
		if strings.Contains(contentCompact, pattern) {
			return fmt.Errorf("命令包含环境变量修改，可能导致系统异常，已被系统拦截")
		}
	}

	return nil
}

// executeOnHost 在单个主机上执行任务
func (h *Handler) executeOnHost(ctx context.Context, hostID uint, scriptType, content string) HostExecutionResult {
	result := HostExecutionResult{
		HostID: hostID,
		Status: "failed",
	}

	// 获取主机信息
	var host assetbiz.Host
	if err := h.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", hostID).First(&host).Error; err != nil {
		result.Error = fmt.Sprintf("获取主机信息失败: %v", err)
		return result
	}

	result.HostName = host.Name
	result.HostIP = host.IP

	// 获取凭证
	if host.CredentialID == 0 {
		result.Error = "主机未配置凭证"
		return result
	}

	var credential assetbiz.Credential
	if err := h.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", host.CredentialID).First(&credential).Error; err != nil {
		result.Error = fmt.Sprintf("获取凭证失败: %v", err)
		return result
	}

	// 解密凭证
	if err := h.decryptCredential(&credential); err != nil {
		result.Error = fmt.Sprintf("解密凭证失败: %v", err)
		return result
	}

	// 建立SSH连接
	sshClient, err := h.createSSHClient(&host, &credential)
	if err != nil {
		result.Error = fmt.Sprintf("SSH连接失败: %v", err)
		return result
	}
	defer sshClient.Close()

	// 创建SSH会话
	session, err := sshClient.NewSession()
	if err != nil {
		result.Error = fmt.Sprintf("创建SSH会话失败: %v", err)
		return result
	}
	defer session.Close()

	// 根据脚本类型构造执行命令
	var cmd string
	if scriptType == "Python" {
		cmd = fmt.Sprintf("python3 -c %s", shellescape(content))
	} else {
		// Shell 脚本
		cmd = content
	}

	// 执行命令
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		result.Error = fmt.Sprintf("执行失败: %v", err)
		result.Output = string(output)
		return result
	}

	result.Status = "success"
	result.Output = string(output)
	return result
}

// createSSHClient 创建SSH客户端
func (h *Handler) createSSHClient(host *assetbiz.Host, credential *assetbiz.Credential) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	// 根据凭证类型选择认证方式
	switch credential.Type {
	case "password":
		authMethods = append(authMethods, ssh.Password(credential.Password))
	case "private_key":
		signer, err := ssh.ParsePrivateKey([]byte(credential.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("解析私钥失败: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	default:
		return nil, fmt.Errorf("不支持的凭证类型: %s", credential.Type)
	}

	// SSH 配置
	config := &ssh.ClientConfig{
		User:            credential.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应该验证host key
		Timeout:         30 * time.Second,
	}

	// 连接
	addr := fmt.Sprintf("%s:%d", host.IP, host.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// decryptCredential 解密凭证
func (h *Handler) decryptCredential(credential *assetbiz.Credential) error {
	// 解密密码
	if credential.Password != "" {
		decrypted, err := h.decrypt(credential.Password)
		if err != nil {
			return fmt.Errorf("解密密码失败: %w", err)
		}
		credential.Password = decrypted
	}

	// 解密私钥
	if credential.PrivateKey != "" {
		decrypted, err := h.decrypt(credential.PrivateKey)
		if err != nil {
			return fmt.Errorf("解密私钥失败: %w", err)
		}
		credential.PrivateKey = decrypted
	}

	return nil
}

// decrypt 解密
func (h *Handler) decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(h.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// shellescape 转义shell命令
func shellescape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

// ptrTime 返回时间指针
func ptrTime(t time.Time) *time.Time {
	return &t
}

// ==================== 执行记录 ====================

// ListExecutionHistory 获取执行记录列表
// @Summary 获取执行记录列表
// @Description 分页获取任务执行记录列表，支持按关键词、任务类型、状态和日期范围筛选
// @Tags 任务管理-执行记录
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键词"
// @Param taskType query string false "任务类型"
// @Param status query string false "执行状态"
// @Param startDate query string false "开始日期"
// @Param endDate query string false "结束日期"
// @Success 200 {object} response.Response "获取成功"
// @Router /task/execution-history [get]
func (h *Handler) ListExecutionHistory(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")
	taskType := c.Query("taskType")
	status := c.Query("status")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	var jobTasks []model.JobTask
	var total int64

	query := h.db.Model(&model.JobTask{}).Where("deleted_at IS NULL")

	// 关键词搜索
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	// 任务类型筛选
	if taskType != "" {
		query = query.Where("task_type = ?", taskType)
	}
	// 状态筛选
	if status != "" {
		query = query.Where("status = ?", status)
	}
	// 日期范围筛选
	if startDate != "" {
		query = query.Where("created_at >= ?", startDate+" 00:00:00")
	}
	if endDate != "" {
		query = query.Where("created_at <= ?", endDate+" 23:59:59")
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	query.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&jobTasks)

	// 获取用户信息
	type UserInfo struct {
		ID       uint   `gorm:"column:id"`
		Username string `gorm:"column:username"`
	}
	userMap := make(map[uint]string)
	var userIDs []uint
	for _, task := range jobTasks {
		if task.CreatedBy > 0 {
			userIDs = append(userIDs, task.CreatedBy)
		}
	}

	if len(userIDs) > 0 {
		var users []UserInfo
		h.db.Table("sys_user").Select("id, username").Where("id IN ?", userIDs).Find(&users)
		for _, u := range users {
			userMap[u.ID] = u.Username
		}
	}

	// 收集所有主机ID
	type HostInfo struct {
		ID   uint   `gorm:"column:id"`
		Name string `gorm:"column:name"`
		IP   string `gorm:"column:ip"`
	}
	hostMap := make(map[uint]HostInfo)
	var allHostIDs []uint
	for _, task := range jobTasks {
		if task.TargetHosts != "" {
			var hostIDs []uint
			if err := json.Unmarshal([]byte(task.TargetHosts), &hostIDs); err == nil {
				allHostIDs = append(allHostIDs, hostIDs...)
			}
		}
	}

	if len(allHostIDs) > 0 {
		var hosts []HostInfo
		h.db.Table("hosts").Select("id, name, ip").Where("id IN ?", allHostIDs).Find(&hosts)
		for _, host := range hosts {
			hostMap[host.ID] = host
		}
	}

	// 构建响应数据
	type ExecutionHistoryItem struct {
		model.JobTask
		CreatedByName      string `json:"createdByName"`
		TargetHostsDisplay string `json:"targetHostsDisplay"`
	}

	items := make([]ExecutionHistoryItem, 0, len(jobTasks))
	for _, task := range jobTasks {
		item := ExecutionHistoryItem{
			JobTask:       task,
			CreatedByName: userMap[task.CreatedBy],
		}

		// 构建主机显示字符串
		if task.TargetHosts != "" {
			var hostIDs []uint
			if err := json.Unmarshal([]byte(task.TargetHosts), &hostIDs); err == nil {
				var hostDisplays []string
				for _, hid := range hostIDs {
					if host, ok := hostMap[hid]; ok {
						if host.Name != "" && host.IP != "" {
							hostDisplays = append(hostDisplays, fmt.Sprintf("%s(%s)", host.Name, host.IP))
						} else if host.Name != "" {
							hostDisplays = append(hostDisplays, host.Name)
						} else if host.IP != "" {
							hostDisplays = append(hostDisplays, host.IP)
						}
					}
				}
				item.TargetHostsDisplay = strings.Join(hostDisplays, ", ")
			}
		}

		items = append(items, item)
	}

	response.Success(c, gin.H{
		"list":     items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetExecutionHistory 获取执行记录详情
// @Summary 获取执行记录详情
// @Description 获取指定执行记录的详细信息
// @Tags 任务管理-执行记录
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "记录ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 404 {object} response.Response "记录不存在"
// @Router /task/execution-history/{id} [get]
func (h *Handler) GetExecutionHistory(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var jobTask model.JobTask
	if err := h.db.Where("id = ? AND deleted_at IS NULL", id).First(&jobTask).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "执行记录不存在")
		return
	}

	// 获取用户名
	var username string
	if jobTask.CreatedBy > 0 {
		h.db.Table("sys_user").Select("username").Where("id = ?", jobTask.CreatedBy).Scan(&username)
	}

	// 获取主机信息
	type HostInfo struct {
		ID   uint   `gorm:"column:id"`
		Name string `gorm:"column:name"`
		IP   string `gorm:"column:ip"`
	}
	var targetHostsDisplay string
	if jobTask.TargetHosts != "" {
		var hostIDs []uint
		if err := json.Unmarshal([]byte(jobTask.TargetHosts), &hostIDs); err == nil && len(hostIDs) > 0 {
			var hosts []HostInfo
			h.db.Table("hosts").Select("id, name, ip").Where("id IN ?", hostIDs).Find(&hosts)
			var hostDisplays []string
			for _, host := range hosts {
				if host.Name != "" && host.IP != "" {
					hostDisplays = append(hostDisplays, fmt.Sprintf("%s(%s)", host.Name, host.IP))
				} else if host.Name != "" {
					hostDisplays = append(hostDisplays, host.Name)
				} else if host.IP != "" {
					hostDisplays = append(hostDisplays, host.IP)
				}
			}
			targetHostsDisplay = strings.Join(hostDisplays, ", ")
		}
	}

	// 构建响应
	type ExecutionHistoryDetail struct {
		model.JobTask
		CreatedByName      string `json:"createdByName"`
		TargetHostsDisplay string `json:"targetHostsDisplay"`
	}

	detail := ExecutionHistoryDetail{
		JobTask:            jobTask,
		CreatedByName:      username,
		TargetHostsDisplay: targetHostsDisplay,
	}

	response.Success(c, detail)
}

// DeleteExecutionHistory 删除执行记录
// @Summary 删除执行记录
// @Description 删除指定的执行记录
// @Tags 任务管理-执行记录
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "记录ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 500 {object} response.Response "删除失败"
// @Router /task/execution-history/{id} [delete]
func (h *Handler) DeleteExecutionHistory(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := h.db.Delete(&model.JobTask{}, id).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败")
		return
	}
	response.Success(c, nil)
}

// BatchDeleteExecutionHistory 批量删除执行记录
// @Summary 批量删除执行记录
// @Description 批量删除多条执行记录
// @Tags 任务管理-执行记录
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body object true "记录ID列表" example({"ids": [1, 2, 3]})
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /task/execution-history/batch [delete]
func (h *Handler) BatchDeleteExecutionHistory(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误")
		return
	}
	if len(req.IDs) == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "请选择要删除的记录")
		return
	}
	if err := h.db.Delete(&model.JobTask{}, req.IDs).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败")
		return
	}
	response.Success(c, nil)
}

// ExportExecutionHistory 导出执行记录
// @Summary 导出执行记录
// @Description 导出执行记录数据
// @Tags 任务管理-执行记录
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body object false "记录ID列表，不传则导出全部" example({"ids": [1, 2, 3]})
// @Success 200 {object} response.Response "导出成功"
// @Router /task/execution-history/export [post]
func (h *Handler) ExportExecutionHistory(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids"`
	}
	c.ShouldBindJSON(&req)

	var jobTasks []model.JobTask
	query := h.db.Model(&model.JobTask{}).Where("deleted_at IS NULL")

	if len(req.IDs) > 0 {
		query = query.Where("id IN ?", req.IDs)
	}
	query.Order("created_at DESC").Find(&jobTasks)

	// 获取用户信息
	type UserInfo struct {
		ID       uint   `gorm:"column:id"`
		Username string `gorm:"column:username"`
	}
	userMap := make(map[uint]string)
	var userIDs []uint
	for _, task := range jobTasks {
		if task.CreatedBy > 0 {
			userIDs = append(userIDs, task.CreatedBy)
		}
	}
	if len(userIDs) > 0 {
		var users []UserInfo
		h.db.Table("sys_user").Select("id, username").Where("id IN ?", userIDs).Find(&users)
		for _, u := range users {
			userMap[u.ID] = u.Username
		}
	}

	// 获取主机信息
	type HostInfo struct {
		ID   uint   `gorm:"column:id"`
		Name string `gorm:"column:name"`
		IP   string `gorm:"column:ip"`
	}
	hostMap := make(map[uint]HostInfo)
	var allHostIDs []uint
	for _, task := range jobTasks {
		if task.TargetHosts != "" {
			var hostIDs []uint
			if err := json.Unmarshal([]byte(task.TargetHosts), &hostIDs); err == nil {
				allHostIDs = append(allHostIDs, hostIDs...)
			}
		}
	}
	if len(allHostIDs) > 0 {
		var hosts []HostInfo
		h.db.Table("hosts").Select("id, name, ip").Where("id IN ?", allHostIDs).Find(&hosts)
		for _, host := range hosts {
			hostMap[host.ID] = host
		}
	}

	// 构建导出数据
	type ExportItem struct {
		ID                 uint   `json:"id"`
		Name               string `json:"name"`
		TaskType           string `json:"taskType"`
		Status             string `json:"status"`
		TargetHostsDisplay string `json:"targetHostsDisplay"`
		CreatedByName      string `json:"createdByName"`
		CreatedAt          string `json:"createdAt"`
		Result             string `json:"result"`
	}

	items := make([]ExportItem, 0, len(jobTasks))
	for _, task := range jobTasks {
		item := ExportItem{
			ID:            task.ID,
			Name:          task.Name,
			TaskType:      task.TaskType,
			Status:        task.Status,
			CreatedByName: userMap[task.CreatedBy],
			CreatedAt:     task.CreatedAt.Format("2006-01-02 15:04:05"),
			Result:        task.Result,
		}

		// 构建主机显示字符串
		if task.TargetHosts != "" {
			var hostIDs []uint
			if err := json.Unmarshal([]byte(task.TargetHosts), &hostIDs); err == nil {
				var hostDisplays []string
				for _, hid := range hostIDs {
					if host, ok := hostMap[hid]; ok {
						if host.Name != "" && host.IP != "" {
							hostDisplays = append(hostDisplays, fmt.Sprintf("%s(%s)", host.Name, host.IP))
						} else if host.Name != "" {
							hostDisplays = append(hostDisplays, host.Name)
						} else if host.IP != "" {
							hostDisplays = append(hostDisplays, host.IP)
						}
					}
				}
				item.TargetHostsDisplay = strings.Join(hostDisplays, ", ")
			}
		}

		items = append(items, item)
	}

	response.Success(c, items)
}

// ==================== 文件分发 ====================

// FileDistributionResult 文件分发结果
type FileDistributionResult struct {
	HostID   uint   `json:"hostId"`
	HostName string `json:"hostName"`
	HostIP   string `json:"hostIp"`
	Status   string `json:"status"` // success, failed
	Error    string `json:"error,omitempty"`
}

// DistributeFiles 文件分发
// @Summary 文件分发
// @Description 将文件分发到指定的目标主机
// @Tags 任务管理-文件分发
// @Accept multipart/form-data
// @Produce json
// @Security Bearer
// @Param files formData file true "上传的文件"
// @Param targetPath formData string true "目标路径"
// @Param hostIds formData string true "主机ID列表(JSON数组)"
// @Success 200 {object} response.Response "分发成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /task/distribute-files [post]
func (h *Handler) DistributeFiles(c *gin.Context) {
	// 解析表单
	form, err := c.MultipartForm()
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无法解析表单数据: "+err.Error())
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "请上传至少一个文件")
		return
	}

	targetPath := c.PostForm("targetPath")
	if targetPath == "" {
		response.ErrorCode(c, http.StatusBadRequest, "请指定目标路径")
		return
	}

	hostIdsStr := c.PostForm("hostIds")
	if hostIdsStr == "" {
		response.ErrorCode(c, http.StatusBadRequest, "请选择目标主机")
		return
	}

	var hostIDs []uint
	if err := json.Unmarshal([]byte(hostIdsStr), &hostIDs); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "主机ID格式错误")
		return
	}

	if len(hostIDs) == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "请选择至少一台目标主机")
		return
	}

	// 从context获取当前用户ID
	var createdBy uint = 1
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uint); ok {
			createdBy = uid
		}
	}

	ctx := c.Request.Context()

	// 创建任务记录
	fileNames := make([]string, 0, len(files))
	for _, f := range files {
		fileNames = append(fileNames, f.Filename)
	}
	taskName := fmt.Sprintf("文件分发 - %s", time.Now().Format("2006-01-02 15:04:05"))

	hostIDsJSON, _ := json.Marshal(hostIDs)
	paramsJSON, _ := json.Marshal(map[string]interface{}{
		"files":      fileNames,
		"targetPath": targetPath,
	})

	jobTask := model.JobTask{
		Name:        taskName,
		TaskType:    "file",
		Status:      "running",
		TargetHosts: string(hostIDsJSON),
		Parameters:  string(paramsJSON),
		CreatedBy:   createdBy,
		ExecuteTime: ptrTime(time.Now()),
	}

	if err := h.db.Create(&jobTask).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建任务记录失败: "+err.Error())
		return
	}

	// 执行分发
	results := make([]FileDistributionResult, 0, len(hostIDs))
	allSuccess := true

	for _, hostID := range hostIDs {
		result := h.distributeToHost(ctx, hostID, files, targetPath)
		results = append(results, result)
		if result.Status != "success" {
			allSuccess = false
		}
	}

	// 更新任务状态
	resultJSON, _ := json.Marshal(results)
	if allSuccess {
		jobTask.Status = "success"
	} else {
		jobTask.Status = "failed"
	}
	jobTask.Result = string(resultJSON)
	h.db.Save(&jobTask)

	response.Success(c, gin.H{
		"taskId":  jobTask.ID,
		"results": results,
	})
}

// distributeToHost 分发文件到单个主机
func (h *Handler) distributeToHost(ctx context.Context, hostID uint, files []*multipart.FileHeader, targetPath string) FileDistributionResult {
	result := FileDistributionResult{
		HostID: hostID,
		Status: "failed",
	}

	// 获取主机信息
	var host assetbiz.Host
	if err := h.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", hostID).First(&host).Error; err != nil {
		result.Error = fmt.Sprintf("获取主机信息失败: %v", err)
		return result
	}

	result.HostName = host.Name
	result.HostIP = host.IP

	// 获取凭证
	if host.CredentialID == 0 {
		result.Error = "主机未配置凭证"
		return result
	}

	var credential assetbiz.Credential
	if err := h.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", host.CredentialID).First(&credential).Error; err != nil {
		result.Error = fmt.Sprintf("获取凭证失败: %v", err)
		return result
	}

	// 解密凭证
	if err := h.decryptCredential(&credential); err != nil {
		result.Error = fmt.Sprintf("解密凭证失败: %v", err)
		return result
	}

	// 建立SSH连接
	sshClient, err := h.createSSHClient(&host, &credential)
	if err != nil {
		result.Error = fmt.Sprintf("SSH连接失败: %v", err)
		return result
	}
	defer sshClient.Close()

	// 创建SFTP客户端
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		result.Error = fmt.Sprintf("创建SFTP客户端失败: %v", err)
		return result
	}
	defer sftpClient.Close()

	// 确保目标目录存在
	if err := sftpClient.MkdirAll(targetPath); err != nil {
		// 目录可能已存在，忽略错误
	}

	// 上传每个文件
	for _, fileHeader := range files {
		srcFile, err := fileHeader.Open()
		if err != nil {
			result.Error = fmt.Sprintf("打开文件 %s 失败: %v", fileHeader.Filename, err)
			return result
		}

		remotePath := targetPath + "/" + fileHeader.Filename
		dstFile, err := sftpClient.Create(remotePath)
		if err != nil {
			srcFile.Close()
			result.Error = fmt.Sprintf("创建远程文件 %s 失败: %v", remotePath, err)
			return result
		}

		_, err = io.Copy(dstFile, srcFile)
		srcFile.Close()
		dstFile.Close()

		if err != nil {
			result.Error = fmt.Sprintf("传输文件 %s 失败: %v", fileHeader.Filename, err)
			return result
		}
	}

	result.Status = "success"
	return result
}
