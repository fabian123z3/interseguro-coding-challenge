package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testSecret   = "secreto-de-prueba-suficientemente-largo"
	testIssuer   = "test-issuer"
	testAudience = "test-audience"
)

func nuevoGestorPrueba(ttl time.Duration) *Gestor {
	return NuevoGestor(testSecret, testIssuer, testAudience, ttl)
}

func TestEmitirYVerificar(t *testing.T) {
	m := nuevoGestorPrueba(15 * time.Minute)

	token, expiresAt, err := m.Emitir("demo")
	if err != nil {
		t.Fatalf("Issue devolvió error: %v", err)
	}
	if token == "" {
		t.Fatal("Issue devolvió un token vacío")
	}
	if time.Until(expiresAt) <= 0 {
		t.Errorf("el token nace expirado: expiresAt = %v", expiresAt)
	}

	subject, err := m.Verificar(token)
	if err != nil {
		t.Fatalf("Verify rechazó un token recién emitido: %v", err)
	}
	if subject != "demo" {
		t.Errorf("subject = %q, se esperaba %q", subject, "demo")
	}
}

func TestVerificarRechazaTokenExpirado(t *testing.T) {
	// TTL negativo: el token nace vencido, sin necesidad de esperar en el test.
	m := nuevoGestorPrueba(-time.Minute)

	token, _, err := m.Emitir("demo")
	if err != nil {
		t.Fatalf("Issue devolvió error: %v", err)
	}

	_, err = m.Verificar(token)
	if !errors.Is(err, ErrTokenExpirado) {
		t.Errorf("error = %v, se esperaba ErrTokenExpirado", err)
	}
}

func TestVerificarRechazaSecretoIncorrecto(t *testing.T) {
	issuer := nuevoGestorPrueba(time.Hour)
	token, _, _ := issuer.Emitir("demo")

	verifier := NuevoGestor("otro-secreto-distinto", testIssuer, testAudience, time.Hour)

	if _, err := verifier.Verificar(token); !errors.Is(err, ErrTokenInvalido) {
		t.Errorf("error = %v, se esperaba ErrTokenInvalido", err)
	}
}

// TestVerificarRechazaAlgoritmoNulo cubre el ataque clásico contra JWT: presentar un
// token con `alg: none` y sin firma para que el verificador lo acepte. La
// restricción explícita de algoritmos en Verify debe bloquearlo.
func TestVerificarRechazaAlgoritmoNulo(t *testing.T) {
	claims := jwt.RegisteredClaims{
		Subject:   "atacante",
		Issuer:    testIssuer,
		Audience:  jwt.ClaimStrings{testAudience},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("no se pudo construir el token del caso de prueba: %v", err)
	}

	if _, err := nuevoGestorPrueba(time.Hour).Verificar(unsigned); err == nil {
		t.Fatal("se aceptó un token firmado con alg=none")
	}
}

func TestVerificarRechazaEmisorYAudienciaIncorrectos(t *testing.T) {
	cases := []struct {
		name             string
		issuer, audience string
	}{
		{"emisor distinto", "otro-issuer", testAudience},
		{"audiencia distinta", testIssuer, "otra-audiencia"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			foreign := NuevoGestor(testSecret, tc.issuer, tc.audience, time.Hour)
			token, _, _ := foreign.Emitir("demo")

			if _, err := nuevoGestorPrueba(time.Hour).Verificar(token); !errors.Is(err, ErrTokenInvalido) {
				t.Errorf("error = %v, se esperaba ErrTokenInvalido", err)
			}
		})
	}
}

func TestVerificarRechazaTokenMalformado(t *testing.T) {
	cases := []string{
		"",
		"no-es-un-jwt",
		"a.b.c",
		"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJkZW1vIn0", // sin firma
	}

	m := nuevoGestorPrueba(time.Hour)
	for _, token := range cases {
		t.Run(token, func(t *testing.T) {
			if _, err := m.Verificar(token); err == nil {
				t.Errorf("se aceptó el token malformado %q", token)
			}
		})
	}
}

func TestVigencia(t *testing.T) {
	want := 42 * time.Minute
	if got := nuevoGestorPrueba(want).Vigencia(); got != want {
		t.Errorf("TTL = %v, se esperaba %v", got, want)
	}
}
