import { Router, type Request, type Response } from 'express';

/** Versión del build, inyectable desde el Dockerfile. */
const VERSION = process.env.SERVICE_VERSION ?? 'dev';

export interface RespuestaSalud {
  status: 'ok';
  service: string;
  version: string;
}

/**
 * Monta GET /health.
 *
 * Queda fuera de la autenticación a propósito: lo consultan el healthcheck de
 * Docker, el balanceador y el chequeo de readiness de la API Go, ninguno de los
 * cuales tiene credenciales. No expone más que el nombre y la versión del
 * servicio.
 */
export function enrutadorSalud(): Router {
  const enrutador = Router();

  enrutador.get('/health', (_req: Request, res: Response<RespuestaSalud>) => {
    res.json({ status: 'ok', service: 'statistics-api-node', version: VERSION });
  });

  return enrutador;
}
