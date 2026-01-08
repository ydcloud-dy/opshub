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
    proxy: {
      '/api': {
        target: 'http://localhost:9876',
        changeOrigin: true
      }
    }
  }
})
