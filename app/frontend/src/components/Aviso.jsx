// Tres estados que aparecen en casi todas las pantallas. Repetirlos en cada
// página era la otra opción; esto es menos código y se ve igual en todos lados.
// Se llaman AvisoError / AvisoExito y no Error / Exito para no tapar el `Error`
// global de JavaScript en los archivos que los importan.

export function Cargando({ texto = 'Cargando…' }) {
  return <p className="aviso">{texto}</p>
}

export function AvisoError({ children }) {
  if (!children) return null
  return <p className="aviso error" role="alert">{children}</p>
}

export function AvisoExito({ children }) {
  if (!children) return null
  return <p className="aviso exito" role="status">{children}</p>
}
