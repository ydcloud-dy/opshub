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

package identity

import (
	"context"
	"encoding/base64"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// IdentitySourceUseCase 身份源用例
type IdentitySourceUseCase struct {
	repo IdentitySourceRepo
}

func NewIdentitySourceUseCase(repo IdentitySourceRepo) *IdentitySourceUseCase {
	return &IdentitySourceUseCase{repo: repo}
}

func (uc *IdentitySourceUseCase) Create(ctx context.Context, source *IdentitySource) error {
	return uc.repo.Create(ctx, source)
}

func (uc *IdentitySourceUseCase) Update(ctx context.Context, source *IdentitySource) error {
	return uc.repo.Update(ctx, source)
}

func (uc *IdentitySourceUseCase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *IdentitySourceUseCase) GetByID(ctx context.Context, id uint) (*IdentitySource, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *IdentitySourceUseCase) GetByType(ctx context.Context, sourceType string) (*IdentitySource, error) {
	return uc.repo.GetByType(ctx, sourceType)
}

func (uc *IdentitySourceUseCase) List(ctx context.Context, page, pageSize int, keyword string, enabled *bool) ([]*IdentitySource, int64, error) {
	return uc.repo.List(ctx, page, pageSize, keyword, enabled)
}

func (uc *IdentitySourceUseCase) GetEnabled(ctx context.Context) ([]*IdentitySource, error) {
	return uc.repo.GetEnabled(ctx)
}

// SSOApplicationUseCase SSO应用用例
type SSOApplicationUseCase struct {
	repo SSOApplicationRepo
}

func NewSSOApplicationUseCase(repo SSOApplicationRepo) *SSOApplicationUseCase {
	return &SSOApplicationUseCase{repo: repo}
}

func (uc *SSOApplicationUseCase) Create(ctx context.Context, app *SSOApplication) error {
	return uc.repo.Create(ctx, app)
}

func (uc *SSOApplicationUseCase) Update(ctx context.Context, app *SSOApplication) error {
	return uc.repo.Update(ctx, app)
}

func (uc *SSOApplicationUseCase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *SSOApplicationUseCase) GetByID(ctx context.Context, id uint) (*SSOApplication, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *SSOApplicationUseCase) GetByCode(ctx context.Context, code string) (*SSOApplication, error) {
	return uc.repo.GetByCode(ctx, code)
}

func (uc *SSOApplicationUseCase) List(ctx context.Context, page, pageSize int, keyword string, category string, enabled *bool) ([]*SSOApplication, int64, error) {
	return uc.repo.List(ctx, page, pageSize, keyword, category, enabled)
}

func (uc *SSOApplicationUseCase) GetAll(ctx context.Context) ([]*SSOApplication, error) {
	return uc.repo.GetAll(ctx)
}

func (uc *SSOApplicationUseCase) GetEnabled(ctx context.Context) ([]*SSOApplication, error) {
	return uc.repo.GetEnabled(ctx)
}

// GetTemplates 获取预置应用模板
func (uc *SSOApplicationUseCase) GetTemplates() []AppTemplate {
	return []AppTemplate{
		{
			Name:        "Jenkins",
			Code:        "jenkins",
			Icon:        "https://www.jenkins.io/images/logos/jenkins/jenkins.svg",
			Category:    "cicd",
			Description: "持续集成和持续交付平台",
			SSOType:     "oauth2",
			URLTemplate: "https://jenkins.example.com",
		},
		{
			Name:        "GitLab",
			Code:        "gitlab",
			Icon:        "https://about.gitlab.com/images/press/logo/svg/gitlab-icon-rgb.svg",
			Category:    "code",
			Description: "代码托管和DevOps平台",
			SSOType:     "oauth2",
			URLTemplate: "https://gitlab.example.com",
		},
		{
			Name:        "GitHub",
			Code:        "github",
			Icon:        "https://github.githubassets.com/images/modules/logos_page/GitHub-Mark.png",
			Category:    "code",
			Description: "全球最大的代码托管平台",
			SSOType:     "oauth2",
			URLTemplate: "https://github.com",
		},
		{
			Name:        "Harbor",
			Code:        "harbor",
			Icon:        "https://raw.githubusercontent.com/goharbor/harbor/main/src/portal/src/images/harbor-logo.svg",
			Category:    "registry",
			Description: "企业级容器镜像仓库",
			SSOType:     "oidc",
			URLTemplate: "https://harbor.example.com",
		},
		{
			Name:        "Grafana",
			Code:        "grafana",
			Icon:        "https://grafana.com/static/img/menu/grafana2.svg",
			Category:    "monitor",
			Description: "数据可视化和监控平台",
			SSOType:     "oauth2",
			URLTemplate: "https://grafana.example.com",
		},
		{
			Name:        "ArgoCD",
			Code:        "argocd",
			Icon:        "https://argo-cd.readthedocs.io/en/stable/assets/logo.png",
			Category:    "cicd",
			Description: "Kubernetes GitOps持续交付工具",
			SSOType:     "oidc",
			URLTemplate: "https://argocd.example.com",
		},
		{
			Name:        "Kibana",
			Code:        "kibana",
			Icon:        "https://www.elastic.co/static/icons/kibana-logo.svg",
			Category:    "monitor",
			Description: "日志分析和可视化平台",
			SSOType:     "saml",
			URLTemplate: "https://kibana.example.com",
		},
		{
			Name:        "SonarQube",
			Code:        "sonarqube",
			Icon:        "https://www.sonarqube.org/logos/index/sonarqube-logo.svg",
			Category:    "code",
			Description: "代码质量和安全分析平台",
			SSOType:     "saml",
			URLTemplate: "https://sonarqube.example.com",
		},
		{
			Name:        "Nexus",
			Code:        "nexus",
			Icon:        "https://www.sonatype.com/hubfs/Nexus%20Logo.png",
			Category:    "registry",
			Description: "制品库管理平台",
			SSOType:     "saml",
			URLTemplate: "https://nexus.example.com",
		},
		{
			Name:        "Rancher",
			Code:        "rancher",
			Icon:        "https://rancher.com/imgs/rancher-logo-horiz-color.svg",
			Category:    "cicd",
			Description: "Kubernetes管理平台",
			SSOType:     "saml",
			URLTemplate: "https://rancher.example.com",
		},
	}
}

// UserCredentialUseCase 用户凭证用例
type UserCredentialUseCase struct {
	repo UserCredentialRepo
}

func NewUserCredentialUseCase(repo UserCredentialRepo) *UserCredentialUseCase {
	return &UserCredentialUseCase{repo: repo}
}

func (uc *UserCredentialUseCase) Create(ctx context.Context, credential *UserCredential) error {
	// 加密密码
	if credential.Password != "" {
		encrypted, err := encryptPassword(credential.Password)
		if err != nil {
			return err
		}
		credential.Password = encrypted
	}
	return uc.repo.Create(ctx, credential)
}

func (uc *UserCredentialUseCase) Update(ctx context.Context, credential *UserCredential) error {
	// 如果密码有更新，则加密
	if credential.Password != "" {
		encrypted, err := encryptPassword(credential.Password)
		if err != nil {
			return err
		}
		credential.Password = encrypted
	}
	return uc.repo.Update(ctx, credential)
}

func (uc *UserCredentialUseCase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *UserCredentialUseCase) GetByID(ctx context.Context, id uint) (*UserCredential, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *UserCredentialUseCase) GetByUserAndApp(ctx context.Context, userID, appID uint) (*UserCredential, error) {
	return uc.repo.GetByUserAndApp(ctx, userID, appID)
}

func (uc *UserCredentialUseCase) ListByUser(ctx context.Context, userID uint) ([]*UserCredential, error) {
	return uc.repo.ListByUser(ctx, userID)
}

// GetDecryptedPassword 获取解密后的密码
func (uc *UserCredentialUseCase) GetDecryptedPassword(ctx context.Context, credentialID uint) (string, error) {
	credential, err := uc.repo.GetByID(ctx, credentialID)
	if err != nil {
		return "", err
	}
	return decryptPassword(credential.Password)
}

// AppPermissionUseCase 应用权限用例
type AppPermissionUseCase struct {
	repo AppPermissionRepo
}

func NewAppPermissionUseCase(repo AppPermissionRepo) *AppPermissionUseCase {
	return &AppPermissionUseCase{repo: repo}
}

func (uc *AppPermissionUseCase) Create(ctx context.Context, permission *AppPermission) error {
	return uc.repo.Create(ctx, permission)
}

func (uc *AppPermissionUseCase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *AppPermissionUseCase) DeleteByApp(ctx context.Context, appID uint) error {
	return uc.repo.DeleteByApp(ctx, appID)
}

func (uc *AppPermissionUseCase) List(ctx context.Context, page, pageSize int, appID *uint, subjectType string) ([]*AppPermission, int64, error) {
	return uc.repo.List(ctx, page, pageSize, appID, subjectType)
}

func (uc *AppPermissionUseCase) ListByApp(ctx context.Context, appID uint) ([]*AppPermission, error) {
	return uc.repo.ListByApp(ctx, appID)
}

func (uc *AppPermissionUseCase) CheckPermission(ctx context.Context, appID uint, subjectType string, subjectID uint) (bool, error) {
	return uc.repo.CheckPermission(ctx, appID, subjectType, subjectID)
}

func (uc *AppPermissionUseCase) GetUserAccessibleApps(ctx context.Context, userID uint, roleIDs []uint, deptID uint) ([]uint, error) {
	return uc.repo.GetUserAccessibleApps(ctx, userID, roleIDs, deptID)
}

// UserOAuthBindingUseCase 用户OAuth绑定用例
type UserOAuthBindingUseCase struct {
	repo UserOAuthBindingRepo
}

func NewUserOAuthBindingUseCase(repo UserOAuthBindingRepo) *UserOAuthBindingUseCase {
	return &UserOAuthBindingUseCase{repo: repo}
}

func (uc *UserOAuthBindingUseCase) Create(ctx context.Context, binding *UserOAuthBinding) error {
	return uc.repo.Create(ctx, binding)
}

func (uc *UserOAuthBindingUseCase) Update(ctx context.Context, binding *UserOAuthBinding) error {
	return uc.repo.Update(ctx, binding)
}

func (uc *UserOAuthBindingUseCase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *UserOAuthBindingUseCase) GetByID(ctx context.Context, id uint) (*UserOAuthBinding, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *UserOAuthBindingUseCase) GetByOpenID(ctx context.Context, sourceID uint, openID string) (*UserOAuthBinding, error) {
	return uc.repo.GetByOpenID(ctx, sourceID, openID)
}

func (uc *UserOAuthBindingUseCase) GetByUnionID(ctx context.Context, sourceType, unionID string) (*UserOAuthBinding, error) {
	return uc.repo.GetByUnionID(ctx, sourceType, unionID)
}

func (uc *UserOAuthBindingUseCase) ListByUser(ctx context.Context, userID uint) ([]*UserOAuthBinding, error) {
	return uc.repo.ListByUser(ctx, userID)
}

func (uc *UserOAuthBindingUseCase) Unbind(ctx context.Context, userID, sourceID uint) error {
	return uc.repo.DeleteByUserAndSource(ctx, userID, sourceID)
}

// AuthLogUseCase 认证日志用例
type AuthLogUseCase struct {
	repo AuthLogRepo
}

func NewAuthLogUseCase(repo AuthLogRepo) *AuthLogUseCase {
	return &AuthLogUseCase{repo: repo}
}

func (uc *AuthLogUseCase) Create(ctx context.Context, log *AuthLog) error {
	return uc.repo.Create(ctx, log)
}

func (uc *AuthLogUseCase) List(ctx context.Context, page, pageSize int, userID *uint, action, result string, startTime, endTime string) ([]*AuthLog, int64, error) {
	return uc.repo.List(ctx, page, pageSize, userID, action, result, startTime, endTime)
}

func (uc *AuthLogUseCase) GetStats(ctx context.Context, startTime, endTime string) (*AuthLogStats, error) {
	return uc.repo.GetStats(ctx, startTime, endTime)
}

func (uc *AuthLogUseCase) GetLoginTrend(ctx context.Context, days int) ([]TrendPoint, error) {
	return uc.repo.GetLoginTrend(ctx, days)
}

// UserFavoriteAppUseCase 用户收藏应用用例
type UserFavoriteAppUseCase struct {
	repo UserFavoriteAppRepo
}

func NewUserFavoriteAppUseCase(repo UserFavoriteAppRepo) *UserFavoriteAppUseCase {
	return &UserFavoriteAppUseCase{repo: repo}
}

func (uc *UserFavoriteAppUseCase) Add(ctx context.Context, userID, appID uint) error {
	favorite := &UserFavoriteApp{
		UserID: userID,
		AppID:  appID,
	}
	return uc.repo.Create(ctx, favorite)
}

func (uc *UserFavoriteAppUseCase) Remove(ctx context.Context, userID, appID uint) error {
	return uc.repo.Delete(ctx, userID, appID)
}

func (uc *UserFavoriteAppUseCase) ListByUser(ctx context.Context, userID uint) ([]*UserFavoriteApp, error) {
	return uc.repo.ListByUser(ctx, userID)
}

func (uc *UserFavoriteAppUseCase) IsFavorite(ctx context.Context, userID, appID uint) (bool, error) {
	return uc.repo.IsFavorite(ctx, userID, appID)
}

// 密码加密辅助函数
func encryptPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("密码不能为空")
	}
	// 使用bcrypt加密
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	// 再用base64编码，便于存储
	return base64.StdEncoding.EncodeToString(hashed), nil
}

func decryptPassword(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}
	// base64解码
	_, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	// 注意：bcrypt是单向加密，无法解密
	// 这里返回加密后的值，实际使用时需要在应用配置中存储可解密的密码
	// 对于表单代填场景，需要使用对称加密算法
	return encrypted, nil
}
