package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ydcloud-dy/opshub/plugins/app-inventory/model"
	"gorm.io/gorm"
)

type Service struct {
	db      *gorm.DB
	cipher  *SecretCipher
	monitor *healthMonitor
}

func New(db *gorm.DB, cipher *SecretCipher) *Service {
	return &Service{db: db, cipher: cipher, monitor: newHealthMonitor()}
}

type ListOptions struct {
	Page     int
	PageSize int
	Keyword  string
	AppID    uint
	EnvID    uint
	Kind     string
	Category string
	Status   string
}

type Page[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

func normalizePage(opts ListOptions) (int, int) {
	page := opts.Page
	if page < 1 {
		page = 1
	}
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

type ApplicationInput struct {
	Code             string `json:"code"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	EnvironmentID    uint   `json:"environmentId"`
	OwnerUserID      uint   `json:"ownerUserId"`
	DepartmentID     uint   `json:"-"`
	Criticality      string `json:"criticality"`
	Status           string `json:"status"`
	RepositoryURL    string `json:"repositoryUrl"`
	DocumentationURL string `json:"documentationUrl"`
	Language         string `json:"language"`
	Tags             string `json:"tags"`
	OwnerName        string `json:"-"`
	Team             string `json:"-"`
	Lifecycle        string `json:"-"`
	HealthStatus     string `json:"-"`
}

type EnvironmentInput struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Region      string `json:"region"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

type DomainInput struct {
	ApplicationID uint   `json:"applicationId"`
	Domain        string `json:"domain"`
	Protocol      string `json:"protocol"`
	Port          int    `json:"port"`
	Path          string `json:"path"`
	DNSProvider   string `json:"-"`
	CertificateID uint   `json:"certificateId"`
	IsPrimary     bool   `json:"isPrimary"`
	Source        string `json:"-"`
	Description   string `json:"description"`
}

type ResourceInput struct {
	ApplicationID uint   `json:"applicationId"`
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	Address       string `json:"address"`
	Port          int    `json:"port"`
	HostID        uint   `json:"hostId"`
	ClusterID     uint   `json:"clusterId"`
	Namespace     string `json:"namespace"`
	ExternalID    string `json:"externalId"`
	CredentialID  uint   `json:"credentialId"`
	Source        string `json:"-"`
	Metadata      string `json:"-"`
	Description   string `json:"description"`
}

type ComponentInput struct {
	ApplicationID uint   `json:"applicationId"`
	Category      string `json:"category"`
	Type          string `json:"type"`
	Name          string `json:"name"`
	Address       string `json:"address"`
	Port          int    `json:"port"`
	DatabaseName  string `json:"databaseName"`
	Version       string `json:"version"`
	CredentialID  uint   `json:"credentialId"`
	TLSEnabled    bool   `json:"tlsEnabled"`
	Source        string `json:"-"`
	Metadata      string `json:"-"`
	Description   string `json:"description"`
}

type DependencyInput struct {
	SourceApplicationID uint   `json:"sourceApplicationId"`
	SourceEnvironmentID uint   `json:"sourceEnvironmentId"`
	TargetApplicationID uint   `json:"targetApplicationId"`
	TargetComponentID   uint   `json:"targetComponentId"`
	TargetResourceID    uint   `json:"targetResourceId"`
	TargetName          string `json:"targetName"`
	RelationType        string `json:"relationType"`
	Protocol            string `json:"protocol"`
	Endpoint            string `json:"endpoint"`
	Port                int    `json:"port"`
	Criticality         string `json:"criticality"`
	Status              string `json:"status"`
	Description         string `json:"description"`
}

type ApplicationSummary struct {
	model.Application
	EnvironmentCount int `json:"environmentCount"`
	DomainCount      int `json:"domainCount"`
	ResourceCount    int `json:"resourceCount"`
	ComponentCount   int `json:"componentCount"`
	DependencyCount  int `json:"dependencyCount"`
}

type ApplicationDetail struct {
	ApplicationSummary
	Environment  *model.Environment  `json:"environment"`
	Environments []model.Environment `json:"environments"`
	Domains      []model.Domain      `json:"domains"`
	Resources    []model.Resource    `json:"resources"`
	Components   []model.Component   `json:"components"`
	Dependencies []model.Dependency  `json:"dependencies"`
}

func validateApplicationInput(in *ApplicationInput) error {
	in.Code = strings.TrimSpace(in.Code)
	in.Name = strings.TrimSpace(in.Name)
	in.Language = strings.TrimSpace(in.Language)
	in.RepositoryURL = strings.TrimSpace(in.RepositoryURL)
	in.DocumentationURL = strings.TrimSpace(in.DocumentationURL)
	if in.Code == "" || in.Name == "" {
		return errors.New("应用编码和名称不能为空")
	}
	if !regexpCode.MatchString(in.Code) {
		return errors.New("应用编码只能包含字母、数字、点、下划线和短横线")
	}
	if len(in.Code) > 80 || len(in.Name) > 120 {
		return errors.New("应用编码或名称过长")
	}
	if in.OwnerUserID == 0 || in.EnvironmentID == 0 {
		return errors.New("负责人和运行环境不能为空")
	}
	if len(in.Language) > 80 || len(in.Description) > 1000 {
		return errors.New("应用资料超过长度限制")
	}
	if in.Criticality == "" {
		in.Criticality = "medium"
	}
	if in.Status == "" {
		in.Status = "active"
	}
	if !oneOf(in.Criticality, "critical", "high", "medium", "low") {
		return errors.New("应用重要级别无效")
	}
	if !oneOf(in.Status, "active", "disabled", "planned") {
		return errors.New("应用状态无效")
	}
	if err := validateOptionalHTTPURL("代码仓库", in.RepositoryURL); err != nil {
		return err
	}
	if err := validateOptionalHTTPURL("文档地址", in.DocumentationURL); err != nil {
		return err
	}
	tags, err := normalizeTagsJSON(in.Tags)
	if err != nil {
		return err
	}
	in.Tags = tags
	return nil
}

func (s *Service) CreateApplication(ctx context.Context, in *ApplicationInput, userID uint) (*model.Application, error) {
	if err := validateApplicationInput(in); err != nil {
		return nil, err
	}
	references, err := s.resolveApplicationReferences(ctx, in)
	if err != nil {
		return nil, err
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.Application{}).Where("code = ?", in.Code).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("应用编码 %q 已存在", in.Code)
	}
	app := &model.Application{
		Code:             in.Code,
		Name:             in.Name,
		Description:      strings.TrimSpace(in.Description),
		EnvironmentID:    in.EnvironmentID,
		OwnerName:        references.OwnerName,
		OwnerUserID:      in.OwnerUserID,
		DepartmentID:     in.DepartmentID,
		Team:             references.DepartmentName,
		Criticality:      in.Criticality,
		Status:           in.Status,
		Lifecycle:        lifecycleFromEnvironment(references.Environment.Kind),
		HealthStatus:     "unknown",
		HealthMessage:    "暂无可探测资产",
		HealthSource:     "asset-aggregation",
		RepositoryURL:    strings.TrimSpace(in.RepositoryURL),
		DocumentationURL: strings.TrimSpace(in.DocumentationURL),
		Language:         strings.TrimSpace(in.Language),
		Tags:             in.Tags,
		CreatedBy:        userID,
		UpdatedBy:        userID,
	}
	if err := s.db.WithContext(ctx).Create(app).Error; err != nil {
		return nil, err
	}
	applyApplicationReferences(app, references)
	return app, nil
}

func (s *Service) UpdateApplication(ctx context.Context, id uint, in *ApplicationInput, userID uint) (*model.Application, error) {
	if err := validateApplicationInput(in); err != nil {
		return nil, err
	}
	references, err := s.resolveApplicationReferences(ctx, in)
	if err != nil {
		return nil, err
	}
	var app model.Application
	if err := s.db.WithContext(ctx).First(&app, id).Error; err != nil {
		return nil, err
	}
	if in.Code != app.Code {
		return nil, errors.New("应用编码创建后不可修改")
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.Application{}).Where("code = ? AND id <> ?", in.Code, id).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("应用编码 %q 已存在", in.Code)
	}
	app.Code = in.Code
	app.Name = in.Name
	app.Description = strings.TrimSpace(in.Description)
	previousEnvironmentID := app.EnvironmentID
	app.EnvironmentID = in.EnvironmentID
	app.OwnerName = references.OwnerName
	app.OwnerUserID = in.OwnerUserID
	app.DepartmentID = in.DepartmentID
	app.Team = references.DepartmentName
	app.Criticality = in.Criticality
	app.Status = in.Status
	app.Lifecycle = lifecycleFromEnvironment(references.Environment.Kind)
	app.RepositoryURL = strings.TrimSpace(in.RepositoryURL)
	app.DocumentationURL = strings.TrimSpace(in.DocumentationURL)
	app.Language = strings.TrimSpace(in.Language)
	app.Tags = in.Tags
	app.UpdatedBy = userID
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&app).Error; err != nil {
			return err
		}
		if previousEnvironmentID != app.EnvironmentID {
			if err := syncApplicationEnvironment(tx, app.ID, app.EnvironmentID); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if err := s.RecomputeApplicationHealth(ctx, app.ID); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Select("health_status", "health_checked_at", "health_message", "health_source").First(&app, app.ID).Error; err != nil {
		return nil, err
	}
	applyApplicationReferences(&app, references)
	return &app, nil
}

func (s *Service) DeleteApplication(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var app model.Application
		if err := tx.First(&app, id).Error; err != nil {
			return err
		}
		componentIDs := tx.Model(&model.Component{}).Select("id").Where("application_id = ?", id)
		resourceIDs := tx.Model(&model.Resource{}).Select("id").Where("application_id = ?", id)
		if err := tx.Where(
			"source_application_id = ? OR target_application_id = ? OR target_component_id IN (?) OR target_resource_id IN (?)",
			id, id, componentIDs, resourceIDs,
		).Delete(&model.Dependency{}).Error; err != nil {
			return err
		}
		for _, item := range []interface{}{&model.Domain{}, &model.Resource{}, &model.Component{}} {
			if err := tx.Where("application_id = ?", id).Delete(item).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("application_id = ?", id).Delete(&model.DiscoveryRun{}).Error; err != nil {
			return err
		}
		return tx.Delete(&app).Error
	})
}

func (s *Service) ListApplications(ctx context.Context, opts ListOptions) (Page[ApplicationSummary], error) {
	page, pageSize := normalizePage(opts)
	query := s.db.WithContext(ctx).Model(&model.Application{})
	if keyword := strings.TrimSpace(opts.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("code LIKE ? OR name LIKE ? OR owner_name LIKE ? OR team LIKE ? OR language LIKE ? OR tags LIKE ? OR description LIKE ?", like, like, like, like, like, like, like)
	}
	if opts.Status != "" {
		query = query.Where("status = ?", opts.Status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return Page[ApplicationSummary]{}, err
	}
	var apps []model.Application
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&apps).Error; err != nil {
		return Page[ApplicationSummary]{}, err
	}
	if err := s.enrichApplications(ctx, apps); err != nil {
		return Page[ApplicationSummary]{}, err
	}
	items := make([]ApplicationSummary, len(apps))
	ids := make([]uint, 0, len(apps))
	for i := range apps {
		items[i].Application = apps[i]
		ids = append(ids, apps[i].ID)
	}
	if len(ids) > 0 {
		countMap := map[string]map[uint]int{}
		for _, spec := range []struct {
			key   string
			model interface{}
			field string
		}{
			{key: "domain", model: &model.Domain{}, field: "application_id"},
			{key: "resource", model: &model.Resource{}, field: "application_id"},
			{key: "component", model: &model.Component{}, field: "application_id"},
			{key: "dependency", model: &model.Dependency{}, field: "source_application_id"},
		} {
			var rows []struct {
				ApplicationID uint
				Count         int
			}
			if err := s.db.WithContext(ctx).Model(spec.model).Select(spec.field+" AS application_id, COUNT(*) AS count").Where(spec.field+" IN ?", ids).Group(spec.field).Find(&rows).Error; err != nil {
				return Page[ApplicationSummary]{}, err
			}
			counts := make(map[uint]int, len(rows))
			for _, row := range rows {
				counts[row.ApplicationID] = row.Count
			}
			countMap[spec.key] = counts
		}
		for i := range items {
			id := items[i].ID
			if items[i].EnvironmentID > 0 {
				items[i].EnvironmentCount = 1
			}
			items[i].DomainCount = countMap["domain"][id]
			items[i].ResourceCount = countMap["resource"][id]
			items[i].ComponentCount = countMap["component"][id]
			items[i].DependencyCount = countMap["dependency"][id]
		}
	}
	return Page[ApplicationSummary]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) GetApplication(ctx context.Context, id uint) (*ApplicationDetail, error) {
	var app model.Application
	if err := s.db.WithContext(ctx).First(&app, id).Error; err != nil {
		return nil, err
	}
	apps := []model.Application{app}
	if err := s.enrichApplications(ctx, apps); err != nil {
		return nil, err
	}
	app = apps[0]
	summary := ApplicationSummary{Application: app}
	if app.EnvironmentID > 0 {
		summary.EnvironmentCount = 1
	}
	for target, item := range map[*int]interface{}{
		&summary.DomainCount:    &model.Domain{},
		&summary.ResourceCount:  &model.Resource{},
		&summary.ComponentCount: &model.Component{},
	} {
		var count int64
		if err := s.db.WithContext(ctx).Model(item).Where("application_id = ?", id).Count(&count).Error; err != nil {
			return nil, err
		}
		*target = int(count)
	}
	var dependencyCount int64
	if err := s.db.WithContext(ctx).Model(&model.Dependency{}).Where("source_application_id = ?", id).Count(&dependencyCount).Error; err != nil {
		return nil, err
	}
	summary.DependencyCount = int(dependencyCount)
	detail := &ApplicationDetail{ApplicationSummary: summary}
	if app.EnvironmentID > 0 {
		var environment model.Environment
		if err := s.db.WithContext(ctx).First(&environment, app.EnvironmentID).Error; err != nil {
			return nil, err
		}
		var applicationCount int64
		if err := s.db.WithContext(ctx).Model(&model.Application{}).Where("environment_id = ?", environment.ID).Count(&applicationCount).Error; err != nil {
			return nil, err
		}
		environment.ApplicationCount = int(applicationCount)
		detail.Environment = &environment
		detail.Environments = []model.Environment{environment}
	}
	if err := s.db.WithContext(ctx).Where("application_id = ?", id).Order("is_primary DESC, id DESC").Find(&detail.Domains).Error; err != nil {
		return nil, err
	}
	if err := s.enrichDomains(ctx, detail.Domains); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("application_id = ?", id).Order("kind, id").Find(&detail.Resources).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("application_id = ?", id).Order("category, id").Find(&detail.Components).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("source_application_id = ?", id).Order("id DESC").Find(&detail.Dependencies).Error; err != nil {
		return nil, err
	}
	return detail, nil
}

func (s *Service) CreateEnvironment(ctx context.Context, in *EnvironmentInput, userID uint) (*model.Environment, error) {
	if err := s.validateEnvironmentInput(in); err != nil {
		return nil, err
	}
	var duplicate int64
	if err := s.db.WithContext(ctx).Model(&model.Environment{}).Where("LOWER(code) = LOWER(?)", in.Code).Count(&duplicate).Error; err != nil {
		return nil, err
	}
	if duplicate > 0 {
		return nil, fmt.Errorf("环境编码 %q 已存在", in.Code)
	}
	env := &model.Environment{Code: in.Code, Name: in.Name, Kind: in.Kind, Region: strings.TrimSpace(in.Region), Status: in.Status, Description: strings.TrimSpace(in.Description), CreatedBy: userID, UpdatedBy: userID}
	if err := s.db.WithContext(ctx).Create(env).Error; err != nil {
		return nil, err
	}
	return env, nil
}

func (s *Service) UpdateEnvironment(ctx context.Context, id uint, in *EnvironmentInput, userID uint) (*model.Environment, error) {
	var env model.Environment
	if err := s.db.WithContext(ctx).First(&env, id).Error; err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Code) == "" {
		in.Code = env.Code
	}
	if in.Kind == "" {
		in.Kind = env.Kind
	}
	if in.Status == "" {
		in.Status = env.Status
	}
	if err := s.validateEnvironmentInput(in); err != nil {
		return nil, err
	}
	var duplicate int64
	if err := s.db.WithContext(ctx).Model(&model.Environment{}).Where("LOWER(code) = LOWER(?) AND id <> ?", in.Code, id).Count(&duplicate).Error; err != nil {
		return nil, err
	}
	if duplicate > 0 {
		return nil, fmt.Errorf("环境编码 %q 已存在", strings.TrimSpace(in.Code))
	}
	var applicationCount int64
	if err := s.db.WithContext(ctx).Model(&model.Application{}).Where("environment_id = ?", id).Count(&applicationCount).Error; err != nil {
		return nil, err
	}
	if in.Status == "disabled" && applicationCount > 0 {
		return nil, fmt.Errorf("该环境仍被 %d 个应用使用，不能停用", applicationCount)
	}
	env.Code, env.Name, env.Kind = in.Code, in.Name, in.Kind
	env.Region, env.Status, env.Description, env.UpdatedBy = strings.TrimSpace(in.Region), in.Status, strings.TrimSpace(in.Description), userID
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&env).Error; err != nil {
			return err
		}
		return tx.Model(&model.Application{}).Where("environment_id = ?", env.ID).Update("lifecycle", lifecycleFromEnvironment(env.Kind)).Error
	}); err != nil {
		return nil, err
	}
	return &env, nil
}

func (s *Service) DeleteEnvironment(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var env model.Environment
		if err := tx.First(&env, id).Error; err != nil {
			return err
		}
		var applicationCount int64
		if err := tx.Model(&model.Application{}).Where("environment_id = ?", id).Count(&applicationCount).Error; err != nil {
			return err
		}
		if applicationCount > 0 {
			return fmt.Errorf("该环境仍被 %d 个应用使用，不能删除", applicationCount)
		}
		for _, relation := range []struct {
			model interface{}
			field string
		}{
			{model: &model.Domain{}, field: "environment_id"},
			{model: &model.Resource{}, field: "environment_id"},
			{model: &model.Component{}, field: "environment_id"},
			{model: &model.Dependency{}, field: "source_environment_id"},
		} {
			var count int64
			if err := tx.Model(relation.model).Where(relation.field+" = ?", id).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return errors.New("环境仍关联域名、资源、组件或调用依赖，不能删除")
			}
		}
		return tx.Delete(&env).Error
	})
}

func (s *Service) ListEnvironments(ctx context.Context, appID uint) ([]model.Environment, error) {
	var items []model.Environment
	q := s.db.WithContext(ctx).Model(&model.Environment{}).Order("kind, id")
	if appID > 0 {
		var app model.Application
		if err := s.db.WithContext(ctx).Select("environment_id").First(&app, appID).Error; err != nil {
			return nil, err
		}
		if app.EnvironmentID == 0 {
			return items, nil
		}
		q = q.Where("id = ?", app.EnvironmentID)
	}
	if err := q.Find(&items).Error; err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return items, nil
	}
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	var counts []struct {
		EnvironmentID uint
		Count         int
	}
	if err := s.db.WithContext(ctx).Model(&model.Application{}).Select("environment_id, COUNT(*) AS count").Where("environment_id IN ?", ids).Group("environment_id").Find(&counts).Error; err != nil {
		return nil, err
	}
	countMap := make(map[uint]int, len(counts))
	for _, count := range counts {
		countMap[count.EnvironmentID] = count.Count
	}
	for i := range items {
		items[i].ApplicationCount = countMap[items[i].ID]
	}
	return items, nil
}

func (s *Service) CreateDomain(ctx context.Context, in *DomainInput) (*model.Domain, error) {
	environmentID, err := s.applicationEnvironmentID(ctx, in.ApplicationID)
	if err != nil {
		return nil, err
	}
	in.Domain = strings.TrimSpace(in.Domain)
	if in.Domain == "" {
		return nil, errors.New("域名不能为空")
	}
	in.Protocol = strings.ToLower(strings.TrimSpace(in.Protocol))
	if in.Protocol == "" {
		in.Protocol = "https"
	}
	if in.Port == 0 {
		if in.Protocol == "https" {
			in.Port = 443
		} else {
			in.Port = 80
		}
	}
	if in.Path == "" {
		in.Path = "/"
	}
	in.Source = "manual"
	if err := validateDomainInput(in); err != nil {
		return nil, err
	}
	if err := s.ensureCertificate(ctx, in.CertificateID); err != nil {
		return nil, err
	}
	item := &model.Domain{ApplicationID: in.ApplicationID, EnvironmentID: environmentID, Domain: in.Domain, Protocol: in.Protocol, Port: in.Port, Path: in.Path, CertificateID: in.CertificateID, IsPrimary: in.IsPrimary, Status: "checking", Source: in.Source, Description: strings.TrimSpace(in.Description)}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var duplicate int64
		if err := tx.Model(&model.Domain{}).Where("application_id = ? AND environment_id = ? AND domain = ? AND protocol = ? AND port = ? AND path = ?", item.ApplicationID, item.EnvironmentID, item.Domain, item.Protocol, item.Port, item.Path).Count(&duplicate).Error; err != nil {
			return err
		}
		if duplicate > 0 {
			return fmt.Errorf("域名入口 %q 已存在", item.Domain+item.Path)
		}
		if item.IsPrimary {
			if err := tx.Model(&model.Domain{}).Where("application_id = ? AND environment_id = ?", item.ApplicationID, item.EnvironmentID).Update("is_primary", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(item).Error
	}); err != nil {
		return nil, err
	}
	if probed, err := s.ProbeDomain(ctx, item.ID); err == nil {
		item = probed
	}
	return item, nil
}

func (s *Service) UpdateDomain(ctx context.Context, id uint, in *DomainInput) (*model.Domain, error) {
	var item model.Domain
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	if in.ApplicationID == 0 {
		in.ApplicationID = item.ApplicationID
	}
	if in.ApplicationID != item.ApplicationID {
		return nil, errors.New("域名所属应用创建后不可修改")
	}
	environmentID, err := s.applicationEnvironmentID(ctx, in.ApplicationID)
	if err != nil {
		return nil, err
	}
	in.Protocol = strings.ToLower(strings.TrimSpace(in.Protocol))
	if in.Protocol == "" {
		in.Protocol = "https"
	}
	if in.Port == 0 {
		if in.Protocol == "https" {
			in.Port = 443
		} else {
			in.Port = 80
		}
	}
	if in.Path == "" {
		in.Path = "/"
	}
	if err := validateDomainInput(in); err != nil {
		return nil, err
	}
	if err := s.ensureCertificate(ctx, in.CertificateID); err != nil {
		return nil, err
	}
	item.ApplicationID, item.EnvironmentID, item.Domain = in.ApplicationID, environmentID, in.Domain
	item.Protocol, item.Port, item.Path = in.Protocol, in.Port, in.Path
	item.CertificateID, item.IsPrimary = in.CertificateID, in.IsPrimary
	item.Description, item.Status = strings.TrimSpace(in.Description), "checking"
	if item.Source == "" {
		item.Source = "manual"
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var duplicate int64
		if err := tx.Model(&model.Domain{}).Where("application_id = ? AND environment_id = ? AND domain = ? AND protocol = ? AND port = ? AND path = ? AND id <> ?", item.ApplicationID, item.EnvironmentID, item.Domain, item.Protocol, item.Port, item.Path, item.ID).Count(&duplicate).Error; err != nil {
			return err
		}
		if duplicate > 0 {
			return fmt.Errorf("域名入口 %q 已存在", item.Domain+item.Path)
		}
		if item.IsPrimary {
			if err := tx.Model(&model.Domain{}).Where("application_id = ? AND environment_id = ? AND id <> ?", item.ApplicationID, item.EnvironmentID, item.ID).Update("is_primary", false).Error; err != nil {
				return err
			}
		}
		return tx.Save(&item).Error
	}); err != nil {
		return nil, err
	}
	if probed, err := s.ProbeDomain(ctx, item.ID); err == nil {
		item = *probed
	}
	return &item, nil
}

func (s *Service) DeleteDomain(ctx context.Context, id uint) error {
	var item model.Domain
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Delete(&item).Error; err != nil {
		return err
	}
	return s.RecomputeApplicationHealth(ctx, item.ApplicationID)
}

func (s *Service) ListDomains(ctx context.Context, opts ListOptions) (Page[model.Domain], error) {
	page, pageSize := normalizePage(opts)
	q := s.db.WithContext(ctx).Model(&model.Domain{})
	if opts.AppID > 0 {
		q = q.Where("application_id = ?", opts.AppID)
	}
	if opts.EnvID > 0 {
		q = q.Where("environment_id = ?", opts.EnvID)
	}
	if opts.Keyword != "" {
		like := "%" + strings.TrimSpace(opts.Keyword) + "%"
		q = q.Where("domain LIKE ? OR description LIKE ?", like, like)
	}
	if opts.Status != "" {
		q = q.Where("status = ?", opts.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return Page[model.Domain]{}, err
	}
	var items []model.Domain
	if err := q.Order("is_primary DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return Page[model.Domain]{}, err
	}
	if err := s.enrichDomains(ctx, items); err != nil {
		return Page[model.Domain]{}, err
	}
	return Page[model.Domain]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) CreateResource(ctx context.Context, in *ResourceInput, userID uint) (*model.Resource, error) {
	environmentID, err := s.applicationEnvironmentID(ctx, in.ApplicationID)
	if err != nil {
		return nil, err
	}
	if err := validateResourceInput(in); err != nil {
		return nil, err
	}
	if err := s.validateCredentialUse(ctx, in.CredentialID, userID); err != nil {
		return nil, err
	}
	if err := s.resolveResourceLocation(ctx, in, userID); err != nil {
		return nil, err
	}
	item := &model.Resource{ApplicationID: in.ApplicationID, EnvironmentID: environmentID, Kind: strings.TrimSpace(in.Kind), Name: strings.TrimSpace(in.Name), Address: strings.TrimSpace(in.Address), Port: in.Port, HostID: in.HostID, ClusterID: in.ClusterID, Namespace: strings.TrimSpace(in.Namespace), ExternalID: strings.TrimSpace(in.ExternalID), CredentialID: in.CredentialID, Status: "checking", Source: "manual", Description: strings.TrimSpace(in.Description)}
	if err := s.db.WithContext(ctx).Create(item).Error; err != nil {
		return nil, err
	}
	if probed, err := s.ProbeResource(ctx, item.ID); err == nil {
		item = probed
	}
	return item, nil
}

func (s *Service) UpdateResource(ctx context.Context, id uint, in *ResourceInput, userID uint) (*model.Resource, error) {
	var item model.Resource
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	if in.ApplicationID == 0 {
		in.ApplicationID = item.ApplicationID
	}
	if in.ApplicationID != item.ApplicationID {
		return nil, errors.New("资源所属应用创建后不可修改")
	}
	if strings.TrimSpace(in.Kind) != item.Kind {
		return nil, errors.New("资源类型创建后不可修改，请删除后重新登记")
	}
	environmentID, err := s.applicationEnvironmentID(ctx, in.ApplicationID)
	if err != nil {
		return nil, err
	}
	if err := validateResourceInput(in); err != nil {
		return nil, err
	}
	if err := s.validateCredentialUse(ctx, in.CredentialID, userID); err != nil {
		return nil, err
	}
	if err := s.resolveResourceLocation(ctx, in, userID); err != nil {
		return nil, err
	}
	item.EnvironmentID = environmentID
	item.Kind, item.Name, item.Address, item.Port = strings.TrimSpace(in.Kind), strings.TrimSpace(in.Name), strings.TrimSpace(in.Address), in.Port
	item.HostID, item.ClusterID, item.Namespace, item.ExternalID = in.HostID, in.ClusterID, strings.TrimSpace(in.Namespace), strings.TrimSpace(in.ExternalID)
	item.CredentialID, item.Status = in.CredentialID, "checking"
	item.Description = strings.TrimSpace(in.Description)
	if item.Source == "" {
		item.Source = "manual"
	}
	if err := s.db.WithContext(ctx).Save(&item).Error; err != nil {
		return nil, err
	}
	if probed, err := s.ProbeResource(ctx, item.ID); err == nil {
		item = *probed
	}
	return &item, nil
}

func (s *Service) DeleteResource(ctx context.Context, id uint) error {
	var item model.Resource
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return err
	}
	var references int64
	if err := s.db.WithContext(ctx).Model(&model.Dependency{}).Where("target_resource_id = ?", id).Count(&references).Error; err != nil {
		return err
	}
	if references > 0 {
		return errors.New("资源仍被调用依赖引用，不能删除")
	}
	if err := s.db.WithContext(ctx).Delete(&item).Error; err != nil {
		return err
	}
	return s.RecomputeApplicationHealth(ctx, item.ApplicationID)
}

func (s *Service) ListResources(ctx context.Context, opts ListOptions) (Page[model.Resource], error) {
	page, pageSize := normalizePage(opts)
	q := s.db.WithContext(ctx).Model(&model.Resource{})
	if opts.AppID > 0 {
		q = q.Where("application_id = ?", opts.AppID)
	}
	if opts.EnvID > 0 {
		q = q.Where("environment_id = ?", opts.EnvID)
	}
	if opts.Kind != "" {
		q = q.Where("kind = ?", opts.Kind)
	}
	if opts.Status != "" {
		q = q.Where("status = ?", opts.Status)
	}
	if opts.Keyword != "" {
		like := "%" + strings.TrimSpace(opts.Keyword) + "%"
		q = q.Where("name LIKE ? OR address LIKE ? OR namespace LIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return Page[model.Resource]{}, err
	}
	var items []model.Resource
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return Page[model.Resource]{}, err
	}
	return Page[model.Resource]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) CreateComponent(ctx context.Context, in *ComponentInput, userID uint) (*model.Component, error) {
	environmentID, err := s.applicationEnvironmentID(ctx, in.ApplicationID)
	if err != nil {
		return nil, err
	}
	if err := validateComponentInput(in); err != nil {
		return nil, err
	}
	if err := s.validateCredentialUse(ctx, in.CredentialID, userID); err != nil {
		return nil, err
	}
	item := &model.Component{ApplicationID: in.ApplicationID, EnvironmentID: environmentID, Category: strings.TrimSpace(in.Category), Type: strings.TrimSpace(in.Type), Name: strings.TrimSpace(in.Name), Address: strings.TrimSpace(in.Address), Port: in.Port, DatabaseName: strings.TrimSpace(in.DatabaseName), Version: strings.TrimSpace(in.Version), CredentialID: in.CredentialID, TLSEnabled: in.TLSEnabled, Status: "checking", Source: "manual", Description: strings.TrimSpace(in.Description)}
	if err := s.db.WithContext(ctx).Create(item).Error; err != nil {
		return nil, err
	}
	if probed, err := s.ProbeComponent(ctx, item.ID); err == nil {
		item = probed
	}
	return item, nil
}

func (s *Service) UpdateComponent(ctx context.Context, id uint, in *ComponentInput, userID uint) (*model.Component, error) {
	var item model.Component
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	if in.ApplicationID == 0 {
		in.ApplicationID = item.ApplicationID
	}
	if in.ApplicationID != item.ApplicationID {
		return nil, errors.New("组件所属应用创建后不可修改")
	}
	environmentID, err := s.applicationEnvironmentID(ctx, in.ApplicationID)
	if err != nil {
		return nil, err
	}
	if err := validateComponentInput(in); err != nil {
		return nil, err
	}
	if err := s.validateCredentialUse(ctx, in.CredentialID, userID); err != nil {
		return nil, err
	}
	item.EnvironmentID = environmentID
	item.Category, item.Type, item.Name, item.Address, item.Port = strings.TrimSpace(in.Category), strings.TrimSpace(in.Type), strings.TrimSpace(in.Name), strings.TrimSpace(in.Address), in.Port
	item.DatabaseName, item.Version, item.CredentialID, item.TLSEnabled = strings.TrimSpace(in.DatabaseName), strings.TrimSpace(in.Version), in.CredentialID, in.TLSEnabled
	item.Status, item.Description = "checking", strings.TrimSpace(in.Description)
	if item.Source == "" {
		item.Source = "manual"
	}
	if err := s.db.WithContext(ctx).Save(&item).Error; err != nil {
		return nil, err
	}
	if probed, err := s.ProbeComponent(ctx, item.ID); err == nil {
		item = *probed
	}
	return &item, nil
}

func (s *Service) DeleteComponent(ctx context.Context, id uint) error {
	var item model.Component
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return err
	}
	var references int64
	if err := s.db.WithContext(ctx).Model(&model.Dependency{}).Where("target_component_id = ?", id).Count(&references).Error; err != nil {
		return err
	}
	if references > 0 {
		return errors.New("组件仍被调用依赖引用，不能删除")
	}
	if err := s.db.WithContext(ctx).Delete(&item).Error; err != nil {
		return err
	}
	return s.RecomputeApplicationHealth(ctx, item.ApplicationID)
}

func (s *Service) ListComponents(ctx context.Context, opts ListOptions) (Page[model.Component], error) {
	page, pageSize := normalizePage(opts)
	q := s.db.WithContext(ctx).Model(&model.Component{})
	if opts.AppID > 0 {
		q = q.Where("application_id = ?", opts.AppID)
	}
	if opts.EnvID > 0 {
		q = q.Where("environment_id = ?", opts.EnvID)
	}
	if opts.Category != "" {
		q = q.Where("category = ?", opts.Category)
	}
	if opts.Status != "" {
		q = q.Where("status = ?", opts.Status)
	}
	if opts.Keyword != "" {
		like := "%" + strings.TrimSpace(opts.Keyword) + "%"
		q = q.Where("name LIKE ? OR type LIKE ? OR address LIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return Page[model.Component]{}, err
	}
	var items []model.Component
	if err := q.Order("category, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return Page[model.Component]{}, err
	}
	return Page[model.Component]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) CreateDependency(ctx context.Context, in *DependencyInput) (*model.Dependency, error) {
	environmentID, err := s.applicationEnvironmentID(ctx, in.SourceApplicationID)
	if err != nil {
		return nil, err
	}
	in.SourceEnvironmentID = environmentID
	if err := validateDependency(in); err != nil {
		return nil, err
	}
	if err := s.validateDependencyTarget(ctx, in); err != nil {
		return nil, err
	}
	if err := s.ensureUniqueDependency(ctx, in, 0); err != nil {
		return nil, err
	}
	item := dependencyFromInput(in)
	if err := s.db.WithContext(ctx).Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) UpdateDependency(ctx context.Context, id uint, in *DependencyInput) (*model.Dependency, error) {
	var item model.Dependency
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	if in.SourceApplicationID == 0 {
		in.SourceApplicationID = item.SourceApplicationID
	}
	if in.SourceApplicationID != item.SourceApplicationID {
		return nil, errors.New("依赖来源应用创建后不可修改")
	}
	environmentID, err := s.applicationEnvironmentID(ctx, in.SourceApplicationID)
	if err != nil {
		return nil, err
	}
	in.SourceEnvironmentID = environmentID
	if err := validateDependency(in); err != nil {
		return nil, err
	}
	if err := s.validateDependencyTarget(ctx, in); err != nil {
		return nil, err
	}
	if err := s.ensureUniqueDependency(ctx, in, item.ID); err != nil {
		return nil, err
	}
	updated := dependencyFromInput(in)
	updated.ID = item.ID
	updated.CreatedAt = item.CreatedAt
	updated.DeletedAt = item.DeletedAt
	if err := s.db.WithContext(ctx).Save(updated).Error; err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) DeleteDependency(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&model.Dependency{}, id).Error
}

func (s *Service) ListDependencies(ctx context.Context, opts ListOptions) (Page[model.Dependency], error) {
	page, pageSize := normalizePage(opts)
	q := s.db.WithContext(ctx).Model(&model.Dependency{})
	if opts.AppID > 0 {
		q = q.Where("source_application_id = ? OR target_application_id = ?", opts.AppID, opts.AppID)
	}
	if opts.Status != "" {
		q = q.Where("status = ?", opts.Status)
	}
	if opts.Keyword != "" {
		like := "%" + strings.TrimSpace(opts.Keyword) + "%"
		q = q.Where("target_name LIKE ? OR endpoint LIKE ? OR relation_type LIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return Page[model.Dependency]{}, err
	}
	var items []model.Dependency
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return Page[model.Dependency]{}, err
	}
	return Page[model.Dependency]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func validateDependency(in *DependencyInput) error {
	in.TargetName = strings.TrimSpace(in.TargetName)
	in.RelationType = strings.ToLower(strings.TrimSpace(in.RelationType))
	in.Protocol = strings.TrimSpace(in.Protocol)
	in.Endpoint = strings.TrimSpace(in.Endpoint)
	in.Criticality = strings.ToLower(strings.TrimSpace(in.Criticality))
	in.Status = strings.ToLower(strings.TrimSpace(in.Status))
	in.Description = strings.TrimSpace(in.Description)
	if in.SourceApplicationID == 0 {
		return errors.New("来源应用不能为空")
	}
	if in.TargetApplicationID > 0 && in.TargetApplicationID == in.SourceApplicationID {
		return errors.New("来源应用不能依赖自身")
	}
	if in.TargetApplicationID == 0 && in.TargetComponentID == 0 && in.TargetResourceID == 0 && strings.TrimSpace(in.TargetName) == "" {
		return errors.New("目标应用、组件、资源或名称至少填写一项")
	}
	targetIDs := 0
	for _, id := range []uint{in.TargetApplicationID, in.TargetComponentID, in.TargetResourceID} {
		if id > 0 {
			targetIDs++
		}
	}
	if targetIDs > 1 {
		return errors.New("目标应用、组件和资源只能选择一项")
	}
	if targetIDs > 0 {
		in.TargetName = ""
	}
	if in.Port < 0 || in.Port > 65535 {
		return errors.New("依赖端口无效")
	}
	if in.RelationType == "" {
		in.RelationType = "http"
	}
	if in.Protocol == "" || in.Endpoint == "" {
		return errors.New("调用协议和调用地址不能为空")
	}
	if len(in.TargetName) > 180 || len(in.Protocol) > 30 || len(in.Endpoint) > 500 || len(in.Description) > 500 {
		return errors.New("依赖资料超过长度限制")
	}
	if !oneOf(in.RelationType, "http", "rpc", "database", "cache", "queue", "external") {
		return errors.New("依赖关系类型无效")
	}
	if in.Criticality == "" {
		in.Criticality = "medium"
	}
	if in.Status == "" {
		in.Status = "active"
	}
	if !oneOf(in.Criticality, "critical", "high", "medium", "low") {
		return errors.New("依赖重要级别无效")
	}
	if !oneOf(in.Status, "active", "disabled") {
		return errors.New("依赖状态无效")
	}
	return nil
}

func validateDomainInput(in *DomainInput) error {
	in.Domain = strings.TrimSuffix(strings.TrimSpace(strings.ToLower(in.Domain)), ".")
	in.Protocol = strings.ToLower(strings.TrimSpace(in.Protocol))
	in.Path = strings.TrimSpace(in.Path)
	if in.Domain == "" {
		return errors.New("域名不能为空")
	}
	if len(in.Domain) > 253 || !validDomainHost(in.Domain) {
		return errors.New("域名格式无效")
	}
	if in.Port < 1 || in.Port > 65535 {
		return errors.New("域名端口无效")
	}
	if !oneOf(in.Protocol, "http", "https", "tcp") {
		return errors.New("域名协议无效")
	}
	if !strings.HasPrefix(in.Path, "/") {
		return errors.New("访问路径必须以 / 开头")
	}
	if len(in.Path) > 255 || len(strings.TrimSpace(in.DNSProvider)) > 80 || len(strings.TrimSpace(in.Description)) > 500 {
		return errors.New("域名资料超过长度限制")
	}
	if in.Protocol != "https" {
		in.CertificateID = 0
	}
	return nil
}

var regexpDomain = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)*$`)

func validDomainHost(value string) bool {
	if parsed := net.ParseIP(value); parsed != nil {
		return parsed.To4() != nil
	}
	return regexpDomain.MatchString(value)
}

func validateResourceInput(in *ResourceInput) error {
	in.Kind = strings.TrimSpace(in.Kind)
	in.Name = strings.TrimSpace(in.Name)
	in.Address = strings.TrimSpace(in.Address)
	in.Namespace = strings.TrimSpace(in.Namespace)
	if in.Kind == "" || in.Name == "" {
		return errors.New("资源类型和名称不能为空")
	}
	if in.Port < 0 || in.Port > 65535 {
		return errors.New("资源端口无效")
	}
	if !oneOf(in.Kind, "Host", "VirtualMachine", "Deployment", "StatefulSet", "DaemonSet", "Service", "Ingress", "Container", "Other") {
		return errors.New("资源类型无效")
	}
	if len(in.Name) > 180 || len(in.Address) > 500 || len(in.Namespace) > 120 || len(strings.TrimSpace(in.ExternalID)) > 500 || len(strings.TrimSpace(in.Description)) > 500 {
		return errors.New("资源资料超过长度限制")
	}
	if isKubernetesKind(in.Kind) {
		if in.ClusterID == 0 || in.Namespace == "" {
			return errors.New("Kubernetes 资源必须关联集群和命名空间")
		}
	} else if oneOf(in.Kind, "Host", "VirtualMachine") && in.HostID == 0 {
		return errors.New("主机类资源必须选择关联主机")
	} else if in.Kind == "Other" && in.Address == "" && strings.TrimSpace(in.ExternalID) == "" {
		return errors.New("其他资源必须填写访问地址或外部标识")
	}
	return nil
}

func validateComponentInput(in *ComponentInput) error {
	in.Category = strings.TrimSpace(in.Category)
	in.Type = strings.TrimSpace(in.Type)
	in.Name = strings.TrimSpace(in.Name)
	in.Address = strings.TrimSpace(in.Address)
	if in.Category == "" || in.Type == "" || in.Name == "" {
		return errors.New("组件分类、类型和名称不能为空")
	}
	if !oneOf(in.Category, "database", "middleware", "cache", "queue", "search", "storage", "external") {
		return errors.New("组件分类无效")
	}
	if in.Address == "" {
		return errors.New("组件连接地址不能为空")
	}
	if in.Port < 1 || in.Port > 65535 {
		return errors.New("组件端口无效")
	}
	if len(in.Type) > 60 || len(in.Name) > 150 || len(in.Address) > 500 || len(strings.TrimSpace(in.DatabaseName)) > 120 || len(strings.TrimSpace(in.Version)) > 80 || len(strings.TrimSpace(in.Description)) > 500 {
		return errors.New("组件资料超过长度限制")
	}
	return nil
}

func isKubernetesKind(kind string) bool {
	return oneOf(kind, "Deployment", "StatefulSet", "DaemonSet", "Service", "Ingress", "Container")
}

func oneOf(value string, values ...string) bool {
	for _, item := range values {
		if value == item {
			return true
		}
	}
	return false
}

func validateOptionalHTTPURL(label, value string) error {
	if value == "" {
		return nil
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || !oneOf(parsed.Scheme, "http", "https") {
		return fmt.Errorf("%s必须是完整的 HTTP 或 HTTPS 地址", label)
	}
	return nil
}

func normalizeTagsJSON(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "[]", nil
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return "", errors.New("应用标签必须是字符串数组")
	}
	if len(values) > 12 {
		return "", errors.New("应用标签最多设置 12 个")
	}
	seen := make(map[string]bool, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		if len(value) > 40 {
			return "", errors.New("单个应用标签不能超过 40 个字符")
		}
		seen[value] = true
		normalized = append(normalized, value)
	}
	encoded, _ := json.Marshal(normalized)
	return string(encoded), nil
}

func (s *Service) validateDependencyTarget(ctx context.Context, in *DependencyInput) error {
	var target interface{}
	switch {
	case in.TargetApplicationID > 0:
		target = &model.Application{}
	case in.TargetComponentID > 0:
		target = &model.Component{}
	case in.TargetResourceID > 0:
		target = &model.Resource{}
	default:
		return nil
	}
	id := in.TargetApplicationID
	label := "目标应用"
	if in.TargetComponentID > 0 {
		id, label = in.TargetComponentID, "目标组件"
	} else if in.TargetResourceID > 0 {
		id, label = in.TargetResourceID, "目标资源"
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(target).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("%s不存在", label)
	}
	return nil
}

func (s *Service) ensureUniqueDependency(ctx context.Context, in *DependencyInput, excludeID uint) error {
	query := s.db.WithContext(ctx).Model(&model.Dependency{}).Where(
		"source_application_id = ? AND target_application_id = ? AND target_component_id = ? AND target_resource_id = ? AND target_name = ? AND relation_type = ? AND protocol = ? AND endpoint = ? AND port = ?",
		in.SourceApplicationID, in.TargetApplicationID, in.TargetComponentID, in.TargetResourceID, in.TargetName, in.RelationType, in.Protocol, in.Endpoint, in.Port,
	)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("相同的调用依赖已存在")
	}
	return nil
}

func dependencyFromInput(in *DependencyInput) *model.Dependency {
	return &model.Dependency{SourceApplicationID: in.SourceApplicationID, SourceEnvironmentID: in.SourceEnvironmentID, TargetApplicationID: in.TargetApplicationID, TargetComponentID: in.TargetComponentID, TargetResourceID: in.TargetResourceID, TargetName: strings.TrimSpace(in.TargetName), RelationType: strings.TrimSpace(in.RelationType), Protocol: strings.TrimSpace(in.Protocol), Endpoint: strings.TrimSpace(in.Endpoint), Port: in.Port, Criticality: in.Criticality, Status: in.Status, Description: strings.TrimSpace(in.Description)}
}

func (s *Service) ensureCertificate(ctx context.Context, certificateID uint) error {
	if certificateID == 0 {
		return nil
	}
	var count int64
	if err := s.db.WithContext(ctx).Table("ssl_certificates").Where("id = ? AND deleted_at IS NULL", certificateID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("关联的 SSL 证书不存在")
	}
	return nil
}

func (s *Service) validateCredentialUse(ctx context.Context, credentialID, userID uint) error {
	if credentialID == 0 {
		return nil
	}
	var credential model.Credential
	if err := s.db.WithContext(ctx).First(&credential, credentialID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("关联的凭据不存在")
		}
		return err
	}
	if !credentialAvailable(&credential) {
		return errors.New("关联的凭据已停用或已过期，不能使用")
	}
	if s.isAdmin(ctx, userID) || credential.OwnerUserID == userID || s.hasCredentialPermission(ctx, credential.ID, userID, model.CredentialPermissionUse) {
		return nil
	}
	return errors.New("没有使用该凭据的权限")
}

func normalizeJSONText(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var value interface{}
	if json.Unmarshal([]byte(raw), &value) == nil {
		encoded, _ := json.Marshal(value)
		return string(encoded)
	}
	// Keep free-form metadata usable while making it valid JSON.
	encoded, _ := json.Marshal(map[string]string{"value": raw})
	return string(encoded)
}

func (s *Service) enrichDomains(ctx context.Context, domains []model.Domain) error {
	ids := make([]uint, 0, len(domains))
	for _, item := range domains {
		if item.CertificateID > 0 {
			ids = append(ids, item.CertificateID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	var certs []struct {
		ID     uint
		Name   string
		Status string
		Expiry *time.Time `gorm:"column:not_after"`
	}
	if err := s.db.WithContext(ctx).Table("ssl_certificates").Select("id, name, status, not_after").Where("id IN ?", ids).Find(&certs).Error; err != nil {
		return err
	}
	certMap := make(map[uint]struct {
		name   string
		status string
		expiry *time.Time
	}, len(certs))
	for _, cert := range certs {
		certMap[cert.ID] = struct {
			name   string
			status string
			expiry *time.Time
		}{cert.Name, cert.Status, cert.Expiry}
	}
	for i := range domains {
		if cert, ok := certMap[domains[i].CertificateID]; ok {
			domains[i].CertificateName = cert.name
			domains[i].CertificateStatus = cert.status
			domains[i].CertificateExpiry = cert.expiry
		}
	}
	return nil
}

func (s *Service) Overview(ctx context.Context) (map[string]interface{}, error) {
	counts := map[string]int64{}
	for key, item := range map[string]interface{}{
		"applications": &model.Application{},
		"environments": &model.Environment{},
		"domains":      &model.Domain{},
		"resources":    &model.Resource{},
		"components":   &model.Component{},
		"dependencies": &model.Dependency{},
		"credentials":  &model.Credential{},
	} {
		var count int64
		if err := s.db.WithContext(ctx).Model(item).Count(&count).Error; err != nil {
			return nil, err
		}
		counts[key] = count
	}
	var activeApplications, unhealthyApplications, applicationEnvironments int64
	if err := s.db.WithContext(ctx).Model(&model.Application{}).Where("status = ?", "active").Count(&activeApplications).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&model.Application{}).Where("health_status IN ?", []string{"unhealthy", "down", "error"}).Count(&unhealthyApplications).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&model.Application{}).Where("environment_id > 0").Distinct("environment_id").Count(&applicationEnvironments).Error; err != nil {
		return nil, err
	}
	var unhealthyResources, unhealthyComponents int64
	if err := s.db.WithContext(ctx).Model(&model.Resource{}).Where("status IN ?", []string{"unhealthy", "down", "error"}).Count(&unhealthyResources).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&model.Component{}).Where("status IN ?", []string{"unhealthy", "down", "error"}).Count(&unhealthyComponents).Error; err != nil {
		return nil, err
	}
	var expiringCertificates int64
	certTable := s.db.WithContext(ctx).Table("ssl_certificates").Where("not_after IS NOT NULL AND not_after <= ? AND not_after >= ?", time.Now().Add(30*24*time.Hour), time.Now())
	if err := certTable.Count(&expiringCertificates).Error; err != nil {
		// The SSL plugin may be disabled in a minimal installation.
		expiringCertificates = 0
	}
	var expiringCredentials int64
	if err := s.db.WithContext(ctx).Model(&model.Credential{}).Where("expires_at IS NOT NULL AND expires_at <= ? AND expires_at >= ?", time.Now().Add(30*24*time.Hour), time.Now()).Count(&expiringCredentials).Error; err != nil {
		return nil, err
	}
	var recent []ApplicationSummary
	recentPage, err := s.ListApplications(ctx, ListOptions{Page: 1, PageSize: 6})
	if err != nil {
		return nil, err
	}
	recent = recentPage.List
	var healthRows []struct {
		Status string
		Count  int64
	}
	if err := s.db.WithContext(ctx).Model(&model.Application{}).Select("health_status AS status, COUNT(*) AS count").Group("health_status").Find(&healthRows).Error; err != nil {
		return nil, err
	}
	health := make(map[string]int64, len(healthRows))
	for _, row := range healthRows {
		health[row.Status] = row.Count
	}
	counts["unhealthyResources"] = unhealthyResources
	counts["unhealthyComponents"] = unhealthyComponents
	counts["activeApplications"] = activeApplications
	counts["unhealthyApplications"] = unhealthyApplications
	counts["applicationEnvironments"] = applicationEnvironments
	counts["expiringCertificates"] = expiringCertificates
	counts["expiringCredentials"] = expiringCredentials
	return map[string]interface{}{"counts": counts, "health": health, "recentApplications": recent}, nil
}

// ReferenceItem is a secret-free option used by forms and discovery dialogs.
type ReferenceItem struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Code         string `json:"code,omitempty"`
	Detail       string `json:"detail,omitempty"`
	Status       string `json:"status,omitempty"`
	ParentID     uint   `json:"parentId,omitempty"`
	DepartmentID uint   `json:"departmentId,omitempty"`
	Address      string `json:"address,omitempty"`
	Port         int    `json:"port,omitempty"`
}

func (s *Service) References(ctx context.Context, userID uint, includeSecuritySubjects bool) (map[string][]ReferenceItem, error) {
	result := map[string][]ReferenceItem{
		"users": {}, "departments": {}, "hosts": {}, "clusters": {}, "certificates": {}, "roles": {},
	}
	var users []struct {
		ID           uint
		Username     string
		RealName     string
		DepartmentID uint
	}
	if err := s.db.WithContext(ctx).Table("sys_user").Select("id, username, real_name, department_id").Where("deleted_at IS NULL AND status = 1").Order("real_name, username").Find(&users).Error; err != nil {
		return nil, err
	}
	for _, item := range users {
		name := strings.TrimSpace(item.RealName)
		if name == "" {
			name = item.Username
		}
		result["users"] = append(result["users"], ReferenceItem{ID: item.ID, Name: name, Code: item.Username, Detail: item.Username, DepartmentID: item.DepartmentID, Status: "active"})
	}
	var departments []struct {
		ID       uint
		Name     string
		Code     string
		ParentID uint
	}
	if err := s.db.WithContext(ctx).Table("sys_department").Select("id, name, code, parent_id").Where("deleted_at IS NULL AND status = 1").Order("sort, id").Find(&departments).Error; err != nil {
		return nil, err
	}
	for _, item := range departments {
		result["departments"] = append(result["departments"], ReferenceItem{ID: item.ID, Name: item.Name, Code: item.Code, ParentID: item.ParentID, Status: "active"})
	}

	hostIDs, err := s.accessibleHostIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	var hosts []struct {
		ID     uint
		Name   string
		IP     string
		Port   int
		Status int
	}
	if len(hostIDs) > 0 {
		if err := s.db.WithContext(ctx).Table("hosts").Select("id, name, ip, port, status").Where("deleted_at IS NULL AND id IN ?", hostIDs).Order("name").Find(&hosts).Error; err != nil {
			return nil, err
		}
		for _, item := range hosts {
			result["hosts"] = append(result["hosts"], ReferenceItem{ID: item.ID, Name: item.Name, Detail: item.IP, Address: item.IP, Port: item.Port, Status: strconv.Itoa(item.Status)})
		}
	}
	clusterIDs, err := s.accessibleClusterIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	var clusters []struct {
		ID     uint
		Name   string
		Alias  string
		Region string
		Status int
	}
	if len(clusterIDs) > 0 {
		if err := s.db.WithContext(ctx).Table("k8s_clusters").Select("id, name, alias, region, status").Where("id IN ?", clusterIDs).Order("name").Find(&clusters).Error; err != nil {
			return nil, err
		}
		for _, item := range clusters {
			detail := firstNonEmpty(item.Alias, item.Region)
			if detail == "未命名" {
				detail = ""
			}
			result["clusters"] = append(result["clusters"], ReferenceItem{ID: item.ID, Name: item.Name, Detail: detail, Status: strconv.Itoa(item.Status)})
		}
	}
	var certs []struct {
		ID     uint
		Name   string
		Domain string
		Status string
	}
	if s.db.Migrator().HasTable("ssl_certificates") {
		if err := s.db.WithContext(ctx).Table("ssl_certificates").Select("id, name, domain, status").Where("deleted_at IS NULL").Order("name").Find(&certs).Error; err != nil {
			return nil, err
		}
		for _, item := range certs {
			result["certificates"] = append(result["certificates"], ReferenceItem{ID: item.ID, Name: item.Name, Detail: item.Domain, Status: item.Status})
		}
	}
	if includeSecuritySubjects {
		var roles []ReferenceItem
		if err := s.db.WithContext(ctx).Table("sys_role").Select("id, name, code").Where("deleted_at IS NULL AND status = 1").Order("name").Find(&roles).Error; err != nil {
			return nil, err
		}
		if len(roles) > 0 {
			result["roles"] = roles
		}
	}
	return result, nil
}
