import type { Plugin, PluginMenuConfig, PluginRouteConfig } from '../types'
import { pluginManager } from '../manager'
import '@/views/logcenter/logcenter.css'

class LogCenterPlugin implements Plugin {
  name = 'logcenter'
  description = 'OpsHub 自带日志采集、ClickHouse 存储、统一查询与采集状态'
  version = '1.0.0'
  author = 'J'

  async install() {
  }

  async uninstall() {
  }

  getMenus(): PluginMenuConfig[] {
    const parentPath = '/logs'
    return [
      {
        name: '日志中心',
        path: parentPath,
        icon: 'Histogram',
        sort: 25,
        hidden: false,
        parentPath: '',
      },
      {
        name: '日志总览',
        path: '/logs/overview',
        icon: 'DataAnalysis',
        sort: 1,
        hidden: false,
        parentPath,
      },
      {
        name: '日志查询',
        path: '/logs/query',
        icon: 'Search',
        sort: 2,
        hidden: false,
        parentPath,
      },
      {
        name: '日志库',
        path: '/logs/library',
        icon: 'FolderOpened',
        sort: 3,
        hidden: false,
        parentPath,
      },
      {
        name: '查询模板',
        path: '/logs/templates',
        icon: 'CollectionTag',
        sort: 4,
        hidden: false,
        parentPath,
      },
      {
        name: '采集接入',
        path: '/logs/collectors',
        icon: 'SetUp',
        sort: 5,
        hidden: false,
        parentPath,
      },
    ]
  }

  getRoutes(): PluginRouteConfig[] {
    return [
      {
        path: '/logs',
        name: 'LogCenter',
        component: () => import('@/views/logcenter/Overview.vue'),
        meta: { title: '日志中心' },
      },
      {
        path: '/logs/overview',
        name: 'LogCenterOverview',
        component: () => import('@/views/logcenter/Overview.vue'),
        meta: { title: '日志总览' },
      },
      {
        path: '/logs/query',
        name: 'LogCenterQuery',
        component: () => import('@/views/logcenter/Query.vue'),
        meta: { title: '日志查询' },
      },
      {
        path: '/logs/library',
        name: 'LogCenterLibrary',
        component: () => import('@/views/logcenter/Library.vue'),
        meta: { title: '日志库' },
      },
      {
        path: '/logs/templates',
        name: 'LogCenterTemplates',
        component: () => import('@/views/logcenter/Templates.vue'),
        meta: { title: '查询模板' },
      },
      {
        path: '/logs/collectors',
        name: 'LogCenterCollectors',
        component: () => import('@/views/logcenter/Collectors.vue'),
        meta: { title: '采集接入' },
      },
    ]
  }
}

const plugin = new LogCenterPlugin()
pluginManager.register(plugin)

export default plugin
