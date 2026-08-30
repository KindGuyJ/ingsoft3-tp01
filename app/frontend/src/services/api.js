// Cliente HTTP. Todas las URLs son RELATIVAS: nunca aparece un host ni un
// puerto en el código del front. Ver vite.config.js y nginx.conf.

const TOKEN_KEY = 'token'
const USUARIO_KEY = 'usuario'

export function guardarToken(t) { localStorage.setItem(TOKEN_KEY, t) }
export function leerToken() { return localStorage.getItem(TOKEN_KEY) }
export function borrarToken() { localStorage.removeItem(TOKEN_KEY) }

// El usuario se guarda al lado del token SOLO para dibujar la interfaz (saludo,
// mostrar u ocultar el panel de admin). No es un control de acceso: quien edite
// localStorage y se ponga es_admin ve el menú, pero el backend le devuelve 403
// igual, porque el rol real viaja firmado adentro del JWT.
export function guardarUsuario(u) { localStorage.setItem(USUARIO_KEY, JSON.stringify(u)) }
export function borrarUsuario() { localStorage.removeItem(USUARIO_KEY) }
export function leerUsuario() {
  try {
    const raw = localStorage.getItem(USUARIO_KEY)
    return raw ? JSON.parse(raw) : null
  } catch {
    // localStorage con basura adentro no debería tirar abajo la app.
    return null
  }
}

async function request(path, { method = 'GET', body, auth = false } = {}) {
  const headers = { 'Content-Type': 'application/json' }
  if (auth) {
    const t = leerToken()
    if (t) headers['Authorization'] = `Bearer ${t}`
  }

  const res = await fetch(`/api${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  })

  if (!res.ok) {
    const data = await res.json().catch(() => ({}))
    const err = new Error(data.error || `Error ${res.status}`)
    // El status viaja en el error para que las pantallas puedan distinguir un
    // 401 (token vencido → hay que relogearse) de un 409 (regla de negocio).
    err.status = res.status
    throw err
  }
  return res.status === 204 ? null : res.json()
}

// Subida de archivos. Es una función aparte y no una opción de request() por un
// motivo concreto: acá NO hay que setear Content-Type a mano. El browser tiene
// que ponerlo él, porque necesita agregarle el `boundary` que separa las partes
// del multipart. Si se lo pisamos con 'multipart/form-data' a secas, el backend
// no puede parsear el body y devuelve 400.
async function requestForm(path, formData) {
  const headers = {}
  const t = leerToken()
  if (t) headers['Authorization'] = `Bearer ${t}`

  const res = await fetch(`/api${path}`, { method: 'POST', headers, body: formData })

  if (!res.ok) {
    const data = await res.json().catch(() => ({}))
    // Un 413 lo corta nginx antes del backend, así que no trae JSON: sin este
    // caso el usuario vería un "Error 413" pelado.
    const mensaje = res.status === 413
      ? 'La imagen es demasiado grande.'
      : data.error || `Error ${res.status}`
    const err = new Error(mensaje)
    err.status = res.status
    throw err
  }
  return res.json()
}

export const api = {
  listarProductos: (categoria) =>
    request(`/productos${categoria ? `?categoria=${encodeURIComponent(categoria)}` : ''}`),
  verProducto: (id) => request(`/productos/${id}`),
  registro: (datos) => request('/usuarios/registro', { method: 'POST', body: datos }),
  login: (datos) => request('/usuarios/login', { method: 'POST', body: datos }),
  checkout: (items) => request('/pedidos', { method: 'POST', body: { items }, auth: true }),
  misPedidos: () => request('/pedidos', { auth: true }),
  cancelarPedido: (id) => request(`/pedidos/${id}/cancelar`, { method: 'POST', auth: true }),

  // --- Admin: el backend los protege con RequiereAuth + RequiereAdmin -------
  // Listado del panel: incluye los productos dados de baja, que el catálogo
  // público no devuelve. Por eso es otra ruta y no un parámetro de la pública.
  listarProductosAdmin: (categoria) =>
    request(`/admin/productos${categoria ? `?categoria=${encodeURIComponent(categoria)}` : ''}`,
      { auth: true }),
  crearProducto: (datos) => request('/productos', { method: 'POST', body: datos, auth: true }),
  editarProducto: (id, datos) => request(`/productos/${id}`, { method: 'PUT', body: datos, auth: true }),
  agregarVariante: (id, datos) =>
    request(`/productos/${id}/variantes`, { method: 'POST', body: datos, auth: true }),
  subirImagen: (id, formData) => requestForm(`/productos/${id}/imagenes`, formData),
  // Corrección manual del stock de una variante. Es PATCH y no PUT porque toca
  // un solo campo, y va por su propia ruta porque editar el producto NO tiene
  // que poder cambiar el inventario de rebote.
  actualizarStock: (productoID, varianteID, stock) =>
    request(`/productos/${productoID}/variantes/${varianteID}`, {
      method: 'PATCH', body: { stock }, auth: true,
    }),
  cambiarEstadoPedido: (id, estado) =>
    request(`/pedidos/${id}/estado`, { method: 'PATCH', body: { estado }, auth: true }),
}
