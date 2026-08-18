package config

import (
	"testing"
	"time"
)

// establecerEntorno aplica las variables dadas y las restaura al terminar el subtest.
// t.Setenv se encarga de la restauración e impide que el test corra en paralelo,
// que es justo lo que se necesita al manipular estado global del proceso.
func establecerEntorno(t *testing.T, vars map[string]string) {
	t.Helper()
	// Las claves obligatorias se limpian primero para que cada caso parta de un
	// entorno conocido, sin heredar lo que haya definido el shell.
	for _, key := range []string{
		"GO_API_PORT", "STATS_API_URL", "STATS_API_TIMEOUT_SECONDS", "STATS_API_MAX_RETRIES",
		"CORS_ALLOWED_ORIGINS",
		"MAX_MATRIX_DIMENSION", "JWT_SECRET", "JWT_ISSUER", "JWT_AUDIENCE", "JWT_TTL_MINUTES",
		"DEMO_USERNAME", "DEMO_PASSWORD",
	} {
		t.Setenv(key, "")
	}
	for key, value := range vars {
		t.Setenv(key, value)
	}
}

// entornoValido es el conjunto mínimo con el que Cargar debe tener éxito.
func entornoValido() map[string]string {
	return map[string]string{
		"JWT_SECRET":    "secreto-de-prueba",
		"DEMO_PASSWORD": "clave-de-prueba",
	}
}

func TestCargarAplicaValoresPredeterminados(t *testing.T) {
	establecerEntorno(t, entornoValido())

	cfg, err := Cargar()
	if err != nil {
		t.Fatalf("Cargar devolvió error: %v", err)
	}

	checks := []struct {
		name      string
		got, want any
	}{
		{"Port", cfg.Puerto, "8080"},
		{"StatsAPIURL", cfg.URLAPIEstadisticas, "http://localhost:3000"},
		{"StatsTimeout", cfg.TiempoEsperaEstadisticas, 5 * time.Second},
		{"StatsMaxRetries", cfg.MaximoReintentosEstadisticas, 1},
		{"MaxMatrixDimension", cfg.DimensionMaximaMatriz, 256},
		{"JWTIssuer", cfg.EmisorJWT, "interseguro-qr-api"},
		{"JWTAudience", cfg.AudienciaJWT, "interseguro-clients"},
		{"JWTTTL", cfg.VigenciaJWT, 15 * time.Minute},
		{"DemoUsername", cfg.UsuarioDemo, "demo"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, se esperaba %v", c.name, c.got, c.want)
		}
	}
}

func TestCargarSobrescribeDesdeEntorno(t *testing.T) {
	env := entornoValido()
	env["GO_API_PORT"] = "9090"
	env["STATS_API_URL"] = "http://api-node:3000"
	env["STATS_API_TIMEOUT_SECONDS"] = "12"
	env["MAX_MATRIX_DIMENSION"] = "64"
	env["JWT_TTL_MINUTES"] = "60"
	env["CORS_ALLOWED_ORIGINS"] = " https://app.ejemplo.cl, http://localhost:5173 "
	establecerEntorno(t, env)

	cfg, err := Cargar()
	if err != nil {
		t.Fatalf("Cargar devolvió error: %v", err)
	}

	if cfg.Puerto != "9090" {
		t.Errorf("Port = %q, se esperaba \"9090\"", cfg.Puerto)
	}
	if cfg.URLAPIEstadisticas != "http://api-node:3000" {
		t.Errorf("StatsAPIURL = %q", cfg.URLAPIEstadisticas)
	}
	if cfg.TiempoEsperaEstadisticas != 12*time.Second {
		t.Errorf("StatsTimeout = %v, se esperaba 12s", cfg.TiempoEsperaEstadisticas)
	}
	if cfg.DimensionMaximaMatriz != 64 {
		t.Errorf("MaxMatrixDimension = %d, se esperaba 64", cfg.DimensionMaximaMatriz)
	}
	if cfg.VigenciaJWT != time.Hour {
		t.Errorf("JWTTTL = %v, se esperaba 1h", cfg.VigenciaJWT)
	}
	if len(cfg.OrigenesCORS) != 2 || cfg.OrigenesCORS[0] != "https://app.ejemplo.cl" || cfg.OrigenesCORS[1] != "http://localhost:5173" {
		t.Errorf("OrigenesCORS = %v, se esperaban los dos orígenes recortados", cfg.OrigenesCORS)
	}
}

func TestCargarRecortaValoresDeEntorno(t *testing.T) {
	env := entornoValido()
	env["GO_API_PORT"] = " 9090 "
	env["STATS_API_URL"] = "  http://localhost:3000/  "
	env["DEMO_USERNAME"] = " demo "
	establecerEntorno(t, env)

	cfg, err := Cargar()
	if err != nil {
		t.Fatalf("Cargar devolviÃ³ error: %v", err)
	}

	if cfg.Puerto != "9090" {
		t.Errorf("Port = %q, se esperaba 9090", cfg.Puerto)
	}
	if cfg.URLAPIEstadisticas != "http://localhost:3000/" {
		t.Errorf("StatsAPIURL = %q", cfg.URLAPIEstadisticas)
	}
	if cfg.UsuarioDemo != "demo" {
		t.Errorf("DemoUsername = %q", cfg.UsuarioDemo)
	}
	if len(cfg.OrigenesCORS) != 4 {
		t.Errorf("OrigenesCORS contiene %d valores, se esperaban 4", len(cfg.OrigenesCORS))
	}
}

func TestCargarRechazaConfiguracionInvalida(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(map[string]string)
		wantFail bool
	}{
		{
			name:     "sin JWT_SECRET",
			mutate:   func(e map[string]string) { delete(e, "JWT_SECRET") },
			wantFail: true,
		},
		{
			name:     "sin DEMO_PASSWORD",
			mutate:   func(e map[string]string) { delete(e, "DEMO_PASSWORD") },
			wantFail: true,
		},
		{
			name:     "URL de estadísticas inválida",
			mutate:   func(e map[string]string) { e["STATS_API_URL"] = "localhost:3000" },
			wantFail: true,
		},
		{
			name:     "dimensión máxima en cero",
			mutate:   func(e map[string]string) { e["MAX_MATRIX_DIMENSION"] = "0" },
			wantFail: true,
		},
		{
			name:     "reintentos negativos",
			mutate:   func(e map[string]string) { e["STATS_API_MAX_RETRIES"] = "-1" },
			wantFail: true,
		},
		{
			name:     "timeout en cero",
			mutate:   func(e map[string]string) { e["STATS_API_TIMEOUT_SECONDS"] = "0" },
			wantFail: true,
		},
		{
			name:     "origen CORS inválido",
			mutate:   func(e map[string]string) { e["CORS_ALLOWED_ORIGINS"] = "app.ejemplo.cl" },
			wantFail: true,
		},
		{
			// Un valor no numérico cae al default documentado en lugar de
			// impedir el arranque: es un error de tipeo recuperable.
			name:     "timeout no numérico usa el valor por defecto",
			mutate:   func(e map[string]string) { e["STATS_API_TIMEOUT_SECONDS"] = "cinco" },
			wantFail: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := entornoValido()
			tc.mutate(env)
			establecerEntorno(t, env)

			_, err := Cargar()
			if tc.wantFail && err == nil {
				t.Error("se esperaba un error de configuración, Cargar tuvo éxito")
			}
			if !tc.wantFail && err != nil {
				t.Errorf("Cargar devolvió error inesperado: %v", err)
			}
		})
	}
}
