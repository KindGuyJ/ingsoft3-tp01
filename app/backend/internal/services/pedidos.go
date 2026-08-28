package services

import (
	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/dao"
	dom "github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/errors"
)

// ---------------------------------------------------------------------------
// Interfaces del repositorio.
//
// Se declaran ACA, en el paquete que las consume, no en repository/. Es el
// idiom de Go y tiene una consecuencia practica importante: el service no
// depende de GORM ni de MySQL, asi que se testea con un fake en memoria y los
// tests del TP5 corren en milisegundos sin levantar la base.
// ---------------------------------------------------------------------------

type VarianteRepo interface {
	BuscarPorID(id uint) (*dao.Variante, error)
	ActualizarStock(id uint, nuevoStock int) error
}

type PedidoRepo interface {
	Crear(p *dao.Pedido) error
	BuscarPorID(id uint) (*dao.Pedido, error)
	ListarPorUsuario(usuarioID uint) ([]dao.Pedido, error)
	ActualizarEstado(id uint, estado string) error
}

// ItemCarrito es lo que llega desde el cliente en el checkout. El carrito no
// vive en la base: se valida entero en este momento.
type ItemCarrito struct {
	VarianteID uint
	Cantidad   int
}

type PedidosService struct {
	variantes         VarianteRepo
	pedidos           PedidoRepo
	umbralEnvioGratis float64
	costoEnvio        float64
}

func NuevoPedidosService(v VarianteRepo, p PedidoRepo, umbral, costoEnvio float64) *PedidosService {
	return &PedidosService{
		variantes:         v,
		pedidos:           p,
		umbralEnvioGratis: umbral,
		costoEnvio:        costoEnvio,
	}
}

// transicionesValidas define que estado puede seguir a cual.
// Es la regla 5: pendiente->pagado->enviado->entregado, y cancelado solo desde
// pendiente o pagado (una vez que salio del deposito ya no se cancela solo).
var transicionesValidas = map[string][]string{
	dao.EstadoPendiente: {dao.EstadoPagado, dao.EstadoCancelado},
	dao.EstadoPagado:    {dao.EstadoEnviado, dao.EstadoCancelado},
	dao.EstadoEnviado:   {dao.EstadoEntregado},
	dao.EstadoEntregado: {},
	dao.EstadoCancelado: {},
}

// Checkout crea un pedido a partir del carrito del cliente.
//
// Reglas que aplica, en orden:
//  1. El carrito no puede estar vacio, ni tener cantidades <= 0.
//  2. Cada variante tiene que existir.
//  3. No se puede pedir mas cantidad que el stock disponible (regla 1).
//  4. El precio se congela como snapshot (regla 4).
//  5. Total = suma de subtotales + envio, gratis desde el umbral (regla 3).
//  6. Se descuenta el stock exactamente por lo comprado (regla 2).
func (s *PedidosService) Checkout(usuarioID uint, carrito []ItemCarrito) (*dao.Pedido, error) {
	if len(carrito) == 0 {
		return nil, dom.Validacion("el carrito esta vacio")
	}

	items := make([]dao.PedidoItem, 0, len(carrito))
	var subtotal float64

	// Primera pasada: validar TODO antes de tocar el stock. Si algo falla a la
	// mitad, no queremos haber descontado la mitad de los items.
	type descuento struct {
		varianteID uint
		nuevoStock int
	}
	descuentos := make([]descuento, 0, len(carrito))

	for _, ic := range carrito {
		if ic.Cantidad <= 0 {
			return nil, dom.Validacion("la cantidad debe ser mayor a cero")
		}

		v, err := s.variantes.BuscarPorID(ic.VarianteID)
		if err != nil {
			return nil, dom.Interno("no se pudo leer la variante", err)
		}
		if v == nil {
			return nil, dom.NoEncontrado("la variante %d no existe", ic.VarianteID)
		}
		if v.Stock < ic.Cantidad {
			return nil, dom.Conflicto(
				"stock insuficiente para la variante %d: hay %d, se pidieron %d",
				v.ID, v.Stock, ic.Cantidad)
		}
		if v.Producto == nil {
			return nil, dom.Interno("la variante no trae su producto cargado", nil)
		}

		item := dao.PedidoItem{
			VarianteID: v.ID,
			Cantidad:   ic.Cantidad,
			// SNAPSHOT: se copia el precio actual. Si manana cambia el precio
			// del producto, este pedido conserva el que se cobro.
			PrecioUnitario:  v.Producto.Precio,
			DescripcionItem: v.Producto.Nombre + " - " + v.Talle + " / " + v.Color,
		}
		items = append(items, item)
		subtotal += item.Subtotal()
		descuentos = append(descuentos, descuento{varianteID: v.ID, nuevoStock: v.Stock - ic.Cantidad})
	}

	total := subtotal + s.calcularEnvio(subtotal)

	pedido := &dao.Pedido{
		UsuarioID: usuarioID,
		Estado:    dao.EstadoPendiente,
		Total:     total,
		Items:     items,
	}
	if err := s.pedidos.Crear(pedido); err != nil {
		return nil, dom.Interno("no se pudo crear el pedido", err)
	}

	// Segunda pasada: recien ahora se descuenta.
	// TODO(TP4+): envolver Crear + ActualizarStock en una transaccion de GORM.
	// Hoy, si falla un descuento a mitad de camino, el pedido queda creado con
	// stock inconsistente. Esta anotado como limitacion en decisiones.md.
	for _, d := range descuentos {
		if err := s.variantes.ActualizarStock(d.varianteID, d.nuevoStock); err != nil {
			return nil, dom.Interno("no se pudo descontar el stock", err)
		}
	}

	return pedido, nil
}

// calcularEnvio implementa la regla 3. El umbral entra por configuracion,
// no hardcodeado: se puede mover por entorno sin recompilar.
//
// Ojo con el borde: justo EN el umbral el envio ya es gratis (>=, no >).
func (s *PedidosService) calcularEnvio(subtotal float64) float64 {
	if subtotal >= s.umbralEnvioGratis {
		return 0
	}
	return s.costoEnvio
}

// ListarDeUsuario devuelve los pedidos de un usuario. Regla 7: nunca los de otro.
func (s *PedidosService) ListarDeUsuario(usuarioID uint) ([]dao.Pedido, error) {
	ps, err := s.pedidos.ListarPorUsuario(usuarioID)
	if err != nil {
		return nil, dom.Interno("no se pudieron listar los pedidos", err)
	}
	return ps, nil
}

// Cancelar implementa las reglas 6 y 7: devuelve el stock, y solo el dueno del
// pedido (o un admin) puede cancelarlo.
func (s *PedidosService) Cancelar(pedidoID, usuarioID uint, esAdmin bool) error {
	p, err := s.pedidos.BuscarPorID(pedidoID)
	if err != nil {
		return dom.Interno("no se pudo leer el pedido", err)
	}
	if p == nil {
		return dom.NoEncontrado("el pedido %d no existe", pedidoID)
	}

	// Regla 7. Se devuelve 404 y no 403 a proposito: si contestaramos "prohibido",
	// le estariamos confirmando a un tercero que ese pedido existe.
	if p.UsuarioID != usuarioID && !esAdmin {
		return dom.NoEncontrado("el pedido %d no existe", pedidoID)
	}

	if !transicionPermitida(p.Estado, dao.EstadoCancelado) {
		return dom.Conflicto("no se puede cancelar un pedido en estado %q", p.Estado)
	}

	// Regla 6: devolver el stock antes de marcar el pedido.
	for _, item := range p.Items {
		v, err := s.variantes.BuscarPorID(item.VarianteID)
		if err != nil {
			return dom.Interno("no se pudo leer la variante", err)
		}
		if v == nil {
			continue // la variante se borro; no hay stock al que volver
		}
		if err := s.variantes.ActualizarStock(v.ID, v.Stock+item.Cantidad); err != nil {
			return dom.Interno("no se pudo devolver el stock", err)
		}
	}

	return s.pedidos.ActualizarEstado(p.ID, dao.EstadoCancelado)
}

// CambiarEstado es la operacion de admin. Regla 5.
func (s *PedidosService) CambiarEstado(pedidoID uint, nuevoEstado string) error {
	p, err := s.pedidos.BuscarPorID(pedidoID)
	if err != nil {
		return dom.Interno("no se pudo leer el pedido", err)
	}
	if p == nil {
		return dom.NoEncontrado("el pedido %d no existe", pedidoID)
	}
	if !transicionPermitida(p.Estado, nuevoEstado) {
		return dom.Conflicto("transicion invalida: %q -> %q", p.Estado, nuevoEstado)
	}
	return s.pedidos.ActualizarEstado(p.ID, nuevoEstado)
}

func transicionPermitida(desde, hasta string) bool {
	for _, e := range transicionesValidas[desde] {
		if e == hasta {
			return true
		}
	}
	return false
}
