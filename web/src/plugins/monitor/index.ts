import type { Plugin, PluginMenuConfig, PluginRouteConfig } from '../types'
import { pluginManager } from '../manager'

/**
 * 监控中心插件
 * 提供域名监控、告警管理等功能
 */
class MonitorPlugin implements Plugin {
  name = 'monitor'
  prettyName = '监控中心'
  description = '监控中心插件，提供域名监控、告警管理等功能'
  version = '1.0.0'
  author = 'J'

  /**
   * 安装插件
   */
  async install() {
    // 初始化操作
  }

  /**
   * 卸载插件
   */
  async uninstall() {
    // 清理资源
  }

  /**
   * 获取插件菜单配置
   */
  getMenus(): PluginMenuConfig[] {
    const parentPath = '/monitor'

    return [
      {
        name: '监控中心',
        path: parentPath,
        icon: 'Monitor',
        sort: 20,
        hidden: false,
        parentPath: '',
      },
      {
        name: '监控面板',
        path: '/monitor/dashboard',
        icon: 'DataAnalysis',
        sort: 1,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '故障中心',
        path: '/monitor/fault-centers',
        icon: 'FirstAidKit',
        sort: 2,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '告警规则',
        path: '/monitor/rules',
        icon: 'Warning',
        sort: 3,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '数据源管理',
        path: '/monitor/datasources',
        icon: 'Connection',
        sort: 4,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '拨测任务',
        path: '/monitor/probe-tasks',
        icon: 'Odometer',
        sort: 5,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '即时拨测',
        path: '/monitor/instant-probe',
        icon: 'VideoPlay',
        sort: 6,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '通知对象',
        path: '/monitor/notice-objects',
        icon: 'MessageBox',
        sort: 7,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '通知模板',
        path: '/monitor/notice-templates',
        icon: 'DocumentCopy',
        sort: 8,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '值班表',
        path: '/monitor/duty-tables',
        icon: 'Calendar',
        sort: 9,
        hidden: false,
        parentPath: parentPath,
      },
    ]
  }

  /**
   * 获取插件路由配置
   */
  getRoutes(): PluginRouteConfig[] {
    return [
      {
        path: '/monitor',
        name: 'Monitor',
        component: () => import('./components/MonitorDashboard.vue'),
        redirect: '/monitor/dashboard',
        meta: { title: '监控中心' },
      },
      {
        path: '/monitor/dashboard',
        name: 'MonitorDashboard',
        component: () => import('./components/MonitorDashboard.vue'),
        meta: { title: '监控面板' },
      },
      {
        path: '/monitor/fault-centers',
        name: 'MonitorFaultCenters',
        component: () => import('./components/FaultCenters.vue'),
        meta: { title: '故障中心' },
      },
      {
        path: '/monitor/fault-centers/:id',
        name: 'MonitorFaultCenterDetail',
        component: () => import('./components/FaultCenterDetail.vue'),
        meta: { title: '故障中心详情', hidden: true },
      },
      {
        path: '/monitor/datasources',
        name: 'MonitorDataSources',
        component: () => import('./components/DataSources.vue'),
        meta: { title: '数据源管理' },
      },
      {
        path: '/monitor/rules',
        name: 'MonitorAlertRules',
        component: () => import('./components/AlertRules.vue'),
        meta: { title: '告警规则' },
      },
      {
        path: '/monitor/probe-tasks',
        name: 'MonitorProbeTasks',
        component: () => import('./components/ProbeTasks.vue'),
        meta: { title: '拨测任务' },
      },
      {
        path: '/monitor/instant-probe',
        name: 'MonitorInstantProbe',
        component: () => import('./components/InstantProbe.vue'),
        meta: { title: '即时拨测' },
      },
      {
        path: '/monitor/notice-objects',
        name: 'MonitorNoticeObjects',
        component: () => import('./components/NoticeObjects.vue'),
        meta: { title: '通知对象' },
      },
      {
        path: '/monitor/notice-templates',
        name: 'MonitorNoticeTemplates',
        component: () => import('./components/NoticeTemplates.vue'),
        meta: { title: '通知模板' },
      },
      {
        path: '/monitor/duty-tables',
        name: 'MonitorDutyTables',
        component: () => import('./components/DutyTables.vue'),
        meta: { title: '值班表' },
      },
    ]
  }
}

// 创建并注册插件实例
const plugin = new MonitorPlugin()
pluginManager.register(plugin)

export default plugin
