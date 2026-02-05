import request from '@/utils/request'

// 用户列表
export const getUserList = (params: any) => {
  return request.get('/api/v1/users', { params })
}

// 获取用户详情
export const getUser = (id: number) => {
  return request.get(`/api/v1/users/${id}`)
}

// 创建用户
export const createUser = (data: any) => {
  return request.post('/api/v1/users', data)
}

// 更新用户
export const updateUser = (id: number, data: any) => {
  return request.put(`/api/v1/users/${id}`, data)
}

// 删除用户
export const deleteUser = (id: number) => {
  return request.delete(`/api/v1/users/${id}`)
}

// 分配用户角色
export const assignUserRoles = (id: number, roleIds: number[]) => {
  return request.post(`/api/v1/users/${id}/roles`, { roleIds })
}

// 分配用户岗位
export const assignUserPositions = (id: number, positionIds: number[]) => {
  return request.post(`/api/v1/users/${id}/positions`, { positionIds })
}

// 重置用户密码
export const resetUserPassword = (id: number, password: string) => {
  return request.put(`/api/v1/users/${id}/reset-password`, { password })
}

// 解锁用户
export const unlockUser = (id: number) => {
  return request.post(`/api/v1/users/${id}/unlock`)
}

// 修改自己的密码
export const changePassword = (oldPassword: string, newPassword: string) => {
  return request.put('/api/v1/profile/password', { oldPassword, newPassword })
}
