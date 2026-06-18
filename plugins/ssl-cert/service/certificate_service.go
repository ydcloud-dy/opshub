package service

import (
	"context"
	"encoding/json"
	"fmt"
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

// CertificateService 证书服务
type CertificateService struct {
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
}

// NewCertificateService 创建证书服务
func NewCertificateService(
	db *gorm.DB,
	deployerDeps *deployer.Dependencies,
	acmeEmail string,
	acmeStaging bool,
) *CertificateService {
	return &CertificateService{
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
	}
}

// CreateCertificateRequest 创建证书请求
type CreateCertificateRequest struct {
	Name            string   `json:"name"`
	Domain          string   `json:"domain"`
	SANDomains      []string `json:"san_domains"`
	ACMEEmail       string   `json:"acme_email"`
	SourceType      string   `json:"source_type"`      // acme/aliyun
	CAProvider      string   `json:"ca_provider"`      // CA提供商: letsencrypt/zerossl/google/buypass
	KeyAlgorithm    string   `json:"key_algorithm"`    // 密钥算法: rsa2048/rsa3072/rsa4096/ec256/ec384
	DNSProviderID   uint     `json:"dns_provider_id"`  // DNS验证配置ID (ACME用)
	CloudAccountID  uint     `json:"cloud_account_id"` // 云账号ID (云厂商证书用)
	AutoRenew       bool     `json:"auto_renew"`
	RenewDaysBefore int      `json:"renew_days_before"`
}

// ImportCertificateRequest 导入证书请求
type ImportCertificateRequest struct {
	Name            string   `json:"name"`
	Domain          string   `json:"domain"`
	SANDomains      []string `json:"san_domains"`
	Certificate     string   `json:"certificate"`
	PrivateKey      string   `json:"private_key"`
	CertChain       string   `json:"cert_chain"`
	AutoRenew       bool     `json:"auto_renew"`
	RenewDaysBefore int      `json:"renew_days_before"`
}

// CreateCertificate 申请证书
func (s *CertificateService) CreateCertificate(ctx context.Context, req *CreateCertificateRequest) (*model.SSLCertificate, error) {
	// 设置默认值
	if req.SourceType == "" {
		req.SourceType = model.SourceTypeACME
	}

	// 根据证书类型选择不同的申请流程
	switch req.SourceType {
	case model.SourceTypeACME:
		return s.createACMECertificate(ctx, req)
	case model.SourceTypeAliyun:
		return s.createCloudCertificate(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", req.SourceType)
	}
}

// createACMECertificate ACME证书申请
func (s *CertificateService) createACMECertificate(ctx context.Context, req *CreateCertificateRequest) (*model.SSLCertificate, error) {
	// 验证DNS Provider
	dnsProvider, err := s.dnsProviderRepo.GetByID(ctx, req.DNSProviderID)
	if err != nil {
		return nil, fmt.Errorf("get dns provider failed: %w", err)
	}

	// 确定ACME邮箱: 优先使用请求中的邮箱,否则使用全局配置
	acmeEmail := req.ACMEEmail
	if acmeEmail == "" {
		acmeEmail = s.acmeEmail
	}
	if acmeEmail == "" {
		return nil, fmt.Errorf("ACME email is required for Let's Encrypt certificates. Please provide an email address or set OPSHUB_ACME_EMAIL environment variable")
	}

	// 创建证书记录
	cert := &model.SSLCertificate{
		Name:            req.Name,
		Domain:          req.Domain,
		SANDomains:      acme.SANDomainsToJSON(req.SANDomains),
		ACMEEmail:       acmeEmail,
		SourceType:      req.SourceType,
		CAProvider:      req.CAProvider,
		KeyAlgorithm:    req.KeyAlgorithm,
		DNSProviderID:   &req.DNSProviderID,
		AutoRenew:       req.AutoRenew,
		RenewDaysBefore: req.RenewDaysBefore,
		Status:          model.CertStatusPending,
	}

	// 设置默认值
	if cert.CAProvider == "" {
		cert.CAProvider = model.CAProviderLetsEncrypt
	}
	if cert.KeyAlgorithm == "" {
		cert.KeyAlgorithm = model.KeyAlgorithmRSA2048
	}
	if cert.RenewDaysBefore == 0 {
		cert.RenewDaysBefore = 30
	}

	if err := s.certRepo.Create(ctx, cert); err != nil {
		return nil, fmt.Errorf("create certificate record failed: %w", err)
	}

	// 创建签发任务
	task := &model.RenewTask{
		CertificateID: cert.ID,
		TaskType:      model.TaskTypeIssue,
		Status:        model.TaskStatusPending,
		TriggerType:   model.TriggerTypeManual,
	}
	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create issue task failed: %w", err)
	}

	// 异步执行签发任务
	go s.executeIssueTask(context.Background(), cert, dnsProvider, task)

	return cert, nil
}

// CloudAccount 云账号(简化版)
type CloudAccount struct {
	ID        uint
	Name      string
	Provider  string
	AccessKey string
	SecretKey string
}

// createCloudCertificate 云厂商证书申请
func (s *CertificateService) createCloudCertificate(ctx context.Context, req *CreateCertificateRequest) (*model.SSLCertificate, error) {
	if req.CloudAccountID == 0 {
		return nil, fmt.Errorf("cloud account is required for %s certificate", req.SourceType)
	}

	// 获取云账号信息
	var cloudAccount CloudAccount
	if err := s.db.Table("cloud_accounts").Where("id = ?", req.CloudAccountID).First(&cloudAccount).Error; err != nil {
		return nil, fmt.Errorf("get cloud account failed: %w", err)
	}

	// 验证云账号类型与证书类型匹配
	if cloudAccount.Provider != req.SourceType {
		return nil, fmt.Errorf("cloud account provider (%s) does not match certificate source type (%s)", cloudAccount.Provider, req.SourceType)
	}

	// 创建云证书Provider
	cloudProvider, err := s.cloudFactory.Create(req.SourceType, cloudAccount.AccessKey, cloudAccount.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("create cloud provider failed: %w", err)
	}

	// 创建证书记录
	cert := &model.SSLCertificate{
		Name:            req.Name,
		Domain:          req.Domain,
		SANDomains:      acme.SANDomainsToJSON(req.SANDomains),
		SourceType:      req.SourceType,
		CAProvider:      req.SourceType, // 云厂商证书使用来源类型作为CA提供商
		KeyAlgorithm:    req.KeyAlgorithm,
		CloudAccountID:  &req.CloudAccountID,
		AutoRenew:       req.AutoRenew,
		RenewDaysBefore: req.RenewDaysBefore,
		Status:          model.CertStatusPending,
	}

	// 设置默认值
	if cert.KeyAlgorithm == "" {
		cert.KeyAlgorithm = model.KeyAlgorithmEC256
	}
	if cert.RenewDaysBefore == 0 {
		cert.RenewDaysBefore = 30
	}

	if err := s.certRepo.Create(ctx, cert); err != nil {
		return nil, fmt.Errorf("create certificate record failed: %w", err)
	}

	// 创建签发任务
	task := &model.RenewTask{
		CertificateID: cert.ID,
		TaskType:      model.TaskTypeIssue,
		Status:        model.TaskStatusPending,
		TriggerType:   model.TriggerTypeManual,
	}
	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create issue task failed: %w", err)
	}

	// 异步执行云证书签发任务
	go s.executeCloudIssueTask(context.Background(), cert, cloudProvider, task)

	return cert, nil
}

// executeCloudIssueTask 执行云厂商证书签发任务
func (s *CertificateService) executeCloudIssueTask(ctx context.Context, cert *model.SSLCertificate, cloudProvider cloud.Provider, task *model.RenewTask) {
	// 云厂商证书申请通常很快，但 DNS 验证完成和证书签发之间可能有延迟。
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// 更新任务状态为运行中
	s.taskRepo.UpdateStatus(ctx, task.ID, model.TaskStatusRunning, "", "")

	// 申请证书
	result, err := cloudProvider.ApplyCertificate(ctx, cert.Domain, acme.JSONToSANDomains(cert.SANDomains))
	if err != nil {
		s.finishTask(ctx, cert, task, false, fmt.Sprintf("apply certificate failed: %v", err))
		return
	}

	// 更新证书云厂商ID
	if result.CertID != "" {
		s.db.Model(cert).Update("cloud_cert_id", result.CertID)
	} else if result.OrderID != "" {
		s.db.Model(cert).Update("cloud_cert_id", result.OrderID)
	}

	// 如果需要DNS验证
	if result.Status == "pending_validation" && result.DNSRecord != nil {
		msg := fmt.Sprintf("证书申请已提交，请添加DNS记录完成验证:\n记录名: %s\n记录类型: %s\n记录值: %s",
			result.DNSRecord.RecordName, result.DNSRecord.RecordType, result.DNSRecord.RecordValue)
		s.taskRepo.UpdateStatus(ctx, task.ID, model.TaskStatusRunning, "", msg)
		s.certRepo.UpdateStatus(ctx, cert.ID, model.CertStatusPending, msg)
		s.waitCloudCertificateIssued(ctx, cert, cloudProvider, result.OrderID, task, msg)
		return
	}

	// 如果已签发，获取证书内容
	if result.Status == "issued" {
		certID := result.CertID
		if certID == "" {
			certID = result.OrderID
		}
		s.downloadAndSaveCloudCert(ctx, cert, cloudProvider, certID, task)
		return
	}

	// 其他情况
	s.taskRepo.UpdateStatus(ctx, task.ID, model.TaskStatusRunning, "", result.Message)
	s.waitCloudCertificateIssued(ctx, cert, cloudProvider, result.OrderID, task, result.Message)
}

func (s *CertificateService) waitCloudCertificateIssued(ctx context.Context, cert *model.SSLCertificate, cloudProvider cloud.Provider, certID string, task *model.RenewTask, lastMessage string) {
	if certID == "" {
		return
	}

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	deadline := time.NewTimer(8 * time.Minute)
	defer deadline.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-deadline.C:
			return
		case <-ticker.C:
			status, err := cloudProvider.CheckCertificateStatus(ctx, certID)
			if err != nil {
				s.taskRepo.UpdateStatus(ctx, task.ID, model.TaskStatusRunning, "", fmt.Sprintf("%s\n正在等待云厂商签发，最近一次检查失败: %v", lastMessage, err))
				continue
			}
			if status.Status == "issued" {
				if status.Certificate != "" {
					s.saveCloudCertificateBundle(ctx, cert, task, status.Certificate, status.PrivateKey, status.CertChain)
				} else {
					s.downloadAndSaveCloudCert(ctx, cert, cloudProvider, certID, task)
				}
				return
			}
			if status.Status == "failed" {
				message := status.Message
				if message == "" {
					message = "cloud certificate issuance failed"
				}
				s.finishTask(ctx, cert, task, false, message)
				return
			}
		}
	}
}

// downloadAndSaveCloudCert 下载并保存云证书
func (s *CertificateService) downloadAndSaveCloudCert(ctx context.Context, cert *model.SSLCertificate, cloudProvider cloud.Provider, certID string, task *model.RenewTask) {
	bundle, err := s.downloadCloudCertificateWithFallback(ctx, cert, cloudProvider, certID)
	if err != nil {
		s.finishTask(ctx, cert, task, false, fmt.Sprintf("download certificate failed: %v", err))
		return
	}

	s.saveCloudCertificateBundle(ctx, cert, task, bundle.Certificate, bundle.PrivateKey, bundle.CertChain)
}

func (s *CertificateService) downloadCloudCertificateWithFallback(ctx context.Context, cert *model.SSLCertificate, cloudProvider cloud.Provider, certID string) (*cloud.CertificateBundle, error) {
	if certID != "" {
		if bundle, err := cloudProvider.DownloadCertificate(ctx, certID); err == nil && bundle.Certificate != "" {
			return bundle, nil
		}
	}

	certs, err := cloudProvider.ListCertificates(ctx)
	if err != nil {
		return nil, err
	}

	for _, item := range certs {
		if item == nil || item.CertID == "" {
			continue
		}
		if item.Domain == cert.Domain || containsDomain(item.SANDomains, cert.Domain) {
			bundle, err := cloudProvider.DownloadCertificate(ctx, item.CertID)
			if err == nil && bundle.Certificate != "" {
				s.db.Model(cert).Update("cloud_cert_id", item.CertID)
				return bundle, nil
			}
		}
	}

	return nil, fmt.Errorf("cloud certificate content not found for domain %s", cert.Domain)
}

func containsDomain(domains []string, domain string) bool {
	for _, item := range domains {
		if item == domain {
			return true
		}
	}
	return false
}

func (s *CertificateService) saveCloudCertificateBundle(ctx context.Context, cert *model.SSLCertificate, task *model.RenewTask, certificate, privateKey, certChain string) {
	if certificate == "" {
		s.finishTask(ctx, cert, task, false, "cloud certificate content is empty")
		return
	}

	// 解析证书信息
	certInfo, err := acme.ParseCertificatePEM(certificate)
	if err != nil {
		s.finishTask(ctx, cert, task, false, fmt.Sprintf("parse certificate failed: %v", err))
		return
	}

	// 更新证书内容
	err = s.certRepo.UpdateCertContent(ctx, cert.ID,
		certificate,
		privateKey,
		certChain,
		&certInfo.NotBefore,
		&certInfo.NotAfter,
		certInfo.Fingerprint,
	)
	if err != nil {
		s.finishTask(ctx, cert, task, false, fmt.Sprintf("update certificate content failed: %v", err))
		return
	}

	// 更新证书issuer
	s.db.Model(cert).Update("issuer", certInfo.Issuer)

	s.finishTask(ctx, cert, task, true, "")
	s.executeAutoDeploy(ctx, cert.ID)
}

// SyncCloudCertificate 同步云证书状态（手动检查云厂商证书是否已签发）
func (s *CertificateService) SyncCloudCertificate(ctx context.Context, id uint) error {
	cert, err := s.certRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get certificate failed: %w", err)
	}

	// 只能同步云厂商证书
	if cert.SourceType != model.SourceTypeAliyun {
		return fmt.Errorf("only cloud certificates can be synced")
	}

	// 只有 pending 状态才需要同步
	if cert.Status != model.CertStatusPending {
		return fmt.Errorf("certificate is not in pending status")
	}

	// 获取云账号
	if cert.CloudAccountID == nil || *cert.CloudAccountID == 0 {
		return fmt.Errorf("cloud account not configured")
	}

	var cloudAccount CloudAccount
	if err := s.db.Table("cloud_accounts").Where("id = ?", *cert.CloudAccountID).First(&cloudAccount).Error; err != nil {
		return fmt.Errorf("get cloud account failed: %w", err)
	}

	// 创建云证书Provider
	cloudProvider, err := s.cloudFactory.Create(cert.SourceType, cloudAccount.AccessKey, cloudAccount.SecretKey)
	if err != nil {
		return fmt.Errorf("create cloud provider failed: %w", err)
	}

	// 获取证书ID
	certID := cert.CloudCertID
	if certID == "" {
		return fmt.Errorf("cloud certificate id not found")
	}

	// 检查证书状态
	status, err := cloudProvider.CheckCertificateStatus(ctx, certID)
	if err != nil {
		return fmt.Errorf("check certificate status failed: %w", err)
	}

	// 如果已签发，直接使用返回的证书内容
	if status.Status == "issued" {
		// 阿里云的 DescribeCertificateState 直接返回证书内容
		certificate := status.Certificate
		privateKey := status.PrivateKey

		// 如果没有返回证书内容，尝试下载
		if certificate == "" {
			bundle, err := s.downloadCloudCertificateWithFallback(ctx, cert, cloudProvider, certID)
			if err != nil {
				return fmt.Errorf("download certificate failed: %w", err)
			}
			certificate = bundle.Certificate
			privateKey = bundle.PrivateKey
			status.CertChain = bundle.CertChain
		}

		return s.saveSyncedCloudCertificate(ctx, cert, certificate, privateKey, status.CertChain)
	}

	// 如果失败
	if status.Status == "failed" {
		s.certRepo.UpdateStatus(ctx, cert.ID, model.CertStatusError, status.Message)
		return fmt.Errorf("certificate issuance failed: %s", status.Message)
	}

	// 仍在等待
	return fmt.Errorf("certificate is still pending: %s", status.Message)
}

func (s *CertificateService) saveSyncedCloudCertificate(ctx context.Context, cert *model.SSLCertificate, certificate, privateKey, certChain string) error {
	if certificate == "" {
		return fmt.Errorf("cloud certificate content is empty")
	}

	certInfo, err := acme.ParseCertificatePEM(certificate)
	if err != nil {
		return fmt.Errorf("parse certificate failed: %w", err)
	}

	err = s.certRepo.UpdateCertContent(ctx, cert.ID,
		certificate,
		privateKey,
		certChain,
		&certInfo.NotBefore,
		&certInfo.NotAfter,
		certInfo.Fingerprint,
	)
	if err != nil {
		return fmt.Errorf("update certificate content failed: %w", err)
	}

	s.db.Model(cert).Updates(map[string]interface{}{
		"issuer": certInfo.Issuer,
		"status": model.CertStatusActive,
	})

	if err := s.taskRepo.UpdatePendingToSuccess(ctx, cert.ID); err != nil {
		return fmt.Errorf("update task status failed: %w", err)
	}

	s.executeAutoDeploy(ctx, cert.ID)
	return nil
}

// ImportCertificate 导入证书
func (s *CertificateService) ImportCertificate(ctx context.Context, req *ImportCertificateRequest) (*model.SSLCertificate, error) {
	// 解析证书信息
	certInfo, err := acme.ParseCertificatePEM(req.Certificate)
	if err != nil {
		return nil, fmt.Errorf("parse certificate failed: %w", err)
	}

	// 创建证书记录
	cert := &model.SSLCertificate{
		Name:            req.Name,
		Domain:          req.Domain,
		SANDomains:      acme.SANDomainsToJSON(req.SANDomains),
		SourceType:      model.SourceTypeManual,
		Certificate:     req.Certificate,
		PrivateKey:      req.PrivateKey,
		CertChain:       req.CertChain,
		Issuer:          certInfo.Issuer,
		NotBefore:       &certInfo.NotBefore,
		NotAfter:        &certInfo.NotAfter,
		Fingerprint:     certInfo.Fingerprint,
		AutoRenew:       req.AutoRenew,
		RenewDaysBefore: req.RenewDaysBefore,
		Status:          model.CertStatusActive,
	}

	if cert.RenewDaysBefore == 0 {
		cert.RenewDaysBefore = 30
	}

	// 检查证书状态
	now := time.Now()
	if cert.NotAfter != nil {
		daysUntilExpiry := int(cert.NotAfter.Sub(now).Hours() / 24)
		if daysUntilExpiry <= 0 {
			cert.Status = model.CertStatusExpired
		} else if daysUntilExpiry <= cert.RenewDaysBefore {
			cert.Status = model.CertStatusExpiring
		}
	}

	if err := s.certRepo.Create(ctx, cert); err != nil {
		return nil, fmt.Errorf("create certificate record failed: %w", err)
	}

	return cert, nil
}

// GetCertificate 获取证书详情
func (s *CertificateService) GetCertificate(ctx context.Context, id uint) (*model.SSLCertificate, error) {
	return s.certRepo.GetByIDWithRelations(ctx, id)
}

// ListCertificates 证书列表
func (s *CertificateService) ListCertificates(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]model.SSLCertificate, int64, error) {
	return s.certRepo.List(ctx, page, pageSize, filters)
}

// UpdateCertificate 更新证书配置
func (s *CertificateService) UpdateCertificate(ctx context.Context, id uint, updates map[string]interface{}) error {
	cert, err := s.certRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if name, ok := updates["name"].(string); ok {
		cert.Name = name
	}
	if autoRenew, ok := updates["auto_renew"].(bool); ok {
		cert.AutoRenew = autoRenew
	}
	// 处理 renew_days_before (JSON数字默认是float64)
	if renewDaysBefore, ok := updates["renew_days_before"].(float64); ok {
		cert.RenewDaysBefore = int(renewDaysBefore)
	} else if renewDaysBefore, ok := updates["renew_days_before"].(int); ok {
		cert.RenewDaysBefore = renewDaysBefore
	}
	// 处理 dns_provider_id (JSON数字默认是float64，也可能是nil)
	if dnsProviderID, exists := updates["dns_provider_id"]; exists {
		if dnsProviderID == nil {
			cert.DNSProviderID = nil
		} else if id, ok := dnsProviderID.(float64); ok {
			uid := uint(id)
			cert.DNSProviderID = &uid
		} else if id, ok := dnsProviderID.(uint); ok {
			cert.DNSProviderID = &id
		} else if id, ok := dnsProviderID.(int); ok {
			uid := uint(id)
			cert.DNSProviderID = &uid
		}
	}
	// 处理 acme_email
	if acmeEmail, ok := updates["acme_email"].(string); ok {
		cert.ACMEEmail = acmeEmail
	}

	return s.certRepo.Update(ctx, cert)
}

// DeleteCertificate 删除证书
func (s *CertificateService) DeleteCertificate(ctx context.Context, id uint) error {
	return s.certRepo.Delete(ctx, id)
}

// RenewCertificate 手动续期证书
func (s *CertificateService) RenewCertificate(ctx context.Context, id uint) (*model.RenewTask, error) {
	cert, err := s.certRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get certificate failed: %w", err)
	}

	if cert.DNSProviderID == nil {
		return nil, fmt.Errorf("certificate has no dns provider configured")
	}

	// 检查是否有进行中的任务
	hasPending, err := s.taskRepo.HasPendingTask(ctx, id, "")
	if err != nil {
		return nil, fmt.Errorf("check pending task failed: %w", err)
	}
	if hasPending {
		return nil, fmt.Errorf("certificate has pending task")
	}

	dnsProvider, err := s.dnsProviderRepo.GetByID(ctx, *cert.DNSProviderID)
	if err != nil {
		return nil, fmt.Errorf("get dns provider failed: %w", err)
	}

	// 创建续期任务
	task := &model.RenewTask{
		CertificateID: cert.ID,
		TaskType:      model.TaskTypeRenew,
		Status:        model.TaskStatusPending,
		TriggerType:   model.TriggerTypeManual,
	}
	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create renew task failed: %w", err)
	}

	// 异步执行续期任务
	go s.executeRenewTask(context.Background(), cert, dnsProvider, task)

	return task, nil
}

// GetCertificateContent 获取证书内容(用于下载)
func (s *CertificateService) GetCertificateContent(ctx context.Context, id uint) (*model.CertBundle, error) {
	cert, err := s.certRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &model.CertBundle{
		Certificate: cert.Certificate,
		PrivateKey:  cert.PrivateKey,
		CertChain:   cert.CertChain,
	}, nil
}

// GetCertificateStats 获取证书统计
func (s *CertificateService) GetCertificateStats(ctx context.Context) (map[string]int64, error) {
	return s.certRepo.CountByStatus(ctx)
}

// CloudAccountVO 云账号视图
type CloudAccountVO struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

// GetCloudAccounts 获取云账号列表(用于证书申请)
func (s *CertificateService) GetCloudAccounts(ctx context.Context, provider string) ([]CloudAccountVO, error) {
	var accounts []CloudAccountVO
	query := s.db.Table("cloud_accounts").
		Select("id, name, provider").
		Where("status = ?", 1).     // 只获取启用的账号
		Where("deleted_at IS NULL") // 排除已删除的账号

	if provider != "" {
		query = query.Where("provider = ?", provider)
	} else {
		// 只获取支持证书服务的云厂商
		query = query.Where("provider IN ?", []string{"aliyun"})
	}

	if err := query.Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

// executeIssueTask 执行签发任务
func (s *CertificateService) executeIssueTask(ctx context.Context, cert *model.SSLCertificate, dnsProvider *model.DNSProvider, task *model.RenewTask) {
	// 设置任务超时时间为10分钟，防止任务无限期挂起
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// 更新任务状态为运行中
	s.taskRepo.UpdateStatus(ctx, task.ID, model.TaskStatusRunning, "", "")
	s.updateProgress(ctx, task.ID, "正在准备签发环境...")

	// 确定ACME邮箱: 优先使用证书配置的邮箱,否则使用全局配置
	acmeEmail := cert.ACMEEmail
	if acmeEmail == "" {
		acmeEmail = s.acmeEmail
	}
	if acmeEmail == "" {
		s.finishTask(ctx, cert, task, false, "ACME email not configured. Please set the email in certificate settings or set environment variable OPSHUB_ACME_EMAIL")
		return
	}

	// 创建DNS Provider
	dnsP, err := s.dnsFactory.Create(dnsProvider)
	if err != nil {
		s.finishTask(ctx, cert, task, false, fmt.Sprintf("create dns provider failed: %v", err))
		return
	}

	// 准备域名列表
	domains := []string{cert.Domain}
	sanDomains := acme.JSONToSANDomains(cert.SANDomains)
	domains = append(domains, sanDomains...)

	// 创建ACME客户端并申请证书
	acmeClient := acme.NewClientWithOptions(acmeEmail, s.acmeStaging, cert.CAProvider, cert.KeyAlgorithm, dnsP)
	s.updateProgress(ctx, task.ID, "正在连接ACME服务器并申请证书，预计需要3-5分钟...")
	bundle, err := acmeClient.ObtainCertificate(ctx, domains)
	if err != nil {
		s.finishTask(ctx, cert, task, false, fmt.Sprintf("obtain certificate failed: %v", err))
		return
	}

	s.updateProgress(ctx, task.ID, "证书申请成功，正在保存...")
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
		s.finishTask(ctx, cert, task, false, fmt.Sprintf("update certificate content failed: %v", err))
		return
	}

	// 更新证书issuer
	s.db.Model(cert).Update("issuer", bundle.Issuer)

	s.finishTask(ctx, cert, task, true, "")

	// 执行自动部署
	s.executeAutoDeploy(ctx, cert.ID)
}

// executeRenewTask 执行续期任务
func (s *CertificateService) executeRenewTask(ctx context.Context, cert *model.SSLCertificate, dnsProvider *model.DNSProvider, task *model.RenewTask) {
	logger.Info("开始手动续期证书", zap.Uint("cert_id", cert.ID), zap.String("domain", cert.Domain), zap.Uint("task_id", task.ID))

	// 设置任务超时时间为10分钟，防止任务无限期挂起
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// 更新任务状态为运行中
	s.taskRepo.UpdateStatus(ctx, task.ID, model.TaskStatusRunning, "", "")
	s.updateProgress(ctx, task.ID, "正在准备续期环境...")

	// 确定ACME邮箱: 优先使用证书配置的邮箱,否则使用全局配置
	acmeEmail := cert.ACMEEmail
	if acmeEmail == "" {
		acmeEmail = s.acmeEmail
	}
	if acmeEmail == "" {
		s.finishTask(ctx, cert, task, false, "ACME email not configured. Please set the email in certificate settings or set environment variable OPSHUB_ACME_EMAIL")
		return
	}

	logger.Info("使用ACME邮箱", zap.String("email", acmeEmail), zap.Uint("task_id", task.ID))

	// 创建DNS Provider
	dnsP, err := s.dnsFactory.Create(dnsProvider)
	if err != nil {
		s.finishTask(ctx, cert, task, false, fmt.Sprintf("create dns provider failed: %v", err))
		return
	}

	// 准备域名列表
	domains := []string{cert.Domain}
	sanDomains := acme.JSONToSANDomains(cert.SANDomains)
	domains = append(domains, sanDomains...)

	logger.Info("开始申请证书", zap.Strings("domains", domains), zap.Uint("task_id", task.ID))
	s.updateProgress(ctx, task.ID, "正在连接ACME服务器并申请证书，预计需要3-5分钟...")

	// 创建ACME客户端并申请证书
	acmeClient := acme.NewClientWithOptions(acmeEmail, s.acmeStaging, cert.CAProvider, cert.KeyAlgorithm, dnsP)
	bundle, err := acmeClient.ObtainCertificate(ctx, domains)
	if err != nil {
		logger.Error("申请证书失败", zap.Uint("task_id", task.ID), zap.Error(err))
		s.finishTask(ctx, cert, task, false, fmt.Sprintf("obtain certificate failed: %v", err))
		return
	}

	logger.Info("证书申请成功，正在保存", zap.Uint("task_id", task.ID))
	s.updateProgress(ctx, task.ID, "证书申请成功，正在保存...")

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
		s.finishTask(ctx, cert, task, false, fmt.Sprintf("update certificate content failed: %v", err))
		return
	}

	// 更新证书issuer
	s.db.Model(cert).Update("issuer", bundle.Issuer)

	logger.Info("手动续期证书成功", zap.Uint("cert_id", cert.ID), zap.String("domain", cert.Domain), zap.Uint("task_id", task.ID))
	s.finishTask(ctx, cert, task, true, "")

	// 执行自动部署
	s.executeAutoDeploy(ctx, cert.ID)
}

// updateProgress 更新任务进度信息
func (s *CertificateService) updateProgress(ctx context.Context, taskID uint, message string) {
	result := model.TaskResult{
		Message: message,
	}
	resultJSON, _ := json.Marshal(result)
	s.db.WithContext(ctx).Model(&model.RenewTask{}).Where("id = ?", taskID).Update("result", string(resultJSON))
}

// finishTask 完成任务
func (s *CertificateService) finishTask(ctx context.Context, cert *model.SSLCertificate, task *model.RenewTask, success bool, errMsg string) {
	status := model.TaskStatusSuccess
	certStatus := model.CertStatusActive

	if !success {
		status = model.TaskStatusFailed
		certStatus = model.CertStatusError
	}

	result := model.TaskResult{
		Success: success,
		Message: errMsg,
	}
	resultJSON, _ := json.Marshal(result)

	s.taskRepo.UpdateStatus(ctx, task.ID, status, errMsg, string(resultJSON))
	s.certRepo.UpdateStatus(ctx, cert.ID, certStatus, errMsg)
}

// executeAutoDeploy 执行自动部署
func (s *CertificateService) executeAutoDeploy(ctx context.Context, certID uint) {
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
