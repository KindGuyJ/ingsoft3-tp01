// Confirmar la compra: arma el pedido a partir del carrito y se lo manda al
// backend.
//
// La función que habla con el backend ENTRA POR PARÁMETRO (`checkout`): la
// pantalla le pasa api.checkout, y el test le pasa un doble hecho con vi.fn().
// Antes esto vivía adentro de Carrito.jsx llamando a api.checkout directo, y la
// única forma de testearlo era montar la pantalla y pisar el fetch global.
export async function confirmarCompra(items, checkout) {
  // La pantalla no ofrece confirmar un carrito vacío, pero la regla no puede
  // depender de un botón escondido: si llega uno vacío, ni se llama al backend.
  if (items.length === 0) {
    throw new Error('El carrito está vacío.')
  }

  // Se manda solo variante_id y cantidad: el precio lo pone el backend.
  // Si el front mandara el precio, cualquiera podría comprar a $1.
  return checkout(items.map((i) => ({
    variante_id: i.variante_id,
    cantidad: i.cantidad,
  })))
}
