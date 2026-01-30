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

type userCredentialRepo struct {
	db *gorm.DB
}

func NewUserCredentialRepo(db *gorm.DB) identity.UserCredentialRepo {
	return &userCredentialRepo{db: db}
}

func (r *userCredentialRepo) Create(ctx context.Context, credential *identity.UserCredential) error {
	return r.db.WithContext(ctx).Create(credential).Error
}

func (r *userCredentialRepo) Update(ctx context.Context, credential *identity.UserCredential) error {
	return r.db.WithContext(ctx).Model(credential).Omit("created_at").Updates(credential).Error
}

func (r *userCredentialRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&identity.UserCredential{}, id).Error
}

func (r *userCredentialRepo) GetByID(ctx context.Context, id uint) (*identity.UserCredential, error) {
	var credential identity.UserCredential
	err := r.db.WithContext(ctx).First(&credential, id).Error
	return &credential, err
}

func (r *userCredentialRepo) GetByUserAndApp(ctx context.Context, userID, appID uint) (*identity.UserCredential, error) {
	var credential identity.UserCredential
	err := r.db.WithContext(ctx).Where("user_id = ? AND app_id = ?", userID, appID).First(&credential).Error
	return &credential, err
}

func (r *userCredentialRepo) ListByUser(ctx context.Context, userID uint) ([]*identity.UserCredential, error) {
	var credentials []*identity.UserCredential
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&credentials).Error
	return credentials, err
}

func (r *userCredentialRepo) ListByApp(ctx context.Context, appID uint) ([]*identity.UserCredential, error) {
	var credentials []*identity.UserCredential
	err := r.db.WithContext(ctx).Where("app_id = ?", appID).Find(&credentials).Error
	return credentials, err
}
