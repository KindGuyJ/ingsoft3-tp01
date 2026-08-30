import { createContext, useContext, useEffect, useMemo, useState } from 'react'

// EL CARRITO NO ESTÁ EN LA BASE. Vive en el cliente y se valida entero contra
// el backend recién en el checkout (ver CLAUDE.md). Por eso guarda también el
// precio y el nombre: son una copia para dibujar la pantalla, no la verdad.
// La verdad la dice el backend cuando confirma el pedido.
const CarritoContext = createContext(null)
const CLAVE = 'carrito'

function leerGuardado() {
  try {
    const raw = localStorage.getItem(CLAVE)
    const items = raw ? JSON.parse(raw) : []
    return Array.isArray(items) ? items : []
  } catch {
    return []
  }
}

export function CarritoProvider({ children }) {
  const [items, setItems] = useState(leerGuardado)

  // Persistir en cada cambio: así el carrito sobrevive a un F5 o a que la
  // usuaria se vaya a loguear y vuelva.
  useEffect(() => {
    localStorage.setItem(CLAVE, JSON.stringify(items))
  }, [items])

  const valor = useMemo(() => ({
    items,

    // La clave del item es la VARIANTE, no el producto: la misma remera en M y
    // en L son dos líneas distintas del pedido.
    agregar(item) {
      setItems((prev) => {
        const i = prev.findIndex((x) => x.variante_id === item.variante_id)
        if (i === -1) return [...prev, item]
        const copia = [...prev]
        const sumada = copia[i].cantidad + item.cantidad
        copia[i] = { ...copia[i], cantidad: Math.min(sumada, item.stock) }
        return copia
      })
    },

    cambiarCantidad(varianteID, cantidad) {
      setItems((prev) => prev.map((x) => (
        x.variante_id === varianteID
          ? { ...x, cantidad: Math.max(1, Math.min(cantidad, x.stock)) }
          : x
      )))
    },

    quitar(varianteID) {
      setItems((prev) => prev.filter((x) => x.variante_id !== varianteID))
    },

    vaciar() { setItems([]) },

    unidades: items.reduce((acc, i) => acc + i.cantidad, 0),
  }), [items])

  return <CarritoContext.Provider value={valor}>{children}</CarritoContext.Provider>
}

export function useCarrito() {
  const ctx = useContext(CarritoContext)
  if (!ctx) throw new Error('useCarrito se usó fuera de <CarritoProvider>')
  return ctx
}
