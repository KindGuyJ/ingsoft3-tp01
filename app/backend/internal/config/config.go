// Package config es el UNICO lugar del backend que lee variables de entorno.
// Esto es lo que hace que la misma imagen sirva en dev, QA y PROD (TP6).
package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Puerto string

	// El commit que esta corriendo. Lo informa /healthz para que el smoke del
	// pipeline pueda comprobar que contesta la version recien desplegada y no
	// la anterior. APP_COMMIT viene horneado en la imagen (ARG COMMIT del
	// Dockerfile, que el pipeline pasa con build-args): desde el TP7 Render
	// ejecuta la imagen y no construye nada, asi que ya no define
	// RENDER_GIT_COMMIT. Vacio en compose: ahi no hace falta.
	Commit string

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
		Commit:     get("APP_COMMIT", ""),
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
	if err := validarHost(c.DBHost); err != nil {
		return nil, err
	}
	return c, nil
}

// validarHost rechaza un DB_HOST que no es solo un host: el error tipico es
// pegar ahi la cadena de conexion entera que da Neon (postgresql://user:pass@...).
//
// Se corta ACA, antes de gorm.Open, por dos motivos: el error que da el driver
// no dice que esta mal (habla de un "%2F" que nadie escribio), y GORM lo
// imprime con la cadena completa, contrasena incluida, en los logs del entorno.
// Por eso el mensaje de este error NUNCA repite el valor recibido.
func validarHost(host string) error {
	if strings.Contains(host, "://") || strings.ContainsAny(host, "@/?") {
		return fmt.Errorf("DB_HOST tiene que ser solo el host (ej: ep-xxx.aws.neon.tech), " +
			"no la cadena de conexion: usuario, contrasena, base y sslmode van en " +
			"DB_USER, DB_PASS, DB_SCHEMA y DB_SSLMODE")
	}
	return nil
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
