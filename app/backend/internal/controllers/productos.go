package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/dto"
	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/services"
)

type ProductosController struct {
	svc *services.ProductosService
}

func NuevoProductosController(s *services.ProductosService) *ProductosController {
	return &ProductosController{svc: s}
}

// GET /api/productos?categoria=remeras
func (ct *ProductosController) Listar(c *gin.Context) {
	ps, err := ct.svc.Listar(c.Query("categoria"))
	if err != nil {
		Responder(c, err)
		return
	}

	resp := make([]dto.ProductoResp, 0, len(ps))
	for i := range ps {
		resp = append(resp, aProductoResp(&ps[i]))
	}
	c.JSON(http.StatusOK, resp)
}

// GET /api/admin/productos?categoria=remeras (admin)
//
// Cuelga de /admin y no de un parametro tipo ?incluir_inactivos=true sobre el
// endpoint publico: asi el catalogo que ve cualquiera no tiene ninguna forma de
// devolver productos dados de baja, ni siquiera por error de tipeo. La
// autorizacion la pone el grupo de rutas, no un if adentro del handler.
func (ct *ProductosController) ListarParaAdmin(c *gin.Context) {
	ps, err := ct.svc.ListarTodos(c.Query("categoria"))
	if err != nil {
		Responder(c, err)
		return
	}

	resp := make([]dto.ProductoResp, 0, len(ps))
	for i := range ps {
		resp = append(resp, aProductoResp(&ps[i]))
	}
	c.JSON(http.StatusOK, resp)
}

// GET /api/productos/:id
//
// Es la ruta PUBLICA, por eso llama a VerDetallePublico: un producto dado de
// baja responde 404 aunque alguien tenga el link guardado.
func (ct *ProductosController) VerDetalle(c *gin.Context) {
	id, ok := idDeRuta(c, "id")
	if !ok {
		return
	}

	p, err := ct.svc.VerDetallePublico(id)
	if err != nil {
		Responder(c, err)
		return
	}
	c.JSON(http.StatusOK, aProductoResp(p))
}

// POST /api/productos (admin)
func (ct *ProductosController) Crear(c *gin.Context) {
	var req dto.ProductoRequest
	if !bindJSON(c, &req) {
		return
	}

	p, err := ct.svc.Crear(services.ProductoNuevo{
		Nombre:      req.Nombre,
		Descripcion: req.Descripcion,
		Precio:      req.Precio,
		Categoria:   req.Categoria,
		Variantes:   aVariantesNuevas(req.Variantes),
	})
	if err != nil {
		Responder(c, err)
		return
	}
	c.JSON(http.StatusCreated, aProductoResp(p))
}

// PUT /api/productos/:id (admin)
func (ct *ProductosController) Editar(c *gin.Context) {
	id, ok := idDeRuta(c, "id")
	if !ok {
		return
	}

	var req dto.ProductoUpdateRequest
	if !bindJSON(c, &req) {
		return
	}

	p, err := ct.svc.Editar(id, services.ProductoEditado{
		Nombre:      req.Nombre,
		Descripcion: req.Descripcion,
		Precio:      req.Precio,
		Categoria:   req.Categoria,
		Activo:      req.Activo,
	})
	if err != nil {
		Responder(c, err)
		return
	}
	c.JSON(http.StatusOK, aProductoResp(p))
}

// POST /api/productos/:id/variantes (admin)
func (ct *ProductosController) AgregarVariante(c *gin.Context) {
	id, ok := idDeRuta(c, "id")
	if !ok {
		return
	}

	var req dto.VarianteRequest
	if !bindJSON(c, &req) {
		return
	}

	v, err := ct.svc.AgregarVariante(id, services.VarianteNueva{
		Talle: req.Talle,
		Color: req.Color,
		Stock: req.Stock,
		SKU:   req.SKU,
	})
	if err != nil {
		Responder(c, err)
		return
	}
	c.JSON(http.StatusCreated, aVarianteResp(v))
}

// PATCH /api/productos/:id/variantes/:varianteId (admin)
func (ct *ProductosController) ActualizarStock(c *gin.Context) {
	productoID, ok := idDeRuta(c, "id")
	if !ok {
		return
	}
	varianteID, ok := idDeRuta(c, "varianteId")
	if !ok {
		return
	}

	var req dto.StockRequest
	if !bindJSON(c, &req) {
		return
	}

	v, err := ct.svc.ActualizarStock(productoID, varianteID, *req.Stock)
	if err != nil {
		Responder(c, err)
		return
	}
	c.JSON(http.StatusOK, aVarianteResp(v))
}

func aVariantesNuevas(reqs []dto.VarianteRequest) []services.VarianteNueva {
	out := make([]services.VarianteNueva, 0, len(reqs))
	for _, r := range reqs {
		out = append(out, services.VarianteNueva{
			Talle: r.Talle,
			Color: r.Color,
			Stock: r.Stock,
			SKU:   r.SKU,
		})
	}
	return out
}
