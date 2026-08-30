import { useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'
import { AvisoError } from '../components/Aviso'
import { emailValido } from '../utils'

export default function Login() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  // Si llegó acá porque una ruta protegida lo mandó, vuelve a donde iba.
  const volverA = location.state?.volverA || '/'

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [enviando, setEnviando] = useState(false)

  const formularioValido = emailValido(email) && password.length > 0

  async function enviar(e) {
    e.preventDefault()
    setError('')
    setEnviando(true)
    try {
      await login(email, password)
      navigate(volverA, { replace: true })
    } catch (err) {
      // El backend devuelve el MISMO 401 para email inexistente, contraseña
      // incorrecta y usuario inactivo, a propósito: distinguirlos le diría a un
      // atacante qué emails existen. El front no intenta adivinar cuál fue.
      setError(err.message)
    } finally {
      setEnviando(false)
    }
  }

  return (
    <section className="formulario">
      <h1>Ingresar</h1>
      <AvisoError>{error}</AvisoError>

      <form onSubmit={enviar} noValidate>
        <label>
          Email
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            autoComplete="email"
          />
        </label>
        {email !== '' && !emailValido(email) && (
          <p className="ayuda error">Ese email no tiene un formato válido.</p>
        )}

        <label>
          Contraseña
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
          />
        </label>

        <button type="submit" className="primario" disabled={!formularioValido || enviando}>
          {enviando ? 'Ingresando…' : 'Ingresar'}
        </button>
      </form>

      <p>¿No tenés cuenta? <Link to="/registro">Crear una</Link></p>
    </section>
  )
}
