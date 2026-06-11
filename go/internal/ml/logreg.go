package ml

import (
	"encoding/json"
	"math"
	"os"
	"sync"

	"riesgo-delictivo/internal/dataset"
)

// LogReg es el modelo
type LogReg struct {
	Pesos     []float64 `json:"pesos"`
	FeatNames []string  `json:"feat_names"`
	MaxBarrio int       `json:"max_barrio"`
	Umbral    int       `json:"umbral_p75"`
}

// Config del entrenamiento paralelo.
type Config struct {
	Epocas     int
	TasaAprend float64
	Workers    int
	L2         float64
	Callback   func(epoca int, costo float64) // para loguear progreso
}

func sigmoide(z float64) float64 { return 1.0 / (1.0 + math.Exp(-z)) }

// Predecir devuelve P(alto_riesgo=1 | x).
func (m *LogReg) Predecir(x []float64) float64 {
	z := 0.0
	for i, w := range m.Pesos {
		z += w * x[i]
	}
	return sigmoide(z)
}

// gradParcial es la contribución de un shard: gradiente acumulado + costo.
type gradParcial struct {
	grad  []float64
	costo float64
	n     int
}

// Entrenar ejecuta descenso de gradiente síncrono con P workers.
func Entrenar(train []dataset.Ejemplo, numFeats int, cfg Config) *LogReg {
	m := &LogReg{Pesos: make([]float64, numFeats)}

	// Partición estática del dataset en shards (uno por worker).
	shards := make([][]dataset.Ejemplo, cfg.Workers)
	for i, ej := range train {
		w := i % cfg.Workers
		shards[w] = append(shards[w], ej)
	}

	for epoca := 0; epoca < cfg.Epocas; epoca++ {
		parciales := make(chan gradParcial, cfg.Workers)

		// Copia de solo lectura de los pesos para esta época.
		pesos := make([]float64, numFeats)
		copy(pesos, m.Pesos)

		// ---- Fan-out: cada worker calcula su gradiente parcial ----
		var wg sync.WaitGroup
		for w := 0; w < cfg.Workers; w++ {
			wg.Add(1)
			go func(shard []dataset.Ejemplo) {
				defer wg.Done()
				g := gradParcial{grad: make([]float64, numFeats), n: len(shard)}
				for _, ej := range shard {
					z := 0.0
					for i := range pesos {
						z += pesos[i] * ej.X[i]
					}
					h := sigmoide(z)
					err := h - ej.Y
					for i := range g.grad {
						g.grad[i] += err * ej.X[i]
					}
					// Entropía cruzada (con clipping numérico).
					hc := math.Min(math.Max(h, 1e-12), 1-1e-12)
					g.costo += -(ej.Y*math.Log(hc) + (1-ej.Y)*math.Log(1-hc))
				}
				parciales <- g
			}(shards[w])
		}
		go func() { wg.Wait(); close(parciales) }()

		// ---- Fan-in: el coordinador agrega y actualiza pesos ----
		total := gradParcial{grad: make([]float64, numFeats)}
		for p := range parciales {
			for i := range total.grad {
				total.grad[i] += p.grad[i]
			}
			total.costo += p.costo
			total.n += p.n
		}
		nf := float64(total.n)
		for i := range m.Pesos {
			g := total.grad[i]/nf + cfg.L2*m.Pesos[i]
			m.Pesos[i] -= cfg.TasaAprend * g
		}
		if cfg.Callback != nil {
			cfg.Callback(epoca, total.costo/nf)
		}
	}
	return m
}

// Guardar serializa el modelo a JSON (insumo de la API en PC4).
func (m *LogReg) Guardar(ruta string) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ruta, b, 0o644)
}

// Cargar lee un modelo previamente guardado (lo usará la API en PC4).
func Cargar(ruta string) (*LogReg, error) {
	b, err := os.ReadFile(ruta)
	if err != nil {
		return nil, err
	}
	var m LogReg
	return &m, json.Unmarshal(b, &m)
}
