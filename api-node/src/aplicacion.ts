import express, { type Express } from 'express';
import proteccionCabeceras from 'helmet';
import { pino, type Logger } from 'pino';
import type { Configuracion } from './configuracion.js';
import { manejarErrores, manejarNoEncontrado, registrarSolicitudes } from './middleware/errores.js';
import { identificarSolicitud } from './middleware/identificadorSolicitud.js';
import { enrutadorSalud } from './routes/salud.ruta.js';
import { enrutadorEstadisticas } from './routes/estadisticas.ruta.js';

/**
 * Construye la aplicación Express sin abrir ningún puerto.
 *
 * Separar la construcción del arranque permite que los tests la ejerciten con
 * Supertest en memoria, sin sockets ni puertos ocupados.
 */
export function crearAplicacion(configuracion: Configuracion, registrador: Logger = crearRegistrador(configuracion)): Express {
  const aplicacion = express();

  // Se confía en el primer proxy de la cadena para resolver la IP de origen:
  // en Docker Compose y en las plataformas cloud el tráfico siempre llega a
  // través de uno.
  aplicacion.set('trust proxy', 1);
  // La cabecera X-Powered-By solo informa a un atacante qué stack se ejecuta.
  aplicacion.disable('x-powered-by');

  // Añade una política defensiva común para MIME sniffing, iframes, origen y
  // referencia. Aunque esta API sea interna, la protección no debe depender
  // de que todas las capas de red estén siempre configuradas correctamente.
  aplicacion.use(proteccionCabeceras());

  // El orden importa: idSolicitud primero, para que el logger y el manejador de
  // errores ya dispongan del identificador de correlación.
  aplicacion.use(identificarSolicitud());
  aplicacion.use(registrarSolicitudes(registrador));

  // El límite del cuerpo acota el gasto de memoria: una matriz de 256×256 en
  // JSON ocupa unos pocos MB, así que 16 MB deja margen de sobra y a la vez
  // impide que un cuerpo enorme se materialice antes de validarse.
  aplicacion.use(express.json({ limit: '16mb' }));

  aplicacion.use(enrutadorSalud());
  aplicacion.use('/api/v1', enrutadorEstadisticas(configuracion));

  // Ambos van al final: el 404 solo aplica si ninguna ruta coincidió, y el
  // manejador de errores debe ser el último middleware de la cadena.
  aplicacion.use(manejarNoEncontrado());
  aplicacion.use(manejarErrores(registrador));

  return aplicacion;
}

/**
 * Crea el logger estructurado.
 *
 * Emite JSON en una línea por evento, que es el formato que los agregadores de
 * las plataformas cloud (Cloud Logging, CloudWatch) indexan sin configuración
 * adicional.
 */
export function crearRegistrador(configuracion: Configuracion): Logger {
  return pino({
    level: configuracion.nivelRegistro,
    // El campo por defecto es `time` en milisegundos; ISO-8601 es legible tanto
    // para una persona como para el agregador.
    timestamp: () => `,"time":"${new Date().toISOString()}"`,
    base: { service: 'statistics-api-node' },
  });
}
