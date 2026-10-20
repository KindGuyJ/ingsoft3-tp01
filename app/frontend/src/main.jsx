import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router'
import * as Sentry from '@sentry/react'
import App from './App'
import { AuthProvider } from './hooks/useAuth'
import { CarritoProvider } from './hooks/useCarrito'
import './styles/index.css'

// TP9: Sentry cuenta que falla, que tarda y que se usa en el navegador.
// El DSN va escrito a proposito: no es un secreto. Solo sirve para ENVIAR
// eventos (no para leerlos) y de todos modos viaja en el JS que baja cualquiera.
// El environment es el host, asi QA y PROD se separan solos sin rebuildear la
// imagen (es la misma en los dos, TP7). En localhost no manda nada.
Sentry.init({
  dsn: 'https://202de8ad45e51fd8640cd36b515c812b@o4512284869459968.ingest.us.sentry.io/4512284917366784',
  environment: window.location.hostname,
  enabled: window.location.hostname !== 'localhost',
  integrations: [Sentry.browserTracingIntegration()],
  tracesSampleRate: 1.0,
  // Las mediciones del navegador incluyen las llamadas a la api del mismo
  // origen (/api/...): asi se ve el pedido entero, front + back.
  tracePropagationTargets: [/^\/api\//],
})

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <BrowserRouter>
      {/* Sesión y carrito envuelven a toda la app: los miran pantallas que no
          son padre-hijo entre sí (la barra, el detalle, el checkout). */}
      <AuthProvider>
        <CarritoProvider>
          <App />
        </CarritoProvider>
      </AuthProvider>
    </BrowserRouter>
  </React.StrictMode>,
)
