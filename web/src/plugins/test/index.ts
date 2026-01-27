import type { Plugin, PluginMenuConfig, PluginRouteConfig } from '../types'
import { pluginManager } from '../manager'
import TestHome from './components/TestHome.vue'

/**
 * 测试插件
 */
class TestPlugin implements Plugin {
  name = 'test'
  description = '这是一个简单的测试插件，用于测试插件安装功能'
  version = '1.0.0'
  author = 'Test Team'

  async install() {
    console.log('[Test Plugin] 插件安装中...')
  }

  async uninstall() {
    console.log('[Test Plugin] 插件卸载中...')
  }

  getMenus(): PluginMenuConfig[] {
    return [
      {
        name: '测试插件',
        path: '/test',
        icon: 'Grape',
        sort: 95,
        hidden: false,
        parentPath: '',
      },
      {
        name: '测试首页',
        path: '/test/home',
        icon: 'House',
        sort: 1,
        hidden: false,
        parentPath: '/test',
      }
    ]
  }

  getRoutes(): PluginRouteConfig[] {
    return [
      {
        path: '/test/home',
        name: 'TestHome',
        component: TestHome,
        meta: { title: '测试首页' }
      }
    ]
  }
}

// 创建插件实例并注册
const testPlugin = new TestPlugin()
pluginManager.register(testPlugin)

export default testPlugin
