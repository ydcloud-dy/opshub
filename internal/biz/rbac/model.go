package rbac

import (
	"time"

	"gorm.io/gorm"
)

// SysUser 用户表
type SysUser struct {
	gorm.Model
	Username    string         `gorm:"type:varchar(50);not null;comment:用户名" json:"username"`
	Password    string         `gorm:"type:varchar(255);not null;comment:密码" json:"password,omitempty"`
	RealName    string         `gorm:"type:varchar(50);comment:真实姓名" json:"realName"`
	Email       string         `gorm:"type:varchar(100);comment:邮箱" json:"email"`
	Phone       string         `gorm:"type:varchar(20);comment:手机号" json:"phone"`
	Avatar      string         `gorm:"type:varchar(255);comment:头像" json:"avatar"`
	Status      int            `gorm:"type:tinyint;default:1;comment:状态 1:启用 0:禁用" json:"status"`
	DepartmentID uint          `gorm:"default:0;comment:部门ID" json:"departmentId"`
	Department  *SysDepartment `gorm:"foreignKey:DepartmentID;references:ID" json:"department,omitempty"`
	Roles       []SysRole      `gorm:"many2many:sys_user_role;joinForeignKey:UserID;joinReferences:RoleID" json:"roles"`
	Positions   []SysPosition `gorm:"many2many:sys_user_position;joinForeignKey:UserID;joinReferences:PositionID" json:"positions,omitempty"`
	Bio         string         `gorm:"type:text;comment:个人简介" json:"bio"`
	LastLoginAt *time.Time     `gorm:"comment:最后登录时间" json:"lastLoginAt,omitempty"`
}

// SysRole 角色表
type SysRole struct {
	gorm.Model
	Name        string       `gorm:"type:varchar(50);uniqueIndex;not null;comment:角色名称" json:"name"`
	Code        string       `gorm:"type:varchar(50);uniqueIndex;not null;comment:角色编码" json:"code"`
	Description string       `gorm:"type:varchar(200);comment:角色描述" json:"description"`
	Sort        int          `gorm:"type:int;default:0;comment:排序" json:"sort"`
	Status      int          `gorm:"type:tinyint;default:1;comment:状态 1:启用 0:禁用" json:"status"`
	Users       []SysUser    `gorm:"many2many:sys_user_role;joinForeignKey:RoleID;joinReferences:UserID" json:"-"`
	Menus       []SysMenu    `gorm:"many2many:sys_role_menu;joinForeignKey:RoleID;joinReferences:MenuID" json:"menus,omitempty"`
}

// SysDepartment 部门表
type SysDepartment struct {
	gorm.Model
	Name        string            `gorm:"type:varchar(50);not null;comment:部门名称" json:"name"`
	Code        string            `gorm:"type:varchar(50);uniqueIndex;comment:部门编码" json:"code"`
	ParentID    uint              `gorm:"column:parent_id;default:0;comment:父部门ID" json:"parentId"`
	Parent      *SysDepartment    `gorm:"-" json:"parent,omitempty"`
	Children    []*SysDepartment   `gorm:"-" json:"children,omitempty"`
	DeptType    int               `gorm:"column:dept_type;type:tinyint;default:3;comment:部门类型 1:公司 2:中心 3:部门" json:"deptType"`
	Sort        int               `gorm:"type:int;default:0;comment:排序" json:"sort"`
	Status      int               `gorm:"type:tinyint;default:1;comment:状态 1:启用 0:禁用" json:"status"`
	UserCount   int               `gorm:"-" json:"userCount"` // 用户数量（仅用于API响应）
}

// DepartmentRequest 部门请求（前端使用）
type DepartmentRequest struct {
	ID         uint   `json:"id"`
	ParentID   uint   `json:"parentId"`
	DeptType   int    `json:"deptType"`
	DeptName   string `json:"deptName" binding:"required,min=2,max=50"`
	Code       string `json:"code" binding:"required,min=2,max=50"`
	Sort       int    `json:"sort"`
	DeptStatus int    `json:"deptStatus" binding:"required"`
}

// ToModel 转换为SysDepartment模型
func (r *DepartmentRequest) ToModel() *SysDepartment {
	return &SysDepartment{
		Model:    gorm.Model{ID: r.ID},
		Name:     r.DeptName,
		Code:     r.Code,
		ParentID: r.ParentID,
		DeptType: r.DeptType,
		Sort:     r.Sort,
		Status:   r.DeptStatus,
	}
}

// DepartmentInfoVO 部门信息VO（用于API响应）
type DepartmentInfoVO struct {
	ID         uint               `json:"id"`
	ParentID   uint               `json:"parentId"`
	DeptType   int                `json:"deptType"`
	DeptName   string             `json:"deptName"`
	Code       string             `json:"code"`
	DeptStatus int                `json:"deptStatus"`
	CreateTime string             `json:"createTime"`
	UserCount  int                `json:"userCount"`
	Children   []*DepartmentInfoVO `json:"children,omitempty"`
}

// DepartmentParentOptionVO 部门父级选项VO（用于级联选择器）
type DepartmentParentOptionVO struct {
	ID       uint                        `json:"id"`
	ParentID uint                        `json:"parentId"`
	Label    string                      `json:"label"`
	Children []*DepartmentParentOptionVO `json:"children,omitempty"`
}

// SysMenu 菜单表
type SysMenu struct {
	gorm.Model
	Name        string        `gorm:"type:varchar(50);not null;comment:菜单名称" json:"name"`
	Code        string        `gorm:"type:varchar(50);uniqueIndex;comment:菜单编码" json:"code"`
	Type        int           `gorm:"type:tinyint;not null;comment:类型 1:目录 2:菜单 3:按钮" json:"type"`
	ParentID    uint          `gorm:"default:0;comment:父菜单ID" json:"parentId"`
	Parent      *SysMenu      `gorm:"-" json:"parent,omitempty"`
	Children    []*SysMenu     `gorm:"-" json:"children,omitempty"`
	Path        string        `gorm:"type:varchar(200);comment:路由路径" json:"path"`
	Component   string        `gorm:"type:varchar(200);comment:组件路径" json:"component"`
	Icon        string        `gorm:"type:varchar(100);comment:图标" json:"icon"`
	Sort        int           `gorm:"type:int;default:0;comment:排序" json:"sort"`
	Visible     int           `gorm:"type:tinyint;default:1;comment:是否显示 1:显示 0:隐藏" json:"visible"`
	Status      int           `gorm:"type:tinyint;default:1;comment:状态 1:启用 0:禁用" json:"status"`
	Roles       []SysRole     `gorm:"many2many:sys_role_menus" json:"-"`
}

// SysUserRole 用户角色关联表
type SysUserRole struct {
	UserID uint `gorm:"primaryKey;column:user_id;comment:用户ID" json:"userId"`
	RoleID uint `gorm:"primaryKey;column:role_id;comment:角色ID" json:"roleId"`
}

// SysRoleMenu 角色菜单关联表
type SysRoleMenu struct {
	RoleID uint `gorm:"primaryKey;column:role_id;comment:角色ID" json:"roleId"`
	MenuID uint `gorm:"primaryKey;column:menu_id;comment:菜单ID" json:"menuId"`
}

// SysPosition 岗位表
type SysPosition struct {
	gorm.Model
	PostName   string    `gorm:"type:varchar(50);not null;comment:岗位名称" json:"postName"`
	PostCode   string    `gorm:"type:varchar(50);uniqueIndex;not null;comment:岗位编码" json:"postCode"`
	PostStatus int       `gorm:"type:tinyint;default:1;comment:状态 1:启用 2:禁用" json:"postStatus"`
	Remark     string    `gorm:"type:varchar(200);comment:备注" json:"remark"`
	Users      []SysUser `gorm:"many2many:sys_user_position;joinForeignKey:PositionID;joinReferences:UserID" json:"users,omitempty"`
}

// SysUserPosition 用户岗位关联表
type SysUserPosition struct {
	UserID     uint `gorm:"primaryKey;column:user_id;comment:用户ID" json:"userId"`
	PositionID uint `gorm:"primaryKey;column:position_id;comment:岗位ID" json:"positionId"`
}

// TableName 指定表名
func (SysUser) TableName() string {
	return "sys_user"
}

func (SysRole) TableName() string {
	return "sys_role"
}

func (SysDepartment) TableName() string {
	return "sys_department"
}

func (SysMenu) TableName() string {
	return "sys_menu"
}

func (SysUserRole) TableName() string {
	return "sys_user_role"
}

func (SysRoleMenu) TableName() string {
	return "sys_role_menu"
}

func (SysPosition) TableName() string {
	return "sys_position"
}

func (SysUserPosition) TableName() string {
	return "sys_user_position"
}
