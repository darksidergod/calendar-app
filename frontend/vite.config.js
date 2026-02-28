import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: '../backend/static',
  },
  server: {
    port: 5173,
    proxy: {
      '/authurl': { target: 'http://localhost:8123', changeOrigin: true },
      '/callback': { target: 'http://localhost:8123', changeOrigin: true },
      '/listevents': { target: 'http://localhost:8123', changeOrigin: true },
      '/createevents': { target: 'http://localhost:8123', changeOrigin: true },
      '/deleteevent': { target: 'http://localhost:8123', changeOrigin: true },
    },
  },
})
