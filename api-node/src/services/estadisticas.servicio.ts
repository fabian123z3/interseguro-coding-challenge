/**
 * Cálculo de estadísticas sobre las matrices que devuelve la factorización QR.
 *
 * El módulo es lógica pura, sin dependencias de Express ni del entorno: recibe
 * matrices y devuelve un resultado. Eso permite probarlo exhaustivamente sin
 * levantar un servidor y reutilizarlo desde otro transporte si hiciera falta.
 */

/**
 * Factor de tolerancia relativa para decidir si un valor "es cero".
 *
 * La factorización QR produce residuos de redondeo: donde la matemática exacta
 * da 0, la aritmética de punto flotante deja valores del orden de 1e-16 veces
 * la escala de la matriz. Comparar con `=== 0` haría que ninguna matriz real
 * pareciera diagonal jamás.
 */
export const FACTOR_TOLERANCIA = 1e-9;

/**
 * Piso absoluto de la tolerancia. Sin él, una matriz de valores minúsculos
 * (o la matriz nula) tendría tolerancia 0 y volveríamos a comparar exacto.
 */
export const TOLERANCIA_MINIMA = 1e-12;

/** Estadísticas de una matriz individual. */
export interface EstadisticasMatriz {
  max: number;
  min: number;
  average: number;
  sum: number;
  /** Cantidad de elementos considerados (filas × columnas). */
  count: number;
  rows: number;
  cols: number;
  isSquare: boolean;
  isDiagonal: boolean;
  /** Tolerancia con la que se evaluó `isDiagonal`, para que el resultado sea auditable. */
  tolerance: number;
}

/** Estadísticas agregadas sobre el conjunto completo de matrices. */
export interface EstadisticasGlobales {
  max: number;
  min: number;
  average: number;
  sum: number;
  count: number;
}

/** Resultado completo del cálculo. */
export interface ResultadoEstadisticas {
  overall: EstadisticasGlobales;
  /** Desglose por matriz, en el mismo orden en que llegaron. */
  perMatrix: Record<string, EstadisticasMatriz>;
  /** True si al menos una de las matrices es diagonal. */
  anyDiagonal: boolean;
  /** Factor relativo usado para derivar la tolerancia de cada matriz. */
  toleranceFactor: number;
}

/**
 * Acumulador de suma con compensación de Neumaier.
 *
 * Sumar miles de valores de magnitudes distintas en punto flotante pierde
 * precisión: los términos pequeños se descartan al sumarse a un acumulador
 * grande. Este algoritmo conserva el error de cada paso en una variable de
 * compensación y lo reintegra al final, con un coste de tres operaciones extra
 * por elemento. Es la variante de Kahan-Babuška, que a diferencia del Kahan
 * clásico también funciona cuando el término entrante es mayor que la suma
 * acumulada, caso frecuente al mezclar los valores de Q (~1) con los de R.
 */
class SumaCompensada {
  #suma = 0;
  #compensacion = 0;

  agregar(valor: number): void {
    const total = this.#suma + valor;
    this.#compensacion +=
      Math.abs(this.#suma) >= Math.abs(valor)
        ? this.#suma - total + valor
        : valor - total + this.#suma;
    this.#suma = total;
  }

  get valor(): number {
    return this.#suma + this.#compensacion;
  }
}

/**
 * Calcula las estadísticas de una matriz en un solo recorrido.
 *
 * Se asume que la matriz ya fue validada (rectangular, no vacía y con valores
 * finitos): la validación vive en la capa de esquema, no acá.
 */
export function calcularEstadisticasMatriz(matriz: number[][]): EstadisticasMatriz {
  const filas = matriz.length;
  const primeraFila = matriz[0];
  if (!primeraFila || primeraFila.length === 0) {
    throw new Error('la matriz validada debe contener al menos un valor');
  }
  const columnas = primeraFila.length;

  let maximo = Number.NEGATIVE_INFINITY;
  let minimo = Number.POSITIVE_INFINITY;
  let maximoAbsoluto = 0;
  const suma = new SumaCompensada();

  for (const fila of matriz) {
    for (const valor of fila) {
      if (valor > maximo) maximo = valor;
      if (valor < minimo) minimo = valor;

      const absoluto = Math.abs(valor);
      if (absoluto > maximoAbsoluto) maximoAbsoluto = absoluto;

      suma.agregar(valor);
    }
  }

  // La tolerancia se deriva de la escala de esta matriz en particular, no del
  // conjunto. Una tolerancia global tomada de la matriz de mayor magnitud haría
  // que en una matriz de valores pequeños se descartaran como "ruido" valores
  // que en su propia escala son perfectamente significativos.
  const tolerancia = Math.max(TOLERANCIA_MINIMA, FACTOR_TOLERANCIA * maximoAbsoluto);
  const cantidad = filas * columnas;

  return {
    max: maximo,
    min: minimo,
    average: suma.valor / cantidad,
    sum: suma.valor,
    count: cantidad,
    rows: filas,
    cols: columnas,
    isSquare: filas === columnas,
    isDiagonal: esDiagonal(matriz, tolerancia),
    tolerance: tolerancia,
  };
}

/**
 * Indica si todos los elementos fuera de la diagonal principal son cero dentro
 * de la tolerancia dada.
 *
 * Se usa la definición generalizada `a[i][j] ≈ 0 para todo i ≠ j`, que también
 * aplica a matrices rectangulares. La definición estricta de "matriz diagonal"
 * exige además que sea cuadrada; por eso el resultado se acompaña siempre del
 * campo `isSquare`, y quien necesite el criterio estricto puede exigir ambos.
 *
 * Casos límite, ambos correctos por la definición: la matriz nula es diagonal
 * (no tiene ningún elemento no nulo fuera de la diagonal) y una matriz de 1×1
 * también lo es (no tiene elementos fuera de la diagonal).
 */
export function esDiagonal(matriz: number[][], tolerancia: number): boolean {
  for (const [i, fila] of matriz.entries()) {
    for (const [j, valor] of fila.entries()) {
      if (i !== j && Math.abs(valor) > tolerancia) {
        return false;
      }
    }
  }
  return true;
}

/**
 * Calcula las estadísticas de cada matriz y las agregadas sobre el conjunto.
 *
 * El enunciado pide "el valor máximo encontrado en las matrices" (en plural),
 * de modo que `overall` recorre todas juntas. El desglose por matriz se entrega
 * además porque el recorrido ya está hecho y responde la pregunta natural
 * siguiente: cuál de las dos aporta cada extremo.
 */
export function calcularEstadisticas(matrices: Record<string, number[][]>): ResultadoEstadisticas {
  const porMatriz: Record<string, EstadisticasMatriz> = {};

  let maximo = Number.NEGATIVE_INFINITY;
  let minimo = Number.POSITIVE_INFINITY;
  let cantidad = 0;
  let algunaDiagonal = false;
  const total = new SumaCompensada();

  for (const [nombre, matriz] of Object.entries(matrices)) {
    const estadisticas = calcularEstadisticasMatriz(matriz);
    porMatriz[nombre] = estadisticas;

    // El agregado se compone a partir de los parciales en lugar de recorrer las
    // matrices otra vez: max, min, suma y conteo son todos asociativos.
    if (estadisticas.max > maximo) maximo = estadisticas.max;
    if (estadisticas.min < minimo) minimo = estadisticas.min;
    total.agregar(estadisticas.sum);
    cantidad += estadisticas.count;
    algunaDiagonal ||= estadisticas.isDiagonal;
  }

  return {
    overall: {
      max: maximo,
      min: minimo,
      average: total.valor / cantidad,
      sum: total.valor,
      count: cantidad,
    },
    perMatrix: porMatriz,
    anyDiagonal: algunaDiagonal,
    toleranceFactor: FACTOR_TOLERANCIA,
  };
}
