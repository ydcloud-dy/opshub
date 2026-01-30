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

type userOAuthBindingRepo struct {
	db *gorm.DB
}

func NewUserOAuthBindingRepo(db *gorm.DB) identity.UserOAuthBindingRepo {
	return &userOAuthBindingRepo{db: db}
}

func (r *userOAuthBindingRepo) Create(ctx context.Context, binding *identity.UserOAuthBinding) error {
	return r.db.WithContext(ctx).Create(binding).Error
}

func (r *userOAuthBindingRepo) Update(ctx context.Context, binding *identity.UserOAuthBinding) error {
	return r.db.WithContext(ctx).Model(binding).Omit("created_at").Updates(binding).Error
}

func (r *userOAuthBindingRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&identity.UserOAuthBinding{}, id).Error
}

func (r *userOAuthBindingRepo) GetByID(ctx context.Context, id uint) (*identity.UserOAuthBinding, error) {
	var binding identity.UserOAuthBinding
	err := r.db.WithContext(ctx).First(&binding, id).Error
	return &binding, err
}

func (r *userOAuthBindingRepo) GetByOpenID(ctx context.Context, sourceID uint, openID string) (*identity.UserOAuthBinding, error) {
	var binding identity.UserOAuthBinding
	err := r.db.WithContext(ctx).Where("source_id = ? AND open_id = ?", sourceID, openID).First(&binding).Error
	return &binding, err
}

func (r *userOAuthBindingRepo) GetByUnionID(ctx context.Context, sourceType, unionID string) (*identity.UserOAuthBinding, error) {
	var binding identity.UserOAuthBinding
	err := r.db.WithContext(ctx).Where("source_type = ? AND union_id = ?", sourceType, unionID).First(&binding).Error
	return &binding, err
}

func (r *userOAuthBindingRepo) ListByUser(ctx context.Context, userID uint) ([]*identity.UserOAuthBinding, error) {
	var bindings []*identity.UserOAuthBinding
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&bindings).Error
	return bindings, err
}

func (r *userOAuthBindingRepo) DeleteByUser(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&identity.UserOAuthBinding{}).Error
}

func (r *userOAuthBindingRepo) DeleteByUserAndSource(ctx context.Context, userID, sourceID uint) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND source_id = ?", userID, sourceID).Delete(&identity.UserOAuthBinding{}).Error
}
