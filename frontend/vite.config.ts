import path from 'node:path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { tanstackRouter } from '@tanstack/router-plugin/vite'

// Маршруты — файловые (src/routes), дерево генерируется плагином в src/routeTree.gen.ts.
export default defineConfig({
  plugins: [tanstackRouter({ target: 'react', autoCodeSplitting: true }), react(), tailwindcss()],
  resolve: {
    alias: { '@': path.resolve(import.meta.dirname, './src') },
  },
  server: {
    port: 5173,
    // В разработке API отдаёт Go-сервер (по умолчанию :8080), cookie остаются на одном origin.
    // Другой порт бэкенда: VITE_API_TARGET=http://localhost:18084 npm run dev
    proxy: { '/api': process.env.VITE_API_TARGET ?? 'http://localhost:8080' },
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
  },
})
