// Package auth encapsula la emisión y verificación de tokens JWT.
//
// Se aísla del framework HTTP a propósito: el middleware de Fiber y el handler
// de login consumen el mismo Manager, de modo que existe un único lugar donde
// se define qué es un token válido.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrTokenExpired indica un token bien formado y correctamente firmado
	// cuya vigencia ya venció. Se distingue del token inválido porque el
	// cliente puede resolverlo simplemente renovando la sesión.
	ErrTokenExpirado = errors.New("el token expiró")
	// ErrTokenInvalid cubre firma incorrecta, algoritmo no permitido, claims
	// que no coinciden y token malformado.
	ErrTokenInvalido = errors.New("el token es inválido")
)

// Gestor emite y verifica tokens HS256.
type Gestor struct {
	secreto   []byte
	emisor    string
	audiencia string
	vigencia  time.Duration
}

// NuevoGestor construye un Gestor con el secreto compartido y los claims de
// identidad del emisor.
func NuevoGestor(secreto, emisor, audiencia string, vigencia time.Duration) *Gestor {
	return &Gestor{
		secreto:   []byte(secreto),
		emisor:    emisor,
		audiencia: audiencia,
		vigencia:  vigencia,
	}
}

// Vigencia expone el plazo configurado, para informarlo en la respuesta de acceso.
func (g *Gestor) Vigencia() time.Duration { return g.vigencia }

// Emitir firma un token para el sujeto dado y devuelve también su instante de
// expiración.
func (g *Gestor) Emitir(sujeto string) (string, time.Time, error) {
	ahora := time.Now()
	expiraEn := ahora.Add(g.vigencia)

	declaraciones := jwt.RegisteredClaims{
		Subject:   sujeto,
		Issuer:    g.emisor,
		Audience:  jwt.ClaimStrings{g.audiencia},
		IssuedAt:  jwt.NewNumericDate(ahora),
		NotBefore: jwt.NewNumericDate(ahora),
		ExpiresAt: jwt.NewNumericDate(expiraEn),
	}

	firmado, err := jwt.NewWithClaims(jwt.SigningMethodHS256, declaraciones).SignedString(g.secreto)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("firmando el token: %w", err)
	}
	return firmado, expiraEn, nil
}

// Verificar valida la firma y los claims del token, devolviendo el sujeto.
//
// El algoritmo se restringe explícitamente a HS256: sin esa restricción, un
// atacante podría presentar un token con `alg: none` o forzar una confusión
// entre algoritmos simétricos y asimétricos para que la clave pública se
// interprete como secreto HMAC.
func (g *Gestor) Verificar(token string) (string, error) {
	analizado, err := jwt.ParseWithClaims(
		token,
		&jwt.RegisteredClaims{},
		func(t *jwt.Token) (any, error) { return g.secreto, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(g.emisor),
		jwt.WithAudience(g.audiencia),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", ErrTokenExpirado
		}
		return "", fmt.Errorf("%w: %v", ErrTokenInvalido, err)
	}

	declaraciones, ok := analizado.Claims.(*jwt.RegisteredClaims)
	if !ok || declaraciones.Subject == "" {
		return "", ErrTokenInvalido
	}
	return declaraciones.Subject, nil
}
