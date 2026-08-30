import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import '@testing-library/jest-dom'
import { MemoryRouter } from 'react-router-dom'

import AdminPanel from './AdminPanel'
import { AuthProvider } from '../hooks/useAuth'

// El panel es la única pantalla que ve los productos dados de baja. Estos tests
// verifican eso mismo: que pida el listado de admin (y no el catálogo público),
// que el filtro los separe, y que la baja y la corrección de stock salgan por
// los endpoints que corresponden.

const ACTIVO = {
  id: 1, nombre: 'Remera basica', descripcion: 'Algodon', precio: 12500,
  categoria: 'remeras', activo: true, imagenes: [],
  variantes: [{ id: 10, talle: 'M', color: 'Negro', stock: 4, sku: 'P1-M-NEGRO' }],
}

const DE_BAJA = {
  id: 2, nombre: 'Campera vieja', descripcion: '', precio: 40000,
  categoria: 'camperas', activo: false, imagenes: [],
  variantes: [{ id: 20, talle: 'L', color: 'Azul', stock: 0, sku: 'P2-L-AZUL' }],
}

let llamadas = []

function montar() {
  return render(
    <MemoryRouter>
      <AuthProvider>
        <AdminPanel />
      </AuthProvider>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  localStorage.clear()
  localStorage.setItem('token', 'un-token')
  localStorage.setItem('usuario', JSON.stringify({ id: 1, nombre: 'Admin', es_admin: true }))
  llamadas = []

  globalThis.fetch = vi.fn((url, opciones = {}) => {
    llamadas.push({ url, metodo: opciones.method || 'GET', body: opciones.body })
    const cuerpo = url.startsWith('/api/admin/productos') ? [ACTIVO, DE_BAJA] : {}
    return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(cuerpo) })
  })
})

describe('AdminPanel', () => {
  it('pide el listado de admin, que incluye los dados de baja', async () => {
    montar()
    await screen.findByText(/Campera vieja/)

    expect(llamadas[0].url).toBe('/api/admin/productos')
    expect(screen.getByText(/dado de baja/i)).toBeInTheDocument()
  })

  it('el filtro separa activos de dados de baja', async () => {
    montar()
    await screen.findByText(/Campera vieja/)

    await userEvent.click(screen.getByRole('button', { name: /Activos \(1\)/ }))
    expect(screen.getByText(/Remera basica/)).toBeInTheDocument()
    expect(screen.queryByText(/Campera vieja/)).toBeNull()

    await userEvent.click(screen.getByRole('button', { name: /Dados de baja \(1\)/ }))
    expect(screen.queryByText(/Remera basica/)).toBeNull()
    expect(screen.getByText(/Campera vieja/)).toBeInTheDocument()
  })

  it('dar de baja manda activo=false sin tocar el resto del producto', async () => {
    montar()
    await screen.findByText(/Remera basica/)

    const botones = screen.getAllByRole('button', { name: 'Dar de baja' })
    await userEvent.click(botones[0])

    await waitFor(() => {
      const put = llamadas.find((l) => l.metodo === 'PUT')
      expect(put).toBeTruthy()
      expect(put.url).toBe('/api/productos/1')
      expect(JSON.parse(put.body)).toMatchObject({
        nombre: 'Remera basica', precio: 12500, activo: false,
      })
    })
  })

  it('el producto dado de baja ofrece darlo de alta', async () => {
    montar()
    await screen.findByText(/Campera vieja/)

    await userEvent.click(screen.getByRole('button', { name: 'Dar de alta' }))

    await waitFor(() => {
      const put = llamadas.find((l) => l.metodo === 'PUT')
      expect(JSON.parse(put.body).activo).toBe(true)
    })
  })

  it('corrige el stock de una variante por su propio endpoint', async () => {
    montar()
    await screen.findByText(/Remera basica/)

    const campo = screen.getByLabelText(/stock de M Negro/i)
    await userEvent.clear(campo)
    await userEvent.type(campo, '7')
    await userEvent.click(screen.getAllByRole('button', { name: 'Guardar' })[0])

    await waitFor(() => {
      const patch = llamadas.find((l) => l.metodo === 'PATCH')
      expect(patch.url).toBe('/api/productos/1/variantes/10')
      expect(JSON.parse(patch.body)).toEqual({ stock: 7 })
    })
  })

  it('no deja guardar un stock vacio ni sin cambios', async () => {
    montar()
    await screen.findByText(/Remera basica/)

    // Sin tocar nada, no hay nada que guardar.
    expect(screen.getAllByRole('button', { name: 'Guardar' })[0]).toBeDisabled()

    const campo = screen.getByLabelText(/stock de M Negro/i)
    await userEvent.clear(campo)
    expect(screen.getAllByRole('button', { name: 'Guardar' })[0]).toBeDisabled()
    expect(llamadas.some((l) => l.metodo === 'PATCH')).toBe(false)
  })
})
