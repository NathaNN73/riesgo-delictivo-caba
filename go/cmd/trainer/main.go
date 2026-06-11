// Trainer
//
//	Etapa 1: carga concurrente del CSV limpio (loader, fan-out/fan-in).
//	Etapa 2: construcción de celdas y target (dataset, percentil 75).
//	Etapa 3: entrenamiento paralelo de la regresión logística (ml).
//	Etapa 4: evaluación (metrics) y exportación del modelo (model.json).
//
// Uso:
//
//	go run ./cmd/trainer -datos ../data/datos_limpios.csv -epocas 300
//	go run -race ./cmd/trainer ...   para detectar condiciones de carrera
package main

import (
	"flag"
	"fmt"
	"runtime"
	"time"

	"riesgo-delictivo/internal/dataset"
	"riesgo-delictivo/internal/loader"
	"riesgo-delictivo/internal/metrics"
	"riesgo-delictivo/internal/ml"
)

func main() {
	rutaDatos := flag.String("datos", "../../../data/datos_limpios.csv", "CSV limpio de entrada")
	rutaModelo := flag.String("modelo", "../../../data/model.json", "salida del modelo entrenado (relativo al CWD)")
	workers := flag.Int("workers", runtime.NumCPU(), "número de goroutines")
	epocas := flag.Int("epocas", 300, "épocas de entrenamiento")
	tasa := flag.Float64("lr", 0.5, "tasa de aprendizaje")
	flag.Parse()

	fmt.Printf("=== CC65 PC3 | Riesgo delictivo por zona y hora (Buenos Aires) ===\n")
	fmt.Printf("workers: %d (CPUs: %d)\n\n", *workers, runtime.NumCPU())

	// ---- Etapa 1: carga concurrente ----
	t0 := time.Now()
	res, err := loader.CargarConcurrente(*rutaDatos, *workers)
	if err != nil {
		panic(err)
	}
	fmt.Printf("[carga    ] %d registros procesados, %d inválidos, %d celdas (barrio,hora,día) en %v\n",
		res.TotalLeidos, res.TotalInvalido, len(res.Conteos), time.Since(t0).Round(time.Millisecond))

	// ---- Etapa 2: dataset etiquetado ----
	ds := dataset.Construir(res, 42)
	train, test := ds.Dividir(0.2)
	fmt.Printf("[dataset  ] umbral P75 = %d delitos/celda | train: %d, test: %d ejemplos\n",
		ds.Umbral, len(train), len(test))

	// ---- Etapa 3: entrenamiento paralelo ----
	t1 := time.Now()
	modelo := ml.Entrenar(train, ds.NumFeats, ml.Config{
		Epocas:     *epocas,
		TasaAprend: *tasa,
		Workers:    *workers,
		L2:         1e-4,
		Callback: func(e int, costo float64) {
			if e%50 == 0 || e == *epocas-1 {
				fmt.Printf("[entrena  ] época %4d | costo (entropía cruzada) = %.4f\n", e, costo)
			}
		},
	})
	fmt.Printf("[entrena  ] %d épocas con %d workers en %v\n",
		*epocas, *workers, time.Since(t1).Round(time.Millisecond))

	// ---- Etapa 4: evaluación y exportación ----
	rep := metrics.Evaluar(modelo, test, 0.5)
	fmt.Printf("\n[métricas ] accuracy=%.3f precision=%.3f recall=%.3f f1=%.3f\n",
		rep.Accuracy, rep.Precision, rep.Recall, rep.F1)
	fmt.Printf("[confusión] TP=%d TN=%d FP=%d FN=%d\n", rep.TP, rep.TN, rep.FP, rep.FN)

	modelo.FeatNames = ds.FeatNames
	modelo.MaxBarrio = ds.MaxBarrio
	modelo.Umbral = ds.Umbral
	if err := modelo.Guardar(*rutaModelo); err != nil {
		panic(err)
	}
	fmt.Printf("[modelo   ] guardado en %s (lo consumirá la API en PC4)\n", *rutaModelo)

	// Demostración de predicción (lo que expondrá la API en PC4):
	ej := dataset.Features(22, 5, 10, 1, ds.MaxBarrio) // sábado 22h, barrio_id=10, comuna 1
	fmt.Printf("\n[ejemplo  ] P(alto riesgo | sábado 22h, barrio_id=10) = %.2f\n", modelo.Predecir(ej))
}
