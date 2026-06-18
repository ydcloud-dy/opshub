package service

import (
	"context"
	"sync"
	"time"

	"github.com/ydcloud-dy/opshub/pkg/logger"
	"github.com/ydcloud-dy/opshub/plugins/ssl-cert/deployer"
	"github.com/ydcloud-dy/opshub/plugins/ssl-cert/model"
	"github.com/ydcloud-dy/opshub/plugins/ssl-cert/provider/acme"
	"github.com/ydcloud-dy/opshub/plugins/ssl-cert/provider/cloud"
	"github.com/ydcloud-dy/opshub/plugins/ssl-cert/provider/dns"
	"github.com/ydcloud-dy/opshub/plugins/ssl-cert/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Scheduler 证书续期调度器
type Scheduler struct {
	db              *gorm.DB
	certRepo        *repository.CertificateRepository
	dnsProviderRepo *repository.DNSProviderRepository
	deployRepo      *repository.DeployConfigRepository
	taskRepo        *repository.RenewTaskRepository
	dnsFactory      *dns.Factory
	cloudFactory    *cloud.Factory
	deployerFactory *deployer.Factory
	deployerDeps    *deployer.Dependencies
	acmeEmail       string
	acmeStaging     bool

	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
	running  bool
	mu       sync.Mutex
}

// NewScheduler 创建调度器
func NewScheduler(
	db *gorm.DB,
	deployerDeps *deployer.Dependencies,
	acmeEmail string,
	acmeStaging bool,
	interval time.Duration,
) *Scheduler {
	if interval == 0 {
		interval = time.Hour // 默认每小时检查一次
	}
	return &Scheduler{
		db:              db,
		certRepo:        repository.NewCertificateRepository(db),
		dnsProviderRepo: repository.NewDNSProviderRepository(db),
		deployRepo:      repository.NewDeployConfigRepository(db),
		taskRepo:        repository.NewRenewTaskRepository(db),
		dnsFactory:      dns.NewFactory(),
		cloudFactory:    cloud.NewFactory(),
		deployerFactory: deployer.NewFactory(),
		deployerDeps:    deployerDeps,
		acmeEmail:       acmeEmail,
		acmeStaging:     acmeStaging,
		interval:        interval,
		stopCh:          make(chan struct{}),
	}
}

// Start 启动调度器
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	// 启动时清理卡住的任务（状态为 running 但程序重启后已丢失的任务）
	s.cleanupStuckTasks()

	s.wg.Add(1)
	go s.run()

	logger.Info("SSL证书续期调度器已启动", zap.Duration("interval", s.interval))
}

// cleanupStuckTasks 清理卡住的任务
func (s *Scheduler) cleanupStuckTasks() {
	ctx := context.Background()

	// 查找所有状态为 running 的任务
	var stuckTasks []model.RenewTask
	err := s.db.Where("status = ?", model.TaskStatusRunning).Find(&stuckTasks).Error
	if err != nil {
		logger.Error("查询卡住的任务失败", zap.Error(err))
		return
	}

	if len(stuckTasks) == 0 {
		return
	}

	logger.Info("发现卡住的任务，正在清理", zap.Int("count", len(stuckTasks)))

	now := time.Now()
	for _, task := range stuckTasks {
		// 将卡住的任务标记为失败
		s.taskRepo.UpdateStatus(ctx, task.ID, model.TaskStatusFailed,
			"任务因服务重启而中断，请重新执行", "")
		// 更新结束时间
		s.db.Model(&model.RenewTask{}).Where("id = ?", task.ID).Update("finished_at", &now)
		logger.Info("已清理卡住的任务", zap.Uint("task_id", task.ID), zap.String("task_type", task.TaskType))
	}
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopCh)
	s.mu.Unlock()

	s.wg.Wait()
	logger.Info("SSL证书续期调度器已停止")
}

// run 运行调度循环
func (s *Scheduler) run() {
	defer s.wg.Done()

	// 启动时立即检查一次
	s.checkAndRenew()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkAndRenew()
		case <-s.stopCh:
			return
		}
	}
}

// checkAndRenew 检查并续期证书
func (s *Scheduler) checkAndRenew() {
	ctx := context.Background()

	// 更新证书状态
	s.updateCertificateStatuses(ctx)

	// 同步待处理的云证书状态
	s.syncPendingCloudCertificates(ctx)

	// 获取需要续期的证书
	certs, err := s.certRepo.ListExpiring(ctx)
	if err != nil {
		logger.Error("获取即将过期证书失败", zap.Error(err))
		return
	}

	if len(certs) == 0 {
		return
	}

	logger.Info("发现即将过期的证书", zap.Int("count", len(certs)))

	for _, cert := range certs {
		// 检查是否有进行中的任务
		hasPending, err := s.taskRepo.HasPendingTask(ctx, cert.ID, "")
		if err != nil {
			logger.Error("检查待处理任务失败", zap.Uint("cert_id", cert.ID), zap.Error(err))
			continue
		}
		if hasPending {
			continue
		}

		// 检查DNS Provider
		if cert.DNSProviderID == nil {
			logger.Warn("证书未配置DNS Provider,无法自动续期", zap.Uint("cert_id", cert.ID), zap.String("domain", cert.Domain))
			continue
		}

		// 执行续期
		go s.renewCertificate(ctx, &cert)
	}
}

// syncPendingCloudCertificates 同步待处理的云证书状态
func (s *Scheduler) syncPendingCloudCertificates(ctx context.Context) {
	// 查询所有待处理的云证书
	var certs []model.SSLCertificate
	err := s.db.Where("status = ?", model.CertStatusPending).
		Where("source_type = ?", model.SourceTypeAliyun).
		Where("cloud_cert_id != ''").
		Find(&certs).Error
	if err != nil {
		logger.Error("查询待处理云证书失败", zap.Error(err))
		return
	}

	if len(certs) == 0 {
		return
	}

	logger.Info("发现待同步的云证书", zap.Int("count", len(certs)))

	for _, cert := range certs {
		s.syncCloudCertificate(ctx, &cert)
	}
}

// syncCloudCertificate 同步单个云证书状态
func (s *Scheduler) syncCloudCertificate(ctx context.Context, cert *model.SSLCertificate) {
	if cert.CloudAccountID == nil || *cert.CloudAccountID == 0 {
		return
	}

	// 获取云账号
	var cloudAccount struct {
		ID        uint
		Provider  string
		AccessKey string
		SecretKey string
	}
	if err := s.db.Table("cloud_accounts").Where("id = ?", *cert.CloudAccountID).First(&cloudAccount).Error; err != nil {
		logger.Error("获取云账号失败", zap.Uint("cert_id", cert.ID), zap.Error(err))
		return
	}

	// 创建云证书Provider
	cloudProvider, err := s.cloudFactory.Create(cert.SourceType, cloudAccount.AccessKey, cloudAccount.SecretKey)
	if err != nil {
		logger.Error("创建云Provider失败", zap.Uint("cert_id", cert.ID), zap.Error(err))
		return
	}

	// 检查证书状态
	status, err := cloudProvider.CheckCertificateStatus(ctx, cert.CloudCertID)
	if err != nil {
		logger.Error("检查云证书状态失败", zap.Uint("cert_id", cert.ID), zap.Error(err))
		return
	}

	// 如果已签发，直接使用返回的证书内容
	if status.Status == "issued" {
		// 阿里云的 DescribeCertificateState 直接返回证书内容
		certificate := status.Certificate
		privateKey := status.PrivateKey

		// 如果没有返回证书内容，尝试下载
		if certificate == "" {
			bundle, err := cloudProvider.DownloadCertificate(ctx, cert.CloudCertID)
			if err != nil {
				logger.Error("下载云证书失败", zap.Uint("cert_id", cert.ID), zap.Error(err))
				return
			}
			certificate = bundle.Certificate
			privateKey = bundle.PrivateKey
		}

		// 解析证书信息
		certInfo, err := acme.ParseCertificatePEM(certificate)
		if err != nil {
			logger.Error("解析证书失败", zap.Uint("cert_id", cert.ID), zap.Error(err))
			return
		}

		// 更新证书内容
		err = s.certRepo.UpdateCertContent(ctx, cert.ID,
			certificate,
			privateKey,
			"",
			&certInfo.NotBefore,
			&certInfo.NotAfter,
			certInfo.Fingerprint,
		)
		if err != nil {
			logger.Error("更新证书内容失败", zap.Uint("cert_id", cert.ID), zap.Error(err))
			return
		}

		// 更新证书状态和issuer
		s.db.Model(cert).Updates(map[string]interface{}{
			"issuer": certInfo.Issuer,
			"status": model.CertStatusActive,
		})

		// 更新任务状态
		s.taskRepo.UpdatePendingToSuccess(ctx, cert.ID)

		logger.Info("云证书同步成功", zap.Uint("cert_id", cert.ID), zap.String("domain", cert.Domain))

		// 执行自动部署
		s.executeAutoDeploy(ctx, cert.ID)
	} else if status.Status == "failed" {
		s.certRepo.UpdateStatus(ctx, cert.ID, model.CertStatusError, status.Message)
		logger.Error("云证书签发失败", zap.Uint("cert_id", cert.ID), zap.String("message", status.Message))
	}
}

// updateCertificateStatuses 更新证书状态
func (s *Scheduler) updateCertificateStatuses(ctx context.Context) {
	now := time.Now()

	// 更新已过期的证书
	s.db.Model(&model.SSLCertificate{}).
		Where("status != ? AND status != ?", model.CertStatusExpired, model.CertStatusError).
		Where("not_after IS NOT NULL AND not_after < ?", now).
		Update("status", model.CertStatusExpired)

	// 更新即将过期的证书
	s.db.Model(&model.SSLCertificate{}).
		Where("status = ?", model.CertStatusActive).
		Where("not_after IS NOT NULL").
		Where("not_after > ? AND not_after <= DATE_ADD(?, INTERVAL renew_days_before DAY)", now, now).
		Update("status", model.CertStatusExpiring)
}

// renewCertificate 续期证书
func (s *Scheduler) renewCertificate(ctx context.Context, cert *model.SSLCertificate) {
	logger.Info("开始自动续期证书", zap.Uint("cert_id", cert.ID), zap.String("domain", cert.Domain))

	// 设置任务超时时间为10分钟，防止任务无限期挂起
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// 创建续期任务（先创建任务，确保所有操作都有记录）
	task := &model.RenewTask{
		CertificateID: cert.ID,
		TaskType:      model.TaskTypeRenew,
		Status:        model.TaskStatusPending,
		TriggerType:   model.TriggerTypeAuto,
	}
	if err := s.taskRepo.Create(ctx, task); err != nil {
		logger.Error("创建续期任务失败", zap.Uint("cert_id", cert.ID), zap.Error(err))
		return
	}

	// 更新任务状态为运行中
	s.taskRepo.UpdateStatus(ctx, task.ID, model.TaskStatusRunning, "", "")

	// 确定ACME邮箱: 优先使用证书配置的邮箱,否则使用全局配置
	acmeEmail := cert.ACMEEmail
	if acmeEmail == "" {
		acmeEmail = s.acmeEmail
	}
	if acmeEmail == "" {
		s.finishTask(ctx, cert, task, false, "ACME email not configured. Please set the email in certificate settings or set environment variable OPSHUB_ACME_EMAIL")
		return
	}

	// 获取DNS Provider
	dnsProvider, err := s.dnsProviderRepo.GetByID(ctx, *cert.DNSProviderID)
	if err != nil {
		s.finishTask(ctx, cert, task, false, "get dns provider failed: "+err.Error())
		return
	}

	// 创建DNS Provider
	dnsP, err := s.dnsFactory.Create(dnsProvider)
	if err != nil {
		s.finishTask(ctx, cert, task, false, "create dns provider failed: "+err.Error())
		return
	}

	// 准备域名列表
	domains := []string{cert.Domain}
	sanDomains := acme.JSONToSANDomains(cert.SANDomains)
	domains = append(domains, sanDomains...)

	// 创建ACME客户端并申请证书
	acmeClient := acme.NewClientWithOptions(acmeEmail, s.acmeStaging, cert.CAProvider, cert.KeyAlgorithm, dnsP)
	bundle, err := acmeClient.ObtainCertificate(ctx, domains)
	if err != nil {
		s.finishTask(ctx, cert, task, false, "obtain certificate failed: "+err.Error())
		return
	}

	// 更新证书内容
	err = s.certRepo.UpdateCertContent(ctx, cert.ID,
		bundle.Certificate,
		bundle.PrivateKey,
		bundle.IssuerCert,
		&bundle.NotBefore,
		&bundle.NotAfter,
		bundle.Fingerprint,
	)
	if err != nil {
		s.finishTask(ctx, cert, task, false, "update certificate content failed: "+err.Error())
		return
	}

	// 更新证书issuer
	s.db.Model(cert).Update("issuer", bundle.Issuer)

	s.finishTask(ctx, cert, task, true, "")

	logger.Info("证书续期成功", zap.Uint("cert_id", cert.ID), zap.String("domain", cert.Domain))

	// 执行自动部署
	s.executeAutoDeploy(ctx, cert.ID)
}

// finishTask 完成任务
func (s *Scheduler) finishTask(ctx context.Context, cert *model.SSLCertificate, task *model.RenewTask, success bool, errMsg string) {
	status := model.TaskStatusSuccess
	certStatus := model.CertStatusActive

	if !success {
		status = model.TaskStatusFailed
		certStatus = model.CertStatusError
		logger.Error("证书续期失败", zap.Uint("cert_id", cert.ID), zap.String("domain", cert.Domain), zap.String("error", errMsg))
	}

	s.taskRepo.UpdateStatus(ctx, task.ID, status, errMsg, "")
	s.certRepo.UpdateStatus(ctx, cert.ID, certStatus, errMsg)
}

// executeAutoDeploy 执行自动部署
func (s *Scheduler) executeAutoDeploy(ctx context.Context, certID uint) {
	deployCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Minute)
	defer cancel()

	runAutoDeploy(
		deployCtx,
		s.certRepo,
		s.deployRepo,
		s.taskRepo,
		s.deployerFactory,
		s.deployerDeps,
		certID,
	)
}

// RunOnce 执行一次检查(用于手动触发)
func (s *Scheduler) RunOnce() {
	s.checkAndRenew()
}
