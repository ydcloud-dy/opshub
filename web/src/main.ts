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
import '@/plugins/monitor'
import '@/plugins/task'
import '@/plugins/test'
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

  for (const plugin of plugins) {
    await pluginManager.install(plugin.name, false) // 不显示消息
  }
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
