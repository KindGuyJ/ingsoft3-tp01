package repository

import (
	"io"
	"os"
	"path/filepath"
)

// AlmacenDisco guarda los archivos subidos en un directorio del filesystem.
// En el compose ese directorio es /app/uploads, respaldado por el volumen
// nombrado uploads_data.
//
// Vive en repository/ porque un archivo tambien es persistencia: es el mismo
// rol que cumple MySQL para las filas. Satisface services.AlmacenImagenes, asi
// que reemplazarlo por object storage en el TP6 es escribir otro tipo con el
// mismo metodo, sin tocar una linea de services/.
type AlmacenDisco struct{ dir string }

func NuevoAlmacenDisco(dir string) *AlmacenDisco { return &AlmacenDisco{dir: dir} }

func (a *AlmacenDisco) Guardar(nombre string, contenido io.Reader) error {
	// El Dockerfile ya crea /app/uploads, pero corriendo el backend fuera del
	// contenedor (go run ./cmd/api) el directorio puede no existir.
	if err := os.MkdirAll(a.dir, 0o755); err != nil {
		return err
	}

	// filepath.Base es defensivo: el nombre lo genera el service y no trae
	// separadores, pero si alguna vez los trajera no podria escapar del
	// directorio de subidas.
	destino := filepath.Join(a.dir, filepath.Base(nombre))

	// O_EXCL: si el nombre ya existiera, falla en vez de pisar el archivo. Con
	// nombres aleatorios no deberia pasar nunca; que falle ruidosamente es
	// mejor que perder una foto en silencio.
	f, err := os.OpenFile(destino, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, contenido); err != nil {
		// No dejar un archivo a medio escribir: seria una imagen rota servida
		// por r.Static, que es peor que no tener imagen.
		os.Remove(destino)
		return err
	}
	return nil
}
