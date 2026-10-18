// Integración: la api DESPLEGADA, con su base de verdad, sin navegador y sin
// ningún doble en el medio. Es el escalón entre los unitarios del TP5 (que
// reemplazan la base por un fake en memoria) y las e2e (que ven la pantalla).
//
// Lo que ve esta suite y los unitarios no: lo que la base acepta, rechaza y
// guarda. Cada GET que sigue a un POST le pregunta a Postgres, no a un fake.

import { expect, test } from '@playwright/test'
import { conToken, crearProducto, darDeBaja, loginAdmin, nombreUnico } from './apoyo.js'

// La API, no el front: son dos servicios con dos URLs. Sin barra al final.
const API = process.env.API_BASE_URL || 'http://localhost:8080'

async function nombresDelCatalogo(request) {
  const r = await request.get(`${API}/api/productos`)
  expect(r.status()).toBe(200)
  return (await r.json()).map((p) => p.nombre)
}

// Lo que cada prueba crea se anota acá, y se da de baja al final AUNQUE la
// prueba haya fallado a la mitad: si no, cada rojo dejaría basura en QA.
let creados = []
test.afterEach(async ({ request }) => {
  if (creados.length === 0) return
  const token = await loginAdmin(request, API)
  for (const p of creados) await darDeBaja(request, API, token, p)
  creados = []
})

test('el alta guarda el producto en la base, y la baja lo saca del catálogo', async ({ request }) => {
  const token = await loginAdmin(request, API)
  const nombre = nombreUnico('api alta')

  const alta = await request.post(`${API}/api/productos`, {
    headers: conToken(token),
    data: {
      nombre,
      precio: 1000,
      categoria: 'pruebas',
      variantes: [{ talle: 'M', color: 'Negro', stock: 3 }],
    },
  })
  expect(alta.status()).toBe(201)                           // la api dice «creado»
  const creado = await alta.json()
  creados.push(creado)

  // ...y la BASE lo devuelve, con su variante: no es un doble contestando.
  const detalle = await request.get(`${API}/api/productos/${creado.id}`)
  expect(detalle.status()).toBe(200)
  expect((await detalle.json()).variantes).toEqual([
    expect.objectContaining({ talle: 'M', color: 'Negro', stock: 3 }),
  ])
  expect(await nombresDelCatalogo(request)).toContain(nombre)

  // La baja (lógica: activo = false) y su efecto, comprobado contra la base:
  // el detalle público da 404 y el catálogo ya no lo lista (regla 9).
  const baja = await request.put(`${API}/api/productos/${creado.id}`, {
    headers: conToken(token),
    data: { nombre, precio: 1000, categoria: 'pruebas', activo: false },
  })
  expect(baja.status()).toBe(200)
  expect((await request.get(`${API}/api/productos/${creado.id}`)).status()).toBe(404)
  expect(await nombresDelCatalogo(request)).not.toContain(nombre)
})

test('un producto con precio 0 lo rechaza la api, y no se crea nada', async ({ request }) => {
  const token = await loginAdmin(request, API)
  const nombre = nombreUnico('api invalido')

  const alta = await request.post(`${API}/api/productos`, {
    headers: conToken(token),
    data: { nombre, precio: 0, variantes: [{ talle: 'M', color: 'Negro', stock: 1 }] },
  })
  expect(alta.status()).toBe(400)                           // el contrato del error (regla 8)

  // Y de verdad no se creó nada: ni siquiera dado de baja (el listado de admin
  // incluye los inactivos). Se busca por el nombre único y no se cuentan los
  // productos: la cuenta depende de lo que otra corrida esté haciendo en QA.
  const lista = await request.get(`${API}/api/admin/productos`, { headers: conToken(token) })
  expect(lista.status()).toBe(200)
  expect((await lista.json()).map((p) => p.nombre)).not.toContain(nombre)
})

// La tercera, elegida: la compra. Es lo que la tienda hace todos los días, y si
// se rompe no vende. Prueba en la base de verdad lo que los unitarios prueban
// contra un fake: que el checkout descuenta el stock (regla 2), que el precio
// del pedido es el del producto (regla 4) y que cancelar lo devuelve (regla 6).
test('comprar descuenta el stock en la base, y cancelar el pedido lo devuelve', async ({ request }) => {
  const token = await loginAdmin(request, API)
  const producto = await crearProducto(request, API, token, nombreUnico('api compra'), 3)
  creados.push(producto)
  const variante = producto.variantes[0]

  async function stock() {
    const r = await request.get(`${API}/api/productos/${producto.id}`)
    expect(r.status()).toBe(200)
    return (await r.json()).variantes[0].stock
  }

  const compra = await request.post(`${API}/api/pedidos`, {
    headers: conToken(token),
    data: { items: [{ variante_id: variante.id, cantidad: 2 }] },
  })
  expect(compra.status()).toBe(201)
  const pedido = await compra.json()
  expect(pedido.items).toEqual([
    expect.objectContaining({ variante_id: variante.id, cantidad: 2, precio_unitario: 1000 }),
  ])
  expect(await stock()).toBe(1)                             // 3 − 2, guardado en la base

  const cancelar = await request.post(`${API}/api/pedidos/${pedido.id}/cancelar`, {
    headers: conToken(token),
  })
  expect(cancelar.status()).toBe(204)
  expect(await stock()).toBe(3)                             // volvió todo

  const mios = await request.get(`${API}/api/pedidos`, { headers: conToken(token) })
  expect((await mios.json()).find((p) => p.id === pedido.id)?.estado).toBe('cancelado')
})
