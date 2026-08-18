import type { SVGProps } from 'react';

export type NombreIcono =
  | 'calcular'
  | 'editar'
  | 'entrar'
  | 'estadisticas'
  | 'matriz'
  | 'rotar'
  | 'salir'
  | 'verificado';

interface PropiedadesIcono extends SVGProps<SVGSVGElement> {
  nombre: NombreIcono;
}

/** Iconos lineales propios que heredan el color y no requieren dependencias. */
export function Icono({ nombre, className = '', ...propiedades }: PropiedadesIcono) {
  return (
    <svg
      aria-hidden="true"
      className={`icono ${className}`.trim()}
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth="1.8"
      {...propiedades}
    >
      {nombre === 'matriz' && (
        <>
          <rect x="4" y="4" width="16" height="16" rx="2" />
          <path d="M4 10h16M10 4v16" />
          <circle cx="15" cy="15" r="1.2" fill="currentColor" stroke="none" />
        </>
      )}
      {nombre === 'rotar' && (
        <>
          <path d="M20 7v5h-5" />
          <path d="M19 12a7 7 0 1 1-2.1-5" />
          <path d="m17 4 2 3" />
        </>
      )}
      {nombre === 'calcular' && (
        <>
          <path d="M5 4h14v16H5z" />
          <path d="M8 8h8M8 12h2M14 12h2M8 16h2M14 16h2" />
        </>
      )}
      {nombre === 'editar' && (
        <>
          <path d="M4 20h4l10.5-10.5a2.1 2.1 0 0 0-3-3L5 17v3Z" />
          <path d="m13.8 8.2 3 3" />
        </>
      )}
      {nombre === 'estadisticas' && (
        <>
          <path d="M5 19V9M12 19V5M19 19v-7" />
          <path d="M3 19h18" />
        </>
      )}
      {nombre === 'entrar' && (
        <>
          <path d="M13 5h6v14h-6" />
          <path d="M10 8 6 12l4 4M6 12h10" />
        </>
      )}
      {nombre === 'salir' && (
        <>
          <path d="M11 5H5v14h6" />
          <path d="m14 8 4 4-4 4M8 12h10" />
        </>
      )}
      {nombre === 'verificado' && (
        <>
          <circle cx="12" cy="12" r="9" />
          <path d="m8 12 2.6 2.6L16.5 9" />
        </>
      )}
    </svg>
  );
}
