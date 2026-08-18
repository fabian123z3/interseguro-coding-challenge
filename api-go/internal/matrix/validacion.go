package matrix

import (
	"fmt"
	"math"
)

// CodigoError identifica de forma estable el motivo por el que una matriz fue
// rechazada. El cliente puede ramificar sobre este código sin tener que parsear
// el mensaje, que está pensado para humanos y puede cambiar.
type CodigoError string

const (
	// CodigoMatrizVacia: la matriz no tiene filas, o tiene filas sin columnas.
	CodigoMatrizVacia CodigoError = "EMPTY_MATRIX"
	// CodeRaggedRows: las filas no tienen todas el mismo largo, por lo que la
	// entrada no representa una matriz rectangular.
	CodigoFilasIrregulares CodigoError = "RAGGED_ROWS"
	// CodeNonFiniteValue: la matriz contiene NaN o ±Inf. Ambos contaminarían
	// todo el resultado de la factorización, así que se rechazan en la entrada.
	CodigoValorNoFinito CodigoError = "NON_FINITE_VALUE"
	// CodeMatrixTooLarge: la matriz supera el límite configurado de dimensión.
	CodigoMatrizDemasiadoGrande CodigoError = "MATRIX_TOO_LARGE"
)

// ErrorValidacion describe una entrada inválida. Detalles lleva la información
// posicional necesaria para que el usuario corrija el problema sin adivinar.
type ErrorValidacion struct {
	Codigo   CodigoError    `json:"code"`
	Mensaje  string         `json:"message"`
	Detalles map[string]any `json:"details,omitempty"`
}

func (e *ErrorValidacion) Error() string { return e.Mensaje }

// Validar comprueba que la entrada sea una matriz rectangular utilizable.
//
// maxDimension acota tanto filas como columnas. Es una defensa deliberada: el
// coste de la factorización crece como O(m·n²), de modo que sin un límite un
// único request podría monopolizar la CPU del servicio.
func Validar(m Matriz, dimensionMaxima int) *ErrorValidacion {
	filas := len(m)
	if filas == 0 {
		return &ErrorValidacion{
			Codigo:  CodigoMatrizVacia,
			Mensaje: "la matriz debe tener al menos una fila",
		}
	}

	columnas := len(m[0])
	if columnas == 0 {
		return &ErrorValidacion{
			Codigo:  CodigoMatrizVacia,
			Mensaje: "la matriz debe tener al menos una columna",
		}
	}

	if filas > dimensionMaxima || columnas > dimensionMaxima {
		return &ErrorValidacion{
			Codigo: CodigoMatrizDemasiadoGrande,
			Mensaje: fmt.Sprintf(
				"la matriz de %d×%d supera el límite de %d por dimensión",
				filas, columnas, dimensionMaxima,
			),
			Detalles: map[string]any{
				"rows": filas, "cols": columnas, "maxDimension": dimensionMaxima,
			},
		}
	}

	for i, fila := range m {
		if len(fila) != columnas {
			return &ErrorValidacion{
				Codigo: CodigoFilasIrregulares,
				Mensaje: fmt.Sprintf(
					"todas las filas deben tener el mismo largo: la fila 0 tiene %d columnas y la fila %d tiene %d",
					columnas, i, len(fila),
				),
				Detalles: map[string]any{
					"expectedCols": columnas, "rowIndex": i, "actualCols": len(fila),
				},
			}
		}
		for j, valor := range fila {
			if math.IsNaN(valor) || math.IsInf(valor, 0) {
				return &ErrorValidacion{
					Codigo: CodigoValorNoFinito,
					Mensaje: fmt.Sprintf(
						"la posición [%d][%d] contiene un valor no finito (NaN o infinito)",
						i, j,
					),
					Detalles: map[string]any{"rowIndex": i, "colIndex": j},
				}
			}
		}
	}

	return nil
}
