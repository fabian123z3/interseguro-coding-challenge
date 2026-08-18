import { useId } from 'react';
import type { Matriz } from '../clienteApi';
import { Icono } from './Icono';

interface PropiedadesEditorMatriz {
  matriz: Matriz;
  alCambiar: (matriz: Matriz) => void;
  deshabilitado: boolean;
}

/** Tope de la interfaz. La API acepta hasta 256, pero más allá de esto la grilla deja de ser editable a mano. */
const DIMENSION_MAXIMA = 12;

/** Matrices de ejemplo, elegidas porque cada una muestra algo distinto del resultado. */
const EJEMPLOS: Array<{ etiqueta: string; ayuda: string; matriz: Matriz }> = [
  {
    etiqueta: 'Clásica 3×3',
    ayuda: 'El ejemplo canónico de la literatura: R queda con diagonal 14, 175 y 35.',
    matriz: [
      [12, -51, 4],
      [6, 167, -68],
      [-4, 24, -41],
    ],
  },
  {
    etiqueta: 'Rectangular 4×2',
    ayuda: 'Más filas que columnas: se aprecia la diferencia entre la forma completa y la reducida.',
    matriz: [
      [1, 2],
      [3, 4],
      [5, 6],
      [7, 8],
    ],
  },
  {
    etiqueta: 'Identidad 3×3',
    ayuda: 'Q y R salen ambas diagonales, así que anyDiagonal queda en verdadero.',
    matriz: [
      [1, 0, 0],
      [0, 1, 0],
      [0, 0, 1],
    ],
  },
  {
    etiqueta: 'Rango deficiente',
    ayuda: 'Columnas linealmente dependientes: donde Gram-Schmidt se degrada y Householder no.',
    matriz: [
      [1, 2, 3],
      [2, 4, 6],
      [3, 6, 9],
    ],
  },
];

/** Editor de la matriz de entrada: dimensiones ajustables y celdas editables. */
export function EditorMatriz({ matriz, alCambiar, deshabilitado }: PropiedadesEditorMatriz) {
  const idFilas = useId();
  const idColumnas = useId();

  const filas = matriz.length;
  const columnas = matriz[0]?.length ?? 0;

  /** Redimensiona conservando los valores que caben en las nuevas dimensiones. */
  const redimensionar = (nuevasFilas: number, nuevasColumnas: number) => {
    const filasAcotadas = acotar(nuevasFilas);
    const columnasAcotadas = acotar(nuevasColumnas);

    alCambiar(
      Array.from({ length: filasAcotadas }, (_, i) =>
        Array.from({ length: columnasAcotadas }, (_, j) => matriz[i]?.[j] ?? 0),
      ),
    );
  };

  const establecerCelda = (i: number, j: number, valorCrudo: string) => {
    // Se acepta la coma como separador decimal: es lo que produce el teclado en
    // configuración regional española.
    const valor = Number.parseFloat(valorCrudo.replace(',', '.'));
    const siguiente = matriz.map((fila) => [...fila]);
    const fila = siguiente[i];
    if (!fila || j >= fila.length) return;
    fila[j] = Number.isFinite(valor) ? valor : 0;
    alCambiar(siguiente);
  };

  return (
    <section className="editor" aria-labelledby="editor-title">
      <div className="titulo-seccion">
        <Icono nombre="matriz" />
        <h2 id="editor-title" className="panel__title">Matriz de entrada</h2>
      </div>

      <div className="editor__dims">
        <label className="field" htmlFor={idFilas}>
          <span className="field__label">Filas</span>
          <input
            id={idFilas}
            className="field__input"
            type="number"
            min={1}
            max={DIMENSION_MAXIMA}
            value={filas}
            disabled={deshabilitado}
            onChange={(evento) => redimensionar(Number(evento.target.value), columnas)}
          />
        </label>

        <span className="editor__times" aria-hidden="true">
          ×
        </span>

        <label className="field" htmlFor={idColumnas}>
          <span className="field__label">Columnas</span>
          <input
            id={idColumnas}
            className="field__input"
            type="number"
            min={1}
            max={DIMENSION_MAXIMA}
            value={columnas}
            disabled={deshabilitado}
            onChange={(evento) => redimensionar(filas, Number(evento.target.value))}
          />
        </label>
      </div>

      <div
        className="editor__grid"
        style={{ gridTemplateColumns: `repeat(${columnas}, minmax(3.5rem, 1fr))` }}
      >
        {matriz.map((fila, i) =>
          fila.map((valor, j) => (
            <input
              key={`${i}-${j}`}
              className="editor__cell"
              type="text"
              inputMode="decimal"
              value={valor}
              disabled={deshabilitado}
              aria-label={`Fila ${i + 1}, columna ${j + 1}`}
              onChange={(evento) => establecerCelda(i, j, evento.target.value)}
              onFocus={(evento) => evento.target.select()}
            />
          )),
        )}
      </div>

      <div className="editor__examples">
        <span className="editor__examples-label">Ejemplos</span>
        <div className="editor__examples-list">
          {EJEMPLOS.map((ejemplo) => (
            <button
              key={ejemplo.etiqueta}
              type="button"
              className="chip"
              title={ejemplo.ayuda}
              disabled={deshabilitado}
              onClick={() => alCambiar(ejemplo.matriz.map((fila) => [...fila]))}
            >
              {ejemplo.etiqueta}
            </button>
          ))}
        </div>
      </div>
    </section>
  );
}

function acotar(valor: number): number {
  if (!Number.isFinite(valor)) return 1;
  return Math.min(Math.max(Math.trunc(valor), 1), DIMENSION_MAXIMA);
}
