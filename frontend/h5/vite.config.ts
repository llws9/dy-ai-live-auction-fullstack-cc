import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      // 直播 WebSocket 直连 auction 独立 WS 端口（gateway 仅做 endpoint discovery，不转发 WS 升级）。
      // 必须排在 '/api' 之前匹配。后端 WS 路由为 '/ws'，需把 '/api/v1/ws' 重写为 '/ws'。
      '/api/v1/ws': {
        target: 'ws://127.0.0.1:8083',
        ws: true,
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/api\/v1\/ws/, '/ws'),
      },
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          // React 核心库
          'react-vendor': ['react', 'react-dom', 'react-router-dom', 'scheduler'],
        },
      },
    },
    // 提高chunk大小警告阈值
    chunkSizeWarningLimit: 500,
  },
})
