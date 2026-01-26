package rbac

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

type DepartmentService struct {
	deptUseCase *rbac.DepartmentUseCase
}

func NewDepartmentService(deptUseCase *rbac.DepartmentUseCase) *DepartmentService {
	return &DepartmentService{
		deptUseCase: deptUseCase,
	}
}

// CreateDepartment 创建部门
// @Summary 创建部门
// @Description 管理员创建新部门
// @Tags 部门管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body rbac.DepartmentRequest true "部门信息"
// @Success 200 {object} response.Response "创建成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/departments [post]
func (s *DepartmentService) CreateDepartment(c *gin.Context) {
	var req rbac.DepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	dept := req.ToModel()
	if err := s.deptUseCase.Create(c.Request.Context(), dept); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, dept)
}

// UpdateDepartment 更新部门
// @Summary 更新部门
// @Description 管理员更新部门信息
// @Tags 部门管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "部门ID"
// @Param body body rbac.DepartmentRequest true "部门信息"
// @Success 200 {object} response.Response "更新成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/departments/{id} [put]
func (s *DepartmentService) UpdateDepartment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的部门ID")
		return
	}

	var req rbac.DepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	dept := req.ToModel()
	dept.ID = uint(id)
	if err := s.deptUseCase.Update(c.Request.Context(), dept); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.Success(c, dept)
}

// DeleteDepartment 删除部门
// @Summary 删除部门
// @Description 管理员删除部门
// @Tags 部门管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "部门ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/departments/{id} [delete]
func (s *DepartmentService) DeleteDepartment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的部门ID")
		return
	}

	if err := s.deptUseCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetDepartment 获取部门详情
// @Summary 获取部门详情
// @Description 获取单个部门的详细信息
// @Tags 部门管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "部门ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 404 {object} response.Response "部门不存在"
// @Router /api/v1/departments/{id} [get]
func (s *DepartmentService) GetDepartment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的部门ID")
		return
	}

	dept, err := s.deptUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "部门不存在")
		return
	}

	response.Success(c, dept)
}

// GetDepartmentTree 获取部门树
// @Summary 获取部门树
// @Description 获取部门树形结构数据
// @Tags 部门管理
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/departments/tree [get]
func (s *DepartmentService) GetDepartmentTree(c *gin.Context) {
	tree, err := s.deptUseCase.GetTree(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	// 转换为VO格式
	var voTree []*rbac.DepartmentInfoVO
	for _, dept := range tree {
		voTree = append(voTree, s.deptUseCase.ToInfoVO(dept))
	}

	response.Success(c, voTree)
}

// GetParentOptions 获取父级部门选项
// @Summary 获取父级部门选项
// @Description 获取可选的父级部门列表
// @Tags 部门管理
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/departments/parent-options [get]
func (s *DepartmentService) GetParentOptions(c *gin.Context) {
	options, err := s.deptUseCase.GetParentOptions(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, options)
}
