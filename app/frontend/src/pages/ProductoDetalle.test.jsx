import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import '@testing-library/jest-dom'
import { MemoryRouter, Route, Routes } from 'react-router-dom'

import ProductoDetalle from './ProductoDetalle'
import { CarritoProvider } from '../hooks/useCarrito'

// Los tests de utils.js cubren las reglas en abstracto. Estos dos verifican que
// la PANTALLA las respeta: que el botón esté deshabilitado de verdad y que el
// talle agotado no se pueda clickear. Sin backend: se reemplaza fetch.

const PRODUCTO = {
  id: 1,
  nombre: 'Remera basica',
  descripcion: 'Algodon peinado',
  precio: 12500,
  categoria: 'remeras',
  imagenes: [{ url: '/productos/remera-basica.jpg', color: '', orden: 0, alt_text: '' }],
  variantes: [
    { id: 10, talle: 'L', color: 'Negro', stock: 0, sku: 'P1-L-NEGRO' },
    { id: 11, talle: 'M', color: 'Negro', stock: 4, sku: 'P1-M-NEGRO' },
  ],
}

function montar() {
  return render(
    <MemoryRouter initialEntries={['/productos/1']}>
      <CarritoProvider>
        <Routes>
          <Route path="/productos/:id" element={<ProductoDetalle />} />
        </Routes>
      </CarritoProvider>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  localStorage.clear()
  globalThis.fetch = vi.fn(() => Promise.resolve({
    ok: true,
    status: 200,
    json: () => Promise.resolve(PRODUCTO),
  }))
})

describe('ProductoDetalle', () => {
  it('deja el boton deshabilitado hasta elegir talle (el color viene preseleccionado)', async () => {
    montar()
    const boton = await screen.findByRole('button', { name: /agregar al carrito/i })
    expect(boton).toBeDisabled()

    await userEvent.click(screen.getByRole('button', { name: 'M' }))
    expect(boton).toBeEnabled()
  })

  it('muestra deshabilitado el talle sin stock', async () => {
    montar()
    await screen.findByRole('button', { name: 'L' })
    expect(screen.getByRole('button', { name: 'L' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'M' })).toBeEnabled()
  })

  it('guarda en el carrito la variante elegida, no el producto', async () => {
    montar()
    await screen.findByRole('button', { name: 'M' })
    await userEvent.click(screen.getByRole('button', { name: 'M' }))
    await userEvent.click(screen.getByRole('button', { name: /agregar al carrito/i }))

    await waitFor(() => {
      const carrito = JSON.parse(localStorage.getItem('carrito'))
      expect(carrito).toHaveLength(1)
      expect(carrito[0]).toMatchObject({ variante_id: 11, talle: 'M', color: 'Negro', cantidad: 1 })
    })
  })
})
