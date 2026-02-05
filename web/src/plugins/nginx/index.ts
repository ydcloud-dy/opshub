import type { Plugin, PluginMenuConfig, PluginRouteConfig } from '../types'
import { pluginManager } from '../manager'

/**
 * Nginx统计插件
 * 提供Nginx访问日志分析和统计功能
 */
class NginxPlugin implements Plugin {
  name = 'nginx'
  prettyName = 'Nginx统计'
  description = 'Nginx统计插件，支持主机Nginx和K8s Ingress-Nginx的访问日志分析和统计'
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
    const parentPath = '/nginx'

    return [
      {
        name: 'Nginx统计',
        path: parentPath,
        icon: 'DataLine',
        sort: 50,
        hidden: false,
        parentPath: '',
      },
      {
        name: '概况',
        path: '/nginx/overview',
        icon: 'PieChart',
        sort: 1,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: 'Top分析',
        path: '/nginx/top-analysis',
        icon: 'Histogram',
        sort: 2,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '数据日报',
        path: '/nginx/daily-report',
        icon: 'Calendar',
        sort: 3,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '访问明细',
        path: '/nginx/access-logs',
        icon: 'List',
        sort: 4,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '数据源配置',
        path: '/nginx/config',
        icon: 'Setting',
        sort: 5,
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
        path: '/nginx',
        name: 'Nginx',
        component: () => import('./components/Overview.vue'),
        redirect: '/nginx/overview',
        meta: { title: 'Nginx统计' },
      },
      {
        path: '/nginx/overview',
        name: 'NginxOverview',
        component: () => import('./components/Overview.vue'),
        meta: { title: '概况' },
      },
      {
        path: '/nginx/top-analysis',
        name: 'NginxTopAnalysis',
        component: () => import('./components/TopAnalysis.vue'),
        meta: { title: 'Top分析' },
      },
      {
        path: '/nginx/daily-report',
        name: 'NginxDailyReport',
        component: () => import('./components/DailyReport.vue'),
        meta: { title: '数据日报' },
      },
      {
        path: '/nginx/access-logs',
        name: 'NginxAccessLogs',
        component: () => import('./components/AccessLogs.vue'),
        meta: { title: '访问明细' },
      },
      {
        path: '/nginx/config',
        name: 'NginxConfig',
        component: () => import('./components/Config.vue'),
        meta: { title: '数据源配置' },
      },
    ]
  }
}

// 创建并注册插件实例
const plugin = new NginxPlugin()
pluginManager.register(plugin)

export default plugin
