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
      {/* Dos pisos: arriba la sesion, la marca centrada y el carrito; abajo la
          barra negra con las secciones. */}
      <header className="cabecera">
        <div className="cabecera-arriba">
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

          <Link to="/" className="marca">
            <img src="/favicon-32.png" alt="" className="marca-logo" />
            <span className="marca-nombre">Tienda</span>
            <span className="marca-bajada">Indumentaria</span>
          </Link>

          <NavLink to="/carrito" className="carrito-link">
            Carrito{unidades > 0 && <span className="badge">{unidades}</span>}
          </NavLink>
        </div>

        <nav className="nav">
          <NavLink to="/" end>Catálogo</NavLink>
          {autenticado && <NavLink to="/mis-pedidos">Mis pedidos</NavLink>}
          {/* El menú de admin se oculta por comodidad, no por seguridad: quien
              lo fuerce igual recibe 403 del backend. */}
          {esAdmin && <NavLink to="/admin">Admin</NavLink>}
        </nav>
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
