package loader

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Registro representa una fila limpia del dataset.
type Registro struct {
	Anio      int
	Mes       int
	DiaSemana int
	Hora      int
	BarrioID  int
	Comuna    int
	Tipo      string
	UsoArma   int
	UsoMoto   int
	Lat, Lon  float64
}

// Celda identifica una combinacion zona-tiempo: la unidad de prediccion.
type Celda struct {
	BarrioID  int
	Hora      int
	DiaSemana int
}

// Resultado de la carga concurrente.
type Resultado struct {
	Conteos       map[Celda]int // delitos historicos por celda
	ComunaBarrio  map[int]int   // barrio_id -> comuna (para features)
	TotalLeidos   int           // filas procesadas
	TotalInvalido int           // filas descartadas por parseo
}

const tamanioBloque = 10000 // filas por bloque enviado a los workers

// resultadoParcial es lo que cada worker devuelve al agregador.
type resultadoParcial struct {
	conteos      map[Celda]int
	comunaBarrio map[int]int
	leidos       int
	invalidos    int
}

// CargarConcurrente lee el CSV en bloques y los procesa con numWorkers goroutines.
func CargarConcurrente(ruta string, numWorkers int) (*Resultado, error) {
	f, err := os.Open(ruta)
	if err != nil {
		return nil, fmt.Errorf("abriendo %s: %w", ruta, err)
	}git add .
	defer f.Close()

	bloques := make(chan []string, numWorkers*2)
	parciales := make(chan resultadoParcial, numWorkers)

	// ---- Fan-out: pool de workers ----
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resultado := resultadoParcial{conteos: map[Celda]int{}, comunaBarrio: map[int]int{}}
			for bloque := range bloques {
				for _, linea := range bloque {
					r, ok := procesarFila(linea)
					if !ok {
						resultado.invalidos++
						continue
					}
					resultado.leidos++
					resultado.conteos[Celda{r.BarrioID, r.Hora, r.DiaSemana}]++
					resultado.comunaBarrio[r.BarrioID] = r.Comuna
				}
			}
			parciales <- resultado // cada worker entrega UNA vez su mapa local
		}()
	}

	// ---- Productor: lee el archivo y publica bloques ----
	go func() {
		defer close(bloques)
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		scanner.Scan() // descarta cabecera
		bloque := make([]string, 0, tamanioBloque)
		for scanner.Scan() {
			bloque = append(bloque, scanner.Text())
			if len(bloque) == tamanioBloque {
				bloques <- bloque
				bloque = make([]string, 0, tamanioBloque)
			}
		}
		if len(bloque) > 0 {
			bloques <- bloque
		}
	}()

	go func() {
		wg.Wait()
		close(parciales)
	}()

	// ---- Fan-in: agregador ----
	res := &Resultado{Conteos: map[Celda]int{}, ComunaBarrio: map[int]int{}}
	for parcial := range parciales {
		for celda, cantidad := range parcial.conteos {
			res.Conteos[celda] += cantidad
		}
		for barrio, comuna := range parcial.comunaBarrio {
			res.ComunaBarrio[barrio] = comuna
		}
		res.TotalLeidos += parcial.leidos
		res.TotalInvalido += parcial.invalidos
	}
	return res, nil
}

func procesarFila(linea string) (Registro, bool) {
	campos := strings.Split(linea, ",")
	if len(campos) < 11 {
		return Registro{}, false
	}
	toInt := func(s string) (int, bool) {
		v, err := strconv.Atoi(strings.TrimSpace(s))
		return v, err == nil
	}
	var r Registro
	var ok bool
	if r.Anio, ok = toInt(campos[0]); !ok {
		return r, false
	}
	if r.Mes, ok = toInt(campos[1]); !ok {
		return r, false
	}
	if r.DiaSemana, ok = toInt(campos[2]); !ok {
		return r, false
	}
	if r.Hora, ok = toInt(campos[3]); !ok || r.Hora < 0 || r.Hora > 23 {
		return r, false
	}
	if r.BarrioID, ok = toInt(campos[4]); !ok {
		return r, false
	}
	if r.Comuna, ok = toInt(campos[5]); !ok {
		return r, false
	}
	r.Tipo = campos[6]
	r.UsoArma, _ = toInt(campos[7])
	r.UsoMoto, _ = toInt(campos[8])
	return r, true
}
