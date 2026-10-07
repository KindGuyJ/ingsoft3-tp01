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
    coverage: {
      provider: 'v8',
      // json-summary deja coverage-summary.json: lo lee el pipeline para el Summary.
      reporter: ['text', 'html', 'json-summary'],
      // Entra TODO src/ (también lo que ningún test importa: un archivo nuevo sin
      // tests baja el número), menos el arranque. Ver decisiones.md (TP5).
      include: ['src/**/*.{js,jsx}'],
      exclude: [
        'src/main.jsx', // monta React en el DOM
        'src/App.jsx',  // solo declara las rutas
        'src/**/*.test.{js,jsx}',
      ],
      // El umbral: medido 56 de líneas y 81 de ramas. Ver decisiones.md (TP5).
      thresholds: { lines: 50, branches: 75 },
    },
  },
})
