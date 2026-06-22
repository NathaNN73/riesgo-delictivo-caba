// Uso:
//
//	go run ./cmd/node -port 9001
package main

import (
	"flag"
	"log"

	"riesgo-delictivo/internal/cluster"
)

func main() {
	puerto := flag.String("port", "9001", "puerto TCP donde escucha el nodo")
	flag.Parse()

	nodo := &cluster.Nodo{Puerto: *puerto}
	log.Printf("Nodo ML iniciado en puerto %s", *puerto)
	if err := nodo.Iniciar(); err != nil {
		log.Fatalf("Error iniciando nodo: %v", err)
	}
}
