// Package loader implementa la carga y procesamiento concurrente del dataset
// limpio (datos_limpios.csv) usando el patrón fan-out / fan-in:
//
//	productor (lee bloques) ──> chan bloques ──> N workers (parsean + cuentan)
//	                                                  │
//	                            chan parciales <──────┘
//	                                  │
//	                              agregador (fusiona conteos por celda)
//
// La comunicación es exclusivamente por channels: no hay memoria compartida
// mutable entre workers, por lo que no existen condiciones de carrera.
package loader

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Registro es una fila limpia del dataset.
// Esquema: anio,mes,dia_semana,hora,barrio_id,comuna,tipo,uso_arma,uso_moto,latitud,longitud
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

// Celda identifica una combinación zona-tiempo: la unidad de predicción.
type Celda struct {
	BarrioID  int
	Hora      int
	DiaSemana int
}

// Resultado de la carga concurrente.
type Resultado struct {
	Conteos       map[Celda]int   // delitos históricos por celda
	ComunaBarrio  map[int]int     // barrio_id -> comuna (para features)
	TotalLeidos   int             // filas procesadas
	TotalInvalido int             // filas descartadas por parseo
}

const tamBloque = 10000 // filas por bloque enviado a los workers

// parcial es lo que cada worker devuelve al agregador.
type parcial struct {
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
	}
	defer f.Close()

	bloques := make(chan []string, numWorkers*2)
	parciales := make(chan parcial, numWorkers)

	// ---- Fan-out: pool de workers ----
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p := parcial{conteos: map[Celda]int{}, comunaBarrio: map[int]int{}}
			for bloque := range bloques {
				for _, linea := range bloque {
					r, ok := parsearLinea(linea)
					if !ok {
						p.invalidos++
						continue
					}
					p.leidos++
					p.conteos[Celda{r.BarrioID, r.Hora, r.DiaSemana}]++
					p.comunaBarrio[r.BarrioID] = r.Comuna
				}
			}
			parciales <- p // cada worker entrega UNA vez su mapa local
		}()
	}

	// ---- Productor: lee el archivo y publica bloques ----
	go func() {
		defer close(bloques)
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1024*1024), 1024*1024)
		sc.Scan() // descarta cabecera
		bloque := make([]string, 0, tamBloque)
		for sc.Scan() {
			bloque = append(bloque, sc.Text())
			if len(bloque) == tamBloque {
				bloques <- bloque
				bloque = make([]string, 0, tamBloque)
			}
		}
		if len(bloque) > 0 {
			bloques <- bloque
		}
	}()

	// Cierra el canal de parciales cuando todos los workers terminan.
	go func() {
		wg.Wait()
		close(parciales)
	}()

	// ---- Fan-in: agregador ----
	res := &Resultado{Conteos: map[Celda]int{}, ComunaBarrio: map[int]int{}}
	for p := range parciales {
		for c, n := range p.conteos {
			res.Conteos[c] += n
		}
		for b, com := range p.comunaBarrio {
			res.ComunaBarrio[b] = com
		}
		res.TotalLeidos += p.leidos
		res.TotalInvalido += p.invalidos
	}
	return res, nil
}

func parsearLinea(linea string) (Registro, bool) {
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
