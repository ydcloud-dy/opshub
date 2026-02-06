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
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"github.com/ydcloud-dy/opshub/internal/biz/asset"
	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	rbacService "github.com/ydcloud-dy/opshub/internal/service/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

type HostService struct {
	hostUseCase            *asset.HostUseCase
	credentialUseCase      *asset.CredentialUseCase
	cloudUseCase           *asset.CloudAccountUseCase
	assetPermissionUseCase *rbac.AssetPermissionUseCase
}

func NewHostService(hostUseCase *asset.HostUseCase, credentialUseCase *asset.CredentialUseCase, cloudUseCase *asset.CloudAccountUseCase, assetPermissionUseCase *rbac.AssetPermissionUseCase) *HostService {
	return &HostService{
		hostUseCase:            hostUseCase,
		credentialUseCase:      credentialUseCase,
		cloudUseCase:           cloudUseCase,
		assetPermissionUseCase: assetPermissionUseCase,
	}
}

// CreateHost 创建主机
// @Summary 创建主机
// @Description 创建新的主机资源
// @Tags 资产管理-主机
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body asset.HostRequest true "主机信息"
// @Success 200 {object} response.Response{} "创建成功"
// @Router /api/v1/hosts [post]
func (s *HostService) CreateHost(c *gin.Context) {
	var req asset.HostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	host, err := s.hostUseCase.Create(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, host)
}

// UpdateHost 更新主机
// @Summary 更新主机信息
// @Description 更新已有的主机信息
// @Tags 资产管理-主机
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "主机ID"
// @Param body body asset.HostRequest true "主机信息"
// @Success 200 {object} response.Response "更新成功"
// @Router /api/v1/hosts/{id} [put]
func (s *HostService) UpdateHost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	var req asset.HostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	req.ID = uint(id)
	if err := s.hostUseCase.Update(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "更新成功", nil)
}

// DeleteHost 删除主机
// @Summary 删除主机
// @Description 删除指定的主机资源
// @Tags 资产管理-主机
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "主机ID"
// @Success 200 {object} response.Response "删除成功"
// @Router /api/v1/hosts/{id} [delete]
func (s *HostService) DeleteHost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	if err := s.hostUseCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetHost 获取主机详情
// @Summary 获取主机详情
// @Description 获取单个主机的详细信息
// @Tags 资产管理-主机
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "主机ID"
// @Success 200 {object} response.Response{} "获取成功"
// @Router /api/v1/hosts/{id} [get]
func (s *HostService) GetHost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	host, err := s.hostUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "主机不存在")
		return
	}

	response.Success(c, host)
}

// ListHosts 主机列表
// @Summary 获取主机列表
// @Description 分页获取主机列表，支持搜索和按分组筛选
// @Tags 资产管理-主机
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键字"
// @Param groupId query int false "分组ID"
// @Param status query int false "主机状态(1:在线 0:离线 -1:未知)"
// @Success 200 {object} response.Response{} "获取成功"
// @Router /api/v1/hosts [get]
func (s *HostService) ListHosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")

	// 支持分组ID筛选
	var groupID *uint
	if groupIDStr := c.Query("groupId"); groupIDStr != "" {
		id, err := strconv.ParseUint(groupIDStr, 10, 32)
		if err == nil {
			gid := uint(id)
			groupID = &gid
		}
	}

	// 支持状态筛选
	var status *int
	if statusStr := c.Query("status"); statusStr != "" {
		s, err := strconv.Atoi(statusStr)
		if err == nil {
			status = &s
		}
	}

	// 获取用户ID并检查权限
	var accessibleHostIDs []uint
	userID := rbacService.GetUserID(c)
	if userID > 0 {
		// 获取用户可访问的主机ID列表
		hostIDs, err := s.assetPermissionUseCase.GetUserAccessibleHostIDs(c.Request.Context(), userID)
		if err == nil {
			accessibleHostIDs = hostIDs
		} else {
			// 如果获取权限出错，返回空列表以保证安全
			accessibleHostIDs = []uint{}
		}
	}

	hosts, total, err := s.hostUseCase.List(c.Request.Context(), page, pageSize, keyword, groupID, accessibleHostIDs, status)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":     hosts,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// CreateCredential 创建凭证
// @Summary 创建凭证
// @Description 创建新的凭证用于主机连接
// @Tags 资产管理-凭证
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body asset.CredentialRequest true "凭证信息"
// @Success 200 {object} response.Response{} "创建成功"
// @Router /api/v1/credentials [post]
func (s *HostService) CreateCredential(c *gin.Context) {
	var req asset.CredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	credential, err := s.credentialUseCase.Create(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, credential)
}

// UpdateCredential 更新凭证
// @Summary 更新凭证
// @Description 更新已有的凭证信息
// @Tags 资产管理-凭证
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "凭证ID"
// @Param body body asset.CredentialRequest true "凭证信息"
// @Success 200 {object} response.Response "更新成功"
// @Router /api/v1/credentials/{id} [put]
func (s *HostService) UpdateCredential(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的凭证ID")
		return
	}

	var req asset.CredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	req.ID = uint(id)
	if err := s.credentialUseCase.Update(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "更新成功", nil)
}

// DeleteCredential 删除凭证
// @Summary 删除凭证
// @Description 删除指定的凭证
// @Tags 资产管理-凭证
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "凭证ID"
// @Success 200 {object} response.Response "删除成功"
// @Router /api/v1/credentials/{id} [delete]
func (s *HostService) DeleteCredential(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的凭证ID")
		return
	}

	if err := s.credentialUseCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetCredential 获取凭证详情
// @Summary 获取凭证详情
// @Description 获取单个凭证的详细信息（包含解密的私钥）
// @Tags 资产管理-凭证
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "凭证ID"
// @Success 200 {object} response.Response{} "获取成功"
// @Router /api/v1/credentials/{id} [get]
func (s *HostService) GetCredential(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的凭证ID")
		return
	}

	// 获取解密后的凭证详情（用于编辑时回显私钥）
	credential, err := s.credentialUseCase.GetByIDDecrypted(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "凭证不存在")
		return
	}

	response.Success(c, credential)
}

// ListCredentials 凭证列表
// @Summary 获取凭证列表
// @Description 分页获取凭证列表，支持搜索
// @Tags 资产管理-凭证
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键字"
// @Success 200 {object} response.Response{} "获取成功"
// @Router /api/v1/credentials [get]
func (s *HostService) ListCredentials(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")

	credentials, total, err := s.credentialUseCase.List(c.Request.Context(), page, pageSize, keyword)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":     credentials,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetAllCredentials 获取所有凭证（用于下拉选择）
// @Summary 获取所有凭证
// @Description 获取所有凭证列表，用于下拉选择
// @Tags 资产管理-凭证
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/credentials/all [get]
func (s *HostService) GetAllCredentials(c *gin.Context) {
	credentials, err := s.credentialUseCase.GetAll(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, credentials)
}

// CreateCloudAccount 创建云平台账号
// @Summary 创建云平台账号
// @Description 创建新的云平台账号配置
// @Tags 资产管理-云账号
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body asset.CloudAccountRequest true "云账号信息"
// @Success 200 {object} response.Response "创建成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/cloud-accounts [post]
func (s *HostService) CreateCloudAccount(c *gin.Context) {
	var req asset.CloudAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	account, err := s.cloudUseCase.Create(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, account)
}

// UpdateCloudAccount 更新云平台账号
// @Summary 更新云平台账号
// @Description 更新指定的云平台账号配置
// @Tags 资产管理-云账号
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "云账号ID"
// @Param body body asset.CloudAccountRequest true "云账号信息"
// @Success 200 {object} response.Response "更新成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/cloud-accounts/{id} [put]
func (s *HostService) UpdateCloudAccount(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的账号ID")
		return
	}

	var req asset.CloudAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	req.ID = uint(id)
	if err := s.cloudUseCase.Update(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "更新成功", nil)
}

// DeleteCloudAccount 删除云平台账号
// @Summary 删除云平台账号
// @Description 删除指定的云平台账号
// @Tags 资产管理-云账号
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "云账号ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/cloud-accounts/{id} [delete]
func (s *HostService) DeleteCloudAccount(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的账号ID")
		return
	}

	if err := s.cloudUseCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetCloudAccount 获取云平台账号详情
// @Summary 获取云平台账号详情
// @Description 获取指定云平台账号的详细信息
// @Tags 资产管理-云账号
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "云账号ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 404 {object} response.Response "账号不存在"
// @Router /api/v1/cloud-accounts/{id} [get]
func (s *HostService) GetCloudAccount(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的账号ID")
		return
	}

	account, err := s.cloudUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "账号不存在")
		return
	}

	response.Success(c, account)
}

// ListCloudAccounts 云平台账号列表
// @Summary 获取云平台账号列表
// @Description 分页获取云平台账号列表
// @Tags 资产管理-云账号
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/cloud-accounts [get]
func (s *HostService) ListCloudAccounts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	accounts, total, err := s.cloudUseCase.List(c.Request.Context(), page, pageSize)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":     accounts,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetAllCloudAccounts 获取所有启用的云平台账号
// @Summary 获取所有云平台账号
// @Description 获取所有启用的云平台账号列表
// @Tags 资产管理-云账号
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/cloud-accounts/all [get]
func (s *HostService) GetAllCloudAccounts(c *gin.Context) {
	accounts, err := s.cloudUseCase.GetAll(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, accounts)
}

// GetCloudInstances 获取云平台的实例列表
// @Summary 获取云平台实例列表
// @Description 获取指定云账号在指定区域的实例列表
// @Tags 资产管理-云账号
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "云账号ID"
// @Param region query string true "区域"
// @Success 200 {object} response.Response "获取成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/cloud-accounts/{id}/instances [get]
func (s *HostService) GetCloudInstances(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的账号ID")
		return
	}

	// 从查询参数获取区域
	region := c.Query("region")
	if region == "" {
		response.ErrorCode(c, http.StatusBadRequest, "请指定区域")
		return
	}

	instances, err := s.cloudUseCase.GetInstances(c.Request.Context(), uint(id), region)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取实例列表失败: "+err.Error())
		return
	}

	response.Success(c, instances)
}

// GetCloudRegions 获取云平台的区域列表
// @Summary 获取云平台区域列表
// @Description 获取指定云账号可用的区域列表
// @Tags 资产管理-云账号
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "云账号ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/cloud-accounts/{id}/regions [get]
func (s *HostService) GetCloudRegions(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的账号ID")
		return
	}

	regions, err := s.cloudUseCase.GetRegions(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取区域列表失败: "+err.Error())
		return
	}

	response.Success(c, regions)
}

// ImportFromCloud 从云平台导入主机
// @Summary 从云平台导入主机
// @Description 从云平台批量导入主机资源
// @Tags 资产管理-云账号
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body asset.CloudImportRequest true "导入配置"
// @Success 200 {object} response.Response "导入成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/cloud-accounts/import [post]
func (s *HostService) ImportFromCloud(c *gin.Context) {
	var req asset.CloudImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.cloudUseCase.ImportFromCloud(c.Request.Context(), &req, s.hostUseCase); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "导入失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "导入成功", nil)
}

// CollectHostInfo 采集主机信息
// @Summary 采集主机信息
// @Description 采集指定主机的系统信息
// @Tags 资产管理-主机
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "主机ID"
// @Success 200 {object} response.Response "采集成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/hosts/{id}/collect [post]
func (s *HostService) CollectHostInfo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	if err := s.hostUseCase.CollectHostInfo(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "采集失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "采集成功", nil)
}

// TestHostConnection 测试主机连接
// @Summary 测试主机连接
// @Description 测试指定主机的SSH连接是否正常
// @Tags 资产管理-主机
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "主机ID"
// @Success 200 {object} response.Response "连接成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/hosts/{id}/test [post]
func (s *HostService) TestHostConnection(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	if err := s.hostUseCase.TestConnection(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "连接失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "连接成功", nil)
}

// BatchCollectHostInfo 批量采集主机信息
// @Summary 批量采集主机信息
// @Description 批量采集多个主机的系统信息
// @Tags 资产管理-主机
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body object true "主机ID列表 {hostIds: []uint}"
// @Success 200 {object} response.Response "采集完成"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/hosts/batch-collect [post]
func (s *HostService) BatchCollectHostInfo(c *gin.Context) {
	var req struct {
		HostIDs []uint `json:"hostIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.hostUseCase.BatchCollectHostInfo(c.Request.Context(), req.HostIDs); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "批量采集失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "批量采集完成", nil)
}

// BatchDeleteHosts 批量删除主机
// @Summary 批量删除主机
// @Description 批量删除多个主机资源
// @Tags 资产管理-主机
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body object true "主机ID列表 {hostIds: []uint}"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/hosts/batch-delete [post]
func (s *HostService) BatchDeleteHosts(c *gin.Context) {
	var req struct {
		HostIDs []uint `json:"hostIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.hostUseCase.BatchDelete(c.Request.Context(), req.HostIDs); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "批量删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "批量删除成功", nil)
}

// DownloadExcelTemplate 下载Excel导入模板
// @Summary 下载导入模板
// @Description 下载主机Excel导入模板文件
// @Tags 资产管理-主机
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security Bearer
// @Success 200 {file} file "Excel模板文件"
// @Router /api/v1/hosts/template/download [get]
func (s *HostService) DownloadExcelTemplate(c *gin.Context) {
	f := excelize.NewFile()
	// 获取默认sheet名称 (默认是 "Sheet1")
	sheetName := f.GetSheetName(0)

	// 设置列标题
	headers := []string{"主机名称*", "分组编码", "SSH用户名*", "IP地址*", "SSH端口*", "凭证名称", "标签", "备注"}

	// 创建标题行样式
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E6E6FA"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})

	// 设置列宽
	f.SetColWidth(sheetName, "A", "H", 20)

	// 写入标题行
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	// 添加示例数据
	examples := [][]string{
		{"Web服务器-01", "BIBF", "root", "192.168.1.100", "22", "生产环境凭证", "web,生产", "Web应用服务器"},
		{"数据库服务器", "TEST", "ubuntu", "192.168.1.200", "22", "", "db,测试", "MySQL数据库服务器"},
	}
	for i, example := range examples {
		for j, val := range example {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			f.SetCellValue(sheetName, cell, val)
		}
	}

	// 添加说明行
	for i, note := range []string{
		"说明：",
		"1. 带*号的是必填项",
		"2. 分组编码：需要在系统中已存在，可在资产分组管理中查看",
		"3. SSH端口：默认22",
		"4. 凭证名称：需要在系统中已存在，可在凭证管理中查看",
		"5. 标签：多个标签用逗号分隔",
	} {
		cell, _ := excelize.CoordinatesToCellName(1, 5+i)
		f.SetCellValue(sheetName, cell, note)
	}

	// 生成到buffer
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "生成模板文件失败")
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=host_import_template.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// ImportFromExcel Excel批量导入主机
// @Summary Excel批量导入主机
// @Description 通过Excel文件批量导入主机
// @Tags 资产管理-主机
// @Accept multipart/form-data
// @Produce json
// @Security Bearer
// @Param file formData file true "Excel文件"
// @Param type formData string false "主机类型" default(self)
// @Param groupId formData int false "分组ID"
// @Success 200 {object} response.Response "导入结果"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/hosts/import [post]
func (s *HostService) ImportFromExcel(c *gin.Context) {
	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "请选择要上传的文件")
		return
	}

	// 获取主机类型（默认为自建主机）
	hostType := c.PostForm("type")
	if hostType == "" {
		hostType = "self"
	}

	// 获取分组ID
	var groupID uint
	if groupIDStr := c.PostForm("groupId"); groupIDStr != "" {
		id, err := strconv.ParseUint(groupIDStr, 10, 32)
		if err == nil {
			groupID = uint(id)
		}
	}

	// 检查文件类型
	if file.Header.Get("Content-Type") != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" &&
		!strings.HasSuffix(file.Filename, ".xlsx") {
		response.ErrorCode(c, http.StatusBadRequest, "请上传Excel文件(.xlsx格式)")
		return
	}

	// 读取文件内容
	src, err := file.Open()
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "打开文件失败")
		return
	}
	defer src.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(src); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "读取文件失败")
		return
	}

	// 调用导入方法
	result, err := s.hostUseCase.ImportFromExcelWithType(c.Request.Context(), buf.Bytes(), hostType, groupID)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "导入失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

// ListHostFiles 列出主机文件
// @Summary 获取主机文件列表
// @Description 获取指定主机上指定目录的文件列表
// @Tags 资产管理-主机文件
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "主机ID"
// @Param path query string false "目录路径" default(~)
// @Success 200 {object} response.Response "获取成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/hosts/{id}/files [get]
func (s *HostService) ListHostFiles(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	// 获取目录路径参数，默认为用户主目录
	remotePath := c.DefaultQuery("path", "~")

	files, err := s.hostUseCase.ListFiles(c.Request.Context(), uint(id), remotePath)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取文件列表失败: "+err.Error())
		return
	}

	response.Success(c, files)
}

// UploadHostFile 上传文件到主机
// @Summary 上传文件到主机
// @Description 将文件上传到指定主机的指定目录
// @Tags 资产管理-主机文件
// @Accept multipart/form-data
// @Produce json
// @Security Bearer
// @Param id path int true "主机ID"
// @Param file formData file true "上传的文件"
// @Param path formData string false "远程目录路径" default(~/)
// @Success 200 {object} response.Response "上传成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/hosts/{id}/files/upload [post]
func (s *HostService) UploadHostFile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "请选择要上传的文件")
		return
	}

	// 获取远程路径参数
	remotePath := c.PostForm("path")
	if remotePath == "" {
		remotePath = "~/"
	}

	// 打开文件
	src, err := file.Open()
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "打开文件失败")
		return
	}
	defer src.Close()

	// 上传文件
	if err := s.hostUseCase.UploadFile(c.Request.Context(), uint(id), src, remotePath, file.Filename); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "上传文件失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "文件上传成功", nil)
}

// DownloadHostFile 从主机下载文件
// @Summary 从主机下载文件
// @Description 从指定主机下载指定路径的文件
// @Tags 资产管理-主机文件
// @Accept json
// @Produce application/octet-stream
// @Security Bearer
// @Param id path int true "主机ID"
// @Param path query string true "文件路径"
// @Success 200 {file} file "文件内容"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/hosts/{id}/files/download [get]
func (s *HostService) DownloadHostFile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	// 获取文件路径参数
	remotePath := c.Query("path")
	if remotePath == "" {
		response.ErrorCode(c, http.StatusBadRequest, "请指定文件路径")
		return
	}

	// 获取文件名
	fileName := remotePath
	if idx := strings.LastIndex(remotePath, "/"); idx >= 0 {
		fileName = remotePath[idx+1:]
	}

	// 设置响应头
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	c.Header("Content-Transfer-Encoding", "binary")

	// 下载文件
	if err := s.hostUseCase.DownloadFile(c.Request.Context(), uint(id), remotePath, c.Writer); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "下载文件失败: "+err.Error())
		return
	}
}

// DeleteHostFile 删除主机文件
// @Summary 删除主机文件
// @Description 删除指定主机上的指定文件
// @Tags 资产管理-主机文件
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "主机ID"
// @Param body body object true "文件路径 {path: string}"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/hosts/{id}/files [delete]
func (s *HostService) DeleteHostFile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.hostUseCase.DeleteFile(c.Request.Context(), uint(id), req.Path); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除文件失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "文件删除成功", nil)
}
