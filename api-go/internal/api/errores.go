// Package api contiene la capa HTTP de la API Go: rutas, handlers, middleware
// y los contratos de entrada y salida.
package api

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

// Códigos de error de la capa HTTP. Son estables y forman parte del contrato
// público: el cliente puede ramificar sobre ellos sin parsear el mensaje, que
// está escrito para personas y puede cambiar sin previo aviso.
//
// Los códigos de validación de la matriz (EMPTY_MATRIX, RAGGED_ROWS,
// NON_FINITE_VALUE, MATRIX_TOO_LARGE) los define el paquete matrix y se
// propagan tal cual.
const (
	CodigoCuerpoInvalido        = "INVALID_BODY"
	CodigoCredencialesInvalidas = "INVALID_CREDENTIALS"
	CodigoNoAutorizado          = "UNAUTHORIZED"
	CodigoTokenExpirado         = "TOKEN_EXPIRED"
	CodigoServicioNoDisponible  = "UPSTREAM_UNAVAILABLE"
	CodigoTiempoAgotadoServicio = "UPSTREAM_TIMEOUT"
	CodigoErrorServicio         = "UPSTREAM_ERROR"
	CodigoNoEncontrado          = "NOT_FOUND"
	CodigoInterno               = "INTERNAL_ERROR"
)

// DetalleError es el cuerpo de un error. Ambas APIs del sistema usan esta misma
// forma, de modo que el frontend tiene un único camino de manejo de errores.
type DetalleError struct {
	Codigo      string         `json:"code"`
	Mensaje     string         `json:"message"`
	Detalles    map[string]any `json:"details,omitempty"`
	IDSolicitud string         `json:"requestId,omitempty"`
}

// ErrorResponse envuelve el payload bajo la clave `error`, para que nunca se
// confunda con una respuesta exitosa.
type RespuestaError struct {
	Error DetalleError `json:"error"`
}

// APIError es un error que ya sabe con qué status HTTP debe responderse.
type ErrorAPI struct {
	Estado   int
	Codigo   string
	Mensaje  string
	Detalles map[string]any
}

func (e *ErrorAPI) Error() string { return e.Mensaje }

// NuevoErrorAPI construye un error de la capa HTTP.
func NuevoErrorAPI(estado int, codigo, mensaje string, detalles map[string]any) *ErrorAPI {
	return &ErrorAPI{Estado: estado, Codigo: codigo, Mensaje: mensaje, Detalles: detalles}
}

// ErrorHandler es el punto único de conversión de errores a respuestas HTTP.
//
// Centralizarlo evita que cada handler repita la serialización y garantiza que
// ningún error se escape con un formato distinto: cualquier error que llegue
// aquí sale como ErrorResponse, incluidos los panics recuperados y las rutas
// inexistentes.
func ManejarError(c fiber.Ctx, err error) error {
	detalle := DetalleError{
		Codigo:      CodigoInterno,
		Mensaje:     "error interno del servidor",
		IDSolicitud: requestid.FromContext(c),
	}
	estado := estadoDesdeError(err)

	var errorAPI *ErrorAPI
	var errorFiber *fiber.Error

	switch {
	case errors.As(err, &errorAPI):
		estado = errorAPI.Estado
		detalle.Codigo = errorAPI.Codigo
		detalle.Mensaje = errorAPI.Mensaje
		detalle.Detalles = errorAPI.Detalles

	case errors.As(err, &errorFiber):
		// Errores generados por el propio framework: sobre todo el 404 de ruta
		// no encontrada y el 405 de método no permitido.
		estado = errorFiber.Code
		detalle.Mensaje = errorFiber.Message
		if estado == http.StatusNotFound {
			detalle.Codigo = CodigoNoEncontrado
		}
	}

	// Los errores no contemplados salen como 500 genérico a propósito: el
	// detalle interno queda en los logs del servidor y no se filtra al cliente.
	return c.Status(estado).JSON(RespuestaError{Error: detalle})
}

// statusFromError permite que el logger conozca el estado que ErrorHandler
// escribirÃ¡ despuÃ©s de que la cadena de middleware termine.
func estadoDesdeError(err error) int {
	var errorAPI *ErrorAPI
	if errors.As(err, &errorAPI) {
		return errorAPI.Estado
	}

	var errorFiber *fiber.Error
	if errors.As(err, &errorFiber) {
		return errorFiber.Code
	}

	return http.StatusInternalServerError
}
