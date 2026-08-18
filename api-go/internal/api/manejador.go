package api

import (
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	"github.com/socius/interseguro-challenge/api-go/internal/auth"
	"github.com/socius/interseguro-challenge/api-go/internal/client"
	"github.com/socius/interseguro-challenge/api-go/internal/config"
	"github.com/socius/interseguro-challenge/api-go/internal/matrix"
)

// Version se sobrescribe en tiempo de compilación con -ldflags. Permite saber
// qué build está corriendo sin entrar al contenedor.
var Version = "dev"

// Manejador agrupa las dependencias de los endpoints.
type Manejador struct {
	configuracion config.Configuracion
	estadisticas  *client.ClienteEstadisticas
	autenticacion *auth.Gestor
	registrador   *slog.Logger
}

// NuevoManejador construye el manejador con sus dependencias ya resueltas. Se
// inyectan en vez de construirse acá para poder sustituirlas en los tests.
func NuevoManejador(configuracion config.Configuracion, estadisticas *client.ClienteEstadisticas, autenticacion *auth.Gestor, registrador *slog.Logger) *Manejador {
	return &Manejador{configuracion: configuracion, estadisticas: estadisticas, autenticacion: autenticacion, registrador: registrador}
}

// IniciarSesion valida las credenciales y emite un JWT.
func (m *Manejador) IniciarSesion(c fiber.Ctx) error {
	var solicitud SolicitudInicioSesion
	if err := c.Bind().JSON(&solicitud); err != nil {
		return NuevoErrorAPI(http.StatusBadRequest, CodigoCuerpoInvalido,
			"el cuerpo debe ser un JSON con los campos 'username' y 'password'", nil)
	}

	// Comparación en tiempo constante: comparar con == permitiría deducir el
	// prefijo correcto de la contraseña midiendo el tiempo de respuesta. Ambas
	// comparaciones se evalúan siempre, sin cortocircuito.
	usuarioValido := subtle.ConstantTimeCompare([]byte(solicitud.Usuario), []byte(m.configuracion.UsuarioDemo)) == 1
	contrasenaValida := subtle.ConstantTimeCompare([]byte(solicitud.Contrasena), []byte(m.configuracion.ContrasenaDemo)) == 1
	if !usuarioValido || !contrasenaValida {
		// El mensaje no distingue entre usuario inexistente y contraseña
		// incorrecta: hacerlo permitiría enumerar usuarios válidos.
		return NuevoErrorAPI(http.StatusUnauthorized, CodigoCredencialesInvalidas,
			"usuario o contraseña incorrectos", nil)
	}

	token, expiraEn, err := m.autenticacion.Emitir(solicitud.Usuario)
	if err != nil {
		m.registrador.ErrorContext(c.Context(), "no se pudo emitir el token", slog.Any("error", err))
		return NuevoErrorAPI(http.StatusInternalServerError, CodigoInterno, "no se pudo emitir el token", nil)
	}

	return c.JSON(RespuestaInicioSesion{
		Token:            token,
		TipoToken:        "Bearer",
		ExpiraEn:         expiraEn,
		ExpiraEnSegundos: int(m.autenticacion.Vigencia().Seconds()),
	})
}

// QR factoriza la matriz recibida y le adjunta las estadísticas calculadas por
// la API Node.
func (m *Manejador) QR(c fiber.Ctx) error {
	var solicitud SolicitudQR
	if err := c.Bind().JSON(&solicitud); err != nil {
		return NuevoErrorAPI(http.StatusBadRequest, CodigoCuerpoInvalido,
			"el cuerpo debe ser un JSON con el campo 'matrix' como array de arrays de números", nil)
	}
	if solicitud.Matriz == nil {
		return NuevoErrorAPI(http.StatusBadRequest, CodigoCuerpoInvalido,
			"falta el campo 'matrix' en el cuerpo del request", nil)
	}

	if errorValidacion := matrix.Validar(solicitud.Matriz, m.configuracion.DimensionMaximaMatriz); errorValidacion != nil {
		return NuevoErrorAPI(http.StatusBadRequest, string(errorValidacion.Codigo), errorValidacion.Mensaje, errorValidacion.Detalles)
	}

	modo, err := interpretarModo(c.Query("mode"))
	if err != nil {
		return err
	}

	inicio := time.Now()
	descomposicion := matrix.QR(solicitud.Matriz, modo)
	duracion := time.Since(inicio)

	respuesta := RespuestaQR{
		Q: descomposicion.Q,
		R: descomposicion.R,
		Metadatos: MetadatosQR{
			Filas:       solicitud.Matriz.Filas(),
			Columnas:    solicitud.Matriz.Columnas(),
			Modo:        string(descomposicion.Modo),
			Algoritmo:   "householder",
			Residuo:     descomposicion.Residuo,
			DuracionMs:  float64(duracion.Microseconds()) / 1000,
			IDSolicitud: requestid.FromContext(c),
		},
	}

	// withStats=false permite ejercitar la API Go de forma aislada, lo que
	// resulta útil para diagnosticar si un fallo viene de este servicio o del
	// upstream.
	if c.Query("withStats") == "false" {
		return c.JSON(respuesta)
	}

	estadisticas, err := m.estadisticas.Calcular(
		c.Context(),
		map[string]matrix.Matriz{"q": descomposicion.Q, "r": descomposicion.R},
		c.Get(fiber.HeaderAuthorization),
		requestid.FromContext(c),
	)
	if err != nil {
		return m.traducirErrorServicio(err)
	}

	respuesta.Estadisticas = estadisticas
	return c.JSON(respuesta)
}

// Rotate rota la matriz 90° en sentido horario y adjunta las estadísticas
// calculadas por la API Node.
func (m *Manejador) Rotar(c fiber.Ctx) error {
	var solicitud SolicitudRotacion
	if err := c.Bind().JSON(&solicitud); err != nil {
		return NuevoErrorAPI(http.StatusBadRequest, CodigoCuerpoInvalido,
			"el cuerpo debe ser un JSON con el campo 'matrix' como array de arrays de números", nil)
	}
	if solicitud.Matriz == nil {
		return NuevoErrorAPI(http.StatusBadRequest, CodigoCuerpoInvalido,
			"falta el campo 'matrix' en el cuerpo del request", nil)
	}
	if errorValidacion := matrix.Validar(solicitud.Matriz, m.configuracion.DimensionMaximaMatriz); errorValidacion != nil {
		return NuevoErrorAPI(http.StatusBadRequest, string(errorValidacion.Codigo), errorValidacion.Mensaje, errorValidacion.Detalles)
	}

	rotada := matrix.Rotar90(solicitud.Matriz)

	respuesta := RespuestaRotacion{
		Rotada: rotada,
		Metadatos: MetadatosRotacion{
			Filas:       rotada.Filas(),
			Columnas:    rotada.Columnas(),
			Direccion:   "clockwise",
			Grados:      90,
			IDSolicitud: requestid.FromContext(c),
		},
	}

	// Mantiene el mismo escape de diagnóstico que QR: permite comprobar la
	// transformación de Go aunque el servicio de estadísticas esté caído.
	if c.Query("withStats") == "false" {
		return c.JSON(respuesta)
	}

	estadisticas, err := m.estadisticas.Calcular(
		c.Context(),
		map[string]matrix.Matriz{"rotated": rotada},
		c.Get(fiber.HeaderAuthorization),
		requestid.FromContext(c),
	)
	if err != nil {
		return m.traducirErrorServicio(err)
	}

	respuesta.Estadisticas = estadisticas
	return c.JSON(respuesta)
}

// Health es el chequeo de vitalidad (liveness). No consulta dependencias
// externas a propósito: si lo hiciera, una caída de la API de estadísticas
// haría que el orquestador reiniciara este servicio, que está perfectamente
// sano, en lugar de aislar el problema donde ocurre.
func (m *Manejador) Salud(c fiber.Ctx) error {
	return c.JSON(RespuestaSalud{Estado: "ok", Servicio: "qr-api-go", Version: Version})
}

// Ready es el chequeo de disponibilidad (readiness): incluye el upstream,
// porque sin él este servicio no puede completar su función principal.
func (m *Manejador) Disponibilidad(c fiber.Ctx) error {
	if err := m.estadisticas.Salud(c.Context()); err != nil {
		return c.Status(http.StatusServiceUnavailable).JSON(RespuestaSalud{
			Estado: "degraded", Servicio: "qr-api-go", Version: Version, ServicioDependiente: "unreachable",
		})
	}
	return c.JSON(RespuestaSalud{Estado: "ok", Servicio: "qr-api-go", Version: Version, ServicioDependiente: "ok"})
}

// parseMode traduce el parámetro `mode` de la query.
func interpretarModo(valor string) (matrix.Modo, error) {
	switch valor {
	case "", string(matrix.ModoCompleto):
		return matrix.ModoCompleto, nil
	case string(matrix.ModoReducido):
		return matrix.ModoReducido, nil
	default:
		return "", NuevoErrorAPI(http.StatusBadRequest, CodigoCuerpoInvalido,
			"el parámetro 'mode' solo admite los valores 'full' o 'reduced'",
			map[string]any{"received": valor, "allowed": []string{"full", "reduced"}})
	}
}

// mapUpstreamError traduce los fallos del cliente de estadísticas a errores
// HTTP, distinguiendo con qué status debe responderse cada caso.
func (m *Manejador) traducirErrorServicio(err error) error {
	switch {
	case errors.Is(err, client.ErrTiempoAgotadoServicio):
		return NuevoErrorAPI(http.StatusGatewayTimeout, CodigoTiempoAgotadoServicio,
			"la API de estadísticas no respondió dentro del tiempo permitido", nil)

	case errors.Is(err, client.ErrServicioNoDisponible):
		return NuevoErrorAPI(http.StatusBadGateway, CodigoServicioNoDisponible,
			"la API de estadísticas no está disponible", nil)
	}

	var errorEstado *client.ErrorEstadoServicio
	if errors.As(err, &errorEstado) {
		// Un 401 del upstream con el mismo token que este servicio ya validó
		// apunta a un desajuste de configuración entre ambos (típicamente
		// JWT_SECRET distinto), no a un problema del cliente.
		if errorEstado.Estado == http.StatusUnauthorized || errorEstado.Estado == http.StatusForbidden {
			return NuevoErrorAPI(http.StatusBadGateway, CodigoErrorServicio,
				"la API de estadísticas rechazó la autenticación: revisar que ambos servicios compartan JWT_SECRET",
				map[string]any{"upstreamStatus": errorEstado.Estado})
		}
		return NuevoErrorAPI(http.StatusBadGateway, CodigoErrorServicio,
			"la API de estadísticas devolvió una respuesta inesperada",
			map[string]any{"upstreamStatus": errorEstado.Estado})
	}

	return NuevoErrorAPI(http.StatusBadGateway, CodigoErrorServicio,
		"no se pudieron obtener las estadísticas", nil)
}
