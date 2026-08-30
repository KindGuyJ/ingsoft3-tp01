// Package middleware tiene el guard de JWT y el de rol admin.
package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	ClaveUsuarioID = "usuarioID"
	ClaveEsAdmin   = "esAdmin"
)

type Claims struct {
	UsuarioID uint `json:"uid"`
	EsAdmin   bool `json:"adm"`
	jwt.RegisteredClaims
}

// GenerarToken firma un JWT para el usuario dado.
func GenerarToken(secret string, usuarioID uint, esAdmin bool, duracion time.Duration) (string, error) {
	claims := Claims{
		UsuarioID: usuarioID,
		EsAdmin:   esAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duracion)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// RequiereAuth valida el token y deja usuarioID/esAdmin en el contexto.
func RequiereAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "falta el token"})
			return
		}
		raw := strings.TrimPrefix(header, "Bearer ")

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
			// Verificar el metodo de firma no es paranoia: sin esto, un atacante
			// puede mandar un token con alg=none y entrar sin credenciales.
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token invalido"})
			return
		}

		c.Set(ClaveUsuarioID, claims.UsuarioID)
		c.Set(ClaveEsAdmin, claims.EsAdmin)
		c.Next()
	}
}

// RequiereAdmin se encadena DESPUES de RequiereAuth.
func RequiereAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if esAdmin, ok := c.Get(ClaveEsAdmin); !ok || esAdmin != true {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "se requiere rol admin"})
			return
		}
		c.Next()
	}
}
