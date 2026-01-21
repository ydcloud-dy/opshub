import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { pluginManager } from '@/plugins/manager'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/Login.vue'),
      meta: { title: '登录' }
    },
    {
      path: '/',
      name: 'Layout',
      component: () => import('@/views/Layout.vue'),
      redirect: '/dashboard',
      children: [
        {
          path: 'dashboard',
          name: 'Dashboard',
          component: () => import('@/views/Dashboard.vue'),
          meta: { title: '首页' }
        },
        {
          path: 'users',
          name: 'Users',
          component: () => import('@/views/system/Users.vue'),
          meta: { title: '用户管理' }
        },
        {
          path: 'roles',
          name: 'Roles',
          component: () => import('@/views/system/Roles.vue'),
          meta: { title: '角色管理' }
        },
        {
          path: 'menus',
          name: 'Menus',
          component: () => import('@/views/system/Menus.vue'),
          meta: { title: '菜单管理' }
        },
        {
          path: 'dept-info',
          name: 'DeptInfo',
          component: () => import('@/views/system/DeptInfo.vue'),
          meta: { title: '部门信息' }
        },
        {
          path: 'position-info',
          name: 'PositionInfo',
          component: () => import('@/views/system/PositionInfo.vue'),
          meta: { title: '岗位信息' }
        },
        {
          path: 'system-config',
          name: 'SystemConfig',
          component: () => import('@/views/system/SystemConfig.vue'),
          meta: { title: '系统配置' }
        },
        {
          path: 'audit/operation-logs',
          name: 'OperationLogs',
          component: () => import('@/views/audit/OperationLogs.vue'),
          meta: { title: '操作日志' }
        },
        {
          path: 'audit/login-logs',
          name: 'LoginLogs',
          component: () => import('@/views/audit/LoginLogs.vue'),
          meta: { title: '登录日志' }
        },
        {
          path: 'audit/data-logs',
          name: 'DataLogs',
          component: () => import('@/views/audit/DataLogs.vue'),
          meta: { title: '数据日志' }
        },
        {
          path: 'asset/hosts',
          name: 'AssetHosts',
          component: () => import('@/views/asset/Hosts.vue'),
          meta: { title: '主机管理' }
        },
        {
          path: 'asset/credentials',
          name: 'AssetCredentials',
          component: () => import('@/views/asset/Credentials.vue'),
          meta: { title: '凭据管理' }
        },
        {
          path: 'asset/cloud-accounts',
          name: 'AssetCloudAccounts',
          component: () => import('@/views/asset/CloudAccounts.vue'),
          meta: { title: '云账号管理' }
        },
        {
          path: 'asset/terminal-audit',
          name: 'AssetTerminalAudit',
          component: () => import('@/views/asset/TerminalAudit.vue'),
          meta: { title: '终端审计' }
        },
        {
          path: 'asset/groups',
          name: 'AssetGroups',
          component: () => import('@/views/asset/Groups.vue'),
          meta: { title: '业务分组' }
        },
        {
          path: 'asset/permissions',
          name: 'AssetPermissions',
          component: () => import('@/views/asset/AssetPermission.vue'),
          meta: { title: '权限配置' }
        },
        {
          path: 'profile',
          name: 'Profile',
          component: () => import('@/views/Profile.vue'),
          meta: { title: '个人信息' }
        },
        {
          path: 'terminal',
          name: 'Terminal',
          component: () => import('@/views/asset/Terminal.vue'),
          meta: { title: 'Web终端', hideSidebar: true }
        },
        {
          path: 'plugin/list',
          name: 'PluginList',
          component: () => import('@/views/plugin/PluginList.vue'),
          meta: { title: '插件列表' }
        },
        {
          path: 'plugin/install',
          name: 'PluginInstall',
          component: () => import('@/views/plugin/PluginInstall.vue'),
          meta: { title: '插件安装' }
        }
      ]
    }
  ]
})

// 注册插件路由
export function registerPluginRoutes() {
  // 修改为使用 getInstalled() 以保持与菜单构建的一致性
  const plugins = pluginManager.getInstalled()

  console.log('[Router] 开始注册插件路由')
  console.log('[Router] 已安装的插件数量:', plugins.length)

  for (const plugin of plugins) {
    if (plugin.getRoutes) {
      const routes = plugin.getRoutes()

      // 添加插件的子路由到 Layout
      routes.forEach(route => {
        router.addRoute('Layout', route)
      })

      console.log(`[Router] 插件 ${plugin.name} 路由注册成功, 路由数量:`, routes.length)
    } else {
      console.log(`[Router] 插件 ${plugin.name} 没有提供路由配置`)
    }
  }

  console.log('[Router] 插件路由注册完成')
}

// 路由守卫
router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  console.log('路由守卫 - 目标路径:', to.path)
  console.log('路由守卫 - 当前路径:', from.path)
  console.log('路由守卫 - Token:', token)

  // 如果访问登录页，且已登录，则跳转到首页
  if (to.path === '/login') {
    if (token) {
      console.log('已登录访问登录页，跳转到首页')
      next('/')
    } else {
      console.log('未登录访问登录页，继续')
      next()
    }
  } else {
    // 访问其他页面，需要检查登录状态
    if (!token) {
      console.log('未登录，跳转到登录页')
      next('/login')
    } else {
      console.log('已登录，继续访问')
      next()
    }
  }
})

export default router

