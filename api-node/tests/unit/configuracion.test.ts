import { describe, expect, it } from 'vitest';
import { ErrorConfiguracion, cargarConfiguracion } from '../../src/configuracion.js';

/** Entorno mínimo con el que cargarConfiguracion debe tener éxito. */
const entornoValido = { JWT_SECRET: 'secreto-de-prueba' } as NodeJS.ProcessEnv;

describe('cargarConfiguracion', () => {
  it('aplica los valores por defecto documentados', () => {
    const configuracion = cargarConfiguracion(entornoValido);

    expect(configuracion).toMatchObject({
      puerto: 3000,
      emisorJwt: 'interseguro-qr-api',
      audienciaJwt: 'interseguro-clients',
      nivelRegistro: 'info',
      dimensionMaximaMatriz: 256,
      maximoMatrices: 16,
    });
  });

  it('toma los valores del entorno cuando están presentes', () => {
    const configuracion = cargarConfiguracion({
      ...entornoValido,
      NODE_API_PORT: '4000',
      JWT_ISSUER: 'otro-emisor',
      JWT_AUDIENCE: 'otra-audiencia',
      LOG_LEVEL: 'debug',
      MAX_MATRIX_DIMENSION: '64',
      MAX_MATRICES: '4',
    });

    expect(configuracion).toMatchObject({
      puerto: 4000,
      emisorJwt: 'otro-emisor',
      audienciaJwt: 'otra-audiencia',
      nivelRegistro: 'debug',
      dimensionMaximaMatriz: 64,
      maximoMatrices: 4,
    });
  });

  it('no arranca sin JWT_SECRET', () => {
    expect(() => cargarConfiguracion({})).toThrow(ErrorConfiguracion);
  });

  it.each([
    ['puerto fuera de rango', { NODE_API_PORT: '70000' }],
    ['dimensión máxima en cero', { MAX_MATRIX_DIMENSION: '0' }],
    ['cantidad máxima de matrices en cero', { MAX_MATRICES: '0' }],
  ])('rechaza una configuración inválida: %s', (_nombre, valores) => {
    expect(() => cargarConfiguracion({ ...entornoValido, ...valores })).toThrow(ErrorConfiguracion);
  });

  it('cae al valor por defecto ante un número mal escrito', () => {
    // Un error de tipeo es recuperable: no debe impedir el arranque por sí solo.
    const configuracion = cargarConfiguracion({ ...entornoValido, NODE_API_PORT: 'tres-mil' });

    expect(configuracion.puerto).toBe(3000);
  });
});
