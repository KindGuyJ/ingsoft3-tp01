package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	dom "github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/errors"
	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/services"
)

type ImagenesController struct {
	svc *services.ImagenesService
}

func NuevoImagenesController(s *services.ImagenesService) *ImagenesController {
	return &ImagenesController{svc: s}
}

// POST /api/productos/:id/imagenes (admin)
//
// Es el unico endpoint que NO recibe JSON: llega un multipart/form-data con el
// archivo en el campo "archivo" y el resto como campos de texto.
//
// El handler desarma el request y nada mas: no valida extension, ni tamano, ni
// contenido. Todo eso son reglas y viven en el service.
func (ct *ImagenesController) Subir(c *gin.Context) {
	id, ok := idDeRuta(c, "id")
	if !ok {
		return
	}

	cabecera, err := c.FormFile("archivo")
	if err != nil {
		Responder(c, dom.Validacion("falta el archivo en el campo \"archivo\": %s", err.Error()))
		return
	}

	archivo, err := cabecera.Open()
	if err != nil {
		Responder(c, dom.Interno("no se pudo abrir el archivo subido", err))
		return
	}
	defer archivo.Close()

	orden, err := ordenDeFormulario(c.PostForm("orden"))
	if err != nil {
		Responder(c, err)
		return
	}

	img, err := ct.svc.Agregar(id, services.ImagenNueva{
		NombreArchivo: cabecera.Filename,
		// Size lo calcula mime/multipart al leer la parte: no es un dato que
		// el cliente declare y pueda mentir.
		Tamanio:   cabecera.Size,
		Contenido: archivo,
		Color:     c.PostForm("color"),
		Orden:     orden,
		AltText:   c.PostForm("alt_text"),
	})
	if err != nil {
		Responder(c, err)
		return
	}
	c.JSON(http.StatusCreated, aImagenResp(img))
}

// ordenDeFormulario traduce el campo de texto a int. Vacio = 0 = principal,
// que es el default razonable: la primera foto que sube el admin es la de tapa.
func ordenDeFormulario(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, dom.Validacion("el campo orden tiene que ser un numero, llego %q", raw)
	}
	return n, nil
}
