package repository

import (
	"context"

	"github.com/ydcloud-dy/opshub/plugins/ssl-cert/model"
	"gorm.io/gorm"
)

// RenewTaskRepository 续期任务仓库
type RenewTaskRepository struct {
	db *gorm.DB
}

// NewRenewTaskRepository 创建续期任务仓库
func NewRenewTaskRepository(db *gorm.DB) *RenewTaskRepository {
	return &RenewTaskRepository{db: db}
}

// Create 创建任务
func (r *RenewTaskRepository) Create(ctx context.Context, task *model.RenewTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// Update 更新任务
func (r *RenewTaskRepository) Update(ctx context.Context, task *model.RenewTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

// GetByID 根据ID获取任务
func (r *RenewTaskRepository) GetByID(ctx context.Context, id uint) (*model.RenewTask, error) {
	var task model.RenewTask
	err := r.db.WithContext(ctx).First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetByIDWithCert 根据ID获取任务(含证书)
func (r *RenewTaskRepository) GetByIDWithCert(ctx context.Context, id uint) (*model.RenewTask, error) {
	var task model.RenewTask
	err := r.db.WithContext(ctx).
		Preload("Certificate").
		First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// List 任务列表
func (r *RenewTaskRepository) List(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]model.RenewTask, int64, error) {
	var tasks []model.RenewTask
	var total int64

	query := r.db.WithContext(ctx).Model(&model.RenewTask{})

	// 应用过滤条件
	if certID, ok := filters["certificate_id"].(uint); ok && certID > 0 {
		query = query.Where("certificate_id = ?", certID)
	}
	if taskType, ok := filters["task_type"].(string); ok && taskType != "" {
		query = query.Where("task_type = ?", taskType)
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if triggerType, ok := filters["trigger_type"].(string); ok && triggerType != "" {
		query = query.Where("trigger_type = ?", triggerType)
	}

	// 计数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).
		Order("created_at DESC").
		Preload("Certificate").
		Find(&tasks).Error
	if err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// ListByCertificateID 根据证书ID获取任务列表
func (r *RenewTaskRepository) ListByCertificateID(ctx context.Context, certID uint, limit int) ([]model.RenewTask, error) {
	var tasks []model.RenewTask
	query := r.db.WithContext(ctx).
		Where("certificate_id = ?", certID).
		Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

// GetLatestByCertificateID 获取证书最新任务
func (r *RenewTaskRepository) GetLatestByCertificateID(ctx context.Context, certID uint, taskType string) (*model.RenewTask, error) {
	var task model.RenewTask
	query := r.db.WithContext(ctx).Where("certificate_id = ?", certID)
	if taskType != "" {
		query = query.Where("task_type = ?", taskType)
	}
	err := query.Order("created_at DESC").First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// HasPendingTask 是否有待处理任务
func (r *RenewTaskRepository) HasPendingTask(ctx context.Context, certID uint, taskType string) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&model.RenewTask{}).
		Where("certificate_id = ?", certID).
		Where("status IN ?", []string{model.TaskStatusPending, model.TaskStatusRunning})
	if taskType != "" {
		query = query.Where("task_type = ?", taskType)
	}
	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// UpdateStatus 更新任务状态
func (r *RenewTaskRepository) UpdateStatus(ctx context.Context, id uint, status string, errorMsg string, result string) error {
	updates := map[string]interface{}{
		"status":        status,
		"error_message": errorMsg,
		"result":        result,
	}
	if status == model.TaskStatusRunning {
		updates["started_at"] = gorm.Expr("NOW()")
	}
	if status == model.TaskStatusSuccess || status == model.TaskStatusFailed {
		updates["finished_at"] = gorm.Expr("NOW()")
	}
	return r.db.WithContext(ctx).Model(&model.RenewTask{}).Where("id = ?", id).Updates(updates).Error
}

// UpdatePendingToSuccess 将证书的待处理任务更新为成功
func (r *RenewTaskRepository) UpdatePendingToSuccess(ctx context.Context, certID uint) error {
	return r.db.WithContext(ctx).Model(&model.RenewTask{}).
		Where("certificate_id = ?", certID).
		Where(
			"status IN ? OR (task_type = ? AND status = ? AND error_message = ?)",
			[]string{model.TaskStatusPending, model.TaskStatusRunning},
			model.TaskTypeIssue,
			model.TaskStatusFailed,
			"任务因服务重启而中断，请重新执行",
		).
		Updates(map[string]interface{}{
			"status":        model.TaskStatusSuccess,
			"finished_at":   gorm.Expr("NOW()"),
			"result":        `{"success":true,"message":"certificate synced from cloud"}`,
			"error_message": "",
		}).Error
}
