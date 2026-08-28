package services

import (
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/dao"
	dom "github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/errors"
)

// LargoMinimoPassword es la regla 8 del TP: sin esto el registro acepta "1234".
// bcrypt ademas ignora todo lo que pase de 72 bytes, asi que tambien hay tope.
const (
	LargoMinimoPassword = 8
	LargoMaximoPassword = 72
)

// UsuarioRepo es lo unico que este service necesita saber de la persistencia.
type UsuarioRepo interface {
	BuscarPorEmail(email string) (*dao.Usuario, error)
	Crear(u *dao.Usuario) error
}

// GeneradorToken firma un JWT para un usuario.
//
// Se inyecta como funcion en lugar de importar middleware/ porque ese paquete
// arrastra Gin: si el service lo importara, el paquete de reglas de negocio
// terminaria dependiendo del framework HTTP. Quien conoce el secreto y la
// duracion es main.go; aca solo se pide "firmame un token para este usuario".
type GeneradorToken func(usuarioID uint, esAdmin bool) (string, error)

type UsuariosService struct {
	usuarios     UsuarioRepo
	generarToken GeneradorToken
}

func NuevoUsuariosService(u UsuarioRepo, g GeneradorToken) *UsuariosService {
	return &UsuariosService{usuarios: u, generarToken: g}
}

// Registro son los datos que llegan del cliente para dar de alta un usuario.
type Registro struct {
	Email    string
	Password string
	Nombre   string
	Apellido string
}

// Registrar crea un usuario nuevo.
//
// Reglas (regla 8 del TP):
//  1. El email tiene que ser sintacticamente valido y unico.
//  2. La password tiene un largo minimo.
//  3. La password NUNCA se guarda: se guarda su hash bcrypt.
func (s *UsuariosService) Registrar(in Registro) (*dao.Usuario, error) {
	email, err := normalizarEmail(in.Email)
	if err != nil {
		return nil, err
	}
	if len(in.Password) < LargoMinimoPassword {
		return nil, dom.Validacion("la contrasena debe tener al menos %d caracteres", LargoMinimoPassword)
	}
	if len(in.Password) > LargoMaximoPassword {
		return nil, dom.Validacion("la contrasena no puede superar los %d caracteres", LargoMaximoPassword)
	}

	nombre := strings.TrimSpace(in.Nombre)
	apellido := strings.TrimSpace(in.Apellido)
	if nombre == "" || apellido == "" {
		return nil, dom.Validacion("nombre y apellido son obligatorios")
	}

	// Email unico. Se consulta antes de insertar para poder devolver un 409
	// entendible en vez del error de indice unico de MySQL.
	existente, err := s.usuarios.BuscarPorEmail(email)
	if err != nil {
		return nil, dom.Interno("no se pudo verificar el email", err)
	}
	if existente != nil {
		return nil, dom.Conflicto("ya existe un usuario con el email %s", email)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, dom.Interno("no se pudo hashear la contrasena", err)
	}

	u := &dao.Usuario{
		Email:        email,
		PasswordHash: string(hash),
		Nombre:       nombre,
		Apellido:     apellido,
		EsAdmin:      false, // el admin no se crea por la API publica: lo hace el seed
		Activo:       true,
	}
	if err := s.usuarios.Crear(u); err != nil {
		return nil, dom.Interno("no se pudo crear el usuario", err)
	}
	return u, nil
}

// Login verifica las credenciales y devuelve el token junto con el usuario.
//
// Los tres motivos de rechazo (email inexistente, hash que no coincide, usuario
// inactivo) devuelven el MISMO mensaje a proposito: distinguirlos le permitiria
// a cualquiera averiguar que emails estan registrados.
func (s *UsuariosService) Login(email, password string) (string, *dao.Usuario, error) {
	emailNorm, err := normalizarEmail(email)
	if err != nil {
		return "", nil, dom.NoAutorizado("email o contrasena incorrectos")
	}

	u, err := s.usuarios.BuscarPorEmail(emailNorm)
	if err != nil {
		return "", nil, dom.Interno("no se pudo leer el usuario", err)
	}
	if u == nil {
		// Se compara igual contra un hash descartable para que el tiempo de
		// respuesta no delate si el email existe o no.
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$invalidinvalidinvalidinvalidinvalidinvalidinvalidinvalidinvalid"), []byte(password))
		return "", nil, dom.NoAutorizado("email o contrasena incorrectos")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", nil, dom.NoAutorizado("email o contrasena incorrectos")
	}
	if !u.Activo {
		return "", nil, dom.NoAutorizado("email o contrasena incorrectos")
	}

	token, err := s.generarToken(u.ID, u.EsAdmin)
	if err != nil {
		return "", nil, dom.Interno("no se pudo emitir el token", err)
	}
	return token, u, nil
}

// normalizarEmail deja el email en minusculas y sin espacios, y valida la
// sintaxis con net/mail (libreria estandar: nada de regex casera).
func normalizarEmail(email string) (string, error) {
	e := strings.ToLower(strings.TrimSpace(email))
	if e == "" {
		return "", dom.Validacion("el email es obligatorio")
	}
	if _, err := mail.ParseAddress(e); err != nil {
		return "", dom.Validacion("el email %q no es valido", email)
	}
	return e, nil
}
