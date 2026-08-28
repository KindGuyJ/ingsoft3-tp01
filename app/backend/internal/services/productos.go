package services

import (
	"fmt"
	"strings"

	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/dao"
	dom "github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/errors"
)

// ProductoRepo es lo que el ABM de productos necesita de la persistencia.
type ProductoRepo interface {
	Listar(categoria string) ([]dao.Producto, error)
	BuscarPorID(id uint) (*dao.Producto, error)
	Crear(p *dao.Producto) error
	Actualizar(p *dao.Producto) error
}

// VarianteAltaRepo esta separada de VarianteRepo (la que usa PedidosService) a
// proposito: el checkout solo lee stock y lo actualiza, mientras que el ABM da
// de alta variantes. Interfaces chicas = fakes chicos en los tests.
type VarianteAltaRepo interface {
	Crear(v *dao.Variante) error
}

type ProductosService struct {
	productos ProductoRepo
	variantes VarianteAltaRepo
}

func NuevoProductosService(p ProductoRepo, v VarianteAltaRepo) *ProductosService {
	return &ProductosService{productos: p, variantes: v}
}

// VarianteNueva es el alta de una combinacion talle x color.
type VarianteNueva struct {
	Talle string
	Color string
	Stock int
	SKU   string // opcional: si viene vacio se genera
}

// ProductoNuevo es el alta de un producto con sus variantes iniciales.
type ProductoNuevo struct {
	Nombre      string
	Descripcion string
	Precio      float64
	Categoria   string
	Variantes   []VarianteNueva
}

// ProductoEditado es la edicion. Activo es puntero para poder distinguir
// "no lo mandaron" (nil, no se toca) de "mandaron false" (dar de baja).
type ProductoEditado struct {
	Nombre      string
	Descripcion string
	Precio      float64
	Categoria   string
	Activo      *bool
}

// Listar devuelve el catalogo activo, opcionalmente filtrado por categoria.
func (s *ProductosService) Listar(categoria string) ([]dao.Producto, error) {
	ps, err := s.productos.Listar(strings.TrimSpace(categoria))
	if err != nil {
		return nil, dom.Interno("no se pudieron listar los productos", err)
	}
	return ps, nil
}

// VerDetalle devuelve un producto con sus variantes e imagenes.
func (s *ProductosService) VerDetalle(id uint) (*dao.Producto, error) {
	p, err := s.productos.BuscarPorID(id)
	if err != nil {
		return nil, dom.Interno("no se pudo leer el producto", err)
	}
	if p == nil {
		return nil, dom.NoEncontrado("el producto %d no existe", id)
	}
	return p, nil
}

// Crear da de alta un producto con sus variantes.
//
// Reglas (regla 8 del TP):
//   - precio > 0
//   - talle dentro de dao.TallesValidos
//   - stock >= 0
//   - no se repite la combinacion (producto, talle, color)
//
// Se valida TODO antes de insertar nada: no queremos un producto a medio cargar
// porque la tercera variante traia un talle invalido.
func (s *ProductosService) Crear(in ProductoNuevo) (*dao.Producto, error) {
	nombre := strings.TrimSpace(in.Nombre)
	if nombre == "" {
		return nil, dom.Validacion("el nombre del producto es obligatorio")
	}
	if in.Precio <= 0 {
		return nil, dom.Validacion("el precio debe ser mayor a cero")
	}
	if len(in.Variantes) == 0 {
		return nil, dom.Validacion("el producto necesita al menos una variante")
	}

	normalizadas := make([]VarianteNueva, 0, len(in.Variantes))
	vistas := map[string]bool{}
	for _, vn := range in.Variantes {
		norm, err := normalizarVariante(vn)
		if err != nil {
			return nil, err
		}
		clave := norm.Talle + "|" + strings.ToLower(norm.Color)
		if vistas[clave] {
			return nil, dom.Conflicto("la variante %s / %s viene repetida", norm.Talle, norm.Color)
		}
		vistas[clave] = true
		normalizadas = append(normalizadas, norm)
	}

	p := &dao.Producto{
		Nombre:      nombre,
		Descripcion: strings.TrimSpace(in.Descripcion),
		Precio:      in.Precio,
		Categoria:   strings.TrimSpace(in.Categoria),
		Activo:      true,
	}
	if err := s.productos.Crear(p); err != nil {
		return nil, dom.Interno("no se pudo crear el producto", err)
	}

	// Las variantes se insertan despues del producto porque el SKU generado
	// lleva el ID, que recien existe una vez insertado.
	// TODO(TP4+): misma deuda que el checkout, envolver en una transaccion.
	for _, vn := range normalizadas {
		v := &dao.Variante{
			ProductoID: p.ID,
			Talle:      vn.Talle,
			Color:      vn.Color,
			Stock:      vn.Stock,
			SKU:        skuDe(vn.SKU, p.ID, vn.Talle, vn.Color),
		}
		if err := s.variantes.Crear(v); err != nil {
			return nil, dom.Interno("no se pudo crear la variante", err)
		}
		p.Variantes = append(p.Variantes, *v)
	}
	return p, nil
}

// Editar cambia los datos del producto. NO toca variantes ni stock: para eso
// estan AgregarVariante y el checkout. Editar un producto no puede alterar
// silenciosamente el inventario.
func (s *ProductosService) Editar(id uint, in ProductoEditado) (*dao.Producto, error) {
	p, err := s.VerDetalle(id)
	if err != nil {
		return nil, err
	}
	nombre := strings.TrimSpace(in.Nombre)
	if nombre == "" {
		return nil, dom.Validacion("el nombre del producto es obligatorio")
	}
	if in.Precio <= 0 {
		return nil, dom.Validacion("el precio debe ser mayor a cero")
	}

	p.Nombre = nombre
	p.Descripcion = strings.TrimSpace(in.Descripcion)
	p.Precio = in.Precio
	p.Categoria = strings.TrimSpace(in.Categoria)
	if in.Activo != nil {
		p.Activo = *in.Activo
	}

	if err := s.productos.Actualizar(p); err != nil {
		return nil, dom.Interno("no se pudo actualizar el producto", err)
	}
	return p, nil
}

// AgregarVariante suma un talle/color a un producto existente.
// Rechaza duplicados contra las variantes que el producto ya tiene.
func (s *ProductosService) AgregarVariante(productoID uint, in VarianteNueva) (*dao.Variante, error) {
	p, err := s.VerDetalle(productoID)
	if err != nil {
		return nil, err
	}
	norm, err := normalizarVariante(in)
	if err != nil {
		return nil, err
	}
	for _, existente := range p.Variantes {
		if strings.EqualFold(existente.Talle, norm.Talle) && strings.EqualFold(existente.Color, norm.Color) {
			return nil, dom.Conflicto(
				"el producto %d ya tiene la variante %s / %s", productoID, norm.Talle, norm.Color)
		}
	}

	v := &dao.Variante{
		ProductoID: p.ID,
		Talle:      norm.Talle,
		Color:      norm.Color,
		Stock:      norm.Stock,
		SKU:        skuDe(norm.SKU, p.ID, norm.Talle, norm.Color),
	}
	if err := s.variantes.Crear(v); err != nil {
		return nil, dom.Interno("no se pudo crear la variante", err)
	}
	return v, nil
}

// normalizarVariante valida talle, color y stock, y devuelve el talle en
// mayusculas para que "m" y "M" no entren como dos variantes distintas.
func normalizarVariante(vn VarianteNueva) (VarianteNueva, error) {
	talle := strings.ToUpper(strings.TrimSpace(vn.Talle))
	if !talleValido(talle) {
		return VarianteNueva{}, dom.Validacion(
			"el talle %q no es valido; los permitidos son %s",
			vn.Talle, strings.Join(dao.TallesValidos, ", "))
	}
	color := strings.TrimSpace(vn.Color)
	if color == "" {
		return VarianteNueva{}, dom.Validacion("el color es obligatorio")
	}
	if vn.Stock < 0 {
		return VarianteNueva{}, dom.Validacion("el stock no puede ser negativo")
	}
	return VarianteNueva{Talle: talle, Color: color, Stock: vn.Stock, SKU: strings.TrimSpace(vn.SKU)}, nil
}

func talleValido(talle string) bool {
	for _, t := range dao.TallesValidos {
		if t == talle {
			return true
		}
	}
	return false
}

// skuDe genera un SKU cuando el admin no manda uno.
//
// No es cosmetico: la columna SKU tiene indice unico, asi que si se dejara
// vacia, la segunda variante sin SKU chocaria contra la primera. Al derivarlo
// de (producto, talle, color) queda unico por construccion, igual que la regla.
func skuDe(sku string, productoID uint, talle, color string) string {
	if sku != "" {
		return sku
	}
	limpio := strings.ToUpper(strings.ReplaceAll(color, " ", "-"))
	return fmt.Sprintf("P%d-%s-%s", productoID, talle, limpio)
}
