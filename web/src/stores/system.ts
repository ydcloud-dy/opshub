import { defineStore } from 'pinia'
import { getPublicConfig, getBasicConfig } from '@/api/system'

interface SystemState {
  systemName: string
  systemLogo: string
  systemDescription: string
  loaded: boolean
}

export const useSystemStore = defineStore('system', {
  state: (): SystemState => ({
    systemName: 'OpsHub',
    systemLogo: '',
    systemDescription: '运维管理平台',
    loaded: false
  }),

  actions: {
    // 加载公开配置（无需认证）
    async loadPublicConfig() {
      try {
        const res = await getPublicConfig()
        if (res) {
          this.systemName = res.systemName || 'OpsHub'
          this.systemLogo = res.systemLogo || ''
          this.systemDescription = res.systemDescription || '运维管理平台'
          this.loaded = true
          this.updatePageMeta()
        }
      } catch (error) {
        console.error('加载系统配置失败', error)
      }
    },

    // 加载完整配置（需要认证）
    async loadFullConfig() {
      try {
        const res = await getBasicConfig()
        if (res) {
          this.systemName = res.systemName || 'OpsHub'
          this.systemLogo = res.systemLogo || ''
          this.systemDescription = res.systemDescription || '运维管理平台'
          this.loaded = true
          this.updatePageMeta()
        }
      } catch (error) {
        console.error('加载系统配置失败', error)
      }
    },

    // 更新配置（保存后调用）
    updateConfig(config: { systemName?: string; systemLogo?: string; systemDescription?: string }) {
      if (config.systemName !== undefined) {
        this.systemName = config.systemName
      }
      if (config.systemLogo !== undefined) {
        this.systemLogo = config.systemLogo
      }
      if (config.systemDescription !== undefined) {
        this.systemDescription = config.systemDescription
      }
      this.updatePageMeta()
    },

    // 更新页面标题和favicon
    updatePageMeta() {
      // 更新页面标题
      document.title = this.systemName || 'OpsHub'

      // 更新favicon
      if (this.systemLogo) {
        const link = document.querySelector("link[rel*='icon']") as HTMLLinkElement
        if (link) {
          link.href = this.systemLogo
        } else {
          const newLink = document.createElement('link')
          newLink.rel = 'icon'
          newLink.href = this.systemLogo
          document.head.appendChild(newLink)
        }
      }
    }
  }
})
