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

import "context"

// IdentitySourceRepo 身份源仓库接口
type IdentitySourceRepo interface {
	Create(ctx context.Context, source *IdentitySource) error
	Update(ctx context.Context, source *IdentitySource) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*IdentitySource, error)
	GetByType(ctx context.Context, sourceType string) (*IdentitySource, error)
	List(ctx context.Context, page, pageSize int, keyword string, enabled *bool) ([]*IdentitySource, int64, error)
	GetEnabled(ctx context.Context) ([]*IdentitySource, error)
}

// SSOApplicationRepo SSO应用仓库接口
type SSOApplicationRepo interface {
	Create(ctx context.Context, app *SSOApplication) error
	Update(ctx context.Context, app *SSOApplication) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*SSOApplication, error)
	GetByCode(ctx context.Context, code string) (*SSOApplication, error)
	List(ctx context.Context, page, pageSize int, keyword string, category string, enabled *bool) ([]*SSOApplication, int64, error)
	GetAll(ctx context.Context) ([]*SSOApplication, error)
	GetEnabled(ctx context.Context) ([]*SSOApplication, error)
}

// UserCredentialRepo 用户凭证仓库接口
type UserCredentialRepo interface {
	Create(ctx context.Context, credential *UserCredential) error
	Update(ctx context.Context, credential *UserCredential) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*UserCredential, error)
	GetByUserAndApp(ctx context.Context, userID, appID uint) (*UserCredential, error)
	ListByUser(ctx context.Context, userID uint) ([]*UserCredential, error)
	ListByApp(ctx context.Context, appID uint) ([]*UserCredential, error)
}

// AppPermissionRepo 应用权限仓库接口
type AppPermissionRepo interface {
	Create(ctx context.Context, permission *AppPermission) error
	Delete(ctx context.Context, id uint) error
	DeleteByApp(ctx context.Context, appID uint) error
	GetByID(ctx context.Context, id uint) (*AppPermission, error)
	List(ctx context.Context, page, pageSize int, appID *uint, subjectType string) ([]*AppPermission, int64, error)
	ListByApp(ctx context.Context, appID uint) ([]*AppPermission, error)
	ListBySubject(ctx context.Context, subjectType string, subjectID uint) ([]*AppPermission, error)
	CheckPermission(ctx context.Context, appID uint, subjectType string, subjectID uint) (bool, error)
	GetUserAccessibleApps(ctx context.Context, userID uint, roleIDs []uint, deptID uint) ([]uint, error)
}

// UserOAuthBindingRepo 用户OAuth绑定仓库接口
type UserOAuthBindingRepo interface {
	Create(ctx context.Context, binding *UserOAuthBinding) error
	Update(ctx context.Context, binding *UserOAuthBinding) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*UserOAuthBinding, error)
	GetByOpenID(ctx context.Context, sourceID uint, openID string) (*UserOAuthBinding, error)
	GetByUnionID(ctx context.Context, sourceType, unionID string) (*UserOAuthBinding, error)
	ListByUser(ctx context.Context, userID uint) ([]*UserOAuthBinding, error)
	DeleteByUser(ctx context.Context, userID uint) error
	DeleteByUserAndSource(ctx context.Context, userID, sourceID uint) error
}

// AuthLogRepo 认证日志仓库接口
type AuthLogRepo interface {
	Create(ctx context.Context, log *AuthLog) error
	List(ctx context.Context, page, pageSize int, userID *uint, action, result string, startTime, endTime string) ([]*AuthLog, int64, error)
	GetStats(ctx context.Context, startTime, endTime string) (*AuthLogStats, error)
	GetLoginTrend(ctx context.Context, days int) ([]TrendPoint, error)
	GetTopApps(ctx context.Context, limit int, startTime, endTime string) ([]TopAppStat, error)
	GetTopUsers(ctx context.Context, limit int, startTime, endTime string) ([]TopUserStat, error)
}

// UserFavoriteAppRepo 用户收藏应用仓库接口
type UserFavoriteAppRepo interface {
	Create(ctx context.Context, favorite *UserFavoriteApp) error
	Delete(ctx context.Context, userID, appID uint) error
	ListByUser(ctx context.Context, userID uint) ([]*UserFavoriteApp, error)
	IsFavorite(ctx context.Context, userID, appID uint) (bool, error)
}
