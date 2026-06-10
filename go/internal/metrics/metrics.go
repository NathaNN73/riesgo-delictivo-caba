// Package metrics calcula métricas de clasificación binaria.
package metrics

import (
	"riesgo-delictivo/internal/dataset"
	"riesgo-delictivo/internal/ml"
)

type Reporte struct {
	Accuracy, Precision, Recall, F1 float64
	TP, TN, FP, FN                  int
}

// Evaluar aplica el modelo sobre un conjunto y devuelve el reporte.
func Evaluar(m *ml.LogReg, ejemplos []dataset.Ejemplo, corte float64) Reporte {
	var r Reporte
	for _, ej := range ejemplos {
		pred := 0.0
		if m.Predecir(ej.X) >= corte {
			pred = 1.0
		}
		switch {
		case pred == 1 && ej.Y == 1:
			r.TP++
		case pred == 0 && ej.Y == 0:
			r.TN++
		case pred == 1 && ej.Y == 0:
			r.FP++
		default:
			r.FN++
		}
	}
	total := float64(r.TP + r.TN + r.FP + r.FN)
	if total > 0 {
		r.Accuracy = float64(r.TP+r.TN) / total
	}
	if r.TP+r.FP > 0 {
		r.Precision = float64(r.TP) / float64(r.TP+r.FP)
	}
	if r.TP+r.FN > 0 {
		r.Recall = float64(r.TP) / float64(r.TP+r.FN)
	}
	if r.Precision+r.Recall > 0 {
		r.F1 = 2 * r.Precision * r.Recall / (r.Precision + r.Recall)
	}
	return r
}
