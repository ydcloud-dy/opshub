package rbac

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

type AssetPermissionService struct {
	assetPermissionUseCase *rbac.AssetPermissionUseCase
}

func NewAssetPermissionService(assetPermissionUseCase *rbac.AssetPermissionUseCase) *AssetPermissionService {
	return &AssetPermissionService{
		assetPermissionUseCase: assetPermissionUseCase,
	}
}

// CreateAssetPermission 创建资产权限
// @Summary 创建资产权限
// @Description 为角色创建资产访问权限
// @Tags 资产权限管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body rbac.AssetPermissionCreateReqWithPermissions true "权限信息"
// @Success 200 {object} response.Response "创建成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/asset-permissions [post]
func (s *AssetPermissionService) CreateAssetPermission(c *gin.Context) {
	var req rbac.AssetPermissionCreateReqWithPermissions
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 如果未指定权限，默认为查看权限
	if req.Permissions == 0 {
		req.Permissions = rbac.PermissionView
	}

	// 批量创建权限（带操作权限）
	if err := s.assetPermissionUseCase.CreateBatchWithPermissions(c.Request.Context(), req.RoleID, req.AssetGroupID, req.HostIDs, req.Permissions); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "创建成功", nil)
}

// DeleteAssetPermission 删除资产权限
// @Summary 删除资产权限
// @Description 删除指定的资产权限配置
// @Tags 资产权限管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "权限ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/asset-permissions/{id} [delete]
func (s *AssetPermissionService) DeleteAssetPermission(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的权限ID")
		return
	}

	if err := s.assetPermissionUseCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetAssetPermissionDetail 获取权限详情
// @Summary 获取资产权限详情
// @Description 获取指定资产权限的详细信息
// @Tags 资产权限管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "权限ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/asset-permissions/{id} [get]
func (s *AssetPermissionService) GetAssetPermissionDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的权限ID")
		return
	}

	detail, err := s.assetPermissionUseCase.GetDetailByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, detail)
}

// UpdateAssetPermission 更新权限配置
// @Summary 更新资产权限
// @Description 更新指定资产权限的配置
// @Tags 资产权限管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "权限ID"
// @Param body body rbac.AssetPermissionCreateReqWithPermissions true "权限信息"
// @Success 200 {object} response.Response "更新成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/asset-permissions/{id} [put]
func (s *AssetPermissionService) UpdateAssetPermission(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的权限ID")
		return
	}

	var req rbac.AssetPermissionCreateReqWithPermissions
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 如果未指定权限，默认为查看权限
	if req.Permissions == 0 {
		req.Permissions = rbac.PermissionView
	}

	if err := s.assetPermissionUseCase.UpdateAssetPermission(
		c.Request.Context(),
		uint(id),
		req.RoleID,
		req.AssetGroupID,
		req.HostIDs,
		req.Permissions,
	); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "更新成功", nil)
}

// DeleteAssetPermissionByRoleAndGroup 删除指定角色对指定资产分组的所有权限
// @Summary 按角色和分组删除权限
// @Description 删除指定角色对指定资产分组的所有权限配置
// @Tags 资产权限管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param roleId query int true "角色ID"
// @Param assetGroupId query int true "资产分组ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/asset-permissions [delete]
func (s *AssetPermissionService) DeleteAssetPermissionByRoleAndGroup(c *gin.Context) {
	roleIDStr := c.Query("roleId")
	assetGroupIDStr := c.Query("assetGroupId")

	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的角色ID")
		return
	}

	assetGroupID, err := strconv.ParseUint(assetGroupIDStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的资产分组ID")
		return
	}

	if err := s.assetPermissionUseCase.DeleteByRoleAndGroup(c.Request.Context(), uint(roleID), uint(assetGroupID)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// ListAssetPermissions 资产权限列表
// @Summary 获取资产权限列表
// @Description 分页获取资产权限列表
// @Tags 资产权限管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param roleId query int false "角色ID"
// @Param assetGroupId query int false "资产分组ID"
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/asset-permissions [get]
func (s *AssetPermissionService) ListAssetPermissions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	var roleID *uint
	var assetGroupID *uint

	if roleIDStr := c.Query("roleId"); roleIDStr != "" {
		id, err := strconv.ParseUint(roleIDStr, 10, 32)
		if err == nil {
			roleIDVal := uint(id)
			roleID = &roleIDVal
		}
	}

	if assetGroupIDStr := c.Query("assetGroupId"); assetGroupIDStr != "" {
		id, err := strconv.ParseUint(assetGroupIDStr, 10, 32)
		if err == nil {
			assetGroupIDVal := uint(id)
			assetGroupID = &assetGroupIDVal
		}
	}

	list, total, err := s.assetPermissionUseCase.List(c.Request.Context(), page, pageSize, roleID, assetGroupID)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"total": total,
		"list":  list,
	})
}

// GetAssetPermissionsByRole 获取角色的所有资产权限
// @Summary 获取角色资产权限
// @Description 获取指定角色的所有资产权限配置
// @Tags 资产权限管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param roleId path int true "角色ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/asset-permissions/role/{roleId} [get]
func (s *AssetPermissionService) GetAssetPermissionsByRole(c *gin.Context) {
	roleIDStr := c.Param("roleId")
	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的角色ID")
		return
	}

	permissions, err := s.assetPermissionUseCase.GetByRoleID(c.Request.Context(), uint(roleID))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, permissions)
}

// GetAssetPermissionsByGroup 获取资产分组的所有权限配置
// @Summary 获取分组资产权限
// @Description 获取指定资产分组的所有权限配置
// @Tags 资产权限管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param assetGroupId path int true "资产分组ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 400 {object} response.Response "参数错误"
// @Router /api/v1/asset-permissions/group/{assetGroupId} [get]
func (s *AssetPermissionService) GetAssetPermissionsByGroup(c *gin.Context) {
	assetGroupIDStr := c.Param("assetGroupId")
	assetGroupID, err := strconv.ParseUint(assetGroupIDStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的资产分组ID")
		return
	}

	permissions, err := s.assetPermissionUseCase.GetByAssetGroupID(c.Request.Context(), uint(assetGroupID))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, permissions)
}

// GetUserHostPermissions 获取当前用户对指定主机的所有操作权限
// @Summary 获取用户主机权限
// @Description 获取当前登录用户对指定主机的操作权限
// @Tags 资产权限管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param hostId query int true "主机ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未授权"
// @Router /api/v1/asset-permissions/user/host [get]
func (s *AssetPermissionService) GetUserHostPermissions(c *gin.Context) {
	hostIDStr := c.Query("hostId")
	if hostIDStr == "" {
		response.ErrorCode(c, http.StatusBadRequest, "主机ID不能为空")
		return
	}

	hostID, err := strconv.ParseUint(hostIDStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		response.ErrorCode(c, http.StatusUnauthorized, "未授权")
		return
	}

	permissions, err := s.assetPermissionUseCase.GetUserHostPermissions(c.Request.Context(), userID, uint(hostID))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"permissions": permissions,
	})
}
