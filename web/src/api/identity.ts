import request from '@/utils/request'

// ============ 类型定义 ============

// 身份源
export interface IdentitySource {
  id: number
  name: string
  type: string
  icon: string
  config: string
  userMapping: string
  autoCreateUser: boolean
  defaultRoleId: number
  enabled: boolean
  sort: number
  createdAt: string
  updatedAt: string
}

// SSO应用
export interface SSOApplication {
  id: number
  name: string
  code: string
  icon: string
  description: string
  category: string
  url: string
  ssoType: string
  ssoConfig: string
  enabled: boolean
  sort: number
  createdAt: string
  updatedAt: string
}

// 应用模板
export interface AppTemplate {
  name: string
  code: string
  icon: string
  category: string
  description: string
  ssoType: string
  urlTemplate: string
}

// 门户应用
export interface PortalApp {
  id: number
  name: string
  code: string
  icon: string
  description: string
  category: string
  url: string
  isFavorite: boolean
}

// 用户凭证
export interface UserCredential {
  id: number
  appId: number
  appName: string
  appIcon: string
  username: string
  extraData: string
  createdAt: string
  updatedAt: string
}

// 应用权限
export interface AppPermission {
  id: number
  appId: number
  subjectType: string
  subjectId: number
  permission: string
  createdAt: string
}

// 认证日志
export interface AuthLog {
  id: number
  userId: number
  username: string
  action: string
  appId: number
  appName: string
  loginType: string
  ip: string
  location: string
  userAgent: string
  result: string
  failReason: string
  createdAt: string
}

// 认证统计
export interface AuthLogStats {
  totalLogins: number
  todayLogins: number
  failedLogins: number
  uniqueUsers: number
  appAccessCount: number
  loginTrend: TrendPoint[]
  topApps: TopAppStat[]
  topUsers: TopUserStat[]
}

export interface TrendPoint {
  date: string
  count: number
}

export interface TopAppStat {
  appId: number
  appName: string
  count: number
}

export interface TopUserStat {
  userId: number
  username: string
  count: number
}

// ============ 身份源管理 API ============

// 获取身份源列表
export const getIdentitySources = (params?: {
  page?: number
  pageSize?: number
  keyword?: string
  enabled?: boolean
}) => {
  return request.get('/api/v1/identity/sources', { params })
}

// 获取启用的身份源
export const getEnabledSources = () => {
  return request.get('/api/v1/identity/sources/enabled')
}

// 获取身份源详情
export const getIdentitySource = (id: number) => {
  return request.get(`/api/v1/identity/sources/${id}`)
}

// 创建身份源
export const createIdentitySource = (data: Partial<IdentitySource>) => {
  return request.post('/api/v1/identity/sources', data)
}

// 更新身份源
export const updateIdentitySource = (id: number, data: Partial<IdentitySource>) => {
  return request.put(`/api/v1/identity/sources/${id}`, data)
}

// 删除身份源
export const deleteIdentitySource = (id: number) => {
  return request.delete(`/api/v1/identity/sources/${id}`)
}

// ============ 应用管理 API ============

// 获取应用列表
export const getSSOApplications = (params?: {
  page?: number
  pageSize?: number
  keyword?: string
  category?: string
  enabled?: boolean
}) => {
  return request.get('/api/v1/identity/apps', { params })
}

// 获取应用模板
export const getAppTemplates = () => {
  return request.get('/api/v1/identity/apps/templates')
}

// 获取应用分类
export const getAppCategories = () => {
  return request.get('/api/v1/identity/apps/categories')
}

// 获取应用详情
export const getSSOApplication = (id: number) => {
  return request.get(`/api/v1/identity/apps/${id}`)
}

// 创建应用
export const createSSOApplication = (data: Partial<SSOApplication>) => {
  return request.post('/api/v1/identity/apps', data)
}

// 更新应用
export const updateSSOApplication = (id: number, data: Partial<SSOApplication>) => {
  return request.put(`/api/v1/identity/apps/${id}`, data)
}

// 删除应用
export const deleteSSOApplication = (id: number) => {
  return request.delete(`/api/v1/identity/apps/${id}`)
}

// ============ 应用门户 API ============

// 获取门户应用列表
export const getPortalApps = (params?: { category?: string }) => {
  return request.get('/api/v1/identity/portal/apps', { params })
}

// 获取收藏的应用
export const getFavoriteApps = () => {
  return request.get('/api/v1/identity/portal/favorites')
}

// 访问应用
export const accessApp = (id: number) => {
  return request.post(`/api/v1/identity/portal/access/${id}`)
}

// 收藏/取消收藏应用
export const toggleFavoriteApp = (id: number) => {
  return request.post(`/api/v1/identity/portal/favorite/${id}`)
}

// ============ 凭证管理 API ============

// 获取凭证列表
export const getUserCredentials = () => {
  return request.get('/api/v1/identity/credentials')
}

// 创建凭证
export const createUserCredential = (data: {
  appId: number
  username: string
  password?: string
  extraData?: string
}) => {
  return request.post('/api/v1/identity/credentials', data)
}

// 更新凭证
export const updateUserCredential = (id: number, data: {
  appId: number
  username: string
  password?: string
  extraData?: string
}) => {
  return request.put(`/api/v1/identity/credentials/${id}`, data)
}

// 删除凭证
export const deleteUserCredential = (id: number) => {
  return request.delete(`/api/v1/identity/credentials/${id}`)
}

// ============ 访问策略 API ============

// 获取权限列表
export const getAppPermissions = (params?: {
  page?: number
  pageSize?: number
  appId?: number
  subjectType?: string
}) => {
  return request.get('/api/v1/identity/permissions', { params })
}

// 创建权限
export const createAppPermission = (data: {
  appId: number
  subjectType: string
  subjectId: number
  permission?: string
}) => {
  return request.post('/api/v1/identity/permissions', data)
}

// 批量创建权限
export const batchCreateAppPermissions = (data: Array<{
  appId: number
  subjectType: string
  subjectId: number
  permission?: string
}>) => {
  return request.post('/api/v1/identity/permissions/batch', data)
}

// 删除权限
export const deleteAppPermission = (id: number) => {
  return request.delete(`/api/v1/identity/permissions/${id}`)
}

// 获取应用的权限列表
export const getAppPermissionsByApp = (appId: number) => {
  return request.get(`/api/v1/identity/permissions/app/${appId}`)
}

// ============ 认证日志 API ============

// 获取认证日志列表
export const getAuthLogs = (params?: {
  page?: number
  pageSize?: number
  userId?: number
  action?: string
  result?: string
  startTime?: string
  endTime?: string
}) => {
  return request.get('/api/v1/identity/logs', { params })
}

// 获取认证统计
export const getAuthLogStats = (params?: {
  startTime?: string
  endTime?: string
}) => {
  return request.get('/api/v1/identity/logs/stats', { params })
}

// 获取登录趋势
export const getLoginTrend = (days?: number) => {
  return request.get('/api/v1/identity/logs/trend', { params: { days } })
}
