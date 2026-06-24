// Coordinador de entrenamiento distribuido
package cluster

import (
	"context"
	"fmt"
	"log"
	"sync"

	"riesgo-delictivo/internal/dataset"
	"riesgo-delictivo/internal/ml"
	"riesgo-delictivo/internal/store"
)

type Coordinador struct {
	Cliente1           *ClienteNodo
	Cliente2           *ClienteNodo
	Modelo             *ml.LogReg // Modelo en memoria
	TasaAprend         float64
	L2                 float64
	CheckpointInterval int // Checkpoint cada N epocas
	MongoStore         *store.MongoStore
	RedisStore         *store.RedisStore
}

// Crea coordinador con 2 nodos
func NewCoordinador(addr1, addr2 string, tasa, l2 float64, checkpointInterval int, mongo *store.MongoStore, redis *store.RedisStore) (*Coordinador, error) {
	c1, err := NewClienteNodo(addr1, 3)
	if err != nil {
		return nil, fmt.Errorf("coordinador: nodo1 %s inalcanzable: %w", addr1, err)
	}
	c2, err := NewClienteNodo(addr2, 3)
	if err != nil {
		c1.Cerrar()
		return nil, fmt.Errorf("coordinador: nodo2 %s inalcanzable: %w", addr2, err)
	}
	if checkpointInterval < 1 {
		checkpointInterval = 1
	}
	return &Coordinador{
		Cliente1:           c1,
		Cliente2:           c2,
		Modelo:             &ml.LogReg{},
		TasaAprend:         tasa,
		L2:                 l2,
		CheckpointInterval: checkpointInterval,
		MongoStore:         mongo,
		RedisStore:         redis,
	}, nil
}

// Entrenamiento distribuido
func (c *Coordinador) EntrenarDistribuido(train []dataset.Ejemplo, numFeats int, epocas int, callback func(int, float64)) error {
	m := c.Modelo
	if m.Pesos == nil {
		m.Pesos = make([]float64, numFeats)
	}

	shards := make([][]dataset.Ejemplo, 2)
	for i, ej := range train {
		shards[i%2] = append(shards[i%2], ej)
	}

	for epoca := 0; epoca < epocas; epoca++ {

		pesos := make([]float64, numFeats)
		copy(pesos, m.Pesos)

		var (
			resp1, resp2 *MensajeRespuesta
			err1, err2   error
			wg           sync.WaitGroup
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			resp1, err1 = c.Cliente1.EnviarEntrenar(MensajeEntrenar{Pesos: pesos, Shard: shards[0]})
		}()
		go func() {
			defer wg.Done()
			resp2, err2 = c.Cliente2.EnviarEntrenar(MensajeEntrenar{Pesos: pesos, Shard: shards[1]})
		}()
		wg.Wait()

		if err1 != nil {
			return fmt.Errorf("coordinador: época %d nodo1: %w", epoca, err1)
		}
		if err2 != nil {
			return fmt.Errorf("coordinador: época %d nodo2: %w", epoca, err2)
		}

		nTotal := resp1.N + resp2.N
		if nTotal == 0 {
			return fmt.Errorf("coordinador: época %d ambos shards vacíos", epoca)
		}
		totalGrad := make([]float64, numFeats)
		for i := 0; i < numFeats; i++ {
			totalGrad[i] = resp1.Grad[i] + resp2.Grad[i]
		}

		totalCosto := resp1.Costo + resp2.Costo
		nf := float64(nTotal)

		for i := range m.Pesos {
			g := totalGrad[i]/nf + c.L2*m.Pesos[i]
			m.Pesos[i] -= c.TasaAprend * g
		}

		// informacion de progreso
		if callback != nil {
			callback(epoca, totalCosto/nf)
		}

		// checkpoint
		esFinal := epoca == epocas-1
		hacerCheckpoint := esFinal || (c.CheckpointInterval > 0 && (epoca+1)%c.CheckpointInterval == 0)
		if hacerCheckpoint {
			costoProm := totalCosto / nf
			if c.MongoStore != nil {
				if err := c.MongoStore.GuardarCheckpoint(context.Background(), m.Pesos, epoca, costoProm, esFinal); err != nil {
					log.Printf("[coordinador] checkpoint mongo época %d: %v", epoca, err)
				}
			}
			if c.RedisStore != nil {
				if err := c.RedisStore.SetModelo(m.Pesos); err != nil {
					log.Printf("[coordinador] checkpoint redis época %d: %v", epoca, err)
				}
			}
		}
	}
	return nil
}

func (c *Coordinador) Shutdown() {
	if c.Cliente1 != nil {
		c.Cliente1.Cerrar()
	}
	if c.Cliente2 != nil {
		c.Cliente2.Cerrar()
	}
}

func (c *Coordinador) Predecir(x []float64) float64 {
	return c.Modelo.Predecir(x)
}
