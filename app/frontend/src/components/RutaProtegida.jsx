import { Navigate, useLocation } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'

// Guarda de navegación. Evita mostrar pantallas que van a fallar, pero NO es el
// control de acceso: ese está en el middleware del backend. Acá solo se decide
// qué se dibuja.
export default function RutaProtegida({ soloAdmin = false, children }) {
  const { autenticado, esAdmin } = useAuth()
  const location = useLocation()

  if (!autenticado) {
    // Se recuerda a dónde quería ir para volver ahí después del login.
    return <Navigate to="/login" replace state={{ volverA: location.pathname }} />
  }
  if (soloAdmin && !esAdmin) return <Navigate to="/" replace />

  return children
}
