// Package config es el UNICO lugar del backend que lee variables de entorno.
// Esto es lo que hace que la misma imagen sirva en dev, QA y PROD (TP6).
package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Puerto string

	DBHost   string
	DBPort   string
	DBUser   string
	DBPass   string
	DBSchema string
	// En el compose la base esta en la misma red y no usa TLS ("disable");
	// Neon (TP6) solo acepta conexiones cifradas ("require").
	DBSSLMode string

	JWTSecret string
	// Cuanto vale un token antes de vencer. Por entorno: en dev conviene
	// largo para no relogearse todo el tiempo, en PROD corto.
	DuracionToken time.Duration

	// Regla de negocio parametrizable: compras por encima de este monto
	// no pagan envio. Va por entorno para poder testear el borde sin recompilar.
	UmbralEnvioGratis float64
	CostoEnvio        float64

	UploadsDir string
	// Tamano maximo de una imagen subida por el admin, en bytes.
	//
	// Va por entorno porque tiene que quedar por DEBAJO del client_max_body_size
	// de nginx: si nginx corta primero, el cliente recibe un 413 crudo en vez
	// del 400 explicado del backend. Ver frontend/default.conf.template.
	MaxImagenBytes int64
}

func Cargar() (*Config, error) {
	c := &Config{
		Puerto:     get("PORT", "8080"),
		DBHost:     get("DB_HOST", "localhost"),
		DBPort:     get("DB_PORT", "5432"),
		DBUser:     get("DB_USER", "tienda"),
		DBPass:     get("DB_PASS", ""),
		DBSchema:   get("DB_SCHEMA", "tienda"),
		DBSSLMode:  get("DB_SSLMODE", "disable"),
		JWTSecret:  get("JWT_SECRET", ""),
		UploadsDir: get("UPLOADS_DIR", "/app/uploads"),
	}

	horas, err := getFloat("JWT_HORAS", 24)
	if err != nil {
		return nil, err
	}
	if horas <= 0 {
		return nil, fmt.Errorf("JWT_HORAS debe ser mayor a cero, llego %v", horas)
	}
	c.DuracionToken = time.Duration(horas * float64(time.Hour))

	if c.UmbralEnvioGratis, err = getFloat("UMBRAL_ENVIO_GRATIS", 50000); err != nil {
		return nil, err
	}
	if c.CostoEnvio, err = getFloat("COSTO_ENVIO", 5000); err != nil {
		return nil, err
	}

	mb, err := getFloat("MAX_IMAGEN_MB", 5)
	if err != nil {
		return nil, err
	}
	if mb <= 0 {
		return nil, fmt.Errorf("MAX_IMAGEN_MB debe ser mayor a cero, llego %v", mb)
	}
	c.MaxImagenBytes = int64(mb * 1024 * 1024)

	// Fallar temprano y ruidoso es mejor que arrancar con un secreto vacio
	// y firmar tokens que cualquiera puede falsificar.
	if c.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET es obligatorio y esta vacio (revisa tu .env)")
	}
	return c, nil
}

// DSN arma la cadena de conexion de Postgres, en formato URL.
//
// Usuario y contrasena pasan por url.UserPassword para que se escapen: una
// contrasena con '@' o '/' romperia la URL si se pegara tal cual.
func (c *Config) DSN() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.DBUser, c.DBPass),
		Host:     net.JoinHostPort(c.DBHost, c.DBPort),
		Path:     c.DBSchema,
		RawQuery: url.Values{"sslmode": {c.DBSSLMode}}.Encode(),
	}
	return u.String()
}

func get(clave, porDefecto string) string {
	if v := os.Getenv(clave); v != "" {
		return v
	}
	return porDefecto
}

func getFloat(clave string, porDefecto float64) (float64, error) {
	v := os.Getenv(clave)
	if v == "" {
		return porDefecto, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("la variable %s no es un numero valido: %q", clave, v)
	}
	return f, nil
}
