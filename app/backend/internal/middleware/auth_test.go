package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Estos tests no levantan el servidor: arman un router de Gin en memoria con
// el middleware adelante de un handler que contesta 200, y le mandan pedidos
// con httptest. Si el pedido llega al handler, el middleware lo dejo pasar.

const secretoTest = "secreto-de-test"

func init() { gin.SetMode(gin.TestMode) }

// routerCon arma GET /privado protegido por los middlewares dados. El handler
// devuelve lo que el middleware dejo en el contexto, para poder verificarlo.
func routerCon(mws ...gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	handlers := append(mws, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"uid": c.GetUint(ClaveUsuarioID),
			"adm": c.GetBool(ClaveEsAdmin),
		})
	})
	r.GET("/privado", handlers...)
	return r
}

func pedir(r *gin.Engine, authorization string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/privado", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func tokenValido(t *testing.T, usuarioID uint, esAdmin bool) string {
	t.Helper()
	tok, err := GenerarToken(secretoTest, usuarioID, esAdmin, time.Hour)
	if err != nil {
		t.Fatalf("no se pudo generar el token: %v", err)
	}
	return tok
}

func TestRequiereAuth_TokenValidoPasaYDejaElUsuarioEnElContexto(t *testing.T) {
	// Arrange
	r := routerCon(RequiereAuth(secretoTest))

	// Act
	w := pedir(r, "Bearer "+tokenValido(t, 42, false))

	// Assert
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", w.Code)
	}
	if want := `{"adm":false,"uid":42}`; w.Body.String() != want {
		t.Errorf("contexto = %s, se esperaba %s", w.Body.String(), want)
	}
}

// Todas las formas de NO estar autenticado terminan en 401, y ninguna llega
// al handler.
func TestRequiereAuth_Rechazos(t *testing.T) {
	vencido, _ := GenerarToken(secretoTest, 1, false, -time.Minute)
	otroSecreto, _ := GenerarToken("otro-secreto", 1, true, time.Hour)
	// alg=none: un token sin firma. Es el ataque clasico contra JWT, y lo
	// que frena el chequeo del metodo de firma en RequiereAuth.
	sinFirma, _ := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{UsuarioID: 1, EsAdmin: true}).
		SignedString(jwt.UnsafeAllowNoneSignatureType)

	casos := []struct {
		nombre        string
		authorization string
	}{
		{"sin header", ""},
		{"sin el prefijo Bearer", tokenValido(t, 1, false)},
		{"token que no es un JWT", "Bearer cualquier-cosa"},
		{"token vencido", "Bearer " + vencido},
		{"firmado con otro secreto", "Bearer " + otroSecreto},
		{"alg none sin firma", "Bearer " + sinFirma},
	}
	r := routerCon(RequiereAuth(secretoTest))

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			w := pedir(r, c.authorization)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, se esperaba 401", w.Code)
			}
		})
	}
}

func TestRequiereAdmin(t *testing.T) {
	casos := []struct {
		nombre  string
		esAdmin bool
		status  int
	}{
		{"un cliente no entra", false, http.StatusForbidden},
		{"un admin entra", true, http.StatusOK},
	}
	r := routerCon(RequiereAuth(secretoTest), RequiereAdmin())

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			w := pedir(r, "Bearer "+tokenValido(t, 1, c.esAdmin))
			if w.Code != c.status {
				t.Errorf("status = %d, se esperaba %d", w.Code, c.status)
			}
		})
	}
}

// RequiereAdmin se encadena despues de RequiereAuth. Si alguien lo monta solo,
// en el contexto no hay rol: tiene que fallar cerrado (403), no abierto.
func TestRequiereAdmin_SinRequiereAuthAdelanteRechaza(t *testing.T) {
	r := routerCon(RequiereAdmin())

	w := pedir(r, "")

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, se esperaba 403", w.Code)
	}
}
