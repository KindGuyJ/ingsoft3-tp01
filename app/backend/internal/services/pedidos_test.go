package services

import (
	"testing"

	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/dao"
	dom "github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/errors"
)

// ---------------------------------------------------------------------------
// Fakes en memoria.
//
// No hay MySQL, no hay Docker, no hay red: el service solo conoce interfaces,
// asi que se lo puede alimentar con esto. Por eso estos tests corren en
// milisegundos y sirven como gate del pipeline (TP4).
// ---------------------------------------------------------------------------

type fakeVarianteRepo struct {
	data map[uint]*dao.Variante
}

func (f *fakeVarianteRepo) BuscarPorID(id uint) (*dao.Variante, error) {
	v, ok := f.data[id]
	if !ok {
		return nil, nil
	}
	copia := *v
	if v.Producto != nil {
		p := *v.Producto
		copia.Producto = &p
	}
	return &copia, nil
}

func (f *fakeVarianteRepo) ActualizarStock(id uint, nuevoStock int) error {
	if v, ok := f.data[id]; ok {
		v.Stock = nuevoStock
	}
	return nil
}

type fakePedidoRepo struct {
	data    map[uint]*dao.Pedido
	proximo uint
	creados []*dao.Pedido
}

func nuevoFakePedidoRepo() *fakePedidoRepo {
	return &fakePedidoRepo{data: map[uint]*dao.Pedido{}, proximo: 1}
}

func (f *fakePedidoRepo) Crear(p *dao.Pedido) error {
	p.ID = f.proximo
	f.proximo++
	f.data[p.ID] = p
	f.creados = append(f.creados, p)
	return nil
}

func (f *fakePedidoRepo) BuscarPorID(id uint) (*dao.Pedido, error) {
	p, ok := f.data[id]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (f *fakePedidoRepo) ListarPorUsuario(usuarioID uint) ([]dao.Pedido, error) {
	var out []dao.Pedido
	for _, p := range f.data {
		if p.UsuarioID == usuarioID {
			out = append(out, *p)
		}
	}
	return out, nil
}

func (f *fakePedidoRepo) ActualizarEstado(id uint, estado string) error {
	if p, ok := f.data[id]; ok {
		p.Estado = estado
	}
	return nil
}

// helper: arma un service con una remera negra M, precio 10000, stock 5.
func setup() (*PedidosService, *fakeVarianteRepo, *fakePedidoRepo) {
	prod := &dao.Producto{ID: 1, Nombre: "Remera basica", Precio: 10000}
	vr := &fakeVarianteRepo{data: map[uint]*dao.Variante{
		1: {ID: 1, ProductoID: 1, Talle: "M", Color: "Negro", Stock: 5, Producto: prod},
	}}
	pr := nuevoFakePedidoRepo()
	// umbral de envio gratis 50000, costo de envio 5000
	return NuevoPedidosService(vr, pr, 50000, 5000), vr, pr
}

func kindDe(t *testing.T, err error) dom.Kind {
	t.Helper()
	de, ok := err.(*dom.DomainError)
	if !ok {
		t.Fatalf("se esperaba un DomainError, llego %T: %v", err, err)
	}
	return de.Kind
}

// --- Regla 1: no se puede comprar mas que el stock ---------------------------

func TestCheckout_StockInsuficiente(t *testing.T) {
	s, vr, _ := setup()

	_, err := s.Checkout(7, []ItemCarrito{{VarianteID: 1, Cantidad: 6}})
	if err == nil {
		t.Fatal("se esperaba error por stock insuficiente")
	}
	if k := kindDe(t, err); k != dom.KindConflicto {
		t.Errorf("kind = %v, se esperaba KindConflicto", k)
	}
	// Y el stock no se tiene que haber tocado.
	if vr.data[1].Stock != 5 {
		t.Errorf("el stock cambio a %d; un checkout fallido no debe descontar", vr.data[1].Stock)
	}
}

// --- Regla 2: el checkout descuenta el stock --------------------------------

func TestCheckout_DescuentaStock(t *testing.T) {
	s, vr, _ := setup()

	if _, err := s.Checkout(7, []ItemCarrito{{VarianteID: 1, Cantidad: 2}}); err != nil {
		t.Fatalf("checkout fallo: %v", err)
	}
	if got := vr.data[1].Stock; got != 3 {
		t.Errorf("stock = %d, se esperaba 3", got)
	}
}

// --- Regla 3: total y umbral de envio gratis --------------------------------

func TestCheckout_TotalYEnvio(t *testing.T) {
	casos := []struct {
		nombre   string
		cantidad int
		total    float64 // precio unitario 10000
	}{
		{"debajo del umbral paga envio", 2, 20000 + 5000},
		{"justo EN el umbral es gratis", 5, 50000},
		{"encima del umbral es gratis", 5, 50000},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			s, _, _ := setup()
			p, err := s.Checkout(7, []ItemCarrito{{VarianteID: 1, Cantidad: c.cantidad}})
			if err != nil {
				t.Fatalf("checkout fallo: %v", err)
			}
			if p.Total != c.total {
				t.Errorf("total = %.2f, se esperaba %.2f", p.Total, c.total)
			}
		})
	}
}

// --- Regla 4: el precio es un snapshot --------------------------------------

func TestCheckout_PrecioEsSnapshot(t *testing.T) {
	s, vr, _ := setup()

	p, err := s.Checkout(7, []ItemCarrito{{VarianteID: 1, Cantidad: 1}})
	if err != nil {
		t.Fatalf("checkout fallo: %v", err)
	}

	// El admin remarca el producto DESPUES de la compra.
	vr.data[1].Producto.Precio = 99000

	if p.Items[0].PrecioUnitario != 10000 {
		t.Errorf("precio del item = %.2f; el pedido no debe seguir el precio nuevo",
			p.Items[0].PrecioUnitario)
	}
}

// --- Regla 5: transiciones de estado ----------------------------------------

func TestCambiarEstado_Transiciones(t *testing.T) {
	casos := []struct {
		desde, hasta string
		valida       bool
	}{
		{dao.EstadoPendiente, dao.EstadoPagado, true},
		{dao.EstadoPagado, dao.EstadoEnviado, true},
		{dao.EstadoEnviado, dao.EstadoEntregado, true},
		{dao.EstadoPendiente, dao.EstadoEntregado, false}, // no se saltea la cadena
		{dao.EstadoEntregado, dao.EstadoPendiente, false}, // no se vuelve atras
		{dao.EstadoCancelado, dao.EstadoPagado, false},    // un cancelado no revive
	}

	for _, c := range casos {
		t.Run(c.desde+"_a_"+c.hasta, func(t *testing.T) {
			s, _, pr := setup()
			pr.data[1] = &dao.Pedido{ID: 1, UsuarioID: 7, Estado: c.desde}

			err := s.CambiarEstado(1, c.hasta)
			if c.valida && err != nil {
				t.Errorf("se esperaba transicion valida, llego error: %v", err)
			}
			if !c.valida && err == nil {
				t.Errorf("se esperaba error para %q -> %q", c.desde, c.hasta)
			}
		})
	}
}

// --- Regla 6: cancelar devuelve el stock ------------------------------------

func TestCancelar_DevuelveStock(t *testing.T) {
	s, vr, _ := setup()

	p, err := s.Checkout(7, []ItemCarrito{{VarianteID: 1, Cantidad: 2}})
	if err != nil {
		t.Fatalf("checkout fallo: %v", err)
	}
	if vr.data[1].Stock != 3 {
		t.Fatalf("precondicion: stock = %d, se esperaba 3", vr.data[1].Stock)
	}

	if err := s.Cancelar(p.ID, 7, false); err != nil {
		t.Fatalf("cancelar fallo: %v", err)
	}
	if got := vr.data[1].Stock; got != 5 {
		t.Errorf("stock despues de cancelar = %d, se esperaba 5", got)
	}
}

// --- Regla 7: un usuario no toca los pedidos de otro ------------------------

func TestCancelar_PedidoDeOtroUsuario(t *testing.T) {
	s, _, _ := setup()

	p, err := s.Checkout(7, []ItemCarrito{{VarianteID: 1, Cantidad: 1}})
	if err != nil {
		t.Fatalf("checkout fallo: %v", err)
	}

	// El usuario 99 intenta cancelar el pedido del 7.
	err = s.Cancelar(p.ID, 99, false)
	if err == nil {
		t.Fatal("un usuario no deberia poder cancelar el pedido de otro")
	}
	// 404 y no 403: no le confirmamos a un tercero que el pedido existe.
	if k := kindDe(t, err); k != dom.KindNoEncontrado {
		t.Errorf("kind = %v, se esperaba KindNoEncontrado", k)
	}
}

// --- Regla 8 (parcial): carrito vacio ---------------------------------------

func TestCheckout_CarritoVacio(t *testing.T) {
	s, _, _ := setup()

	if _, err := s.Checkout(7, nil); err == nil {
		t.Fatal("se esperaba error con carrito vacio")
	} else if k := kindDe(t, err); k != dom.KindValidacion {
		t.Errorf("kind = %v, se esperaba KindValidacion", k)
	}
}
