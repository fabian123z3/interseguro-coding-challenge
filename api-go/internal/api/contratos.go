package api

import (
	"time"

	"github.com/socius/interseguro-challenge/api-go/internal/client"
	"github.com/socius/interseguro-challenge/api-go/internal/matrix"
)

// SolicitudQR es el cuerpo de POST /api/v1/qr.
type SolicitudQR struct {
	// Matriz es la matriz rectangular de entrada, como array de arrays.
	Matriz matrix.Matriz `json:"matrix"`
}

// MetadatosQR acompaña al resultado con información de trazabilidad y calidad
// numérica. Residual permite al consumidor comprobar por sí mismo que la
// factorización reconstruye la matriz original, sin tener que confiar a ciegas
// en el servicio.
type MetadatosQR struct {
	Filas       int     `json:"rows"`
	Columnas    int     `json:"cols"`
	Modo        string  `json:"mode"`
	Algoritmo   string  `json:"algorithm"`
	Residuo     float64 `json:"residual"`
	DuracionMs  float64 `json:"durationMs"`
	IDSolicitud string  `json:"requestId,omitempty"`
}

// RespuestaQR es la respuesta de POST /api/v1/qr.
type RespuestaQR struct {
	Q         matrix.Matriz `json:"q"`
	R         matrix.Matriz `json:"r"`
	Metadatos MetadatosQR   `json:"meta"`
	// Estadisticas viene de la API Node. Es nil cuando se invoca con
	// ?withStats=false, útil para aislar fallos entre ambos servicios.
	Estadisticas *client.RespuestaEstadisticas `json:"statistics,omitempty"`
}

// SolicitudRotacion es el cuerpo de POST /api/v1/rotate.
type SolicitudRotacion struct {
	Matriz matrix.Matriz `json:"matrix"`
}

// RespuestaRotacion es la respuesta de POST /api/v1/rotate.
type RespuestaRotacion struct {
	Rotada       matrix.Matriz                 `json:"rotated"`
	Metadatos    MetadatosRotacion             `json:"meta"`
	Estadisticas *client.RespuestaEstadisticas `json:"statistics,omitempty"`
}

// MetadatosRotacion describe la transformación aplicada.
type MetadatosRotacion struct {
	Filas       int    `json:"rows"`
	Columnas    int    `json:"cols"`
	Direccion   string `json:"direction"`
	Grados      int    `json:"degrees"`
	IDSolicitud string `json:"requestId,omitempty"`
}

// SolicitudInicioSesion es el cuerpo de POST /api/v1/auth/login.
type SolicitudInicioSesion struct {
	Usuario    string `json:"username"`
	Contrasena string `json:"password"`
}

// RespuestaInicioSesion entrega el token y su vencimiento, para que el cliente pueda
// renovarlo antes de que caduque en lugar de esperar el primer 401.
type RespuestaInicioSesion struct {
	Token            string    `json:"token"`
	TipoToken        string    `json:"tokenType"`
	ExpiraEn         time.Time `json:"expiresAt"`
	ExpiraEnSegundos int       `json:"expiresIn"`
}

// RespuestaSalud es la respuesta de los endpoints de salud.
type RespuestaSalud struct {
	Estado   string `json:"status"`
	Servicio string `json:"service"`
	Version  string `json:"version"`
	// ServicioDependiente solo aparece en el chequeo de disponibilidad.
	ServicioDependiente string `json:"upstream,omitempty"`
}
