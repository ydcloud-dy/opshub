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

type userRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) rbac.UserRepo {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *rbac.SysUser) error {
	// 如果 department_id 为 0，设置为 NULL
	if user.DepartmentID == 0 {
		user.DepartmentID = 0
		return r.db.WithContext(ctx).Omit("department_id").Create(user).Error
	}
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepo) Update(ctx context.Context, user *rbac.SysUser) error {
	return r.db.WithContext(ctx).Model(user).Omit("created_at").Updates(user).Error
}

func (r *userRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&rbac.SysUser{}, id).Error
}

func (r *userRepo) GetByID(ctx context.Context, id uint) (*rbac.SysUser, error) {
	var user rbac.SysUser
	err := r.db.WithContext(ctx).Preload("Department").Preload("Roles").Preload("Positions").First(&user, id).Error
	return &user, err
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*rbac.SysUser, error) {
	var user rbac.SysUser
	// 直接使用 Preload 加载关联数据
	err := r.db.WithContext(ctx).
		Preload("Department").
		Preload("Roles").
		Preload("Positions").
		Where("username = ?", username).
		First(&user).Error
	if err != nil {
		return &user, err
	}
	return &user, nil
}

func (r *userRepo) List(ctx context.Context, page, pageSize int, keyword string, departmentID uint) ([]*rbac.SysUser, int64, error) {
	var users []*rbac.SysUser
	var total int64

	query := r.db.WithContext(ctx).Model(&rbac.SysUser{})
	if keyword != "" {
		query = query.Where("username LIKE ? OR real_name LIKE ? OR email LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if departmentID > 0 {
		query = query.Where("department_id = ?", departmentID)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Preload("Department").Preload("Roles").Preload("Positions").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Order("created_at DESC").
		Find(&users).Error

	return users, total, err
}

func (r *userRepo) AssignRoles(ctx context.Context, userID uint, roleIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除原有角色
		if err := tx.Where("user_id = ?", userID).Delete(&rbac.SysUserRole{}).Error; err != nil {
			return err
		}

		// 添加新角色
		for _, roleID := range roleIDs {
			userRole := &rbac.SysUserRole{
				UserID: userID,
				RoleID: roleID,
			}
			if err := tx.Create(userRole).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *userRepo) AssignPositions(ctx context.Context, userID uint, positionIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除原有岗位
		if err := tx.Where("user_id = ?", userID).Delete(&rbac.SysUserPosition{}).Error; err != nil {
			return err
		}

		// 添加新岗位
		for _, positionID := range positionIDs {
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

func (r *userRepo) UpdateLastLogin(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Model(&rbac.SysUser{}).
		Where("id = ?", userID).
		Update("last_login_at", gorm.Expr("NOW()")).Error
}
