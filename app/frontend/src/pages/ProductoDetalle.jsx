import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../services/api'
import { useCarrito } from '../hooks/useCarrito'
import { Cargando, AvisoError, AvisoExito } from '../components/Aviso'
import SelectorCantidad from '../components/SelectorCantidad'
import {
  coloresDisponibles, tallesDisponibles, ordenarTalles, puedeAgregarAlCarrito,
  varianteDe, imagenPrincipal, formatearPrecio,
} from '../utils'

export default function ProductoDetalle() {
  const { id } = useParams()
  const { agregar } = useCarrito()

  const [producto, setProducto] = useState(null)
  const [cargando, setCargando] = useState(true)
  const [error, setError] = useState('')

  const [color, setColor] = useState('')
  const [talle, setTalle] = useState('')
  const [cantidad, setCantidad] = useState(1)
  const [agregado, setAgregado] = useState('')

  useEffect(() => {
    let vigente = true
    setCargando(true)
    api.verProducto(id)
      .then((p) => {
        if (!vigente) return
        setProducto(p)
        // Se preselecciona el primer color para que la pantalla no arranque
        // con todos los talles deshabilitados y parezca rota.
        setColor(coloresDisponibles(p.variantes)[0] || '')
      })
      .catch((e) => { if (vigente) setError(e.message) })
      .finally(() => { if (vigente) setCargando(false) })
    return () => { vigente = false }
  }, [id])

  const colores = useMemo(
    () => (producto ? coloresDisponibles(producto.variantes) : []),
    [producto],
  )

  // Talles del color elegido, cada uno con su bandera de disponibilidad. Los
  // sin stock se muestran DESHABILITADOS, no ocultos: que el talle exista pero
  // esté agotado es información para quien compra.
  const talles = useMemo(
    () => (producto ? ordenarTalles(tallesDisponibles(producto.variantes, color)) : []),
    [producto, color],
  )

  const variante = producto ? varianteDe(producto.variantes, talle, color) : null
  const habilitado = producto
    ? puedeAgregarAlCarrito({ talle, color, variantes: producto.variantes })
    : false

  function elegirColor(c) {
    setColor(c)
    // El talle elegido puede no existir en el color nuevo: se limpia en vez de
    // quedar apuntando a una combinación inexistente.
    setTalle('')
    setCantidad(1)
    setAgregado('')
  }

  function alAgregar() {
    const img = imagenPrincipal(producto.imagenes, color)
    agregar({
      variante_id: variante.id,
      producto_id: producto.id,
      nombre: producto.nombre,
      talle,
      color,
      precio: producto.precio,
      cantidad,
      stock: variante.stock,
      imagen: img ? img.url : '',
    })
    setAgregado(`${producto.nombre} (${talle} / ${color}) × ${cantidad}`)
  }

  if (cargando) return <Cargando />
  if (error) return <AvisoError>{error}</AvisoError>
  if (!producto) return null

  const foto = imagenPrincipal(producto.imagenes, color)

  return (
    <section className="detalle">
      <div className="detalle-foto">
        {foto
          ? <img src={foto.url} alt={foto.alt_text || producto.nombre} />
          : <span className="sin-foto">sin foto</span>}
      </div>

      <div className="detalle-datos">
        <p className="categoria">{producto.categoria}</p>
        <h1>{producto.nombre}</h1>
        <p className="precio grande">{formatearPrecio(producto.precio)}</p>
        <p className="descripcion">{producto.descripcion}</p>

        <fieldset className="opciones">
          <legend>Color</legend>
          {colores.map((c) => (
            <button
              key={c}
              type="button"
              className={`opcion ${color === c ? 'activo' : ''}`}
              onClick={() => elegirColor(c)}
            >
              {c}
            </button>
          ))}
        </fieldset>

        <fieldset className="opciones">
          <legend>Talle</legend>
          {talles.map(({ talle: t, disponible }) => (
            <button
              key={t}
              type="button"
              disabled={!disponible}
              title={disponible ? '' : 'Sin stock en este color'}
              className={`opcion ${talle === t ? 'activo' : ''}`}
              onClick={() => { setTalle(t); setCantidad(1); setAgregado('') }}
            >
              {t}
            </button>
          ))}
        </fieldset>

        <div className="cantidad">
          <span className="etiqueta-campo">Cantidad</span>
          <SelectorCantidad
            valor={cantidad}
            max={variante ? variante.stock : 1}
            etiqueta={producto.nombre}
            onCambiar={setCantidad}
          />
          {variante && <span className="stock">{variante.stock} disponibles</span>}
        </div>

        {/* Regla 1 del TP5: el botón está deshabilitado hasta que haya talle Y
            color elegidos, y esa combinación tenga stock. La condición vive en
            utils.js (puedeAgregarAlCarrito), que es lo que se testea. */}
        <button
          type="button"
          className="primario"
          disabled={!habilitado}
          onClick={alAgregar}
        >
          Agregar al carrito
        </button>

        {!habilitado && (
          <p className="ayuda">Elegí talle y color para agregar al carrito.</p>
        )}

        <AvisoExito>
          {agregado && <>Agregado: {agregado}. <Link to="/carrito">Ir al carrito</Link></>}
        </AvisoExito>

        <p className="volver"><Link to="/">← Volver al catálogo</Link></p>
      </div>
    </section>
  )
}
