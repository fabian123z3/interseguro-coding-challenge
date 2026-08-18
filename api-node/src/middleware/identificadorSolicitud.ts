import { randomUUID } from 'node:crypto';
import type { NextFunction, Request, Response } from 'express';

/** Encabezado con que viaja el identificador de correlación entre servicios. */
export const ENCABEZADO_ID_SOLICITUD = 'x-request-id';

declare global {
  // eslint-disable-next-line @typescript-eslint/no-namespace
  namespace Express {
    interface Request {
      /** Identificador de correlación del request. */
      idSolicitud: string;
    }
  }
}

/**
 * Adopta el identificador que envía la API Go, o genera uno si el request llega
 * directo.
 *
 * Conservar el mismo identificador a lo largo de la cadena es lo que permite
 * seguir una traza completa —frontend, API Go y API Node— con una sola búsqueda
 * en los logs.
 *
 * El valor recibido se sanea antes de usarse: se acepta solo un subconjunto
 * seguro de caracteres y se acota el largo, porque termina copiado en un
 * encabezado de respuesta y en las líneas de log.
 */
export function identificarSolicitud() {
  return (req: Request, res: Response, next: NextFunction): void => {
    const identificadorEntrante = req.header(ENCABEZADO_ID_SOLICITUD);
    req.idSolicitud = esIdSolicitudSeguro(identificadorEntrante) ? identificadorEntrante : randomUUID();
    res.setHeader(ENCABEZADO_ID_SOLICITUD, req.idSolicitud);
    next();
  };
}

function esIdSolicitudSeguro(valor: string | undefined): valor is string {
  return valor !== undefined && valor.length > 0 && valor.length <= 128 && /^[\w.:-]+$/.test(valor);
}
