package config

import (
	"strings"
	"testing"
)

// La cadena que da el boton Connect de Neon, pegada en DB_HOST: el error real
// del primer deploy a Render (con una contrasena de mentira).
const cadenaNeon = "postgresql://neondb_owner:secreto-de-prueba@ep-xxx.sa-east-1.aws.neon.tech/qa?sslmode=require"

func TestCargarRechazaLaCadenaDeConexionEnDBHost(t *testing.T) {
	t.Setenv("JWT_SECRET", "x")
	t.Setenv("DB_HOST", cadenaNeon)

	_, err := Cargar()
	if err == nil {
		t.Fatal("se esperaba un error con la cadena de conexion en DB_HOST")
	}
	// Lo que importa: el error se va a ver en los logs del entorno, y no puede
	// llevarse la contrasena con el.
	if strings.Contains(err.Error(), "secreto-de-prueba") {
		t.Errorf("el error filtra la contrasena: %q", err)
	}
	if !strings.Contains(err.Error(), "DB_HOST") {
		t.Errorf("el error no dice que variable esta mal: %q", err)
	}
}

func TestValidarHost(t *testing.T) {
	casos := []struct {
		host   string
		valido bool
	}{
		{"db", true}, // el nombre del servicio en compose
		{"localhost", true},
		{"ep-round-dream-b6dtkwum.c-2.sa-east-1.aws.neon.tech", true},
		{cadenaNeon, false},
		{"neondb_owner:pass@ep-xxx.aws.neon.tech", false}, // sin el esquema
		{"ep-xxx.aws.neon.tech/qa", false},                // con la base
		{"ep-xxx.aws.neon.tech?sslmode=require", false},   // con parametros
	}
	for _, c := range casos {
		if err := validarHost(c.host); (err == nil) != c.valido {
			t.Errorf("validarHost(%q): valido=%v, se esperaba %v", c.host, err == nil, c.valido)
		}
	}
}
