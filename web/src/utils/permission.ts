// 权限常量定义 - 位掩码
export const PERMISSION = {
  VIEW: 1 << 0,     // 1 - 查看
  EDIT: 1 << 1,     // 2 - 编辑
  DELETE: 1 << 2,   // 4 - 删除
  TERMINAL: 1 << 3, // 8 - 终端
  FILE: 1 << 4,     // 16 - 文件管理
  COLLECT: 1 << 5,  // 32 - 采集信息
} as const

/**
 * 检查是否拥有指定权限
 * @param userPermissions 用户权限位掩码
 * @param requiredPermission 所需权限
 * @returns 是否拥有该权限
 */
export function hasPermission(userPermissions: number, requiredPermission: number): boolean {
  return (userPermissions & requiredPermission) > 0
}

/**
 * 获取权限的中文名称
 * @param permission 权限值
 * @returns 权限名称
 */
export function getPermissionName(permission: number): string {
  switch (permission) {
    case PERMISSION.VIEW:
      return '查看'
    case PERMISSION.EDIT:
      return '编辑'
    case PERMISSION.DELETE:
      return '删除'
    case PERMISSION.TERMINAL:
      return '终端'
    case PERMISSION.FILE:
      return '文件管理'
    case PERMISSION.COLLECT:
      return '采集信息'
    default:
      return '未知'
  }
}

/**
 * 获取权限位掩码中包含的所有权限名称
 * @param permissions 权限位掩码
 * @returns 权限名称数组
 */
export function getPermissionNames(permissions: number): string[] {
  const names: string[] = []
  if ((permissions & PERMISSION.VIEW) > 0) names.push('查看')
  if ((permissions & PERMISSION.EDIT) > 0) names.push('编辑')
  if ((permissions & PERMISSION.DELETE) > 0) names.push('删除')
  if ((permissions & PERMISSION.TERMINAL) > 0) names.push('终端')
  if ((permissions & PERMISSION.FILE) > 0) names.push('文件管理')
  if ((permissions & PERMISSION.COLLECT) > 0) names.push('采集信息')
  return names
}

/**
 * 将权限名称数组转换为权限位掩码
 * @param permissionNames 权限名称数组
 * @returns 权限位掩码
 */
export function permissionNamesToMask(permissionNames: string[]): number {
  let mask = 0
  for (const name of permissionNames) {
    switch (name) {
      case '查看':
        mask |= PERMISSION.VIEW
        break
      case '编辑':
        mask |= PERMISSION.EDIT
        break
      case '删除':
        mask |= PERMISSION.DELETE
        break
      case '终端':
        mask |= PERMISSION.TERMINAL
        break
      case '文件管理':
        mask |= PERMISSION.FILE
        break
      case '采集信息':
        mask |= PERMISSION.COLLECT
        break
    }
  }
  return mask
}

/**
 * 添加权限
 * @param permissions 原权限位掩码
 * @param permission 要添加的权限
 * @returns 新权限位掩码
 */
export function addPermission(permissions: number, permission: number): number {
  return permissions | permission
}

/**
 * 移除权限
 * @param permissions 原权限位掩码
 * @param permission 要移除的权限
 * @returns 新权限位掩码
 */
export function removePermission(permissions: number, permission: number): number {
  return permissions & ~permission
}

/**
 * 权限配置项，用于表单和显示
 */
export const PERMISSION_OPTIONS = [
  { label: '查看', value: PERMISSION.VIEW, description: '查看主机详情' },
  { label: '编辑', value: PERMISSION.EDIT, description: '创建、修改主机配置' },
  { label: '删除', value: PERMISSION.DELETE, description: '删除单个或批量删除主机' },
  { label: '连接终端', value: PERMISSION.TERMINAL, description: 'SSH连接到主机' },
  { label: '文件管理', value: PERMISSION.FILE, description: '文件上传、下载、删除' },
  { label: '采集信息', value: PERMISSION.COLLECT, description: '采集主机系统信息' },
]
