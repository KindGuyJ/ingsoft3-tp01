// Package errors define los errores de dominio. Los services devuelven ESTOS,
// nunca codigos HTTP ni gin.H. El mapeo a HTTP se hace en los controllers.
package errors

import "fmt"

type Kind int

const (
	KindValidacion   Kind = iota // 400
	KindNoAutorizado             // 401
	KindProhibido                // 403
	KindNoEncontrado             // 404
	KindConflicto                // 409
	KindInterno                  // 500
)

type DomainError struct {
	Kind    Kind
	Mensaje string
	Causa   error
}

func (e *DomainError) Error() string {
	if e.Causa != nil {
		return fmt.Sprintf("%s: %v", e.Mensaje, e.Causa)
	}
	return e.Mensaje
}

func (e *DomainError) Unwrap() error { return e.Causa }

func Validacion(msg string, args ...any) *DomainError {
	return &DomainError{Kind: KindValidacion, Mensaje: fmt.Sprintf(msg, args...)}
}
func NoAutorizado(msg string) *DomainError {
	return &DomainError{Kind: KindNoAutorizado, Mensaje: msg}
}
func Prohibido(msg string) *DomainError {
	return &DomainError{Kind: KindProhibido, Mensaje: msg}
}
func NoEncontrado(msg string, args ...any) *DomainError {
	return &DomainError{Kind: KindNoEncontrado, Mensaje: fmt.Sprintf(msg, args...)}
}
func Conflicto(msg string, args ...any) *DomainError {
	return &DomainError{Kind: KindConflicto, Mensaje: fmt.Sprintf(msg, args...)}
}
func Interno(msg string, causa error) *DomainError {
	return &DomainError{Kind: KindInterno, Mensaje: msg, Causa: causa}
}
