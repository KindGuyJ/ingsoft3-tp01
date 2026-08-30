// Package dao contiene las entidades GORM que mapean a las tablas de MySQL.
package dao

import "time"

// Talles permitidos. Cualquier valor fuera de esta lista se rechaza en el service.
var TallesValidos = []string{"XS", "S", "M", "L", "XL", "XXL"}

// Estados posibles de un pedido.
const (
	EstadoPendiente = "pendiente"
	EstadoPagado    = "pagado"
	EstadoEnviado   = "enviado"
	EstadoEntregado = "entregado"
	EstadoCancelado = "cancelado"
)

type Usuario struct {
	ID            uint      `gorm:"primaryKey"`
	Email         string    `gorm:"uniqueIndex;size:255;not null"`
	PasswordHash  string    `gorm:"size:255;not null"`
	Nombre        string    `gorm:"size:100;not null"`
	Apellido      string    `gorm:"size:100;not null"`
	EsAdmin       bool      `gorm:"default:false"`
	Activo        bool      `gorm:"default:true"`
	FechaRegistro time.Time `gorm:"autoCreateTime"`
}

type Producto struct {
	ID          uint    `gorm:"primaryKey"`
	Nombre      string  `gorm:"size:200;not null"`
	Descripcion string  `gorm:"type:text"`
	Precio      float64 `gorm:"type:decimal(10,2);not null"` // > 0, validado en el service
	Categoria   string  `gorm:"size:60;index"`
	Activo      bool    `gorm:"default:true"`

	Variantes []Variante `gorm:"foreignKey:ProductoID"`
	Imagenes  []Imagen   `gorm:"foreignKey:ProductoID"`
}

// Variante es la combinacion talle x color de un producto. El stock vive ACA,
// no en Producto: no tiene sentido un stock global si se vende por talle.
type Variante struct {
	ID         uint   `gorm:"primaryKey"`
	ProductoID uint   `gorm:"not null;uniqueIndex:idx_variante_unica"`
	Talle      string `gorm:"size:10;not null;uniqueIndex:idx_variante_unica"`
	Color      string `gorm:"size:40;not null;uniqueIndex:idx_variante_unica"`
	Stock      int    `gorm:"not null;default:0"` // >= 0
	SKU        string `gorm:"uniqueIndex;size:60"`

	Producto *Producto `gorm:"foreignKey:ProductoID"`
}

// Imagen cuelga del producto, no de la variante: las fotos cambian por color,
// no por talle. Color vacio = imagen generica del producto.
type Imagen struct {
	ID         uint   `gorm:"primaryKey"`
	ProductoID uint   `gorm:"not null;index"`
	URL        string `gorm:"size:500;not null"` // "/productos/x.jpg" o "/uploads/x.jpg"
	Color      string `gorm:"size:40"`
	Orden      int    `gorm:"default:0"` // 0 = principal
	AltText    string `gorm:"size:200"`
}

type Pedido struct {
	ID        uint      `gorm:"primaryKey"`
	UsuarioID uint      `gorm:"not null;index"`
	Estado    string    `gorm:"size:20;not null;default:'pendiente';index"`
	Total     float64   `gorm:"type:decimal(10,2);not null"`
	Fecha     time.Time `gorm:"autoCreateTime"`

	Items []PedidoItem `gorm:"foreignKey:PedidoID"`
}

// PedidoItem congela el precio al momento de la compra. Si manana el admin sube
// el precio del producto, este pedido no cambia. Es una regla de negocio, no un
// detalle de implementacion.
type PedidoItem struct {
	ID             uint    `gorm:"primaryKey"`
	PedidoID       uint    `gorm:"not null;index"`
	VarianteID     uint    `gorm:"not null"`
	Cantidad       int     `gorm:"not null"`
	PrecioUnitario float64 `gorm:"type:decimal(10,2);not null"`

	// Denormalizado a proposito: si el producto se borra o se renombra, el
	// historial del pedido sigue siendo legible.
	DescripcionItem string `gorm:"size:300"`
}

// Subtotal del item.
func (i PedidoItem) Subtotal() float64 {
	return float64(i.Cantidad) * i.PrecioUnitario
}
