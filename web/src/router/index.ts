import { createRouter, createWebHistory } from 'vue-router'
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
      path: '/mfa/verify',
      name: 'MFAVerify',
      component: () => import('@/views/auth/MFAVerify.vue'),
      meta: { title: 'MFA验证', public: true }
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
          path: 'mfa-settings',
          name: 'MFASettings',
          component: () => import('@/views/system/MFASettings.vue'),
          meta: { title: '两步验证' }
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
          path: 'asset/agents',
          name: 'AssetAgents',
          component: () => import('@/views/asset/Agents.vue'),
          meta: { title: 'Agent管理' }
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
        },
        {
          path: 'identity/portal',
          name: 'IdentityPortal',
          component: () => import('@/views/identity/Portal.vue'),
          meta: { title: '应用门户' }
        },
        {
          path: 'identity/sources',
          name: 'IdentitySources',
          component: () => import('@/views/identity/IdentitySources.vue'),
          meta: { title: '身份源管理' }
        },
        {
          path: 'identity/apps',
          name: 'IdentityApps',
          component: () => import('@/views/identity/SSOApplications.vue'),
          meta: { title: '应用管理' }
        },
        {
          path: 'identity/credentials',
          name: 'IdentityCredentials',
          component: () => import('@/views/identity/Credentials.vue'),
          meta: { title: '凭证管理' }
        },
        {
          path: 'identity/permissions',
          name: 'IdentityPermissions',
          component: () => import('@/views/identity/Permissions.vue'),
          meta: { title: '访问策略' }
        },
        {
          path: 'identity/logs',
          name: 'IdentityLogs',
          component: () => import('@/views/identity/AuthLogs.vue'),
          meta: { title: '认证日志' }
        },
        {
          path: 'aiops/assistant',
          name: 'AIOpsAssistant',
          component: () => import('@/views/aiops/Assistant.vue'),
          meta: { title: 'AI助手' }
        },
        {
          path: 'aiops/diagnosis',
          name: 'AIOpsDiagnosis',
          component: () => import('@/views/aiops/Diagnosis.vue'),
          meta: { title: '智能诊断' }
        },
        {
          path: 'aiops/logs',
          name: 'AIOpsLogAnalysis',
          component: () => import('@/views/aiops/LogAnalysis.vue'),
          meta: { title: '日志分析' }
        },
        {
          path: 'aiops/alerts',
          name: 'AIOpsAlerts',
          component: () => import('@/views/aiops/Alerts.vue'),
          meta: { title: '告警分析' }
        },
        {
          path: 'aiops/sessions',
          name: 'AIOpsSessions',
          component: () => import('@/views/aiops/Sessions.vue'),
          meta: { title: 'AI会话记录' }
        },
        {
          path: 'aiops/settings',
          name: 'AIOpsSettings',
          component: () => import('@/views/aiops/Settings.vue'),
          meta: { title: 'AI配置' }
        },
        {
          path: 'logs',
          name: 'LogCenter',
          redirect: '/logs/overview',
<<<<<<< HEAD
          meta: { title: '日志中心' }
=======
          meta: { title: '日志中心', keepAlive: true }
>>>>>>> feat: update log
        },
        {
          path: 'logs/overview',
          name: 'LogCenterOverview',
          component: () => import('@/views/logcenter/Overview.vue'),
<<<<<<< HEAD
          meta: { title: '日志总览' }
=======
          meta: { title: '日志总览', keepAlive: true }
>>>>>>> feat: update log
        },
        {
          path: 'logs/query',
          name: 'LogCenterQuery',
          component: () => import('@/views/logcenter/Query.vue'),
<<<<<<< HEAD
          meta: { title: '日志查询' }
=======
          meta: { title: '日志查询', keepAlive: true }
>>>>>>> feat: update log
        },
        {
          path: 'logs/library',
          name: 'LogCenterLibrary',
          component: () => import('@/views/logcenter/Library.vue'),
<<<<<<< HEAD
          meta: { title: '日志库' }
=======
          meta: { title: '日志库', keepAlive: true }
>>>>>>> feat: update log
        },
        {
          path: 'logs/templates',
          name: 'LogCenterTemplates',
          component: () => import('@/views/logcenter/Templates.vue'),
<<<<<<< HEAD
          meta: { title: '查询模板' }
=======
          meta: { title: '查询模板', keepAlive: true }
>>>>>>> feat: update log
        },
        {
          path: 'logs/collectors',
          name: 'LogCenterCollectors',
          component: () => import('@/views/logcenter/Collectors.vue'),
<<<<<<< HEAD
          meta: { title: '采集接入' }
=======
          meta: { title: '采集接入', keepAlive: true }
>>>>>>> feat: update log
        }
      ]
    }
  ]
})

// 注册插件路由
export function registerPluginRoutes() {
  // 修改为使用 getInstalled() 以保持与菜单构建的一致性
  const plugins = pluginManager.getInstalled()

  for (const plugin of plugins) {
    if (plugin.getRoutes) {
      const routes = plugin.getRoutes()

      // 添加插件的子路由到 Layout
      routes.forEach(route => {
        if (route.name && router.hasRoute(route.name)) {
          return
        }
        router.addRoute('Layout', route)
      })
    }
  }
}

// MFA检查缓存
let mfaCheckCache: { enforced: boolean; enabled: boolean; timestamp: number } | null = null
const MFA_CACHE_DURATION = 5 * 60 * 1000 // 5分钟缓存

// 更新MFA缓存状态（供外部调用）
export function updateMFACache(enabled: boolean, enforced: boolean = false) {
  mfaCheckCache = {
    enforced,
    enabled,
    timestamp: Date.now()
  }
  // 同步更新localStorage标记
  if (enforced && !enabled) {
    localStorage.setItem('mfa_setup_required', 'true')
  } else {
    localStorage.removeItem('mfa_setup_required')
  }
}

// 清除MFA强制标记（登出时调用）
export function clearMFAFlag() {
  mfaCheckCache = null
  localStorage.removeItem('mfa_setup_required')
}

// 路由守卫
router.beforeEach(async (to, _from, next) => {
  const token = localStorage.getItem('token')

  // 公开路由（登录页、OAuth回调等）
  if (to.path === '/login' || to.meta.public) {
    if (to.path === '/login' && token) {
      next('/')
    } else {
      next()
    }
    return
  }

  // 访问其他页面，需要检查登录状态
  if (!token) {
    next('/login')
    return
  }

  // 检查MFA强制设置
  if (to.path !== '/mfa-settings') {
    // 先做同步检查：如果localStorage中有强制MFA标记，立即拦截
    const mfaRequired = localStorage.getItem('mfa_setup_required')
    if (mfaRequired === 'true') {
      next('/mfa-settings?force=true')
      return
    }

    try {
      // 检查缓存是否有效
      const now = Date.now()
      if (mfaCheckCache && (now - mfaCheckCache.timestamp) < MFA_CACHE_DURATION) {
        // 使用缓存数据
        if (mfaCheckCache.enforced && !mfaCheckCache.enabled) {
          localStorage.setItem('mfa_setup_required', 'true')
          next('/mfa-settings?force=true')
          return
        }
      } else {
        // 缓存过期或不存在，重新获取
        const { getSecurityConfig } = await import('@/api/system')
        const { getMFAStatus } = await import('@/api/mfa')

        const [securityConfigResult, mfaStatusResult] = await Promise.all([
          getSecurityConfig(),
          getMFAStatus()
        ])
        const securityConfig = securityConfigResult as any
        const mfaStatus = mfaStatusResult as any

        // 更新缓存
        mfaCheckCache = {
          enforced: securityConfig.mfaEnforced,
          enabled: mfaStatus.isEnabled,
          timestamp: now
        }

        // 如果系统开启了强制MFA，且用户未启用MFA，则强制跳转到MFA设置页面
        if (securityConfig.mfaEnforced && !mfaStatus.isEnabled) {
          localStorage.setItem('mfa_setup_required', 'true')
          next('/mfa-settings?force=true')
          return
        } else {
          localStorage.removeItem('mfa_setup_required')
        }
      }
    } catch (error) {
      console.error('检查MFA状态失败', error)
      // 如果检查失败但之前已知需要MFA设置，仍然阻止访问
      // （这里mfaRequired已经在上面检查过了，到这里说明之前没有标记，允许通过）
    }
  }

  next()
})

export default router
