// Lógica pura del frontend. Está separada de los componentes a propósito:
// se testea sin montar React ni simular clics (los 4 tests del TP5).

export function calcularSubtotal(items) {
  return items.reduce((acc, i) => acc + i.precio * i.cantidad, 0)
}

export function calcularTotal(items, { umbralEnvioGratis = 50000, costoEnvio = 5000 } = {}) {
  const subtotal = calcularSubtotal(items)
  // Ojo con el borde: JUSTO en el umbral ya es gratis (>=, no >).
  const envio = subtotal >= umbralEnvioGratis ? 0 : costoEnvio
  return { subtotal, envio, total: subtotal + envio }
}

export function emailValido(email) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)
}

// El botón "Agregar al carrito" se habilita solo si hay talle Y color elegidos
// y esa combinación tiene stock.
export function puedeAgregarAlCarrito({ talle, color, variantes }) {
  if (!talle || !color) return false
  const v = variantes.find((x) => x.talle === talle && x.color === color)
  return !!v && v.stock > 0
}

export function tallesDisponibles(variantes, color) {
  return variantes
    .filter((v) => !color || v.color === color)
    .map((v) => ({ talle: v.talle, disponible: v.stock > 0 }))
}

// --- Helpers de presentación -----------------------------------------------
// No son reglas de negocio: son derivaciones del catálogo que usan varias
// pantallas. Viven acá y no dentro de un componente para no repetirlas.

export function coloresDisponibles(variantes) {
  return [...new Set(variantes.map((v) => v.color))]
}

// Imagen a mostrar: la del color elegido con orden más bajo; si ese color no
// tiene foto propia, la genérica del producto (Color vacío). Devuelve null si
// el producto no tiene ninguna, y la vista decide qué poner en su lugar.
export function imagenPrincipal(imagenes = [], color = '') {
  const candidatas = imagenes.filter((i) => !color || !i.color || i.color === color)
  if (candidatas.length === 0) return null
  const exactas = candidatas.filter((i) => i.color === color)
  const elegidas = exactas.length > 0 ? exactas : candidatas
  return [...elegidas].sort((a, b) => a.orden - b.orden)[0]
}

// El backend devuelve las variantes en el orden en que las lee la base, que no
// es el orden de la talla. Se ordenan acá para que los talles se vean XS, S, M,
// L… y no salteados: es presentación, no una regla.
export const ORDEN_TALLES = ['XS', 'S', 'M', 'L', 'XL', 'XXL']

export function ordenarTalles(items) {
  const pos = (t) => {
    const i = ORDEN_TALLES.indexOf(t)
    return i === -1 ? ORDEN_TALLES.length : i
  }
  return [...items].sort((a, b) => pos(a.talle) - pos(b.talle))
}

export function stockDe(variantes, talle, color) {
  const v = variantes.find((x) => x.talle === talle && x.color === color)
  return v ? v.stock : 0
}

export function varianteDe(variantes, talle, color) {
  return variantes.find((x) => x.talle === talle && x.color === color) || null
}

export function formatearPrecio(n) {
  return new Intl.NumberFormat('es-AR', {
    style: 'currency', currency: 'ARS', maximumFractionDigits: 0,
  }).format(n)
}

// Un pedido solo se puede cancelar desde pendiente o pagado (regla 5 del
// backend). El front lo repite para no ofrecer un botón que va a fallar; el
// backend sigue siendo el que decide.
export function sePuedeCancelar(estado) {
  return estado === 'pendiente' || estado === 'pagado'
}
