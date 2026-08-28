// Package config es el UNICO lugar del backend que lee variables de entorno.
// Esto es lo que hace que la misma imagen sirva en dev, QA y PROD (TP6).
package config

import (
	"fmt"
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

	JWTSecret string
	// Cuanto vale un token antes de vencer. Por entorno: en dev conviene
	// largo para no relogearse todo el tiempo, en PROD corto.
	DuracionToken time.Duration

	// Regla de negocio parametrizable: compras por encima de este monto
	// no pagan envio. Va por entorno para poder testear el borde sin recompilar.
	UmbralEnvioGratis float64
	CostoEnvio        float64

	UploadsDir string
}

func Cargar() (*Config, error) {
	c := &Config{
		Puerto:     get("PORT", "8080"),
		DBHost:     get("DB_HOST", "localhost"),
		DBPort:     get("DB_PORT", "3306"),
		DBUser:     get("DB_USER", "root"),
		DBPass:     get("DB_PASS", ""),
		DBSchema:   get("DB_SCHEMA", "tienda"),
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

	// Fallar temprano y ruidoso es mejor que arrancar con un secreto vacio
	// y firmar tokens que cualquiera puede falsificar.
	if c.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET es obligatorio y esta vacio (revisa tu .env)")
	}
	return c, nil
}

// DSN arma la cadena de conexion de MySQL.
func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.DBUser, c.DBPass, c.DBHost, c.DBPort, c.DBSchema)
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
