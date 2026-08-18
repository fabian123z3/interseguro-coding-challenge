/**
 * Verificación del JWT emitido por la API Go.
 *
 * Este servicio no emite tokens: valida los que la API Go firmó con el mismo
 * secreto compartido y propagó en el encabezado Authorization. De ese modo la
 * identidad del usuario final sobrevive el salto entre servicios, en lugar de
 * reemplazarse por una credencial de máquina que perdería la trazabilidad.
 */

import type { NextFunction, Request, Response } from 'express';
import jwt from 'jsonwebtoken';
import { ErrorApi, CodigoError } from '../errores.js';
import type { Configuracion } from '../configuracion.js';

declare global {
  // eslint-disable-next-line @typescript-eslint/no-namespace
  namespace Express {
    interface Request {
      /** Sujeto autenticado, disponible una vez que pasó el middleware. */
      sujetoAutenticado?: string;
    }
  }
}

/** Middleware que exige un token Bearer válido. */
export function exigirJwt(configuracion: Configuracion) {
  return (req: Request, _res: Response, next: NextFunction): void => {
    let token: string;
    try {
      token = extraerTokenBearer(req.header('authorization'));
    } catch (error) {
      next(error);
      return;
    }

    try {
      const contenido = jwt.verify(token, configuracion.secretoJwt, {
        // Restringir el algoritmo es imprescindible: sin esta lista, un token
        // con `alg: none` o firmado con otro esquema podría llegar a aceptarse.
        algorithms: ['HS256'],
        issuer: configuracion.emisorJwt,
        audience: configuracion.audienciaJwt,
      });

      const sujeto = typeof contenido === 'string' ? undefined : contenido.sub;
      if (!sujeto) {
        next(ErrorApi.noAutorizado('el token no identifica a ningún sujeto'));
        return;
      }

      req.sujetoAutenticado = sujeto;
      next();
    } catch (error) {
      if (error instanceof jwt.TokenExpiredError) {
        next(
          ErrorApi.noAutorizado(
            'el token expiró: solicitar uno nuevo en POST /api/v1/auth/login de la API Go',
            CodigoError.TOKEN_EXPIRED,
          ),
        );
        return;
      }
      // El motivo exacto no se expone: detallar por qué una firma no valida le
      // daría a un atacante información útil para afinar el siguiente intento.
      next(ErrorApi.noAutorizado('el token es inválido'));
    }
  };
}

/**
 * Extrae el token del encabezado `Authorization: Bearer <token>`.
 * El esquema se compara sin distinguir mayúsculas, como exige RFC 7235.
 */
export function extraerTokenBearer(encabezado: string | undefined): string {
  if (!encabezado) {
    throw ErrorApi.noAutorizado('falta el encabezado Authorization');
  }

  const indiceSeparador = encabezado.indexOf(' ');
  if (indiceSeparador === -1) {
    throw ErrorApi.noAutorizado("el encabezado Authorization debe tener el formato 'Bearer <token>'");
  }

  const esquema = encabezado.slice(0, indiceSeparador);
  const token = encabezado.slice(indiceSeparador + 1).trim();

  if (esquema.toLowerCase() !== 'bearer') {
    throw ErrorApi.noAutorizado("el encabezado Authorization debe tener el formato 'Bearer <token>'");
  }
  if (!token) {
    throw ErrorApi.noAutorizado('el token está vacío');
  }
  return token;
}
