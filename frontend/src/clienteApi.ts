/**
 * Cliente de la API Go.
 *
 * Todas las rutas son relativas: en desarrollo las redirige el proxy de Vite y
 * en producción las redirige nginx. Así el frontend no necesita conocer la URL
 * de la API ni cambiarla entre entornos, y no hay CORS que resolver.
 */

export type Matriz = number[][];

export interface RespuestaInicioSesion {
  token: string;
  tokenType: string;
  expiresAt: string;
  expiresIn: number;
}

export interface EstadisticasMatriz {
  max: number;
  min: number;
  average: number;
  sum: number;
  count: number;
  rows: number;
  cols: number;
  isSquare: boolean;
  isDiagonal: boolean;
  tolerance: number;
}

export interface Estadisticas {
  overall: {
    max: number;
    min: number;
    average: number;
    sum: number;
    count: number;
  };
  perMatrix: Record<string, EstadisticasMatriz>;
  anyDiagonal: boolean;
  toleranceFactor: number;
}

export interface ResultadoQR {
  q: Matriz;
  r: Matriz;
  meta: {
    rows: number;
    cols: number;
    mode: 'full' | 'reduced';
    algorithm: 'householder';
    residual: number;
    durationMs: number;
    requestId?: string;
  };
  statistics?: Estadisticas;
}

export interface ResultadoRotacion {
  rotated: Matriz;
  meta: {
    rows: number;
    cols: number;
    direction: 'clockwise';
    degrees: 90;
    requestId?: string;
  };
  statistics?: Estadisticas;
}

/** Error de la API, con el código estable que devuelven ambos servicios. */
export class ErrorApi extends Error {
  constructor(
    readonly codigo: string,
    mensaje: string,
    readonly detalles?: Record<string, unknown>,
  ) {
    super(mensaje);
    this.name = 'ErrorApi';
  }
}

const TIEMPO_ESPERA_SOLICITUD_MS = 15_000;

type ValidadorRespuesta<T> = (valor: unknown) => valor is T;

/** Envía la petición y convierte cualquier fallo en un ErrorApi legible. */
async function enviar<T>(
  ruta: string,
  opciones: RequestInit,
  validarRespuesta: ValidadorRespuesta<T>,
): Promise<T> {
  let respuesta: Response;
  try {
    respuesta = await fetch(ruta, {
      ...opciones,
      signal: AbortSignal.timeout(TIEMPO_ESPERA_SOLICITUD_MS),
    });
  } catch (causa) {
    if (causa instanceof DOMException && ['AbortError', 'TimeoutError'].includes(causa.name)) {
      throw new ErrorApi('TIMEOUT_ERROR', 'El servidor tardó demasiado en responder. Intenta nuevamente.');
    }
    // fetch solo rechaza por fallos de red, no por códigos de error HTTP.
    throw new ErrorApi('NETWORK_ERROR', 'No se pudo contactar al servidor. ¿Está la API en marcha?');
  }

  let cuerpo: unknown;
  try {
    cuerpo = await respuesta.json();
  } catch {
    throw new ErrorApi(
      'INVALID_RESPONSE',
      respuesta.ok
        ? 'El servidor devolvió una respuesta que no es JSON válido.'
        : `El servidor respondió ${respuesta.status} sin un error JSON válido.`,
    );
  }

  if (!respuesta.ok) {
    const error = esRegistro(cuerpo) && esRegistro(cuerpo.error) ? cuerpo.error : undefined;
    throw new ErrorApi(
      typeof error?.code === 'string' ? error.code : 'UNKNOWN_ERROR',
      typeof error?.message === 'string' ? error.message : `El servidor respondió ${respuesta.status}.`,
      esRegistro(error?.details) ? error.details : undefined,
    );
  }

  if (!validarRespuesta(cuerpo)) {
    throw new ErrorApi('INVALID_RESPONSE', 'El servidor devolvió una respuesta con una estructura inesperada.');
  }

  return cuerpo;
}

export function iniciarSesion(usuario: string, contrasena: string): Promise<RespuestaInicioSesion> {
  return enviar('/api/v1/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: usuario, password: contrasena }),
  }, esRespuestaInicioSesion);
}

export function factorizar(
  token: string,
  matriz: Matriz,
  modo: 'full' | 'reduced',
): Promise<ResultadoQR> {
  return enviar(`/api/v1/qr?mode=${modo}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ matrix: matriz }),
  }, esResultadoQR);
}

/** Solicita a Go la rotación y las estadísticas calculadas por Node. */
export function rotar(token: string, matriz: Matriz): Promise<ResultadoRotacion> {
  return enviar('/api/v1/rotate', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ matrix: matriz }),
  }, esResultadoRotacion);
}

function esRegistro(valor: unknown): valor is Record<string, unknown> {
  return typeof valor === 'object' && valor !== null && !Array.isArray(valor);
}

function esNumeroFinito(valor: unknown): valor is number {
  return typeof valor === 'number' && Number.isFinite(valor);
}

function esMatriz(valor: unknown): valor is Matriz {
  return (
    Array.isArray(valor) &&
    valor.length > 0 &&
    valor.every(
      (fila) => Array.isArray(fila) && fila.length > 0 && fila.every((celda) => esNumeroFinito(celda)),
    )
  );
}

function esRespuestaInicioSesion(valor: unknown): valor is RespuestaInicioSesion {
  return (
    esRegistro(valor) &&
    typeof valor.token === 'string' &&
    valor.token.length > 0 &&
    typeof valor.tokenType === 'string' &&
    typeof valor.expiresAt === 'string' &&
    esNumeroFinito(valor.expiresIn)
  );
}

function esEstadisticasMatriz(valor: unknown): valor is EstadisticasMatriz {
  return (
    esRegistro(valor) &&
    esNumeroFinito(valor.max) &&
    esNumeroFinito(valor.min) &&
    esNumeroFinito(valor.average) &&
    esNumeroFinito(valor.sum) &&
    esNumeroFinito(valor.count) &&
    esNumeroFinito(valor.rows) &&
    esNumeroFinito(valor.cols) &&
    typeof valor.isSquare === 'boolean' &&
    typeof valor.isDiagonal === 'boolean' &&
    esNumeroFinito(valor.tolerance)
  );
}

function esEstadisticas(valor: unknown): valor is Estadisticas {
  if (!esRegistro(valor) || !esRegistro(valor.overall) || !esRegistro(valor.perMatrix)) return false;

  const resumen = valor.overall;
  return (
    esNumeroFinito(resumen.max) &&
    esNumeroFinito(resumen.min) &&
    esNumeroFinito(resumen.average) &&
    esNumeroFinito(resumen.sum) &&
    esNumeroFinito(resumen.count) &&
    Object.values(valor.perMatrix).every((matriz) => esEstadisticasMatriz(matriz)) &&
    typeof valor.anyDiagonal === 'boolean' &&
    esNumeroFinito(valor.toleranceFactor)
  );
}

function tieneEstadisticasValidas(valor: Record<string, unknown>): boolean {
  return valor.statistics === undefined || esEstadisticas(valor.statistics);
}

function esResultadoQR(valor: unknown): valor is ResultadoQR {
  if (!esRegistro(valor) || !esRegistro(valor.meta)) return false;

  const metadatos = valor.meta;
  return (
    esMatriz(valor.q) &&
    esMatriz(valor.r) &&
    esNumeroFinito(metadatos.rows) &&
    esNumeroFinito(metadatos.cols) &&
    (metadatos.mode === 'full' || metadatos.mode === 'reduced') &&
    metadatos.algorithm === 'householder' &&
    esNumeroFinito(metadatos.residual) &&
    esNumeroFinito(metadatos.durationMs) &&
    (metadatos.requestId === undefined || typeof metadatos.requestId === 'string') &&
    tieneEstadisticasValidas(valor)
  );
}

function esResultadoRotacion(valor: unknown): valor is ResultadoRotacion {
  if (!esRegistro(valor) || !esRegistro(valor.meta)) return false;

  const metadatos = valor.meta;
  return (
    esMatriz(valor.rotated) &&
    esNumeroFinito(metadatos.rows) &&
    esNumeroFinito(metadatos.cols) &&
    metadatos.direction === 'clockwise' &&
    metadatos.degrees === 90 &&
    (metadatos.requestId === undefined || typeof metadatos.requestId === 'string') &&
    tieneEstadisticasValidas(valor)
  );
}
