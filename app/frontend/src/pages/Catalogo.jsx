import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { api } from '../services/api'
import ProductoCard from '../components/ProductoCard'
import { Cargando, AvisoError } from '../components/Aviso'

export default function Catalogo() {
  // La categoría vive en la URL (?categoria=remeras) y no en un useState: así
  // el filtro se puede compartir por link y el botón "atrás" del navegador hace
  // lo que se espera.
  const [params, setParams] = useSearchParams()
  const categoria = params.get('categoria') || ''

  const [productos, setProductos] = useState([])
  const [categorias, setCategorias] = useState([])
  const [cargando, setCargando] = useState(true)
  const [error, setError] = useState('')

  // El filtro lo resuelve el backend (GET /api/productos?categoria=), que ya
  // sabe hacer esa consulta. Reimplementarlo en el front sería duplicarlo.
  useEffect(() => {
    let vigente = true
    setCargando(true)
    setError('')

    api.listarProductos(categoria)
      .then((ps) => { if (vigente) setProductos(ps) })
      .catch((e) => { if (vigente) setError(e.message) })
      .finally(() => { if (vigente) setCargando(false) })

    // La respuesta de una petición vieja puede llegar después de una nueva; el
    // flag evita que pise el resultado correcto.
    return () => { vigente = false }
  }, [categoria])

  // Las categorías salen del catálogo COMPLETO y se piden una sola vez: si se
  // derivaran de la lista ya filtrada, al elegir "remeras" desaparecerían todas
  // las demás opciones y no habría forma de volver.
  useEffect(() => {
    api.listarProductos()
      .then((ps) => setCategorias([...new Set(ps.map((p) => p.categoria).filter(Boolean))].sort()))
      .catch(() => setCategorias([]))
  }, [])

  function filtrar(cat) {
    setParams(cat ? { categoria: cat } : {})
  }

  return (
    <section>
      <h1>Catálogo</h1>

      <div className="filtros">
        <button
          type="button"
          className={`chip ${categoria === '' ? 'activo' : ''}`}
          onClick={() => filtrar('')}
        >
          Todas
        </button>
        {categorias.map((c) => (
          <button
            key={c}
            type="button"
            className={`chip ${categoria === c ? 'activo' : ''}`}
            onClick={() => filtrar(c)}
          >
            {c}
          </button>
        ))}
      </div>

      <AvisoError>{error}</AvisoError>
      {cargando && <Cargando />}

      {!cargando && !error && productos.length === 0 && (
        <p className="aviso">No hay productos en esta categoría.</p>
      )}

      <div className="grilla">
        {productos.map((p) => <ProductoCard key={p.id} producto={p} />)}
      </div>
    </section>
  )
}
