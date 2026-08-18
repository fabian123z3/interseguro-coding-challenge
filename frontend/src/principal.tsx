import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { Aplicacion } from './Aplicacion';
import './estilos.css';

const contenedor = document.getElementById('root');
if (!contenedor) {
  throw new Error('no se encontró el contenedor #root en index.html');
}

createRoot(contenedor).render(
  <StrictMode>
    <Aplicacion />
  </StrictMode>,
);
