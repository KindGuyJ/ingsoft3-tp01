import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '../services/api'
import { useAuth } from '../hooks/useAuth'
import { Cargando, AvisoError, AvisoExito } from '../components/Aviso'
import { formatearPrecio, sePuedeCancelar } from '../utils'

export default function MisPedidos() {
  const { sesionVencida } = useAuth()
  const navigate = useNavigate()

  const [pedidos, setPedidos] = useState([])
  const [cargando, setCargando] = useState(true)
  const [error, setError] = useState('')
  const [aviso, setAviso] = useState('')

  // El backend filtra por el usuario del token (regla 7): esta pantalla NO
  // manda ningún id de usuario, ni podría ver los pedidos de otro si quisiera.
  const cargar = useCallback(async () => {
    setCargando(true)
    setError('')
    try {
      setPedidos(await api.misPedidos())
    } catch (e) {
      if (sesionVencida(e)) { navigate('/login', { state: { volverA: '/mis-pedidos' } }); return }
      setError(e.message)
    } finally {
      setCargando(false)
    }
  }, [navigate, sesionVencida])

  useEffect(() => { cargar() }, [cargar])

  async function cancelar(id) {
    setError('')
    setAviso('')
    try {
      await api.cancelarPedido(id)
      setAviso(`Pedido #${id} cancelado. El stock volvió a las variantes.`)
      // Se recarga desde el backend en vez de tocar el estado local: el estado
      // que vale es el que quedó guardado, no el que el front supone.
      await cargar()
    } catch (e) {
      if (sesionVencida(e)) { navigate('/login', { state: { volverA: '/mis-pedidos' } }); return }
      setError(e.message)
    }
  }

  if (cargando) return <Cargando />

  return (
    <section>
      <h1>Mis pedidos</h1>
      <AvisoError>{error}</AvisoError>
      <AvisoExito>{aviso}</AvisoExito>

      {pedidos.length === 0 && (
        <p className="aviso">Todavía no hiciste ningún pedido. <Link to="/">Ir al catálogo</Link></p>
      )}

      {pedidos.map((p) => (
        <article key={p.id} className="pedido">
          <header>
            <h2>Pedido #{p.id}</h2>
            <span className={`estado ${p.estado}`}>{p.estado}</span>
            <span className="fecha">{new Date(p.fecha).toLocaleString('es-AR')}</span>
          </header>

          <ul className="items">
            {p.items.map((i) => (
              <li key={i.variante_id}>
                {i.descripcion} × {i.cantidad} — {formatearPrecio(i.precio_unitario)} c/u
                {' '}= <strong>{formatearPrecio(i.subtotal)}</strong>
              </li>
            ))}
          </ul>

          <footer>
            <span className="total">Total: {formatearPrecio(p.total)}</span>
            {/* Solo se ofrece cancelar donde la transición es válida
                (pendiente o pagado). Si igual se intentara, el backend
                responde 409 y el mensaje se muestra arriba. */}
            {sePuedeCancelar(p.estado) && (
              <button type="button" className="link" onClick={() => cancelar(p.id)}>
                Cancelar pedido
              </button>
            )}
          </footer>
        </article>
      ))}
    </section>
  )
}
