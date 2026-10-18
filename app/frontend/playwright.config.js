import { defineConfig } from '@playwright/test'

// Dos suites con la misma herramienta, en e2e/:
//   api.spec.js     integración: pedidos HTTP a la api, sin navegador
//   tienda.spec.js  e2e: un Chromium de verdad usando el front
// Cada job del pipeline corre la suya por nombre de archivo.
//
// Las direcciones entran por variables de entorno, nunca escritas acá: la misma
// suite corre contra el compose local o contra QA.
//   E2E_BASE_URL  el FRONT (lo usa tienda.spec.js, y la api a través del proxy
//                 /api de nginx para preparar y limpiar datos)
//   API_BASE_URL  la API directa (lo usa api.spec.js). No es la del front.
export default defineConfig({
  testDir: './e2e',
  // Topes generosos: QA es plan gratuito y despierta lento. El smoke del job
  // de deploy ya lo despertó, pero un pedido suelto por internet puede tardar.
  timeout: 60_000,                 // tope de CADA test
  expect: { timeout: 15_000 },     // tope de CADA aserción (no hereda el de arriba; el default es 5 s)
  use: {
    baseURL: process.env.E2E_BASE_URL || 'http://localhost:3000',
    trace: 'on-first-retry',       // la traza se graba en el reintento de un fallo
    screenshot: 'only-on-failure',
  },
  // UN reintento: absorbe una demora suelta. Si un test pasa recién ahí, el
  // reporte lo marca "flaky" aunque la corrida quede verde: hay que mirarlo.
  retries: 1,
  reporter: [['html', { open: 'never' }], ['list']],
})
