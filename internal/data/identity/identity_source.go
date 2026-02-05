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

type identitySourceRepo struct {
	db *gorm.DB
}

func NewIdentitySourceRepo(db *gorm.DB) identity.IdentitySourceRepo {
	return &identitySourceRepo{db: db}
}

func (r *identitySourceRepo) Create(ctx context.Context, source *identity.IdentitySource) error {
	return r.db.WithContext(ctx).Create(source).Error
}

func (r *identitySourceRepo) Update(ctx context.Context, source *identity.IdentitySource) error {
	return r.db.WithContext(ctx).Model(source).Omit("created_at").Updates(source).Error
}

func (r *identitySourceRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&identity.IdentitySource{}, id).Error
}

func (r *identitySourceRepo) GetByID(ctx context.Context, id uint) (*identity.IdentitySource, error) {
	var source identity.IdentitySource
	err := r.db.WithContext(ctx).First(&source, id).Error
	return &source, err
}

func (r *identitySourceRepo) GetByType(ctx context.Context, sourceType string) (*identity.IdentitySource, error) {
	var source identity.IdentitySource
	err := r.db.WithContext(ctx).Where("type = ? AND enabled = ?", sourceType, true).First(&source).Error
	return &source, err
}

func (r *identitySourceRepo) List(ctx context.Context, page, pageSize int, keyword string, enabled *bool) ([]*identity.IdentitySource, int64, error) {
	var sources []*identity.IdentitySource
	var total int64

	query := r.db.WithContext(ctx).Model(&identity.IdentitySource{})
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
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
		Find(&sources).Error

	return sources, total, err
}

func (r *identitySourceRepo) GetEnabled(ctx context.Context) ([]*identity.IdentitySource, error) {
	var sources []*identity.IdentitySource
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Order("sort ASC").Find(&sources).Error
	return sources, err
}
