package services

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/dao"
	dom "github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/errors"
)

// MaxImagenesPorProducto es una regla de dominio, no configuracion: no cambia
// entre dev, QA y PROD. Un producto con treinta fotos es un error de carga.
const MaxImagenesPorProducto = 8

// PrefijoURLSubidas es el prefijo publico bajo el que main.go monta el
// directorio de subidas: r.Static("/uploads", cfg.UploadsDir).
//
// Lo que se guarda en la base es una URL RELATIVA, igual que las estaticas del
// seed ("/productos/x.jpg"). Nunca un host: la misma fila tiene que servir en
// dev, QA y PROD sin reescribirse (TP6).
const PrefijoURLSubidas = "/uploads/"

// firmasPorExtension mapea cada extension permitida a los bytes con los que
// TIENE que empezar el archivo.
//
// Validar la extension sola no alcanza: la elige quien sube. Un .exe renombrado
// a .jpg pasaria el filtro y quedaria servido por r.Static. Los primeros bytes,
// en cambio, no se renombran.
var firmasPorExtension = map[string][]byte{
	".jpg":  {0xFF, 0xD8, 0xFF},
	".jpeg": {0xFF, 0xD8, 0xFF},
	".png":  {0x89, 0x50, 0x4E, 0x47},
}

// ImagenRepo es lo que la subida necesita de la persistencia relacional.
type ImagenRepo interface {
	Crear(i *dao.Imagen) error
}

// ProductoLectorRepo es la parte de ProductoRepo que la subida usa: leer el
// producto para saber si existe y que imagenes ya tiene. Interfaz chica a
// proposito, igual que VarianteAltaRepo.
type ProductoLectorRepo interface {
	BuscarPorID(id uint) (*dao.Producto, error)
}

// AlmacenImagenes es el destino fisico del archivo.
//
// Es la pieza clave del diseno: el service NO sabe si atras hay un disco, un
// volumen de Docker o un bucket. En el TP6, cuando haya que decidir que pasa
// con las subidas entre deploys, se escribe otra implementacion de esta
// interfaz y services/ no se toca. Y en los tests se le pasa una en memoria,
// que es lo que permite testear las reglas sin escribir un solo archivo.
type AlmacenImagenes interface {
	Guardar(nombre string, contenido io.Reader) error
}

type ImagenesService struct {
	productos ProductoLectorRepo
	imagenes  ImagenRepo
	almacen   AlmacenImagenes
	maxBytes  int64
}

func NuevoImagenesService(p ProductoLectorRepo, i ImagenRepo, a AlmacenImagenes, maxBytes int64) *ImagenesService {
	return &ImagenesService{productos: p, imagenes: i, almacen: a, maxBytes: maxBytes}
}

// ImagenNueva es la subida de una foto.
//
// No hay ningun tipo de net/http ni de mime/multipart aca: el controller
// desarma el request y pasa datos planos mas un io.Reader. Si este struct
// tuviera un *multipart.FileHeader, services/ dejaria de poder testearse solo.
type ImagenNueva struct {
	// NombreArchivo es el que mando el cliente. Se usa SOLO para leerle la
	// extension: el nombre con el que se guarda lo genera el service.
	NombreArchivo string
	// Tamanio en bytes. Viene del parseo del multipart (lo cuenta el servidor
	// al leer la parte), no de un campo que el cliente pueda mentir.
	Tamanio   int64
	Contenido io.Reader
	Color     string
	Orden     int
	AltText   string
}

// Agregar valida y guarda una imagen de producto.
//
// Reglas que aplica, en orden:
//  1. El producto tiene que existir.
//  2. Como maximo MaxImagenesPorProducto imagenes por producto.
//  3. Una sola imagen principal (orden 0) por color.
//  4. Extension dentro de la lista permitida.
//  5. Archivo no vacio y por debajo del maximo configurado.
//  6. El contenido tiene que ser realmente una imagen de esa extension.
//
// Se valida TODO antes de escribir nada, por el mismo motivo que en Crear: no
// queremos un archivo en el disco porque la fila fallo despues.
func (s *ImagenesService) Agregar(productoID uint, in ImagenNueva) (*dao.Imagen, error) {
	p, err := s.productos.BuscarPorID(productoID)
	if err != nil {
		return nil, dom.Interno("no se pudo leer el producto", err)
	}
	if p == nil {
		return nil, dom.NoEncontrado("el producto %d no existe", productoID)
	}

	if len(p.Imagenes) >= MaxImagenesPorProducto {
		return nil, dom.Conflicto(
			"el producto %d ya tiene el maximo de %d imagenes", productoID, MaxImagenesPorProducto)
	}

	if in.Orden < 0 {
		return nil, dom.Validacion("el orden no puede ser negativo")
	}

	// Orden 0 = principal. Color vacio = imagen generica del producto, y esa
	// tambien puede tener su principal: se compara color contra color.
	color := strings.TrimSpace(in.Color)
	if in.Orden == 0 {
		for _, ya := range p.Imagenes {
			if ya.Orden == 0 && strings.EqualFold(strings.TrimSpace(ya.Color), color) {
				return nil, dom.Conflicto(
					"el producto %d ya tiene una imagen principal para el color %q", productoID, color)
			}
		}
	}

	ext := strings.ToLower(filepath.Ext(in.NombreArchivo))
	firma, permitida := firmasPorExtension[ext]
	if !permitida {
		return nil, dom.Validacion(
			"la extension %q no esta permitida; las validas son %s", ext, extensionesPermitidas())
	}

	if in.Tamanio <= 0 {
		return nil, dom.Validacion("el archivo esta vacio")
	}
	if in.Tamanio > s.maxBytes {
		return nil, dom.Validacion(
			"el archivo supera el maximo de %.1f MB", float64(s.maxBytes)/(1024*1024))
	}

	// Se leen los primeros bytes para comparar contra la firma. Un error de
	// lectura aca tambien descalifica el archivo: si no se puede verificar que
	// es una imagen, no se guarda.
	cabecera := make([]byte, len(firma))
	if _, err := io.ReadFull(in.Contenido, cabecera); err != nil || !bytes.Equal(cabecera, firma) {
		return nil, dom.Validacion(
			"el contenido del archivo no corresponde a una imagen %s", strings.TrimPrefix(ext, "."))
	}
	// La cabecera ya se consumio del reader: se vuelve a pegar adelante para
	// que el almacen reciba el archivo COMPLETO.
	contenido := io.MultiReader(bytes.NewReader(cabecera), in.Contenido)

	nombre := nombreDeArchivo(productoID, ext)
	if err := s.almacen.Guardar(nombre, contenido); err != nil {
		return nil, dom.Interno("no se pudo guardar el archivo", err)
	}

	// Primero el archivo, despues la fila. Si falla la fila queda un archivo
	// huerfano: basura inofensiva. Al reves quedaria una fila apuntando a un
	// 404, que el catalogo mostraria como imagen rota.
	img := &dao.Imagen{
		ProductoID: p.ID,
		URL:        PrefijoURLSubidas + nombre,
		Color:      color,
		Orden:      in.Orden,
		AltText:    strings.TrimSpace(in.AltText),
	}
	if err := s.imagenes.Crear(img); err != nil {
		return nil, dom.Interno("no se pudo registrar la imagen", err)
	}
	return img, nil
}

// nombreDeArchivo genera el nombre con el que se guarda el archivo.
//
// NO se reusa el que mando el cliente, por dos motivos: traeria path traversal
// ("../../etc/algo") y colisiones entre dos admins que suben "foto.jpg". Solo
// se le respeta la extension, que ya se valido contra el contenido.
func nombreDeArchivo(productoID uint, ext string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand fallando es catastrofico y no deberia pasar nunca; el
		// timestamp al menos evita pisar un archivo existente.
		return fmt.Sprintf("p%d-%d%s", productoID, time.Now().UnixNano(), ext)
	}
	return fmt.Sprintf("p%d-%s%s", productoID, hex.EncodeToString(b), ext)
}

// extensionesPermitidas arma la lista para el mensaje de error. Ordenada a
// mano: recorrer un map en Go da un orden distinto en cada corrida, y un
// mensaje de error que cambia solo es imposible de testear.
func extensionesPermitidas() string {
	return ".jpg, .jpeg, .png"
}
