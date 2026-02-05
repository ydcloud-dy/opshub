<template>
  <el-container class="layout-container">
    <el-aside width="260px" v-if="!hideSidebar">
      <div class="logo">
        <!-- 始终显示文字Logo（系统名称） -->
        <span class="logo-text">
          <span class="logo-ops">{{ systemNameFirst }}</span><span class="logo-hub">{{ systemNameSecond }}</span>
        </span>
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
          <div class="header-logo">
            <img :src="headerImage" alt="Header" class="header-image" />
          </div>
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
        <!-- 无权限时显示无权限页面 -->
        <NoPermission v-if="hasNoPermission" />
        <!-- 有权限时显示正常内容 -->
        <router-view v-else />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted, shallowRef } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useSystemStore } from '@/stores/system'
import { ElMessage } from 'element-plus'
import NoPermission from '@/views/NoPermission.vue'
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

// Header 图片路径（来自 public 文件夹）
const headerImage = '/header.png'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const systemStore = useSystemStore()

// 系统名称分割（用于文字Logo显示）
const systemNameFirst = computed(() => {
  const name = systemStore.systemName || 'OpsHub'
  // 尝试智能分割：如果名字有明显的大写字母分隔
  const match = name.match(/^([A-Z][a-z]*)(.*)$/)
  if (match && match[2]) {
    return match[1]
  }
  // 否则取前半部分
  const mid = Math.ceil(name.length / 2)
  return name.substring(0, mid)
})

const systemNameSecond = computed(() => {
  const name = systemStore.systemName || 'OpsHub'
  const match = name.match(/^([A-Z][a-z]*)(.*)$/)
  if (match && match[2]) {
    return match[2]
  }
  const mid = Math.ceil(name.length / 2)
  return name.substring(mid)
})

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
const hasNoPermission = ref(false) // 用户是否没有任何权限

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

  // 检查当前用户是否是超级管理员
  const roles = userStore.userInfo?.roles || []
  const isSuperAdmin = roles.some((r: any) => r.code === 'admin')

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
  } catch (error) {
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
        return new Map(Object.entries(sortMap))
      }
    } catch (error) {
    }
    return new Map()
  })()


  allPlugins.forEach(plugin => {
    // 只处理已启用的插件
    if (!enabledPluginNames.has(plugin.name)) {
      return
    }

    if (plugin.getMenus) {
      const menus = plugin.getMenus()

      menus.forEach(menu => {
        // 权限过滤：
        // 1. 超级管理员显示所有菜单
        // 2. 普通用户只显示有权限的菜单
        // 3. 用户没有任何权限且不是超级管理员，不显示任何菜单
        if (!isSuperAdmin && !authorizedPaths.has(menu.path)) {
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

      })
    } else {
    }
  })

  return pluginMenus
}

// 构建菜单树
const buildMenuTree = (menus: any[]) => {
  // 只过滤掉不可见的菜单，禁用的菜单仍然显示但标记为禁用状态
  const filteredMenus = menus.filter(menu => {
    const isVisible = menu.visible === undefined || menu.visible === 1

    if (!isVisible) {
    }

    return isVisible
  })

  // 通过code/path/name去重，去掉完全相同的菜单
  const uniqueMenus: any[] = []
  const seenSignatures = new Set<string>()

  for (const menu of filteredMenus) {
    // 生成菜单的唯一标识
    // 对于顶级菜单（parentId=0或parentPath为空），都使用'root'作为parentKey
    let parentKey = 'root'
    if (menu.parentId !== undefined && menu.parentId !== 0) {
      parentKey = `parent_${menu.parentId}`
    } else if (menu.parentPath !== undefined && menu.parentPath !== '' && menu.parentPath !== '/') {
      parentKey = menu.parentPath
    }

    const signature = `${menu.name}_${parentKey}`

    if (seenSignatures.has(signature)) {
      continue
    }
    seenSignatures.add(signature)
    uniqueMenus.push(menu)
  }

  // 创建一个 Map 来快速查找菜单
  const menuMap = new Map()
  // 为系统菜单创建一个path到menu的映射（用于插件菜单查找父菜单）
  const pathToMenuMap = new Map()

  uniqueMenus.forEach(menu => {
    // 统一使用 ID 或 path 作为唯一标识
    const menuId = menu.ID || menu.id || menu.path
    if (!menuId) {
      return
    }

    // 克隆菜单对象,避免修改原始数据，并移除原有的children
    const { children, ...menuWithoutChildren } = menu
    // 不设置 children 属性，只在需要时动态添加
    menuMap.set(menuId, menuWithoutChildren)

    // 为系统菜单添加path映射（用于插件菜单查找）
    if (menu.path && menu.path.startsWith('/')) {
      pathToMenuMap.set(menu.path, menuWithoutChildren)
    }
  })

  const tree: any[] = []

  // 构建树结构
  filteredMenus.forEach(menu => {
    const menuId = menu.ID || menu.id || menu.path
    const menuItem = menuMap.get(menuId)

    if (!menuItem) {
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

    if (parentId && menuMap.has(parentId)) {
      // 有父菜单，添加到父菜单的 children（通过数字ID查找）
      const parent = menuMap.get(parentId)
      // 确保 children 数组存在
      if (!parent.children) {
        parent.children = []
      }
      parent.children.push(menuItem)
    } else if (parentId && pathToMenuMap.has(parentId)) {
      // 通过path查找父菜单（用于插件菜单）
      const parent = pathToMenuMap.get(parentId)
      if (!parent.children) {
        parent.children = []
      }
      parent.children.push(menuItem)
    } else if (parentId) {
      // parentId 存在但找不到父菜单,作为顶级菜单
      tree.push(menuItem)
    } else {
      // 没有父菜单，添加到根节点
      tree.push(menuItem)
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

      // 检查 children 是否存在且为空数组
      if (Array.isArray(node.children) && node.children.length === 0) {
        delete node.children
        // 关键修复：明确设置 hasChildren 为 false
        node.hasChildren = false
      } else if (Array.isArray(node.children) && node.children.length > 0) {
        // 关键修复：明确设置 hasChildren 为 true
        node.hasChildren = true
        cleanEmptyChildren(node.children)
      } else if (node.children) {
        // children 存在但不是数组，删除它
        delete node.children
        node.hasChildren = false
      } else {
        // 明确设置 hasChildren 为 false
        node.hasChildren = false
      }
    }
  }

  cleanEmptyChildren(tree)


  return tree
}

// 加载菜单
const loadMenu = async () => {
  try {
    // 清空现有菜单,避免重复
    menuList.value = []

    // 1. 获取系统菜单（后端已根据用户权限过滤）
    const systemMenus = await getUserMenu() || []

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

    // 3. 获取插件菜单（根据授权路径过滤）
    const pluginMenus = await buildPluginMenus(allAuthorizedPaths)

    // 4. 展平系统菜单树，并过滤掉那些已经由插件提供的菜单
    const pluginProvidedMenuCodes = new Set(['kubernetes_application_diagnosis', 'kubernetes_cluster_inspection', 'monitor_domain', 'monitor_alert_channels', 'monitor_alert_receivers', 'monitor_alert_logs', 'task_templates', 'task_execute', 'task_file_distribution', 'kubernetes_clusters', 'kubernetes_nodes', 'kubernetes_namespaces', 'kubernetes_workloads', 'kubernetes_network', 'kubernetes_config', 'kubernetes_storage', 'kubernetes_access', 'kubernetes_audit'])

    const flattenMenus = (menus: any[], result: any[] = []) => {
      menus.forEach(menu => {
        // 如果这个菜单的code在插件提供的列表中，跳过它（由插件提供）
        if (menu.code && pluginProvidedMenuCodes.has(menu.code)) {
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

    // 5. 合并所有菜单
    const allMenus = [...flatSystemMenus, ...pluginMenus]

    // 6. 构建菜单树
    menuList.value = buildMenuTree(allMenus)

    // 检查用户是否有权限
    // 如果不是超级管理员且没有任何菜单，则显示无权限页面
    const roles = userStore.userInfo?.roles || []
    const isSuperAdmin = roles.some((r: any) => r.code === 'admin')
    if (!isSuperAdmin && menuList.value.length === 0) {
      hasNoPermission.value = true
    } else {
      hasNoPermission.value = false
    }
  } catch (error) {
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
  // 加载系统配置
  if (!systemStore.loaded) {
    await systemStore.loadFullConfig()
  }

  // 如果用户信息为空，先获取用户信息
  if (!userStore.userInfo) {
    try {
      await userStore.getProfile()
    } catch (error) {
    }
  }

  // 等待一小段时间确保插件完全加载
  await new Promise(resolve => setTimeout(resolve, 100))
  loadMenu()

  // 监听插件变化，自动刷新菜单
  const handlePluginChange = () => {
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
  height: 80px;
  text-align: center;
  background: #000000;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  margin: 0;
  flex-shrink: 0;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.logo-image {
  max-height: 60px;
  max-width: 200px;
  object-fit: contain;
}

.logo-text {
  display: inline-flex;
  align-items: center;
  font-size: 32px;
  font-weight: 600;
  font-style: italic;
  line-height: 1;
}

.logo-ops {
  color: #ffffff;
}

.logo-hub {
  background-color: #FFAF35;
  color: #000000;
  padding: 4px 10px;
  border-radius: 6px;
  margin-left: 2px;
  line-height: 1;
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
  gap: 20px;
}

.header-logo {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 64px;
}

.header-image {
  max-height: 50px;
  max-width: 400px;
  width: auto;
  height: auto;
  object-fit: contain;
}

.breadcrumb {
  flex: 1;
  display: flex;
  align-items: center;
}

.el-main {
  background-color: #f0f2f5;
  padding: 20px;
}

</style>
