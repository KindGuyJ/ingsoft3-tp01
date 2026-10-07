package services

import (
	"reflect"
	"testing"

	dom "github.com/KindGuyJ/ingsoft3-tp01/app/backend/internal/errors"
)

// Regla 10: cuotas. Un test por cada camino que declara cuotas.go.

const minimoTest = 5000

func TestCalcularCuotas_PlanesValidos(t *testing.T) {
	casos := []struct {
		nombre string
		total  float64
		cuotas int
		want   PlanCuotas
	}{
		{"una cuota es el total", 60000, 1, PlanCuotas{Cuotas: 1, Recargo: 0, Total: 60000, Cuota: 60000}},
		{"tres cuotas sin interes", 60000, 3, PlanCuotas{Cuotas: 3, Recargo: 0, Total: 60000, Cuota: 20000}},
		{"seis cuotas con 10%", 60000, 6, PlanCuotas{Cuotas: 6, Recargo: 0.10, Total: 66000, Cuota: 11000}},
		{"doce cuotas con 25%", 60000, 12, PlanCuotas{Cuotas: 12, Recargo: 0.25, Total: 75000, Cuota: 6250}},
		{"redondea la cuota a centavos", 10000, 3, PlanCuotas{Cuotas: 3, Recargo: 0, Total: 10000, Cuota: 3333.33}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			plan, err := CalcularCuotas(c.total, c.cuotas, 0)
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if *plan != c.want {
				t.Errorf("plan = %+v, se esperaba %+v", *plan, c.want)
			}
		})
	}
}

func TestCalcularCuotas_Rechazos(t *testing.T) {
	casos := []struct {
		nombre string
		total  float64
		cuotas int
	}{
		{"total cero", 0, 1},
		{"total negativo", -100, 3},
		{"cantidad de cuotas no permitida", 60000, 5},
		{"cero cuotas", 60000, 0},
		{"la cuota queda por debajo del minimo", 30000, 12}, // 37500 / 12 = 3125 < 5000
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			_, err := CalcularCuotas(c.total, c.cuotas, minimoTest)
			if k := kindDe(t, err); k != dom.KindValidacion {
				t.Errorf("kind = %v, se esperaba KindValidacion", k)
			}
		})
	}
}

// Borde del minimo: una cuota IGUAL al minimo se acepta (es <, no <=).
func TestCalcularCuotas_CuotaJustoEnElMinimo(t *testing.T) {
	plan, err := CalcularCuotas(15000, 3, minimoTest) // 15000 / 3 = 5000

	if err != nil {
		t.Fatalf("una cuota igual al minimo tiene que aceptarse: %v", err)
	}
	if plan.Cuota != 5000 {
		t.Errorf("cuota = %v, se esperaba 5000", plan.Cuota)
	}
}

// En una sola cuota no hay minimo: un pedido chico se puede pagar entero.
func TestCalcularCuotas_UnaCuotaNoTieneMinimo(t *testing.T) {
	if _, err := CalcularCuotas(1000, 1, minimoTest); err != nil {
		t.Errorf("una cuota no deberia exigir minimo: %v", err)
	}
}

func TestPlanesDisponibles_OrdenadosYSinLosQueNoLleganAlMinimo(t *testing.T) {
	// 30000: 1 y 3 cuotas pasan (10000), 6 cuotas = 33000/6 = 5500 pasa,
	// 12 cuotas = 37500/12 = 3125 no llega al minimo.
	planes := PlanesDisponibles(30000, minimoTest)

	var cuotas []int
	for _, p := range planes {
		cuotas = append(cuotas, p.Cuotas)
	}
	if want := []int{1, 3, 6}; !reflect.DeepEqual(cuotas, want) {
		t.Errorf("cuotas ofrecidas = %v, se esperaba %v", cuotas, want)
	}
}

func TestPlanesDisponibles_TotalInvalidoNoOfreceNada(t *testing.T) {
	if planes := PlanesDisponibles(0, minimoTest); len(planes) != 0 {
		t.Errorf("con total 0 no se ofrece ningun plan, llegaron %d", len(planes))
	}
}
