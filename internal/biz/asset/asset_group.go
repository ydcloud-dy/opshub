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

import "gorm.io/gorm"

// AssetGroup 资产分组表（支持多级分组）
type AssetGroup struct {
	gorm.Model
	Name        string        `gorm:"type:varchar(100);not null;comment:分组名称" json:"name"`
	Code        string        `gorm:"type:varchar(50);uniqueIndex;comment:分组编码" json:"code"`
	ParentID    uint          `gorm:"column:parent_id;default:0;comment:父分组ID" json:"parentId"`
	Parent      *AssetGroup   `gorm:"-" json:"parent,omitempty"`
	Children    []*AssetGroup `gorm:"-" json:"children,omitempty"`
	Description string        `gorm:"type:varchar(500);comment:分组描述" json:"description"`
	Sort        int           `gorm:"type:int;default:0;comment:排序" json:"sort"`
	Status      int           `gorm:"type:tinyint;default:1;comment:状态 1:启用 0:禁用" json:"status"`
	HostCount   int           `gorm:"-" json:"hostCount"` // 主机数量（不存储在数据库）
}

// AssetGroupRequest 资产分组请求
type AssetGroupRequest struct {
	ID          uint   `json:"id"`
	ParentID    uint   `json:"parentId"`
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Code        string `json:"code" binding:"required,min=2,max=50"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
	Status      int    `json:"status" binding:"required"`
}

// ToModel 转换为AssetGroup模型
func (r *AssetGroupRequest) ToModel() *AssetGroup {
	return &AssetGroup{
		Model:       gorm.Model{ID: r.ID},
		Name:        r.Name,
		Code:        r.Code,
		ParentID:    r.ParentID,
		Description: r.Description,
		Sort:        r.Sort,
		Status:      r.Status,
	}
}

// AssetGroupInfoVO 资产分组信息VO
type AssetGroupInfoVO struct {
	ID          uint                 `json:"id"`
	ParentID    uint                 `json:"parentId"`
	Name        string               `json:"name"`
	Code        string               `json:"code"`
	Description string               `json:"description"`
	Sort        int                  `json:"sort"`
	Status      int                  `json:"status"`
	HostCount   int                  `json:"hostCount"`
	CreateTime  string               `json:"createTime"`
	Children    []*AssetGroupInfoVO  `json:"children,omitempty"`
}

// AssetGroupParentOptionVO 资产分组父级选项VO（用于级联选择器）
type AssetGroupParentOptionVO struct {
	ID       uint                        `json:"id"`
	ParentID uint                        `json:"parentId"`
	Label    string                      `json:"label"`
	Children []*AssetGroupParentOptionVO `json:"children,omitempty"`
}

// TableName 指定表名
func (AssetGroup) TableName() string {
	return "asset_group"
}
