import { describe, expect, it } from 'vitest';
import { extraerTokenBearer } from '../../src/middleware/autenticacion.js';
import { ErrorApi } from '../../src/errores.js';

describe('extraerTokenBearer', () => {
  it.each([
    ['formato estándar', 'Bearer abc.def.ghi'],
    ['esquema en minúsculas', 'bearer abc.def.ghi'],
    ['esquema en mayúsculas', 'BEARER abc.def.ghi'],
    ['con espacios sobrantes', 'Bearer   abc.def.ghi  '],
  ])('extrae el token con %s', (_nombre, encabezado) => {
    expect(extraerTokenBearer(encabezado)).toBe('abc.def.ghi');
  });

  it.each([
    ['encabezado ausente', undefined],
    ['encabezado vacío', ''],
    ['sin esquema', 'abc.def.ghi'],
    ['otro esquema', 'Basic dXNlcjpwYXNz'],
    ['token vacío', 'Bearer '],
    ['solo espacios', 'Bearer    '],
  ])('rechaza %s', (_nombre, encabezado) => {
    expect(() => extraerTokenBearer(encabezado)).toThrow(ErrorApi);
  });

  it('responde con estado 401', () => {
    try {
      extraerTokenBearer(undefined);
      expect.unreachable('debía lanzar');
    } catch (error) {
      expect(error).toBeInstanceOf(ErrorApi);
      expect((error as ErrorApi).estado).toBe(401);
    }
  });
});
