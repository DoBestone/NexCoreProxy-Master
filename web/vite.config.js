import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  server: {
    port: 9300,
    proxy: {
      '/api': {
        target: 'http://localhost:9310',
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: 'dist',
    chunkSizeWarningLimit: 800,
    rollupOptions: {
      output: {
        manualChunks: {
          // Vue 生态独立
          'vue-vendor': ['vue', 'vue-router'],
          // AntD 主体（最大头）
          'antd': ['ant-design-vue'],
          // AntD 图标 (按需收敛)
          'antd-icons': ['@ant-design/icons-vue'],
          // ECharts 重型
          'echarts': ['echarts'],
        },
      },
    },
  },
})