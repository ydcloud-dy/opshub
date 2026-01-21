package server

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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

func (h *Handler) GetJobTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var jobTask model.JobTask
	if err := h.db.Where("id = ? AND deleted_at IS NULL", id).First(&jobTask).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "任务不存在")
		return
	}
	response.Success(c, jobTask)
}

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

func (h *Handler) DeleteJobTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	h.db.Delete(&model.JobTask{}, id)
	response.SuccessWithMessage(c, "删除成功", nil)
}

// ==================== 任务模板 ====================

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

func (h *Handler) GetJobTemplate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var template model.JobTemplate
	if err := h.db.Where("id = ? AND deleted_at IS NULL", id).First(&template).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "模板不存在")
		return
	}
	response.Success(c, template)
}

func (h *Handler) CreateJobTemplate(c *gin.Context) {
	var template model.JobTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误")
		return
	}
	template.Status = 1
	template.CreatedBy = 1
	if err := h.db.Create(&template).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败")
		return
	}
	response.Success(c, template)
}

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

func (h *Handler) DeleteJobTemplate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	h.db.Delete(&model.JobTemplate{}, id)
	response.SuccessWithMessage(c, "删除成功", nil)
}

// ==================== Ansible任务 ====================

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

func (h *Handler) GetAnsibleTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var task model.AnsibleTask
	if err := h.db.Where("id = ? AND deleted_at IS NULL", id).First(&task).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "任务不存在")
		return
	}
	response.Success(c, task)
}

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

func (h *Handler) DeleteAnsibleTask(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	h.db.Delete(&model.AnsibleTask{}, id)
	response.SuccessWithMessage(c, "删除成功", nil)
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
func (h *Handler) ExecuteTask(c *gin.Context) {
	var req ExecuteTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	ctx := c.Request.Context()

	// 创建任务记录
	taskName := req.Name
	if taskName == "" {
		taskName = fmt.Sprintf("手动执行任务 - %s", time.Now().Format("2006-01-02 15:04:05"))
	}

	hostIDsJSON, _ := json.Marshal(req.HostIDs)
	jobTask := model.JobTask{
		Name:        taskName,
		TaskType:    "manual",
		Status:      "running",
		TargetHosts: string(hostIDsJSON),
		Parameters:  "", // 空字符串而不是nil
		CreatedBy:   1, // TODO: 从JWT获取用户ID
		ExecuteTime: ptrTime(time.Now()),
	}

	if err := h.db.Create(&jobTask).Error; err != nil {
		// 记录详细的错误信息
		fmt.Printf("创建任务记录失败: %v\n", err)
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
