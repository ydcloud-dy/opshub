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

import "context"

type UserRepo interface {
	Create(ctx context.Context, user *SysUser) error
	Update(ctx context.Context, user *SysUser) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*SysUser, error)
	GetByUsername(ctx context.Context, username string) (*SysUser, error)
	List(ctx context.Context, page, pageSize int, keyword string, departmentID uint) ([]*SysUser, int64, error)
	AssignRoles(ctx context.Context, userID uint, roleIDs []uint) error
	AssignPositions(ctx context.Context, userID uint, positionIDs []uint) error
	UpdateLastLogin(ctx context.Context, userID uint) error
}

type RoleRepo interface {
	Create(ctx context.Context, role *SysRole) error
	Update(ctx context.Context, role *SysRole) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*SysRole, error)
	List(ctx context.Context, page, pageSize int, keyword string) ([]*SysRole, int64, error)
	GetAll(ctx context.Context) ([]*SysRole, error)
	AssignMenus(ctx context.Context, roleID uint, menuIDs []uint) error
	GetByUserID(ctx context.Context, userID uint) ([]*SysRole, error)
}

type DepartmentRepo interface {
	Create(ctx context.Context, dept *SysDepartment) error
	Update(ctx context.Context, dept *SysDepartment) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*SysDepartment, error)
	GetTree(ctx context.Context) ([]*SysDepartment, error)
	GetAll(ctx context.Context) ([]*SysDepartment, error)
}

type MenuRepo interface {
	Create(ctx context.Context, menu *SysMenu) error
	Update(ctx context.Context, menu *SysMenu) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*SysMenu, error)
	GetTree(ctx context.Context) ([]*SysMenu, error)
	GetByUserID(ctx context.Context, userID uint) ([]*SysMenu, error)
	GetByRoleID(ctx context.Context, roleID uint) ([]*SysMenu, error)
}

type PositionRepo interface {
	Create(ctx context.Context, position *SysPosition) error
	Update(ctx context.Context, position *SysPosition) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*SysPosition, error)
	List(ctx context.Context, page, pageSize int, postCode, postName string) ([]*SysPosition, int64, error)
	GetAll(ctx context.Context) ([]*SysPosition, error)
	GetUsers(ctx context.Context, positionID uint, page, pageSize int) ([]*SysUser, int64, error)
	AssignUsers(ctx context.Context, positionID uint, userIDs []uint) error
	RemoveUser(ctx context.Context, positionID, userID uint) error
}

type AssetPermissionRepo interface {
	// 创建资产权限（批量）
	CreateBatch(ctx context.Context, roleID, assetGroupID uint, hostIDs []uint) error
	// 创建资产权限（支持操作权限）
	CreateBatchWithPermissions(ctx context.Context, roleID, assetGroupID uint, hostIDs []uint, permissions uint) error
	// 删除指定角色对指定资产分组的所有权限
	DeleteByRoleAndGroup(ctx context.Context, roleID, assetGroupID uint) error
	// 删除单个权限
	Delete(ctx context.Context, id uint) error
	// 根据ID获取权限详情（用于编辑）
	GetDetailByID(ctx context.Context, id uint) (*AssetPermissionDetailVO, error)
	// 更新权限配置（支持修改角色、分组、主机、权限）
	UpdateAssetPermission(ctx context.Context, id uint, roleID, assetGroupID uint, hostIDs []uint, permissions uint) error
	// 获取角色的所有资产权限
	GetByRoleID(ctx context.Context, roleID uint) ([]*AssetPermissionInfo, error)
	// 获取资产分组的所有权限配置
	GetByAssetGroupID(ctx context.Context, assetGroupID uint) ([]*AssetPermissionInfo, error)
	// 分页查询权限列表
	List(ctx context.Context, page, pageSize int, roleID, assetGroupID *uint) ([]*AssetPermissionInfo, int64, error)
	// 检查用户是否有访问指定主机的权限
	CheckHostPermission(ctx context.Context, userID, hostID uint) (bool, error)
	// 检查用户是否有对指定主机的特定操作权限
	CheckHostOperationPermission(ctx context.Context, userID, hostID uint, operation uint) (bool, error)
	// 获取用户对指定主机的所有操作权限
	GetUserHostPermissions(ctx context.Context, userID, hostID uint) (uint, error)
	// 获取用户有权限访问的所有主机ID列表
	GetUserAccessibleHostIDs(ctx context.Context, userID uint) ([]uint, error)
}
