package services

import (
	"testing"

	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/dao"
	dom "github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/errors"
)

// Fake en memoria del repo de usuarios. Igual que el de pedidos: sin MySQL.
type fakeUsuarioRepo struct {
	porEmail map[string]*dao.Usuario
	proximo  uint
}

func nuevoFakeUsuarioRepo() *fakeUsuarioRepo {
	return &fakeUsuarioRepo{porEmail: map[string]*dao.Usuario{}, proximo: 1}
}

func (f *fakeUsuarioRepo) BuscarPorEmail(email string) (*dao.Usuario, error) {
	u, ok := f.porEmail[email]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (f *fakeUsuarioRepo) Crear(u *dao.Usuario) error {
	u.ID = f.proximo
	f.proximo++
	f.porEmail[u.Email] = u
	return nil
}

func setupUsuarios() (*UsuariosService, *fakeUsuarioRepo) {
	repo := nuevoFakeUsuarioRepo()
	// Generador de tokens de mentira: al service solo le importa que le den
	// una cadena. Quien firma de verdad es middleware/, cableado en main.go.
	return NuevoUsuariosService(repo, func(id uint, admin bool) (string, error) {
		return "token-de-prueba", nil
	}), repo
}

func registroValido() Registro {
	return Registro{Email: "Ana@Ejemplo.com", Password: "unaClave123", Nombre: "Ana", Apellido: "Perez"}
}

// --- Regla 8: email unico ---------------------------------------------------

func TestRegistrar_EmailUnico(t *testing.T) {
	s, _ := setupUsuarios()

	if _, err := s.Registrar(registroValido()); err != nil {
		t.Fatalf("el primer registro fallo: %v", err)
	}

	// El mismo email con otra combinacion de mayusculas sigue siendo el mismo.
	repetido := registroValido()
	repetido.Email = "ana@ejemplo.com"
	_, err := s.Registrar(repetido)
	if err == nil {
		t.Fatal("se esperaba error por email duplicado")
	}
	if k := kindDe(t, err); k != dom.KindConflicto {
		t.Errorf("kind = %v, se esperaba KindConflicto", k)
	}
}

func TestRegistrar_NormalizaYNoGuardaLaPassword(t *testing.T) {
	s, repo := setupUsuarios()

	u, err := s.Registrar(registroValido())
	if err != nil {
		t.Fatalf("registro fallo: %v", err)
	}
	if u.Email != "ana@ejemplo.com" {
		t.Errorf("email = %q, se esperaba normalizado a minusculas", u.Email)
	}
	if repo.porEmail["ana@ejemplo.com"].PasswordHash == "unaClave123" {
		t.Fatal("la contrasena se guardo en texto plano")
	}
	if u.EsAdmin {
		t.Error("el registro publico nunca puede crear un admin")
	}
}

// --- Regla 8: password minima y email valido --------------------------------

func TestRegistrar_Validaciones(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*Registro)
	}{
		{"password corta", func(r *Registro) { r.Password = "corta" }},
		{"email sin arroba", func(r *Registro) { r.Email = "ana-ejemplo.com" }},
		{"email vacio", func(r *Registro) { r.Email = "  " }},
		{"nombre vacio", func(r *Registro) { r.Nombre = "" }},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			s, _ := setupUsuarios()
			in := registroValido()
			c.mutar(&in)

			_, err := s.Registrar(in)
			if err == nil {
				t.Fatal("se esperaba un error de validacion")
			}
			if k := kindDe(t, err); k != dom.KindValidacion {
				t.Errorf("kind = %v, se esperaba KindValidacion", k)
			}
		})
	}
}

// --- Login ------------------------------------------------------------------

func TestLogin_Correcto(t *testing.T) {
	s, _ := setupUsuarios()
	if _, err := s.Registrar(registroValido()); err != nil {
		t.Fatalf("registro fallo: %v", err)
	}

	token, u, err := s.Login("ana@ejemplo.com", "unaClave123")
	if err != nil {
		t.Fatalf("login fallo: %v", err)
	}
	if token == "" {
		t.Error("no se emitio token")
	}
	if u.Email != "ana@ejemplo.com" {
		t.Errorf("usuario = %q, se esperaba ana@ejemplo.com", u.Email)
	}
}

func TestLogin_PasswordIncorrecta(t *testing.T) {
	s, _ := setupUsuarios()
	if _, err := s.Registrar(registroValido()); err != nil {
		t.Fatalf("registro fallo: %v", err)
	}

	_, _, err := s.Login("ana@ejemplo.com", "otraClave123")
	if err == nil {
		t.Fatal("se esperaba error de credenciales")
	}
	if k := kindDe(t, err); k != dom.KindNoAutorizado {
		t.Errorf("kind = %v, se esperaba KindNoAutorizado", k)
	}
}

// Un email que no existe tiene que dar el MISMO error que una password mal
// puesta: si no, la API se convierte en un verificador de emails registrados.
func TestLogin_EmailInexistenteNoSeDistingue(t *testing.T) {
	s, _ := setupUsuarios()
	if _, err := s.Registrar(registroValido()); err != nil {
		t.Fatalf("registro fallo: %v", err)
	}

	_, _, errInexistente := s.Login("nadie@ejemplo.com", "unaClave123")
	_, _, errPassword := s.Login("ana@ejemplo.com", "otraClave123")

	if errInexistente == nil || errPassword == nil {
		t.Fatal("ambos casos tienen que fallar")
	}
	if errInexistente.Error() != errPassword.Error() {
		t.Errorf("los mensajes difieren (%q vs %q); eso filtra que emails existen",
			errInexistente, errPassword)
	}
}

// --- Regla: un usuario inactivo no loguea -----------------------------------

func TestLogin_UsuarioInactivo(t *testing.T) {
	s, repo := setupUsuarios()
	if _, err := s.Registrar(registroValido()); err != nil {
		t.Fatalf("registro fallo: %v", err)
	}
	repo.porEmail["ana@ejemplo.com"].Activo = false

	_, _, err := s.Login("ana@ejemplo.com", "unaClave123")
	if err == nil {
		t.Fatal("un usuario dado de baja no deberia poder loguear")
	}
	if k := kindDe(t, err); k != dom.KindNoAutorizado {
		t.Errorf("kind = %v, se esperaba KindNoAutorizado", k)
	}
}
