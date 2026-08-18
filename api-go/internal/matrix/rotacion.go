package matrix

// Rotar90 devuelve la matriz rotada 90° en sentido horario: una matriz de m×n
// produce una de n×m.
//
// Este endpoint existe por una razón de interpretación del enunciado, no
// matemática. El PDF del desafío se contradice: la sección "Arquitectura"
// habla de "realizar la rotación de la matriz", mientras que "Funcionalidad
// requerida" pide explícitamente "la factorización QR de dicha matriz". Se
// implementó QR por ser el requisito funcional explícito, y esta rotación
// clásica queda disponible para cubrir la lectura alternativa sin ambigüedad.
// Ver docs/DECISIONES.md.
func Rotar90(m Matriz) Matriz {
	filas, columnas := m.Filas(), m.Columnas()
	salida := Nueva(columnas, filas)
	for i := 0; i < filas; i++ {
		for j := 0; j < columnas; j++ {
			// El elemento (i, j) pasa a la fila j y a la columna espejada
			// rows−1−i: la primera fila del original se convierte en la última
			// columna del resultado.
			salida[j][filas-1-i] = m[i][j]
		}
	}
	return salida
}
