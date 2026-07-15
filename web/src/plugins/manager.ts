import type { Plugin } from './types'
import { ElMessage } from 'element-plus'
import router from '@/router'

class PluginManagerImpl {
  private plugins: Map<string, Plugin> = new Map()
  private installedPlugins: Set<string> = new Set()
  private STORAGE_KEY = 'opshub_installed_plugins'

  constructor() {
    // 从 localStorage 恢复已安装插件列表
    this.loadInstalledPlugins()
  }

  /**
   * 从 localStorage 加载已安装插件列表
   */
  private loadInstalledPlugins() {
    try {
      const stored = localStorage.getItem(this.STORAGE_KEY)
      if (stored) {
        const plugins = JSON.parse(stored)
        this.installedPlugins = new Set(plugins)
      }
    } catch (error) {
      // 加载失败时静默处理
    }
  }

  /**
   * 保存已安装插件列表到 localStorage
   */
  private saveInstalledPlugins() {
    try {
      const plugins = Array.from(this.installedPlugins)
      localStorage.setItem(this.STORAGE_KEY, JSON.stringify(plugins))
    } catch (error) {
      // 保存失败时静默处理
    }
  }

  /**
   * 注册插件（仅注册，不安装）
   */
  register(plugin: Plugin) {
    this.plugins.set(plugin.name, plugin)
  }

  /**
   * 安装插件
   */
  async install(name: string, showMessage: boolean = true): Promise<boolean> {
    const plugin = this.plugins.get(name)
    if (!plugin) {
      if (showMessage) {
        ElMessage.error(`插件 ${name} 不存在`)
      }
      return false
    }

    // 如果已经安装过，静默跳过
    if (this.installedPlugins.has(name)) {
      // 仍然需要注册路由（因为刷新页面后路由会丢失）
      if (plugin.getRoutes) {
        const routes = plugin.getRoutes()
        routes.forEach(route => {
          if (route.name && router.hasRoute(route.name)) {
            return
          }
          router.addRoute('Layout', route)
        })
      }

      return true
    }

    try {
      // 执行插件的安装方法
      await plugin.install()

      // 注册路由
      if (plugin.getRoutes) {
        const routes = plugin.getRoutes()
        routes.forEach(route => {
          if (route.name && router.hasRoute(route.name)) {
            return
          }
          router.addRoute('Layout', route)
        })
      }

      // 标记为已安装
      this.installedPlugins.add(name)
      this.saveInstalledPlugins()

      if (showMessage) {
        ElMessage.success(`插件 ${name} 安装成功`)
      }
      return true
    } catch (error) {
      if (showMessage) {
        ElMessage.error(`插件 ${name} 安装失败`)
      }
      return false
    }
  }

  /**
   * 卸载插件
   */
  async uninstall(name: string): Promise<boolean> {
    const plugin = this.plugins.get(name)
    if (!plugin) {
      ElMessage.error(`插件 ${name} 不存在`)
      return false
    }

    if (!this.installedPlugins.has(name)) {
      ElMessage.warning(`插件 ${name} 未安装`)
      return false
    }

    try {
      // 执行插件的卸载方法
      await plugin.uninstall()

      // 移除路由（注意：vue-router 不支持直接移除路由，需要刷新页面）
      // 这里只能标记为未安装，实际移除路由需要刷新页面
      this.installedPlugins.delete(name)
      this.saveInstalledPlugins()

      ElMessage.success(`插件 ${name} 已卸载，请刷新页面以完全移除`)
      return true
    } catch (error) {
      ElMessage.error(`插件 ${name} 卸载失败`)
      return false
    }
  }

  /**
   * 获取插件
   */
  get(name: string): Plugin | undefined {
    return this.plugins.get(name)
  }

  /**
   * 获取所有已注册的插件
   */
  getAll(): Plugin[] {
    return Array.from(this.plugins.values())
  }

  /**
   * 检查插件是否已安装
   */
  isInstalled(name: string): boolean {
    return this.installedPlugins.has(name)
  }

  /**
   * 获取所有已安装的插件
   */
  getInstalled(): Plugin[] {
    return Array.from(this.plugins.values()).filter(p => this.installedPlugins.has(p.name))
  }
}

export const pluginManager = new PluginManagerImpl()
