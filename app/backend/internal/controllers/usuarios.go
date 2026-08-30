package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/dto"
	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/services"
)

// UsuariosController expone registro y login. Como todos los controllers:
// parsea, delega, mapea. Ninguna regla de negocio vive aca.
type UsuariosController struct {
	svc *services.UsuariosService
}

func NuevoUsuariosController(s *services.UsuariosService) *UsuariosController {
	return &UsuariosController{svc: s}
}

// POST /api/usuarios/registro
func (ct *UsuariosController) Registro(c *gin.Context) {
	var req dto.RegistroRequest
	if !bindJSON(c, &req) {
		return
	}

	u, err := ct.svc.Registrar(services.Registro{
		Email:    req.Email,
		Password: req.Password,
		Nombre:   req.Nombre,
		Apellido: req.Apellido,
	})
	if err != nil {
		Responder(c, err)
		return
	}

	// 201 y el usuario sin token: el registro no loguea. Que el front decida
	// si manda al login o pide las credenciales de nuevo.
	c.JSON(http.StatusCreated, aUsuarioResp(u))
}

// POST /api/usuarios/login
func (ct *UsuariosController) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !bindJSON(c, &req) {
		return
	}

	token, u, err := ct.svc.Login(req.Email, req.Password)
	if err != nil {
		Responder(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.LoginResponse{Token: token, Usuario: aUsuarioResp(u)})
}
