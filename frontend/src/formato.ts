/**
 * Formateo de números para la interfaz.
 *
 * La factorización devuelve valores como -0.857142857142857 y 1.6e-16. Volcarlos
 * crudos en la grilla la vuelve ilegible y esconde la estructura del resultado,
 * que es justo lo que se quiere ver.
 */

/** Umbral bajo el cual conviene la notación exponencial. */
const UMBRAL_PEQUENO = 1e-4;
/** Umbral sobre el cual conviene la notación exponencial. */
const UMBRAL_GRANDE = 1e6;

/**
 * Formatea un valor con la cantidad de decimales pedida.
 *
 * Los valores muy pequeños o muy grandes pasan a notación exponencial: con
 * decimales fijos, 1.6e-16 se mostraría como "0.0000" y parecería un cero
 * exacto, que es precisamente la distinción que importa al leer una R.
 */
export function formatearValor(valor: number, decimales: number): string {
  if (Object.is(valor, -0) || valor === 0) return '0';
  if (!Number.isFinite(valor)) return String(valor);

  const magnitud = Math.abs(valor);
  if (magnitud < UMBRAL_PEQUENO || magnitud >= UMBRAL_GRANDE) {
    return valor.toExponential(Math.min(decimales, 4));
  }
  return valor.toFixed(decimales);
}

/** Formatea un valor para las tarjetas de estadísticas, siempre legible. */
export function formatearEstadistica(valor: number): string {
  if (!Number.isFinite(valor)) return '—';
  if (valor === 0) return '0';

  const magnitud = Math.abs(valor);
  if (magnitud < 1e-3 || magnitud >= 1e7) return valor.toExponential(3);

  // Hasta cuatro decimales, sin ceros de relleno a la derecha.
  return Number(valor.toFixed(4)).toString();
}

/**
 * Intensidad relativa de una celda, entre 0 y 1, usada para el sombreado que
 * revela la estructura de la matriz de un vistazo.
 *
 * La escala es la raíz cuadrada de la proporción: una escala lineal deja casi
 * invisible todo lo que no sea el máximo, porque en una matriz típica la mayoría
 * de los valores están muy por debajo del mayor.
 */
export function intensidad(valor: number, maximoAbsoluto: number): number {
  if (maximoAbsoluto === 0) return 0;
  return Math.sqrt(Math.min(Math.abs(valor) / maximoAbsoluto, 1));
}

/** Mayor valor absoluto de la matriz, escala de referencia del sombreado. */
export function maximoAbsoluto(matriz: number[][]): number {
  let maximo = 0;
  for (const fila of matriz) {
    for (const valor of fila) {
      const absoluto = Math.abs(valor);
      if (absoluto > maximo) maximo = absoluto;
    }
  }
  return maximo;
}
