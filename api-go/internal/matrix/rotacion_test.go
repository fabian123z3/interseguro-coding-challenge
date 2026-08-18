package matrix

import "testing"

func TestRotar90(t *testing.T) {
	cases := []struct {
		name  string
		input Matriz
		want  Matriz
	}{
		{
			name:  "cuadrada",
			input: Matriz{{1, 2}, {3, 4}},
			want:  Matriz{{3, 1}, {4, 2}},
		},
		{
			name:  "rectangular ancha se vuelve alta",
			input: Matriz{{1, 2, 3}, {4, 5, 6}},
			want:  Matriz{{4, 1}, {5, 2}, {6, 3}},
		},
		{
			name:  "vector fila se vuelve columna",
			input: Matriz{{1, 2, 3}},
			want:  Matriz{{1}, {2}, {3}},
		},
		{
			name:  "un solo elemento",
			input: Matriz{{7}},
			want:  Matriz{{7}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Rotar90(tc.input)

			verificarDimensiones(t, "rotada", got, tc.want.Filas(), tc.want.Columnas())
			for i := range tc.want {
				for j := range tc.want[i] {
					if got[i][j] != tc.want[i][j] {
						t.Errorf("rotada[%d][%d] = %g, se esperaba %g", i, j, got[i][j], tc.want[i][j])
					}
				}
			}
		})
	}
}

// TestRotate90FourTimesIsIdentity comprueba la propiedad de grupo: cuatro
// rotaciones de 90° devuelven la matriz original.
func TestCuatroRotacionesRecuperanIdentidad(t *testing.T) {
	original := Matriz{{1, 2, 3}, {4, 5, 6}}

	got := Rotar90(Rotar90(Rotar90(Rotar90(original))))

	verificarDimensiones(t, "rotada 4 veces", got, original.Filas(), original.Columnas())
	for i := range original {
		for j := range original[i] {
			if got[i][j] != original[i][j] {
				t.Errorf("[%d][%d] = %g, se esperaba %g", i, j, got[i][j], original[i][j])
			}
		}
	}
}
