package services

import (
	"testing"

	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/dao"
	dom "github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/errors"
)

type fakeProductoRepo struct {
	data    map[uint]*dao.Producto
	proximo uint
}

func nuevoFakeProductoRepo() *fakeProductoRepo {
	return &fakeProductoRepo{data: map[uint]*dao.Producto{}, proximo: 1}
}

func (f *fakeProductoRepo) Listar(categoria string) ([]dao.Producto, error) {
	var out []dao.Producto
	for _, p := range f.data {
		if p.Activo && (categoria == "" || p.Categoria == categoria) {
			out = append(out, *p)
		}
	}
	return out, nil
}

func (f *fakeProductoRepo) BuscarPorID(id uint) (*dao.Producto, error) {
	p, ok := f.data[id]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (f *fakeProductoRepo) Crear(p *dao.Producto) error {
	p.ID = f.proximo
	f.proximo++
	f.data[p.ID] = p
	return nil
}

func (f *fakeProductoRepo) Actualizar(p *dao.Producto) error {
	f.data[p.ID] = p
	return nil
}

type fakeVarianteAltaRepo struct {
	creadas []dao.Variante
	proximo uint
}

func (f *fakeVarianteAltaRepo) Crear(v *dao.Variante) error {
	f.proximo++
	v.ID = f.proximo
	f.creadas = append(f.creadas, *v)
	return nil
}

func setupProductos() (*ProductosService, *fakeProductoRepo, *fakeVarianteAltaRepo) {
	pr := nuevoFakeProductoRepo()
	vr := &fakeVarianteAltaRepo{}
	return NuevoProductosService(pr, vr), pr, vr
}

func productoValido() ProductoNuevo {
	return ProductoNuevo{
		Nombre:    "Remera basica",
		Precio:    10000,
		Categoria: "remeras",
		Variantes: []VarianteNueva{
			{Talle: "M", Color: "Negro", Stock: 5},
			{Talle: "L", Color: "Negro", Stock: 0},
		},
	}
}

func TestCrearProducto_Ok(t *testing.T) {
	s, _, vr := setupProductos()

	p, err := s.Crear(productoValido())
	if err != nil {
		t.Fatalf("crear fallo: %v", err)
	}
	if len(vr.creadas) != 2 {
		t.Fatalf("se crearon %d variantes, se esperaban 2", len(vr.creadas))
	}
	if !p.Activo {
		t.Error("un producto recien creado tiene que quedar activo")
	}
	// El SKU se genera solo: la columna tiene indice unico y dos variantes
	// sin SKU chocarian entre si.
	if vr.creadas[0].SKU == "" || vr.creadas[0].SKU == vr.creadas[1].SKU {
		t.Errorf("SKUs invalidos: %q y %q", vr.creadas[0].SKU, vr.creadas[1].SKU)
	}
}

// --- Regla 8: precio > 0 ----------------------------------------------------

func TestCrearProducto_PrecioInvalido(t *testing.T) {
	for _, precio := range []float64{0, -1500} {
		s, _, vr := setupProductos()
		in := productoValido()
		in.Precio = precio

		_, err := s.Crear(in)
		if err == nil {
			t.Fatalf("se esperaba error con precio %.2f", precio)
		}
		if k := kindDe(t, err); k != dom.KindValidacion {
			t.Errorf("kind = %v, se esperaba KindValidacion", k)
		}
		if len(vr.creadas) != 0 {
			t.Error("no se tiene que crear ninguna variante si el producto no valida")
		}
	}
}

// --- Regla 8: talle dentro de dao.TallesValidos -----------------------------

func TestCrearProducto_TalleInvalido(t *testing.T) {
	s, pr, _ := setupProductos()
	in := productoValido()
	in.Variantes = []VarianteNueva{{Talle: "XXXL", Color: "Negro", Stock: 1}}

	_, err := s.Crear(in)
	if err == nil {
		t.Fatal("se esperaba error por talle invalido")
	}
	if k := kindDe(t, err); k != dom.KindValidacion {
		t.Errorf("kind = %v, se esperaba KindValidacion", k)
	}
	// Se valida ANTES de insertar: la base no queda con un producto huerfano.
	if len(pr.data) != 0 {
		t.Error("no se tiene que crear el producto si una variante no valida")
	}
}

func TestCrearProducto_TalleEnMinusculaSeNormaliza(t *testing.T) {
	s, _, vr := setupProductos()
	in := productoValido()
	in.Variantes = []VarianteNueva{{Talle: "m", Color: "Negro", Stock: 1}}

	if _, err := s.Crear(in); err != nil {
		t.Fatalf("crear fallo: %v", err)
	}
	if vr.creadas[0].Talle != "M" {
		t.Errorf("talle = %q, se esperaba M", vr.creadas[0].Talle)
	}
}

// --- Regla 8: stock >= 0 ----------------------------------------------------

func TestCrearProducto_StockNegativo(t *testing.T) {
	s, _, _ := setupProductos()
	in := productoValido()
	in.Variantes = []VarianteNueva{{Talle: "M", Color: "Negro", Stock: -1}}

	_, err := s.Crear(in)
	if err == nil {
		t.Fatal("se esperaba error por stock negativo")
	}
	if k := kindDe(t, err); k != dom.KindValidacion {
		t.Errorf("kind = %v, se esperaba KindValidacion", k)
	}
}

// --- Regla 8: no se duplica (producto, talle, color) ------------------------

func TestCrearProducto_VarianteDuplicadaEnElMismoAlta(t *testing.T) {
	s, _, _ := setupProductos()
	in := productoValido()
	in.Variantes = []VarianteNueva{
		{Talle: "M", Color: "Negro", Stock: 5},
		{Talle: "m", Color: "negro", Stock: 3}, // la misma, escrita distinto
	}

	_, err := s.Crear(in)
	if err == nil {
		t.Fatal("se esperaba error por variante duplicada")
	}
	if k := kindDe(t, err); k != dom.KindConflicto {
		t.Errorf("kind = %v, se esperaba KindConflicto", k)
	}
}

func TestAgregarVariante_DuplicadaContraLasExistentes(t *testing.T) {
	s, pr, _ := setupProductos()
	p, err := s.Crear(productoValido())
	if err != nil {
		t.Fatalf("crear fallo: %v", err)
	}
	// El fake devuelve el producto tal cual quedo, con sus variantes.
	pr.data[p.ID] = p

	_, err = s.AgregarVariante(p.ID, VarianteNueva{Talle: "M", Color: "Negro", Stock: 2})
	if err == nil {
		t.Fatal("se esperaba error: esa combinacion ya existe")
	}
	if k := kindDe(t, err); k != dom.KindConflicto {
		t.Errorf("kind = %v, se esperaba KindConflicto", k)
	}

	// Una combinacion nueva sí entra.
	if _, err := s.AgregarVariante(p.ID, VarianteNueva{Talle: "S", Color: "Negro", Stock: 2}); err != nil {
		t.Errorf("una variante nueva deberia entrar: %v", err)
	}
}

// --- Detalle de un producto inexistente -------------------------------------

func TestVerDetalle_NoExiste(t *testing.T) {
	s, _, _ := setupProductos()

	_, err := s.VerDetalle(42)
	if err == nil {
		t.Fatal("se esperaba 404 de dominio")
	}
	if k := kindDe(t, err); k != dom.KindNoEncontrado {
		t.Errorf("kind = %v, se esperaba KindNoEncontrado", k)
	}
}

// --- Editar no toca el stock ------------------------------------------------

func TestEditar_NoTocaLasVariantes(t *testing.T) {
	s, pr, vr := setupProductos()
	p, err := s.Crear(productoValido())
	if err != nil {
		t.Fatalf("crear fallo: %v", err)
	}
	pr.data[p.ID] = p
	variantesAntes := len(vr.creadas)

	editado, err := s.Editar(p.ID, ProductoEditado{
		Nombre:    "Remera basica premium",
		Precio:    13000,
		Categoria: "remeras",
	})
	if err != nil {
		t.Fatalf("editar fallo: %v", err)
	}
	if editado.Nombre != "Remera basica premium" || editado.Precio != 13000 {
		t.Errorf("la edicion no se aplico: %+v", editado)
	}
	if len(vr.creadas) != variantesAntes {
		t.Error("editar el producto no puede crear ni borrar variantes")
	}
	// Activo nil = no se toca.
	if !editado.Activo {
		t.Error("no se mando Activo, no deberia haberse dado de baja")
	}
}
