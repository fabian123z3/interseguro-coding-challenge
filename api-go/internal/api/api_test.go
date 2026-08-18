package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/socius/interseguro-challenge/api-go/internal/config"
)

const (
	testSecret   = "secreto-de-prueba"
	testUser     = "demo"
	testPassword = "clave-de-prueba"
)

// descartarRegistrador evita ensuciar la salida del test con las líneas de request.
func descartarRegistrador() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func configuracionPrueba(urlEstadisticas string) config.Configuracion {
	return config.Configuracion{
		Puerto:                       "0",
		URLAPIEstadisticas:           urlEstadisticas,
		TiempoEsperaEstadisticas:     2 * time.Second,
		MaximoReintentosEstadisticas: 0, // sin reintentos: los tests deben ser deterministas y rápidos
		DimensionMaximaMatriz:        8,
		OrigenesCORS:                 []string{"https://permitido.ejemplo.cl"},
		SecretoJWT:                   testSecret,
		EmisorJWT:                    "test-issuer",
		AudienciaJWT:                 "test-audience",
		VigenciaJWT:                  15 * time.Minute,
		UsuarioDemo:                  testUser,
		ContrasenaDemo:               testPassword,
	}
}

// respuestaEstadisticasSimulada reproduce literalmente la forma que devuelve la API Node.
//
// Es una copia de una respuesta real del servicio, no una aproximación escrita a
// mano: un stub que solo se parezca al contrato deja pasar los desajustes entre
// ambos servicios, que es justo lo que estos tests deben detectar.
const respuestaEstadisticasSimulada = `{
  "overall": {"max": 10, "min": -2, "average": 3.5, "sum": 42, "count": 12},
  "perMatrix": {
    "q": {"max": 1, "min": -1, "average": 0, "sum": 0, "count": 4, "rows": 2, "cols": 2, "isSquare": true, "isDiagonal": true, "tolerance": 1e-9},
    "r": {"max": 10, "min": 0, "average": 5, "sum": 20, "count": 4, "rows": 2, "cols": 2, "isSquare": true, "isDiagonal": false, "tolerance": 1e-8}
  },
  "anyDiagonal": true,
  "toleranceFactor": 1e-9
}`

// solicitudCapturada guarda lo que el upstream recibió, para poder afirmar sobre
// la propagación de encabezados y el cuerpo enviado.
type solicitudCapturada struct {
	autorizacion string
	idSolicitud  string
	cuerpo       []byte
	llamadas     int
}

// nuevoSimuladorEstadisticas levanta un upstream simulado. manejador puede ser nil para usar la
// respuesta exitosa por defecto.
func nuevoSimuladorEstadisticas(t *testing.T, captured *solicitudCapturada, manejador http.HandlerFunc) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if captured != nil {
			captured.llamadas++
			captured.autorizacion = r.Header.Get("Authorization")
			captured.idSolicitud = r.Header.Get("X-Request-ID")
			captured.cuerpo, _ = io.ReadAll(r.Body)
		}
		if manejador != nil {
			manejador(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, respuestaEstadisticasSimulada)
	}))
	t.Cleanup(server.Close)
	return server
}

// iniciarSesionPrueba obtiene un token válido a través del endpoint real, en lugar de firmar
// uno a mano: así el test también cubre que ambos extremos usen el mismo
// emisor, audiencia y secreto.
func iniciarSesionPrueba(t *testing.T, app *fiber.App) string {
	t.Helper()

	resp := hacerSolicitudPrueba(t, app, http.MethodPost, "/api/v1/auth/login", "",
		map[string]string{"username": testUser, "password": testPassword})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("iniciarSesionPrueba falló con status %d", resp.StatusCode)
	}

	var cuerpo RespuestaInicioSesion
	decodificarCuerpo(t, resp, &cuerpo)
	return cuerpo.Token
}

// hacerSolicitudPrueba ejecuta un request contra la app en memoria, sin abrir un puerto.
func hacerSolicitudPrueba(t *testing.T, app *fiber.App, method, path, token string, contenido any) *http.Response {
	t.Helper()

	var lector io.Reader
	if contenido != nil {
		encoded, err := json.Marshal(contenido)
		if err != nil {
			t.Fatalf("no se pudo serializar el contenido: %v", err)
		}
		lector = bytes.NewReader(encoded)
	}

	req := httptest.NewRequest(method, path, lector)
	if contenido != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// El timeout por defecto de app.Test es 1 s, insuficiente para los casos
	// que ejercitan deliberadamente un upstream lento.
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second, FailOnTimeout: true})
	if err != nil {
		t.Fatalf("app.Test devolvió error: %v", err)
	}
	return resp
}

func decodificarCuerpo(t *testing.T, resp *http.Response, out any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("no se pudo decodificar la respuesta: %v", err)
	}
}

func verificarCodigoError(t *testing.T, resp *http.Response, wantStatus int, wantCode string) {
	t.Helper()

	var cuerpo RespuestaError
	decodificarCuerpo(t, resp, &cuerpo)

	if resp.StatusCode != wantStatus {
		t.Errorf("status = %d, se esperaba %d (código %s)", resp.StatusCode, wantStatus, cuerpo.Error.Codigo)
	}
	if cuerpo.Error.Codigo != wantCode {
		t.Errorf("código = %q, se esperaba %q", cuerpo.Error.Codigo, wantCode)
	}
	if cuerpo.Error.Mensaje == "" {
		t.Error("el error no trae mensaje legible")
	}
}

// --- Autenticación ---------------------------------------------------------

func TestIniciarSesion(t *testing.T) {
	app := NuevaAplicacion(configuracionPrueba("http://unused"), descartarRegistrador())

	cases := []struct {
		name       string
		contenido  any
		wantStatus int
		wantCode   string
	}{
		{
			name:       "credenciales válidas",
			contenido:  map[string]string{"username": testUser, "password": testPassword},
			wantStatus: http.StatusOK,
		},
		{
			name:       "contraseña incorrecta",
			contenido:  map[string]string{"username": testUser, "password": "incorrecta"},
			wantStatus: http.StatusUnauthorized,
			wantCode:   CodigoCredencialesInvalidas,
		},
		{
			name:       "usuario inexistente",
			contenido:  map[string]string{"username": "fantasma", "password": testPassword},
			wantStatus: http.StatusUnauthorized,
			wantCode:   CodigoCredencialesInvalidas,
		},
		{
			name:       "cuerpo vacío",
			contenido:  map[string]string{},
			wantStatus: http.StatusUnauthorized,
			wantCode:   CodigoCredencialesInvalidas,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := hacerSolicitudPrueba(t, app, http.MethodPost, "/api/v1/auth/login", "", tc.contenido)

			if tc.wantStatus != http.StatusOK {
				verificarCodigoError(t, resp, tc.wantStatus, tc.wantCode)
				return
			}

			var cuerpo RespuestaInicioSesion
			decodificarCuerpo(t, resp, &cuerpo)
			if cuerpo.Token == "" {
				t.Error("no se devolvió token")
			}
			if cuerpo.TipoToken != "Bearer" {
				t.Errorf("tokenType = %q, se esperaba \"Bearer\"", cuerpo.TipoToken)
			}
			if cuerpo.ExpiraEnSegundos != 900 {
				t.Errorf("expiresIn = %d, se esperaban 900 segundos", cuerpo.ExpiraEnSegundos)
			}
		})
	}
}

func TestEndpointsProtegidosExigenToken(t *testing.T) {
	app := NuevaAplicacion(configuracionPrueba("http://unused"), descartarRegistrador())
	contenido := map[string]any{"matrix": [][]float64{{1, 2}, {3, 4}}}

	cases := []struct {
		name   string
		header string
	}{
		{"sin encabezado", ""},
		{"esquema incorrecto", "Basic abc123"},
		{"token vacío", "Bearer "},
		{"token inventado", "Bearer no-es-un-jwt"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, _ := json.Marshal(contenido)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/qr", bytes.NewReader(encoded))
			req.Header.Set("Content-Type", "application/json")
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test devolvió error: %v", err)
			}
			verificarCodigoError(t, resp, http.StatusUnauthorized, CodigoNoAutorizado)
		})
	}
}

func TestTokenExpiradoSeInforma(t *testing.T) {
	cfg := configuracionPrueba("http://unused")
	cfg.VigenciaJWT = -time.Minute // el token nace vencido
	app := NuevaAplicacion(cfg, descartarRegistrador())

	resp := hacerSolicitudPrueba(t, app, http.MethodPost, "/api/v1/auth/login", "",
		map[string]string{"username": testUser, "password": testPassword})
	var loginBody RespuestaInicioSesion
	decodificarCuerpo(t, resp, &loginBody)

	resp = hacerSolicitudPrueba(t, app, http.MethodPost, "/api/v1/qr", loginBody.Token,
		map[string]any{"matrix": [][]float64{{1, 2}, {3, 4}}})

	verificarCodigoError(t, resp, http.StatusUnauthorized, CodigoTokenExpirado)
}

// --- Endpoint QR -----------------------------------------------------------

func TestQRExitoso(t *testing.T) {
	captured := &solicitudCapturada{}
	stub := nuevoSimuladorEstadisticas(t, captured, nil)
	app := NuevaAplicacion(configuracionPrueba(stub.URL), descartarRegistrador())
	token := iniciarSesionPrueba(t, app)

	resp := hacerSolicitudPrueba(t, app, http.MethodPost, "/api/v1/qr", token,
		map[string]any{"matrix": [][]float64{{12, -51, 4}, {6, 167, -68}, {-4, 24, -41}}})

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", resp.StatusCode)
	}

	var cuerpo RespuestaQR
	decodificarCuerpo(t, resp, &cuerpo)

	if cuerpo.Q.Filas() != 3 || cuerpo.Q.Columnas() != 3 {
		t.Errorf("Q es %d×%d, se esperaba 3×3", cuerpo.Q.Filas(), cuerpo.Q.Columnas())
	}
	if cuerpo.R.Filas() != 3 || cuerpo.R.Columnas() != 3 {
		t.Errorf("R es %d×%d, se esperaba 3×3", cuerpo.R.Filas(), cuerpo.R.Columnas())
	}
	if cuerpo.Metadatos.Algoritmo != "householder" {
		t.Errorf("algorithm = %q", cuerpo.Metadatos.Algoritmo)
	}
	if cuerpo.Metadatos.Modo != "full" {
		t.Errorf("mode = %q, se esperaba \"full\"", cuerpo.Metadatos.Modo)
	}
	if cuerpo.Metadatos.Residuo > 1e-10 {
		t.Errorf("residual = %g: la factorización no reconstruye la matriz", cuerpo.Metadatos.Residuo)
	}
	if cuerpo.Metadatos.IDSolicitud == "" {
		t.Error("meta no trae requestId")
	}
	if cuerpo.Estadisticas == nil {
		t.Fatal("no se adjuntaron las estadísticas del upstream")
	}
	if cuerpo.Estadisticas.Global.Suma != 42 {
		t.Errorf("sum = %g, se esperaba el valor del upstream simulado (42)", cuerpo.Estadisticas.Global.Suma)
	}
}

// TestContratoEstadisticasSeDecodificaCompleto verifica que la estructura Go cubra todos
// los campos que emite la API Node.
//
// Existe porque un desajuste de este tipo no rompe nada de forma visible: los
// campos que Go no declara se descartan en silencio al deserializar, y el
// cliente recibe ceros donde debería haber datos. Solo se detecta comparando el
// contrato campo por campo.
func TestContratoEstadisticasSeDecodificaCompleto(t *testing.T) {
	stub := nuevoSimuladorEstadisticas(t, nil, nil)
	app := NuevaAplicacion(configuracionPrueba(stub.URL), descartarRegistrador())
	token := iniciarSesionPrueba(t, app)

	resp := hacerSolicitudPrueba(t, app, http.MethodPost, "/api/v1/qr", token,
		map[string]any{"matrix": [][]float64{{1, 2}, {3, 4}}})

	var cuerpo RespuestaQR
	decodificarCuerpo(t, resp, &cuerpo)

	stats := cuerpo.Estadisticas
	if stats == nil {
		t.Fatal("no se adjuntaron las estadísticas")
	}
	if stats.FactorTolerancia != 1e-9 {
		t.Errorf("toleranceFactor = %g, se esperaba 1e-9: el campo no se está deserializando",
			stats.FactorTolerancia)
	}
	if !stats.AlgunaDiagonal {
		t.Error("anyDiagonal = false, el upstream simulado devuelve true")
	}

	q, ok := stats.PorMatriz["q"]
	if !ok {
		t.Fatal("falta la matriz 'q' en perMatrix")
	}
	if q.Tolerancia != 1e-9 {
		t.Errorf("perMatrix.q.tolerance = %g, se esperaba 1e-9: el campo no se está deserializando",
			q.Tolerancia)
	}
	if !q.EsCuadrada || !q.EsDiagonal {
		t.Errorf("perMatrix.q = %+v: los booleanos no se están deserializando", q)
	}
	if q.Filas != 2 || q.Columnas != 2 || q.Cantidad != 4 {
		t.Errorf("perMatrix.q dimensiones = %d×%d (count %d), se esperaba 2×2 (4)", q.Filas, q.Columnas, q.Cantidad)
	}
}

// TestQRPropagaEncabezadosAlServicio verifica el contrato entre servicios: la API
// Node exige el mismo JWT y usa X-Request-ID para correlacionar sus logs con
// los de este servicio.
func TestQRPropagaEncabezadosAlServicio(t *testing.T) {
	captured := &solicitudCapturada{}
	stub := nuevoSimuladorEstadisticas(t, captured, nil)
	app := NuevaAplicacion(configuracionPrueba(stub.URL), descartarRegistrador())
	token := iniciarSesionPrueba(t, app)

	hacerSolicitudPrueba(t, app, http.MethodPost, "/api/v1/qr", token,
		map[string]any{"matrix": [][]float64{{1, 2}, {3, 4}}})

	if captured.llamadas != 1 {
		t.Fatalf("el upstream recibió %d llamadas, se esperaba 1", captured.llamadas)
	}
	if captured.autorizacion != "Bearer "+token {
		t.Errorf("Authorization propagado = %q, se esperaba el token del cliente", captured.autorizacion)
	}
	if captured.idSolicitud == "" {
		t.Error("no se propagó X-Request-ID")
	}

	// El cuerpo debe llevar ambas matrices bajo las claves acordadas.
	var sent struct {
		Matrices map[string][][]float64 `json:"matrices"`
	}
	if err := json.Unmarshal(captured.cuerpo, &sent); err != nil {
		t.Fatalf("el upstream recibió un cuerpo ilegible: %v", err)
	}
	for _, key := range []string{"q", "r"} {
		if _, ok := sent.Matrices[key]; !ok {
			t.Errorf("falta la matriz %q en el cuerpo enviado al upstream", key)
		}
	}
}

func TestQRSinEstadisticas(t *testing.T) {
	captured := &solicitudCapturada{}
	stub := nuevoSimuladorEstadisticas(t, captured, nil)
	app := NuevaAplicacion(configuracionPrueba(stub.URL), descartarRegistrador())
	token := iniciarSesionPrueba(t, app)

	resp := hacerSolicitudPrueba(t, app, http.MethodPost, "/api/v1/qr?withStats=false", token,
		map[string]any{"matrix": [][]float64{{1, 2}, {3, 4}}})

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", resp.StatusCode)
	}

	var cuerpo RespuestaQR
	decodificarCuerpo(t, resp, &cuerpo)
	if cuerpo.Estadisticas != nil {
		t.Error("se adjuntaron estadísticas pese a withStats=false")
	}
	if captured.llamadas != 0 {
		t.Errorf("el upstream fue invocado %d veces, no debía invocarse", captured.llamadas)
	}
}

func TestQRModoReducido(t *testing.T) {
	stub := nuevoSimuladorEstadisticas(t, nil, nil)
	app := NuevaAplicacion(configuracionPrueba(stub.URL), descartarRegistrador())
	token := iniciarSesionPrueba(t, app)

	resp := hacerSolicitudPrueba(t, app, http.MethodPost, "/api/v1/qr?mode=reduced", token,
		map[string]any{"matrix": [][]float64{{1, 2}, {3, 4}, {5, 6}, {7, 8}}})

	var cuerpo RespuestaQR
	decodificarCuerpo(t, resp, &cuerpo)

	if cuerpo.Metadatos.Modo != "reduced" {
		t.Errorf("mode = %q, se esperaba \"reduced\"", cuerpo.Metadatos.Modo)
	}
	// La variante reducida recorta Q de 4×4 a 4×2 y R de 4×2 a 2×2.
	if cuerpo.Q.Filas() != 4 || cuerpo.Q.Columnas() != 2 {
		t.Errorf("Q es %d×%d, se esperaba 4×2", cuerpo.Q.Filas(), cuerpo.Q.Columnas())
	}
	if cuerpo.R.Filas() != 2 || cuerpo.R.Columnas() != 2 {
		t.Errorf("R es %d×%d, se esperaba 2×2", cuerpo.R.Filas(), cuerpo.R.Columnas())
	}
}

func TestQRRechazaEntradaInvalida(t *testing.T) {
	stub := nuevoSimuladorEstadisticas(t, nil, nil)
	app := NuevaAplicacion(configuracionPrueba(stub.URL), descartarRegistrador())
	token := iniciarSesionPrueba(t, app)

	cases := []struct {
		name      string
		path      string
		contenido any
		wantCode  string
	}{
		{
			name:      "falta el campo matrix",
			path:      "/api/v1/qr",
			contenido: map[string]any{},
			wantCode:  CodigoCuerpoInvalido,
		},
		{
			name:      "matriz sin filas",
			path:      "/api/v1/qr",
			contenido: map[string]any{"matrix": [][]float64{}},
			wantCode:  "EMPTY_MATRIX",
		},
		{
			name:      "filas de distinto largo",
			path:      "/api/v1/qr",
			contenido: map[string]any{"matrix": [][]float64{{1, 2, 3}, {4, 5}}},
			wantCode:  "RAGGED_ROWS",
		},
		{
			// Filas nulas: la primera fila no tiene columnas, de modo que la
			// matriz se descarta como vacía antes de mirar el resto.
			name:      "filas nulas",
			path:      "/api/v1/qr",
			contenido: map[string]any{"matrix": make([][]float64, 4)},
			wantCode:  "EMPTY_MATRIX",
		},
		{
			name:      "modo inexistente",
			path:      "/api/v1/qr?mode=oblicuo",
			contenido: map[string]any{"matrix": [][]float64{{1, 2}, {3, 4}}},
			wantCode:  CodigoCuerpoInvalido,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := hacerSolicitudPrueba(t, app, http.MethodPost, tc.path, token, tc.contenido)
			verificarCodigoError(t, resp, http.StatusBadRequest, tc.wantCode)
		})
	}
}

func TestQRRechazaMatrizDemasiadoGrande(t *testing.T) {
	stub := nuevoSimuladorEstadisticas(t, nil, nil)
	app := NuevaAplicacion(configuracionPrueba(stub.URL), descartarRegistrador()) // DimensionMaximaMatriz = 8
	token := iniciarSesionPrueba(t, app)

	// 9×9 supera el límite configurado.
	oversized := make([][]float64, 9)
	for i := range oversized {
		oversized[i] = make([]float64, 9)
	}

	resp := hacerSolicitudPrueba(t, app, http.MethodPost, "/api/v1/qr", token,
		map[string]any{"matrix": oversized})

	verificarCodigoError(t, resp, http.StatusBadRequest, "MATRIX_TOO_LARGE")
}

// TestQRRechazaNumeroNoRepresentable documenta dónde se corta un valor que no
// cabe en un float64.
//
// JSON no tiene literales para NaN ni infinito, así que un valor no finito solo
// puede llegar como un número fuera de rango (1e400). El decodificador lo
// rechaza antes de que la matriz exista, por lo que el error es INVALID_BODY y
// no NON_FINITE_VALUE. Esa validación del paquete matrix sigue siendo útil como
// defensa en profundidad para quien use el paquete fuera de la capa HTTP, y se
// prueba en su propio test.
func TestQRRechazaNumeroNoRepresentable(t *testing.T) {
	stub := nuevoSimuladorEstadisticas(t, nil, nil)
	app := NuevaAplicacion(configuracionPrueba(stub.URL), descartarRegistrador())
	token := iniciarSesionPrueba(t, app)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/qr",
		bytes.NewReader([]byte(`{"matrix": [[1, 2], [3, 1e400]]}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test devolvió error: %v", err)
	}
	verificarCodigoError(t, resp, http.StatusBadRequest, CodigoCuerpoInvalido)
}

// --- Fallos del upstream ---------------------------------------------------

func TestFallosServicioEstadisticas(t *testing.T) {
	cases := []struct {
		name       string
		manejador  http.HandlerFunc
		wantStatus int
		wantCode   string
	}{
		{
			name: "el upstream devuelve 500",
			manejador: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantStatus: http.StatusBadGateway,
			wantCode:   CodigoErrorServicio,
		},
		{
			name: "el upstream rechaza el token",
			manejador: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantStatus: http.StatusBadGateway,
			wantCode:   CodigoErrorServicio,
		},
		{
			name: "el upstream devuelve algo que no es JSON",
			manejador: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, "<html>error del proxy</html>")
			},
			wantStatus: http.StatusBadGateway,
			wantCode:   CodigoServicioNoDisponible,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := nuevoSimuladorEstadisticas(t, nil, tc.manejador)
			app := NuevaAplicacion(configuracionPrueba(stub.URL), descartarRegistrador())
			token := iniciarSesionPrueba(t, app)

			resp := hacerSolicitudPrueba(t, app, http.MethodPost, "/api/v1/qr", token,
				map[string]any{"matrix": [][]float64{{1, 2}, {3, 4}}})

			verificarCodigoError(t, resp, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestServicioEstadisticasInalcanzable(t *testing.T) {
	// Puerto cerrado: la conexión se rechaza de inmediato.
	app := NuevaAplicacion(configuracionPrueba("http://127.0.0.1:1"), descartarRegistrador())
	token := iniciarSesionPrueba(t, app)

	resp := hacerSolicitudPrueba(t, app, http.MethodPost, "/api/v1/qr", token,
		map[string]any{"matrix": [][]float64{{1, 2}, {3, 4}}})

	verificarCodigoError(t, resp, http.StatusBadGateway, CodigoServicioNoDisponible)
}

func TestTiempoAgotadoServicioEstadisticas(t *testing.T) {
	stub := nuevoSimuladorEstadisticas(t, nil, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(500 * time.Millisecond)
		_, _ = io.WriteString(w, respuestaEstadisticasSimulada)
	})

	cfg := configuracionPrueba(stub.URL)
	cfg.TiempoEsperaEstadisticas = 50 * time.Millisecond
	app := NuevaAplicacion(cfg, descartarRegistrador())
	token := iniciarSesionPrueba(t, app)

	resp := hacerSolicitudPrueba(t, app, http.MethodPost, "/api/v1/qr", token,
		map[string]any{"matrix": [][]float64{{1, 2}, {3, 4}}})

	verificarCodigoError(t, resp, http.StatusGatewayTimeout, CodigoTiempoAgotadoServicio)
}

// TestReintentoAnteErrorServidor comprueba que un 5xx transitorio se reintente
// y que el segundo intento pueda tener éxito.
func TestReintentoAnteErrorServidor(t *testing.T) {
	attempts := 0
	stub := nuevoSimuladorEstadisticas(t, nil, func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, respuestaEstadisticasSimulada)
	})

	cfg := configuracionPrueba(stub.URL)
	cfg.MaximoReintentosEstadisticas = 1
	app := NuevaAplicacion(cfg, descartarRegistrador())
	token := iniciarSesionPrueba(t, app)

	resp := hacerSolicitudPrueba(t, app, http.MethodPost, "/api/v1/qr", token,
		map[string]any{"matrix": [][]float64{{1, 2}, {3, 4}}})

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, se esperaba que el reintento tuviera éxito", resp.StatusCode)
	}
	if attempts != 2 {
		t.Errorf("intentos = %d, se esperaban 2 (original + 1 reintento)", attempts)
	}
}

// TestNoReintentaErrorCliente verifica que un 4xx no se reintente:
// repetir un request mal formado solo gastaría tiempo y carga.
func TestNoReintentaErrorCliente(t *testing.T) {
	attempts := 0
	stub := nuevoSimuladorEstadisticas(t, nil, func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
	})

	cfg := configuracionPrueba(stub.URL)
	cfg.MaximoReintentosEstadisticas = 3
	app := NuevaAplicacion(cfg, descartarRegistrador())
	token := iniciarSesionPrueba(t, app)

	hacerSolicitudPrueba(t, app, http.MethodPost, "/api/v1/qr", token,
		map[string]any{"matrix": [][]float64{{1, 2}, {3, 4}}})

	if attempts != 1 {
		t.Errorf("intentos = %d, se esperaba 1 (los 4xx no se reintentan)", attempts)
	}
}

// --- Endpoint de rotación --------------------------------------------------

func TestRotar(t *testing.T) {
	captured := &solicitudCapturada{}
	stub := nuevoSimuladorEstadisticas(t, captured, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
  "overall": {"max": 6, "min": 1, "average": 3.5, "sum": 21, "count": 6},
  "perMatrix": {
    "rotated": {"max": 6, "min": 1, "average": 3.5, "sum": 21, "count": 6, "rows": 3, "cols": 2, "isSquare": false, "isDiagonal": false, "tolerance": 6e-9}
  },
  "anyDiagonal": false,
  "toleranceFactor": 1e-9
}`)
	})
	app := NuevaAplicacion(configuracionPrueba(stub.URL), descartarRegistrador())
	token := iniciarSesionPrueba(t, app)

	resp := hacerSolicitudPrueba(t, app, http.MethodPost, "/api/v1/rotate", token,
		map[string]any{"matrix": [][]float64{{1, 2, 3}, {4, 5, 6}}})

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", resp.StatusCode)
	}

	var cuerpo RespuestaRotacion
	decodificarCuerpo(t, resp, &cuerpo)

	want := [][]float64{{4, 1}, {5, 2}, {6, 3}}
	if cuerpo.Rotada.Filas() != 3 || cuerpo.Rotada.Columnas() != 2 {
		t.Fatalf("la rotada es %d×%d, se esperaba 3×2", cuerpo.Rotada.Filas(), cuerpo.Rotada.Columnas())
	}
	for i := range want {
		for j := range want[i] {
			if cuerpo.Rotada[i][j] != want[i][j] {
				t.Errorf("rotada[%d][%d] = %g, se esperaba %g", i, j, cuerpo.Rotada[i][j], want[i][j])
			}
		}
	}
	if cuerpo.Metadatos.Grados != 90 || cuerpo.Metadatos.Direccion != "clockwise" {
		t.Errorf("meta = %+v, se esperaba rotación de 90° en sentido horario", cuerpo.Metadatos)
	}
	if cuerpo.Estadisticas == nil {
		t.Fatal("la respuesta no incluyó las estadísticas de la matriz rotada")
	}
	if _, ok := cuerpo.Estadisticas.PorMatriz["rotated"]; !ok {
		t.Errorf("perMatrix = %+v, se esperaba la clave rotated", cuerpo.Estadisticas.PorMatriz)
	}

	var upstreamBody struct {
		Matrices map[string][][]float64 `json:"matrices"`
	}
	if err := json.Unmarshal(captured.cuerpo, &upstreamBody); err != nil {
		t.Fatalf("el cuerpo enviado a Node no es JSON válido: %v", err)
	}
	upstreamRotated, ok := upstreamBody.Matrices["rotated"]
	if !ok {
		t.Fatalf("la API Go no envió la matriz rotada a Node: %s", captured.cuerpo)
	}
	if len(upstreamRotated) != 3 || len(upstreamRotated[0]) != 2 || upstreamRotated[0][0] != 4 {
		t.Errorf("matriz enviada a Node = %v, se esperaba la rotación 3×2", upstreamRotated)
	}
	if captured.autorizacion != "Bearer "+token {
		t.Errorf("Authorization no se propagó a Node")
	}
	if captured.idSolicitud == "" {
		t.Error("X-Request-ID no se propagó a Node")
	}
}

func TestRotarSinEstadisticas(t *testing.T) {
	captured := &solicitudCapturada{}
	stub := nuevoSimuladorEstadisticas(t, captured, nil)
	app := NuevaAplicacion(configuracionPrueba(stub.URL), descartarRegistrador())
	token := iniciarSesionPrueba(t, app)

	resp := hacerSolicitudPrueba(t, app, http.MethodPost, "/api/v1/rotate?withStats=false", token,
		map[string]any{"matrix": [][]float64{{1, 2}, {3, 4}}})

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", resp.StatusCode)
	}
	var cuerpo RespuestaRotacion
	decodificarCuerpo(t, resp, &cuerpo)
	if cuerpo.Estadisticas != nil {
		t.Error("statistics debe omitirse con withStats=false")
	}
	if captured.llamadas != 0 {
		t.Errorf("Node recibió %d llamadas, se esperaban 0", captured.llamadas)
	}
}

// --- Salud y rutas ---------------------------------------------------------

func TestSaludEsPublica(t *testing.T) {
	app := NuevaAplicacion(configuracionPrueba("http://127.0.0.1:1"), descartarRegistrador())

	resp := hacerSolicitudPrueba(t, app, http.MethodGet, "/health", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200 sin token", resp.StatusCode)
	}

	var cuerpo RespuestaSalud
	decodificarCuerpo(t, resp, &cuerpo)
	if cuerpo.Estado != "ok" {
		t.Errorf("status = %q, se esperaba \"ok\"", cuerpo.Estado)
	}
	if cuerpo.Servicio != "qr-api-go" {
		t.Errorf("service = %q", cuerpo.Servicio)
	}
}

func TestCabecerasDeSeguridad(t *testing.T) {
	app := NuevaAplicacion(configuracionPrueba("http://127.0.0.1:1"), descartarRegistrador())

	resp := hacerSolicitudPrueba(t, app, http.MethodGet, "/health", "", nil)
	if valor := resp.Header.Get("X-Content-Type-Options"); valor != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, se esperaba nosniff", valor)
	}
	if valor := resp.Header.Get("X-Frame-Options"); valor == "" {
		t.Error("X-Frame-Options debe estar presente")
	}
}

func TestCORSRestringeOrigenes(t *testing.T) {
	app := NuevaAplicacion(configuracionPrueba("http://127.0.0.1:1"), descartarRegistrador())

	probarOrigen := func(origen string) string {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("Origin", origen)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test devolvió error: %v", err)
		}
		return resp.Header.Get("Access-Control-Allow-Origin")
	}

	if valor := probarOrigen("https://permitido.ejemplo.cl"); valor != "https://permitido.ejemplo.cl" {
		t.Errorf("origen permitido recibió Access-Control-Allow-Origin = %q", valor)
	}
	if valor := probarOrigen("https://bloqueado.ejemplo.cl"); valor != "" {
		t.Errorf("origen bloqueado recibió Access-Control-Allow-Origin = %q", valor)
	}
}

// TestDisponibilidadReflejaServicioDependiente comprueba que liveness y readiness respondan
// distinto: el servicio está vivo aunque su dependencia no lo esté.
func TestDisponibilidadReflejaServicioDependiente(t *testing.T) {
	t.Run("upstream disponible", func(t *testing.T) {
		stub := nuevoSimuladorEstadisticas(t, nil, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"status":"ok"}`)
		})
		app := NuevaAplicacion(configuracionPrueba(stub.URL), descartarRegistrador())

		resp := hacerSolicitudPrueba(t, app, http.MethodGet, "/health/ready", "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, se esperaba 200", resp.StatusCode)
		}
	})

	t.Run("upstream caído", func(t *testing.T) {
		app := NuevaAplicacion(configuracionPrueba("http://127.0.0.1:1"), descartarRegistrador())

		resp := hacerSolicitudPrueba(t, app, http.MethodGet, "/health/ready", "", nil)
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("status = %d, se esperaba 503", resp.StatusCode)
		}

		var cuerpo RespuestaSalud
		decodificarCuerpo(t, resp, &cuerpo)
		if cuerpo.ServicioDependiente != "unreachable" {
			t.Errorf("upstream = %q, se esperaba \"unreachable\"", cuerpo.ServicioDependiente)
		}
	})
}

func TestRutaDesconocidaDevuelveErrorEstructurado(t *testing.T) {
	app := NuevaAplicacion(configuracionPrueba("http://unused"), descartarRegistrador())

	resp := hacerSolicitudPrueba(t, app, http.MethodGet, "/api/v1/inexistente", "", nil)

	verificarCodigoError(t, resp, http.StatusNotFound, CodigoNoEncontrado)
}

func TestTokenBearer(t *testing.T) {
	cases := []struct {
		name    string
		header  string
		want    string
		wantErr bool
	}{
		{"formato estándar", "Bearer abc.def.ghi", "abc.def.ghi", false},
		{"esquema en minúsculas", "bearer abc.def.ghi", "abc.def.ghi", false},
		{"esquema en mayúsculas", "BEARER abc.def.ghi", "abc.def.ghi", false},
		{"encabezado vacío", "", "", true},
		{"sin esquema", "abc.def.ghi", "", true},
		{"otro esquema", "Basic dXNlcjpwYXNz", "", true},
		{"token vacío", "Bearer ", "", true},
		{"solo espacios", "Bearer    ", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extraerTokenBearer(tc.header)

			if tc.wantErr {
				if err == nil {
					t.Errorf("se esperaba error para %q, se obtuvo %q", tc.header, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if got != tc.want {
				t.Errorf("token = %q, se esperaba %q", got, tc.want)
			}
		})
	}
}
