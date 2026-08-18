/**
 * Validación del cuerpo de POST /api/v1/statistics.
 *
 * Los errores de forma se detectan con Zod, pero cada regla de dominio adjunta
 * su propio código estable en `params`. Así el cliente recibe RAGGED_ROWS en
 * lugar de un mensaje genérico de validación, y —lo que importa más— recibe
 * exactamente los mismos códigos que emite la API Go ante el mismo problema.
 */

import { z } from 'zod';
import { ErrorApi, CodigoError, type ValorCodigoError } from '../errores.js';

/** Información que cada regla adjunta a su issue para poder traducirlo. */
interface ParametrosProblema {
  codigoError: ValorCodigoError;
  detalles?: Record<string, unknown>;
}

/** Límites aplicados a la entrada. */
export interface LimitesEsquema {
  dimensionMaximaMatriz: number;
  maximoMatrices: number;
}

/** Cuerpo válido de la petición. */
export interface SolicitudEstadisticas {
  matrices: Record<string, number[][]>;
}

/**
 * Construye el esquema con los límites de la configuración.
 *
 * Los límites se inyectan en vez de leerse del entorno acá para que los tests
 * puedan ejercitar los rechazos con matrices pequeñas.
 */
export function crearEsquemaEstadisticas(limites: LimitesEsquema) {
  const esquemaMatriz = z
    .array(z.array(z.number()))
    .superRefine((filas: number[][], contexto: z.RefinementCtx) => {
      if (filas.length === 0) {
        agregarProblema(contexto, CodigoError.EMPTY_MATRIX, 'la matriz debe tener al menos una fila');
        return;
      }

      const primeraFila = filas[0];
      if (!primeraFila) {
        agregarProblema(contexto, CodigoError.EMPTY_MATRIX, 'la matriz debe tener al menos una fila');
        return;
      }

      const columnas = primeraFila.length;
      if (columnas === 0) {
        agregarProblema(contexto, CodigoError.EMPTY_MATRIX, 'la matriz debe tener al menos una columna');
        return;
      }

      if (filas.length > limites.dimensionMaximaMatriz || columnas > limites.dimensionMaximaMatriz) {
        agregarProblema(
          contexto,
          CodigoError.MATRIX_TOO_LARGE,
          `la matriz de ${filas.length}×${columnas} supera el límite de ${limites.dimensionMaximaMatriz} por dimensión`,
          { rows: filas.length, cols: columnas, maxDimension: limites.dimensionMaximaMatriz },
        );
        return;
      }

      for (const [i, fila] of filas.entries()) {
        if (fila.length !== columnas) {
          agregarProblema(
            contexto,
            CodigoError.RAGGED_ROWS,
            `todas las filas deben tener el mismo largo: la fila 0 tiene ${columnas} columnas y la fila ${i} tiene ${fila.length}`,
            { expectedCols: columnas, rowIndex: i, actualCols: fila.length },
          );
          return;
        }

        // Defensa en profundidad: JSON no tiene literales para NaN ni infinito,
        // pero el módulo puede invocarse desde otro transporte.
        for (const [j, valor] of fila.entries()) {
          if (!Number.isFinite(valor)) {
            agregarProblema(
              contexto,
              CodigoError.NON_FINITE_VALUE,
              `la posición [${i}][${j}] contiene un valor no finito (NaN o infinito)`,
              { rowIndex: i, colIndex: j },
            );
            return;
          }
        }
      }
    });

  return z.object({
    matrices: z
      .record(z.string(), esquemaMatriz)
      .superRefine((matrices: Record<string, number[][]>, contexto: z.RefinementCtx) => {
        const nombres = Object.keys(matrices);
        if (nombres.length === 0) {
          agregarProblema(contexto, CodigoError.NO_MATRICES, "se requiere al menos una matriz en 'matrices'");
          return;
        }
        if (nombres.length > limites.maximoMatrices) {
          agregarProblema(
            contexto,
            CodigoError.TOO_MANY_MATRICES,
            `se recibieron ${nombres.length} matrices y el límite es ${limites.maximoMatrices}`,
            { received: nombres.length, maxMatrices: limites.maximoMatrices },
          );
        }
      }),
  });
}

/** Adjunta un issue con su código de dominio. */
function agregarProblema(
  contexto: z.RefinementCtx,
  codigoError: ValorCodigoError,
  mensaje: string,
  detalles?: Record<string, unknown>,
): void {
  const parametros: ParametrosProblema = detalles ? { codigoError, detalles } : { codigoError };
  contexto.addIssue({ code: 'custom', message: mensaje, params: parametros });
}

/**
 * Traduce un error de Zod al formato de error de la API.
 *
 * Se reporta solo el primer problema encontrado: enumerar todos los issues de
 * una matriz malformada produce ruido sin agregar información accionable, ya
 * que suelen ser el mismo defecto repetido en cada fila.
 */
export function convertirErrorZodAErrorApi(error: z.ZodError): ErrorApi {
  const problema = error.issues[0];
  if (!problema) {
    return ErrorApi.solicitudIncorrecta(CodigoError.INVALID_BODY, 'el cuerpo del request es inválido');
  }

  // Las reglas de dominio traen su propio código; el resto son fallos de forma
  // (falta `matrices`, un elemento no es número, etc.).
  const parametros = (problema as { params?: ParametrosProblema }).params;
  if (parametros?.codigoError) {
    return ErrorApi.solicitudIncorrecta(parametros.codigoError, problema.message, {
      ...parametros.detalles,
      path: problema.path.join('.'),
    });
  }

  const ubicacion = problema.path.length > 0 ? ` en '${problema.path.join('.')}'` : '';
  return ErrorApi.solicitudIncorrecta(CodigoError.INVALID_BODY, `${problema.message}${ubicacion}`, {
    path: problema.path.join('.'),
  });
}
