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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/identity"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

// PortalService 门户服务
type PortalService struct {
	appUseCase      *identity.SSOApplicationUseCase
	permUseCase     *identity.AppPermissionUseCase
	favoriteUseCase *identity.UserFavoriteAppUseCase
	authLogUseCase  *identity.AuthLogUseCase
}

func NewPortalService(
	appUseCase *identity.SSOApplicationUseCase,
	permUseCase *identity.AppPermissionUseCase,
	favoriteUseCase *identity.UserFavoriteAppUseCase,
	authLogUseCase *identity.AuthLogUseCase,
) *PortalService {
	return &PortalService{
		appUseCase:      appUseCase,
		permUseCase:     permUseCase,
		favoriteUseCase: favoriteUseCase,
		authLogUseCase:  authLogUseCase,
	}
}

// GetPortalApps 获取用户可访问的应用
// @Summary 获取门户应用列表
// @Description 获取当前用户可访问的应用列表
// @Tags 身份认证-应用门户
// @Accept json
// @Produce json
// @Param category query string false "分类筛选"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/portal/apps [get]
func (s *PortalService) GetPortalApps(c *gin.Context) {
	userID := c.GetUint("userID")
	category := c.Query("category")

	// 获取所有启用的应用
	apps, err := s.appUseCase.GetEnabled(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取应用列表失败: "+err.Error())
		return
	}

	// 获取用户收藏的应用
	favorites, _ := s.favoriteUseCase.ListByUser(c.Request.Context(), userID)
	favoriteMap := make(map[uint]bool)
	for _, f := range favorites {
		favoriteMap[f.AppID] = true
	}

	// TODO: 根据用户权限过滤应用
	// 现在先返回所有启用的应用

	var portalApps []identity.PortalApp
	for _, app := range apps {
		if category != "" && app.Category != category {
			continue
		}
		portalApps = append(portalApps, identity.PortalApp{
			ID:          app.ID,
			Name:        app.Name,
			Code:        app.Code,
			Icon:        app.Icon,
			Description: app.Description,
			Category:    app.Category,
			URL:         app.URL,
			IsFavorite:  favoriteMap[app.ID],
		})
	}

	response.Success(c, portalApps)
}

// AccessApp 访问应用
// @Summary 访问应用
// @Description 访问应用并记录日志
// @Tags 身份认证-应用门户
// @Accept json
// @Produce json
// @Param id path int true "应用ID"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/portal/access/{id} [post]
func (s *PortalService) AccessApp(c *gin.Context) {
	userID := c.GetUint("userID")
	username := c.GetString("username")

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的ID")
		return
	}

	// 获取应用信息
	app, err := s.appUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "应用不存在")
		return
	}

	// TODO: 检查用户权限

	// 记录访问日志
	authLog := &identity.AuthLog{
		UserID:    userID,
		Username:  username,
		Action:    "access_app",
		AppID:     app.ID,
		AppName:   app.Name,
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Result:    "success",
		CreatedAt: time.Now(),
	}
	s.authLogUseCase.Create(c.Request.Context(), authLog)

	// 返回跳转URL
	response.Success(c, gin.H{
		"url": app.URL,
	})
}

// FavoriteApp 收藏应用
// @Summary 收藏应用
// @Description 添加或取消收藏应用
// @Tags 身份认证-应用门户
// @Accept json
// @Produce json
// @Param id path int true "应用ID"
// @Success 200 {object} response.Response
// @Router /api/v1/identity/portal/favorite/{id} [post]
func (s *PortalService) FavoriteApp(c *gin.Context) {
	userID := c.GetUint("userID")

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的ID")
		return
	}

	// 检查是否已收藏
	isFavorite, _ := s.favoriteUseCase.IsFavorite(c.Request.Context(), userID, uint(id))

	if isFavorite {
		// 取消收藏
		if err := s.favoriteUseCase.Remove(c.Request.Context(), userID, uint(id)); err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "取消收藏失败: "+err.Error())
			return
		}
		response.Success(c, gin.H{"isFavorite": false})
	} else {
		// 添加收藏
		if err := s.favoriteUseCase.Add(c.Request.Context(), userID, uint(id)); err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "收藏失败: "+err.Error())
			return
		}
		response.Success(c, gin.H{"isFavorite": true})
	}
}

// GetFavoriteApps 获取收藏的应用
// @Summary 获取收藏的应用
// @Description 获取当前用户收藏的应用列表
// @Tags 身份认证-应用门户
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/identity/portal/favorites [get]
func (s *PortalService) GetFavoriteApps(c *gin.Context) {
	userID := c.GetUint("userID")

	// 获取用户收藏的应用
	favorites, err := s.favoriteUseCase.ListByUser(c.Request.Context(), userID)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取收藏列表失败: "+err.Error())
		return
	}

	var portalApps []identity.PortalApp
	for _, f := range favorites {
		app, err := s.appUseCase.GetByID(c.Request.Context(), f.AppID)
		if err != nil {
			continue
		}
		portalApps = append(portalApps, identity.PortalApp{
			ID:          app.ID,
			Name:        app.Name,
			Code:        app.Code,
			Icon:        app.Icon,
			Description: app.Description,
			Category:    app.Category,
			URL:         app.URL,
			IsFavorite:  true,
		})
	}

	response.Success(c, portalApps)
}
