import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '../services/api'
import { useAuth } from '../hooks/useAuth'
import { AvisoError } from '../components/Aviso'
import { emailValido } from '../utils'

const LARGO_MINIMO = 8 // igual que el binding:"min=8" del backend

export default function Register() {
  const { login } = useAuth()
  const navigate = useNavigate()

  const [datos, setDatos] = useState({
    nombre: '', apellido: '', email: '', password: '',
  })
  const [error, setError] = useState('')
  const [enviando, setEnviando] = useState(false)

  function cambiar(campo) {
    return (e) => setDatos((d) => ({ ...d, [campo]: e.target.value }))
  }

  // Comportamiento 4 del TP5: con un email inválido el formulario NO se puede
  // enviar. La validación es emailValido() de utils.js — la misma función que
  // testea utils.test.js, no una copia dentro del componente.
  const emailOk = emailValido(datos.email)
  const formularioValido =
    datos.nombre.trim() !== '' &&
    datos.apellido.trim() !== '' &&
    emailOk &&
    datos.password.length >= LARGO_MINIMO

  async function enviar(e) {
    e.preventDefault()
    setError('')
    setEnviando(true)
    try {
      await api.registro(datos)
      // El registro NO devuelve token (el backend lo decidió así). El front
      // hace el login inmediatamente después para no pedir las credenciales
      // dos veces seguidas.
      await login(datos.email, datos.password)
      navigate('/', { replace: true })
    } catch (err) {
      setError(err.message)
    } finally {
      setEnviando(false)
    }
  }

  return (
    <section className="formulario">
      <h1>Crear cuenta</h1>
      <AvisoError>{error}</AvisoError>

      {/* noValidate: la validación la hace React, no el navegador. Si no, cada
          browser muestra su propio cartel y el comportamiento deja de ser el
          mismo en todos lados (y de ser testeable). */}
      <form onSubmit={enviar} noValidate>
        <label>
          Nombre
          <input value={datos.nombre} onChange={cambiar('nombre')} />
        </label>

        <label>
          Apellido
          <input value={datos.apellido} onChange={cambiar('apellido')} />
        </label>

        <label>
          Email
          <input
            type="email"
            value={datos.email}
            onChange={cambiar('email')}
            autoComplete="email"
            aria-invalid={datos.email !== '' && !emailOk}
          />
        </label>
        {datos.email !== '' && !emailOk && (
          <p className="ayuda error">Ese email no tiene un formato válido.</p>
        )}

        <label>
          Contraseña
          <input
            type="password"
            value={datos.password}
            onChange={cambiar('password')}
            autoComplete="new-password"
          />
        </label>
        {datos.password !== '' && datos.password.length < LARGO_MINIMO && (
          <p className="ayuda error">La contraseña necesita al menos {LARGO_MINIMO} caracteres.</p>
        )}

        <button type="submit" className="primario" disabled={!formularioValido || enviando}>
          {enviando ? 'Creando…' : 'Crear cuenta'}
        </button>
      </form>

      <p>¿Ya tenés cuenta? <Link to="/login">Ingresar</Link></p>
    </section>
  )
}
