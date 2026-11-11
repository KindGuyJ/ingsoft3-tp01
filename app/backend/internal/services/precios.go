package services

import (
	"fmt"
	"math"

	dom "github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/errors"
)

// Guarda de cambio de precio: protege al admin de un tipeo.
//
// Pasar una remera de 12500 a 125000 (un cero de mas) o a 1250 (uno de menos)
// es un error facil de cometer en el panel, y caro: el catalogo publico lo
// muestra al instante. La guarda no prohibe el cambio grande, pide que se
// confirme a proposito.
//
// Regla 11 (propuesta):
//   - El precio nuevo tiene que ser mayor a cero.
//   - Una baja de mas del 50% o una suba de mas del 100% necesitan confirmacion.
//   - Con confirmacion, cualquier precio valido pasa.
const (
	bajaMaximaSinConfirmar = 0.50 // -50%
	subaMaximaSinConfirmar = 1.00 // +100%
)

type CambioDePrecio struct {
	Anterior  float64
	Nuevo     float64
	Variacion float64 // 0.25 = +25%, -0.40 = -40%
}

// ValidarCambioDePrecio decide si el admin puede pasar de `anterior` a `nuevo`.
func ValidarCambioDePrecio(anterior, nuevo float64, confirmado bool) (*CambioDePrecio, error) {
	if nuevo <= 0 {
		return nil, dom.Validacion("el precio debe ser mayor a cero")
	}
	if anterior <= 0 {
		// Un producto que nunca tuvo precio valido no tiene contra que comparar.
		return &CambioDePrecio{Anterior: anterior, Nuevo: nuevo}, nil
	}

	cambio := &CambioDePrecio{
		Anterior:  anterior,
		Nuevo:     nuevo,
		Variacion: math.Round((nuevo-anterior)/anterior*10000) / 10000,
	}

	if confirmado {
		return cambio, nil
	}
	if cambio.Variacion < -bajaMaximaSinConfirmar {
		return nil, dom.Conflicto("el precio baja un %s: si es a proposito, confirmalo", porcentaje(-cambio.Variacion))
	}
	if cambio.Variacion > subaMaximaSinConfirmar {
		return nil, dom.Conflicto("el precio sube un %s: si es a proposito, confirmalo", porcentaje(cambio.Variacion))
	}
	return cambio, nil
}

// Describir arma el texto que ve el admin antes de guardar.
func (c CambioDePrecio) Describir() string {
	switch {
	case c.Variacion == 0:
		return "el precio no cambia"
	case c.Variacion > 0:
		return fmt.Sprintf("sube un %s (de $%.2f a $%.2f)", porcentaje(c.Variacion), c.Anterior, c.Nuevo)
	default:
		return fmt.Sprintf("baja un %s (de $%.2f a $%.2f)", porcentaje(-c.Variacion), c.Anterior, c.Nuevo)
	}
}

// AplicarAumento sube (o baja, con un porcentaje negativo) una lista de precios
// de una vez, por ejemplo toda una categoria. El resultado se redondea a la
// centena, porque en una tienda los precios terminan en 00. Pasa cada precio
// por la misma guarda: un aumento masivo no esquiva la regla.
func AplicarAumento(precios []float64, aumento float64, confirmado bool) ([]float64, error) {
	if len(precios) == 0 {
		return nil, dom.Validacion("no hay precios para actualizar")
	}
	if aumento == 0 {
		return nil, dom.Validacion("el aumento no puede ser 0%%")
	}

	nuevos := make([]float64, 0, len(precios))
	for i, anterior := range precios {
		nuevo := math.Round(anterior*(1+aumento)/100) * 100
		if _, err := ValidarCambioDePrecio(anterior, nuevo, confirmado); err != nil {
			return nil, fmt.Errorf("precio %d de la lista: %w", i+1, err)
		}
		nuevos = append(nuevos, nuevo)
	}
	return nuevos, nil
}

func porcentaje(x float64) string {
	return fmt.Sprintf("%.0f%%", x*100)
}
