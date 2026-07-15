import request from '@/utils/request'

// 获取菜单树
export const getMenuTree = () => {
  return request.get<any, any[]>('/api/v1/menus/tree')
}

// 获取当前用户的菜单树
export const getUserMenu = () => {
  return request.get<any, any[]>('/api/v1/menus/user')
}

// 获取菜单详情
export const getMenu = (id: number) => {
  return request.get(`/api/v1/menus/${id}`)
}

// 创建菜单
export const createMenu = (data: any) => {
  return request.post('/api/v1/menus', data)
}

// 更新菜单
export const updateMenu = (id: number, data: any) => {
  return request.put(`/api/v1/menus/${id}`, data)
}

// 删除菜单
export const deleteMenu = (id: number) => {
  return request.delete(`/api/v1/menus/${id}`)
}
