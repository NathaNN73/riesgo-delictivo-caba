package dataset

import (
	"math"
	"math/rand"
	"sort"

	"riesgo-delictivo/internal/loader"
)

// Ejemplo es una fila de entrenamiento: vector de features + etiqueta.
type Ejemplo struct {
	X []float64
	Y float64
}

// Dataset etiquetado, con metadatos para reconstruir predicciones en la API (PC4).
type Dataset struct {
	Ejemplos  []Ejemplo
	NumFeats  int
	Umbral    int // conteo del percentil 75 usado como corte
	MaxBarrio int
	FeatNames []string
}

// Construir genera un ejemplo por celda observada, etiquetado contra el P75.
func Construir(res *loader.Resultado, semilla int64) *Dataset {
	conteos := make([]int, 0, len(res.Conteos))
	maxBarrio := 0
	for c, n := range res.Conteos {
		conteos = append(conteos, n)
		if c.BarrioID > maxBarrio {
			maxBarrio = c.BarrioID
		}
	}
	sort.Ints(conteos)
	umbral := conteos[int(0.75*float64(len(conteos)-1))]

	ds := &Dataset{Umbral: umbral, MaxBarrio: maxBarrio}
	ds.FeatNames = []string{"bias", "sin_hora", "cos_hora", "dia_norm", "finde", "barrio_norm", "comuna_norm"}
	ds.NumFeats = len(ds.FeatNames)

	for c, n := range res.Conteos {
		x := Features(c.Hora, c.DiaSemana, c.BarrioID, res.ComunaBarrio[c.BarrioID], maxBarrio)
		y := 0.0
		if n >= umbral {
			y = 1.0
		}
		ds.Ejemplos = append(ds.Ejemplos, Ejemplo{X: x, Y: y})
	}

	// Barajar para que los shards de entrenamiento sean homogéneos.
	rng := rand.New(rand.NewSource(semilla))
	rng.Shuffle(len(ds.Ejemplos), func(i, j int) {
		ds.Ejemplos[i], ds.Ejemplos[j] = ds.Ejemplos[j], ds.Ejemplos[i]
	})
	return ds
}

// Features construye el vector de entrada para una combinación zona-tiempo.
func Features(hora, diaSemana, barrioID, comuna, maxBarrio int) []float64 {
	finde := 0.0
	if diaSemana >= 5 { // sábado o domingo
		finde = 1.0
	}
	return []float64{
		1.0, // bias
		math.Sin(2 * math.Pi * float64(hora) / 24.0),
		math.Cos(2 * math.Pi * float64(hora) / 24.0),
		float64(diaSemana) / 6.0,
		finde,
		float64(barrioID) / float64(max(maxBarrio, 1)),
		float64(comuna) / 15.0,
	}
}

// Dividir separa el dataset en entrenamiento y prueba (holdout).
func (d *Dataset) Dividir(propTest float64) (train, test []Ejemplo) {
	n := int(float64(len(d.Ejemplos)) * (1 - propTest))
	return d.Ejemplos[:n], d.Ejemplos[n:]
}
