import { Link, NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'
import { useCarrito } from '../hooks/useCarrito'

// Cáscara común: barra de navegación + el contenido de la ruta actual.
export default function Layout() {
  const { usuario, autenticado, esAdmin, logout } = useAuth()
  const { unidades } = useCarrito()
  const navigate = useNavigate()

  function salir() {
    logout()
    navigate('/')
  }

  return (
    <div className="app">
      <header className="barra">
        <Link to="/" className="marca">Tienda</Link>

        <nav className="nav">
          <NavLink to="/">Catálogo</NavLink>
          <NavLink to="/carrito">
            Carrito{unidades > 0 && <span className="badge">{unidades}</span>}
          </NavLink>
          {autenticado && <NavLink to="/mis-pedidos">Mis pedidos</NavLink>}
          {/* El menú de admin se oculta por comodidad, no por seguridad: quien
              lo fuerce igual recibe 403 del backend. */}
          {esAdmin && <NavLink to="/admin">Admin</NavLink>}
        </nav>

        <div className="sesion">
          {autenticado ? (
            <>
              <span className="hola">Hola, {usuario.nombre}</span>
              <button type="button" className="link" onClick={salir}>Salir</button>
            </>
          ) : (
            <>
              <NavLink to="/login">Ingresar</NavLink>
              <NavLink to="/registro">Crear cuenta</NavLink>
            </>
          )}
        </div>
      </header>

      <main className="contenido">
        <Outlet />
      </main>

      <footer className="pie">
        Ingeniería del Software 3 — UCC 2026
      </footer>
    </div>
  )
}
