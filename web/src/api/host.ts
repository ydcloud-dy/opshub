import request from '@/utils/request'

// 主机管理
export const getHostList = (params: any) => {
  return request.get('/api/v1/hosts', { params })
}

export const getHost = (id: number) => {
  return request.get(`/api/v1/hosts/${id}`)
}

export const createHost = (data: any) => {
  return request.post('/api/v1/hosts', data)
}

export const updateHost = (id: number, data: any) => {
  return request.put(`/api/v1/hosts/${id}`, data)
}

export const deleteHost = (id: number) => {
  return request.delete(`/api/v1/hosts/${id}`)
}

// 凭证管理
export const getCredentialList = (params: any) => {
  return request.get('/api/v1/credentials', { params })
}

export const getCredentials = () => {
  return request.get('/api/v1/credentials/all')
}

export const getCredential = (id: number) => {
  return request.get(`/api/v1/credentials/${id}`)
}

export const createCredential = (data: any) => {
  return request.post('/api/v1/credentials', data)
}

export const updateCredential = (id: number, data: any) => {
  return request.put(`/api/v1/credentials/${id}`, data)
}

export const deleteCredential = (id: number) => {
  return request.delete(`/api/v1/credentials/${id}`)
}

// 云平台账号管理
export const getCloudAccountList = (params: any) => {
  return request.get('/api/v1/cloud-accounts', { params })
}

export const getCloudAccounts = () => {
  return request.get('/api/v1/cloud-accounts/all')
}

export const getCloudAccount = (id: number) => {
  return request.get(`/api/v1/cloud-accounts/${id}`)
}

export const createCloudAccount = (data: any) => {
  return request.post('/api/v1/cloud-accounts', data)
}

export const updateCloudAccount = (id: number, data: any) => {
  return request.put(`/api/v1/cloud-accounts/${id}`, data)
}

export const deleteCloudAccount = (id: number) => {
  return request.delete(`/api/v1/cloud-accounts/${id}`)
}

export const importFromCloud = (data: any) => {
  return request.post('/api/v1/cloud-accounts/import', data)
}

export const getCloudInstances = (accountId: number, region: string) => {
  return request.get(`/api/v1/cloud-accounts/${accountId}/instances`, { params: { region } })
}

export const getCloudRegions = (accountId: number) => {
  return request.get(`/api/v1/cloud-accounts/${accountId}/regions`)
}

// 采集主机信息
export const collectHostInfo = (id: number) => {
  return request.post(`/api/v1/hosts/${id}/collect`)
}

// 测试主机连接
export const testHostConnection = (id: number) => {
  return request.post(`/api/v1/hosts/${id}/test`)
}

// 批量采集主机信息
export const batchCollectHostInfo = (data: { hostIds: number[] }) => {
  return request.post('/api/v1/hosts/batch-collect', data)
}

// 下载Excel导入模板
export const downloadExcelTemplate = () => {
  return request.get('/api/v1/hosts/template/download', { responseType: 'blob' })
}

// Excel批量导入主机
export const importFromExcel = (file: File, type?: string, groupId?: number) => {
  const formData = new FormData()
  formData.append('file', file)
  if (type) formData.append('type', type)
  if (groupId) formData.append('groupId', String(groupId))
  return request.post('/api/v1/hosts/import', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

// 批量删除主机
export const batchDeleteHosts = (hostIds: number[]) => {
  return request.post('/api/v1/hosts/batch-delete', { hostIds })
}
