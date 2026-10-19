package main

import (
	"log"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/config"
	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/controllers"
	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/dao"
	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/middleware"
	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/repository"
	"github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/services"
)

func main() {
	cfg, err := config.Cargar()
	if err != nil {
		log.Fatalf("configuracion invalida: %v", err)
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("no se pudo conectar a la base: %v", err)
	}

	// AutoMigrate crea las tablas al arrancar. La base (del compose o de Neon) nace
	// vacia, asi que sin esto la app levanta y falla en la primera query.
	// TODO(TP5+): migrar a migraciones versionadas; AutoMigrate no borra ni
	// renombra columnas y se vuelve insuficiente en cuanto el schema evoluciona.
	if err := db.AutoMigrate(
		&dao.Usuario{}, &dao.Producto{}, &dao.Variante{},
		&dao.Imagen{}, &dao.Pedido{}, &dao.PedidoItem{},
	); err != nil {
		log.Fatalf("fallo la migracion: %v", err)
	}

	// --- Cableado: repositorios -> services -> controllers -------------------
	//
	// Es el unico lugar donde se elige la implementacion concreta. Los services
	// solo ven interfaces, por eso en los tests se les pasa un fake en memoria.
	varianteRepo := repository.NuevoVarianteRepo(db)
	pedidoRepo := repository.NuevoPedidoRepo(db)
	productoRepo := repository.NuevoProductoRepo(db)
	usuarioRepo := repository.NuevoUsuarioRepo(db)
	imagenRepo := repository.NuevoImagenRepo(db)

	// El almacen es lo unico que sabe que las imagenes van a un disco. El
	// service solo ve la interfaz: en el TP6, cambiarlo por object storage es
	// reemplazar esta linea.
	almacen := repository.NuevoAlmacenDisco(cfg.UploadsDir)

	pedidosSvc := services.NuevoPedidosService(
		varianteRepo, pedidoRepo, cfg.UmbralEnvioGratis, cfg.CostoEnvio,
	)
	productosSvc := services.NuevoProductosService(productoRepo, varianteRepo)
	imagenesSvc := services.NuevoImagenesService(
		productoRepo, imagenRepo, almacen, cfg.MaxImagenBytes,
	)

	// El secreto y la duracion del token se resuelven ACA y se le pasan al
	// service como una funcion. Asi services/ no importa middleware/, que
	// arrastraria Gin al paquete de reglas de negocio.
	usuariosSvc := services.NuevoUsuariosService(usuarioRepo, func(usuarioID uint, esAdmin bool) (string, error) {
		return middleware.GenerarToken(cfg.JWTSecret, usuarioID, esAdmin, cfg.DuracionToken)
	})

	usuariosCtl := controllers.NuevoUsuariosController(usuariosSvc)
	productosCtl := controllers.NuevoProductosController(productosSvc)
	pedidosCtl := controllers.NuevoPedidosController(pedidosSvc)
	imagenesCtl := controllers.NuevoImagenesController(imagenesSvc)

	// TP9: Sentry cuenta que falla, que tarda y que se usa. Sin DSN queda
	// apagado y todo lo demas sigue igual.
	if cfg.SentryDSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.SentryDSN,
			Environment:      cfg.SentryEntorno,
			Release:          cfg.Commit, // cada error queda atado a la imagen que lo produjo
			EnableTracing:    true,
			TracesSampleRate: 1.0, // medir todos los pedidos: con este trafico sobra cuota
		}); err != nil {
			log.Printf("sentry no arranco (la app sigue): %v", err)
		}
		// Lo que quede en el buffer se manda antes de salir.
		defer sentry.Flush(2 * time.Second)
	}

	r := gin.Default()
	// Va primero para medir cada pedido entero y capturar los panics: Repanic
	// los devuelve al Recovery de gin.Default, que sigue contestando 500.
	r.Use(sentrygin.New(sentrygin.Options{Repanic: true}))

	// Cuanto de un multipart se buferea en memoria antes de spoolear a disco.
	// No es el limite de subida (ese es cfg.MaxImagenBytes, y lo aplica el
	// service): es solo cuanta RAM se usa mientras se parsea.
	r.MaxMultipartMemory = 8 << 20 // 8 MB

	// Healthcheck: lo usa el healthcheck del compose y, mas adelante, el
	// pipeline de CD del TP6 para saber si el deploy quedo sano.
	// TP6: ademas informa el commit, para que el smoke compruebe que contesta
	// la version recien desplegada (Render sigue sirviendo la anterior mientras
	// construye).
	// TP9: tambien HEAD. El monitor externo gratuito (UptimeRobot) pregunta con
	// HEAD, y Gin no lo deriva del GET: sin esta ruta contestaba 404 y el
	// monitor marcaba PROD caida estando viva. HEAD no lleva cuerpo; Gin lo
	// descarta solo.
	salud := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "commit": cfg.Commit})
	}
	r.GET("/healthz", salud)
	r.HEAD("/healthz", salud)

	// Archivos subidos por el admin. Se sirven desde el volumen montado.
	// Recordar: nginx tiene que proxear /uploads/ ademas de /api/.
	r.Static("/uploads", cfg.UploadsDir)

	api := r.Group("/api")
	{
		// --- Publicos: el catalogo y la puerta de entrada -------------------
		api.GET("/productos", productosCtl.Listar)
		api.GET("/productos/:id", productosCtl.VerDetalle)
		api.POST("/usuarios/registro", usuariosCtl.Registro)
		api.POST("/usuarios/login", usuariosCtl.Login)

		// --- Autenticados: todo lo que es "mio" -----------------------------
		auth := api.Group("", middleware.RequiereAuth(cfg.JWTSecret))
		auth.POST("/pedidos", pedidosCtl.Checkout)
		auth.GET("/pedidos", pedidosCtl.MisPedidos)
		auth.POST("/pedidos/:id/cancelar", pedidosCtl.Cancelar)

		// --- Admin: se encadena DESPUES de auth, nunca en paralelo ----------
		admin := auth.Group("", middleware.RequiereAdmin())
		admin.POST("/productos", productosCtl.Crear)
		admin.PUT("/productos/:id", productosCtl.Editar)
		admin.POST("/productos/:id/variantes", productosCtl.AgregarVariante)
		admin.PATCH("/productos/:id/variantes/:varianteId", productosCtl.ActualizarStock)
		// El listado del panel: incluye los productos dados de baja, que el
		// catalogo publico no devuelve.
		admin.GET("/admin/productos", productosCtl.ListarParaAdmin)
		admin.POST("/productos/:id/imagenes", imagenesCtl.Subir)
		admin.PATCH("/pedidos/:id/estado", pedidosCtl.CambiarEstado)
	}

	log.Printf("escuchando en :%s (token valido por %s)", cfg.Puerto, cfg.DuracionToken)
	if err := r.Run(":" + cfg.Puerto); err != nil {
		log.Fatalf("el servidor se cayo: %v", err)
	}
}
