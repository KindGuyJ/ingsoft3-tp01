// Lo que comparten las dos suites. No es un spec: Playwright solo corre los
// archivos *.spec.js, así que este no aparece como test.
//
// Las pruebas trabajan siempre sobre un producto QUE ELLAS CREAN, con un nombre
// que no se repite, y al final lo dan de baja. Nunca tocan el catálogo del seed:
// QA es uno solo y lo comparten todas las corridas.

import { expect } from '@playwright/test'

// El admin de QA. Crear productos es de admin, y la api no tiene otra puerta.
// Llega por variables de entorno (en el pipeline, secrets del environment qa):
// nunca escrito acá, porque el repo es público. Sin ellas se corta con un
// mensaje claro, en vez de un 401 que haga pensar que la api está rota.
export function credencialesAdmin() {
  const email = process.env.E2E_ADMIN_EMAIL
  const password = process.env.E2E_ADMIN_PASSWORD
  if (!email || !password) {
    throw new Error('Faltan E2E_ADMIN_EMAIL y E2E_ADMIN_PASSWORD (el admin del entorno contra el que se corre).')
  }
  return { email, password }
}

// Un nombre que ninguna otra corrida usa: la hora en milisegundos y un sufijo
// al azar, por si dos pruebas arrancan en el mismo milisegundo.
export function nombreUnico(prefijo) {
  return `${prefijo} ${Date.now()}-${Math.floor(Math.random() * 1000)}`
}

export const conToken = (token) => ({ Authorization: `Bearer ${token}` })

// `base` es la dirección de la api: API_BASE_URL en la suite de integración, y
// '' (relativa al front, por el proxy /api de nginx) en la de navegador.
export async function loginAdmin(request, base) {
  const r = await request.post(`${base}/api/usuarios/login`, { data: credencialesAdmin() })
  expect(r.status(), 'login del admin').toBe(200)
  return (await r.json()).token
}

export async function crearProducto(request, base, token, nombre, stock = 3) {
  const r = await request.post(`${base}/api/productos`, {
    headers: conToken(token),
    data: {
      nombre,
      precio: 1000,
      categoria: 'pruebas',
      variantes: [{ talle: 'M', color: 'Negro', stock }],
    },
  })
  expect(r.status(), 'alta del producto de prueba').toBe(201)
  return r.json()
}

// La limpieza: la baja es lógica (activo = false), como en el panel. Un
// producto no se borra de la base: los pedidos lo referencian. Es idempotente,
// así que se puede llamar aunque la prueba ya lo haya dado de baja.
// Y se COMPRUEBA: dado de baja, el detalle público contesta 404 (regla 9).
export async function darDeBaja(request, base, token, producto) {
  const r = await request.put(`${base}/api/productos/${producto.id}`, {
    headers: conToken(token),
    data: {
      nombre: producto.nombre,
      descripcion: producto.descripcion,
      precio: producto.precio,
      categoria: producto.categoria,
      activo: false,
    },
  })
  expect(r.status(), 'baja del producto de prueba').toBe(200)
  expect((await request.get(`${base}/api/productos/${producto.id}`)).status()).toBe(404)
}
