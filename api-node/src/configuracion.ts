/**
 * Configuración del servicio, resuelta desde variables de entorno.
 *
 * Se valida al arrancar y el proceso no levanta si algo falta: es preferible
 * fallar de inmediato, cuando el problema es evidente, que descubrirlo en el
 * primer request en producción.
 */

export interface Configuracion {
  /** Puerto TCP en que escucha el servidor. */
  puerto: number;
  /** Secreto HS256 compartido con la API Go, que es quien emite los tokens. */
  secretoJwt: string;
  /** Claims `iss` y `aud` que debe traer todo token aceptado. */
  emisorJwt: string;
  audienciaJwt: string;
  /** Nivel de log de pino. */
  nivelRegistro: string;
  /** Límite de filas y columnas por matriz. */
  dimensionMaximaMatriz: number;
  /** Límite de matrices por request. */
  maximoMatrices: number;
}

/** Error de configuración: distingue un arranque mal configurado de un bug. */
export class ErrorConfiguracion extends Error {
  override readonly name = 'ErrorConfiguracion';
}

/**
 * Construye la configuración a partir del entorno.
 *
 * Recibe `env` como parámetro en lugar de leer `process.env` directamente para
 * poder probar cada combinación sin ensuciar el entorno del proceso de test.
 */
export function cargarConfiguracion(env: NodeJS.ProcessEnv = process.env): Configuracion {
  const configuracion: Configuracion = {
    puerto: leerEntero(env.NODE_API_PORT, 3000),
    secretoJwt: env.JWT_SECRET ?? '',
    emisorJwt: env.JWT_ISSUER || 'interseguro-qr-api',
    audienciaJwt: env.JWT_AUDIENCE || 'interseguro-clients',
    nivelRegistro: env.LOG_LEVEL || 'info',
    dimensionMaximaMatriz: leerEntero(env.MAX_MATRIX_DIMENSION, 256),
    maximoMatrices: leerEntero(env.MAX_MATRICES, 16),
  };

  // Sin secreto no hay forma de verificar ninguna firma. Generar uno al vuelo
  // sería peor: los tokens emitidos por la API Go dejarían de validar y el
  // fallo aparecería recién en el primer request, no al arrancar.
  if (!configuracion.secretoJwt) {
    throw new ErrorConfiguracion(
      'JWT_SECRET es obligatorio y debe coincidir con el de la API Go (ver .env.example)',
    );
  }
  if (configuracion.puerto < 1 || configuracion.puerto > 65535) {
    throw new ErrorConfiguracion(`NODE_API_PORT fuera de rango: ${configuracion.puerto}`);
  }
  if (configuracion.dimensionMaximaMatriz < 1) {
    throw new ErrorConfiguracion(`MAX_MATRIX_DIMENSION debe ser positivo: ${configuracion.dimensionMaximaMatriz}`);
  }
  if (configuracion.maximoMatrices < 1) {
    throw new ErrorConfiguracion(`MAX_MATRICES debe ser positivo: ${configuracion.maximoMatrices}`);
  }

  return configuracion;
}

/**
 * Devuelve el valor por defecto si la variable está ausente o no es un entero.
 * Un valor mal escrito no debe impedir el arranque por sí solo: las
 * validaciones de rango de arriba se encargan de los casos imposibles.
 */
function leerEntero(valor: string | undefined, valorAlternativo: number): number {
  if (valor === undefined || valor.trim() === '') return valorAlternativo;
  const numero = Number.parseInt(valor, 10);
  return Number.isNaN(numero) ? valorAlternativo : numero;
}
