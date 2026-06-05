/// <reference types="vitest" />
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

// 前端开发端口 5174（避开 admin 默认 5173）；生产由 gateway 反代
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { '@': path.resolve(__dirname, 'src') },
    // 防止 recharts 等三方库引入第二份 React 副本，导致 "Invalid hook call"
    dedupe: ['react', 'react-dom'],
  },
  optimizeDeps: {
    include: ['react', 'react-dom', 'recharts'],
  },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    globals: true,
  },
  server: {
    port: 5174,
    host: '0.0.0.0',
    // dev 阶段通过 gateway:8080 访问 /api 和 /ws
    proxy: {
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
      '/ws': { target: 'http://localhost:8080', changeOrigin: true, ws: true },
    },
  },
});
