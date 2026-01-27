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

package asset

import (
	"context"
	"github.com/ydcloud-dy/opshub/internal/biz/asset"
	"gorm.io/gorm"
)

type assetGroupRepo struct {
	db *gorm.DB
}

func NewAssetGroupRepo(db *gorm.DB) asset.AssetGroupRepo {
	return &assetGroupRepo{db: db}
}

func (r *assetGroupRepo) Create(ctx context.Context, group *asset.AssetGroup) error {
	return r.db.WithContext(ctx).Create(group).Error
}

func (r *assetGroupRepo) Update(ctx context.Context, group *asset.AssetGroup) error {
	return r.db.WithContext(ctx).Model(group).Omit("created_at").Updates(group).Error
}

func (r *assetGroupRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查是否有子分组
		var count int64
		if err := tx.Model(&asset.AssetGroup{}).Unscoped().Where("parent_id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return gorm.ErrRegistered // 存在子分组，不能删除
		}

		// 硬删除分组
		return tx.Unscoped().Delete(&asset.AssetGroup{}, id).Error
	})
}

func (r *assetGroupRepo) GetByID(ctx context.Context, id uint) (*asset.AssetGroup, error) {
	var group asset.AssetGroup
	err := r.db.WithContext(ctx).First(&group, id).Error
	return &group, err
}

func (r *assetGroupRepo) GetTree(ctx context.Context) ([]*asset.AssetGroup, error) {
	var groups []*asset.AssetGroup
	err := r.db.WithContext(ctx).Order("sort ASC").Find(&groups).Error
	if err != nil {
		return nil, err
	}

	// 统计每个分组的主机数量（排除软删除的记录）
	hostCounts := make(map[uint]int)
	var results []struct {
		GroupID uint
		Count   int64
	}
	err = r.db.WithContext(ctx).Model(&asset.Host{}).Select("group_id, COUNT(*) as count").Where("group_id > 0").Group("group_id").Scan(&results).Error
	if err == nil {
		for _, result := range results {
			hostCounts[result.GroupID] = int(result.Count)
		}
	}

	// 为每个分组设置主机数量
	for _, group := range groups {
		group.HostCount = hostCounts[group.ID]
	}

	return r.buildTree(groups, 0), nil
}

func (r *assetGroupRepo) buildTree(groups []*asset.AssetGroup, parentID uint) []*asset.AssetGroup {
	var tree []*asset.AssetGroup
	for _, group := range groups {
		if group.ParentID == parentID {
			children := r.buildTree(groups, group.ID)
			if len(children) > 0 {
				group.Children = children
				// 累加子分组的主机数量
				for _, child := range children {
					group.HostCount += child.HostCount
				}
			}
			tree = append(tree, group)
		}
	}
	return tree
}

func (r *assetGroupRepo) GetAll(ctx context.Context) ([]*asset.AssetGroup, error) {
	var groups []*asset.AssetGroup
	err := r.db.WithContext(ctx).Order("sort ASC").Find(&groups).Error
	return groups, err
}

func (r *assetGroupRepo) List(ctx context.Context, page, pageSize int, keyword string) ([]*asset.AssetGroup, int64, error) {
	var groups []*asset.AssetGroup
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.AssetGroup{})
	if keyword != "" {
		query = query.Where("name LIKE ? OR code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("sort ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&groups).Error
	return groups, total, err
}

// GetDescendantIDs 获取指定分组的所有子孙分组ID
func (r *assetGroupRepo) GetDescendantIDs(ctx context.Context, id uint) ([]uint, error) {
	var ids []uint

	// 递归获取子分组ID
	var getChildren func(uint) error
	getChildren = func(parentID uint) error {
		var children []uint
		err := r.db.WithContext(ctx).Model(&asset.AssetGroup{}).Where("parent_id = ?", parentID).Pluck("id", &children).Error
		if err != nil {
			return err
		}

		ids = append(ids, children...)
		for _, childID := range children {
			if err := getChildren(childID); err != nil {
				return err
			}
		}
		return nil
	}

	if err := getChildren(id); err != nil {
		return nil, err
	}

	return ids, nil
}
