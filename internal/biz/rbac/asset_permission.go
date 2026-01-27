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
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Permission constants - using bitmask
const (
	PermissionView     = 1 << 0  // 1 (查看)
	PermissionEdit     = 1 << 1  // 2 (编辑)
	PermissionDelete   = 1 << 2  // 4 (删除)
	PermissionTerminal = 1 << 3  // 8 (终端)
	PermissionFile     = 1 << 4  // 16 (文件管理)
	PermissionCollect  = 1 << 5  // 32 (采集信息)
	PermissionAll      = 0x3F    // 63 (所有权限)
)

// UintArray 用于处理JSON格式的uint数组
type UintArray []uint

// Value 实现 driver.Valuer 接口，用于数据库存储
func (ua UintArray) Value() (driver.Value, error) {
	return json.Marshal(ua)
}

// Scan 实现 sql.Scanner 接口，用于从数据库读取
func (ua *UintArray) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, &ua)
}

// SysRoleAssetPermission 角色资产权限模型
// 用于配置角色对资产分组和主机的访问权限
type SysRoleAssetPermission struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	RoleID       uint           `gorm:"not null;index:idx_role_asset" json:"roleId"`        // 角色ID
	AssetGroupID uint           `gorm:"not null;index:idx_role_asset" json:"assetGroupId"` // 资产分组ID
	HostIDs      UintArray      `gorm:"type:json" json:"hostIds"`                          // 主机ID列表（为空表示整个分组）
	Permissions  uint           `gorm:"type:int unsigned;default:1;comment:操作权限位掩码：1=查看,2=编辑,4=删除,8=终端,16=文件,32=采集;index" json:"permissions"`
}

// TableName 指定表名
func (SysRoleAssetPermission) TableName() string {
	return "sys_role_asset_permission"
}

// AssetPermissionInfo 资产权限信息（用于前端展示）
type AssetPermissionInfo struct {
	ID             uint      `json:"id"`
	RoleID         uint      `json:"roleId"`
	RoleName       string    `json:"roleName"`
	RoleCode       string    `json:"roleCode"`
	AssetGroupID   uint      `json:"assetGroupId"`
	AssetGroupName string    `json:"assetGroupName"`
	HostIDs        []uint    `json:"hostIds"`        // 主机ID列表（为空表示整个分组）
	HostNames      []string  `json:"hostNames,omitempty"` // 主机名称列表
	IsAllHosts     bool      `json:"isAllHosts"`    // 是否授权所有主机
	Permissions    uint      `json:"permissions"`
	CreatedAt      time.Time `json:"createdAt"`
}

// AssetPermissionCreateReq 创建资产权限请求
type AssetPermissionCreateReq struct {
	RoleID       uint   `json:"roleId" binding:"required"`
	AssetGroupID uint   `json:"assetGroupId" binding:"required"`
	HostIDs      []uint `json:"hostIds"` // 空数组表示整个分组，非空表示指定主机
}

// AssetPermissionUpdateReq 更新资产权限请求
type AssetPermissionUpdateReq struct {
	ID      uint   `json:"id" binding:"required"`
	HostIDs []uint `json:"hostIds"`
}

// AssetPermissionCreateReqWithPermissions 创建资产权限请求（支持操作权限）
type AssetPermissionCreateReqWithPermissions struct {
	RoleID      uint   `json:"roleId" binding:"required"`
	AssetGroupID uint   `json:"assetGroupId" binding:"required"`
	HostIDs     []uint `json:"hostIds"` // 空数组表示整个分组，非空表示指定主机
	Permissions uint   `json:"permissions"`
}

// AssetPermissionDetailVO 资产权限详情（用于编辑）
type AssetPermissionDetailVO struct {
	ID            uint      `json:"id"`
	RoleID        uint      `json:"roleId"`
	RoleName      string    `json:"roleName"`
	AssetGroupID  uint      `json:"assetGroupId"`
	AssetGroupName string   `json:"assetGroupName"`
	HostIDs       []uint    `json:"hostIds"`       // 指定的主机ID列表（为空表示全部）
	Permissions   uint      `json:"permissions"`
	CreatedAt     time.Time `json:"createdAt"`
}

// HasPermission 检查是否具有指定权限
func (p *SysRoleAssetPermission) HasPermission(perm uint) bool {
	return (p.Permissions & perm) > 0
}

// AddPermission 添加权限
func (p *SysRoleAssetPermission) AddPermission(perm uint) {
	p.Permissions |= perm
}

// RemovePermission 移除权限
func (p *SysRoleAssetPermission) RemovePermission(perm uint) {
	p.Permissions &^= perm
}

// GetPermissionName 获取权限名称
func GetPermissionName(perm uint) string {
	switch perm {
	case PermissionView:
		return "查看"
	case PermissionEdit:
		return "编辑"
	case PermissionDelete:
		return "删除"
	case PermissionTerminal:
		return "终端"
	case PermissionFile:
		return "文件管理"
	case PermissionCollect:
		return "采集信息"
	default:
		return "未知"
	}
}

// GetAllPermissionNames 获取所有权限名称（如果该权限位包含在权限中）
func GetAllPermissionNames(permissions uint) []string {
	var names []string
	if (permissions & PermissionView) > 0 {
		names = append(names, "查看")
	}
	if (permissions & PermissionEdit) > 0 {
		names = append(names, "编辑")
	}
	if (permissions & PermissionDelete) > 0 {
		names = append(names, "删除")
	}
	if (permissions & PermissionTerminal) > 0 {
		names = append(names, "终端")
	}
	if (permissions & PermissionFile) > 0 {
		names = append(names, "文件管理")
	}
	if (permissions & PermissionCollect) > 0 {
		names = append(names, "采集信息")
	}
	return names
}

