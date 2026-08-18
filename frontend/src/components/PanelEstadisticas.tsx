import type { Estadisticas } from '../clienteApi';
import { formatearEstadistica } from '../formato';
import { Icono } from './Icono';

interface PropiedadesPanelEstadisticas {
  estadisticas: Estadisticas;
}

/** Etiquetas de las cinco medidas que pide el enunciado. */
const MEDIDAS = [
  { key: 'max', label: 'Máximo' },
  { key: 'min', label: 'Mínimo' },
  { key: 'average', label: 'Promedio' },
  { key: 'sum', label: 'Suma total' },
] as const;

const ETIQUETAS_MATRIZ: Record<string, string> = {
  q: 'Q',
  r: 'R',
  rotated: 'ROTADA',
};

/**
 * Resultado de la API Node.
 *
 * Se muestra primero el agregado sobre las matrices procesadas y debajo el
 * desglose por matriz, que permite ubicar de cuál viene cada extremo.
 */
export function PanelEstadisticas({ estadisticas }: PropiedadesPanelEstadisticas) {
  const entradas = Object.entries(estadisticas.perMatrix);

  return (
    <section className="stats" aria-labelledby="stats-title">
      <div className="stats__header">
        <div className="titulo-seccion">
          <Icono nombre="estadisticas" />
          <h2 id="stats-title" className="panel__title">Estadísticas</h2>
        </div>
      </div>

      <div className="stats__overall">
        {MEDIDAS.map((medida) => (
          <div key={medida.key} className="stat">
            <span className="stat__label">{medida.label}</span>
            <span className="stat__value">{formatearEstadistica(estadisticas.overall[medida.key])}</span>
          </div>
        ))}
        <div className="stat">
          <span className="stat__label">Valores</span>
          <span className="stat__value">{estadisticas.overall.count}</span>
        </div>
      </div>

      <div className={`verdict${estadisticas.anyDiagonal ? ' verdict--yes' : ''}`}>
        <span className="verdict__question">¿Alguna matriz es diagonal?</span>
        <span className="verdict__answer">{estadisticas.anyDiagonal ? 'Sí' : 'No'}</span>
      </div>

      <div className="breakdown-scroll">
        <table className="breakdown">
          <caption className="breakdown__caption">Desglose por matriz</caption>
          <thead>
            <tr>
              <th scope="col">Matriz</th>
              <th scope="col">Máximo</th>
              <th scope="col">Mínimo</th>
              <th scope="col">Promedio</th>
              <th scope="col">Suma</th>
              <th scope="col">Diagonal</th>
            </tr>
          </thead>
          <tbody>
            {entradas.map(([nombre, estadistica]) => (
              <tr key={nombre}>
                <th scope="row" className={`breakdown__name breakdown__name--${nombre}`}>
                  {ETIQUETAS_MATRIZ[nombre] ?? nombre.toUpperCase()}
                </th>
                <td>{formatearEstadistica(estadistica.max)}</td>
                <td>{formatearEstadistica(estadistica.min)}</td>
                <td>{formatearEstadistica(estadistica.average)}</td>
                <td>{formatearEstadistica(estadistica.sum)}</td>
                <td>
                  <span
                    className={`flag${estadistica.isDiagonal ? ' flag--on' : ''}`}
                    // La tolerancia se deriva de la magnitud de cada matriz, así
                    // que puede diferir entre matrices: exponerla hace auditable el juicio.
                    title={`Evaluado con una tolerancia de ${estadistica.tolerance.toExponential(2)}`}
                  >
                    {estadistica.isDiagonal ? 'Sí' : 'No'}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
