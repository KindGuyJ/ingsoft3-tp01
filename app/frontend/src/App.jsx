import { Routes, Route, Link } from 'react-router-dom'

import Layout from './components/Layout'
import RutaProtegida from './components/RutaProtegida'
import Catalogo from './pages/Catalogo'
import ProductoDetalle from './pages/ProductoDetalle'
import Carrito from './pages/Carrito'
import Login from './pages/Login'
import Register from './pages/Register'
import MisPedidos from './pages/MisPedidos'
import AdminPanel from './pages/AdminPanel'

export default function App() {
  return (
    <Routes>
      {/* Todas las rutas cuelgan del Layout, que dibuja la barra de navegación
          una sola vez y renderiza la pantalla actual en su <Outlet />. */}
      <Route element={<Layout />}>
        <Route path="/" element={<Catalogo />} />
        <Route path="/productos/:id" element={<ProductoDetalle />} />
        <Route path="/carrito" element={<Carrito />} />
        <Route path="/login" element={<Login />} />
        <Route path="/registro" element={<Register />} />

        <Route
          path="/mis-pedidos"
          element={<RutaProtegida><MisPedidos /></RutaProtegida>}
        />
        <Route
          path="/admin"
          element={<RutaProtegida soloAdmin><AdminPanel /></RutaProtegida>}
        />

        <Route path="*" element={<NoEncontrado />} />
      </Route>
    </Routes>
  )
}

function NoEncontrado() {
  return (
    <section>
      <h1>404</h1>
      <p>Esa página no existe. <Link to="/">Volver al catálogo</Link></p>
    </section>
  )
}
