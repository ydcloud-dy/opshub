package rbac

import (
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

// SysRoleAssetPermission 角色资产权限模型
// 用于配置角色对资产分组和主机的访问权限
type SysRoleAssetPermission struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	RoleID       uint           `gorm:"not null;index:idx_role_asset" json:"roleId"`        // 角色ID
	AssetGroupID uint           `gorm:"not null;index:idx_role_asset" json:"assetGroupId"` // 资产分组ID
	HostID       *uint          `gorm:"index:idx_host" json:"hostId"`                       // 主机ID（可选，为null表示整个分组）
	Permissions  uint           `gorm:"type:int unsigned;default:1;comment:操作权限位掩码：1=查看,2=编辑,4=删除,8=终端,16=文件,32=采集;index" json:"permissions"`
}

// TableName 指定表名
func (SysRoleAssetPermission) TableName() string {
	return "sys_role_asset_permission"
}

// AssetPermissionInfo 资产权限信息（用于前端展示）
type AssetPermissionInfo struct {
	ID            uint      `json:"id"`
	RoleID        uint      `json:"roleId"`
	RoleName      string    `json:"roleName"`
	RoleCode      string    `json:"roleCode"`
	AssetGroupID  uint      `json:"assetGroupId"`
	AssetGroupName string   `json:"assetGroupName"`
	HostID        *uint     `json:"hostId"`
	HostName      string    `json:"hostName,omitempty"`
	HostIP        string    `json:"hostIp,omitempty"`
	Permissions   uint      `json:"permissions"`
	CreatedAt     time.Time `json:"createdAt"`
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
	HostIDs       []uint    `json:"hostIds"`       // 指定的主机ID列表
	IsAllHosts    bool      `json:"isAllHosts"`    // 是否授权所有主机
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

