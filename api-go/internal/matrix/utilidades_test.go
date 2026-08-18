package matrix

import (
	"math"
	"testing"
)

// tolerancia usada al comparar resultados numéricos. Es varios órdenes de
// magnitud mayor que el épsilon de máquina (2.2e-16) para absorber el error
// acumulado de las operaciones, pero lo bastante estricta como para detectar
// un algoritmo realmente incorrecto.
const tolerancia = 1e-10

// verificarDimensiones falla si la matriz no tiene el tamaño esperado.
func verificarDimensiones(t *testing.T, nombre string, m Matriz, filas, columnas int) {
	t.Helper()
	if m.Filas() != filas || m.Columnas() != columnas {
		t.Errorf("%s: se esperaba %d×%d, se obtuvo %d×%d", nombre, filas, columnas, m.Filas(), m.Columnas())
	}
}

// verificarColumnasOrtonormales verifica que QᵀQ = I, es decir, que las columnas de
// Q sean ortogonales entre sí y de norma 1. Es la propiedad que define a Q y la
// que se degrada primero cuando el algoritmo es numéricamente inestable.
func verificarColumnasOrtonormales(t *testing.T, q Matriz) {
	t.Helper()
	gram := q.Transponer().Multiplicar(q)
	for i := range gram {
		for j := range gram[i] {
			esperado := 0.0
			if i == j {
				esperado = 1.0
			}
			if math.Abs(gram[i][j]-esperado) > tolerancia {
				t.Errorf("QᵀQ[%d][%d] = %g, se esperaba %g: Q no es ortogonal", i, j, gram[i][j], esperado)
			}
		}
	}
}

// verificarReconstruccion verifica que Q·R devuelva la matriz original.
func verificarReconstruccion(t *testing.T, a, q, r Matriz) {
	t.Helper()
	producto := q.Multiplicar(r)
	if producto == nil {
		t.Fatalf("Q(%d×%d)·R(%d×%d): dimensiones incompatibles", q.Filas(), q.Columnas(), r.Filas(), r.Columnas())
	}
	verificarDimensiones(t, "Q·R", producto, a.Filas(), a.Columnas())

	// La comparación es relativa a la escala de la matriz: un error absoluto de
	// 1e-6 es irrelevante en una matriz de valores ~1e150 y catastrófico en una
	// de valores ~1e-150.
	escala := math.Max(a.MaximoAbsoluto(), 1)
	for i := range a {
		for j := range a[i] {
			if math.Abs(producto[i][j]-a[i][j]) > tolerancia*escala {
				t.Errorf("Q·R[%d][%d] = %g, se esperaba %g", i, j, producto[i][j], a[i][j])
			}
		}
	}
}
