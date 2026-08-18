// Command server levanta la API Go de factorización QR.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/socius/interseguro-challenge/api-go/internal/api"
	"github.com/socius/interseguro-challenge/api-go/internal/config"
)

func main() {
	// Log estructurado en JSON: es lo que esperan los agregadores de las
	// plataformas cloud (Cloud Logging, CloudWatch) para indexar los campos.
	registrador := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(registrador)

	configuracion, err := config.Cargar()
	if err != nil {
		registrador.Error("configuración inválida", slog.Any("error", err))
		os.Exit(1)
	}

	aplicacion := api.NuevaAplicacion(configuracion, registrador)

	// Se escucha en 0.0.0.0 para ser alcanzable desde fuera del contenedor.
	direccion := "0.0.0.0:" + configuracion.Puerto

	// El servidor corre en su propia goroutine para que el hilo principal
	// quede libre esperando la señal de apagado.
	errorServidor := make(chan error, 1)
	go func() {
		registrador.Info("servidor iniciado",
			slog.String("addr", direccion),
			slog.String("version", api.Version),
			slog.String("statsApi", configuracion.URLAPIEstadisticas),
		)
		errorServidor <- aplicacion.Listen(direccion, fiber.ListenConfig{
			// El banner ASCII de arranque ensuciaría el log estructurado: la
			// misma información ya sale como evento JSON justo arriba.
			DisableStartupMessage: true,
		})
	}()

	// SIGTERM es la señal que envía Docker (y Kubernetes) al detener un
	// contenedor; SIGINT llega con Ctrl+C en desarrollo.
	apagado := make(chan os.Signal, 1)
	signal.Notify(apagado, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errorServidor:
		if err != nil {
			registrador.Error("el servidor se detuvo con error", slog.Any("error", err))
			os.Exit(1)
		}

	case senal := <-apagado:
		registrador.Info("apagado solicitado, drenando conexiones", slog.String("signal", senal.String()))

		// Apagado ordenado: se da margen a los requests en curso para terminar
		// antes de cerrar. Sin esto, un despliegue cortaría respuestas a medio
		// camino y el cliente vería errores de red sin causa aparente.
		contexto, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelar()

		if err := aplicacion.ShutdownWithContext(contexto); err != nil {
			registrador.Error("el apagado ordenado no completó", slog.Any("error", err))
			os.Exit(1)
		}
		registrador.Info("servidor detenido")
	}
}
