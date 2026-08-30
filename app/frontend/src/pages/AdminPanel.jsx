import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../services/api'
import { useAuth } from '../hooks/useAuth'
import { Cargando, AvisoError, AvisoExito } from '../components/Aviso'
import { formatearPrecio } from '../utils'

// Los mismos talles que valida el backend (dao.TallesValidos). Están repetidos
// acá para poder ofrecer un <select> en vez de un campo libre; el backend los
// valida igual, así que si la lista se desincroniza el alta falla con 400 en
// vez de guardar algo inválido.
const TALLES = ['XS', 'S', 'M', 'L', 'XL', 'XXL']

const VARIANTE_VACIA = { talle: 'M', color: '', stock: 0 }

// Filtros de la lista. Se aplican sobre lo ya traído: el panel pide SIEMPRE
// todo (activos y dados de baja) y elige qué mostrar. Filtrar contra el backend
// sería un pedido más por cada click para una lista de quince productos.
const FILTROS = [
  { clave: 'todos', etiqueta: 'Todos', incluye: () => true },
  { clave: 'activos', etiqueta: 'Activos', incluye: (p) => p.activo },
  { clave: 'baja', etiqueta: 'Dados de baja', incluye: (p) => !p.activo },
]

export default function AdminPanel() {
  const { sesionVencida } = useAuth()
  const navigate = useNavigate()

  const [productos, setProductos] = useState([])
  const [cargando, setCargando] = useState(true)
  const [error, setError] = useState('')
  const [aviso, setAviso] = useState('')
  const [editando, setEditando] = useState(null) // id del producto en edición
  const [filtro, setFiltro] = useState('todos')

  const cargar = useCallback(async () => {
    setCargando(true)
    try {
      // listarProductosAdmin, no listarProductos: el catálogo público filtra
      // los dados de baja, y desde el panel hay que poder verlos para
      // reactivarlos.
      setProductos(await api.listarProductosAdmin())
    } catch (e) {
      setError(e.message)
    } finally {
      setCargando(false)
    }
  }, [])

  useEffect(() => { cargar() }, [cargar])

  // Un solo lugar donde se manejan los errores de las llamadas de admin: un 401
  // manda al login, cualquier otro se muestra tal como lo explicó el backend.
  function manejarError(e) {
    if (sesionVencida(e)) { navigate('/login', { state: { volverA: '/admin' } }); return }
    setError(e.message)
  }

  async function correr(accion, mensajeOk) {
    setError('')
    setAviso('')
    try {
      await accion()
      setAviso(mensajeOk)
      await cargar()
      return true
    } catch (e) {
      manejarError(e)
      return false
    }
  }

  const visibles = productos.filter(
    FILTROS.find((f) => f.clave === filtro).incluye,
  )

  if (cargando) return <Cargando />

  return (
    <section className="admin">
      <h1>Panel de administración</h1>
      <AvisoError>{error}</AvisoError>
      <AvisoExito>{aviso}</AvisoExito>

      <FormularioNuevo onCrear={(datos) => correr(
        () => api.crearProducto(datos),
        `Producto "${datos.nombre}" creado.`,
      )} />

      <div className="fila encabezado-lista">
        <h2>Productos ({visibles.length} de {productos.length})</h2>
        <div className="filtros">
          {FILTROS.map((f) => (
            <button
              key={f.clave}
              type="button"
              className={`chip ${filtro === f.clave ? 'activo' : ''}`}
              onClick={() => setFiltro(f.clave)}
            >
              {f.etiqueta} ({productos.filter(f.incluye).length})
            </button>
          ))}
        </div>
      </div>

      <div className="lista-admin">
        {visibles.length === 0 && (
          <p className="aviso">No hay productos que coincidan con este filtro.</p>
        )}
        {visibles.map((p) => (
          <FilaProducto
            key={p.id}
            producto={p}
            enEdicion={editando === p.id}
            onEditar={() => { setEditando(p.id); setAviso(''); setError('') }}
            onCancelar={() => setEditando(null)}
            onGuardar={async (datos) => {
              const ok = await correr(
                () => api.editarProducto(p.id, datos),
                `Producto #${p.id} actualizado.`,
              )
              if (ok) setEditando(null)
            }}
            onAgregarVariante={(v) => correr(
              () => api.agregarVariante(p.id, v),
              `Variante ${v.talle}/${v.color} agregada a "${p.nombre}".`,
            )}
            onActualizarStock={(varianteID, stock, etiqueta) => correr(
              () => api.actualizarStock(p.id, varianteID, stock),
              `Stock de ${etiqueta} actualizado a ${stock}.`,
            )}
            onCambiarActivo={(activo) => correr(
              // El PUT pide el producto entero (nombre y precio son
              // obligatorios), así que se reenvían los valores actuales y solo
              // cambia `activo`. La baja es lógica: la fila no se borra, deja
              // de listarse en el catálogo. Borrarla de verdad rompería los
              // pedidos que la referencian.
              () => api.editarProducto(p.id, {
                nombre: p.nombre,
                descripcion: p.descripcion,
                precio: p.precio,
                categoria: p.categoria,
                activo,
              }),
              activo
                ? `"${p.nombre}" volvió al catálogo.`
                : `"${p.nombre}" se dio de baja: ya no aparece en el catálogo.`,
            )}
            onSubirImagen={(formData) => correr(
              () => api.subirImagen(p.id, formData),
              `Imagen agregada a "${p.nombre}".`,
            )}
          />
        ))}
      </div>
    </section>
  )
}

// --- Alta de producto -------------------------------------------------------
// El backend exige al menos una variante (binding:"min=1"): no existe un
// producto sin ninguna combinación talle × color, porque el stock vive ahí.

function FormularioNuevo({ onCrear }) {
  const [abierto, setAbierto] = useState(false)
  const [datos, setDatos] = useState({ nombre: '', descripcion: '', precio: '', categoria: '' })
  const [variantes, setVariantes] = useState([{ ...VARIANTE_VACIA }])

  function cambiar(campo) {
    return (e) => setDatos((d) => ({ ...d, [campo]: e.target.value }))
  }

  function cambiarVariante(i, campo, valor) {
    setVariantes((vs) => vs.map((v, j) => (j === i ? { ...v, [campo]: valor } : v)))
  }

  const precio = Number(datos.precio)
  const valido =
    datos.nombre.trim() !== '' &&
    precio > 0 && // el backend valida binding:"gt=0"; el front no ofrece enviarlo mal
    variantes.every((v) => v.color.trim() !== '' && Number(v.stock) >= 0)

  async function enviar(e) {
    e.preventDefault()
    const ok = await onCrear({
      nombre: datos.nombre,
      descripcion: datos.descripcion,
      precio,
      categoria: datos.categoria,
      variantes: variantes.map((v) => ({
        talle: v.talle,
        color: v.color,
        stock: Number(v.stock),
        // Sin SKU: el service lo autogenera como P<id>-<TALLE>-<COLOR>. Dos SKU
        // vacíos chocarían contra el índice único.
      })),
    })
    if (ok) {
      setDatos({ nombre: '', descripcion: '', precio: '', categoria: '' })
      setVariantes([{ ...VARIANTE_VACIA }])
      setAbierto(false)
    }
  }

  if (!abierto) {
    return (
      <button type="button" className="primario" onClick={() => setAbierto(true)}>
        + Nuevo producto
      </button>
    )
  }

  return (
    <form className="tarjeta-admin" onSubmit={enviar} noValidate>
      <h2>Nuevo producto</h2>

      <div className="fila">
        <label>Nombre<input value={datos.nombre} onChange={cambiar('nombre')} /></label>
        <label>Categoría<input value={datos.categoria} onChange={cambiar('categoria')} /></label>
        <label>
          Precio
          <input type="number" min="1" step="1" value={datos.precio} onChange={cambiar('precio')} />
        </label>
      </div>

      <label>
        Descripción
        <textarea rows="2" value={datos.descripcion} onChange={cambiar('descripcion')} />
      </label>

      <h3>Variantes</h3>
      {variantes.map((v, i) => (
        <div className="fila" key={i}>
          <label>
            Talle
            <select value={v.talle} onChange={(e) => cambiarVariante(i, 'talle', e.target.value)}>
              {TALLES.map((t) => <option key={t} value={t}>{t}</option>)}
            </select>
          </label>
          <label>
            Color
            <input value={v.color} onChange={(e) => cambiarVariante(i, 'color', e.target.value)} />
          </label>
          <label>
            Stock
            <input
              type="number" min="0"
              value={v.stock}
              onChange={(e) => cambiarVariante(i, 'stock', e.target.value)}
            />
          </label>
          {variantes.length > 1 && (
            <button
              type="button" className="link"
              onClick={() => setVariantes((vs) => vs.filter((_, j) => j !== i))}
            >
              Quitar
            </button>
          )}
        </div>
      ))}

      <button
        type="button" className="link"
        onClick={() => setVariantes((vs) => [...vs, { ...VARIANTE_VACIA }])}
      >
        + Agregar variante
      </button>

      <div className="acciones">
        <button type="submit" className="primario" disabled={!valido}>Crear producto</button>
        <button type="button" className="link" onClick={() => setAbierto(false)}>Cancelar</button>
      </div>
    </form>
  )
}

// --- Edición y variantes de un producto existente ---------------------------

function FilaProducto({
  producto, enEdicion, onEditar, onCancelar, onGuardar, onAgregarVariante,
  onActualizarStock, onCambiarActivo, onSubirImagen,
}) {
  const [datos, setDatos] = useState({
    nombre: producto.nombre,
    descripcion: producto.descripcion,
    precio: String(producto.precio),
    categoria: producto.categoria,
  })
  const [nueva, setNueva] = useState({ ...VARIANTE_VACIA })

  function cambiar(campo) {
    return (e) => setDatos((d) => ({ ...d, [campo]: e.target.value }))
  }

  return (
    <article className="tarjeta-admin">
      <header className="fila">
        <h3>#{producto.id} — {producto.nombre}</h3>
        <span className="categoria">{producto.categoria}</span>
        <span className="precio">{formatearPrecio(producto.precio)}</span>
        {!producto.activo && <span className="estado cancelado">dado de baja</span>}
        {!enEdicion && (
          <button type="button" className="link" onClick={onEditar}>Editar</button>
        )}
        <button
          type="button"
          className="link"
          onClick={() => onCambiarActivo(!producto.activo)}
        >
          {producto.activo ? 'Dar de baja' : 'Dar de alta'}
        </button>
      </header>

      {enEdicion && (
        <div className="edicion">
          <div className="fila">
            <label>Nombre<input value={datos.nombre} onChange={cambiar('nombre')} /></label>
            <label>Categoría<input value={datos.categoria} onChange={cambiar('categoria')} /></label>
            <label>
              Precio
              <input type="number" min="1" value={datos.precio} onChange={cambiar('precio')} />
            </label>
          </div>
          <label>
            Descripción
            <textarea rows="2" value={datos.descripcion} onChange={cambiar('descripcion')} />
          </label>
          <div className="acciones">
            {/* Editar el producto NO toca variantes ni stock: son endpoints
                distintos, a propósito. Cambiar el precio tampoco altera pedidos
                viejos, porque el precio del pedido es un snapshot (regla 4). */}
            <button
              type="button" className="primario"
              onClick={() => onGuardar({ ...datos, precio: Number(datos.precio) })}
              disabled={datos.nombre.trim() === '' || !(Number(datos.precio) > 0)}
            >
              Guardar
            </button>
            <button type="button" className="link" onClick={onCancelar}>Cancelar</button>
          </div>
        </div>
      )}

      {/* <details> nativo en vez de un useState con un botón: el navegador ya
          sabe abrir y cerrar esto, y con quince productos en pantalla la lista
          es ilegible si todas las tablas de variantes están desplegadas. El
          resumen adelanta lo que importa sin abrir: cuántas variantes hay y
          cuánto stock suman. */}
      <details className="variantes">
        <summary>
          Variantes ({producto.variantes.length}) — stock total{' '}
          <strong>{stockTotal(producto.variantes)}</strong>
          {agotadas(producto.variantes) > 0 && (
            <span className="sin-stock"> · {agotadas(producto.variantes)} sin stock</span>
          )}
        </summary>

        <table className="tabla chica">
          <thead>
            <tr><th>Talle</th><th>Color</th><th>Stock</th><th>SKU</th></tr>
          </thead>
          <tbody>
            {producto.variantes.map((v) => (
              <FilaVariante
                key={v.id}
                variante={v}
                onGuardar={(stock) => onActualizarStock(v.id, stock, `${v.talle}/${v.color}`)}
              />
            ))}
          </tbody>
        </table>
      </details>

      <div className="fila">
        <label>
          Talle
          <select value={nueva.talle} onChange={(e) => setNueva({ ...nueva, talle: e.target.value })}>
            {TALLES.map((t) => <option key={t} value={t}>{t}</option>)}
          </select>
        </label>
        <label>
          Color
          <input value={nueva.color} onChange={(e) => setNueva({ ...nueva, color: e.target.value })} />
        </label>
        <label>
          Stock
          <input
            type="number" min="0" value={nueva.stock}
            onChange={(e) => setNueva({ ...nueva, stock: e.target.value })}
          />
        </label>
        <button
          type="button"
          className="link"
          disabled={nueva.color.trim() === ''}
          onClick={async () => {
            await onAgregarVariante({
              talle: nueva.talle, color: nueva.color, stock: Number(nueva.stock),
            })
            setNueva({ ...VARIANTE_VACIA })
          }}
        >
          + Agregar variante
        </button>
      </div>

      <div className="imagenes-admin">
        <strong>Imágenes ({producto.imagenes.length})</strong>
        <div className="tira">
          {producto.imagenes.map((i) => (
            <img
              key={i.id || i.url}
              src={i.url}
              alt={i.alt_text || producto.nombre}
              className="mini"
            />
          ))}
          {producto.imagenes.length === 0 && <span className="ayuda">sin imágenes</span>}
        </div>
        {/* Conviven dos orígenes en la misma tira: las estáticas del seed
            (/productos/x.jpg, servidas por nginx) y las subidas (/uploads/x.jpg,
            servidas por el backend desde el volumen). El <img> no las distingue
            porque las dos son rutas relativas. */}
        <FormularioImagen producto={producto} onSubir={onSubirImagen} />
      </div>
    </article>
  )
}

// --- Stock de una variante --------------------------------------------------
//
// El stock se corrige con su propio endpoint (PATCH .../variantes/:id) y no
// dentro del formulario de edición del producto: editar el nombre o el precio
// no tiene por qué poder tocar el inventario de rebote.

function FilaVariante({ variante, onGuardar }) {
  // El valor se guarda como TEXTO mientras se escribe. Convertirlo a número en
  // cada tecla haría que borrar el campo lo devuelva a 0 y que no se pueda
  // escribir "12" (quedaría "012"). Se valida al guardar.
  const [valor, setValor] = useState(String(variante.stock))
  const [guardando, setGuardando] = useState(false)

  // Si el backend devolvió otro número (por ejemplo tras recargar la lista), el
  // campo tiene que reflejarlo y no quedarse con lo que se había tipeado.
  useEffect(() => { setValor(String(variante.stock)) }, [variante.stock])

  const n = Number(valor)
  const valido = valor.trim() !== '' && Number.isInteger(n) && n >= 0
  const cambio = valido && n !== variante.stock

  async function guardar() {
    setGuardando(true)
    await onGuardar(n)
    setGuardando(false)
  }

  return (
    <tr>
      <td>{variante.talle}</td>
      <td>{variante.color}</td>
      <td className={variante.stock === 0 ? 'sin-stock' : ''}>
        <input
          type="number"
          min="0"
          className="stock-input"
          value={valor}
          aria-label={`Stock de ${variante.talle} ${variante.color}`}
          onChange={(e) => setValor(e.target.value)}
        />
        <button
          type="button"
          className="link"
          disabled={!cambio || guardando}
          onClick={guardar}
        >
          {guardando ? 'Guardando…' : 'Guardar'}
        </button>
        {!valido && <span className="ayuda error">Entero ≥ 0</span>}
      </td>
      <td><code>{variante.sku}</code></td>
    </tr>
  )
}

function stockTotal(variantes) {
  return variantes.reduce((acc, v) => acc + v.stock, 0)
}

function agotadas(variantes) {
  return variantes.filter((v) => v.stock === 0).length
}

// --- Subida de imágenes -----------------------------------------------------
//
// El backend valida extensión, tamaño y que el contenido sea REALMENTE una
// imagen. Lo de acá (el `accept`, el botón deshabilitado) es comodidad, no
// control: quien mande el POST a mano se choca igual contra el service.

const EXTENSIONES = '.jpg,.jpeg,.png'

function FormularioImagen({ producto, onSubir }) {
  const inputArchivo = useRef(null)
  const [archivo, setArchivo] = useState(null)
  const [color, setColor] = useState('')
  const [altText, setAltText] = useState('')
  // La primera foto de un producto es su tapa: viene marcada por defecto.
  const [principal, setPrincipal] = useState(producto.imagenes.length === 0)
  const [subiendo, setSubiendo] = useState(false)

  async function enviar() {
    if (!archivo) return

    const datos = new FormData()
    datos.append('archivo', archivo)
    datos.append('color', color)
    datos.append('alt_text', altText)
    // orden 0 = principal. Si no lo es, se manda un orden que no puede chocar
    // con la principal; el backend rechaza con 409 una segunda orden 0 del
    // mismo color, y el panel muestra ese mensaje tal cual.
    datos.append('orden', principal ? '0' : String(producto.imagenes.length + 1))

    setSubiendo(true)
    const ok = await onSubir(datos)
    setSubiendo(false)

    if (ok) {
      setArchivo(null)
      setColor('')
      setAltText('')
      // Un <input type="file"> es no controlado: para vaciarlo hay que tocarle
      // el value directamente.
      if (inputArchivo.current) inputArchivo.current.value = ''
    }
  }

  return (
    <div className="fila">
      <label>
        Archivo
        <input
          type="file"
          ref={inputArchivo}
          accept={EXTENSIONES}
          onChange={(e) => setArchivo(e.target.files[0] || null)}
        />
      </label>
      <label>
        Color
        <input
          value={color}
          placeholder="vacío = genérica"
          onChange={(e) => setColor(e.target.value)}
        />
      </label>
      <label>
        Texto alternativo
        <input value={altText} onChange={(e) => setAltText(e.target.value)} />
      </label>
      <label className="check">
        <input
          type="checkbox"
          checked={principal}
          onChange={(e) => setPrincipal(e.target.checked)}
        />
        Principal de este color
      </label>
      <button
        type="button"
        className="link"
        disabled={!archivo || subiendo}
        onClick={enviar}
      >
        {subiendo ? 'Subiendo…' : '+ Subir imagen'}
      </button>
    </div>
  )
}
