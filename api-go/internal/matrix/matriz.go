// Package matrix implementa las operaciones de álgebra lineal del desafío:
// validación de matrices rectangulares y factorización QR mediante reflexiones
// de Householder.
//
// El paquete es deliberadamente independiente del framework HTTP y no usa
// librerías externas de álgebra lineal: así la lógica numérica puede probarse
// de forma aislada y el algoritmo queda a la vista, que es lo que evalúa el
// desafío.
package matrix

import "math"

// Matriz representa una matriz densa de m filas por n columnas, almacenada como
// un slice de filas. Es exactamente la forma en que viaja por JSON
// (`[[1,2],[3,4]]`), lo que evita conversiones en la capa de transporte.
type Matriz [][]float64

// Nueva construye una matriz de filas×columnas inicializada en ceros.
func Nueva(filas, columnas int) Matriz {
	m := make(Matriz, filas)
	// Una única reserva contigua para todos los elementos: mejora la localidad
	// de caché frente a asignar cada fila por separado.
	respaldo := make([]float64, filas*columnas)
	for i := range m {
		m[i] = respaldo[i*columnas : (i+1)*columnas : (i+1)*columnas]
	}
	return m
}

// Identidad construye la matriz identidad de tamaño n×n.
func Identidad(n int) Matriz {
	m := Nueva(n, n)
	for i := 0; i < n; i++ {
		m[i][i] = 1
	}
	return m
}

// Filas devuelve la cantidad de filas (m).
func (m Matriz) Filas() int { return len(m) }

// Columnas devuelve la cantidad de columnas (n). Asume que la matriz ya pasó
// por Validar, es decir, que todas las filas tienen el mismo largo.
func (m Matriz) Columnas() int {
	if len(m) == 0 {
		return 0
	}
	return len(m[0])
}

// Clonar devuelve una copia profunda, para no mutar la matriz de entrada.
func (m Matriz) Clonar() Matriz {
	salida := Nueva(m.Filas(), m.Columnas())
	for i := range m {
		copy(salida[i], m[i])
	}
	return salida
}

// Transponer devuelve la transpuesta de la matriz.
func (m Matriz) Transponer() Matriz {
	salida := Nueva(m.Columnas(), m.Filas())
	for i := range m {
		for j := range m[i] {
			salida[j][i] = m[i][j]
		}
	}
	return salida
}

// Multiplicar devuelve el producto matricial m·otra, o nil si las dimensiones no son
// compatibles. Se usa para verificar la factorización (Q·R debe reconstruir A)
// y en el endpoint de diagnóstico.
func (m Matriz) Multiplicar(otra Matriz) Matriz {
	if m.Columnas() != otra.Filas() {
		return nil
	}
	salida := Nueva(m.Filas(), otra.Columnas())
	for i := 0; i < m.Filas(); i++ {
		for k := 0; k < m.Columnas(); k++ {
			// Recorrido i-k-j: mantiene fija la fila de `other` en el bucle
			// interno y recorre memoria contigua en ambas matrices.
			a := m[i][k]
			if a == 0 {
				continue
			}
			for j := 0; j < otra.Columnas(); j++ {
				salida[i][j] += a * otra[k][j]
			}
		}
	}
	return salida
}

// MaximoAbsoluto devuelve el mayor valor absoluto presente en la matriz. Sirve como
// escala de referencia para construir tolerancias relativas.
func (m Matriz) MaximoAbsoluto() float64 {
	maximo := 0.0
	for i := range m {
		for _, v := range m[i] {
			if absoluto := math.Abs(v); absoluto > maximo {
				maximo = absoluto
			}
		}
	}
	return maximo
}

// EsTriangularSuperior indica si todos los elementos bajo la diagonal principal
// son cero dentro de la tolerancia dada.
func (m Matriz) EsTriangularSuperior(tolerancia float64) bool {
	for i := range m {
		for j := 0; j < i && j < len(m[i]); j++ {
			if math.Abs(m[i][j]) > tolerancia {
				return false
			}
		}
	}
	return true
}

// normalizarCeroNegativo reemplaza -0.0 por 0.0 en toda la matriz.
//
// El cero negativo es un valor válido de IEEE-754 y aparece de forma natural al
// multiplicar por cero, pero se serializa como `-0` en JSON, lo que resulta
// confuso para quien consume la API. Como -0.0 == 0.0 en toda comparación
// aritmética, normalizarlo no altera ningún resultado.
func (m Matriz) normalizarCeroNegativo() {
	for i := range m {
		for j := range m[i] {
			if m[i][j] == 0 {
				m[i][j] = 0
			}
		}
	}
}
