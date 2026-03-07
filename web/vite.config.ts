import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { visualizer } from 'rollup-plugin-visualizer'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    // 5.1.11 打包体积分析
    visualizer({
      open: false,
      filename: 'dist/stats.html',
      gzipSize: true,
      brotliSize: true,
    }),
  ],
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setupTests.ts',
    globals: false,
    css: true,
    testTimeout: 15000,
    hookTimeout: 15000,
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://127.0.0.1:8080',
        ws: true,
      },
    },
  },
  // 5.1.8 Vite 打包优化
  build: {
    // 5.1.10 启用代码压缩
    minify: 'terser',
    terserOptions: {
      compress: {
        drop_console: true, // 生产环境移除 console
        drop_debugger: true,
      },
    },
    // 代码分割优化
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules')) {
            if (id.includes('/react/') || id.includes('/react-dom/') || id.includes('/react-router-dom/')) {
              return 'react-vendor'
            }
            // antd 和 @ant-design/icons 必须在同一个 chunk 中，避免循环依赖导致的初始化错误
            if (id.includes('/antd/') || id.includes('/@ant-design/icons/') || id.includes('/@ant-design/cssinjs/') || id.includes('/rc-')) {
              return 'antd-vendor'
            }
            // @ant-design/x 和 @ant-design/x-markdown 合并到 ai-ui，避免循环依赖
            if (id.includes('/@ant-design/x/') || id.includes('/@ant-design/x-markdown/')) {
              return 'ai-ui'
            }
            if (id.includes('/framer-motion/')) {
              return 'animation-vendor'
            }
            // react-markdown 等依赖 react，合并到 react-vendor 避免循环依赖
            if (
              id.includes('/react-markdown/') ||
              id.includes('/react-syntax-highlighter/') ||
              id.includes('/remark-gfm/') ||
              id.includes('/unified/') ||
              id.includes('/remark/') ||
              id.includes('/rehype/')
            ) {
              return 'react-vendor'
            }
            if (
              id.includes('/@monaco-editor/') ||
              id.includes('/monaco-editor/') ||
              id.includes('/xterm/') ||
              id.includes('/xterm-addon-fit/')
            ) {
              return 'editor-vendor'
            }
            if (id.includes('/@ant-design/charts/') || id.includes('/recharts/')) {
              return 'charts-vendor'
            }
            if (id.includes('/axios/') || id.includes('/dayjs/') || id.includes('/ahooks/') || id.includes('/cmdk/')) {
              return 'utils-vendor'
            }
          }

          return undefined
        },
        // 文件命名
        chunkFileNames: 'assets/js/[name]-[hash].js',
        entryFileNames: 'assets/js/[name]-[hash].js',
        assetFileNames: 'assets/[ext]/[name]-[hash].[ext]',
      },
    },
    // 5.1.9 启用 Tree Shaking (默认启用)
    // chunk 大小警告限制
    chunkSizeWarningLimit: 1000,
    // 启用 CSS 代码分割
    cssCodeSplit: true,
    // 生成 source map
    sourcemap: false,
  },
  // 优化依赖预构建
  optimizeDeps: {
    include: [
      'react',
      'react-dom',
      'react-router-dom',
      'antd',
      '@ant-design/icons',
      'framer-motion',
      'axios',
    ],
  },
})
