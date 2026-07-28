import type { Plugin, PluginMenuConfig, PluginRouteConfig } from '../types'
import { pluginManager } from '../manager'
import './styles.css'

class AppInventoryPlugin implements Plugin {
  name = 'app-inventory'
  description = '应用、环境、域名、证书、资源、数据库、中间件和依赖拓扑资产中心'
  version = '0.1.0'
  author = 'OpsHub'

  async install() {}
  async uninstall() {}

  getMenus(): PluginMenuConfig[] {
    const parent = '/app-inventory'
    return [
      { name: '应用资产', path: parent, icon: 'Collection', sort: 18, hidden: false, parentPath: '' },
      { name: '资产总览', path: `${parent}/overview`, icon: 'DataAnalysis', sort: 1, hidden: false, parentPath: parent },
      { name: '应用台账', path: `${parent}/apps`, icon: 'Grid', sort: 2, hidden: false, parentPath: parent },
      { name: '环境管理', path: `${parent}/environments`, icon: 'SetUp', sort: 3, hidden: false, parentPath: parent },
      { name: '依赖拓扑', path: `${parent}/topology`, icon: 'Share', sort: 4, hidden: false, parentPath: parent },
      { name: '资源与域名', path: `${parent}/resources`, icon: 'Coin', sort: 5, hidden: false, parentPath: parent },
      { name: '凭据中心', path: `${parent}/credentials`, icon: 'Lock', sort: 6, hidden: false, parentPath: parent },
    ]
  }

  getRoutes(): PluginRouteConfig[] {
    const parent = '/app-inventory'
    return [
      { path: parent, name: 'AppInventory', redirect: `${parent}/overview`, component: () => import('./components/Overview.vue'), meta: { title: '应用资产' } } as PluginRouteConfig,
      { path: `${parent}/overview`, name: 'AppInventoryOverview', component: () => import('./components/Overview.vue'), meta: { title: '资产总览' } },
      { path: `${parent}/apps`, name: 'AppInventoryApps', component: () => import('./components/Applications.vue'), meta: { title: '应用台账' } },
      { path: `${parent}/apps/:id`, name: 'AppInventoryDetail', component: () => import('./components/ApplicationDetail.vue'), meta: { title: '应用详情', activeMenu: `${parent}/apps` } },
      { path: `${parent}/environments`, name: 'AppInventoryEnvironments', component: () => import('./components/Environments.vue'), meta: { title: '环境管理' } },
      { path: `${parent}/topology`, name: 'AppInventoryTopology', component: () => import('./components/Topology.vue'), meta: { title: '依赖拓扑' } },
      { path: `${parent}/resources`, name: 'AppInventoryResources', component: () => import('./components/Resources.vue'), meta: { title: '资源与域名' } },
      { path: `${parent}/credentials`, name: 'AppInventoryCredentials', component: () => import('@/views/credentials/UnifiedCredentials.vue'), meta: { title: '凭据中心' } },
    ]
  }
}

const plugin = new AppInventoryPlugin()
pluginManager.register(plugin)

export default plugin
