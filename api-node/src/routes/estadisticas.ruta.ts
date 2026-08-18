import { Router, type Request, type Response, type NextFunction } from 'express';
import type { Configuracion } from '../configuracion.js';
import { exigirJwt } from '../middleware/autenticacion.js';
import { crearEsquemaEstadisticas, convertirErrorZodAErrorApi } from '../schemas/estadisticas.esquema.js';
import { calcularEstadisticas, type ResultadoEstadisticas } from '../services/estadisticas.servicio.js';

/**
 * Monta POST /api/v1/statistics.
 *
 * Es el endpoint que consume la API Go tras factorizar: recibe las matrices Q y
 * R y devuelve las estadísticas pedidas por el enunciado.
 */
export function enrutadorEstadisticas(configuracion: Configuracion): Router {
  const enrutador = Router();
  const esquema = crearEsquemaEstadisticas({
    dimensionMaximaMatriz: configuracion.dimensionMaximaMatriz,
    maximoMatrices: configuracion.maximoMatrices,
  });

  enrutador.post(
    '/statistics',
    exigirJwt(configuracion),
    (req: Request, res: Response<ResultadoEstadisticas>, next: NextFunction) => {
      const resultadoValidacion = esquema.safeParse(req.body);
      if (!resultadoValidacion.success) {
        next(convertirErrorZodAErrorApi(resultadoValidacion.error));
        return;
      }

      res.json(calcularEstadisticas(resultadoValidacion.data.matrices));
    },
  );

  return enrutador;
}
