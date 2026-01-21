package rbac

import (
	"context"

	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	"gorm.io/gorm"
)

type assetPermissionRepo struct {
	db *gorm.DB
}

// NewAssetPermissionRepo 创建资产权限仓储
func NewAssetPermissionRepo(db *gorm.DB) rbac.AssetPermissionRepo {
	return &assetPermissionRepo{db: db}
}

// CreateBatch 批量创建资产权限
func (r *assetPermissionRepo) CreateBatch(ctx context.Context, roleID, assetGroupID uint, hostIDs []uint) error {
	// 先删除该角色对该资产分组的所有现有权限
	if err := r.DeleteByRoleAndGroup(ctx, roleID, assetGroupID); err != nil {
		return err
	}

	// 如果hostIDs为空，表示授权整个资产分组
	if len(hostIDs) == 0 {
		permission := &rbac.SysRoleAssetPermission{
			RoleID:       roleID,
			AssetGroupID: assetGroupID,
			HostID:       nil,
			Permissions: rbac.PermissionView, // 默认权限
		}
		return r.db.WithContext(ctx).Create(permission).Error
	}

	// 批量创建主机权限
	permissions := make([]*rbac.SysRoleAssetPermission, 0, len(hostIDs))
	for _, hostID := range hostIDs {
		hID := hostID
		permissions = append(permissions, &rbac.SysRoleAssetPermission{
			RoleID:       roleID,
			AssetGroupID: assetGroupID,
			HostID:       &hID,
			Permissions: rbac.PermissionView, // 默认权限
		})
	}

	return r.db.WithContext(ctx).Create(&permissions).Error
}

// CreateBatchWithPermissions 批量创建资产权限（支持指定操作权限）
func (r *assetPermissionRepo) CreateBatchWithPermissions(ctx context.Context, roleID, assetGroupID uint, hostIDs []uint, permissions uint) error {
	// 先删除该角色对该资产分组的所有现有权限
	if err := r.DeleteByRoleAndGroup(ctx, roleID, assetGroupID); err != nil {
		return err
	}

	// 如果权限为0，默认为查看权限
	if permissions == 0 {
		permissions = rbac.PermissionView
	}

	// 如果hostIDs为空，表示授权整个资产分组
	if len(hostIDs) == 0 {
		permission := &rbac.SysRoleAssetPermission{
			RoleID:       roleID,
			AssetGroupID: assetGroupID,
			HostID:       nil,
			Permissions: permissions,
		}
		return r.db.WithContext(ctx).Create(permission).Error
	}

	// 批量创建主机权限
	perms := make([]*rbac.SysRoleAssetPermission, 0, len(hostIDs))
	for _, hostID := range hostIDs {
		hID := hostID
		perms = append(perms, &rbac.SysRoleAssetPermission{
			RoleID:       roleID,
			AssetGroupID: assetGroupID,
			HostID:       &hID,
			Permissions: permissions,
		})
	}

	return r.db.WithContext(ctx).Create(&perms).Error
}

// DeleteByRoleAndGroup 删除指定角色对指定资产分组的所有权限
func (r *assetPermissionRepo) DeleteByRoleAndGroup(ctx context.Context, roleID, assetGroupID uint) error {
	return r.db.WithContext(ctx).
		Where("role_id = ? AND asset_group_id = ?", roleID, assetGroupID).
		Delete(&rbac.SysRoleAssetPermission{}).Error
}

// Delete 删除单个权限
func (r *assetPermissionRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&rbac.SysRoleAssetPermission{}, id).Error
}

// GetByID 根据ID获取权限详情
func (r *assetPermissionRepo) GetByID(ctx context.Context, id uint) (*rbac.SysRoleAssetPermission, error) {
	var permission rbac.SysRoleAssetPermission
	err := r.db.WithContext(ctx).First(&permission, id).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

// UpdatePermissions 更新权限配置
func (r *assetPermissionRepo) UpdatePermissions(ctx context.Context, id uint, permissions uint) error {
	return r.db.WithContext(ctx).Model(&rbac.SysRoleAssetPermission{}).
		Where("id = ?", id).
		Update("permissions", permissions).Error
}

// GetDetailByID 根据ID获取权限详情（用于编辑）
func (r *assetPermissionRepo) GetDetailByID(ctx context.Context, id uint) (*rbac.AssetPermissionDetailVO, error) {
	var permission rbac.SysRoleAssetPermission
	if err := r.db.WithContext(ctx).First(&permission, id).Error; err != nil {
		return nil, err
	}

	// 获取角色名称
	var role rbac.SysRole
	if err := r.db.WithContext(ctx).First(&role, permission.RoleID).Error; err != nil {
		return nil, err
	}

	// 获取资产分组名称
	var group struct {
		Name string
	}
	if err := r.db.WithContext(ctx).
		Table("asset_group").
		Select("name").
		Where("id = ?", permission.AssetGroupID).
		First(&group).Error; err != nil {
		return nil, err
	}

	// 获取该权限相关的所有主机ID
	var hostIDs []uint
	if permission.HostID != nil {
		// 如果是单个主机权限
		hostIDs = []uint{*permission.HostID}
	} else {
		// 如果是分组权限，查询该分组下的所有主机
		err := r.db.WithContext(ctx).
			Table("hosts").
			Select("id").
			Where("group_id = ? AND deleted_at IS NULL", permission.AssetGroupID).
			Pluck("id", &hostIDs).Error
		if err != nil {
			return nil, err
		}
	}

	isAllHosts := permission.HostID == nil

	return &rbac.AssetPermissionDetailVO{
		ID:            permission.ID,
		RoleID:        permission.RoleID,
		RoleName:      role.Name,
		AssetGroupID:  permission.AssetGroupID,
		AssetGroupName: group.Name,
		HostIDs:       hostIDs,
		IsAllHosts:    isAllHosts,
		Permissions:   permission.Permissions,
		CreatedAt:     permission.CreatedAt,
	}, nil
}

// UpdateAssetPermission 更新权限配置（支持修改角色、分组、主机、权限）
func (r *assetPermissionRepo) UpdateAssetPermission(ctx context.Context, id uint, roleID, assetGroupID uint, hostIDs []uint, permissions uint) error {
	// 首先删除该权限的旧记录
	if err := r.db.WithContext(ctx).Delete(&rbac.SysRoleAssetPermission{}, id).Error; err != nil {
		return err
	}

	// 如果权限为0，默认为查看权限
	if permissions == 0 {
		permissions = rbac.PermissionView
	}

	// 如果hostIDs为空，表示授权整个资产分组
	if len(hostIDs) == 0 {
		permission := &rbac.SysRoleAssetPermission{
			RoleID:       roleID,
			AssetGroupID: assetGroupID,
			HostID:       nil,
			Permissions: permissions,
		}
		return r.db.WithContext(ctx).Create(permission).Error
	}

	// 批量创建新的主机权限
	perms := make([]*rbac.SysRoleAssetPermission, 0, len(hostIDs))
	for _, hostID := range hostIDs {
		hID := hostID
		perms = append(perms, &rbac.SysRoleAssetPermission{
			RoleID:       roleID,
			AssetGroupID: assetGroupID,
			HostID:       &hID,
			Permissions: permissions,
		})
	}

	return r.db.WithContext(ctx).Create(&perms).Error
}

// GetByRoleID 获取角色的所有资产权限
func (r *assetPermissionRepo) GetByRoleID(ctx context.Context, roleID uint) ([]*rbac.AssetPermissionInfo, error) {
	var permissions []*rbac.AssetPermissionInfo

	err := r.db.WithContext(ctx).
		Table("sys_role_asset_permission AS p").
		Select(`
			p.id,
			p.role_id,
			r.name AS role_name,
			r.code AS role_code,
			p.asset_group_id,
			g.name AS asset_group_name,
			p.host_id,
			h.name AS host_name,
			h.ip AS host_ip,
			p.permissions,
			p.created_at
		`).
		Joins("LEFT JOIN sys_role AS r ON p.role_id = r.id").
		Joins("LEFT JOIN asset_group AS g ON p.asset_group_id = g.id").
		Joins("LEFT JOIN hosts AS h ON p.host_id = h.id").
		Where("p.role_id = ?", roleID).
		Order("p.created_at DESC").
		Find(&permissions).Error

	return permissions, err
}

// GetByAssetGroupID 获取资产分组的所有权限配置
func (r *assetPermissionRepo) GetByAssetGroupID(ctx context.Context, assetGroupID uint) ([]*rbac.AssetPermissionInfo, error) {
	var permissions []*rbac.AssetPermissionInfo

	err := r.db.WithContext(ctx).
		Table("sys_role_asset_permission AS p").
		Select(`
			p.id,
			p.role_id,
			r.name AS role_name,
			r.code AS role_code,
			p.asset_group_id,
			g.name AS asset_group_name,
			p.host_id,
			h.name AS host_name,
			h.ip AS host_ip,
			p.permissions,
			p.created_at
		`).
		Joins("LEFT JOIN sys_role AS r ON p.role_id = r.id").
		Joins("LEFT JOIN asset_group AS g ON p.asset_group_id = g.id").
		Joins("LEFT JOIN hosts AS h ON p.host_id = h.id").
		Where("p.asset_group_id = ?", assetGroupID).
		Order("p.created_at DESC").
		Find(&permissions).Error

	return permissions, err
}

// List 分页查询权限列表
func (r *assetPermissionRepo) List(ctx context.Context, page, pageSize int, roleID, assetGroupID *uint) ([]*rbac.AssetPermissionInfo, int64, error) {
	var permissions []*rbac.AssetPermissionInfo
	var total int64

	query := r.db.WithContext(ctx).
		Table("sys_role_asset_permission AS p").
		Select(`
			p.id,
			p.role_id,
			r.name AS role_name,
			r.code AS role_code,
			p.asset_group_id,
			g.name AS asset_group_name,
			p.host_id,
			h.name AS host_name,
			h.ip AS host_ip,
			p.permissions,
			p.created_at
		`).
		Joins("LEFT JOIN sys_role AS r ON p.role_id = r.id").
		Joins("LEFT JOIN asset_group AS g ON p.asset_group_id = g.id").
		Joins("LEFT JOIN hosts AS h ON p.host_id = h.id")

	if roleID != nil {
		query = query.Where("p.role_id = ?", *roleID)
	}
	if assetGroupID != nil {
		query = query.Where("p.asset_group_id = ?", *assetGroupID)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	err := query.
		Order("p.created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&permissions).Error

	return permissions, total, err
}

// CheckHostPermission 检查用户是否有访问指定主机的权限
func (r *assetPermissionRepo) CheckHostPermission(ctx context.Context, userID, hostID uint) (bool, error) {
	// 首先检查用户是否是管理员
	var adminCount int64
	err := r.db.WithContext(ctx).
		Table("sys_user_role AS ur").
		Joins("JOIN sys_role AS r ON ur.role_id = r.id").
		Where("ur.user_id = ? AND r.code = ?", userID, "admin").
		Count(&adminCount).Error

	if err != nil {
		return false, err
	}

	// 管理员拥有所有权限
	if adminCount > 0 {
		return true, nil
	}

	// 获取主机所属的资产分组ID
	var groupID uint
	err = r.db.WithContext(ctx).
		Table("hosts").
		Select("group_id").
		Where("id = ?", hostID).
		Scan(&groupID).Error

	if err != nil {
		return false, err
	}

	// 检查用户的角色是否有权限访问该主机
	// 1. 检查是否有整个分组的权限（host_id IS NULL）
	// 2. 检查是否有特定主机的权限（host_id = hostID）
	var permCount int64
	err = r.db.WithContext(ctx).
		Table("sys_role_asset_permission AS p").
		Joins("JOIN sys_user_role AS ur ON p.role_id = ur.role_id").
		Where("ur.user_id = ? AND p.asset_group_id = ?", userID, groupID).
		Where("p.host_id IS NULL OR p.host_id = ?", hostID).
		Count(&permCount).Error

	if err != nil {
		return false, err
	}

	return permCount > 0, nil
}

// GetUserAccessibleHostIDs 获取用户有权限访问的所有主机ID列表
func (r *assetPermissionRepo) GetUserAccessibleHostIDs(ctx context.Context, userID uint) ([]uint, error) {
	// 首先检查用户是否是管理员
	var adminCount int64
	err := r.db.WithContext(ctx).
		Table("sys_user_role AS ur").
		Joins("JOIN sys_role AS r ON ur.role_id = r.id").
		Where("ur.user_id = ? AND r.code = ?", userID, "admin").
		Count(&adminCount).Error

	if err != nil {
		return nil, err
	}

	// 管理员可以访问所有主机
	if adminCount > 0 {
		var allHostIDs []uint
		err = r.db.WithContext(ctx).
			Table("hosts").
			Where("deleted_at IS NULL").
			Pluck("id", &allHostIDs).Error
		return allHostIDs, err
	}

	// 获取用户有权限的主机ID列表
	// 1. 直接授权的主机（host_id IS NOT NULL）
	// 2. 通过资产分组授权的所有主机（host_id IS NULL）
	var hostIDs []uint

	// 查询直接授权的主机和通过分组授权的主机
	err = r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT h.id
		FROM hosts AS h
		JOIN sys_role_asset_permission AS p ON (
			p.asset_group_id = h.group_id
			AND (p.host_id IS NULL OR p.host_id = h.id)
		)
		JOIN sys_user_role AS ur ON p.role_id = ur.role_id
		WHERE ur.user_id = ? AND h.deleted_at IS NULL
	`, userID).Scan(&hostIDs).Error

	return hostIDs, err
}

// CheckHostOperationPermission 检查用户是否有对指定主机的特定操作权限
func (r *assetPermissionRepo) CheckHostOperationPermission(ctx context.Context, userID, hostID uint, operation uint) (bool, error) {
	// 首先检查用户是否是管理员
	var adminCount int64
	err := r.db.WithContext(ctx).
		Table("sys_user_role AS ur").
		Joins("JOIN sys_role AS r ON ur.role_id = r.id").
		Where("ur.user_id = ? AND r.code = ?", userID, "admin").
		Count(&adminCount).Error

	if err != nil {
		return false, err
	}

	// 管理员拥有所有权限
	if adminCount > 0 {
		return true, nil
	}

	// 获取主机所属的资产分组ID
	var groupID uint
	err = r.db.WithContext(ctx).
		Table("hosts").
		Select("group_id").
		Where("id = ?", hostID).
		Scan(&groupID).Error

	if err != nil {
		return false, err
	}

	// 检查用户的角色是否有该操作权限
	var permCount int64
	err = r.db.WithContext(ctx).
		Table("sys_role_asset_permission AS p").
		Joins("JOIN sys_user_role AS ur ON p.role_id = ur.role_id").
		Where("ur.user_id = ? AND p.asset_group_id = ?", userID, groupID).
		Where("(p.host_id IS NULL OR p.host_id = ?) AND (p.permissions & ?) > 0", hostID, operation).
		Count(&permCount).Error

	if err != nil {
		return false, err
	}

	return permCount > 0, nil
}

// GetUserHostPermissions 获取用户对指定主机的所有操作权限
func (r *assetPermissionRepo) GetUserHostPermissions(ctx context.Context, userID, hostID uint) (uint, error) {
	// 首先检查用户是否是管理员
	var adminCount int64
	err := r.db.WithContext(ctx).
		Table("sys_user_role AS ur").
		Joins("JOIN sys_role AS r ON ur.role_id = r.id").
		Where("ur.user_id = ? AND r.code = ?", userID, "admin").
		Count(&adminCount).Error

	if err != nil {
		return 0, err
	}

	// 管理员拥有所有权限
	if adminCount > 0 {
		return rbac.PermissionAll, nil
	}

	// 获取主机所属的资产分组ID
	var groupID uint
	err = r.db.WithContext(ctx).
		Table("hosts").
		Select("group_id").
		Where("id = ?", hostID).
		Scan(&groupID).Error

	if err != nil {
		return 0, err
	}

	// 查询用户对该主机的所有权限（通过OR操作组合权限）
	var permissions uint
	err = r.db.WithContext(ctx).Raw(`
		SELECT COALESCE(BIT_OR(p.permissions), 0) as permissions
		FROM sys_role_asset_permission AS p
		JOIN sys_user_role AS ur ON p.role_id = ur.role_id
		WHERE ur.user_id = ? AND p.asset_group_id = ? AND (p.host_id IS NULL OR p.host_id = ?)
	`, userID, groupID, hostID).Scan(&permissions).Error

	return permissions, err
}
