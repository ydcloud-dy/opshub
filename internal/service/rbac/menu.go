package rbac

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

type MenuService struct {
	menuUseCase *rbac.MenuUseCase
	roleUseCase *rbac.RoleUseCase
}

func NewMenuService(menuUseCase *rbac.MenuUseCase, roleUseCase *rbac.RoleUseCase) *MenuService {
	return &MenuService{
		menuUseCase: menuUseCase,
		roleUseCase: roleUseCase,
	}
}

// CreateMenu 创建菜单
// @Summary 创建菜单
// @Description 管理员创建新菜单
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body rbac.SysMenu true "菜单信息"
// @Success 200 {object} response.Response "创建成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 409 {object} response.Response "菜单编码已存在"
// @Router /api/v1/menus [post]
func (s *MenuService) CreateMenu(c *gin.Context) {
	var req rbac.SysMenu
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.menuUseCase.Create(c.Request.Context(), &req); err != nil {
		// 检查是否是重复键错误
		if containsDuplicateKeyError(err.Error()) {
			response.ErrorCode(c, http.StatusConflict, "菜单编码已存在，请使用不同的编码")
			return
		}
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, req)
}

// 检查是否包含重复键错误
func containsDuplicateKeyError(err string) bool {
	return contains(err, "Duplicate entry") || contains(err, "duplicate key")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// UpdateMenu 更新菜单
// @Summary 更新菜单
// @Description 管理员更新菜单信息
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "菜单ID"
// @Param body body rbac.SysMenu true "菜单信息"
// @Success 200 {object} response.Response "更新成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 409 {object} response.Response "菜单编码已存在"
// @Router /api/v1/menus/{id} [put]
func (s *MenuService) UpdateMenu(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的菜单ID")
		return
	}

	var req rbac.SysMenu
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	req.ID = uint(id)
	if err := s.menuUseCase.Update(c.Request.Context(), &req); err != nil {
		// 检查是否是重复键错误
		if containsDuplicateKeyError(err.Error()) {
			response.ErrorCode(c, http.StatusConflict, "菜单编码已存在，请使用不同的编码")
			return
		}
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.Success(c, req)
}

// DeleteMenu 删除菜单
// @Summary 删除菜单
// @Description 管理员删除菜单
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "菜单ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/menus/{id} [delete]
func (s *MenuService) DeleteMenu(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的菜单ID")
		return
	}

	if err := s.menuUseCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetMenu 获取菜单详情
// @Summary 获取菜单详情
// @Description 获取单个菜单的详细信息
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "菜单ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 404 {object} response.Response "菜单不存在"
// @Router /api/v1/menus/{id} [get]
func (s *MenuService) GetMenu(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的菜单ID")
		return
	}

	menu, err := s.menuUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "菜单不存在")
		return
	}

	response.Success(c, menu)
}

// GetMenuTree 获取菜单树
// @Summary 获取菜单树
// @Description 获取完整的菜单树形结构
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/menus/tree [get]
func (s *MenuService) GetMenuTree(c *gin.Context) {
	tree, err := s.menuUseCase.GetTree(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, tree)
}

// GetUserMenu 获取当前用户的菜单树
// @Summary 获取用户菜单
// @Description 获取当前登录用户有权限的菜单树
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response "获取成功"
// @Failure 401 {object} response.Response "未登录"
// @Router /api/v1/menus/user [get]
func (s *MenuService) GetUserMenu(c *gin.Context) {
	userID := GetUserID(c)
	if userID == 0 {
		response.ErrorCode(c, http.StatusUnauthorized, "未登录")
		return
	}

	// 获取用户的角色
	roles, err := s.roleUseCase.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询角色失败: "+err.Error())
		return
	}

	// 检查是否是超级管理员（admin角色）
	isSuperAdmin := false
	for _, role := range roles {
		if role.Code == "admin" {
			isSuperAdmin = true
			break
		}
	}

	var tree []*rbac.SysMenu

	// 如果是超级管理员，返回所有菜单
	if isSuperAdmin {
		tree, err = s.menuUseCase.GetTree(c.Request.Context())
	} else {
		// 普通用户根据角色权限获取菜单
		tree, err = s.menuUseCase.GetByUserID(c.Request.Context(), userID)
	}

	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, tree)
}
