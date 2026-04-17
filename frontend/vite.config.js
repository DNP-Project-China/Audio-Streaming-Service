import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': 'http://mock-api:8080',
      '/tracking': 'http://mock-tracking:8081',
      '/stats': 'http://mock-analytics:8082'
    }
  }
})