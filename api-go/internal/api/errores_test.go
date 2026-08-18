package api

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestEstadoDesdeError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"APIError", NuevoErrorAPI(http.StatusBadGateway, CodigoErrorServicio, "upstream", nil), http.StatusBadGateway},
		{"FiberError", fiber.ErrNotFound, http.StatusNotFound},
		{"desconocido", errors.New("fallo"), http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := estadoDesdeError(tc.err); got != tc.want {
				t.Fatalf("estadoDesdeError() = %d, se esperaba %d", got, tc.want)
			}
		})
	}
}
