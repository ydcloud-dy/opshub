package service

import (
	"context"

	"github.com/ydcloud-dy/opshub/plugins/ssl-cert/model"
	"github.com/ydcloud-dy/opshub/plugins/ssl-cert/repository"
	"gorm.io/gorm"
)

// TaskService 任务服务
type TaskService struct {
	db      *gorm.DB
	repo    *repository.RenewTaskRepository
	certSvc *CertificateService
}

// NewTaskService 创建任务服务
func NewTaskService(db *gorm.DB, certSvc ...*CertificateService) *TaskService {
	var certificateService *CertificateService
	if len(certSvc) > 0 {
		certificateService = certSvc[0]
	}
	return &TaskService{
		db:      db,
		repo:    repository.NewRenewTaskRepository(db),
		certSvc: certificateService,
	}
}

// GetTask 获取任务详情
func (s *TaskService) GetTask(ctx context.Context, id uint) (*model.RenewTask, error) {
	task, err := s.repo.GetByIDWithCert(ctx, id)
	if err != nil {
		return nil, err
	}
	s.syncTaskCloudCertificate(ctx, task)
	return s.repo.GetByIDWithCert(ctx, id)
}

// ListTasks 任务列表
func (s *TaskService) ListTasks(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]model.RenewTask, int64, error) {
	tasks, total, err := s.repo.List(ctx, page, pageSize, filters)
	if err != nil {
		return nil, 0, err
	}
	changed := false
	for i := range tasks {
		if s.syncTaskCloudCertificate(ctx, &tasks[i]) {
			changed = true
		}
	}
	if changed {
		tasks, total, err = s.repo.List(ctx, page, pageSize, filters)
	}
	return tasks, total, err
}

// ListTasksByCertificate 根据证书获取任务列表
func (s *TaskService) ListTasksByCertificate(ctx context.Context, certID uint, limit int) ([]model.RenewTask, error) {
	return s.repo.ListByCertificateID(ctx, certID, limit)
}

// GetLatestTask 获取证书最新任务
func (s *TaskService) GetLatestTask(ctx context.Context, certID uint, taskType string) (*model.RenewTask, error) {
	return s.repo.GetLatestByCertificateID(ctx, certID, taskType)
}

func (s *TaskService) syncTaskCloudCertificate(ctx context.Context, task *model.RenewTask) bool {
	if s.certSvc == nil || task == nil || task.Certificate == nil {
		return false
	}
	if task.TaskType != model.TaskTypeIssue {
		return false
	}
	if task.Status != model.TaskStatusPending &&
		task.Status != model.TaskStatusRunning &&
		!(task.Status == model.TaskStatusFailed && task.ErrorMessage == "任务因服务重启而中断，请重新执行") {
		return false
	}
	if task.Certificate.SourceType != model.SourceTypeAliyun || task.Certificate.Status != model.CertStatusPending {
		return false
	}
	return s.certSvc.SyncCloudCertificate(ctx, task.CertificateID) == nil
}
