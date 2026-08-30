package services

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/dao"
	dom "github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/errors"
)

// --- Fakes ------------------------------------------------------------------

type fakeImagenRepo struct {
	creadas []dao.Imagen
	proximo uint
}

func (f *fakeImagenRepo) Crear(i *dao.Imagen) error {
	f.proximo++
	i.ID = f.proximo
	f.creadas = append(f.creadas, *i)
	return nil
}

// fakeAlmacen reemplaza al disco. Es la razon de ser de la interfaz
// AlmacenImagenes: estos tests no escriben un solo archivo.
type fakeAlmacen struct {
	guardados map[string][]byte
}

func nuevoFakeAlmacen() *fakeAlmacen {
	return &fakeAlmacen{guardados: map[string][]byte{}}
}

func (f *fakeAlmacen) Guardar(nombre string, contenido io.Reader) error {
	b, err := io.ReadAll(contenido)
	if err != nil {
		return err
	}
	f.guardados[nombre] = b
	return nil
}

const maxBytesTest = 1024 * 1024 // 1 MB

func setupImagenes() (*ImagenesService, *fakeProductoRepo, *fakeImagenRepo, *fakeAlmacen) {
	pr := nuevoFakeProductoRepo()
	ir := &fakeImagenRepo{}
	al := nuevoFakeAlmacen()
	// Un producto listo para colgarle fotos.
	_ = pr.Crear(&dao.Producto{Nombre: "Remera basica", Precio: 10000, Activo: true})
	return NuevoImagenesService(pr, ir, al, maxBytesTest), pr, ir, al
}

// jpegDe arma un contenido que empieza con la firma de un JPEG real.
func jpegDe(resto string) *bytes.Reader {
	return bytes.NewReader(append([]byte{0xFF, 0xD8, 0xFF}, []byte(resto)...))
}

func imagenValida() ImagenNueva {
	cuerpo := jpegDe("cuerpo del archivo")
	return ImagenNueva{
		NombreArchivo: "foto.jpg",
		Tamanio:       int64(cuerpo.Len()),
		Contenido:     cuerpo,
		Color:         "Negro",
		Orden:         0,
		AltText:       "Remera negra de frente",
	}
}

// --- Camino feliz -----------------------------------------------------------

func TestAgregarImagen_Ok(t *testing.T) {
	svc, _, ir, al := setupImagenes()

	img, err := svc.Agregar(1, imagenValida())
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if !strings.HasPrefix(img.URL, PrefijoURLSubidas) {
		t.Errorf("la URL tiene que ser relativa bajo %s, llego %q", PrefijoURLSubidas, img.URL)
	}
	if !strings.HasSuffix(img.URL, ".jpg") {
		t.Errorf("la URL tiene que conservar la extension, llego %q", img.URL)
	}
	if len(ir.creadas) != 1 {
		t.Fatalf("se esperaba 1 fila creada, hay %d", len(ir.creadas))
	}
	if ir.creadas[0].ProductoID != 1 {
		t.Errorf("la imagen quedo colgada del producto %d", ir.creadas[0].ProductoID)
	}
	if len(al.guardados) != 1 {
		t.Fatalf("se esperaba 1 archivo guardado, hay %d", len(al.guardados))
	}
}

// El service consume los primeros bytes para verificar la firma. Si no los
// vuelve a pegar adelante, el archivo se guarda mutilado y la foto no abre:
// es el bug silencioso mas facil de cometer aca.
func TestAgregarImagen_GuardaElArchivoCompleto(t *testing.T) {
	svc, _, _, al := setupImagenes()

	if _, err := svc.Agregar(1, imagenValida()); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	esperado := append([]byte{0xFF, 0xD8, 0xFF}, []byte("cuerpo del archivo")...)
	for _, contenido := range al.guardados {
		if !bytes.Equal(contenido, esperado) {
			t.Errorf("el archivo guardado no es el original: %v", contenido)
		}
	}
}

// El nombre lo genera el service. Si se reusara el del cliente, este caso
// escribiria fuera del directorio de subidas.
func TestAgregarImagen_NoUsaElNombreDelCliente(t *testing.T) {
	svc, _, _, al := setupImagenes()

	in := imagenValida()
	in.NombreArchivo = "../../etc/passwd.jpg"

	if _, err := svc.Agregar(1, in); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	for nombre := range al.guardados {
		if strings.Contains(nombre, "..") || strings.ContainsAny(nombre, "/\\") {
			t.Errorf("el nombre generado no puede tener separadores ni '..': %q", nombre)
		}
		if !strings.HasPrefix(nombre, "p1-") {
			t.Errorf("el nombre tendria que derivar del producto, llego %q", nombre)
		}
	}
}

// --- Producto ---------------------------------------------------------------

func TestAgregarImagen_ProductoInexistente(t *testing.T) {
	svc, _, _, al := setupImagenes()

	_, err := svc.Agregar(999, imagenValida())
	if k := kindDe(t, err); k != dom.KindNoEncontrado {
		t.Errorf("se esperaba KindNoEncontrado, llego %v", k)
	}
	if len(al.guardados) != 0 {
		t.Error("no se puede guardar un archivo de un producto que no existe")
	}
}

// --- Regla: una sola principal (orden 0) por color --------------------------

func TestAgregarImagen_SegundaPrincipalDelMismoColor(t *testing.T) {
	svc, pr, _, _ := setupImagenes()
	pr.data[1].Imagenes = []dao.Imagen{{ProductoID: 1, URL: "/uploads/vieja.jpg", Color: "Negro", Orden: 0}}

	_, err := svc.Agregar(1, imagenValida()) // tambien Negro y orden 0
	if k := kindDe(t, err); k != dom.KindConflicto {
		t.Errorf("se esperaba KindConflicto, llego %v", k)
	}
}

func TestAgregarImagen_PrincipalDeOtroColorSiEntra(t *testing.T) {
	svc, pr, _, _ := setupImagenes()
	pr.data[1].Imagenes = []dao.Imagen{{ProductoID: 1, URL: "/uploads/vieja.jpg", Color: "Negro", Orden: 0}}

	in := imagenValida()
	in.Color = "Beige"
	if _, err := svc.Agregar(1, in); err != nil {
		t.Fatalf("otro color tiene su propia principal: %v", err)
	}
}

// Un orden distinto de 0 no es principal, asi que no compite.
func TestAgregarImagen_SegundaSecundariaDelMismoColor(t *testing.T) {
	svc, pr, _, _ := setupImagenes()
	pr.data[1].Imagenes = []dao.Imagen{{ProductoID: 1, URL: "/uploads/vieja.jpg", Color: "Negro", Orden: 0}}

	in := imagenValida()
	in.Orden = 1
	if _, err := svc.Agregar(1, in); err != nil {
		t.Fatalf("una secundaria del mismo color tiene que entrar: %v", err)
	}
}

func TestAgregarImagen_OrdenNegativo(t *testing.T) {
	svc, _, _, _ := setupImagenes()

	in := imagenValida()
	in.Orden = -1
	_, err := svc.Agregar(1, in)
	if k := kindDe(t, err); k != dom.KindValidacion {
		t.Errorf("se esperaba KindValidacion, llego %v", k)
	}
}

// --- Regla: maximo de imagenes por producto ---------------------------------

func TestAgregarImagen_MaximoPorProducto(t *testing.T) {
	svc, pr, _, _ := setupImagenes()
	llenas := make([]dao.Imagen, MaxImagenesPorProducto)
	for i := range llenas {
		llenas[i] = dao.Imagen{ProductoID: 1, URL: "/uploads/x.jpg", Orden: i + 1}
	}
	pr.data[1].Imagenes = llenas

	_, err := svc.Agregar(1, imagenValida())
	if k := kindDe(t, err); k != dom.KindConflicto {
		t.Errorf("se esperaba KindConflicto, llego %v", k)
	}
}

// --- Regla: extension y contenido -------------------------------------------

func TestAgregarImagen_ExtensionNoPermitida(t *testing.T) {
	svc, _, _, al := setupImagenes()

	for _, nombre := range []string{"virus.exe", "documento.pdf", "sin-extension", "foto.JPG.exe"} {
		in := imagenValida()
		in.NombreArchivo = nombre

		_, err := svc.Agregar(1, in)
		if k := kindDe(t, err); k != dom.KindValidacion {
			t.Errorf("%s: se esperaba KindValidacion, llego %v", nombre, k)
		}
	}
	if len(al.guardados) != 0 {
		t.Error("no se guarda nada si la extension no esta permitida")
	}
}

// La extension en mayusculas es la misma extension: no puede ser un rodeo.
func TestAgregarImagen_ExtensionEnMayusculas(t *testing.T) {
	svc, _, _, _ := setupImagenes()

	in := imagenValida()
	in.NombreArchivo = "FOTO.JPG"
	if _, err := svc.Agregar(1, in); err != nil {
		t.Fatalf("una extension en mayusculas es valida: %v", err)
	}
}

// El caso que la lista de extensiones sola no atrapa: un ejecutable renombrado.
func TestAgregarImagen_ContenidoQueNoEsImagen(t *testing.T) {
	svc, _, ir, al := setupImagenes()

	cuerpo := bytes.NewReader([]byte("MZ esto es un ejecutable, no un jpeg"))
	in := imagenValida()
	in.Contenido = cuerpo
	in.Tamanio = int64(cuerpo.Len())

	_, err := svc.Agregar(1, in)
	if k := kindDe(t, err); k != dom.KindValidacion {
		t.Errorf("se esperaba KindValidacion, llego %v", k)
	}
	if len(al.guardados) != 0 || len(ir.creadas) != 0 {
		t.Error("un archivo que no es imagen no se guarda ni se registra")
	}
}

// Un PNG subido como .jpg tampoco pasa: la firma se compara contra la
// extension declarada, no contra "cualquier imagen".
func TestAgregarImagen_FirmaQueNoCoincideConLaExtension(t *testing.T) {
	svc, _, _, _ := setupImagenes()

	cuerpo := bytes.NewReader([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A})
	in := imagenValida()
	in.NombreArchivo = "en-realidad-es-png.jpg"
	in.Contenido = cuerpo
	in.Tamanio = int64(cuerpo.Len())

	_, err := svc.Agregar(1, in)
	if k := kindDe(t, err); k != dom.KindValidacion {
		t.Errorf("se esperaba KindValidacion, llego %v", k)
	}
}

func TestAgregarImagen_PNGValido(t *testing.T) {
	svc, _, _, _ := setupImagenes()

	cuerpo := bytes.NewReader([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	in := imagenValida()
	in.NombreArchivo = "foto.png"
	in.Contenido = cuerpo
	in.Tamanio = int64(cuerpo.Len())

	img, err := svc.Agregar(1, in)
	if err != nil {
		t.Fatalf("un PNG valido tiene que entrar: %v", err)
	}
	if !strings.HasSuffix(img.URL, ".png") {
		t.Errorf("la URL tiene que terminar en .png, llego %q", img.URL)
	}
}

// --- Regla: tamano ----------------------------------------------------------

func TestAgregarImagen_ArchivoVacio(t *testing.T) {
	svc, _, _, _ := setupImagenes()

	in := imagenValida()
	in.Tamanio = 0
	_, err := svc.Agregar(1, in)
	if k := kindDe(t, err); k != dom.KindValidacion {
		t.Errorf("se esperaba KindValidacion, llego %v", k)
	}
}

func TestAgregarImagen_DemasiadoGrande(t *testing.T) {
	svc, _, _, al := setupImagenes()

	in := imagenValida()
	in.Tamanio = maxBytesTest + 1
	_, err := svc.Agregar(1, in)
	if k := kindDe(t, err); k != dom.KindValidacion {
		t.Errorf("se esperaba KindValidacion, llego %v", k)
	}
	if len(al.guardados) != 0 {
		t.Error("un archivo por encima del maximo no se guarda")
	}
}

// El borde exacto entra: el maximo es inclusivo.
func TestAgregarImagen_TamanoJustoEnElMaximo(t *testing.T) {
	svc, _, _, _ := setupImagenes()

	in := imagenValida()
	in.Tamanio = maxBytesTest
	if _, err := svc.Agregar(1, in); err != nil {
		t.Fatalf("el maximo es inclusivo: %v", err)
	}
}
