import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import 'element-plus/dist/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import App from './App.vue'
import router, { registerPluginRoutes } from './router'
import { pluginManager } from './plugins/manager'

// 导入插件（插件会自动注册到 pluginManager）
import '@/plugins/kubernetes'

const app = createApp(App)
const pinia = createPinia()

// 注册所有图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.use(pinia)

// 自动安装所有已注册的插件
async function installPlugins() {
  const plugins = pluginManager.getAll()
  console.log('=== 开始安装插件 ===')
  console.log('已注册的插件数量:', plugins.length)
  console.log('已注册的插件列表:', plugins.map(p => p.name))

  for (const plugin of plugins) {
    console.log(`正在安装插件: ${plugin.name}`)
    const result = await pluginManager.install(plugin.name, false) // 不显示消息
    console.log(`插件 ${plugin.name} 安装${result ? '成功' : '失败'}`)
  }

  console.log('=== 插件安装完成 ===')
  const installed = pluginManager.getInstalled()
  console.log('已安装的插件数量:', installed.length)
  console.log('已安装的插件列表:', installed.map(p => p.name))
}

// 安装插件并注册路由
installPlugins().then(() => {
  // 注册插件路由 - 必须在 app.use(router) 之前
  registerPluginRoutes()

  app.use(router)
  app.use(ElementPlus, {
    locale: zhCn,
  })

  app.mount('#app')

  // 全局字体大小调整
  document.documentElement.style.fontSize = '20px'
})
