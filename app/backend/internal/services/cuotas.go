package services

import (
	"math"
	"sort"

	dom "github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/errors"
)

// Cuotas: cuanto paga el cliente si divide el total del pedido en cuotas.
//
// Es solo un CALCULO para mostrar las opciones antes de confirmar: no habla
// con ningun medio de pago (el checkout sigue marcando el pedido como pagado
// sin salir a la red, ver CLAUDE.md).
//
// Regla 10:
//   - Se aceptan 1, 3, 6 o 12 cuotas.
//   - 1 y 3 son sin interes; 6 y 12 tienen recargo sobre el total.
//   - Ninguna cuota puede quedar por debajo del minimo por cuota: un pedido
//     chico no se financia en 12 cuotas de $800.

// recargoPorCuotas es la politica comercial: cantidad de cuotas -> recargo.
var recargoPorCuotas = map[int]float64{
	1:  0,
	3:  0,
	6:  0.10,
	12: 0.25,
}

type PlanCuotas struct {
	Cuotas  int
	Recargo float64 // 0.10 = 10%
	Total   float64 // total con el recargo aplicado
	Cuota   float64 // valor de cada cuota, redondeado a centavos
}

// CalcularCuotas arma el plan para una cantidad de cuotas concreta.
func CalcularCuotas(total float64, cuotas int, minimoPorCuota float64) (*PlanCuotas, error) {
	if total <= 0 {
		return nil, dom.Validacion("el total debe ser mayor a cero")
	}
	recargo, ok := recargoPorCuotas[cuotas]
	if !ok {
		return nil, dom.Validacion("no se puede pagar en %d cuotas: se aceptan 1, 3, 6 o 12", cuotas)
	}

	conRecargo := total * (1 + recargo)
	cuota := redondearCentavos(conRecargo / float64(cuotas))

	// En una sola cuota no hay minimo: es pagar el total.
	if cuotas > 1 && cuota < minimoPorCuota {
		return nil, dom.Validacion("en %d cuotas cada una quedaria en $%.2f, por debajo del minimo de $%.2f",
			cuotas, cuota, minimoPorCuota)
	}

	return &PlanCuotas{
		Cuotas:  cuotas,
		Recargo: recargo,
		Total:   redondearCentavos(conRecargo),
		Cuota:   cuota,
	}, nil
}

// PlanesDisponibles devuelve todos los planes validos para un total, de menos
// a mas cuotas. Los que no llegan al minimo por cuota no se ofrecen.
func PlanesDisponibles(total float64, minimoPorCuota float64) []PlanCuotas {
	opciones := make([]int, 0, len(recargoPorCuotas))
	for c := range recargoPorCuotas {
		opciones = append(opciones, c)
	}
	// Un map de Go no tiene orden: sin esto, la lista saldria mezclada.
	sort.Ints(opciones)

	planes := []PlanCuotas{}
	for _, c := range opciones {
		plan, err := CalcularCuotas(total, c, minimoPorCuota)
		if err != nil {
			continue
		}
		planes = append(planes, *plan)
	}
	return planes
}

func redondearCentavos(x float64) float64 {
	return math.Round(x*100) / 100
}
