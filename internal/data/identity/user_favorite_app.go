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
	"time"

	"github.com/ydcloud-dy/opshub/internal/biz/identity"
	"gorm.io/gorm"
)

type userFavoriteAppRepo struct {
	db *gorm.DB
}

func NewUserFavoriteAppRepo(db *gorm.DB) identity.UserFavoriteAppRepo {
	return &userFavoriteAppRepo{db: db}
}

func (r *userFavoriteAppRepo) Create(ctx context.Context, favorite *identity.UserFavoriteApp) error {
	if favorite.CreatedAt.IsZero() {
		favorite.CreatedAt = time.Now()
	}
	return r.db.WithContext(ctx).Create(favorite).Error
}

func (r *userFavoriteAppRepo) Delete(ctx context.Context, userID, appID uint) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND app_id = ?", userID, appID).Delete(&identity.UserFavoriteApp{}).Error
}

func (r *userFavoriteAppRepo) ListByUser(ctx context.Context, userID uint) ([]*identity.UserFavoriteApp, error) {
	var favorites []*identity.UserFavoriteApp
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&favorites).Error
	return favorites, err
}

func (r *userFavoriteAppRepo) IsFavorite(ctx context.Context, userID, appID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&identity.UserFavoriteApp{}).
		Where("user_id = ? AND app_id = ?", userID, appID).
		Count(&count).Error
	return count > 0, err
}
