package client

import (
	"testing"
	"time"
)

func TestNuevoClienteEstadisticasNormalizaURLBase(t *testing.T) {
	cliente := NuevoClienteEstadisticas("  http://localhost:3000/  ", time.Second, 0, nil)

	if cliente.urlBase != "http://localhost:3000" {
		t.Fatalf("baseURL = %q, se esperaba http://localhost:3000", cliente.urlBase)
	}
}
