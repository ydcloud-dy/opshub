import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  optimizeDeps: {
    include: ['js-yaml']
  },
  server: {
    port: 5173,
    host: "0.0.0.0",
    proxy: {
      '/api/v1/logs': {
        target: process.env.VITE_LOG_GATEWAY_URL || 'http://localhost:9880',
        changeOrigin: true,
        xfwd: true
      },
      '/api': {
        target: process.env.VITE_API_BASE_URL || 'http://localhost:9876',
        changeOrigin: true,
        xfwd: true,
        ws: true,  // 启用 WebSocket 代理
        cookieDomainRewrite: '',  // 重写 cookie domain
        cookiePathRewrite: '/'    // 重写 cookie path
      },
      '/oauth2': {
        target: process.env.VITE_API_BASE_URL || 'http://localhost:9876',
        changeOrigin: true,
        cookieDomainRewrite: '',  // 重写 cookie domain
        cookiePathRewrite: '/'    // 重写 cookie path
      }
    }
  }
})
