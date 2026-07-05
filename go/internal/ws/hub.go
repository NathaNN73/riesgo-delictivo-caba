package ws

import "sync"

// Hub gestiona clientes WebSocket suscritos a actualizaciones de entrenamiento.
type Hub struct {
	mu       sync.RWMutex
	clientes map[chan []byte]struct{}
}

// NewHub crea un nuevo hub.
func NewHub() *Hub {
	return &Hub{clientes: make(map[chan []byte]struct{})}
}

// Suscribir registra un canal que recibirá mensajes de broadcast.
// Devuelve una función para cancelar la suscripción.
func (h *Hub) Suscribir() (ch chan []byte, cancelar func()) {
	ch = make(chan []byte, 16)
	h.mu.Lock()
	h.clientes[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.clientes, ch)
		close(ch)
		h.mu.Unlock()
	}
}

// Broadcast envía el mensaje a todos los clientes suscritos.
// Si un cliente tiene el buffer lleno, se saltea sin bloquear.
func (h *Hub) Broadcast(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clientes {
		select {
		case ch <- msg:
		default:
		}
	}
}
