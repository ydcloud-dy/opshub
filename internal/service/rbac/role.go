package rbac

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

type RoleService struct {
	roleUseCase *rbac.RoleUseCase
}

func NewRoleService(roleUseCase *rbac.RoleUseCase) *RoleService {
	return &RoleService{
		roleUseCase: roleUseCase,
	}
}

// CreateRole 创建角色
// @Summary 创建角色
// @Description 管理员创建新角色
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body rbac.SysRole true "角色信息"
// @Success 200 {object} response.Response "创建成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/roles [post]
func (s *RoleService) CreateRole(c *gin.Context) {
	var req rbac.SysRole
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.roleUseCase.Create(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, req)
}

// UpdateRole 更新角色
// @Summary 更新角色
// @Description 管理员更新角色信息
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "角色ID"
// @Param body body rbac.SysRole true "角色信息"
// @Success 200 {object} response.Response "更新成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/roles/{id} [put]
func (s *RoleService) UpdateRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的角色ID")
		return
	}

	var req rbac.SysRole
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	req.ID = uint(id)
	if err := s.roleUseCase.Update(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.Success(c, req)
}

// DeleteRole 删除角色
// @Summary 删除角色
// @Description 管理员删除角色
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "角色ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/roles/{id} [delete]
func (s *RoleService) DeleteRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的角色ID")
		return
	}

	if err := s.roleUseCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetRole 获取角色详情
// @Summary 获取角色详情
// @Description 获取单个角色的详细信息
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "角色ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 404 {object} response.Response "角色不存在"
// @Router /api/v1/roles/{id} [get]
func (s *RoleService) GetRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的角色ID")
		return
	}

	role, err := s.roleUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "角色不存在")
		return
	}

	response.Success(c, role)
}

// ListRoles 角色列表
// @Summary 获取角色列表
// @Description 分页获取角色列表，支持关键字搜索
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键字"
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/roles [get]
func (s *RoleService) ListRoles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")

	roles, total, err := s.roleUseCase.List(c.Request.Context(), page, pageSize, keyword)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":     roles,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetAllRoles 获取所有角色（不分页）
// @Summary 获取所有角色
// @Description 获取所有角色列表，不分页
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/roles/all [get]
func (s *RoleService) GetAllRoles(c *gin.Context) {
	roles, err := s.roleUseCase.GetAll(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, roles)
}

// AssignRoleMenusRequest 分配角色菜单请求
type AssignRoleMenusRequest struct {
	MenuIDs []uint `json:"menuIds" binding:"required"`
}

// AssignRoleMenus 分配角色菜单
// @Summary 分配角色菜单
// @Description 为角色分配菜单权限
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "角色ID"
// @Param body body AssignRoleMenusRequest true "菜单IDs"
// @Success 200 {object} response.Response "分配成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/roles/{id}/menus [post]
func (s *RoleService) AssignRoleMenus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的角色ID")
		return
	}

	var req AssignRoleMenusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.roleUseCase.AssignMenus(c.Request.Context(), uint(id), req.MenuIDs); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "分配失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}
