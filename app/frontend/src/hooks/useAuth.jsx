import { createContext, useContext, useMemo, useState } from 'react'
import {
  api, guardarToken, leerToken, borrarToken,
  guardarUsuario, leerUsuario, borrarUsuario,
} from '../services/api'

// Sesión del usuario. Es un contexto y no un estado suelto porque lo miran tres
// lugares que no son padre-hijo entre sí: la barra de navegación, las rutas
// protegidas y las pantallas que llaman a endpoints autenticados.
const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  // El estado inicial sale de localStorage: si no, al recargar la página el
  // token seguiría estando pero la UI mostraría "no logueado".
  const [usuario, setUsuario] = useState(() => leerUsuario())

  const valor = useMemo(() => ({
    usuario,
    autenticado: !!usuario && !!leerToken(),
    esAdmin: !!usuario?.es_admin,

    async login(email, password) {
      const { token, usuario: u } = await api.login({ email, password })
      guardarToken(token)
      guardarUsuario(u)
      setUsuario(u)
      return u
    },

    logout() {
      borrarToken()
      borrarUsuario()
      setUsuario(null)
    },

    // El token vence (JWT_HORAS). Cuando el backend contesta 401, la sesión
    // guardada ya no sirve: se limpia acá para que la UI no siga mostrando al
    // usuario como logueado. Quien llama decide si además redirige.
    sesionVencida(err) {
      if (err?.status !== 401) return false
      borrarToken()
      borrarUsuario()
      setUsuario(null)
      return true
    },
  }), [usuario])

  return <AuthContext.Provider value={valor}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth se usó fuera de <AuthProvider>')
  return ctx
}
