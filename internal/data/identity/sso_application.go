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

	"github.com/ydcloud-dy/opshub/internal/biz/identity"
	"gorm.io/gorm"
)

type ssoApplicationRepo struct {
	db *gorm.DB
}

func NewSSOApplicationRepo(db *gorm.DB) identity.SSOApplicationRepo {
	return &ssoApplicationRepo{db: db}
}

func (r *ssoApplicationRepo) Create(ctx context.Context, app *identity.SSOApplication) error {
	return r.db.WithContext(ctx).Create(app).Error
}

func (r *ssoApplicationRepo) Update(ctx context.Context, app *identity.SSOApplication) error {
	return r.db.WithContext(ctx).Model(app).Omit("created_at").Updates(app).Error
}

func (r *ssoApplicationRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&identity.SSOApplication{}, id).Error
}

func (r *ssoApplicationRepo) GetByID(ctx context.Context, id uint) (*identity.SSOApplication, error) {
	var app identity.SSOApplication
	err := r.db.WithContext(ctx).First(&app, id).Error
	return &app, err
}

func (r *ssoApplicationRepo) GetByCode(ctx context.Context, code string) (*identity.SSOApplication, error) {
	var app identity.SSOApplication
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&app).Error
	return &app, err
}

func (r *ssoApplicationRepo) List(ctx context.Context, page, pageSize int, keyword string, category string, enabled *bool) ([]*identity.SSOApplication, int64, error) {
	var apps []*identity.SSOApplication
	var total int64

	query := r.db.WithContext(ctx).Model(&identity.SSOApplication{})
	if keyword != "" {
		query = query.Where("name LIKE ? OR code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("sort ASC, created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&apps).Error

	return apps, total, err
}

func (r *ssoApplicationRepo) GetAll(ctx context.Context) ([]*identity.SSOApplication, error) {
	var apps []*identity.SSOApplication
	err := r.db.WithContext(ctx).Order("sort ASC, created_at DESC").Find(&apps).Error
	return apps, err
}

func (r *ssoApplicationRepo) GetEnabled(ctx context.Context) ([]*identity.SSOApplication, error) {
	var apps []*identity.SSOApplication
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Order("sort ASC").Find(&apps).Error
	return apps, err
}
