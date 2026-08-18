package api

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	cabecerasSeguras "github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	"github.com/socius/interseguro-challenge/api-go/internal/auth"
	"github.com/socius/interseguro-challenge/api-go/internal/client"
	"github.com/socius/interseguro-challenge/api-go/internal/config"
)

// NuevaAplicacion arma la aplicación Fiber con su cadena de middleware y sus rutas.
//
// Devolver *fiber.App en vez de arrancar el servidor acá permite que los tests
// usen app.Test() sin abrir un puerto real.
func NuevaAplicacion(configuracion config.Configuracion, registrador *slog.Logger) *fiber.App {
	clienteEstadisticas := client.NuevoClienteEstadisticas(configuracion.URLAPIEstadisticas, configuracion.TiempoEsperaEstadisticas, configuracion.MaximoReintentosEstadisticas, registrador)
	gestorAutenticacion := auth.NuevoGestor(configuracion.SecretoJWT, configuracion.EmisorJWT, configuracion.AudienciaJWT, configuracion.VigenciaJWT)
	manejador := NuevoManejador(configuracion, clienteEstadisticas, gestorAutenticacion, registrador)

	app := fiber.New(fiber.Config{
		ErrorHandler: ManejarError,
		AppName:      "Interseguro QR API (Go + Fiber)",
		// El cuerpo se acota a 16 MB: una matriz de 256×256 en JSON ocupa unos
		// pocos MB, así que este techo deja margen suficiente y a la vez impide
		// que un cuerpo enorme agote la memoria antes de llegar a validarse.
		BodyLimit: 16 * 1024 * 1024,
		// Fiber no impone tiempos de espera por defecto. Estos límites evitan
		// conexiones lentas o abandonadas consumiendo recursos indefinidamente.
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	})

	// El orden importa. recover va primero para atrapar los panics de todo lo
	// que venga después; requestid antes del logger para que el identificador
	// ya exista al momento de registrar la línea.
	app.Use(recover.New())
	app.Use(cabecerasSeguras.New())
	app.Use(requestid.New())
	app.Use(RegistrarSolicitudes(registrador))
	app.Use(cors.New(cors.Config{
		AllowOrigins: configuracion.OrigenesCORS,
		AllowMethods: []string{fiber.MethodGet, fiber.MethodPost, fiber.MethodOptions},
		AllowHeaders: []string{fiber.HeaderContentType, fiber.HeaderAuthorization, "X-Request-ID"},
	}))

	// Endpoints de salud: públicos, porque los consultan el orquestador y el
	// balanceador, que no tienen credenciales.
	app.Get("/health", manejador.Salud)
	app.Get("/health/ready", manejador.Disponibilidad)

	v1 := app.Group("/api/v1")
	v1.Post("/auth/login", manejador.IniciarSesion)

	// El middleware de autenticación se declara ruta por ruta en lugar de sobre
	// un grupo. Un grupo con prefijo vacío se comporta como un Use() sobre todo
	// /api/v1: interceptaría también las rutas inexistentes —devolviendo 401
	// donde corresponde un 404— y dejaría que el carácter público o protegido
	// de cada endpoint dependiera del orden en que se registró.
	exigirAutenticacion := ExigirJWT(gestorAutenticacion)
	v1.Post("/qr", exigirAutenticacion, manejador.QR)
	v1.Post("/rotate", exigirAutenticacion, manejador.Rotar)

	return app
}
