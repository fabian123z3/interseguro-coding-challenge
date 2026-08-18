/**
 * Manejo centralizado de errores.
 *
 * Todo error termina acá y sale con la misma forma. Centralizarlo garantiza que
 * ninguna ruta invente su propio formato y que los detalles internos no se
 * filtren al cliente por descuido.
 */

import type { NextFunction, Request, Response } from 'express';
import type { Logger } from 'pino';
import { ErrorApi, CodigoError, type RespuestaError } from '../errores.js';

/** Responde 404 a cualquier ruta no registrada. */
export function manejarNoEncontrado() {
  return (req: Request, _res: Response, next: NextFunction): void => {
    next(ErrorApi.noEncontrado(`la ruta ${req.method} ${req.path} no existe`));
  };
}

/**
 * Manejador final de errores.
 *
 * Express identifica un manejador de errores por su aridad de cuatro
 * parámetros, de modo que `next` debe declararse aunque no se use.
 */
export function manejarErrores(registrador: Logger) {
  return (errorCrudo: unknown, req: Request, res: Response, _next: NextFunction): void => {
    const idSolicitud = req.idSolicitud;
    const error = normalizarErrorAnalizadorCuerpo(errorCrudo);

    if (error instanceof ErrorApi) {
      // Los errores esperados (validación, autenticación) no son incidentes:
      // se registran en warn para no contaminar las alertas de error.
      registrador.warn(
        { requestId: idSolicitud, code: error.codigo, status: error.estado, path: req.path },
        error.message,
      );

      const cuerpo: RespuestaError = {
        error: {
          code: error.codigo,
          message: error.message,
          ...(error.detalles ? { details: error.detalles } : {}),
          requestId: idSolicitud,
        },
      };
      res.status(error.estado).json(cuerpo);
      return;
    }

    // Un error no contemplado sí es un incidente: se registra completo, con
    // stack, pero al cliente solo le llega un mensaje genérico. Devolver el
    // detalle interno filtraría rutas de archivos y estructura del código.
    registrador.error({ requestId: idSolicitud, path: req.path, err: error }, 'error no controlado');

    const cuerpo: RespuestaError = {
      error: {
        code: CodigoError.INTERNAL_ERROR,
        message: 'error interno del servidor',
        requestId: idSolicitud,
      },
    };
    res.status(500).json(cuerpo);
  };
}

/** Forma de los errores que emite body-parser, el parser que usa express.json(). */
interface ErrorAnalizadorCuerpo extends Error {
  type?: string;
  status?: number;
}

/**
 * Convierte los errores de `express.json()` en ErrorApi.
 *
 * Sin esta traducción, un cuerpo con JSON malformado terminaría en la rama del
 * error no controlado y saldría como 500: el cliente no sabría que el problema
 * es suyo y corregible, y en el servidor cada request mal formado se registraría
 * como incidente, disparando alertas sin motivo.
 */
function normalizarErrorAnalizadorCuerpo(error: unknown): unknown {
  if (!(error instanceof Error)) return error;

  const { type: tipo, status: estadoAnalizador } = error as ErrorAnalizadorCuerpo;

  if (tipo === 'entity.parse.failed') {
    return ErrorApi.solicitudIncorrecta(CodigoError.INVALID_BODY, 'el cuerpo del request no es JSON válido');
  }
  if (tipo === 'entity.too.large') {
    return new ErrorApi(
      413,
      CodigoError.PAYLOAD_TOO_LARGE,
      'el cuerpo del request supera el tamaño máximo permitido',
    );
  }
  // Resto de fallos de parseo (charset o codificación no soportados): siguen
  // siendo problemas del request, no del servidor.
  if (typeof tipo === 'string' && tipo.startsWith('entity.') && estadoAnalizador && estadoAnalizador < 500) {
    return new ErrorApi(estadoAnalizador, CodigoError.INVALID_BODY, error.message);
  }

  return error;
}

/**
 * Registra una línea por request al terminar la respuesta.
 *
 * Usa las mismas claves que el logger de la API Go (`requestId`, `method`,
 * `path`, `status`, `durationMs`), de modo que una sola consulta sirve para
 * seguir una traza a través de ambos servicios.
 */
export function registrarSolicitudes(registrador: Logger) {
  return (req: Request, res: Response, next: NextFunction): void => {
    const inicio = process.hrtime.bigint();

    // 'finish' se dispara cuando la respuesta se envió por completo, que es el
    // único momento en que el status y la duración son definitivos.
    res.on('finish', () => {
      const duracionMs = Number(process.hrtime.bigint() - inicio) / 1e6;
      registrador.info(
        {
          requestId: req.idSolicitud,
          method: req.method,
          path: req.path,
          status: res.statusCode,
          durationMs: Number(duracionMs.toFixed(3)),
        },
        'request finalizado',
      );
    });

    next();
  };
}
