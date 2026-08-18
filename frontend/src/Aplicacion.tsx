import { useEffect, useRef, useState } from 'react';
import {
  ErrorApi,
  factorizar,
  rotar,
  type Matriz,
  type ResultadoQR,
  type ResultadoRotacion,
} from './clienteApi';
import { PanelInicioSesion } from './components/PanelInicioSesion';
import { EditorMatriz } from './components/EditorMatriz';
import { VistaMatriz } from './components/VistaMatriz';
import { PanelEstadisticas } from './components/PanelEstadisticas';
import { Icono } from './components/Icono';
import interseguroLogo from './assets/descarga.png';

/** Matriz inicial: el ejemplo canónico de la literatura sobre QR. */
const MATRIZ_INICIAL: Matriz = [
  [12, -51, 4],
  [6, 167, -68],
  [-4, 24, -41],
];

type Modo = 'full' | 'reduced';
type Operacion = 'qr' | 'rotate';
type ResultadoCalculo =
  | { operacion: 'qr'; datos: ResultadoQR; matrizEntrada: Matriz }
  | { operacion: 'rotate'; datos: ResultadoRotacion; matrizEntrada: Matriz };

export function Aplicacion() {
  const [token, setToken] = useState<string | null>(null);
  const [matriz, establecerMatriz] = useState<Matriz>(MATRIZ_INICIAL);
  const [operacion, setOperacion] = useState<Operacion>('qr');
  const [modo, establecerModo] = useState<Modo>('full');
  const [decimales, establecerDecimales] = useState(4);
  const [resultado, establecerResultado] = useState<ResultadoCalculo | null>(null);
  const [error, establecerError] = useState<ErrorApi | null>(null);
  const [pendiente, establecerPendiente] = useState(false);
  const referenciaEntrada = useRef<HTMLDivElement>(null);
  const referenciaResultado = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!resultado || !window.matchMedia('(max-width: 760px)').matches) return;

    const idAnimacion = window.requestAnimationFrame(() => {
      const panelResultado = referenciaResultado.current;
      if (!panelResultado) return;

      const reducirMovimiento = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
      panelResultado.scrollIntoView({
        behavior: reducirMovimiento ? 'auto' : 'smooth',
        block: 'start',
      });
      panelResultado.focus({ preventScroll: true });
    });

    return () => window.cancelAnimationFrame(idAnimacion);
  }, [resultado]);

  const ejecutar = async () => {
    if (!token) return;
    establecerPendiente(true);
    establecerError(null);
    const matrizEntrada = matriz.map((fila) => [...fila]);

    try {
      if (operacion === 'qr') {
        const datos = await factorizar(token, matrizEntrada, modo);
        establecerResultado({ operacion, datos, matrizEntrada });
      } else {
        const datos = await rotar(token, matrizEntrada);
        establecerResultado({ operacion, datos, matrizEntrada });
      }
    } catch (causa) {
      establecerResultado(null);
      establecerError(
        causa instanceof ErrorApi
          ? causa
          : new ErrorApi('UNKNOWN_ERROR', 'Ocurrió un error inesperado.'),
      );
      // Un token vencido no se puede recuperar reintentando: se vuelve al login.
      if (causa instanceof ErrorApi && causa.codigo === 'TOKEN_EXPIRED') {
        setToken(null);
      }
    } finally {
      establecerPendiente(false);
    }
  };

  if (!token) {
    return (
      <div className="login-page">
        <Cabecera />
        <Presentacion />
        <main className="shell shell--login">
          <PanelInicioSesion
            alAutenticar={(tokenEmitido) => {
              setToken(tokenEmitido);
              establecerResultado(null);
              establecerError(null);
            }}
          />
        </main>
      </div>
    );
  }

  return (
    <div className="workspace-page">
      <Cabecera alCerrarSesion={() => setToken(null)} />
      <Presentacion compacta />

      <main className="shell shell--workspace">
        <div className="layout">
          <div ref={referenciaEntrada} className="panel panel--input">
            <SelectorOperacion
              operacion={operacion}
              disabled={pendiente}
              onChange={(nuevaOperacion) => {
                setOperacion(nuevaOperacion);
                establecerResultado(null);
                establecerError(null);
              }}
            />

            <EditorMatriz matriz={matriz} alCambiar={establecerMatriz} deshabilitado={pendiente} />

            <div className="controls">
              {operacion === 'qr' && (
                <fieldset className="segmented">
                  <legend className="field__label">Forma</legend>
                  {(['full', 'reduced'] as const).map((option) => (
                    <label key={option} className="segmented__option">
                      <input
                        type="radio"
                        name="mode"
                        value={option}
                        checked={modo === option}
                        disabled={pendiente}
                        onChange={() => establecerModo(option)}
                      />
                      <span>{option === 'full' ? 'Completa' : 'Reducida'}</span>
                    </label>
                  ))}
                </fieldset>
              )}

              <label className="field">
                <span className="field__label">Decimales</span>
                <select
                  className="field__input"
                  value={decimales}
                  disabled={pendiente}
                  onChange={(evento) => establecerDecimales(Number(evento.target.value))}
                >
                  {[2, 4, 6, 10].map((option) => (
                    <option key={option} value={option}>
                      {option}
                    </option>
                  ))}
                </select>
              </label>
            </div>

            <button
              className="button button--primary button--block"
              type="button"
              onClick={ejecutar}
              disabled={pendiente}
            >
              {pendiente ? (
                <span className="indicador-carga" aria-hidden="true" />
              ) : (
                <Icono nombre={operacion === 'qr' ? 'calcular' : 'rotar'} />
              )}
              {pendiente
                ? operacion === 'qr'
                  ? 'Factorizando…'
                  : 'Rotando…'
                : operacion === 'qr'
                  ? 'Factorizar'
                  : 'Rotar 90°'}
            </button>

            {error && (
              <div className="alert" role="alert">
                <span className="alert__code">{error.codigo}</span>
                <span>{error.message}</span>
              </div>
            )}
          </div>

          <div
            ref={referenciaResultado}
            className={`panel panel--output${resultado ? ' panel--resultado' : ' panel--placeholder'}`}
            aria-live="polite"
            aria-busy={pendiente}
            tabIndex={resultado ? -1 : undefined}
          >
            {resultado ? (
              <>
                <button
                  className="button button--ghost resultado__volver"
                  type="button"
                  onClick={() =>
                    referenciaEntrada.current?.scrollIntoView({ behavior: 'smooth', block: 'start' })
                  }
                >
                  <Icono nombre="editar" />
                  Editar matriz
                </button>
                {resultado.operacion === 'qr' ? (
                  <ResultadoQR
                    resultado={resultado.datos}
                    matriz={resultado.matrizEntrada}
                    decimales={decimales}
                  />
                ) : (
                  <ResultadoRotado
                    resultado={resultado.datos}
                    matriz={resultado.matrizEntrada}
                    decimales={decimales}
                  />
                )}
              </>
            ) : (
              <EstadoInicial operacion={operacion} />
            )}
          </div>
        </div>
      </main>
    </div>
  );
}

function Cabecera({ alCerrarSesion }: { alCerrarSesion?: () => void }) {
  return (
    <header className="masthead">
      <img className="masthead__logo" src={interseguroLogo} alt="Interseguro" />

      {alCerrarSesion && (
        <button className="button button--ghost" type="button" onClick={alCerrarSesion}>
          <Icono nombre="salir" />
          <span className="signout__long">Cerrar sesión</span>
          <span className="signout__short">Salir</span>
        </button>
      )}
    </header>
  );
}

/**
 * Franja de marca bajo la cabecera, con el mismo azul de interseguro.pe. Es lo
 * que ancla visualmente la app al resto del sitio: sin ella, el panel de
 * cálculo podría ser la herramienta de cualquier proveedor.
 */
function Presentacion({ compacta = false }: { compacta?: boolean }) {
  return (
    <div className={`hero${compacta ? ' hero--compact' : ''}`}>
      <div className="hero__inner">
        <p className="hero__eyebrow">Herramienta de cálculo</p>
        <h1 className="hero__title">Operaciones con matrices</h1>
        {!compacta && (
          <p className="hero__subtitle">
            Factoriza o rota cualquier matriz y obtén estadísticas del resultado en segundos.
          </p>
        )}
      </div>
    </div>
  );
}

/**
 * El resultado se presenta como la ecuación que representa, no como dos
 * tarjetas independientes: A = Q · R es la afirmación que el servicio acaba de
 * comprobar, y verla escrita hace evidente qué relación tienen las tres piezas.
 */
function SelectorOperacion({
  operacion,
  disabled,
  onChange,
}: {
  operacion: Operacion;
  disabled: boolean;
  onChange: (operacion: Operacion) => void;
}) {
  return (
    <fieldset className="operation-selector">
      <legend className="field__label">Operación</legend>
      <div className="operation-selector__options">
        {(
          [
            ['qr', 'Factorización QR', 'matriz'],
            ['rotate', 'Rotación 90°', 'rotar'],
          ] as const
        ).map(([valor, etiqueta, icono]) => (
          <label key={valor} className="operation-selector__option">
            <input
              type="radio"
              name="operation"
              value={valor}
              checked={operacion === valor}
              disabled={disabled}
              onChange={() => onChange(valor)}
            />
            <span>
              <Icono nombre={icono} />
              {etiqueta}
            </span>
          </label>
        ))}
      </div>
    </fieldset>
  );
}

function ResultadoQR({
  resultado,
  matriz,
  decimales,
}: {
  resultado: ResultadoQR;
  matriz: Matriz;
  decimales: number;
}) {
  return (
    <>
      <EstadoCompletado texto="Factorización completada" />
      <div className="equation">
        <VistaMatriz matriz={matriz} simbolo="A" acento="input" decimales={decimales} />
        <span className="equation__operator" aria-label="es igual a">
          =
        </span>
        <VistaMatriz matriz={resultado.q} simbolo="Q" acento="q" decimales={decimales} />
        <span className="equation__operator" aria-label="multiplicado por">
          ·
        </span>
        <VistaMatriz
          matriz={resultado.r}
          simbolo="R"
          acento="r"
          decimales={decimales}
          revelarTriangulo
        />
      </div>

      <dl className="meta">
        <div className="meta__item">
          <dt>Método</dt>
          <dd style={{ textTransform: 'capitalize' }}>{resultado.meta.algorithm}</dd>
        </div>
        <div className="meta__item">
          <dt>Forma</dt>
          <dd>{resultado.meta.mode === 'full' ? 'completa' : 'reducida'}</dd>
        </div>
        <div className="meta__item">
          {/* El residual es la prueba de que el resultado es correcto: mide
              cuánto se aleja Q·R de la matriz original. */}
          <dt title="Error relativo de reconstrucción ‖Q·R − A‖ / ‖A‖">Residual</dt>
          <dd>{resultado.meta.residual.toExponential(2)}</dd>
        </div>
        <div className="meta__item">
          <dt>Cálculo</dt>
          <dd>{resultado.meta.durationMs.toFixed(2)} ms</dd>
        </div>
      </dl>

      {resultado.statistics && <PanelEstadisticas estadisticas={resultado.statistics} />}
    </>
  );
}

function ResultadoRotado({
  resultado,
  matriz,
  decimales,
}: {
  resultado: ResultadoRotacion;
  matriz: Matriz;
  decimales: number;
}) {
  return (
    <>
      <EstadoCompletado texto="Rotación completada" />
      <div className="equation">
        <VistaMatriz matriz={matriz} simbolo="A" acento="input" decimales={decimales} />
        <span className="equation__operator" aria-label="rotada noventa grados en sentido horario">
          ↻
        </span>
        <VistaMatriz
          matriz={resultado.rotated}
          simbolo="A′"
          acento="rotated"
          decimales={decimales}
        />
      </div>

      <dl className="meta">
        <div className="meta__item">
          <dt>Transformación</dt>
          <dd>Rotación</dd>
        </div>
        <div className="meta__item">
          <dt>Ángulo</dt>
          <dd>{resultado.meta.degrees}°</dd>
        </div>
        <div className="meta__item">
          <dt>Dirección</dt>
          <dd>horaria</dd>
        </div>
        <div className="meta__item">
          <dt>Dimensiones</dt>
          <dd>
            {resultado.meta.rows}×{resultado.meta.cols}
          </dd>
        </div>
      </dl>

      {resultado.statistics && <PanelEstadisticas estadisticas={resultado.statistics} />}
    </>
  );
}

function EstadoInicial({ operacion }: { operacion: Operacion }) {
  const esQR = operacion === 'qr';
  return (
    <div className="placeholder">
      <span className={`placeholder__icon placeholder__icon--${operacion}`} aria-hidden="true">
        <Icono nombre={esQR ? 'matriz' : 'rotar'} />
      </span>
      <p className="placeholder__equation" aria-hidden="true">
        {esQR ? 'A = Q · R' : 'A ↻ A′'}
      </p>
      <p className="placeholder__text">
        Ingresa una matriz y pulsa <strong>{esQR ? 'Factorizar' : 'Rotar 90°'}</strong> para
        obtener {esQR ? 'su descomposición Q · R' : 'su rotación en sentido horario'} y las
        estadísticas asociadas al instante.
      </p>
    </div>
  );
}

function EstadoCompletado({ texto }: { texto: string }) {
  return (
    <div className="estado-completado" role="status">
      <Icono nombre="verificado" />
      <span>{texto}</span>
    </div>
  );
}
