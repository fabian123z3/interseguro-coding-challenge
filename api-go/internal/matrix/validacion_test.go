package matrix

import (
	"math"
	"testing"
)

func TestValidar(t *testing.T) {
	const maxDim = 256

	cases := []struct {
		name     string
		input    Matriz
		maxDim   int
		wantCode CodigoError // vacío significa que se espera que la matriz sea válida
	}{
		{
			name:   "matriz cuadrada válida",
			input:  Matriz{{1, 2}, {3, 4}},
			maxDim: maxDim,
		},
		{
			name:   "matriz rectangular válida",
			input:  Matriz{{1, 2, 3}, {4, 5, 6}},
			maxDim: maxDim,
		},
		{
			name:   "matriz de un solo elemento",
			input:  Matriz{{42}},
			maxDim: maxDim,
		},
		{
			name:     "sin filas",
			input:    Matriz{},
			maxDim:   maxDim,
			wantCode: CodigoMatrizVacia,
		},
		{
			name:     "fila sin columnas",
			input:    Matriz{{}},
			maxDim:   maxDim,
			wantCode: CodigoMatrizVacia,
		},
		{
			name:     "filas de distinto largo",
			input:    Matriz{{1, 2, 3}, {4, 5}},
			maxDim:   maxDim,
			wantCode: CodigoFilasIrregulares,
		},
		{
			name:     "fila vacía intercalada",
			input:    Matriz{{1, 2}, {}},
			maxDim:   maxDim,
			wantCode: CodigoFilasIrregulares,
		},
		{
			name:     "contiene NaN",
			input:    Matriz{{1, 2}, {3, math.NaN()}},
			maxDim:   maxDim,
			wantCode: CodigoValorNoFinito,
		},
		{
			name:     "contiene infinito positivo",
			input:    Matriz{{math.Inf(1), 2}},
			maxDim:   maxDim,
			wantCode: CodigoValorNoFinito,
		},
		{
			name:     "contiene infinito negativo",
			input:    Matriz{{1, math.Inf(-1)}},
			maxDim:   maxDim,
			wantCode: CodigoValorNoFinito,
		},
		{
			name:     "demasiadas filas",
			input:    Nueva(5, 2),
			maxDim:   4,
			wantCode: CodigoMatrizDemasiadoGrande,
		},
		{
			name:     "demasiadas columnas",
			input:    Nueva(2, 5),
			maxDim:   4,
			wantCode: CodigoMatrizDemasiadoGrande,
		},
		{
			name:   "justo en el límite",
			input:  Nueva(4, 4),
			maxDim: 4,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validar(tc.input, tc.maxDim)

			if tc.wantCode == "" {
				if err != nil {
					t.Fatalf("se esperaba una matriz válida, se obtuvo %s: %s", err.Codigo, err.Mensaje)
				}
				return
			}

			if err == nil {
				t.Fatalf("se esperaba el error %s, la matriz fue aceptada", tc.wantCode)
			}
			if err.Codigo != tc.wantCode {
				t.Errorf("código = %s, se esperaba %s (mensaje: %s)", err.Codigo, tc.wantCode, err.Mensaje)
			}
			if err.Mensaje == "" {
				t.Error("el error no trae mensaje legible")
			}
		})
	}
}

// TestValidarDetallesError comprueba que el error posicional sea accionable:
// debe indicar qué fila rompe el rectángulo, no solo que algo falló.
func TestValidarDetallesError(t *testing.T) {
	err := Validar(Matriz{{1, 2, 3}, {4, 5, 6}, {7, 8}}, 256)
	if err == nil {
		t.Fatal("se esperaba el error RAGGED_ROWS")
	}
	if got := err.Detalles["rowIndex"]; got != 2 {
		t.Errorf("rowIndex = %v, se esperaba 2", got)
	}
	if got := err.Detalles["expectedCols"]; got != 3 {
		t.Errorf("expectedCols = %v, se esperaba 3", got)
	}
	if got := err.Detalles["actualCols"]; got != 2 {
		t.Errorf("actualCols = %v, se esperaba 2", got)
	}
}
