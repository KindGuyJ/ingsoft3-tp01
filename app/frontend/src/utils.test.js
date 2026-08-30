import { describe, it, expect } from 'vitest'
import {
  calcularTotal, emailValido, puedeAgregarAlCarrito, tallesDisponibles,
  imagenPrincipal, coloresDisponibles, sePuedeCancelar, ordenarTalles,
} from './utils'

describe('calcularTotal', () => {
  it('suma cantidad x precio y agrega envio por debajo del umbral', () => {
    const { total } = calcularTotal([{ precio: 10000, cantidad: 2 }])
    expect(total).toBe(25000) // 20000 + 5000 de envio
  })

  it('el envio es gratis JUSTO en el umbral', () => {
    const { envio, total } = calcularTotal([{ precio: 10000, cantidad: 5 }])
    expect(envio).toBe(0)
    expect(total).toBe(50000)
  })

  it('recalcula al cambiar la cantidad', () => {
    const uno = calcularTotal([{ precio: 10000, cantidad: 1 }])
    const tres = calcularTotal([{ precio: 10000, cantidad: 3 }])
    expect(tres.subtotal).toBe(uno.subtotal * 3)
  })
})

describe('emailValido', () => {
  it.each([
    ['brisa@ucc.edu.ar', true],
    ['sin-arroba.com', false],
    ['doble@@arroba.com', false],
    ['', false],
  ])('%s -> %s', (email, esperado) => {
    expect(emailValido(email)).toBe(esperado)
  })
})

describe('puedeAgregarAlCarrito', () => {
  const variantes = [
    { talle: 'M', color: 'Negro', stock: 3 },
    { talle: 'L', color: 'Negro', stock: 0 },
  ]

  it('no habilita sin talle elegido', () => {
    expect(puedeAgregarAlCarrito({ talle: '', color: 'Negro', variantes })).toBe(false)
  })

  it('no habilita sin color elegido', () => {
    expect(puedeAgregarAlCarrito({ talle: 'M', color: '', variantes })).toBe(false)
  })

  it('no habilita si esa combinacion no tiene stock', () => {
    expect(puedeAgregarAlCarrito({ talle: 'L', color: 'Negro', variantes })).toBe(false)
  })

  it('habilita con talle, color y stock', () => {
    expect(puedeAgregarAlCarrito({ talle: 'M', color: 'Negro', variantes })).toBe(true)
  })
})

describe('tallesDisponibles', () => {
  it('marca como no disponibles los talles sin stock', () => {
    const r = tallesDisponibles(
      [{ talle: 'M', color: 'Negro', stock: 2 }, { talle: 'L', color: 'Negro', stock: 0 }],
      'Negro',
    )
    expect(r).toEqual([
      { talle: 'M', disponible: true },
      { talle: 'L', disponible: false },
    ])
  })
})

// --- Helpers de presentación agregados en la Fase 3 -------------------------

describe('imagenPrincipal', () => {
  const imagenes = [
    { url: '/productos/a.jpg', color: '', orden: 0 },
    { url: '/productos/a-rojo-2.jpg', color: 'Rojo', orden: 2 },
    { url: '/productos/a-rojo-1.jpg', color: 'Rojo', orden: 1 },
  ]

  it('prefiere la foto del color elegido, la de orden mas bajo', () => {
    expect(imagenPrincipal(imagenes, 'Rojo').url).toBe('/productos/a-rojo-1.jpg')
  })

  it('cae en la generica si el color no tiene foto propia', () => {
    expect(imagenPrincipal(imagenes, 'Azul').url).toBe('/productos/a.jpg')
  })

  it('devuelve null si el producto no tiene imagenes', () => {
    expect(imagenPrincipal([], 'Rojo')).toBe(null)
  })
})

describe('coloresDisponibles', () => {
  it('no repite colores aunque haya varios talles', () => {
    expect(coloresDisponibles([
      { talle: 'M', color: 'Negro' },
      { talle: 'L', color: 'Negro' },
      { talle: 'M', color: 'Blanco' },
    ])).toEqual(['Negro', 'Blanco'])
  })
})

describe('sePuedeCancelar', () => {
  it.each([
    ['pendiente', true],
    ['pagado', true],
    ['enviado', false],
    ['entregado', false],
    ['cancelado', false],
  ])('%s -> %s', (estado, esperado) => {
    expect(sePuedeCancelar(estado)).toBe(esperado)
  })
})

describe('ordenarTalles', () => {
  it('ordena por talle y no por el orden en que vinieron de la base', () => {
    const r = ordenarTalles([
      { talle: 'L', disponible: true },
      { talle: 'XS', disponible: false },
      { talle: 'M', disponible: true },
    ])
    expect(r.map((x) => x.talle)).toEqual(['XS', 'M', 'L'])
  })
})
