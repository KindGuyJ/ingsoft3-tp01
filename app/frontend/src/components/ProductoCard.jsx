import { Link } from 'react-router-dom'
import { formatearPrecio, imagenPrincipal } from '../utils'

export default function ProductoCard({ producto }) {
  const img = imagenPrincipal(producto.imagenes)
  const sinStock = producto.variantes.every((v) => v.stock === 0)

  return (
    <Link to={`/productos/${producto.id}`} className="card">
      <div className="card-foto">
        {img
          ? <img src={img.url} alt={img.alt_text || producto.nombre} loading="lazy" />
          : <span className="sin-foto">sin foto</span>}
        {sinStock && <span className="etiqueta">sin stock</span>}
      </div>
      <div className="card-datos">
        <h3>{producto.nombre}</h3>
        <p className="categoria">{producto.categoria}</p>
        <p className="precio">{formatearPrecio(producto.precio)}</p>
      </div>
    </Link>
  )
}
