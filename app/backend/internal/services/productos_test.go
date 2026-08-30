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

func (f *fakeProductoRepo) Listar(categoria string, soloActivos bool) ([]dao.Producto, error) {
	var out []dao.Producto
	for _, p := range f.data {
		if soloActivos && !p.Activo {
			continue
		}
		if categoria == "" || p.Categoria == categoria {
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

type fakeVarianteABMRepo struct {
	creadas []dao.Variante
	proximo uint
}

func (f *fakeVarianteABMRepo) Crear(v *dao.Variante) error {
	f.proximo++
	v.ID = f.proximo
	f.creadas = append(f.creadas, *v)
	return nil
}

func (f *fakeVarianteABMRepo) BuscarPorID(id uint) (*dao.Variante, error) {
	for i := range f.creadas {
		if f.creadas[i].ID == id {
			return &f.creadas[i], nil
		}
	}
	return nil, nil
}

func (f *fakeVarianteABMRepo) ActualizarStock(id uint, nuevoStock int) error {
	for i := range f.creadas {
		if f.creadas[i].ID == id {
			f.creadas[i].Stock = nuevoStock
		}
	}
	return nil
}

func setupProductos() (*ProductosService, *fakeProductoRepo, *fakeVarianteABMRepo) {
	pr := nuevoFakeProductoRepo()
	vr := &fakeVarianteABMRepo{}
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

// --- Baja / alta de productos ----------------------------------------------

func TestEditar_DarDeBajaLoSacaDelCatalogoPeroNoLoBorra(t *testing.T) {
	s, _, _ := setupProductos()
	p, err := s.Crear(productoValido())
	if err != nil {
		t.Fatalf("no se pudo crear: %v", err)
	}

	baja := false
	editado, err := s.Editar(p.ID, ProductoEditado{
		Nombre: p.Nombre, Precio: p.Precio, Categoria: p.Categoria, Activo: &baja,
	})
	if err != nil {
		t.Fatalf("no se pudo dar de baja: %v", err)
	}
	if editado.Activo {
		t.Fatal("se mando Activo=false y siguio activo")
	}

	// Desaparece del catalogo publico...
	publico, _ := s.Listar("")
	if len(publico) != 0 {
		t.Errorf("el catalogo publico devolvio %d productos dados de baja", len(publico))
	}

	// ...pero el admin lo sigue viendo, que es lo que permite reactivarlo.
	todos, _ := s.ListarTodos("")
	if len(todos) != 1 {
		t.Fatalf("el admin deberia ver 1 producto, vio %d", len(todos))
	}

	alta := true
	revivido, err := s.Editar(p.ID, ProductoEditado{
		Nombre: p.Nombre, Precio: p.Precio, Categoria: p.Categoria, Activo: &alta,
	})
	if err != nil || !revivido.Activo {
		t.Fatalf("no se pudo volver a dar de alta: %v", err)
	}
	if publico, _ := s.Listar(""); len(publico) != 1 {
		t.Error("el producto reactivado no volvio al catalogo")
	}
}

// --- Correccion manual de stock --------------------------------------------

func TestActualizarStock_Ok(t *testing.T) {
	s, _, vr := setupProductos()
	p, _ := s.Crear(productoValido())
	variante := vr.creadas[0]

	v, err := s.ActualizarStock(p.ID, variante.ID, 42)
	if err != nil {
		t.Fatalf("no se pudo actualizar el stock: %v", err)
	}
	if v.Stock != 42 {
		t.Errorf("stock devuelto = %d, se esperaba 42", v.Stock)
	}
	if guardada, _ := vr.BuscarPorID(variante.ID); guardada.Stock != 42 {
		t.Errorf("stock guardado = %d, se esperaba 42", guardada.Stock)
	}
}

func TestActualizarStock_CeroEsValido(t *testing.T) {
	s, _, vr := setupProductos()
	p, _ := s.Crear(productoValido())

	// Agotar una variante es una operacion legitima, no un error de carga.
	if _, err := s.ActualizarStock(p.ID, vr.creadas[0].ID, 0); err != nil {
		t.Fatalf("poner el stock en cero deberia estar permitido: %v", err)
	}
}

func TestActualizarStock_NegativoSeRechaza(t *testing.T) {
	s, _, vr := setupProductos()
	p, _ := s.Crear(productoValido())

	_, err := s.ActualizarStock(p.ID, vr.creadas[0].ID, -1)
	if k := kindDe(t, err); k != dom.KindValidacion {
		t.Errorf("kind = %v, se esperaba KindValidacion", k)
	}
}

func TestActualizarStock_DeOtroProductoNoSeToca(t *testing.T) {
	s, _, vr := setupProductos()
	uno, _ := s.Crear(productoValido())
	otro, _ := s.Crear(ProductoNuevo{
		Nombre: "Buzo", Precio: 30000, Categoria: "buzos",
		Variantes: []VarianteNueva{{Talle: "M", Color: "Gris", Stock: 3}},
	})
	varianteDeOtro := vr.creadas[len(vr.creadas)-1]
	if varianteDeOtro.ProductoID != otro.ID {
		t.Fatal("el fixture quedo mal armado")
	}

	// Se responde "no existe" y no "es de otro": la ruta la elige quien llama.
	_, err := s.ActualizarStock(uno.ID, varianteDeOtro.ID, 99)
	if k := kindDe(t, err); k != dom.KindNoEncontrado {
		t.Errorf("kind = %v, se esperaba KindNoEncontrado", k)
	}
	if guardada, _ := vr.BuscarPorID(varianteDeOtro.ID); guardada.Stock != 3 {
		t.Errorf("el stock de la otra variante cambio a %d", guardada.Stock)
	}
}

// --- Regla 9 (lado catalogo): el detalle publico esconde los de baja --------

func TestVerDetallePublico_EscondeElDadoDeBaja(t *testing.T) {
	s, _, _ := setupProductos()
	p, _ := s.Crear(productoValido())

	if _, err := s.VerDetallePublico(p.ID); err != nil {
		t.Fatalf("un producto activo tiene que verse: %v", err)
	}

	baja := false
	if _, err := s.Editar(p.ID, ProductoEditado{
		Nombre: p.Nombre, Precio: p.Precio, Activo: &baja,
	}); err != nil {
		t.Fatalf("no se pudo dar de baja: %v", err)
	}

	_, err := s.VerDetallePublico(p.ID)
	if k := kindDe(t, err); k != dom.KindNoEncontrado {
		t.Errorf("kind = %v, se esperaba KindNoEncontrado", k)
	}
}

// Este test existe por una trampa concreta: si el filtro de "activo" se pusiera
// DENTRO de VerDetalle en vez de en VerDetallePublico, Editar y AgregarVariante
// —que lo llaman— dejarian de funcionar sobre un producto de baja, y no se
// podria reactivar ni corregir nada desde el panel.
func TestEditarYAgregarVariante_FuncionanSobreUnProductoDeBaja(t *testing.T) {
	s, _, _ := setupProductos()
	p, _ := s.Crear(productoValido())
	baja := false
	if _, err := s.Editar(p.ID, ProductoEditado{Nombre: p.Nombre, Precio: p.Precio, Activo: &baja}); err != nil {
		t.Fatalf("no se pudo dar de baja: %v", err)
	}

	if _, err := s.AgregarVariante(p.ID, VarianteNueva{Talle: "XL", Color: "Negro", Stock: 2}); err != nil {
		t.Errorf("no se pudo agregar una variante a un producto de baja: %v", err)
	}

	alta := true
	revivido, err := s.Editar(p.ID, ProductoEditado{Nombre: p.Nombre, Precio: p.Precio, Activo: &alta})
	if err != nil || !revivido.Activo {
		t.Fatalf("no se pudo reactivar: %v", err)
	}
	if _, err := s.VerDetallePublico(p.ID); err != nil {
		t.Errorf("el producto reactivado tiene que volver a verse: %v", err)
	}
}
