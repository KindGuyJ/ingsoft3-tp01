// Cantidad con botones − / + en vez de un <input type="number">.
//
// No es capricho: con un campo de texto siempre existe un estado intermedio
// inválido (el campo vacío mientras se reescribe el número), y recortarlo al
// stock en cada tecla hace que escribir "3" sobre un "1" quede en "13". Con dos
// botones, el valor SIEMPRE está entre 1 y el stock y no hay caso raro que
// explicar.
export default function SelectorCantidad({ valor, max, etiqueta, onCambiar }) {
  return (
    <span className="selector-cantidad">
      <button
        type="button"
        className="paso"
        aria-label={`Quitar una unidad de ${etiqueta}`}
        disabled={valor <= 1}
        onClick={() => onCambiar(valor - 1)}
      >
        −
      </button>

      <span className="valor" aria-label={`Cantidad de ${etiqueta}`}>{valor}</span>

      <button
        type="button"
        className="paso"
        aria-label={`Agregar una unidad de ${etiqueta}`}
        disabled={valor >= max}
        onClick={() => onCambiar(valor + 1)}
      >
        +
      </button>
    </span>
  )
}
