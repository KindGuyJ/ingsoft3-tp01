package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/dao"
	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/dto"
	dom "github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/errors"
	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/middleware"
)

// ---------------------------------------------------------------------------
// Helpers compartidos por los handlers.
// ---------------------------------------------------------------------------

// idDeRuta lee un parametro numerico de la URL. Un ":id" que no es numero es
// un error del cliente (400), no un 404 ni un 500.
func idDeRuta(c *gin.Context, nombre string) (uint, bool) {
	raw := c.Param(nombre)
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || n == 0 {
		Responder(c, dom.Validacion("el parametro %s debe ser un numero valido", nombre))
		return 0, false
	}
	return uint(n), true
}

// bindJSON parsea el body y traduce el error del validador de Gin a un error
// de dominio, para que la respuesta tenga el mismo formato que todas las demas.
func bindJSON(c *gin.Context, destino any) bool {
	if err := c.ShouldBindJSON(destino); err != nil {
		Responder(c, dom.Validacion("body invalido: %s", err.Error()))
		return false
	}
	return true
}

// usuarioAutenticado devuelve lo que RequiereAuth dejo en el contexto.
// Si el handler esta bien cableado detras del middleware, siempre esta.
func usuarioAutenticado(c *gin.Context) (usuarioID uint, esAdmin bool, ok bool) {
	valor, existe := c.Get(middleware.ClaveUsuarioID)
	id, esUint := valor.(uint)
	if !existe || !esUint {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token invalido"})
		return 0, false, false
	}
	admin, _ := c.Get(middleware.ClaveEsAdmin)
	esAdminBool, _ := admin.(bool)
	return id, esAdminBool, true
}

// ---------------------------------------------------------------------------
// Mapeo dao -> dto.
//
// Vive en controllers porque es parte de "mapear la respuesta". Los services
// devuelven entidades; que se expone de ellas es una decision de la API. Por
// eso, por ejemplo, PasswordHash no aparece en ningun mapper.
// ---------------------------------------------------------------------------

func aUsuarioResp(u *dao.Usuario) dto.UsuarioResp {
	return dto.UsuarioResp{
		ID:       u.ID,
		Email:    u.Email,
		Nombre:   u.Nombre,
		Apellido: u.Apellido,
		EsAdmin:  u.EsAdmin,
	}
}

func aProductoResp(p *dao.Producto) dto.ProductoResp {
	// Slices inicializados: el front recibe [] y no null, asi puede hacer
	// .map() sin chequear.
	variantes := make([]dto.VarianteResp, 0, len(p.Variantes))
	for _, v := range p.Variantes {
		variantes = append(variantes, dto.VarianteResp{
			ID:    v.ID,
			Talle: v.Talle,
			Color: v.Color,
			Stock: v.Stock,
			SKU:   v.SKU,
		})
	}

	imagenes := make([]dto.ImagenResp, 0, len(p.Imagenes))
	for i := range p.Imagenes {
		imagenes = append(imagenes, aImagenResp(&p.Imagenes[i]))
	}

	return dto.ProductoResp{
		ID:          p.ID,
		Nombre:      p.Nombre,
		Descripcion: p.Descripcion,
		Precio:      p.Precio,
		Categoria:   p.Categoria,
		Activo:      p.Activo,
		Variantes:   variantes,
		Imagenes:    imagenes,
	}
}

func aImagenResp(i *dao.Imagen) dto.ImagenResp {
	return dto.ImagenResp{ID: i.ID, URL: i.URL, Color: i.Color, Orden: i.Orden, AltText: i.AltText}
}

func aVarianteResp(v *dao.Variante) dto.VarianteResp {
	return dto.VarianteResp{ID: v.ID, Talle: v.Talle, Color: v.Color, Stock: v.Stock, SKU: v.SKU}
}

func aPedidoResp(p *dao.Pedido) dto.PedidoResp {
	items := make([]dto.PedidoItemResp, 0, len(p.Items))
	for _, i := range p.Items {
		items = append(items, dto.PedidoItemResp{
			VarianteID:     i.VarianteID,
			Descripcion:    i.DescripcionItem,
			Cantidad:       i.Cantidad,
			PrecioUnitario: i.PrecioUnitario,
			Subtotal:       i.Subtotal(),
		})
	}
	return dto.PedidoResp{
		ID:     p.ID,
		Estado: p.Estado,
		Total:  p.Total,
		Fecha:  p.Fecha.Format(time.RFC3339),
		Items:  items,
	}
}
