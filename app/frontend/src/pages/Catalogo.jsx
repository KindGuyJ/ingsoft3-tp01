import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router'
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
  // Reintentar = sumarle uno a este contador. Está en las dependencias de los dos
  // efectos, así que cambiarlo vuelve a pedir productos y categorías.
  const [intento, setIntento] = useState(0)

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
  }, [categoria, intento])

  // Las categorías salen del catálogo COMPLETO y se piden una sola vez (o al
  // reintentar): si se derivaran de la lista ya filtrada, al elegir "remeras"
  // desaparecerían todas las demás opciones y no habría forma de volver.
  useEffect(() => {
    api.listarProductos()
      .then((ps) => setCategorias([...new Set(ps.map((p) => p.categoria).filter(Boolean))].sort()))
      .catch(() => setCategorias([]))
  }, [intento])

  function filtrar(cat) {
    setParams(cat ? { categoria: cat } : {})
  }

  return (
    <section>
      {/* Portada: solo texto, sin foto, para no depender de una imagen mas. */}
      <div className="portada">
        <p className="portada-titulo">Indumentaria esencial</p>
        <p className="portada-bajada">Básicos de algodón, buzos y camperas para todos los días.</p>
      </div>

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
      {error && (
        <div className="acciones">
          <button type="button" className="primario" onClick={() => setIntento((n) => n + 1)}>
            Reintentar
          </button>
        </div>
      )}
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
