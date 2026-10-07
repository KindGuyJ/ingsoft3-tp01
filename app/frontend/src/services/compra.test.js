import { describe, it, expect, vi } from 'vitest'
import { confirmarCompra } from './compra'

// Sin DOM y sin red: el backend lo reemplaza un doble hecho con vi.fn().
// En el primer test el doble es un MOCK (el assert mira cómo se lo llamó);
// en el último es un STUB (solo contesta, y el assert mira el resultado).

const REMERA = { variante_id: 11, nombre: 'Remera basica', talle: 'M', precio: 10000, cantidad: 2, stock: 5 }
const BUZO = { variante_id: 20, nombre: 'Buzo canguro', talle: 'L', precio: 30000, cantidad: 1, stock: 4 }

describe('confirmarCompra', () => {
  it('le manda al backend solo variante y cantidad, nunca el precio', async () => {
    // Arrange
    const checkout = vi.fn().mockResolvedValue({ id: 1 })

    // Act
    await confirmarCompra([REMERA, BUZO], checkout)

    // Assert: la interacción, no un valor devuelto
    expect(checkout).toHaveBeenCalledTimes(1)
    expect(checkout).toHaveBeenCalledWith([
      { variante_id: 11, cantidad: 2 },
      { variante_id: 20, cantidad: 1 },
    ])
  })

  it('un carrito vacío se rechaza sin llamar al backend', async () => {
    const checkout = vi.fn()

    await expect(confirmarCompra([], checkout)).rejects.toThrow('vacío')
    expect(checkout).not.toHaveBeenCalled()
  })

  it('devuelve el pedido tal como lo creó el backend', async () => {
    const creado = { id: 7, estado: 'pendiente', total: 55000 }
    const checkout = vi.fn().mockResolvedValue(creado)

    const pedido = await confirmarCompra([REMERA], checkout)

    expect(pedido).toEqual(creado)
  })

  it('si el backend rechaza (stock agotado), el error llega con su status', async () => {
    const rechazo = Object.assign(new Error('stock insuficiente'), { status: 409 })
    const checkout = vi.fn().mockRejectedValue(rechazo)

    await expect(confirmarCompra([REMERA], checkout)).rejects.toMatchObject({ status: 409 })
  })
})
