import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import '@testing-library/jest-dom'
import { MemoryRouter } from 'react-router'

import Catalogo from './Catalogo'
import { CarritoProvider } from '../hooks/useCarrito'

// Issue #11: con el backend todavía arrancando contesta nginx (502) o Render
// (429). El catálogo tiene que avisar (no quedar vacío) y poder recuperarse sin
// recargar la página.

const PRODUCTOS = [
  { id: 1, nombre: 'Remera basica', precio: 12500, categoria: 'remeras', imagenes: [], variantes: [] },
]

const vivo = { ok: true, status: 200, json: () => Promise.resolve(PRODUCTOS) }

describe('Catalogo', () => {
  it.each([429, 502])('con %i avisa que el backend no responde y se recupera con Reintentar', async (status) => {
    const caido = { ok: false, status, json: () => Promise.reject(new Error('no es JSON')) }
    // Las dos primeras llamadas (productos y categorías) fallan; las siguientes andan.
    globalThis.fetch = vi.fn()
      .mockResolvedValueOnce(caido)
      .mockResolvedValueOnce(caido)
      .mockResolvedValue(vivo)

    render(
      <MemoryRouter>
        <CarritoProvider>
          <Catalogo />
        </CarritoProvider>
      </MemoryRouter>,
    )

    expect(await screen.findByRole('alert')).toHaveTextContent(/todavía no responde/i)
    expect(screen.queryByText(/no hay productos/i)).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: /reintentar/i }))

    expect(await screen.findByText('Remera basica')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'remeras' })).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})
