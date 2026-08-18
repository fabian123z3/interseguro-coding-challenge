/**
 * Contrato de errores de la API.
 *
 * Replica deliberadamente el formato de la API Go: ambos servicios responden
 * `{ error: { code, message, details, requestId } }`, de modo que el frontend
 * tiene un único camino de manejo de errores sin importar cuál falló.
 */

/**
 * Códigos de error estables. Forman parte del contrato público: el cliente
 * puede ramificar sobre ellos sin parsear el mensaje, que está escrito para
 * personas y puede cambiar.
 */
export const CodigoError = {
  INVALID_BODY: 'INVALID_BODY',
  NO_MATRICES: 'NO_MATRICES',
  EMPTY_MATRIX: 'EMPTY_MATRIX',
  RAGGED_ROWS: 'RAGGED_ROWS',
  NON_FINITE_VALUE: 'NON_FINITE_VALUE',
  MATRIX_TOO_LARGE: 'MATRIX_TOO_LARGE',
  TOO_MANY_MATRICES: 'TOO_MANY_MATRICES',
  PAYLOAD_TOO_LARGE: 'PAYLOAD_TOO_LARGE',
  UNAUTHORIZED: 'UNAUTHORIZED',
  TOKEN_EXPIRED: 'TOKEN_EXPIRED',
  NOT_FOUND: 'NOT_FOUND',
  INTERNAL_ERROR: 'INTERNAL_ERROR',
} as const;

export type ValorCodigoError = (typeof CodigoError)[keyof typeof CodigoError];

/** Cuerpo de un error. */
export interface DetalleError {
  code: ValorCodigoError;
  message: string;
  details?: Record<string, unknown>;
  requestId?: string;
}

/** Respuesta de error: el payload va bajo `error` para no confundirse con un éxito. */
export interface RespuestaError {
  error: DetalleError;
}

/** Error que ya sabe con qué status HTTP debe responderse. */
export class ErrorApi extends Error {
  override readonly name = 'ErrorApi';

  constructor(
    readonly estado: number,
    readonly codigo: ValorCodigoError,
    mensaje: string,
    readonly detalles?: Record<string, unknown>,
  ) {
    super(mensaje);
  }

  /** 400: el request no cumple el contrato de entrada. */
  static solicitudIncorrecta(codigo: ValorCodigoError, mensaje: string, detalles?: Record<string, unknown>) {
    return new ErrorApi(400, codigo, mensaje, detalles);
  }

  /** 401: falta el token, es inválido o expiró. */
  static noAutorizado(mensaje: string, codigo: ValorCodigoError = CodigoError.UNAUTHORIZED) {
    return new ErrorApi(401, codigo, mensaje);
  }

  /** 404: ruta inexistente. */
  static noEncontrado(mensaje: string) {
    return new ErrorApi(404, CodigoError.NOT_FOUND, mensaje);
  }
}
