/** Punto de entrada: levanta la API de estadísticas. */

import { crearAplicacion, crearRegistrador } from './aplicacion.js';
import { ErrorConfiguracion, cargarConfiguracion } from './configuracion.js';

function main(): void {
  let configuracion;
  try {
    configuracion = cargarConfiguracion();
  } catch (error) {
    if (error instanceof ErrorConfiguracion) {
      // Todavía no hay logger configurado, así que el error va a stderr crudo.
      process.stderr.write(`configuración inválida: ${error.message}\n`);
      process.exit(1);
    }
    throw error;
  }

  const registrador = crearRegistrador(configuracion);
  const aplicacion = crearAplicacion(configuracion, registrador);

  // 0.0.0.0 en lugar de localhost: dentro de un contenedor, escuchar solo en la
  // interfaz de loopback lo haría inalcanzable desde fuera.
  const servidor = aplicacion.listen(configuracion.puerto, '0.0.0.0', () => {
    registrador.info({ puerto: configuracion.puerto }, 'servidor iniciado');
  });

  // Los valores predeterminados de Node pueden cambiar entre versiones. Al
  // declararlos se evita que clientes lentos retengan conexiones sin límite y
  // el comportamiento queda documentado junto al servidor.
  servidor.requestTimeout = 30_000;
  servidor.headersTimeout = 10_000;
  servidor.keepAliveTimeout = 5_000;

  // Apagado ordenado: se deja de aceptar conexiones y se espera a que terminen
  // los requests en curso. Sin esto, un despliegue cortaría respuestas a medio
  // camino y el cliente vería errores de red sin causa aparente.
  let apagadoIniciado = false;
  const apagar = (senal: string): void => {
    if (apagadoIniciado) return;
    apagadoIniciado = true;
    registrador.info({ signal: senal }, 'apagado solicitado, drenando conexiones');

    // Red de seguridad: si alguna conexión no cierra, no se puede quedar
    // colgado indefinidamente bloqueando el despliegue.
    const salidaForzada = setTimeout(() => {
      registrador.error('el apagado ordenado no completó a tiempo, forzando la salida');
      process.exit(1);
    }, 10_000);
    salidaForzada.unref();

    servidor.close((error) => {
      if (error) {
        registrador.error({ err: error }, 'error al cerrar el servidor');
        process.exit(1);
      }
      registrador.info('servidor detenido');
      process.exit(0);
    });
  };

  // SIGTERM es la señal que envía Docker al detener un contenedor;
  // SIGINT llega con Ctrl+C en desarrollo.
  process.once('SIGTERM', () => apagar('SIGTERM'));
  process.once('SIGINT', () => apagar('SIGINT'));
}

main();
