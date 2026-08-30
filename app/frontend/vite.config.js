import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    // El front SIEMPRE llama a rutas relativas (/api/...), sin host ni puerto.
    // En desarrollo las traduce este proxy; en el contenedor, nginx.
    // Consecuencia: la MISMA imagen sirve en dev, QA y PROD (importa en el TP6),
    // y como para el browser todo sale del mismo origen, no hay CORS.
    proxy: {
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
      '/uploads': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
  },
})
