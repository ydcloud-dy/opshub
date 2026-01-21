import type { Plugin, PluginMenuConfig, PluginRouteConfig } from '../types'
import { pluginManager } from '../manager'

/**
 * Task 任务中心插件
 * 提供执行任务、模板管理和文件分发功能
 */
class TaskPlugin implements Plugin {
  name = 'task'
  description = '任务中心插件，提供执行任务、模板管理和文件分发功能'
  version = '1.0.0'
  author = 'OpsHub Team'

  /**
   * 安装插件
   */
  async install() {
    console.log('Task 插件安装中...')
  }

  /**
   * 卸载插件
   */
  async uninstall() {
    console.log('Task 插件卸载中...')
  }

  /**
   * 获取插件菜单配置
   */
  getMenus(): PluginMenuConfig[] {
    const parentPath = '/task'

    return [
      {
        name: '任务中心',
        path: parentPath,
        icon: 'Tickets',
        sort: 90,
        hidden: false,
        parentPath: '',
      },
      {
        name: '执行任务',
        path: '/task/execute',
        icon: 'VideoPlay',
        sort: 1,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '模板管理',
        path: '/task/templates',
        icon: 'Document',
        sort: 2,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '文件分发',
        path: '/task/file-distribution',
        icon: 'FolderOpened',
        sort: 3,
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
        path: '/task',
        name: 'Task',
        component: () => import('@/views/task/Index.vue'),
        meta: { title: '任务中心' },
        children: [
          {
            path: 'execute',
            name: 'TaskExecute',
            component: () => import('@/views/task/Execute.vue'),
            meta: { title: '执行任务' },
          },
          {
            path: 'templates',
            name: 'TaskTemplates',
            component: () => import('@/views/task/Templates.vue'),
            meta: { title: '模板管理' },
          },
          {
            path: 'file-distribution',
            name: 'TaskFileDistribution',
            component: () => import('@/views/task/FileDistribution.vue'),
            meta: { title: '文件分发' },
          },
        ],
      },
    ]
  }
}

// 创建并注册插件实例
const plugin = new TaskPlugin()
console.log('[Task Plugin] 正在注册插件...')
pluginManager.register(plugin)
console.log('[Task Plugin] 插件注册完成')

export default plugin
