import { describe, expect, it } from 'vitest';
import {
  calcularEstadisticasMatriz,
  calcularEstadisticas,
  esDiagonal,
  TOLERANCIA_MINIMA,
  FACTOR_TOLERANCIA,
} from '../../src/services/estadisticas.servicio.js';

describe('calcularEstadisticasMatriz', () => {
  it('calcula los cuatro agregados que pide el enunciado', () => {
    const stats = calcularEstadisticasMatriz([
      [1, 2, 3],
      [4, 5, 6],
    ]);

    expect(stats.max).toBe(6);
    expect(stats.min).toBe(1);
    expect(stats.sum).toBe(21);
    expect(stats.average).toBe(3.5);
    expect(stats.count).toBe(6);
    expect(stats.rows).toBe(2);
    expect(stats.cols).toBe(3);
    expect(stats.isSquare).toBe(false);
  });

  it('maneja valores negativos en los extremos', () => {
    const stats = calcularEstadisticasMatriz([
      [-10, -2],
      [-7, -1],
    ]);

    expect(stats.max).toBe(-1);
    expect(stats.min).toBe(-10);
    expect(stats.sum).toBe(-20);
    expect(stats.average).toBe(-5);
  });

  it('trata una matriz de un solo elemento', () => {
    const stats = calcularEstadisticasMatriz([[42]]);

    expect(stats).toMatchObject({
      max: 42,
      min: 42,
      sum: 42,
      average: 42,
      count: 1,
      isSquare: true,
      isDiagonal: true,
    });
  });

  /**
   * Sin compensación, sumar 1e16 + 1 pierde el 1 por completo (el resultado no
   * es representable) y el total daría 0 en lugar de 1.
   */
  it('conserva la precisión al sumar magnitudes muy distintas', () => {
    const stats = calcularEstadisticasMatriz([[1e16, 1, -1e16]]);

    expect(stats.sum).toBe(1);

    // Comprobación explícita de que la suma ingenua sí falla, para que el test
    // documente por qué existe el acumulador compensado.
    const naive = [1e16, 1, -1e16].reduce((acc, value) => acc + value, 0);
    expect(naive).toBe(0);
  });
});

describe('esDiagonal', () => {
  it('reconoce una matriz diagonal exacta', () => {
    expect(
      esDiagonal(
        [
          [5, 0, 0],
          [0, 3, 0],
          [0, 0, 1],
        ],
        TOLERANCIA_MINIMA,
      ),
    ).toBe(true);
  });

  it('rechaza una matriz con un elemento fuera de la diagonal', () => {
    expect(
      esDiagonal(
        [
          [5, 0],
          [7, 3],
        ],
        TOLERANCIA_MINIMA,
      ),
    ).toBe(false);
  });

  it('acepta residuos de redondeo dentro de la tolerancia', () => {
    expect(
      esDiagonal(
        [
          [5, 1e-15],
          [3e-16, 3],
        ],
        1e-12,
      ),
    ).toBe(true);
  });

  it('rechaza valores fuera de la diagonal que superan la tolerancia', () => {
    expect(
      esDiagonal(
        [
          [5, 1e-6],
          [0, 3],
        ],
        1e-12,
      ),
    ).toBe(false);
  });

  it('considera diagonal la matriz nula', () => {
    // No tiene ningún elemento no nulo fuera de la diagonal, así que cumple la
    // definición.
    expect(
      esDiagonal(
        [
          [0, 0],
          [0, 0],
        ],
        TOLERANCIA_MINIMA,
      ),
    ).toBe(true);
  });

  it('aplica la definición generalizada a matrices rectangulares', () => {
    // Una matriz 3×2 sin elementos no nulos fuera de la diagonal principal.
    expect(
      esDiagonal(
        [
          [5, 0],
          [0, 3],
          [0, 0],
        ],
        TOLERANCIA_MINIMA,
      ),
    ).toBe(true);
  });
});

describe('tolerancia relativa', () => {
  it('escala la tolerancia con la magnitud de cada matriz', () => {
    const stats = calcularEstadisticasMatriz([
      [1e9, 0],
      [0, 2e9],
    ]);

    expect(stats.tolerance).toBeCloseTo(FACTOR_TOLERANCIA * 2e9, 10);
    expect(stats.isDiagonal).toBe(true);
  });

  it('aplica el piso absoluto cuando la matriz es de magnitud despreciable', () => {
    const stats = calcularEstadisticasMatriz([
      [0, 0],
      [0, 0],
    ]);

    expect(stats.tolerance).toBe(TOLERANCIA_MINIMA);
  });

  /**
   * Este es el motivo de derivar la tolerancia por matriz y no del conjunto:
   * con una tolerancia global tomada de la matriz de mayor magnitud (1e9 →
   * tolerancia 1), el 1e-3 fuera de la diagonal de la matriz pequeña quedaría
   * enmascarado y esa matriz se reportaría como diagonal sin serlo.
   */
  it('no enmascara valores significativos de una matriz de magnitud pequeña', () => {
    const result = calcularEstadisticas({
      pequena: [
        [1, 1e-3],
        [0, 1],
      ],
      grande: [
        [1e9, 0],
        [0, 1e9],
      ],
    });

    expect(result.perMatrix.pequena).toMatchObject({ isDiagonal: false });
    expect(result.perMatrix.grande).toMatchObject({ isDiagonal: true });
  });
});

describe('calcularEstadisticas', () => {
  const q = [
    [1, 0],
    [0, 1],
  ];
  const r = [
    [10, 20],
    [0, 40],
  ];

  it('agrega los valores de todas las matrices', () => {
    const result = calcularEstadisticas({ q, r });

    expect(result.overall.max).toBe(40);
    expect(result.overall.min).toBe(0);
    expect(result.overall.sum).toBe(72); // (1+0+0+1) + (10+20+0+40)
    expect(result.overall.count).toBe(8);
    expect(result.overall.average).toBe(9);
  });

  it('entrega el desglose por matriz', () => {
    const result = calcularEstadisticas({ q, r });

    expect(Object.keys(result.perMatrix)).toEqual(['q', 'r']);
    expect(result.perMatrix.q).toMatchObject({ sum: 2 });
    expect(result.perMatrix.r).toMatchObject({ sum: 70 });
  });

  it('marca anyDiagonal cuando al menos una matriz lo es', () => {
    const result = calcularEstadisticas({ q, r });

    expect(result.perMatrix.q).toMatchObject({ isDiagonal: true });
    expect(result.perMatrix.r).toMatchObject({ isDiagonal: false });
    expect(result.anyDiagonal).toBe(true);
  });

  it('deja anyDiagonal en false cuando ninguna lo es', () => {
    const result = calcularEstadisticas({
      a: [
        [1, 2],
        [3, 4],
      ],
      b: [
        [5, 6],
        [7, 8],
      ],
    });

    expect(result.anyDiagonal).toBe(false);
  });

  it('funciona con una sola matriz', () => {
    const result = calcularEstadisticas({ unica: r });

    expect(result.overall.sum).toBe(70);
    expect(result.overall.count).toBe(4);
  });

  it('acepta matrices de dimensiones distintas entre sí', () => {
    const result = calcularEstadisticas({
      alta: [[1], [2], [3]],
      ancha: [[4, 5, 6]],
    });

    expect(result.overall.count).toBe(6);
    expect(result.overall.sum).toBe(21);
    expect(result.overall.max).toBe(6);
    expect(result.overall.min).toBe(1);
  });

  it('informa el factor de tolerancia usado', () => {
    expect(calcularEstadisticas({ q }).toleranceFactor).toBe(FACTOR_TOLERANCIA);
  });
});
