package rbac

import (
	"context"
	"errors"
	"golang.org/x/crypto/bcrypt"
)

type UserUseCase struct {
	userRepo UserRepo
}

func NewUserUseCase(userRepo UserRepo) *UserUseCase {
	return &UserUseCase{
		userRepo: userRepo,
	}
}

func (uc *UserUseCase) Create(ctx context.Context, user *SysUser) error {
	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)
	return uc.userRepo.Create(ctx, user)
}

func (uc *UserUseCase) Update(ctx context.Context, user *SysUser) error {
	return uc.userRepo.Update(ctx, user)
}

func (uc *UserUseCase) Delete(ctx context.Context, id uint) error {
	return uc.userRepo.Delete(ctx, id)
}

func (uc *UserUseCase) GetByID(ctx context.Context, id uint) (*SysUser, error) {
	return uc.userRepo.GetByID(ctx, id)
}

func (uc *UserUseCase) GetByUsername(ctx context.Context, username string) (*SysUser, error) {
	return uc.userRepo.GetByUsername(ctx, username)
}

func (uc *UserUseCase) List(ctx context.Context, page, pageSize int, keyword string, departmentID uint) ([]*SysUser, int64, error) {
	return uc.userRepo.List(ctx, page, pageSize, keyword, departmentID)
}

func (uc *UserUseCase) AssignRoles(ctx context.Context, userID uint, roleIDs []uint) error {
	return uc.userRepo.AssignRoles(ctx, userID, roleIDs)
}

func (uc *UserUseCase) AssignPositions(ctx context.Context, userID uint, positionIDs []uint) error {
	return uc.userRepo.AssignPositions(ctx, userID, positionIDs)
}

func (uc *UserUseCase) ValidatePassword(ctx context.Context, username, password string) (*SysUser, error) {
	user, err := uc.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	return user, nil
}

func (uc *UserUseCase) UpdatePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return errors.New("用户不存在")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword))
	if err != nil {
		return errors.New("原密码错误")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashedPassword)
	return uc.userRepo.Update(ctx, user)
}

func (uc *UserUseCase) ResetPassword(ctx context.Context, userID uint, newPassword string) error {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return errors.New("用户不存在")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashedPassword)
	return uc.userRepo.Update(ctx, user)
}

type RoleUseCase struct {
	roleRepo RoleRepo
}

func NewRoleUseCase(roleRepo RoleRepo) *RoleUseCase {
	return &RoleUseCase{
		roleRepo: roleRepo,
	}
}

func (uc *RoleUseCase) Create(ctx context.Context, role *SysRole) error {
	return uc.roleRepo.Create(ctx, role)
}

func (uc *RoleUseCase) Update(ctx context.Context, role *SysRole) error {
	return uc.roleRepo.Update(ctx, role)
}

func (uc *RoleUseCase) Delete(ctx context.Context, id uint) error {
	return uc.roleRepo.Delete(ctx, id)
}

func (uc *RoleUseCase) GetByID(ctx context.Context, id uint) (*SysRole, error) {
	return uc.roleRepo.GetByID(ctx, id)
}

func (uc *RoleUseCase) List(ctx context.Context, page, pageSize int, keyword string) ([]*SysRole, int64, error) {
	return uc.roleRepo.List(ctx, page, pageSize, keyword)
}

func (uc *RoleUseCase) GetAll(ctx context.Context) ([]*SysRole, error) {
	return uc.roleRepo.GetAll(ctx)
}

func (uc *RoleUseCase) AssignMenus(ctx context.Context, roleID uint, menuIDs []uint) error {
	return uc.roleRepo.AssignMenus(ctx, roleID, menuIDs)
}

func (uc *RoleUseCase) GetByUserID(ctx context.Context, userID uint) ([]*SysRole, error) {
	return uc.roleRepo.GetByUserID(ctx, userID)
}

type DepartmentUseCase struct {
	deptRepo DepartmentRepo
}

func NewDepartmentUseCase(deptRepo DepartmentRepo) *DepartmentUseCase {
	return &DepartmentUseCase{
		deptRepo: deptRepo,
	}
}

func (uc *DepartmentUseCase) Create(ctx context.Context, dept *SysDepartment) error {
	return uc.deptRepo.Create(ctx, dept)
}

func (uc *DepartmentUseCase) Update(ctx context.Context, dept *SysDepartment) error {
	return uc.deptRepo.Update(ctx, dept)
}

func (uc *DepartmentUseCase) Delete(ctx context.Context, id uint) error {
	return uc.deptRepo.Delete(ctx, id)
}

func (uc *DepartmentUseCase) GetByID(ctx context.Context, id uint) (*SysDepartment, error) {
	return uc.deptRepo.GetByID(ctx, id)
}

func (uc *DepartmentUseCase) GetTree(ctx context.Context) ([]*SysDepartment, error) {
	return uc.deptRepo.GetTree(ctx)
}

// GetParentOptions 获取父级部门选项（用于级联选择器）
func (uc *DepartmentUseCase) GetParentOptions(ctx context.Context) ([]*DepartmentParentOptionVO, error) {
	departments, err := uc.deptRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	return uc.buildParentOptions(departments, 0), nil
}

func (uc *DepartmentUseCase) buildParentOptions(departments []*SysDepartment, parentID uint) []*DepartmentParentOptionVO {
	var options []*DepartmentParentOptionVO
	for _, dept := range departments {
		if dept.ParentID == parentID {
			option := &DepartmentParentOptionVO{
				ID:       dept.ID,
				ParentID: dept.ParentID,
				Label:    dept.Name,
			}
			children := uc.buildParentOptions(departments, dept.ID)
			if len(children) > 0 {
				option.Children = children
			}
			options = append(options, option)
		}
	}
	return options
}

// ToInfoVO 将SysDepartment转换为DepartmentInfoVO
func (uc *DepartmentUseCase) ToInfoVO(dept *SysDepartment) *DepartmentInfoVO {
	if dept == nil {
		return nil
	}
	vo := &DepartmentInfoVO{
		ID:         dept.ID,
		ParentID:   dept.ParentID,
		DeptType:   dept.DeptType,
		DeptName:   dept.Name,
		Code:       dept.Code,
		DeptStatus: dept.Status,
		CreateTime: dept.CreatedAt.Format("2006-01-02 15:04:05"),
		UserCount:  dept.UserCount,
	}
	if len(dept.Children) > 0 {
		for _, child := range dept.Children {
			vo.Children = append(vo.Children, uc.ToInfoVO(child))
		}
	}
	return vo
}

type MenuUseCase struct {
	menuRepo MenuRepo
}

func NewMenuUseCase(menuRepo MenuRepo) *MenuUseCase {
	return &MenuUseCase{
		menuRepo: menuRepo,
	}
}

func (uc *MenuUseCase) Create(ctx context.Context, menu *SysMenu) error {
	return uc.menuRepo.Create(ctx, menu)
}

func (uc *MenuUseCase) Update(ctx context.Context, menu *SysMenu) error {
	return uc.menuRepo.Update(ctx, menu)
}

func (uc *MenuUseCase) Delete(ctx context.Context, id uint) error {
	return uc.menuRepo.Delete(ctx, id)
}

func (uc *MenuUseCase) GetByID(ctx context.Context, id uint) (*SysMenu, error) {
	return uc.menuRepo.GetByID(ctx, id)
}

func (uc *MenuUseCase) GetTree(ctx context.Context) ([]*SysMenu, error) {
	return uc.menuRepo.GetTree(ctx)
}

func (uc *MenuUseCase) GetByUserID(ctx context.Context, userID uint) ([]*SysMenu, error) {
	return uc.menuRepo.GetByUserID(ctx, userID)
}

func (uc *MenuUseCase) GetByRoleID(ctx context.Context, roleID uint) ([]*SysMenu, error) {
	return uc.menuRepo.GetByRoleID(ctx, roleID)
}

type PositionUseCase struct {
	positionRepo PositionRepo
}

func NewPositionUseCase(positionRepo PositionRepo) *PositionUseCase {
	return &PositionUseCase{
		positionRepo: positionRepo,
	}
}

func (uc *PositionUseCase) Create(ctx context.Context, position *SysPosition) error {
	return uc.positionRepo.Create(ctx, position)
}

func (uc *PositionUseCase) Update(ctx context.Context, position *SysPosition) error {
	return uc.positionRepo.Update(ctx, position)
}

func (uc *PositionUseCase) Delete(ctx context.Context, id uint) error {
	return uc.positionRepo.Delete(ctx, id)
}

func (uc *PositionUseCase) GetByID(ctx context.Context, id uint) (*SysPosition, error) {
	return uc.positionRepo.GetByID(ctx, id)
}

func (uc *PositionUseCase) List(ctx context.Context, page, pageSize int, postCode, postName string) ([]*SysPosition, int64, error) {
	return uc.positionRepo.List(ctx, page, pageSize, postCode, postName)
}

func (uc *PositionUseCase) GetAll(ctx context.Context) ([]*SysPosition, error) {
	return uc.positionRepo.GetAll(ctx)
}

func (uc *PositionUseCase) GetUsers(ctx context.Context, positionID uint, page, pageSize int) ([]*SysUser, int64, error) {
	return uc.positionRepo.GetUsers(ctx, positionID, page, pageSize)
}

func (uc *PositionUseCase) AssignUsers(ctx context.Context, positionID uint, userIDs []uint) error {
	return uc.positionRepo.AssignUsers(ctx, positionID, userIDs)
}

func (uc *PositionUseCase) RemoveUser(ctx context.Context, positionID, userID uint) error {
	return uc.positionRepo.RemoveUser(ctx, positionID, userID)
}

type AssetPermissionUseCase struct {
	assetPermissionRepo AssetPermissionRepo
}

func NewAssetPermissionUseCase(assetPermissionRepo AssetPermissionRepo) *AssetPermissionUseCase {
	return &AssetPermissionUseCase{
		assetPermissionRepo: assetPermissionRepo,
	}
}

// CreateBatch 批量创建资产权限
func (uc *AssetPermissionUseCase) CreateBatch(ctx context.Context, roleID, assetGroupID uint, hostIDs []uint) error {
	return uc.assetPermissionRepo.CreateBatch(ctx, roleID, assetGroupID, hostIDs)
}

// DeleteByRoleAndGroup 删除指定角色对指定资产分组的所有权限
func (uc *AssetPermissionUseCase) DeleteByRoleAndGroup(ctx context.Context, roleID, assetGroupID uint) error {
	return uc.assetPermissionRepo.DeleteByRoleAndGroup(ctx, roleID, assetGroupID)
}

// Delete 删除单个权限
func (uc *AssetPermissionUseCase) Delete(ctx context.Context, id uint) error {
	return uc.assetPermissionRepo.Delete(ctx, id)
}

// GetDetailByID 根据ID获取权限详情（用于编辑）
func (uc *AssetPermissionUseCase) GetDetailByID(ctx context.Context, id uint) (*AssetPermissionDetailVO, error) {
	return uc.assetPermissionRepo.GetDetailByID(ctx, id)
}

// UpdateAssetPermission 更新权限配置（支持修改角色、分组、主机、权限）
func (uc *AssetPermissionUseCase) UpdateAssetPermission(ctx context.Context, id uint, roleID, assetGroupID uint, hostIDs []uint, permissions uint) error {
	return uc.assetPermissionRepo.UpdateAssetPermission(ctx, id, roleID, assetGroupID, hostIDs, permissions)
}

// GetByRoleID 获取角色的所有资产权限
func (uc *AssetPermissionUseCase) GetByRoleID(ctx context.Context, roleID uint) ([]*AssetPermissionInfo, error) {
	return uc.assetPermissionRepo.GetByRoleID(ctx, roleID)
}

// GetByAssetGroupID 获取资产分组的所有权限配置
func (uc *AssetPermissionUseCase) GetByAssetGroupID(ctx context.Context, assetGroupID uint) ([]*AssetPermissionInfo, error) {
	return uc.assetPermissionRepo.GetByAssetGroupID(ctx, assetGroupID)
}

// List 分页查询权限列表
func (uc *AssetPermissionUseCase) List(ctx context.Context, page, pageSize int, roleID, assetGroupID *uint) ([]*AssetPermissionInfo, int64, error) {
	return uc.assetPermissionRepo.List(ctx, page, pageSize, roleID, assetGroupID)
}

// CheckHostPermission 检查用户是否有访问指定主机的权限
func (uc *AssetPermissionUseCase) CheckHostPermission(ctx context.Context, userID, hostID uint) (bool, error) {
	return uc.assetPermissionRepo.CheckHostPermission(ctx, userID, hostID)
}

// GetUserAccessibleHostIDs 获取用户有权限访问的所有主机ID列表
func (uc *AssetPermissionUseCase) GetUserAccessibleHostIDs(ctx context.Context, userID uint) ([]uint, error) {
	return uc.assetPermissionRepo.GetUserAccessibleHostIDs(ctx, userID)
}

// CreateBatchWithPermissions 批量创建资产权限（支持指定操作权限）
func (uc *AssetPermissionUseCase) CreateBatchWithPermissions(ctx context.Context, roleID, assetGroupID uint, hostIDs []uint, permissions uint) error {
	return uc.assetPermissionRepo.CreateBatchWithPermissions(ctx, roleID, assetGroupID, hostIDs, permissions)
}

// CheckHostOperationPermission 检查用户是否有对指定主机的特定操作权限
func (uc *AssetPermissionUseCase) CheckHostOperationPermission(ctx context.Context, userID, hostID uint, operation uint) (bool, error) {
	return uc.assetPermissionRepo.CheckHostOperationPermission(ctx, userID, hostID, operation)
}

// GetUserHostPermissions 获取用户对指定主机的所有操作权限
func (uc *AssetPermissionUseCase) GetUserHostPermissions(ctx context.Context, userID, hostID uint) (uint, error) {
	return uc.assetPermissionRepo.GetUserHostPermissions(ctx, userID, hostID)
}
