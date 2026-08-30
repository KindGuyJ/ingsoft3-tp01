package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/dto"
	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/services"
)

type PedidosController struct {
	svc *services.PedidosService
}

func NuevoPedidosController(s *services.PedidosService) *PedidosController {
	return &PedidosController{svc: s}
}

// POST /api/pedidos (autenticado) — el checkout.
//
// El carrito llega entero en el body: no vive en la base. El usuario NO viene
// del body sino del token; si viniera del body, cualquiera podria comprar a
// nombre de otro.
func (ct *PedidosController) Checkout(c *gin.Context) {
	usuarioID, _, ok := usuarioAutenticado(c)
	if !ok {
		return
	}

	var req dto.CheckoutRequest
	if !bindJSON(c, &req) {
		return
	}

	carrito := make([]services.ItemCarrito, 0, len(req.Items))
	for _, i := range req.Items {
		carrito = append(carrito, services.ItemCarrito{VarianteID: i.VarianteID, Cantidad: i.Cantidad})
	}

	p, err := ct.svc.Checkout(usuarioID, carrito)
	if err != nil {
		Responder(c, err)
		return
	}
	c.JSON(http.StatusCreated, aPedidoResp(p))
}

// GET /api/pedidos (autenticado) — solo los propios (regla 7).
func (ct *PedidosController) MisPedidos(c *gin.Context) {
	usuarioID, _, ok := usuarioAutenticado(c)
	if !ok {
		return
	}

	ps, err := ct.svc.ListarDeUsuario(usuarioID)
	if err != nil {
		Responder(c, err)
		return
	}

	resp := make([]dto.PedidoResp, 0, len(ps))
	for i := range ps {
		resp = append(resp, aPedidoResp(&ps[i]))
	}
	c.JSON(http.StatusOK, resp)
}

// POST /api/pedidos/:id/cancelar (autenticado)
func (ct *PedidosController) Cancelar(c *gin.Context) {
	usuarioID, esAdmin, ok := usuarioAutenticado(c)
	if !ok {
		return
	}
	id, ok := idDeRuta(c, "id")
	if !ok {
		return
	}

	if err := ct.svc.Cancelar(id, usuarioID, esAdmin); err != nil {
		Responder(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// PATCH /api/pedidos/:id/estado (admin)
func (ct *PedidosController) CambiarEstado(c *gin.Context) {
	id, ok := idDeRuta(c, "id")
	if !ok {
		return
	}

	var req dto.CambiarEstadoRequest
	if !bindJSON(c, &req) {
		return
	}

	if err := ct.svc.CambiarEstado(id, req.Estado); err != nil {
		Responder(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
