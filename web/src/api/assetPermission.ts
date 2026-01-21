import request from '@/utils/request'

// 获取资产权限列表
export const getAssetPermissions = (params: {
  page: number
  pageSize: number
  roleId?: number
  assetGroupId?: number
}) => {
  return request.get('/api/v1/asset-permissions', { params })
}

// 创建资产权限
export const createAssetPermission = (data: {
  roleId: number
  assetGroupId: number
  hostIds: number[]
  permissions?: number
}) => {
  return request.post('/api/v1/asset-permissions', data)
}

// 删除资产权限
export const deleteAssetPermission = (id: number) => {
  return request.delete(`/api/v1/asset-permissions/${id}`)
}

// 获取资产权限详情
export const getAssetPermissionDetail = (id: number) => {
  return request.get(`/api/v1/asset-permissions/${id}`)
}

// 更新资产权限
export const updateAssetPermission = (id: number, data: {
  roleId: number
  assetGroupId: number
  hostIds: number[]
  permissions?: number
}) => {
  return request.put(`/api/v1/asset-permissions/${id}`, data)
}

// 获取角色的资产权限
export const getAssetPermissionsByRole = (roleId: number) => {
  return request.get(`/api/v1/asset-permissions/role/${roleId}`)
}

// 获取资产分组的权限配置
export const getAssetPermissionsByGroup = (assetGroupId: number) => {
  return request.get(`/api/v1/asset-permissions/group/${assetGroupId}`)
}

// 获取当前用户对指定主机的操作权限
export const getUserHostPermissions = (hostId: number) => {
  return request.get('/api/v1/asset-permissions/user/host', {
    params: { hostId }
  })
}
