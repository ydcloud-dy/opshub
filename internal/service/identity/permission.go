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

package identity

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/identity"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

// PermissionService 权限服务
type PermissionService struct {
	useCase *identity.AppPermissionUseCase
}

func NewPermissionService(useCase *identity.AppPermissionUseCase) *PermissionService {
	return &PermissionService{useCase: useCase}
}

// PermissionRequest 权限请求
type PermissionRequest struct {
	AppID       uint   `json:"appId" binding:"required"`
	SubjectType string `json:"subjectType" binding:"required"`
	SubjectID   uint   `json:"subjectId" binding:"required"`
	Permission  string `json:"permission"`
}

// ListPermissions 获取权限列表
// @Summary 获取权限列表
// @Description 分页获取应用权限列表
// @Tags 身份认证-访问策略
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param appId query int false "应用ID"
// @Param subjectType query string false "主体类型(user/role/dept)"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/permissions [get]
func (s *PermissionService) ListPermissions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	subjectType := c.Query("subjectType")

	var appID *uint
	if id := c.Query("appId"); id != "" {
		idVal, _ := strconv.ParseUint(id, 10, 64)
		uidVal := uint(idVal)
		appID = &uidVal
	}

	permissions, total, err := s.useCase.List(c.Request.Context(), page, pageSize, appID, subjectType)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取权限列表失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":  permissions,
		"total": total,
	})
}

// CreatePermission 创建权限
// @Summary 创建权限
// @Description 创建应用访问权限
// @Tags 身份认证-访问策略
// @Accept json
// @Produce json
// @Param body body PermissionRequest true "权限信息"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/permissions [post]
func (s *PermissionService) CreatePermission(c *gin.Context) {
	var req PermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	permission := &identity.AppPermission{
		AppID:       req.AppID,
		SubjectType: req.SubjectType,
		SubjectID:   req.SubjectID,
		Permission:  req.Permission,
	}
	if permission.Permission == "" {
		permission.Permission = "access"
	}

	if err := s.useCase.Create(c.Request.Context(), permission); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建权限失败: "+err.Error())
		return
	}

	response.Success(c, "创建成功")
}

// DeletePermission 删除权限
// @Summary 删除权限
// @Description 删除应用访问权限
// @Tags 身份认证-访问策略
// @Accept json
// @Produce json
// @Param id path int true "权限ID"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/permissions/{id} [delete]
func (s *PermissionService) DeletePermission(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := s.useCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除权限失败: "+err.Error())
		return
	}

	response.Success(c, "删除成功")
}

// BatchCreatePermissions 批量创建权限
// @Summary 批量创建权限
// @Description 批量创建应用访问权限
// @Tags 身份认证-访问策略
// @Accept json
// @Produce json
// @Param body body []PermissionRequest true "权限信息列表"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/permissions/batch [post]
func (s *PermissionService) BatchCreatePermissions(c *gin.Context) {
	var reqs []PermissionRequest
	if err := c.ShouldBindJSON(&reqs); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	for _, req := range reqs {
		permission := &identity.AppPermission{
			AppID:       req.AppID,
			SubjectType: req.SubjectType,
			SubjectID:   req.SubjectID,
			Permission:  req.Permission,
		}
		if permission.Permission == "" {
			permission.Permission = "access"
		}
		s.useCase.Create(c.Request.Context(), permission)
	}

	response.Success(c, "创建成功")
}

// ListByApp 获取应用的权限列表
// @Summary 获取应用的权限列表
// @Description 获取指定应用的所有权限配置
// @Tags 身份认证-访问策略
// @Accept json
// @Produce json
// @Param id path int true "应用ID"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/permissions/app/{id} [get]
func (s *PermissionService) ListByApp(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的ID")
		return
	}

	permissions, err := s.useCase.ListByApp(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取权限列表失败: "+err.Error())
		return
	}

	response.Success(c, permissions)
}
