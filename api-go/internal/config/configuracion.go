// Package config centraliza la lectura de la configuración del servicio desde
// variables de entorno.
//
// Toda la configuración se resuelve una sola vez al arrancar y se valida de
// inmediato: si algo falta o es inválido el proceso no levanta. Es preferible
// fallar al iniciar, cuando el problema es evidente, que descubrirlo en el
// primer request en producción.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Configuracion agrupa todos los parámetros de ejecución de la API Go.
type Configuracion struct {
	// Puerto es el puerto TCP en que escucha el servidor.
	Puerto string
	// URLAPIEstadisticas es la URL base de la API Node de estadísticas.
	URLAPIEstadisticas string
	// TiempoEsperaEstadisticas acota cada intento de llamada a la API de estadísticas.
	TiempoEsperaEstadisticas time.Duration
	// MaximoReintentosEstadisticas es la cantidad de reintentos tras el primer intento.
	MaximoReintentosEstadisticas int
	// DimensionMaximaMatriz limita filas y columnas de la matriz de entrada.
	DimensionMaximaMatriz int
	// OrigenesCORS enumera los orígenes web autorizados para llamadas directas
	// a la API. El frontend desplegado usa el proxy del mismo origen, pero estos
	// valores permiten el desarrollo local sin abrir la API a cualquier sitio.
	OrigenesCORS []string
	// SecretoJWT es el secreto HS256 compartido con la API de estadísticas.
	SecretoJWT string
	// EmisorJWT y AudienciaJWT se emiten y validan como claims `iss` y `aud`.
	EmisorJWT    string
	AudienciaJWT string
	// VigenciaJWT es la vigencia del token emitido.
	VigenciaJWT time.Duration
	// UsuarioDemo y ContrasenaDemo son las credenciales aceptadas por el
	// endpoint de login. Reemplazan a una base de usuarios real, que está
	// fuera del alcance del desafío.
	UsuarioDemo    string
	ContrasenaDemo string
}

// Cargar construye la configuración desde el entorno, aplicando los valores por
// defecto documentados en .env.example.
func Cargar() (Configuracion, error) {
	configuracion := Configuracion{
		Puerto:                       textoEntorno("GO_API_PORT", "8080"),
		URLAPIEstadisticas:           textoEntorno("STATS_API_URL", "http://localhost:3000"),
		TiempoEsperaEstadisticas:     time.Duration(enteroEntorno("STATS_API_TIMEOUT_SECONDS", 5)) * time.Second,
		MaximoReintentosEstadisticas: enteroEntorno("STATS_API_MAX_RETRIES", 1),
		DimensionMaximaMatriz:        enteroEntorno("MAX_MATRIX_DIMENSION", 256),
		OrigenesCORS: listaEntorno("CORS_ALLOWED_ORIGINS", []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"http://localhost:8081",
			"http://127.0.0.1:8081",
		}),
		SecretoJWT:     os.Getenv("JWT_SECRET"),
		EmisorJWT:      textoEntorno("JWT_ISSUER", "interseguro-qr-api"),
		AudienciaJWT:   textoEntorno("JWT_AUDIENCE", "interseguro-clients"),
		VigenciaJWT:    time.Duration(enteroEntorno("JWT_TTL_MINUTES", 15)) * time.Minute,
		UsuarioDemo:    textoEntorno("DEMO_USERNAME", "demo"),
		ContrasenaDemo: os.Getenv("DEMO_PASSWORD"),
	}

	if err := configuracion.validar(); err != nil {
		return Configuracion{}, err
	}
	return configuracion, nil
}

// validar rechaza combinaciones que dejarían al servicio en un estado
// inseguro o inoperable.
func (c Configuracion) validar() error {
	// Sin secreto no hay forma de firmar ni verificar tokens. Generar uno al
	// vuelo sería peor: cada réplica del servicio firmaría con un secreto
	// distinto y los tokens dejarían de ser válidos entre instancias.
	if c.SecretoJWT == "" {
		return errors.New("JWT_SECRET es obligatorio: definirlo en el entorno (ver .env.example)")
	}
	if c.ContrasenaDemo == "" {
		return errors.New("DEMO_PASSWORD es obligatorio: definirlo en el entorno (ver .env.example)")
	}
	if c.URLAPIEstadisticas == "" {
		return errors.New("STATS_API_URL es obligatorio")
	}
	urlEstadisticas, err := url.Parse(c.URLAPIEstadisticas)
	if err != nil || (urlEstadisticas.Scheme != "http" && urlEstadisticas.Scheme != "https") || urlEstadisticas.Host == "" {
		return fmt.Errorf("STATS_API_URL debe ser una URL HTTP válida, se recibió %q", c.URLAPIEstadisticas)
	}
	if c.DimensionMaximaMatriz < 1 {
		return fmt.Errorf("MAX_MATRIX_DIMENSION debe ser positivo, se recibió %d", c.DimensionMaximaMatriz)
	}
	for _, origen := range c.OrigenesCORS {
		urlOrigen, err := url.Parse(origen)
		if err != nil || (urlOrigen.Scheme != "http" && urlOrigen.Scheme != "https") || urlOrigen.Host == "" {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS contiene un origen HTTP inválido: %q", origen)
		}
	}
	if c.MaximoReintentosEstadisticas < 0 {
		return fmt.Errorf("STATS_API_MAX_RETRIES no puede ser negativo, se recibió %d", c.MaximoReintentosEstadisticas)
	}
	if c.TiempoEsperaEstadisticas <= 0 {
		return errors.New("STATS_API_TIMEOUT_SECONDS debe ser positivo")
	}
	if c.VigenciaJWT <= 0 {
		return errors.New("JWT_TTL_MINUTES debe ser positivo")
	}
	return nil
}

func textoEntorno(clave, alternativo string) string {
	if valor := strings.TrimSpace(os.Getenv(clave)); valor != "" {
		return valor
	}
	return alternativo
}

func listaEntorno(clave string, alternativos []string) []string {
	valor := strings.TrimSpace(os.Getenv(clave))
	if valor == "" {
		return append([]string(nil), alternativos...)
	}

	partes := strings.Split(valor, ",")
	valores := make([]string, 0, len(partes))
	for _, parte := range partes {
		if texto := strings.TrimSpace(parte); texto != "" {
			valores = append(valores, texto)
		}
	}
	return valores
}

// enteroEntorno devuelve el valor por defecto si la variable está ausente o no es un
// entero válido. Un valor mal escrito no debe tumbar el arranque por sí solo:
// validar() se encarga después de rechazar los rangos imposibles.
func enteroEntorno(clave string, alternativo int) int {
	valor, err := strconv.Atoi(os.Getenv(clave))
	if err != nil {
		return alternativo
	}
	return valor
}
