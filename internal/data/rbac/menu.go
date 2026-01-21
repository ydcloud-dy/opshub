package rbac

import (
	"context"
	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	"gorm.io/gorm"
)

type menuRepo struct {
	db *gorm.DB
}

func NewMenuRepo(db *gorm.DB) rbac.MenuRepo {
	return &menuRepo{db: db}
}

func (r *menuRepo) Create(ctx context.Context, menu *rbac.SysMenu) error {
	return r.db.WithContext(ctx).Create(menu).Error
}

func (r *menuRepo) Update(ctx context.Context, menu *rbac.SysMenu) error {
	return r.db.WithContext(ctx).Model(&rbac.SysMenu{}).Where("id = ?", menu.ID).Updates(map[string]interface{}{
		"name":       menu.Name,
		"code":       menu.Code,
		"type":       menu.Type,
		"parent_id":  menu.ParentID,
		"path":       menu.Path,
		"component":  menu.Component,
		"icon":       menu.Icon,
		"sort":       menu.Sort,
		"visible":    menu.Visible,
		"status":     menu.Status,
	}).Error
}

func (r *menuRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除角色菜单关联
		if err := tx.Where("menu_id = ?", id).Delete(&rbac.SysRoleMenu{}).Error; err != nil {
			return err
		}

		// 检查是否有子菜单
		var count int64
		if err := tx.Model(&rbac.SysMenu{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return gorm.ErrRegistered // 存在子菜单，不能删除
		}

		// 删除菜单
		return tx.Delete(&rbac.SysMenu{}, id).Error
	})
}

func (r *menuRepo) GetByID(ctx context.Context, id uint) (*rbac.SysMenu, error) {
	var menu rbac.SysMenu
	err := r.db.WithContext(ctx).First(&menu, id).Error
	return &menu, err
}

func (r *menuRepo) GetTree(ctx context.Context) ([]*rbac.SysMenu, error) {
	var menus []*rbac.SysMenu
	err := r.db.WithContext(ctx).
		Where("visible = ?", 1).
		Order("sort ASC").
		Find(&menus).Error
	if err != nil {
		return nil, err
	}
	return r.buildTree(menus, 0), nil
}

func (r *menuRepo) buildTree(menus []*rbac.SysMenu, parentID uint) []*rbac.SysMenu {
	var tree []*rbac.SysMenu
	for _, menu := range menus {
		if menu.ParentID == parentID {
			children := r.buildTree(menus, menu.ID)
			if len(children) > 0 {
				menu.Children = children
			}
			tree = append(tree, menu)
		}
	}
	return tree
}

func (r *menuRepo) GetByUserID(ctx context.Context, userID uint) ([]*rbac.SysMenu, error) {
	var menus []*rbac.SysMenu
	err := r.db.WithContext(ctx).
		Joins("JOIN sys_role_menu ON sys_role_menu.menu_id = sys_menu.id").
		Joins("JOIN sys_user_role ON sys_user_role.role_id = sys_role_menu.role_id").
		Where("sys_user_role.user_id = ? AND sys_menu.status = 1 AND sys_menu.visible = 1", userID).
		Distinct().
		Order("sys_menu.sort ASC").
		Find(&menus).Error

	if err != nil {
		return nil, err
	}

	return r.buildTree(menus, 0), nil
}

func (r *menuRepo) GetByRoleID(ctx context.Context, roleID uint) ([]*rbac.SysMenu, error) {
	var menus []*rbac.SysMenu
	err := r.db.WithContext(ctx).
		Joins("JOIN sys_role_menus ON sys_role_menus.menu_id = sys_menu.id").
		Where("sys_role_menus.role_id = ?", roleID).
		Find(&menus).Error
	return menus, err
}
