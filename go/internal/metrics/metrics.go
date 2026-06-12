package metrics

import (
	"riesgo-delictivo/internal/dataset"
	"riesgo-delictivo/internal/ml"
)

// Reporte contiene las metricas de clasificacion binaria.
type Reporte struct {
	Accuracy  float64
	Precision float64
	Recall    float64
	F1        float64
	TP        int // verdaderos positivos
	TN        int // verdaderos negativos
	FP        int // falsos positivos
	FN        int // falsos negativos
}

// Evaluar clasifica cada ejemplo con el modelo y devuelve el reporte.
func Evaluar(modelo *ml.LogReg, ejemplos []dataset.Ejemplo, umbral float64) Reporte {
	r := Reporte{}

	for _, ej := range ejemplos {
		// Clasificar: 1 si P(alto_riesgo) >= umbral, 0 si no
		prob := modelo.Predecir(ej.X)
		predicho := 0.0
		if prob >= umbral {
			predicho = 1.0
		}

		// Actualizar matriz de confusion
		switch {
		case predicho == 1.0 && ej.Y == 1.0:
			r.TP++
		case predicho == 0.0 && ej.Y == 0.0:
			r.TN++
		case predicho == 1.0 && ej.Y == 0.0:
			r.FP++
		case predicho == 0.0 && ej.Y == 1.0:
			r.FN++
		}
	}

	// Calcular metricas derivadas
	total := float64(r.TP + r.TN + r.FP + r.FN)
	if total == 0 {
		return r
	}

	r.Accuracy = float64(r.TP+r.TN) / total

	if r.TP+r.FP > 0 {
		r.Precision = float64(r.TP) / float64(r.TP+r.FP)
	}
	if r.TP+r.FN > 0 {
		r.Recall = float64(r.TP) / float64(r.TP+r.FN)
	}
	if r.Precision+r.Recall > 0 {
		r.F1 = 2.0 * r.Precision * r.Recall / (r.Precision + r.Recall)
	}

	return r
}