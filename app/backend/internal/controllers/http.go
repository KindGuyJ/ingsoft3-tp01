// Package controllers traduce entre HTTP y los services.
// Ningun handler tiene logica de negocio: parsea, delega, mapea la respuesta.
package controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	dom "github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/errors"
)

// Responder es el UNICO lugar donde un error de dominio se convierte en status
// HTTP. Si aparece un switch de errores en otro archivo, algo se desacomodo.
func Responder(c *gin.Context, err error) {
	var de *dom.DomainError
	if !errors.As(err, &de) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno"})
		return
	}

	status := map[dom.Kind]int{
		dom.KindValidacion:   http.StatusBadRequest,
		dom.KindNoAutorizado: http.StatusUnauthorized,
		dom.KindProhibido:    http.StatusForbidden,
		dom.KindNoEncontrado: http.StatusNotFound,
		dom.KindConflicto:    http.StatusConflict,
		dom.KindInterno:      http.StatusInternalServerError,
	}[de.Kind]

	// Los errores internos no exponen la causa al cliente: puede filtrar
	// nombres de tablas, rutas o detalles del driver.
	if de.Kind == dom.KindInterno {
		c.JSON(status, gin.H{"error": "error interno"})
		return
	}
	c.JSON(status, gin.H{"error": de.Mensaje})
}
