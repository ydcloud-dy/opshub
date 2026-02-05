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

type appPermissionRepo struct {
	db *gorm.DB
}

func NewAppPermissionRepo(db *gorm.DB) identity.AppPermissionRepo {
	return &appPermissionRepo{db: db}
}

func (r *appPermissionRepo) Create(ctx context.Context, permission *identity.AppPermission) error {
	return r.db.WithContext(ctx).Create(permission).Error
}

func (r *appPermissionRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&identity.AppPermission{}, id).Error
}

func (r *appPermissionRepo) DeleteByApp(ctx context.Context, appID uint) error {
	return r.db.WithContext(ctx).Where("app_id = ?", appID).Delete(&identity.AppPermission{}).Error
}

func (r *appPermissionRepo) GetByID(ctx context.Context, id uint) (*identity.AppPermission, error) {
	var permission identity.AppPermission
	err := r.db.WithContext(ctx).First(&permission, id).Error
	return &permission, err
}

func (r *appPermissionRepo) List(ctx context.Context, page, pageSize int, appID *uint, subjectType string) ([]*identity.AppPermission, int64, error) {
	var permissions []*identity.AppPermission
	var total int64

	query := r.db.WithContext(ctx).Model(&identity.AppPermission{})
	if appID != nil {
		query = query.Where("app_id = ?", *appID)
	}
	if subjectType != "" {
		query = query.Where("subject_type = ?", subjectType)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&permissions).Error

	return permissions, total, err
}

func (r *appPermissionRepo) ListByApp(ctx context.Context, appID uint) ([]*identity.AppPermission, error) {
	var permissions []*identity.AppPermission
	err := r.db.WithContext(ctx).Where("app_id = ?", appID).Find(&permissions).Error
	return permissions, err
}

func (r *appPermissionRepo) ListBySubject(ctx context.Context, subjectType string, subjectID uint) ([]*identity.AppPermission, error) {
	var permissions []*identity.AppPermission
	err := r.db.WithContext(ctx).Where("subject_type = ? AND subject_id = ?", subjectType, subjectID).Find(&permissions).Error
	return permissions, err
}

func (r *appPermissionRepo) CheckPermission(ctx context.Context, appID uint, subjectType string, subjectID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&identity.AppPermission{}).
		Where("app_id = ? AND subject_type = ? AND subject_id = ?", appID, subjectType, subjectID).
		Count(&count).Error
	return count > 0, err
}

func (r *appPermissionRepo) GetUserAccessibleApps(ctx context.Context, userID uint, roleIDs []uint, deptID uint) ([]uint, error) {
	var appIDs []uint

	// 查询用户直接拥有的权限
	var userAppIDs []uint
	r.db.WithContext(ctx).Model(&identity.AppPermission{}).
		Where("subject_type = ? AND subject_id = ?", "user", userID).
		Pluck("app_id", &userAppIDs)

	appIDs = append(appIDs, userAppIDs...)

	// 查询用户角色拥有的权限
	if len(roleIDs) > 0 {
		var roleAppIDs []uint
		r.db.WithContext(ctx).Model(&identity.AppPermission{}).
			Where("subject_type = ? AND subject_id IN ?", "role", roleIDs).
			Pluck("app_id", &roleAppIDs)
		appIDs = append(appIDs, roleAppIDs...)
	}

	// 查询用户部门拥有的权限
	if deptID > 0 {
		var deptAppIDs []uint
		r.db.WithContext(ctx).Model(&identity.AppPermission{}).
			Where("subject_type = ? AND subject_id = ?", "dept", deptID).
			Pluck("app_id", &deptAppIDs)
		appIDs = append(appIDs, deptAppIDs...)
	}

	// 去重
	appIDMap := make(map[uint]bool)
	var uniqueAppIDs []uint
	for _, id := range appIDs {
		if !appIDMap[id] {
			appIDMap[id] = true
			uniqueAppIDs = append(uniqueAppIDs, id)
		}
	}

	return uniqueAppIDs, nil
}
