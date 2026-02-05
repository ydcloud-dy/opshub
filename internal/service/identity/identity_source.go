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

// IdentitySourceService 身份源服务
type IdentitySourceService struct {
	useCase *identity.IdentitySourceUseCase
}

func NewIdentitySourceService(useCase *identity.IdentitySourceUseCase) *IdentitySourceService {
	return &IdentitySourceService{useCase: useCase}
}

// ListSources 获取身份源列表
// @Summary 获取身份源列表
// @Description 分页获取身份源列表
// @Tags 身份认证-身份源管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param keyword query string false "关键词"
// @Param enabled query bool false "是否启用"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/sources [get]
func (s *IdentitySourceService) ListSources(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")

	var enabled *bool
	if e := c.Query("enabled"); e != "" {
		b := e == "true"
		enabled = &b
	}

	sources, total, err := s.useCase.List(c.Request.Context(), page, pageSize, keyword, enabled)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取身份源列表失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":  sources,
		"total": total,
	})
}

// GetSource 获取身份源详情
// @Summary 获取身份源详情
// @Description 根据ID获取身份源详情
// @Tags 身份认证-身份源管理
// @Accept json
// @Produce json
// @Param id path int true "身份源ID"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/sources/{id} [get]
func (s *IdentitySourceService) GetSource(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的ID")
		return
	}

	source, err := s.useCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "身份源不存在")
		return
	}

	response.Success(c, source)
}

// CreateSource 创建身份源
// @Summary 创建身份源
// @Description 创建新的身份源
// @Tags 身份认证-身份源管理
// @Accept json
// @Produce json
// @Param body body identity.IdentitySource true "身份源信息"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/sources [post]
func (s *IdentitySourceService) CreateSource(c *gin.Context) {
	var source identity.IdentitySource
	if err := c.ShouldBindJSON(&source); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.useCase.Create(c.Request.Context(), &source); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建身份源失败: "+err.Error())
		return
	}

	response.Success(c, source)
}

// UpdateSource 更新身份源
// @Summary 更新身份源
// @Description 更新身份源信息
// @Tags 身份认证-身份源管理
// @Accept json
// @Produce json
// @Param id path int true "身份源ID"
// @Param body body identity.IdentitySource true "身份源信息"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/sources/{id} [put]
func (s *IdentitySourceService) UpdateSource(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的ID")
		return
	}

	var source identity.IdentitySource
	if err := c.ShouldBindJSON(&source); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	source.ID = uint(id)
	if err := s.useCase.Update(c.Request.Context(), &source); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新身份源失败: "+err.Error())
		return
	}

	response.Success(c, "更新成功")
}

// DeleteSource 删除身份源
// @Summary 删除身份源
// @Description 删除身份源
// @Tags 身份认证-身份源管理
// @Accept json
// @Produce json
// @Param id path int true "身份源ID"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/sources/{id} [delete]
func (s *IdentitySourceService) DeleteSource(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := s.useCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除身份源失败: "+err.Error())
		return
	}

	response.Success(c, "删除成功")
}

// GetEnabledSources 获取启用的身份源列表
// @Summary 获取启用的身份源列表
// @Description 获取所有启用的身份源（用于登录页展示）
// @Tags 身份认证-身份源管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/identity/sources/enabled [get]
func (s *IdentitySourceService) GetEnabledSources(c *gin.Context) {
	sources, err := s.useCase.GetEnabled(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取身份源列表失败: "+err.Error())
		return
	}

	response.Success(c, sources)
}
