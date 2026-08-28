// Package repository implementa el acceso a datos con GORM.
// Las interfaces que estos tipos satisfacen estan declaradas en services/.
package repository

import (
	"errors"

	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/dao"
	"gorm.io/gorm"
)

type VarianteRepo struct{ db *gorm.DB }

func NuevoVarianteRepo(db *gorm.DB) *VarianteRepo { return &VarianteRepo{db: db} }

// BuscarPorID devuelve (nil, nil) cuando no existe. El "no encontrado" es un
// caso de negocio, no un error tecnico: quien decide que hacer es el service.
func (r *VarianteRepo) BuscarPorID(id uint) (*dao.Variante, error) {
	var v dao.Variante
	err := r.db.Preload("Producto").First(&v, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// Crear da de alta una variante. La usa el ABM de productos.
func (r *VarianteRepo) Crear(v *dao.Variante) error {
	return r.db.Create(v).Error
}

func (r *VarianteRepo) ActualizarStock(id uint, nuevoStock int) error {
	return r.db.Model(&dao.Variante{}).Where("id = ?", id).Update("stock", nuevoStock).Error
}

type PedidoRepo struct{ db *gorm.DB }

func NuevoPedidoRepo(db *gorm.DB) *PedidoRepo { return &PedidoRepo{db: db} }

func (r *PedidoRepo) Crear(p *dao.Pedido) error {
	return r.db.Create(p).Error
}

func (r *PedidoRepo) BuscarPorID(id uint) (*dao.Pedido, error) {
	var p dao.Pedido
	err := r.db.Preload("Items").First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PedidoRepo) ListarPorUsuario(usuarioID uint) ([]dao.Pedido, error) {
	var ps []dao.Pedido
	err := r.db.Preload("Items").Where("usuario_id = ?", usuarioID).Order("fecha desc").Find(&ps).Error
	return ps, err
}

func (r *PedidoRepo) ActualizarEstado(id uint, estado string) error {
	return r.db.Model(&dao.Pedido{}).Where("id = ?", id).Update("estado", estado).Error
}

type ProductoRepo struct{ db *gorm.DB }

func NuevoProductoRepo(db *gorm.DB) *ProductoRepo { return &ProductoRepo{db: db} }

func (r *ProductoRepo) Listar(categoria string) ([]dao.Producto, error) {
	q := r.db.Preload("Variantes").Preload("Imagenes").Where("activo = ?", true)
	if categoria != "" {
		q = q.Where("categoria = ?", categoria)
	}
	var ps []dao.Producto
	return ps, q.Find(&ps).Error
}

func (r *ProductoRepo) BuscarPorID(id uint) (*dao.Producto, error) {
	var p dao.Producto
	err := r.db.Preload("Variantes").Preload("Imagenes").First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductoRepo) Crear(p *dao.Producto) error { return r.db.Create(p).Error }

// Actualizar guarda los campos del producto.
//
// El Omit no es decorativo: sin el, Save reescribe tambien las asociaciones
// precargadas, y editar el nombre de un producto podria pisar el stock de sus
// variantes con los valores que se leyeron un instante antes.
func (r *ProductoRepo) Actualizar(p *dao.Producto) error {
	return r.db.Omit("Variantes", "Imagenes").Save(p).Error
}

type UsuarioRepo struct{ db *gorm.DB }

func NuevoUsuarioRepo(db *gorm.DB) *UsuarioRepo { return &UsuarioRepo{db: db} }

func (r *UsuarioRepo) BuscarPorEmail(email string) (*dao.Usuario, error) {
	var u dao.Usuario
	err := r.db.Where("email = ?", email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UsuarioRepo) Crear(u *dao.Usuario) error { return r.db.Create(u).Error }
