import type { Matriz } from '../clienteApi';
import { formatearValor, intensidad, maximoAbsoluto } from '../formato';

interface PropiedadesVistaMatriz {
  matriz: Matriz;
  /** Símbolo de la matriz: A, Q, R o A′. */
  simbolo: string;
  /** Acento cromático. Q y R llevan colores distintos para leerse como piezas separadas. */
  acento: 'input' | 'q' | 'r' | 'rotated';
  decimales: number;
  /**
   * Atenúa los ceros bajo la diagonal principal. Solo se activa en R, donde esos
   * ceros son la consecuencia visible del algoritmo y no un dato más.
   */
  revelarTriangulo?: boolean;
}

/**
 * Dibuja una matriz con la notación de corchetes.
 *
 * El fondo de cada celda se sombrea según la magnitud del valor respecto del
 * mayor de esa matriz. Eso convierte la salida numérica en una imagen de la
 * estructura: en R el triángulo superior queda cargado y el inferior en blanco,
 * y en Q el sombreado muestra de inmediato si es diagonal.
 */
export function VistaMatriz({
  matriz,
  simbolo,
  acento,
  decimales,
  revelarTriangulo = false,
}: PropiedadesVistaMatriz) {
  const escala = maximoAbsoluto(matriz);
  const filas = matriz.length;
  const columnas = matriz[0]?.length ?? 0;

  return (
    <figure className={`matrix matrix--${acento}`}>
      <figcaption className="matrix__caption">
        <span className="matrix__symbol">{simbolo}</span>
        <span className="matrix__dims">
          {filas}×{columnas}
        </span>
      </figcaption>

      <div className="matrix__frame">
        <span className="matrix__bracket matrix__bracket--left" aria-hidden="true" />

        <div
          className="matrix__grid"
          style={{ gridTemplateColumns: `repeat(${columnas}, minmax(0, 1fr))` }}
          role="table"
          aria-label={`Matriz ${simbolo} de ${filas} por ${columnas}`}
        >
          {matriz.map((fila, i) =>
            fila.map((valor, j) => {
              // Los ceros estructurales bajo la diagonal se atenúan en lugar de
              // ocultarse: la forma triangular se ve, pero el dato sigue ahí.
              const esCeroEstructural = revelarTriangulo && i > j && valor === 0;

              return (
                <span
                  key={`${i}-${j}`}
                  role="cell"
                  className={`matrix__cell${esCeroEstructural ? ' matrix__cell--structural' : ''}`}
                  style={{
                    // El retraso escalonado por fila evoca el avance del
                    // algoritmo, que procesa una columna por vez.
                    animationDelay: `${Math.min(i * 45, 400)}ms`,
                    ...(esCeroEstructural
                      ? {}
                      : { '--cell-intensity': intensidad(valor, escala) } as React.CSSProperties),
                  }}
                  // El valor completo queda accesible sin ocupar la grilla.
                  title={String(valor)}
                >
                  {formatearValor(valor, decimales)}
                </span>
              );
            }),
          )}
        </div>

        <span className="matrix__bracket matrix__bracket--right" aria-hidden="true" />
      </div>
    </figure>
  );
}
