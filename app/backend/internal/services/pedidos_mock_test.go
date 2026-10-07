package services

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/dao"
	dom "github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/errors"
)

// ---------------------------------------------------------------------------
// Mocks (TP5).
//
// Los fakes de pedidos_test.go son STUBS con estado: los tests miran como
// quedo el stock despues del checkout. Estos son MOCKS: anotan cada llamada
// que reciben, en orden, y el assert compara esa lista. Lo que verifican es
// la INTERACCION del service con sus dependencias, que el estado final no
// muestra: en que orden se llamaron, cuantas veces y con que argumentos.
//
// Estan escritos a mano y no con una libreria (gomock, testify/mock): en Go
// alcanza con un struct que implemente la interfaz, y asi no hay nada que
// explicar mas alla del propio archivo.
// ---------------------------------------------------------------------------

// registro es la lista de llamadas que comparten los dos mocks, para poder
// verificar el orden ENTRE repositorios (primero Crear, despues el stock).
type registro struct {
	llamadas []string
}

func (r *registro) anotar(formato string, args ...any) {
	r.llamadas = append(r.llamadas, fmt.Sprintf(formato, args...))
}

type mockVarianteRepo struct {
	reg       *registro
	variantes map[uint]*dao.Variante
}

func (m *mockVarianteRepo) BuscarPorID(id uint) (*dao.Variante, error) {
	// Las lecturas no se anotan: lo que importa del checkout es que escribe.
	return m.variantes[id], nil
}

func (m *mockVarianteRepo) ActualizarStock(id uint, nuevoStock int) error {
	m.reg.anotar("ActualizarStock(%d, %d)", id, nuevoStock)
	return nil
}

type mockPedidoRepo struct {
	reg        *registro
	errorCrear error // si no es nil, Crear falla con este error
}

func (m *mockPedidoRepo) Crear(p *dao.Pedido) error {
	m.reg.anotar("Crear(%d items)", len(p.Items))
	return m.errorCrear
}

func (m *mockPedidoRepo) BuscarPorID(uint) (*dao.Pedido, error)       { return nil, nil }
func (m *mockPedidoRepo) ListarPorUsuario(uint) ([]dao.Pedido, error) { return nil, nil }
func (m *mockPedidoRepo) ActualizarEstado(uint, string) error         { return nil }

// dosVariantes arma los mocks con una remera (stock 5) y un buzo (stock 4).
func dosVariantes() (*registro, *mockVarianteRepo, *mockPedidoRepo) {
	reg := &registro{}
	remera := &dao.Producto{ID: 1, Nombre: "Remera basica", Precio: 10000, Activo: true}
	buzo := &dao.Producto{ID: 2, Nombre: "Buzo canguro", Precio: 30000, Activo: true}
	vr := &mockVarianteRepo{reg: reg, variantes: map[uint]*dao.Variante{
		1: {ID: 1, ProductoID: 1, Talle: "M", Color: "Negro", Stock: 5, Producto: remera},
		2: {ID: 2, ProductoID: 2, Talle: "L", Color: "Gris", Stock: 4, Producto: buzo},
	}}
	return reg, vr, &mockPedidoRepo{reg: reg}
}

// Regla 2, vista desde la interaccion: se crea el pedido UNA vez y recien
// despues se descuenta el stock, una llamada por variante y con el stock
// final exacto (no con la cantidad comprada).
func TestCheckout_CreaElPedidoYDespuesDescuentaUnaVezPorVariante(t *testing.T) {
	// Arrange
	reg, vr, pr := dosVariantes()
	s := NuevoPedidosService(vr, pr, 50000, 5000)

	// Act
	_, err := s.Checkout(7, []ItemCarrito{{VarianteID: 1, Cantidad: 2}, {VarianteID: 2, Cantidad: 1}})

	// Assert
	if err != nil {
		t.Fatalf("checkout fallo: %v", err)
	}
	esperadas := []string{
		"Crear(2 items)",
		"ActualizarStock(1, 3)",
		"ActualizarStock(2, 3)",
	}
	if !reflect.DeepEqual(reg.llamadas, esperadas) {
		t.Errorf("llamadas = %q\nse esperaba  %q", reg.llamadas, esperadas)
	}
}

// Caso de error con mock: si la base no pudo guardar el pedido, no se
// descuenta stock. Con un fake este camino no se puede provocar, porque el
// fake nunca falla; el mock lo hace fallar a pedido.
func TestCheckout_SiNoSePudoCrearElPedidoNoTocaElStock(t *testing.T) {
	// Arrange
	reg, vr, pr := dosVariantes()
	pr.errorCrear = errors.New("se cayo la conexion")
	s := NuevoPedidosService(vr, pr, 50000, 5000)

	// Act
	_, err := s.Checkout(7, []ItemCarrito{{VarianteID: 1, Cantidad: 2}})

	// Assert
	if k := kindDe(t, err); k != dom.KindInterno {
		t.Errorf("kind = %v, se esperaba KindInterno", k)
	}
	esperadas := []string{"Crear(1 items)"}
	if !reflect.DeepEqual(reg.llamadas, esperadas) {
		t.Errorf("llamadas = %q\nse esperaba  %q", reg.llamadas, esperadas)
	}
}
