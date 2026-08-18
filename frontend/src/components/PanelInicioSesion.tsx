import { useId, useState } from 'react';
import { ErrorApi, iniciarSesion } from '../clienteApi';
import { Icono } from './Icono';

interface PropiedadesPanelInicioSesion {
  alAutenticar: (token: string) => void;
}

/**
 * Pantalla de acceso.
 *
 * El token que devuelve la API se guarda en memoria (estado de React) y no en
 * localStorage: un token en localStorage queda accesible a cualquier script de
 * la página, de modo que un XSS bastaría para robarlo. El costo es que recargar
 * la página obliga a entrar de nuevo, aceptable para una sesión de 15 minutos.
 */
export function PanelInicioSesion({ alAutenticar }: PropiedadesPanelInicioSesion) {
  const idUsuario = useId();
  const idContrasena = useId();

  const [usuario, establecerUsuario] = useState('demo');
  const [contrasena, establecerContrasena] = useState('');
  const [error, establecerError] = useState<string | null>(null);
  const [pendiente, establecerPendiente] = useState(false);

  const enviar = async (evento: React.FormEvent) => {
    evento.preventDefault();
    establecerPendiente(true);
    establecerError(null);

    try {
      const respuesta = await iniciarSesion(usuario, contrasena);
      alAutenticar(respuesta.token);
    } catch (causa) {
      establecerError(causa instanceof ErrorApi ? causa.message : 'No se pudo iniciar sesión.');
      establecerPendiente(false);
    }
  };

  return (
    <div className="gate">
      <form className="gate__card" onSubmit={enviar}>
        <div className="gate__heading">
          <span className="gate__icon" aria-hidden="true">
            <Icono nombre="matriz" />
          </span>
          <h2 className="gate__title">Iniciar sesión</h2>
        </div>
        <p className="gate__lead">
        Ingresa tus credenciales para acceder a las operaciones con matrices.
        </p>

        <label className="field" htmlFor={idUsuario}>
          <span className="field__label">Usuario</span>
          <input
            id={idUsuario}
            className="field__input field__input--wide"
            type="text"
            autoComplete="username"
            value={usuario}
            onChange={(evento) => establecerUsuario(evento.target.value)}
            required
          />
        </label>

        <label className="field" htmlFor={idContrasena}>
          <span className="field__label">Contraseña</span>
          <input
            id={idContrasena}
            className="field__input field__input--wide"
            type="password"
            autoComplete="current-password"
            value={contrasena}
            onChange={(evento) => establecerContrasena(evento.target.value)}
            required
          />
        </label>

        {error && (
          <p className="alert" role="alert">
            {error}
          </p>
        )}

        <button className="button button--primary" type="submit" disabled={pendiente}>
          {pendiente ? <span className="indicador-carga" aria-hidden="true" /> : <Icono nombre="entrar" />}
          {pendiente ? 'Verificando…' : 'Entrar'}
        </button>

        <p className="gate__note">
          Cuenta de demostración: usuario <code>demo</code>, contraseña <code>demo1234</code>.
        </p>
      </form>
    </div>
  );
}
