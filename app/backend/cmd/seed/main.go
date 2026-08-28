// Comando de carga inicial de datos.
//
// Es un binario APARTE de la API a proposito: sembrar la base es una operacion
// de un solo uso, no algo que tenga que correr en cada arranque de cada replica
// (en el TP6 hay mas de una). Se ejecuta a mano:
//
//	docker compose exec backend /bin/seed
//	docker compose exec backend /bin/seed -reset        # borra el catalogo y recarga
//
// Es idempotente: si ya hay productos cargados, no hace nada y avisa.
package main

import (
	"flag"
	"log"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/config"
	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/dao"
	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/repository"
	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/services"
)

// Parametros por flag y no por variable de entorno: config/ es el unico lugar
// del backend que lee el entorno, y estos datos son de la herramienta, no de
// la aplicacion.
var (
	adminEmail = flag.String("admin-email", "admin@tienda.local", "email del usuario admin")
	adminPass  = flag.String("admin-pass", "admin12345", "contrasena del usuario admin (cambiala en cualquier entorno real)")
	reset      = flag.Bool("reset", false, "borra el catalogo antes de cargarlo de nuevo")
)

func main() {
	flag.Parse()

	cfg, err := config.Cargar()
	if err != nil {
		log.Fatalf("configuracion invalida: %v", err)
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("no se pudo conectar a MySQL: %v", err)
	}

	// Mismo AutoMigrate que la API: el seed puede correr antes que ella.
	if err := db.AutoMigrate(
		&dao.Usuario{}, &dao.Producto{}, &dao.Variante{},
		&dao.Imagen{}, &dao.Pedido{}, &dao.PedidoItem{},
	); err != nil {
		log.Fatalf("fallo la migracion: %v", err)
	}

	if *reset {
		borrarCatalogo(db)
	}

	crearAdmin(db)
	crearCatalogo(db)

	log.Println("seed terminado")
}

// crearAdmin da de alta el usuario administrador.
//
// Pasa por el mismo service que el registro publico (asi el hash y las
// validaciones son exactamente los mismos) y recien despues se lo marca como
// admin con un UPDATE explicito: la API publica NUNCA puede crear un admin,
// por eso Registrar deja EsAdmin en false y no hay forma de pedir otra cosa.
func crearAdmin(db *gorm.DB) {
	usuarioRepo := repository.NuevoUsuarioRepo(db)
	svc := services.NuevoUsuariosService(usuarioRepo, func(uint, bool) (string, error) {
		// El seed no emite tokens; nadie llama a Login aca.
		return "", nil
	})

	existente, err := usuarioRepo.BuscarPorEmail(strings.ToLower(*adminEmail))
	if err != nil {
		log.Fatalf("no se pudo consultar el admin: %v", err)
	}
	if existente != nil {
		log.Printf("el admin %s ya existe, no se toca", existente.Email)
		return
	}

	u, err := svc.Registrar(services.Registro{
		Email:    *adminEmail,
		Password: *adminPass,
		Nombre:   "Admin",
		Apellido: "Tienda",
	})
	if err != nil {
		log.Fatalf("no se pudo crear el admin: %v", err)
	}
	if err := db.Model(&dao.Usuario{}).Where("id = ?", u.ID).Update("es_admin", true).Error; err != nil {
		log.Fatalf("no se pudo marcar el admin: %v", err)
	}
	log.Printf("admin creado: %s (contrasena: %s) — cambiala fuera de dev", u.Email, *adminPass)
}

// crearCatalogo carga los productos pasando por ProductosService.
//
// Esto no es un detalle: el seed entra por la MISMA puerta que el admin, asi
// que no puede sembrar datos que violen las reglas (precio <= 0, talle fuera
// de la lista, variante duplicada). Si el seed carga, la regla se cumple.
func crearCatalogo(db *gorm.DB) {
	var cantidad int64
	if err := db.Model(&dao.Producto{}).Count(&cantidad).Error; err != nil {
		log.Fatalf("no se pudo contar los productos: %v", err)
	}
	if cantidad > 0 {
		log.Printf("ya hay %d productos cargados; no se siembra nada (usa -reset para rehacerlo)", cantidad)
		return
	}

	svc := services.NuevoProductosService(
		repository.NuevoProductoRepo(db),
		repository.NuevoVarianteRepo(db),
	)

	for _, p := range catalogo {
		creado, err := svc.Crear(services.ProductoNuevo{
			Nombre:      p.Nombre,
			Descripcion: p.Descripcion,
			Precio:      p.Precio,
			Categoria:   p.Categoria,
			Variantes:   p.variantes(),
		})
		if err != nil {
			log.Fatalf("no se pudo crear %q: %v", p.Nombre, err)
		}

		// Las imagenes se insertan directo: el endpoint de carga es del TP2
		// Fase 2. Son las estaticas, las que sirve nginx desde
		// frontend/public/productos/.
		img := dao.Imagen{
			ProductoID: creado.ID,
			URL:        "/productos/" + p.Imagen,
			Orden:      0, // 0 = principal
			AltText:    p.Nombre,
		}
		if err := db.Create(&img).Error; err != nil {
			log.Fatalf("no se pudo crear la imagen de %q: %v", p.Nombre, err)
		}

		log.Printf("cargado: %s (%d variantes)", creado.Nombre, len(creado.Variantes))
	}
	log.Printf("%d productos cargados", len(catalogo))
}

// borrarCatalogo deja la base sin catalogo. No toca usuarios ni pedidos: si
// hay pedidos viejos, sus items conservan la descripcion y el precio que se
// cobro (por eso PedidoItem esta denormalizado).
func borrarCatalogo(db *gorm.DB) {
	// Where("1 = 1") porque GORM se niega a hacer un DELETE sin condicion.
	if err := db.Where("1 = 1").Delete(&dao.Imagen{}).Error; err != nil {
		log.Fatalf("no se pudieron borrar las imagenes: %v", err)
	}
	if err := db.Where("1 = 1").Delete(&dao.Variante{}).Error; err != nil {
		log.Fatalf("no se pudieron borrar las variantes: %v", err)
	}
	if err := db.Where("1 = 1").Delete(&dao.Producto{}).Error; err != nil {
		log.Fatalf("no se pudieron borrar los productos: %v", err)
	}
	log.Println("catalogo borrado")
}

// ---------------------------------------------------------------------------
// El catalogo.
//
// Los nombres de archivo de Imagen son el contrato con la Fase 2: las fotos
// van en frontend/public/productos/ con exactamente estos nombres.
// ---------------------------------------------------------------------------

type itemCatalogo struct {
	Nombre      string
	Descripcion string
	Categoria   string
	Precio      float64
	Imagen      string
	Colores     []string
	Talles      []string
	// Stock por talle, en el mismo orden que Talles. Hay ceros a proposito:
	// el front tiene que mostrar los talles sin stock deshabilitados, y sin
	// un cero en el seed esa regla no se puede demostrar en la defensa.
	Stock []int
}

func (i itemCatalogo) variantes() []services.VarianteNueva {
	out := make([]services.VarianteNueva, 0, len(i.Colores)*len(i.Talles))
	for _, color := range i.Colores {
		for idx, talle := range i.Talles {
			out = append(out, services.VarianteNueva{
				Talle: talle,
				Color: color,
				Stock: i.Stock[idx],
			})
		}
	}
	return out
}

var catalogo = []itemCatalogo{
	{
		Nombre:      "Remera basica de algodon",
		Descripcion: "Remera de algodon peinado 24/1, cuello redondo reforzado. Corte regular.",
		Categoria:   "remeras",
		Precio:      12500,
		Imagen:      "remera-basica.jpg",
		Colores:     []string{"Negro", "Blanco", "Gris"},
		Talles:      []string{"S", "M", "L", "XL"},
		Stock:       []int{8, 12, 10, 4},
	},
	{
		Nombre:      "Remera oversize estampada",
		Descripcion: "Corte oversize, estampa serigrafiada al frente. Algodon 100%.",
		Categoria:   "remeras",
		Precio:      15900,
		Imagen:      "remera-oversize.jpg",
		Colores:     []string{"Negro", "Beige"},
		Talles:      []string{"S", "M", "L"},
		Stock:       []int{5, 9, 0},
	},
	{
		Nombre:      "Musculosa deportiva",
		Descripcion: "Tela con secado rapido, ideal para entrenamiento.",
		Categoria:   "remeras",
		Precio:      9800,
		Imagen:      "musculosa-deportiva.jpg",
		Colores:     []string{"Negro", "Azul"},
		Talles:      []string{"XS", "S", "M", "L"},
		Stock:       []int{6, 7, 7, 3},
	},
	{
		Nombre:      "Buzo canguro con capucha",
		Descripcion: "Frisa perchada, bolsillo canguro y cordon ajustable.",
		Categoria:   "buzos",
		Precio:      32900,
		Imagen:      "buzo-canguro.jpg",
		Colores:     []string{"Gris", "Negro", "Verde"},
		Talles:      []string{"S", "M", "L", "XL"},
		Stock:       []int{4, 8, 6, 2},
	},
	{
		Nombre:      "Buzo cuello redondo",
		Descripcion: "Frisa liviana, puños y cintura con elastico.",
		Categoria:   "buzos",
		Precio:      28500,
		Imagen:      "buzo-cuello-redondo.jpg",
		Colores:     []string{"Beige", "Negro"},
		Talles:      []string{"M", "L", "XL"},
		Stock:       []int{7, 5, 0},
	},
	{
		Nombre:      "Campera rompeviento",
		Descripcion: "Impermeable liviana, con capucha desmontable y forro de red.",
		Categoria:   "camperas",
		Precio:      54900,
		Imagen:      "campera-rompeviento.jpg",
		Colores:     []string{"Negro", "Azul"},
		Talles:      []string{"S", "M", "L", "XL"},
		Stock:       []int{3, 5, 4, 2},
	},
	{
		Nombre:      "Campera de jean",
		Descripcion: "Denim rigido, bolsillos con tapa y botones metalicos.",
		Categoria:   "camperas",
		Precio:      61500,
		Imagen:      "campera-jean.jpg",
		Colores:     []string{"Azul"},
		Talles:      []string{"S", "M", "L"},
		Stock:       []int{4, 6, 3},
	},
	{
		Nombre:      "Jean recto clasico",
		Descripcion: "Denim de algodon con elastano, cinco bolsillos, tiro medio.",
		Categoria:   "pantalones",
		Precio:      38900,
		Imagen:      "jean-recto.jpg",
		Colores:     []string{"Azul", "Negro"},
		Talles:      []string{"S", "M", "L", "XL"},
		Stock:       []int{6, 9, 7, 3},
	},
	{
		Nombre:      "Pantalon jogger",
		Descripcion: "Frisa liviana, puno elastizado y cordon.",
		Categoria:   "pantalones",
		Precio:      26900,
		Imagen:      "pantalon-jogger.jpg",
		Colores:     []string{"Gris", "Negro"},
		Talles:      []string{"S", "M", "L"},
		Stock:       []int{8, 10, 5},
	},
	{
		Nombre:      "Pantalon cargo",
		Descripcion: "Gabardina resistente con bolsillos laterales.",
		Categoria:   "pantalones",
		Precio:      34500,
		Imagen:      "pantalon-cargo.jpg",
		Colores:     []string{"Verde", "Beige"},
		Talles:      []string{"M", "L", "XL"},
		Stock:       []int{5, 6, 0},
	},
	{
		Nombre:      "Camisa de lino",
		Descripcion: "Lino y viscosa, manga larga, caida suelta.",
		Categoria:   "camisas",
		Precio:      31900,
		Imagen:      "camisa-lino.jpg",
		Colores:     []string{"Blanco", "Beige"},
		Talles:      []string{"S", "M", "L"},
		Stock:       []int{5, 7, 4},
	},
	{
		Nombre:      "Camisa a cuadros",
		Descripcion: "Franela de algodon, doble bolsillo al frente.",
		Categoria:   "camisas",
		Precio:      29900,
		Imagen:      "camisa-cuadros.jpg",
		Colores:     []string{"Rojo", "Verde"},
		Talles:      []string{"S", "M", "L", "XL"},
		Stock:       []int{4, 6, 5, 2},
	},
	{
		Nombre:      "Vestido midi",
		Descripcion: "Viscosa con caida, tiras regulables y espalda descubierta.",
		Categoria:   "vestidos",
		Precio:      42500,
		Imagen:      "vestido-midi.jpg",
		Colores:     []string{"Negro", "Rojo"},
		Talles:      []string{"XS", "S", "M", "L"},
		Stock:       []int{3, 6, 6, 2},
	},
	{
		Nombre:      "Short de gabardina",
		Descripcion: "Gabardina de algodon, cintura con cordon.",
		Categoria:   "pantalones",
		Precio:      21900,
		Imagen:      "short-gabardina.jpg",
		Colores:     []string{"Beige", "Negro"},
		Talles:      []string{"S", "M", "L"},
		Stock:       []int{7, 8, 4},
	},
	{
		Nombre:      "Gorra clasica",
		Descripcion: "Gabardina con visera curva y cierre regulable.",
		Categoria:   "accesorios",
		Precio:      13900,
		Imagen:      "gorra-clasica.jpg",
		Colores:     []string{"Negro", "Blanco", "Azul"},
		// Los accesorios no tienen talle real, pero la variante lo exige:
		// se usa un unico talle M para no inventar un caso especial en el modelo.
		Talles: []string{"M"},
		Stock:  []int{15},
	},
}
