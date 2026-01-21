<template>
  <el-container class="layout-container">
    <el-aside width="260px" v-if="!hideSidebar">
      <div class="logo">
        <h3>OpsHub</h3>
      </div>

      <el-menu
        :default-active="activeMenu"
        class="el-menu-vertical"
        router
        :unique-opened="true"
        background-color="#001529"
        text-color="#fff"
        active-text-color="#fff"
      >
        <template v-for="menu in menuList" :key="menu.ID">
          <!-- 有子菜单的情况 -->
          <el-sub-menu v-if="menu.children && menu.children.length > 0" :index="String(menu.ID)" :class="{ 'menu-disabled': menu.status === 0 }">
            <template #title>
              <el-icon><component :is="getIcon(menu.icon)" /></el-icon>
              <span>{{ menu.name }}</span>
            </template>
            <el-menu-item
              v-for="subMenu in menu.children"
              :key="subMenu.ID"
              :index="subMenu.status === 0 ? undefined : subMenu.path"
              :class="{ 'menu-disabled': subMenu.status === 0 }"
            >
              <el-icon><component :is="getIcon(subMenu.icon)" /></el-icon>
              <span>{{ subMenu.name }}</span>
            </el-menu-item>
          </el-sub-menu>

          <!-- 没有子菜单的情况 -->
          <el-menu-item
            v-else
            :index="menu.status === 0 ? undefined : menu.path"
            :class="{ 'menu-disabled': menu.status === 0 }"
          >
            <el-icon><component :is="getIcon(menu.icon)" /></el-icon>
            <span>{{ menu.name }}</span>
          </el-menu-item>
        </template>
      </el-menu>

      <!-- 用户信息区域 - 放在底部 -->
      <div class="user-section">
        <el-dropdown trigger="click" @command="handleUserCommand">
          <div class="user-info-wrapper">
            <div class="user-avatar">
              <el-avatar :size="40" :src="avatarUrl" :key="userStore.avatarTimestamp">
                <el-icon><UserFilled /></el-icon>
              </el-avatar>
            </div>
            <div class="user-details">
              <div class="user-name">{{ userStore.userInfo?.realName || userStore.userInfo?.username }}</div>
              <div class="user-role">{{ userRoleDisplay }}</div>
            </div>
            <el-icon class="dropdown-icon"><ArrowDown /></el-icon>
          </div>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="profile">
                <el-icon><User /></el-icon>
                <span>个人信息</span>
              </el-dropdown-item>
              <el-dropdown-item command="logout" divided>
                <el-icon><SwitchButton /></el-icon>
                <span>退出登录</span>
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </el-aside>

    <el-container>
      <el-header>
        <div class="header-content">
          <div class="breadcrumb">
            <el-breadcrumb separator="/">
              <el-breadcrumb-item :to="{ path: '/' }">首页</el-breadcrumb-item>
              <el-breadcrumb-item v-if="currentRoute.meta.title">
                {{ currentRoute.meta.title }}
              </el-breadcrumb-item>
            </el-breadcrumb>
          </div>
        </div>
      </el-header>

      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted, shallowRef } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { ElMessage } from 'element-plus'
import {
  HomeFilled,
  User,
  UserFilled,
  OfficeBuilding,
  Menu,
  SwitchButton,
  ArrowDown,
  Platform,
  Setting,
  Document,
  Tools,
  Monitor,
  FolderOpened,
  Connection,
  Files,
  Lock,
  View,
  Odometer,
  Tickets,
  List,
  Grid,
  Cloudy,
  Grape,
  House
} from '@element-plus/icons-vue'
import { getUserMenu } from '@/api/menu'
import { pluginManager } from '@/plugins/manager'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

const activeMenu = computed(() => {
  // 如果路由 meta 中指定了 activeMenu，使用指定的菜单路径
  if (route.meta?.activeMenu) {
    return route.meta.activeMenu as string
  }
  return route.path
})

// 是否隐藏侧边栏
const hideSidebar = computed(() => {
  return route.meta?.hideSidebar === true || false
})

// 头像URL - 添加时间戳破坏缓存
const avatarUrl = computed(() => {
  const avatar = userStore.userInfo?.avatar || ''
  if (!avatar) return ''

  // 如果是base64图片，直接返回
  if (avatar.startsWith('data:')) return avatar

  // 添加时间戳参数破坏浏览器缓存（使用 store 中的时间戳）
  const separator = avatar.includes('?') ? '&' : '?'
  return `${avatar}${separator}t=${userStore.avatarTimestamp}`
})

const currentRoute = computed(() => route)

// 获取用户角色显示名称
const userRoleDisplay = computed(() => {
  const roles = userStore.userInfo?.roles || []
  if (roles.length === 0) return '普通用户'

  // 如果有多个角色，显示第一个角色的名称
  // 优先显示 admin 角色
  const adminRole = roles.find((r: any) => r.code === 'admin')
  if (adminRole) {
    return adminRole.name || '管理员'
  }

  return roles[0]?.name || '普通用户'
})

const menuList = ref<any[]>([])

// 图标映射
const iconMap: Record<string, any> = {
  'HomeFilled': HomeFilled,
  'User': User,
  'UserFilled': UserFilled,
  'OfficeBuilding': OfficeBuilding,
  'Menu': Menu,
  'Platform': Platform,
  'Setting': Setting,
  'Document': Document,
  'Tools': Tools,
  'Monitor': Monitor,
  'FolderOpened': FolderOpened,
  'Connection': Connection,
  'Files': Files,
  'Lock': Lock,
  'View': View,
  'Odometer': Odometer,
  'Tickets': Tickets,
  'List': List,
  'Grid': Grid,
  'Cloudy': Cloudy,
  'Grape': Grape,
  'House': House
}

// 获取图标组件
const getIcon = (iconName: string) => {
  return iconMap[iconName] || Menu
}

// 从插件管理器构建菜单（只包含已启用的插件，并根据权限过滤）
const buildPluginMenus = async (authorizedPaths: Set<string>) => {
  const pluginMenus: any[] = []
  const allPlugins = pluginManager.getAll() // 获取所有注册的插件

  // 从后端API获取插件启用状态
  let enabledPluginNames: Set<string> = new Set()
  try {
    const { listPlugins } = await import('@/api/plugin')
    const backendPlugins = await listPlugins()
    enabledPluginNames = new Set(
      backendPlugins
        .filter((p: any) => p.enabled)
        .map((p: any) => p.name)
    )
    console.log('[Layout] 后端已启用的插件:', Array.from(enabledPluginNames))
  } catch (error) {
    console.error('[Layout] 获取插件启用状态失败，默认显示所有插件菜单:', error)
    // 如果获取失败，显示所有已安装的插件菜单
    const installedPlugins = pluginManager.getInstalled()
    enabledPluginNames = new Set(installedPlugins.map(p => p.name))
  }

  // 从 localStorage 加载自定义排序
  const PLUGIN_MENU_SORT_KEY = 'opshub_plugin_menu_sort'
  const customSort: Map<string, number> = (() => {
    try {
      const stored = localStorage.getItem(PLUGIN_MENU_SORT_KEY)
      if (stored) {
        const sortMap = JSON.parse(stored)
        console.log('[Layout] 加载的自定义排序:', sortMap)
        return new Map(Object.entries(sortMap))
      }
    } catch (error) {
      console.error('[Layout] 加载插件菜单排序失败:', error)
    }
    return new Map()
  })()

  console.log('=== 开始构建插件菜单 ===')
  console.log('所有插件数量:', allPlugins.length)
  console.log('已启用的插件数量:', enabledPluginNames.size)
  console.log('授权的路径数量:', authorizedPaths.size)

  allPlugins.forEach(plugin => {
    // 只处理已启用的插件
    if (!enabledPluginNames.has(plugin.name)) {
      console.log(`跳过未启用的插件: ${plugin.name}`)
      return
    }

    console.log(`处理插件: ${plugin.name}`)
    if (plugin.getMenus) {
      const menus = plugin.getMenus()
      console.log(`  - 菜单数量: ${menus.length}`)

      menus.forEach(menu => {
        // 权限过滤：只显示用户有权限的菜单
        // authorizedPaths为空表示是超级管理员，显示所有菜单
        if (authorizedPaths.size > 0 && !authorizedPaths.has(menu.path)) {
          console.log(`  - 跳过无权限的菜单: ${menu.name} (${menu.path})`)
          return
        }

        // 优先使用自定义排序，如果没有则使用默认排序
        const sort = customSort.get(menu.path) ?? menu.sort

        pluginMenus.push({
          ID: menu.path,
          name: menu.name,
          path: menu.path,
          icon: menu.icon,
          sort: sort, // 使用自定义排序或默认排序
          hidden: menu.hidden,
          parentPath: menu.parentPath
          // 不设置 children 属性，让 buildMenuTree 根据实际子菜单动态设置
        })

        console.log(`  - 菜单: ${menu.name}, sort: ${sort}`)
      })
    } else {
      console.log(`  - 插件没有提供 getMenus 方法`)
    }
  })

  console.log('插件菜单构建完成，总数:', pluginMenus.length)
  return pluginMenus
}

// 构建菜单树
const buildMenuTree = (menus: any[]) => {
  // 只过滤掉不可见的菜单，禁用的菜单仍然显示但标记为禁用状态
  const filteredMenus = menus.filter(menu => {
    const isVisible = menu.visible === undefined || menu.visible === 1

    if (!isVisible) {
      console.log(`[Layout buildMenuTree] 过滤掉不可见的菜单: ${menu.name}`)
    }

    return isVisible
  })

  console.log('[Layout buildMenuTree] 过滤后菜单数量:', filteredMenus.length, '原始数量:', menus.length)

  // 创建一个 Map 来快速查找菜单
  const menuMap = new Map()

  console.log('[Layout buildMenuTree] 开始构建菜单树,菜单数量:', filteredMenus.length)

  filteredMenus.forEach(menu => {
    // 统一使用 ID 或 path 作为唯一标识
    const menuId = menu.ID || menu.id || menu.path
    if (!menuId) {
      console.warn('[Layout buildMenuTree] 菜单缺少ID:', menu)
      return
    }

    // 克隆菜单对象,避免修改原始数据，并移除原有的children
    const { children, ...menuWithoutChildren } = menu
    // 不设置 children 属性，只在需要时动态添加
    menuMap.set(menuId, menuWithoutChildren)

    console.log(`[Layout buildMenuTree] 添加菜单到Map: ${menu.name} (ID: ${menuId})`)
  })

  const tree: any[] = []

  // 构建树结构
  filteredMenus.forEach(menu => {
    const menuId = menu.ID || menu.id || menu.path
    const menuItem = menuMap.get(menuId)

    if (!menuItem) {
      console.warn('[Layout buildMenuTree] 找不到菜单:', menu)
      return
    }

    // 判断父菜单ID - 支持系统菜单(parentId)和插件菜单(parentPath)
    let parentId = null

    if (menu.parentPath !== undefined) {
      // 插件菜单,使用 parentPath
      parentId = menu.parentPath || null
    } else if (menu.parentId !== undefined) {
      // 系统菜单,使用 parentId
      // 注意:系统菜单的 parentId 可能是 0 或数字
      parentId = menu.parentId === 0 ? null : menu.parentId
    }

    console.log(`[Layout buildMenuTree] 处理菜单 ${menu.name}:`, {
      menuId,
      parentId,
      hasParent: parentId && menuMap.has(parentId)
    })

    if (parentId && menuMap.has(parentId)) {
      // 有父菜单，添加到父菜单的 children
      const parent = menuMap.get(parentId)
      // 确保 children 数组存在
      if (!parent.children) {
        parent.children = []
      }
      parent.children.push(menuItem)
      console.log(`[Layout buildMenuTree] 将 ${menu.name} 添加到父菜单 ${parent.name}`)
    } else if (parentId) {
      // parentId 存在但找不到父菜单,作为顶级菜单
      console.warn(`[Layout buildMenuTree] 找不到 ${menu.name} 的父菜单 (parentId: ${parentId}), 作为顶级菜单`)
      tree.push(menuItem)
    } else {
      // 没有父菜单，添加到根节点
      tree.push(menuItem)
      console.log(`[Layout buildMenuTree] 将 ${menu.name} 作为顶级菜单`)
    }
  })

  // 对每个层级的菜单按 sort 排序
  const sortMenus = (menus: any[]) => {
    menus.sort((a, b) => (a.sort || 0) - (b.sort || 0))
    menus.forEach(menu => {
      if (menu.children && menu.children.length > 0) {
        sortMenus(menu.children)
      }
    })
  }

  sortMenus(tree)

  // 清理空的children数组 - 关键修复！
  const cleanEmptyChildren = (nodes: any[]) => {
    for (let i = 0; i < nodes.length; i++) {
      const node = nodes[i]
      // 详细日志
      console.log(`[Layout cleanEmptyChildren] 处理 ${node.name}, children:`, node.children)

      // 检查 children 是否存在且为空数组
      if (Array.isArray(node.children) && node.children.length === 0) {
        console.log(`[Layout cleanEmptyChildren] 删除 ${node.name} 的空 children 数组`)
        delete node.children
        // 关键修复：明确设置 hasChildren 为 false
        node.hasChildren = false
      } else if (Array.isArray(node.children) && node.children.length > 0) {
        console.log(`[Layout cleanEmptyChildren] ${node.name} 有 ${node.children.length} 个子菜单，递归处理`)
        // 关键修复：明确设置 hasChildren 为 true
        node.hasChildren = true
        cleanEmptyChildren(node.children)
      } else if (node.children) {
        // children 存在但不是数组，删除它
        console.log(`[Layout cleanEmptyChildren] ${node.name} 的 children 不是数组，删除它:`, typeof node.children)
        delete node.children
        node.hasChildren = false
      } else {
        console.log(`[Layout cleanEmptyChildren] ${node.name} 没有 children 属性`)
        // 明确设置 hasChildren 为 false
        node.hasChildren = false
      }
    }
  }

  console.log('[Layout buildMenuTree] 清理前，顶级菜单:', tree.map(m => ({ name: m.name, hasChildren: !!m.children, childrenCount: m.children?.length || 0, childrenType: typeof m.children })))
  cleanEmptyChildren(tree)
  console.log('[Layout buildMenuTree] 清理后，顶级菜单:', tree.map(m => ({ name: m.name, hasChildren: !!m.children, childrenCount: m.children?.length || 0, hasChildrenAttr: m.hasChildren })))

  console.log('[Layout buildMenuTree] 菜单树构建完成,顶级菜单数:', tree.length)
  console.log('[Layout buildMenuTree] 最终菜单树:', tree)

  return tree
}

// 加载菜单
const loadMenu = async () => {
  try {
    // 清空现有菜单,避免重复
    menuList.value = []

    // 1. 获取系统菜单（后端已根据用户权限过滤）
    const systemMenus = await getUserMenu() || []
    console.log('[Layout] 系统菜单(后端过滤):', systemMenus)

    // 2. 从系统菜单中提取所有授权的路径（用于插件菜单权限过滤）
    const pluginPathPrefixes = ['/kubernetes', '/monitor', '/task']
    const extractPaths = (menus: any[]): Set<string> => {
      const paths = new Set<string>()
      const traverse = (items: any[]) => {
        items.forEach(item => {
          if (item.path) {
            paths.add(item.path)
          }
          if (item.children && item.children.length > 0) {
            traverse(item.children)
          }
        })
      }
      traverse(menus)
      return paths
    }

    const allAuthorizedPaths = extractPaths(systemMenus)
    console.log('[Layout] 所有授权的路径(含插件):', Array.from(allAuthorizedPaths))

    // 3. 获取插件菜单（根据授权路径过滤）
    const pluginMenus = await buildPluginMenus(allAuthorizedPaths)
    console.log('[Layout] 插件菜单(权限过滤后):', pluginMenus)

    // 4. 展平系统菜单树并过滤掉插件路径的菜单
    // （这些菜单仅用于授权，实际显示由插件管理器提供）
    const flattenMenus = (menus: any[], result: any[] = []) => {
      menus.forEach(menu => {
        // 跳过插件路径的菜单
        if (menu.path && pluginPathPrefixes.some(prefix => menu.path.startsWith(prefix))) {
          console.log(`[Layout] 跳过数据库中的插件菜单: ${menu.name} (${menu.path})`)
          // 仍然需要处理子菜单（如果有的话）
          if (menu.children && menu.children.length > 0) {
            flattenMenus(menu.children, result)
          }
          return
        }

        // 移除children属性，避免旧的children数据干扰
        const { children, ...menuWithoutChildren } = menu
        result.push(menuWithoutChildren)
        if (children && children.length > 0) {
          flattenMenus(children, result)
        }
      })
      return result
    }

    const flatSystemMenus = flattenMenus(systemMenus)
    console.log('[Layout] 展平后的系统菜单数(已过滤插件):', flatSystemMenus.length)

    // 5. 合并所有菜单
    const allMenus = [...flatSystemMenus, ...pluginMenus]
    console.log('[Layout] 合并后的所有菜单数:', allMenus.length)

    // 6. 构建菜单树
    menuList.value = buildMenuTree(allMenus)
    console.log('[Layout] 最终菜单树:', menuList.value)
  } catch (error) {
    console.error('[Layout] 加载菜单失败:', error)
    ElMessage.error('加载菜单失败')
  }
}

const handleUserCommand = (command: string) => {
  if (command === 'logout') {
    handleLogout()
  } else if (command === 'profile') {
    router.push('/profile')
  }
}

const handleLogout = () => {
  userStore.logout()
  router.push('/login')
}

onMounted(async () => {
  // 如果用户信息为空，先获取用户信息
  if (!userStore.userInfo) {
    try {
      await userStore.getProfile()
    } catch (error) {
      console.error('获取用户信息失败:', error)
    }
  }

  // 等待一小段时间确保插件完全加载
  await new Promise(resolve => setTimeout(resolve, 100))
  loadMenu()

  // 监听插件变化，自动刷新菜单
  const handlePluginChange = () => {
    console.log('检测到插件变化，重新加载菜单')
    loadMenu()
  }

  // 移除旧的监听器(如果存在)避免重复
  window.removeEventListener('plugins-changed', handlePluginChange)
  // 添加新的监听器
  window.addEventListener('plugins-changed', handlePluginChange)

  // 组件卸载时清理监听器
  onUnmounted(() => {
    window.removeEventListener('plugins-changed', handlePluginChange)
  })
})
</script>

<style scoped>
.layout-container {
  height: 100vh;
}

.el-aside {
  background-color: #000000 !important;
  color: #fff;
  box-shadow: 2px 0 6px rgba(0, 0, 0, 0.1);
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
}

.logo {
  height: 64px;
  line-height: 64px;
  text-align: center;
  background: #000000;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  margin: 0;
  flex-shrink: 0;
  width: 100%;
}

.logo h3 {
  margin: 0;
  color: #fff;
  font-weight: 700;
  letter-spacing: 2px;
  font-size: 22px;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
}

/* 用户信息区域 */
.user-section {
  padding: 0;
  flex-shrink: 0;
  width: 100%;
  min-width: 260px;
  max-width: 260px;
}

.user-info-wrapper {
  padding: 16px 20px;
  display: flex;
  align-items: center;
  gap: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  background: #000000;
  cursor: pointer;
  transition: all 0.3s ease;
  width: 100%;
  min-width: 260px;
  max-width: 260px;
  box-sizing: border-box;
}

.user-info-wrapper:hover {
  background-color: rgba(255, 175, 53, 0.1);
}

/* 确保 dropdown 也填满宽度 */
.user-section :deep(.el-dropdown) {
  width: 100%;
}

.user-avatar :deep(.el-avatar) {
  background-color: #FFAF35;
  border: 2px solid rgba(255, 255, 255, 0.2);
}

.user-avatar :deep(.el-icon) {
  font-size: 20px;
  color: #fff;
}

.user-details {
  flex: 1;
  min-width: 0;
}

.user-name {
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  margin-bottom: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-role {
  color: hsla(0, 0%, 100%, 0.45);
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.dropdown-icon {
  color: hsla(0, 0%, 100%, 0.45);
  font-size: 14px;
  transition: transform 0.3s;
}

:deep(.el-dropdown:hover .dropdown-icon) {
  transform: rotate(180deg);
}

/* 下拉菜单样式 */
:deep(.el-dropdown-menu) {
  background-color: #ffffff !important;
  border: 1px solid #e4e7ed;
  padding: 4px 0;
}

:deep(.el-dropdown-menu__item) {
  color: #606266 !important;
  line-height: 40px;
  padding: 0 16px;
}

:deep(.el-dropdown-menu__item:hover) {
  background-color: #ecf5ff !important;
  color: #409eff !important;
}

:deep(.el-dropdown-menu__item.is-divided) {
  border-top: 1px solid #e4e7ed;
  margin-top: 4px;
  padding-top: 8px;
}

:deep(.el-dropdown-menu__item .el-icon) {
  color: #606266 !important;
  margin-right: 8px;
  font-size: 16px;
}

:deep(.el-dropdown-menu__item:hover .el-icon) {
  color: #409eff !important;
}

.el-menu {
  border-right: none !important;
  background-color: #000000 !important;
  flex: 1 1 auto;
  overflow-y: auto; /* 允许垂直滚动 */
  overflow-x: hidden; /* 隐藏水平滚动 */
}

/* 自定义滚动条样式 */
.el-menu::-webkit-scrollbar {
  width: 6px;
}

.el-menu::-webkit-scrollbar-track {
  background: transparent;
}

.el-menu::-webkit-scrollbar-thumb {
  background-color: rgba(255, 255, 255, 0.2);
  border-radius: 3px;
  transition: background-color 0.3s;
}

.el-menu::-webkit-scrollbar-thumb:hover {
  background-color: rgba(255, 255, 255, 0.3);
}

/* 覆盖 Element Plus 菜单样式 */
:deep(.el-menu) {
  background-color: #000000 !important;
}

:deep(.el-menu-item) {
  color: #fff !important;
  background-color: transparent !important;
  font-size: 16px !important;
  padding-left: 20px !important; /* 从24px改为20px,往左移 */
  height: 48px !important;
  line-height: 48px !important;
  transition: background-color 0.3s ease, color 0.3s ease;
  margin: 4px 12px; /* 上下4px间距,左右12px */
  border-radius: 8px; /* 圆角效果 */
}

:deep(.el-menu-item:hover) {
  background-color: transparent !important;
  color: #FFAF35 !important;
}

:deep(.el-menu-item.is-active) {
  background-color: #FFAF35 !important;
  color: #000000 !important;
  border-radius: 8px; /* 圆角效果 */
}

:deep(.el-menu-item .el-icon) {
  color: inherit;
  font-size: 18px !important;
  margin-right: 12px !important;
  transition: color 0.3s ease;
}

/* 子菜单标题样式 */
:deep(.el-sub-menu__title) {
  color: #fff !important;
  background-color: transparent !important;
  font-size: 16px !important;
  padding-left: 20px !important; /* 从24px改为20px,往左移 */
  height: 48px !important;
  line-height: 48px !important;
  transition: background-color 0.3s ease, color 0.3s ease;
  margin: 4px 12px; /* 上下4px间距,左右12px */
  border-radius: 8px; /* 圆角效果 */
}

:deep(.el-sub-menu__title:hover) {
  background-color: transparent !important;
  color: #FFAF35 !important;
}

:deep(.el-sub-menu.is-active > .el-sub-menu__title) {
  background-color: #FFAF35 !important;
  color: #000000 !important;
  border-radius: 8px; /* 圆角效果 */
}

:deep(.el-sub-menu__title .el-icon) {
  color: inherit;
  font-size: 18px !important;
  margin-right: 12px !important;
  transition: color 0.3s ease;
}

/* 子菜单项样式 - 内联菜单的子项 */
:deep(.el-menu--inline .el-menu-item) {
  padding-left: 48px !important; /* 从56px改为48px,与父菜单的间距保持一致 */
  margin: 4px 20px; /* 上下4px间距,左右20px(更大,使背景更小) */
  border-radius: 6px; /* 子菜单圆角稍小 */
}

/* 禁用子菜单展开动画，防止抖动 */
:deep(.el-menu--collapse-transition) {
  transition: none !important;
}

:deep(.el-menu--inline) {
  transition: none !important;
}

:deep(.el-sub-menu__title) {
  transition: background-color 0.3s ease, color 0.3s ease !important;
}

/* 子菜单展开时不使用动画 */
:deep(.el-menu--vertical .el-sub-menu .el-menu) {
  transition: none !important;
}

/* 彻底禁用 el-menu 的折叠转换动画 */
:deep(.el-menu.el-menu--vertical) {
  --el-transition-duration: 0s;
}

:deep(.el-menu--vertical .el-menu--popup) {
  animation: none !important;
}

/* 禁用子菜单折叠器的过渡动画 */
:deep(.el-menu--vertical .el-sub-menu .el-sub-menu__title .el-sub-menu__icon-arrow) {
  transition: none !important;
}

:deep(.el-menu--vertical > .el-sub-menu.is-opened > .el-sub-menu__title .el-sub-menu__icon-arrow) {
  transform: none !important;
}

/* 子菜单项选中状态 */
:deep(.el-menu--inline .el-menu-item.is-active) {
  background-color: #FFAF35 !important;
  color: #000000 !important;
  border-radius: 6px; /* 圆角效果 */
}

/* 禁用菜单样式 */
:deep(.menu-disabled) {
  opacity: 0.4 !important;
  cursor: not-allowed !important;
}

:deep(.menu-disabled.el-menu-item) {
  pointer-events: none !important;
}

:deep(.menu-disabled.el-sub-menu__title) {
  pointer-events: none !important;
}

:deep(.menu-disabled .el-icon) {
  opacity: 0.5 !important;
}

:deep(.menu-disabled span) {
  opacity: 0.7 !important;
}

.el-header {
  background-color: #fff;
  border-bottom: 1px solid #e6e6e6;
  display: flex;
  align-items: center;
  padding: 0 20px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
}

.header-content {
  width: 100%;
  display: flex;
  align-items: center;
}

.el-main {
  background-color: #f0f2f5;
  padding: 20px;
}
</style>
