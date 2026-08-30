import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import '@testing-library/jest-dom'
import { MemoryRouter } from 'react-router-dom'

import Carrito from './Carrito'
import { CarritoProvider } from '../hooks/useCarrito'
import { AuthProvider } from '../hooks/useAuth'

// Verifica el comportamiento 3 del TP5 sobre la pantalla: al cambiar la
// cantidad, el total se recalcula. El cálculo en sí lo testea utils.test.js.

const ITEM = {
  variante_id: 11, producto_id: 1, nombre: 'Remera basica',
  talle: 'M', color: 'Negro', precio: 10000, cantidad: 1, stock: 5, imagen: '',
}

function montar() {
  return render(
    <MemoryRouter>
      <AuthProvider>
        <CarritoProvider>
          <Carrito />
        </CarritoProvider>
      </AuthProvider>
    </MemoryRouter>,
  )
}

// El total incluye separadores de miles; se compara sobre los dígitos para no
// atarse al formato exacto de Intl, que cambia entre versiones de Node.
function totalMostrado() {
  return screen.getByText(/Total estimado/i).textContent.replace(/\D/g, '')
}

beforeEach(() => {
  localStorage.clear()
  localStorage.setItem('carrito', JSON.stringify([ITEM]))
  globalThis.fetch = vi.fn()
})

describe('Carrito', () => {
  it('recalcula el total al cambiar la cantidad', async () => {
    montar()
    // 1 x 10000 = 10000 + 5000 de envio
    expect(totalMostrado()).toBe('15000')

    const mas = screen.getByRole('button', { name: /agregar una unidad/i })
    await userEvent.click(mas)
    await userEvent.click(mas)

    // 3 x 10000 = 30000 + 5000 de envio
    expect(screen.getByLabelText(/cantidad de remera basica/i)).toHaveTextContent('3')
    expect(totalMostrado()).toBe('35000')
  })

  it('el envio pasa a gratis al cruzar el umbral', async () => {
    montar()
    const mas = screen.getByRole('button', { name: /agregar una unidad/i })
    for (let i = 0; i < 4; i += 1) await userEvent.click(mas) // 5 x 10000 = 50000

    expect(screen.getByText(/envío/i)).toHaveTextContent(/gratis/i)
    expect(totalMostrado()).toBe('50000')
  })

  it('no deja pasar de la cantidad del stock', async () => {
    montar()
    const mas = screen.getByRole('button', { name: /agregar una unidad/i })
    for (let i = 0; i < 10; i += 1) await userEvent.click(mas)

    expect(screen.getByLabelText(/cantidad de remera basica/i)).toHaveTextContent('5')
    expect(mas).toBeDisabled() // el stock del item
  })

  it('el carrito vacio no ofrece confirmar la compra', () => {
    localStorage.setItem('carrito', '[]')
    montar()
    expect(screen.getByText(/tu carrito está vacío/i)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /confirmar compra/i })).toBeNull()
  })
})
