// Package dto define los contratos de entrada y salida de la API.
// Se mantienen separados de dao/ a proposito: la forma en que se guarda algo
// no tiene por que ser la forma en que se expone (ej: nunca sale el PasswordHash).
package dto

type RegistroRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Nombre   string `json:"nombre" binding:"required"`
	Apellido string `json:"apellido" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token   string      `json:"token"`
	Usuario UsuarioResp `json:"usuario"`
}

type UsuarioResp struct {
	ID       uint   `json:"id"`
	Email    string `json:"email"`
	Nombre   string `json:"nombre"`
	Apellido string `json:"apellido"`
	EsAdmin  bool   `json:"es_admin"`
}

type ItemCarritoReq struct {
	VarianteID uint `json:"variante_id" binding:"required"`
	Cantidad   int  `json:"cantidad" binding:"required,min=1"`
}

type CheckoutRequest struct {
	Items []ItemCarritoReq `json:"items" binding:"required,min=1,dive"`
}

type VarianteResp struct {
	ID    uint   `json:"id"`
	Talle string `json:"talle"`
	Color string `json:"color"`
	Stock int    `json:"stock"`
	SKU   string `json:"sku"`
}

type ImagenResp struct {
	URL     string `json:"url"`
	Color   string `json:"color,omitempty"`
	Orden   int    `json:"orden"`
	AltText string `json:"alt_text,omitempty"`
}

type ProductoResp struct {
	ID          uint           `json:"id"`
	Nombre      string         `json:"nombre"`
	Descripcion string         `json:"descripcion"`
	Precio      float64        `json:"precio"`
	Categoria   string         `json:"categoria"`
	Variantes   []VarianteResp `json:"variantes"`
	Imagenes    []ImagenResp   `json:"imagenes"`
}

type PedidoItemResp struct {
	VarianteID     uint    `json:"variante_id"`
	Descripcion    string  `json:"descripcion"`
	Cantidad       int     `json:"cantidad"`
	PrecioUnitario float64 `json:"precio_unitario"`
	Subtotal       float64 `json:"subtotal"`
}

type PedidoResp struct {
	ID     uint             `json:"id"`
	Estado string           `json:"estado"`
	Total  float64          `json:"total"`
	Fecha  string           `json:"fecha"`
	Items  []PedidoItemResp `json:"items"`
}

type CambiarEstadoRequest struct {
	Estado string `json:"estado" binding:"required"`
}

// --- ABM de productos (admin) ----------------------------------------------

type VarianteRequest struct {
	Talle string `json:"talle" binding:"required"`
	Color string `json:"color" binding:"required"`
	// Sin binding:"required": un stock inicial de 0 es legitimo (producto
	// cargado antes de que llegue la mercaderia) y required lo rechazaria.
	Stock int    `json:"stock"`
	SKU   string `json:"sku"`
}

type ProductoRequest struct {
	Nombre      string            `json:"nombre" binding:"required"`
	Descripcion string            `json:"descripcion"`
	Precio      float64           `json:"precio" binding:"gt=0"`
	Categoria   string            `json:"categoria"`
	Variantes   []VarianteRequest `json:"variantes" binding:"required,min=1,dive"`
}

type ProductoUpdateRequest struct {
	Nombre      string  `json:"nombre" binding:"required"`
	Descripcion string  `json:"descripcion"`
	Precio      float64 `json:"precio" binding:"gt=0"`
	Categoria   string  `json:"categoria"`
	// Puntero para distinguir "no vino en el body" de "vino en false".
	Activo *bool `json:"activo"`
}
