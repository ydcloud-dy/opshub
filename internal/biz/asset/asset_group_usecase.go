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
)

type AssetGroupUseCase struct {
	groupRepo AssetGroupRepo
}

func NewAssetGroupUseCase(groupRepo AssetGroupRepo) *AssetGroupUseCase {
	return &AssetGroupUseCase{
		groupRepo: groupRepo,
	}
}

func (uc *AssetGroupUseCase) Create(ctx context.Context, group *AssetGroup) error {
	return uc.groupRepo.Create(ctx, group)
}

func (uc *AssetGroupUseCase) Update(ctx context.Context, group *AssetGroup) error {
	return uc.groupRepo.Update(ctx, group)
}

func (uc *AssetGroupUseCase) Delete(ctx context.Context, id uint) error {
	return uc.groupRepo.Delete(ctx, id)
}

func (uc *AssetGroupUseCase) GetByID(ctx context.Context, id uint) (*AssetGroup, error) {
	return uc.groupRepo.GetByID(ctx, id)
}

func (uc *AssetGroupUseCase) GetTree(ctx context.Context) ([]*AssetGroup, error) {
	return uc.groupRepo.GetTree(ctx)
}

// GetParentOptions 获取父级分组选项（用于级联选择器）
func (uc *AssetGroupUseCase) GetParentOptions(ctx context.Context) ([]*AssetGroupParentOptionVO, error) {
	groups, err := uc.groupRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	return uc.buildParentOptions(groups, 0), nil
}

func (uc *AssetGroupUseCase) buildParentOptions(groups []*AssetGroup, parentID uint) []*AssetGroupParentOptionVO {
	var options []*AssetGroupParentOptionVO
	for _, group := range groups {
		if group.ParentID == parentID {
			option := &AssetGroupParentOptionVO{
				ID:       group.ID,
				ParentID: group.ParentID,
				Label:    group.Name,
			}
			children := uc.buildParentOptions(groups, group.ID)
			if len(children) > 0 {
				option.Children = children
			}
			options = append(options, option)
		}
	}
	return options
}

// ToInfoVO 将AssetGroup转换为AssetGroupInfoVO
func (uc *AssetGroupUseCase) ToInfoVO(group *AssetGroup) *AssetGroupInfoVO {
	if group == nil {
		return nil
	}
	vo := &AssetGroupInfoVO{
		ID:          group.ID,
		ParentID:    group.ParentID,
		Name:        group.Name,
		Code:        group.Code,
		Description: group.Description,
		Sort:        group.Sort,
		Status:      group.Status,
		HostCount:   group.HostCount,
		CreateTime:  group.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if len(group.Children) > 0 {
		for _, child := range group.Children {
			vo.Children = append(vo.Children, uc.ToInfoVO(child))
		}
	}
	return vo
}
