// e2e: un Chromium de verdad usando el front desplegado, que le habla a su api
// y a su base. Nada mockeado: si el navegador no le pega a la api, no es e2e.
//
// Pocas y críticas: los tres flujos que, si se rompen, la tienda no sirve. Lo
// que ya cubren los unitarios (cuentas del carrito, talles deshabilitados,
// validación del email) no se repite acá.
//
// Se busca como una persona: por el texto de los botones, los labels y los
// roles (getByRole, getByLabel), nunca por clases de CSS, que cambian con el
// diseño. Y sin sleeps: cada expect espera solo hasta que aparezca lo que busca.

import { expect, test } from '@playwright/test'
import { conToken, credencialesAdmin, crearProducto, darDeBaja, loginAdmin, nombreUnico } from './apoyo.js'

async function ingresarComoAdmin(page) {
  const { email, password } = credencialesAdmin()
  await page.goto('/login')
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Contraseña').fill(password)
  await page.getByRole('button', { name: 'Ingresar' }).click()
  await expect(page.getByText(/^Hola, /)).toBeVisible()
}

// Para afirmar que algo NO está, primero hay que esperar a que la pantalla
// termine de cargar: si no, la aserción pasaría contra el «Cargando…».
async function abrirCatalogo(page) {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Catálogo' })).toBeVisible()
  await expect(page.getByText('Cargando…')).toHaveCount(0)
}

async function abrirPanel(page) {
  await page.getByRole('link', { name: 'Admin', exact: true }).click()
  // El panel muestra su título recién cuando terminó de traer la lista.
  await expect(page.getByRole('heading', { name: 'Panel de administración' })).toBeVisible()
}

// Lo que cada test crea se limpia al final aunque haya fallado a la mitad: se
// buscan por nombre en el listado de admin (que incluye los inactivos) y se dan
// de baja los que sigan activos. Va por la api, a través del proxy del front.
let nombres = []
test.afterEach(async ({ request }) => {
  if (nombres.length === 0) return
  const token = await loginAdmin(request, '')
  const r = await request.get('/api/admin/productos', { headers: conToken(token) })
  for (const p of await r.json()) {
    if (nombres.includes(p.nombre) && p.activo) await darDeBaja(request, '', token, p)
  }
  nombres = []
})

// --- 1. Creación: crear → verlo → darlo de baja → ya no está ----------------

test('el admin crea un producto, aparece en el catálogo, y al darlo de baja desaparece', async ({ page }) => {
  const nombre = nombreUnico('e2e alta')
  nombres.push(nombre)
  await ingresarComoAdmin(page)
  await abrirPanel(page)

  await page.getByRole('button', { name: '+ Nuevo producto' }).click()
  const form = page.getByRole('form', { name: 'Nuevo producto' })
  await form.getByLabel('Nombre').fill(nombre)
  await form.getByLabel('Categoría').fill('pruebas')
  await form.getByLabel('Precio').fill('1000')
  await form.getByLabel('Color').fill('Negro')
  await form.getByLabel('Stock').fill('3')
  await form.getByRole('button', { name: 'Crear producto' }).click()
  await expect(page.getByRole('status')).toHaveText(`Producto "${nombre}" creado.`)

  // Lo ve cualquiera en el catálogo: el dato salió de la base, no del panel.
  await abrirCatalogo(page)
  await expect(page.getByRole('heading', { name: nombre })).toBeVisible()

  await abrirPanel(page)
  await page.getByRole('article').filter({ hasText: nombre })
    .getByRole('button', { name: 'Dar de baja' }).click()
  await expect(page.getByRole('status')).toContainText('se dio de baja')

  await abrirCatalogo(page)
  await expect(page.getByRole('heading', { name: nombre })).toHaveCount(0)
})

// --- 2. Error: dato inválido → el usuario ve el error → no se creó nada -----
// Una variante repetida (mismo talle y color dos veces). El front no lo impide:
// lo rechaza el backend (409), y el panel tiene que mostrar SU mensaje.

test('una variante repetida muestra el error del backend y no crea el producto', async ({ page }) => {
  const nombre = nombreUnico('e2e repetida')
  nombres.push(nombre)
  await ingresarComoAdmin(page)
  await abrirPanel(page)

  await page.getByRole('button', { name: '+ Nuevo producto' }).click()
  const form = page.getByRole('form', { name: 'Nuevo producto' })
  await form.getByLabel('Nombre').fill(nombre)
  await form.getByLabel('Precio').fill('1000')
  await form.getByLabel('Color').fill('Negro')               // M / Negro
  await form.getByRole('button', { name: '+ Agregar variante' }).click()
  await form.getByLabel('Color').nth(1).fill('Negro')        // M / Negro otra vez
  await form.getByRole('button', { name: 'Crear producto' }).click()

  await expect(page.getByRole('alert')).toContainText('repetida')

  // Y no quedó nada en la base: el panel recargado (lista activos e inactivos)
  // no lo tiene.
  await page.reload()
  await expect(page.getByRole('heading', { name: 'Panel de administración' })).toBeVisible()
  await expect(page.getByRole('article').filter({ hasText: nombre })).toHaveCount(0)
})

// --- 3. El de todos los días: comprar ----------------------------------------
// Si mañana no anda, la tienda no vende. Toca la base de punta a punta: el
// checkout descuenta stock y cancelar lo devuelve. El producto se prepara por
// la api: lo que se prueba es la compra, no el alta (eso es el test 1).

test('comprar deja el pedido en Mis pedidos, y cancelarlo devuelve el stock', async ({ page, request }) => {
  const token = await loginAdmin(request, '')
  const producto = await crearProducto(request, '', token, nombreUnico('e2e compra'), 2)
  nombres.push(producto.nombre)

  await ingresarComoAdmin(page)
  await page.goto(`/productos/${producto.id}`)
  await expect(page.getByRole('heading', { name: producto.nombre })).toBeVisible()
  await page.getByRole('button', { name: 'M', exact: true }).click()
  await page.getByRole('button', { name: 'Agregar al carrito' }).click()
  await page.getByRole('link', { name: 'Ir al carrito' }).click()

  await expect(page.getByRole('link', { name: producto.nombre })).toBeVisible()
  await page.getByRole('button', { name: 'Confirmar compra' }).click()
  await expect(page.getByRole('heading', { name: '¡Pedido confirmado!' })).toBeVisible()
  const [, numero] = (await page.getByRole('status').textContent()).match(/Pedido #(\d+)/)

  await page.getByRole('link', { name: 'Ver mis pedidos' }).click()
  const pedido = page.getByRole('article')
    .filter({ has: page.getByRole('heading', { name: `Pedido #${numero}`, exact: true }) })
  await expect(pedido).toContainText(producto.nombre)
  await pedido.getByRole('button', { name: 'Cancelar pedido' }).click()
  await expect(pedido).toContainText('cancelado')

  // El stock volvió: el detalle ofrece otra vez las 2 unidades.
  await page.goto(`/productos/${producto.id}`)
  await page.getByRole('button', { name: 'M', exact: true }).click()
  await expect(page.getByText('2 disponibles')).toBeVisible()
})
