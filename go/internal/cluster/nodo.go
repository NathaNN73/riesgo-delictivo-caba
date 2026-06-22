package cluster

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"

	"riesgo-delictivo/internal/dataset"
)

type Nodo struct {
	Puerto string
}

func (n *Nodo) Iniciar() error {
	ln, err := net.Listen("tcp", ":"+n.Puerto)
	if err != nil {
		return fmt.Errorf("nodo: no se pudo escuchar en :%s — %w", n.Puerto, err)
	}
	log.Printf("[nodo] escuchando en :%s", n.Puerto)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[nodo] error aceptando conexión: %v", err)
			continue
		}
		go n.manejarConexion(conn)
	}
}

func (n *Nodo) manejarConexion(conn net.Conn) {
	defer conn.Close()
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 4*1024*1024), 4*1024*1024) //4MB

	for sc.Scan() {
		var msg MensajeEntrenar
		if err := json.Unmarshal(sc.Bytes(), &msg); err != nil {
			log.Printf("[nodo] error decodificando mensaje: %v", err)
			return
		}

		resp := calcularGradiente(msg.Pesos, msg.Shard)

		data, err := json.Marshal(resp)
		if err != nil {
			log.Printf("[nodo] error codificando respuesta: %v", err)
			return
		}
		data = append(data, '\n')
		if _, err := conn.Write(data); err != nil {
			log.Printf("[nodo] error escribiendo respuesta: %v", err)
			return
		}
	}

	if err := sc.Err(); err != nil {
		log.Printf("[nodo] error en el scanner de la conexión: %v", err)
	}
}

func calcularGradiente(pesos []float64, shard []dataset.Ejemplo) MensajeRespuesta {
	numFeats := len(pesos)
	grad := make([]float64, numFeats)
	var costo float64

	for _, ej := range shard {
		z := 0.0
		for i := range pesos {
			z += pesos[i] * ej.X[i]
		}
		h := 1.0 / (1.0 + math.Exp(-z)) // sigmoide
		err := h - ej.Y
		for i := range grad {
			grad[i] += err * ej.X[i]
		}
		// Entropía cruzada con clipping numérico
		hc := math.Min(math.Max(h, 1e-12), 1-1e-12)
		costo += -(ej.Y*math.Log(hc) + (1-ej.Y)*math.Log(1-hc))
	}

	return MensajeRespuesta{
		Grad:  grad,
		Costo: costo,
		N:     len(shard),
	}
}
