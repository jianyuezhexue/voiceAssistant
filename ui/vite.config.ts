import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// 后端地址：dev 容器内通过 VITE_API_TARGET 注入 http://app:2500
// 本地非容器开发默认回退到 localhost:2500
const apiTarget = process.env.VITE_API_TARGET || 'http://localhost:2500'

export default defineConfig({
  plugins: [vue()],
  server: {
    host: '0.0.0.0',
    port: 2501,
    proxy: {
      '/api': {
        target: apiTarget,
        changeOrigin: true,
        ws: true, // 透传 /api/v1/chat/ws 的 WebSocket 升级
      },
    },
  },
})
