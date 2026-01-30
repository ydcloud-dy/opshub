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

// SSOApplicationService SSO应用服务
type SSOApplicationService struct {
	useCase *identity.SSOApplicationUseCase
}

func NewSSOApplicationService(useCase *identity.SSOApplicationUseCase) *SSOApplicationService {
	return &SSOApplicationService{useCase: useCase}
}

// ListApps 获取应用列表
// @Summary 获取应用列表
// @Description 分页获取SSO应用列表
// @Tags 身份认证-应用管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param keyword query string false "关键词"
// @Param category query string false "分类"
// @Param enabled query bool false "是否启用"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/apps [get]
func (s *SSOApplicationService) ListApps(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")
	category := c.Query("category")

	var enabled *bool
	if e := c.Query("enabled"); e != "" {
		b := e == "true"
		enabled = &b
	}

	apps, total, err := s.useCase.List(c.Request.Context(), page, pageSize, keyword, category, enabled)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取应用列表失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":  apps,
		"total": total,
	})
}

// GetApp 获取应用详情
// @Summary 获取应用详情
// @Description 根据ID获取应用详情
// @Tags 身份认证-应用管理
// @Accept json
// @Produce json
// @Param id path int true "应用ID"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/apps/{id} [get]
func (s *SSOApplicationService) GetApp(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的ID")
		return
	}

	app, err := s.useCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "应用不存在")
		return
	}

	response.Success(c, app)
}

// CreateApp 创建应用
// @Summary 创建应用
// @Description 创建新的SSO应用
// @Tags 身份认证-应用管理
// @Accept json
// @Produce json
// @Param body body identity.SSOApplication true "应用信息"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/apps [post]
func (s *SSOApplicationService) CreateApp(c *gin.Context) {
	var app identity.SSOApplication
	if err := c.ShouldBindJSON(&app); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.useCase.Create(c.Request.Context(), &app); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建应用失败: "+err.Error())
		return
	}

	response.Success(c, app)
}

// UpdateApp 更新应用
// @Summary 更新应用
// @Description 更新应用信息
// @Tags 身份认证-应用管理
// @Accept json
// @Produce json
// @Param id path int true "应用ID"
// @Param body body identity.SSOApplication true "应用信息"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/apps/{id} [put]
func (s *SSOApplicationService) UpdateApp(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的ID")
		return
	}

	var app identity.SSOApplication
	if err := c.ShouldBindJSON(&app); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	app.ID = uint(id)
	if err := s.useCase.Update(c.Request.Context(), &app); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新应用失败: "+err.Error())
		return
	}

	response.Success(c, "更新成功")
}

// DeleteApp 删除应用
// @Summary 删除应用
// @Description 删除应用
// @Tags 身份认证-应用管理
// @Accept json
// @Produce json
// @Param id path int true "应用ID"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/apps/{id} [delete]
func (s *SSOApplicationService) DeleteApp(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := s.useCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除应用失败: "+err.Error())
		return
	}

	response.Success(c, "删除成功")
}

// GetTemplates 获取预置应用模板
// @Summary 获取预置应用模板
// @Description 获取预置的应用模板列表
// @Tags 身份认证-应用管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/identity/apps/templates [get]
func (s *SSOApplicationService) GetTemplates(c *gin.Context) {
	templates := s.useCase.GetTemplates()
	response.Success(c, templates)
}

// GetCategories 获取应用分类
// @Summary 获取应用分类
// @Description 获取应用分类列表
// @Tags 身份认证-应用管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/identity/apps/categories [get]
func (s *SSOApplicationService) GetCategories(c *gin.Context) {
	categories := []map[string]string{
		{"value": "cicd", "label": "CI/CD"},
		{"value": "code", "label": "代码管理"},
		{"value": "monitor", "label": "监控告警"},
		{"value": "registry", "label": "镜像仓库"},
		{"value": "other", "label": "其他"},
	}
	response.Success(c, categories)
}
