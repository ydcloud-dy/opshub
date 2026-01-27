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

type departmentRepo struct {
	db *gorm.DB
}

func NewDepartmentRepo(db *gorm.DB) rbac.DepartmentRepo {
	return &departmentRepo{db: db}
}

func (r *departmentRepo) Create(ctx context.Context, dept *rbac.SysDepartment) error {
	return r.db.WithContext(ctx).Create(dept).Error
}

func (r *departmentRepo) Update(ctx context.Context, dept *rbac.SysDepartment) error {
	return r.db.WithContext(ctx).Model(dept).Omit("created_at").Updates(dept).Error
}

func (r *departmentRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查是否有子部门（包括软删除的）
		var count int64
		if err := tx.Model(&rbac.SysDepartment{}).Unscoped().Where("parent_id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return gorm.ErrRegistered // 存在子部门，不能删除
		}

		// 硬删除部门（包括软删除的记录）
		return tx.Unscoped().Delete(&rbac.SysDepartment{}, id).Error
	})
}

func (r *departmentRepo) GetByID(ctx context.Context, id uint) (*rbac.SysDepartment, error) {
	var dept rbac.SysDepartment
	err := r.db.WithContext(ctx).First(&dept, id).Error
	return &dept, err
}

func (r *departmentRepo) GetTree(ctx context.Context) ([]*rbac.SysDepartment, error) {
	var departments []*rbac.SysDepartment
	err := r.db.WithContext(ctx).Order("sort ASC").Find(&departments).Error
	if err != nil {
		return nil, err
	}

	// 统计每个部门的用户数量
	userCounts := make(map[uint]int)
	var results []struct {
		DepartmentID uint
		Count        int64
	}

	err = r.db.WithContext(ctx).Table("sys_user").
		Select("department_id, COUNT(*) as count").
		Where("department_id > 0").
		Group("department_id").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	for _, result := range results {
		userCounts[result.DepartmentID] = int(result.Count)
	}

	tree := r.buildTreeWithUserCount(departments, 0, userCounts)
	return tree, nil
}

func (r *departmentRepo) buildTreeWithUserCount(departments []*rbac.SysDepartment, parentID uint, userCounts map[uint]int) []*rbac.SysDepartment {
	var tree []*rbac.SysDepartment
	for _, dept := range departments {
		if dept.ParentID == parentID {
			children := r.buildTreeWithUserCount(departments, dept.ID, userCounts)
			if len(children) > 0 {
				dept.Children = children
				// 累加子部门的用户数量
				totalUserCount := userCounts[dept.ID]
				for _, child := range children {
					totalUserCount += child.UserCount
				}
				dept.UserCount = totalUserCount
			} else {
				dept.UserCount = userCounts[dept.ID]
			}
			tree = append(tree, dept)
		}
	}
	return tree
}

func (r *departmentRepo) GetAll(ctx context.Context) ([]*rbac.SysDepartment, error) {
	var departments []*rbac.SysDepartment
	err := r.db.WithContext(ctx).Order("sort ASC").Find(&departments).Error
	return departments, err
}
