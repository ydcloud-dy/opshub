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

// CredentialService 凭证服务
type CredentialService struct {
	useCase    *identity.UserCredentialUseCase
	appUseCase *identity.SSOApplicationUseCase
}

func NewCredentialService(useCase *identity.UserCredentialUseCase, appUseCase *identity.SSOApplicationUseCase) *CredentialService {
	return &CredentialService{
		useCase:    useCase,
		appUseCase: appUseCase,
	}
}

// CredentialRequest 凭证请求
type CredentialRequest struct {
	AppID     uint   `json:"appId" binding:"required"`
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password"`
	ExtraData string `json:"extraData"`
}

// CredentialResponse 凭证响应
type CredentialResponse struct {
	ID        uint   `json:"id"`
	AppID     uint   `json:"appId"`
	AppName   string `json:"appName"`
	AppIcon   string `json:"appIcon"`
	Username  string `json:"username"`
	ExtraData string `json:"extraData"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// ListCredentials 获取凭证列表
// @Summary 获取凭证列表
// @Description 获取当前用户的凭证列表
// @Tags 身份认证-凭证管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/identity/credentials [get]
func (s *CredentialService) ListCredentials(c *gin.Context) {
	userID := c.GetUint("userID")

	credentials, err := s.useCase.ListByUser(c.Request.Context(), userID)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取凭证列表失败: "+err.Error())
		return
	}

	var result []CredentialResponse
	for _, cred := range credentials {
		app, _ := s.appUseCase.GetByID(c.Request.Context(), cred.AppID)
		resp := CredentialResponse{
			ID:        cred.ID,
			AppID:     cred.AppID,
			Username:  cred.Username,
			ExtraData: cred.ExtraData,
			CreatedAt: cred.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: cred.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		if app != nil {
			resp.AppName = app.Name
			resp.AppIcon = app.Icon
		}
		result = append(result, resp)
	}

	response.Success(c, result)
}

// CreateCredential 创建凭证
// @Summary 创建凭证
// @Description 为当前用户创建应用凭证
// @Tags 身份认证-凭证管理
// @Accept json
// @Produce json
// @Param body body CredentialRequest true "凭证信息"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/credentials [post]
func (s *CredentialService) CreateCredential(c *gin.Context) {
	userID := c.GetUint("userID")

	var req CredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 检查应用是否存在
	_, err := s.appUseCase.GetByID(c.Request.Context(), req.AppID)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "应用不存在")
		return
	}

	// 检查是否已存在凭证
	existing, _ := s.useCase.GetByUserAndApp(c.Request.Context(), userID, req.AppID)
	if existing != nil && existing.ID > 0 {
		response.ErrorCode(c, http.StatusBadRequest, "该应用已存在凭证")
		return
	}

	credential := &identity.UserCredential{
		UserID:    userID,
		AppID:     req.AppID,
		Username:  req.Username,
		Password:  req.Password,
		ExtraData: req.ExtraData,
	}

	if err := s.useCase.Create(c.Request.Context(), credential); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建凭证失败: "+err.Error())
		return
	}

	response.Success(c, "创建成功")
}

// UpdateCredential 更新凭证
// @Summary 更新凭证
// @Description 更新凭证信息
// @Tags 身份认证-凭证管理
// @Accept json
// @Produce json
// @Param id path int true "凭证ID"
// @Param body body CredentialRequest true "凭证信息"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/credentials/{id} [put]
func (s *CredentialService) UpdateCredential(c *gin.Context) {
	userID := c.GetUint("userID")

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的ID")
		return
	}

	// 检查凭证是否存在且属于当前用户
	credential, err := s.useCase.GetByID(c.Request.Context(), uint(id))
	if err != nil || credential.UserID != userID {
		response.ErrorCode(c, http.StatusNotFound, "凭证不存在")
		return
	}

	var req CredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	credential.Username = req.Username
	if req.Password != "" {
		credential.Password = req.Password
	}
	credential.ExtraData = req.ExtraData

	if err := s.useCase.Update(c.Request.Context(), credential); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新凭证失败: "+err.Error())
		return
	}

	response.Success(c, "更新成功")
}

// DeleteCredential 删除凭证
// @Summary 删除凭证
// @Description 删除凭证
// @Tags 身份认证-凭证管理
// @Accept json
// @Produce json
// @Param id path int true "凭证ID"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/credentials/{id} [delete]
func (s *CredentialService) DeleteCredential(c *gin.Context) {
	userID := c.GetUint("userID")

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的ID")
		return
	}

	// 检查凭证是否存在且属于当前用户
	credential, err := s.useCase.GetByID(c.Request.Context(), uint(id))
	if err != nil || credential.UserID != userID {
		response.ErrorCode(c, http.StatusNotFound, "凭证不存在")
		return
	}

	if err := s.useCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除凭证失败: "+err.Error())
		return
	}

	response.Success(c, "删除成功")
}
