package rbac

import (
	"context"
	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	"gorm.io/gorm"
)

type roleRepo struct {
	db *gorm.DB
}

func NewRoleRepo(db *gorm.DB) rbac.RoleRepo {
	return &roleRepo{db: db}
}

func (r *roleRepo) Create(ctx context.Context, role *rbac.SysRole) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *roleRepo) Update(ctx context.Context, role *rbac.SysRole) error {
	return r.db.WithContext(ctx).Model(role).Omit("created_at").Updates(role).Error
}

func (r *roleRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除角色菜单关联
		if err := tx.Where("role_id = ?", id).Delete(&rbac.SysRoleMenu{}).Error; err != nil {
			return err
		}
		// 删除用户角色关联
		if err := tx.Where("role_id = ?", id).Delete(&rbac.SysUserRole{}).Error; err != nil {
			return err
		}
		// 删除角色
		return tx.Delete(&rbac.SysRole{}, id).Error
	})
}

func (r *roleRepo) GetByID(ctx context.Context, id uint) (*rbac.SysRole, error) {
	var role rbac.SysRole
	err := r.db.WithContext(ctx).Preload("Menus").First(&role, id).Error
	return &role, err
}

func (r *roleRepo) List(ctx context.Context, page, pageSize int, keyword string) ([]*rbac.SysRole, int64, error) {
	var roles []*rbac.SysRole
	var total int64

	query := r.db.WithContext(ctx).Model(&rbac.SysRole{})
	if keyword != "" {
		query = query.Where("name LIKE ? OR code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset((page - 1) * pageSize).Limit(pageSize).
		Order("sort ASC, created_at DESC").
		Find(&roles).Error

	return roles, total, err
}

func (r *roleRepo) GetAll(ctx context.Context) ([]*rbac.SysRole, error) {
	var roles []*rbac.SysRole
	err := r.db.WithContext(ctx).Where("status = 1").Order("sort ASC").Find(&roles).Error
	return roles, err
}

func (r *roleRepo) AssignMenus(ctx context.Context, roleID uint, menuIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除原有菜单
		if err := tx.Where("role_id = ?", roleID).Delete(&rbac.SysRoleMenu{}).Error; err != nil {
			return err
		}

		// 如果没有菜单ID，直接返回
		if len(menuIDs) == 0 {
			return nil
		}

		// 批量添加新菜单
		roleMenus := make([]rbac.SysRoleMenu, 0, len(menuIDs))
		for _, menuID := range menuIDs {
			roleMenus = append(roleMenus, rbac.SysRoleMenu{
				RoleID: roleID,
				MenuID: menuID,
			})
		}

		// 使用批量插入
		if err := tx.Create(&roleMenus).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *roleRepo) GetByUserID(ctx context.Context, userID uint) ([]*rbac.SysRole, error) {
	var roles []*rbac.SysRole
	err := r.db.WithContext(ctx).
		Joins("JOIN sys_user_role ON sys_user_role.role_id = sys_role.id").
		Where("sys_user_role.user_id = ?", userID).
		Find(&roles).Error
	return roles, err
}
