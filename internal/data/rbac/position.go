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

package rbac

import (
	"context"
	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	"gorm.io/gorm"
)

type positionRepo struct {
	db *gorm.DB
}

func NewPositionRepo(db *gorm.DB) rbac.PositionRepo {
	return &positionRepo{db: db}
}

func (r *positionRepo) Create(ctx context.Context, position *rbac.SysPosition) error {
	return r.db.WithContext(ctx).Create(position).Error
}

func (r *positionRepo) Update(ctx context.Context, position *rbac.SysPosition) error {
	return r.db.WithContext(ctx).Model(position).Omit("created_at").Updates(position).Error
}

func (r *positionRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除用户岗位关联
		if err := tx.Where("position_id = ?", id).Delete(&rbac.SysUserPosition{}).Error; err != nil {
			return err
		}
		// 删除岗位
		return tx.Delete(&rbac.SysPosition{}, id).Error
	})
}

func (r *positionRepo) GetByID(ctx context.Context, id uint) (*rbac.SysPosition, error) {
	var position rbac.SysPosition
	err := r.db.WithContext(ctx).First(&position, id).Error
	return &position, err
}

func (r *positionRepo) List(ctx context.Context, page, pageSize int, postCode, postName string) ([]*rbac.SysPosition, int64, error) {
	var positions []*rbac.SysPosition
	var total int64

	query := r.db.WithContext(ctx).Model(&rbac.SysPosition{})
	if postCode != "" {
		query = query.Where("post_code LIKE ?", "%"+postCode+"%")
	}
	if postName != "" {
		query = query.Where("post_name LIKE ?", "%"+postName+"%")
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset((page - 1) * pageSize).Limit(pageSize).
		Order("created_at DESC").
		Find(&positions).Error

	return positions, total, err
}

func (r *positionRepo) GetAll(ctx context.Context) ([]*rbac.SysPosition, error) {
	var positions []*rbac.SysPosition
	err := r.db.WithContext(ctx).Where("post_status = 1").Order("created_at DESC").Find(&positions).Error
	return positions, err
}

func (r *positionRepo) GetUsers(ctx context.Context, positionID uint, page, pageSize int) ([]*rbac.SysUser, int64, error) {
	var users []*rbac.SysUser
	var total int64

	query := r.db.WithContext(ctx).Model(&rbac.SysUser{}).
		Joins("JOIN sys_user_position ON sys_user_position.user_id = sys_user.id").
		Where("sys_user_position.position_id = ?", positionID)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&users).Error

	return users, total, err
}

func (r *positionRepo) AssignUsers(ctx context.Context, positionID uint, userIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除原有用户
		if err := tx.Where("position_id = ?", positionID).Delete(&rbac.SysUserPosition{}).Error; err != nil {
			return err
		}

		// 添加新用户
		for _, userID := range userIDs {
			userPosition := &rbac.SysUserPosition{
				UserID:     userID,
				PositionID: positionID,
			}
			if err := tx.Create(userPosition).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *positionRepo) RemoveUser(ctx context.Context, positionID, userID uint) error {
	return r.db.WithContext(ctx).
		Where("position_id = ? AND user_id = ?", positionID, userID).
		Delete(&rbac.SysUserPosition{}).Error
}
