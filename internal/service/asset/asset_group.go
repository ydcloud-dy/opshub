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

package asset

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/asset"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

type AssetGroupService struct {
	groupUseCase *asset.AssetGroupUseCase
}

func NewAssetGroupService(groupUseCase *asset.AssetGroupUseCase) *AssetGroupService {
	return &AssetGroupService{
		groupUseCase: groupUseCase,
	}
}

// CreateGroup 创建分组
// @Summary 创建资产分组
// @Description 创建新的资产分组
// @Tags 资产分组管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body asset.AssetGroupRequest true "分组信息"
// @Success 200 {object} response.Response "创建成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/asset-groups [post]
func (s *AssetGroupService) CreateGroup(c *gin.Context) {
	var req asset.AssetGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	group := req.ToModel()
	if err := s.groupUseCase.Create(c.Request.Context(), group); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, group)
}

// UpdateGroup 更新分组
// @Summary 更新资产分组
// @Description 更新指定的资产分组信息
// @Tags 资产分组管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "分组ID"
// @Param body body asset.AssetGroupRequest true "分组信息"
// @Success 200 {object} response.Response "更新成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/asset-groups/{id} [put]
func (s *AssetGroupService) UpdateGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的分组ID")
		return
	}

	var req asset.AssetGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	group := req.ToModel()
	group.ID = uint(id)
	if err := s.groupUseCase.Update(c.Request.Context(), group); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.Success(c, group)
}

// DeleteGroup 删除分组
// @Summary 删除资产分组
// @Description 删除指定的资产分组
// @Tags 资产分组管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "分组ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/asset-groups/{id} [delete]
func (s *AssetGroupService) DeleteGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的分组ID")
		return
	}

	if err := s.groupUseCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetGroup 获取分组详情
// @Summary 获取资产分组详情
// @Description 获取指定资产分组的详细信息
// @Tags 资产分组管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "分组ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 404 {object} response.Response "分组不存在"
// @Router /api/v1/asset-groups/{id} [get]
func (s *AssetGroupService) GetGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的分组ID")
		return
	}

	group, err := s.groupUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "分组不存在")
		return
	}

	response.Success(c, group)
}

// GetGroupTree 获取分组树
// @Summary 获取资产分组树
// @Description 获取资产分组的树形结构数据
// @Tags 资产分组管理
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/asset-groups/tree [get]
func (s *AssetGroupService) GetGroupTree(c *gin.Context) {
	tree, err := s.groupUseCase.GetTree(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	// 转换为VO格式
	var voTree []*asset.AssetGroupInfoVO
	for _, group := range tree {
		voTree = append(voTree, s.groupUseCase.ToInfoVO(group))
	}

	response.Success(c, voTree)
}

// GetParentOptions 获取父级分组选项
// @Summary 获取父级分组选项
// @Description 获取可选的父级资产分组列表
// @Tags 资产分组管理
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/asset-groups/parent-options [get]
func (s *AssetGroupService) GetParentOptions(c *gin.Context) {
	options, err := s.groupUseCase.GetParentOptions(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, options)
}
