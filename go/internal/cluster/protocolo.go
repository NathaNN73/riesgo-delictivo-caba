// Protocolo de comunicación TCP entre la API (coordinador) y los nodos.
package cluster

import "riesgo-delictivo/internal/dataset"

// MensajeEntrenar es lo que la API envía a un nodo en cada época.
type MensajeEntrenar struct {
	Pesos []float64         `json:"pesos"`
	Shard []dataset.Ejemplo `json:"shard"`
}

// MensajeRespuesta es lo que el nodo devuelve tras calcular el gradiente.
type MensajeRespuesta struct {
	Grad  []float64 `json:"grad"`
	Costo float64   `json:"costo"`
	N     int       `json:"n"`
}
