package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ydcloud-dy/opshub/pkg/logger"
	"github.com/ydcloud-dy/opshub/plugins/ssl-cert/deployer"
	"github.com/ydcloud-dy/opshub/plugins/ssl-cert/model"
	"github.com/ydcloud-dy/opshub/plugins/ssl-cert/repository"
	"go.uber.org/zap"
)

func runAutoDeploy(
	ctx context.Context,
	certRepo *repository.CertificateRepository,
	deployRepo *repository.DeployConfigRepository,
	taskRepo *repository.RenewTaskRepository,
	deployerFactory *deployer.Factory,
	deployerDeps *deployer.Dependencies,
	certID uint,
) {
	configs, err := deployRepo.ListAutoDeploy(ctx, certID)
	if err != nil {
		logger.Error("获取自动部署配置失败", zap.Uint("cert_id", certID), zap.Error(err))
		return
	}
	if len(configs) == 0 {
		return
	}

	cert, err := certRepo.GetByID(ctx, certID)
	if err != nil {
		logger.Error("获取证书失败", zap.Uint("cert_id", certID), zap.Error(err))
		return
	}
	if cert.Certificate == "" || cert.PrivateKey == "" {
		logger.Warn("证书内容为空，跳过自动部署", zap.Uint("cert_id", certID), zap.String("domain", cert.Domain))
		for _, config := range configs {
			now := time.Now()
			_ = deployRepo.UpdateDeployResult(ctx, config.ID, false, &now, "certificate content is empty")
			task := createAutoDeployTask(ctx, taskRepo, certID, &config)
			finishAutoDeployTask(ctx, taskRepo, task, &config, false, "certificate content is empty")
		}
		return
	}

	bundle := &model.CertBundle{
		Certificate: cert.Certificate,
		PrivateKey:  cert.PrivateKey,
		CertChain:   cert.CertChain,
	}

	for _, config := range configs {
		task := createAutoDeployTask(ctx, taskRepo, certID, &config)

		d, err := deployerFactory.Create(config.DeployType, deployerDeps)
		if err != nil {
			now := time.Now()
			_ = deployRepo.UpdateDeployResult(ctx, config.ID, false, &now, err.Error())
			finishAutoDeployTask(ctx, taskRepo, task, &config, false, fmt.Sprintf("create deployer failed: %v", err))
			logger.Error("创建部署器失败", zap.Uint("config_id", config.ID), zap.String("name", config.Name), zap.Error(err))
			continue
		}

		err = d.Deploy(ctx, bundle, &config)
		now := time.Now()
		if err != nil {
			_ = deployRepo.UpdateDeployResult(ctx, config.ID, false, &now, err.Error())
			finishAutoDeployTask(ctx, taskRepo, task, &config, false, err.Error())
			logger.Error("自动部署失败", zap.Uint("config_id", config.ID), zap.String("name", config.Name), zap.Error(err))
			continue
		}

		_ = deployRepo.UpdateDeployResult(ctx, config.ID, true, &now, "")
		finishAutoDeployTask(ctx, taskRepo, task, &config, true, "")
		logger.Info("自动部署成功", zap.Uint("config_id", config.ID), zap.String("name", config.Name))
	}
}

func createAutoDeployTask(ctx context.Context, taskRepo *repository.RenewTaskRepository, certID uint, config *model.DeployConfig) *model.RenewTask {
	now := time.Now()
	task := &model.RenewTask{
		CertificateID: certID,
		TaskType:      model.TaskTypeDeploy,
		Status:        model.TaskStatusRunning,
		TriggerType:   model.TriggerTypeAuto,
		StartedAt:     &now,
	}
	if err := taskRepo.Create(ctx, task); err != nil {
		logger.Error("创建自动部署任务失败", zap.Uint("cert_id", certID), zap.Uint("config_id", config.ID), zap.Error(err))
		return nil
	}
	return task
}

func finishAutoDeployTask(ctx context.Context, taskRepo *repository.RenewTaskRepository, task *model.RenewTask, config *model.DeployConfig, success bool, message string) {
	if task == nil {
		return
	}

	status := model.TaskStatusSuccess
	if !success {
		status = model.TaskStatusFailed
	}

	result := map[string]interface{}{
		"success":            success,
		"message":            message,
		"deploy_config_id":   config.ID,
		"deploy_config_name": config.Name,
		"deploy_type":        config.DeployType,
	}
	if success {
		result["deployed_to"] = []string{config.Name}
	} else {
		result["deploy_errors"] = []string{message}
	}

	resultJSON, _ := json.Marshal(result)
	if err := taskRepo.UpdateStatus(ctx, task.ID, status, message, string(resultJSON)); err != nil {
		logger.Error("更新自动部署任务状态失败", zap.Uint("task_id", task.ID), zap.Error(err))
	}
}
