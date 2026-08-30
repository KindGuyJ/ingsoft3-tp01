import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '../services/api'
import { useCarrito } from '../hooks/useCarrito'
import { useAuth } from '../hooks/useAuth'
import { AvisoError, AvisoExito } from '../components/Aviso'
import SelectorCantidad from '../components/SelectorCantidad'
import { calcularTotal, formatearPrecio } from '../utils'

export default function Carrito() {
  const { items, cambiarCantidad, quitar, vaciar } = useCarrito()
  const { autenticado, sesionVencida } = useAuth()
  const navigate = useNavigate()

  const [error, setError] = useState('')
  const [pedido, setPedido] = useState(null)
  const [enviando, setEnviando] = useState(false)

  // El total se recalcula solo en cada render, porque sale de `items`: no hay
  // un total guardado que pueda quedar desincronizado (regla 3 del TP5).
  // OJO: este total es una ESTIMACIÓN para mostrar. El total real lo calcula el
  // backend con su propia configuración de umbral y costo de envío, y es el que
  // queda guardado en el pedido.
  const { subtotal, envio, total } = calcularTotal(items)

  async function confirmar() {
    setError('')
    if (!autenticado) {
      navigate('/login', { state: { volverA: '/carrito' } })
      return
    }

    setEnviando(true)
    try {
      // Se manda solo variante_id y cantidad: el precio lo pone el backend.
      // Si el front mandara el precio, cualquiera podría comprar a $1.
      const creado = await api.checkout(items.map((i) => ({
        variante_id: i.variante_id,
        cantidad: i.cantidad,
      })))
      setPedido(creado)
      vaciar()
    } catch (e) {
      if (sesionVencida(e)) {
        navigate('/login', { state: { volverA: '/carrito' } })
        return
      }
      // Acá caen los rechazos del backend que el front no puede prever: stock
      // que se agotó entre que se armó el carrito y se confirmó (409).
      setError(e.message)
    } finally {
      setEnviando(false)
    }
  }

  if (pedido) {
    return (
      <section>
        <h1>¡Pedido confirmado!</h1>
        <AvisoExito>
          Pedido #{pedido.id} — {pedido.estado} — total {formatearPrecio(pedido.total)}
        </AvisoExito>
        <p>El total y el estado son los que devolvió el backend, no los del carrito.</p>
        <p>
          <Link to="/mis-pedidos" className="primario boton">Ver mis pedidos</Link>{' '}
          <Link to="/">Seguir comprando</Link>
        </p>
      </section>
    )
  }

  if (items.length === 0) {
    return (
      <section>
        <h1>Carrito</h1>
        <p className="aviso">Tu carrito está vacío.</p>
        <p><Link to="/">Ir al catálogo</Link></p>
      </section>
    )
  }

  return (
    <section>
      <h1>Carrito</h1>
      <AvisoError>{error}</AvisoError>

      <table className="tabla">
        <thead>
          <tr>
            <th>Producto</th><th>Precio</th><th>Cantidad</th><th>Subtotal</th><th />
          </tr>
        </thead>
        <tbody>
          {items.map((i) => (
            <tr key={i.variante_id}>
              <td className="celda-producto">
                {i.imagen && <img src={i.imagen} alt="" className="mini" />}
                <span>
                  <Link to={`/productos/${i.producto_id}`}>{i.nombre}</Link>
                  <small>{i.talle} / {i.color}</small>
                </span>
              </td>
              <td>{formatearPrecio(i.precio)}</td>
              <td>
                <SelectorCantidad
                  valor={i.cantidad}
                  max={i.stock}
                  etiqueta={i.nombre}
                  onCambiar={(n) => cambiarCantidad(i.variante_id, n)}
                />
              </td>
              <td>{formatearPrecio(i.precio * i.cantidad)}</td>
              <td>
                <button type="button" className="link" onClick={() => quitar(i.variante_id)}>
                  Quitar
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      <div className="totales">
        <p>Subtotal: <strong>{formatearPrecio(subtotal)}</strong></p>
        <p>Envío: <strong>{envio === 0 ? 'gratis' : formatearPrecio(envio)}</strong></p>
        <p className="total">Total estimado: <strong>{formatearPrecio(total)}</strong></p>
      </div>

      <div className="acciones">
        <button type="button" className="primario" onClick={confirmar} disabled={enviando}>
          {enviando ? 'Confirmando…' : 'Confirmar compra'}
        </button>
        <button type="button" className="link" onClick={vaciar}>Vaciar carrito</button>
      </div>
    </section>
  )
}
