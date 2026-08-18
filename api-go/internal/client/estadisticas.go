// Package client implementa el consumo de la API Node de estadísticas.
//
// Es el único punto del servicio que conoce el contrato del upstream: si ese
// contrato cambia, solo hay que tocar este paquete.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/socius/interseguro-challenge/api-go/internal/matrix"
)

var (
	// ErrUpstreamTimeout indica que la API de estadísticas no respondió dentro
	// del plazo configurado.
	ErrTiempoAgotadoServicio = errors.New("la API de estadísticas no respondió a tiempo")
	// ErrUpstreamUnavailable indica que no se pudo establecer la comunicación
	// (DNS, conexión rechazada, servicio caído).
	ErrServicioNoDisponible = errors.New("la API de estadísticas no está disponible")
)

// ErrorEstadoServicio representa una respuesta HTTP no exitosa del servicio.
// Conserva el status y el cuerpo para poder diagnosticar sin revisar los logs
// del otro servicio.
type ErrorEstadoServicio struct {
	Estado int
	Cuerpo string
}

func (e *ErrorEstadoServicio) Error() string {
	return fmt.Sprintf("la API de estadísticas respondió %d: %s", e.Estado, e.Cuerpo)
}

// EstadisticasMatriz son las estadísticas de una matriz individual.
type EstadisticasMatriz struct {
	Maximo     float64 `json:"max"`
	Minimo     float64 `json:"min"`
	Promedio   float64 `json:"average"`
	Suma       float64 `json:"sum"`
	Cantidad   int     `json:"count"`
	Filas      int     `json:"rows"`
	Columnas   int     `json:"cols"`
	EsCuadrada bool    `json:"isSquare"`
	EsDiagonal bool    `json:"isDiagonal"`
	// Tolerance es el umbral con que se evaluó IsDiagonal. Se deriva de la
	// magnitud de cada matriz por separado, de modo que difiere entre Q y R.
	Tolerancia float64 `json:"tolerance"`
}

// EstadisticasGlobales son las estadísticas agregadas sobre todas las matrices.
type EstadisticasGlobales struct {
	Maximo   float64 `json:"max"`
	Minimo   float64 `json:"min"`
	Promedio float64 `json:"average"`
	Suma     float64 `json:"sum"`
	Cantidad int     `json:"count"`
}

// RespuestaEstadisticas es la respuesta de la API Node.
type RespuestaEstadisticas struct {
	Global         EstadisticasGlobales          `json:"overall"`
	PorMatriz      map[string]EstadisticasMatriz `json:"perMatrix"`
	AlgunaDiagonal bool                          `json:"anyDiagonal"`
	// ToleranceFactor es el factor relativo del que se deriva la tolerancia de
	// cada matriz; el umbral concreto va en cada entrada de PerMatrix.
	FactorTolerancia float64 `json:"toleranceFactor"`
}

// solicitudEstadisticas es el cuerpo que se envía al servicio de estadísticas.
type solicitudEstadisticas struct {
	Matrices map[string]matrix.Matriz `json:"matrices"`
}

// ClienteEstadisticas consume la API de estadísticas con tiempo límite y reintentos.
type ClienteEstadisticas struct {
	urlBase          string
	clienteHTTP      *http.Client
	maximoReintentos int
	registrador      *slog.Logger
}

// NuevoClienteEstadisticas construye el cliente. El tiempo límite se aplica a cada intento por
// separado, no al conjunto: el plazo total lo gobierna el contexto del request
// entrante, que se propaga hasta acá.
func NuevoClienteEstadisticas(urlBase string, tiempoEspera time.Duration, maximoReintentos int, registrador *slog.Logger) *ClienteEstadisticas {
	return &ClienteEstadisticas{
		urlBase: strings.TrimSuffix(strings.TrimSpace(urlBase), "/"),
		clienteHTTP: &http.Client{
			Timeout: tiempoEspera,
			Transport: &http.Transport{
				// Reutilizar conexiones importa: en el camino caliente se hace
				// una llamada al upstream por cada request entrante.
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		maximoReintentos: maximoReintentos,
		registrador:      registrador,
	}
}

// Calcular envía las matrices al servicio y devuelve las estadísticas.
//
// authHeader y requestID se propagan tal cual: el primero porque la API de
// estadísticas exige el mismo JWT del usuario final, y el segundo para poder
// correlacionar en los logs una traza que atraviesa los dos servicios.
func (c *ClienteEstadisticas) Calcular(
	contexto context.Context,
	matrices map[string]matrix.Matriz,
	encabezadoAutorizacion, idSolicitud string,
) (*RespuestaEstadisticas, error) {
	cuerpo, err := json.Marshal(solicitudEstadisticas{Matrices: matrices})
	if err != nil {
		return nil, fmt.Errorf("serializando las matrices: %w", err)
	}

	url := c.urlBase + "/api/v1/statistics"
	var ultimoError error

	// Un intento inicial más maxRetries reintentos.
	for intento := 0; intento <= c.maximoReintentos; intento++ {
		if intento > 0 {
			// Backoff exponencial: 200 ms, 400 ms, 800 ms…
			demora := time.Duration(200*(1<<(intento-1))) * time.Millisecond
			select {
			case <-contexto.Done():
				return nil, ErrTiempoAgotadoServicio
			case <-time.After(demora):
			}
			c.registrador.WarnContext(contexto, "reintentando la llamada a la API de estadísticas",
				slog.Int("attempt", intento), slog.String("requestId", idSolicitud), slog.Any("error", ultimoError))
		}

		estadisticas, err := c.hacerSolicitud(contexto, url, cuerpo, encabezadoAutorizacion, idSolicitud)
		if err == nil {
			return estadisticas, nil
		}
		ultimoError = err

		// Un 4xx es un problema del request, no del upstream: reintentarlo
		// solo repetiría el mismo fallo y gastaría el presupuesto de tiempo.
		var errorEstado *ErrorEstadoServicio
		if errors.As(err, &errorEstado) && errorEstado.Estado < 500 {
			return nil, err
		}
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
	}

	return nil, ultimoError
}

// Salud consulta el endpoint de salud del servicio. Lo usa el chequeo de
// readiness: sin la API de estadísticas disponible, este servicio puede
// responder pero no completar su función principal.
func (c *ClienteEstadisticas) Salud(contexto context.Context) error {
	solicitud, err := http.NewRequestWithContext(contexto, http.MethodGet, c.urlBase+"/health", nil)
	if err != nil {
		return fmt.Errorf("construyendo el request: %w", err)
	}

	respuesta, err := c.clienteHTTP.Do(solicitud)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || esTiempoAgotado(err) {
			return ErrTiempoAgotadoServicio
		}
		return fmt.Errorf("%w: %v", ErrServicioNoDisponible, err)
	}
	defer respuesta.Body.Close()
	// El cuerpo se descarta pero se drena, para que la conexión pueda volver
	// al pool de keep-alive en lugar de cerrarse.
	_, _ = io.Copy(io.Discard, io.LimitReader(respuesta.Body, 4<<10))

	if respuesta.StatusCode != http.StatusOK {
		return &ErrorEstadoServicio{Estado: respuesta.StatusCode}
	}
	return nil
}

// hacerSolicitud ejecuta un único intento.
func (c *ClienteEstadisticas) hacerSolicitud(
	contexto context.Context,
	url string,
	cuerpo []byte,
	encabezadoAutorizacion, idSolicitud string,
) (*RespuestaEstadisticas, error) {
	// El cuerpo se envuelve en un lector nuevo en cada intento: un reintento
	// sobre un lector ya consumido enviaría un cuerpo vacío.
	solicitud, err := http.NewRequestWithContext(contexto, http.MethodPost, url, bytes.NewReader(cuerpo))
	if err != nil {
		return nil, fmt.Errorf("construyendo el request: %w", err)
	}
	solicitud.Header.Set("Content-Type", "application/json")
	solicitud.Header.Set("Accept", "application/json")
	if encabezadoAutorizacion != "" {
		solicitud.Header.Set("Authorization", encabezadoAutorizacion)
	}
	if idSolicitud != "" {
		solicitud.Header.Set("X-Request-ID", idSolicitud)
	}

	respuesta, err := c.clienteHTTP.Do(solicitud)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || esTiempoAgotado(err) {
			return nil, ErrTiempoAgotadoServicio
		}
		if errors.Is(err, context.Canceled) {
			return nil, context.Canceled
		}
		return nil, fmt.Errorf("%w: %v", ErrServicioNoDisponible, err)
	}
	defer respuesta.Body.Close()

	// Se acota la lectura: un upstream comprometido o con un bug no debe poder
	// agotar la memoria de este proceso con una respuesta gigante.
	contenido, err := io.ReadAll(io.LimitReader(respuesta.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: leyendo la respuesta: %v", ErrServicioNoDisponible, err)
	}

	if respuesta.StatusCode != http.StatusOK {
		return nil, &ErrorEstadoServicio{Estado: respuesta.StatusCode, Cuerpo: truncar(string(contenido), 512)}
	}

	var estadisticas RespuestaEstadisticas
	if err := json.Unmarshal(contenido, &estadisticas); err != nil {
		return nil, fmt.Errorf("%w: respuesta ilegible: %v", ErrServicioNoDisponible, err)
	}
	return &estadisticas, nil
}

// esTiempoAgotado detecta los tiempos agotados de red, que no siempre se envuelven como
// context.DeadlineExceeded (el Timeout del http.Client produce su propio tipo).
func esTiempoAgotado(err error) bool {
	var errorRed interface{ Timeout() bool }
	return errors.As(err, &errorRed) && errorRed.Timeout()
}

func truncar(texto string, maximo int) string {
	if len(texto) <= maximo {
		return texto
	}
	return texto[:maximo] + "…"
}
