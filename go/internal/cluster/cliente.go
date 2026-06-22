// Cliente TCP de un nodo compute
package cluster

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

const backoffReintento = 20 * time.Millisecond

// maxIntentosEnvio es el número máximo de intentos al enviar un mensaje
const maxIntentosEnvio = 3

type ClienteNodo struct {
	addr    string // e.g. "node1:9001"
	retries int    // máximo de reintentos al (re)conectar
	conn    net.Conn
	enc     *json.Encoder
	sc      *bufio.Scanner
}

func NewClienteNodo(addr string, retries int) (*ClienteNodo, error) {
	if retries < 1 {
		retries = 1
	}
	c := &ClienteNodo{addr: addr, retries: retries}
	if err := c.Conectar(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *ClienteNodo) Conectar() error {
	var lastErr error
	for i := 0; i < c.retries; i++ {
		conn, err := net.Dial("tcp", c.addr)
		if err != nil {
			lastErr = err
			if i < c.retries-1 {
				time.Sleep(backoffReintento)
			}
			continue
		}
		c.conn = conn
		c.enc = json.NewEncoder(conn)
		sc := bufio.NewScanner(conn)
		sc.Buffer(make([]byte, 4*1024*1024), 4*1024*1024) // 4MB
		c.sc = sc
		return nil
	}
	return fmt.Errorf("cliente: no se pudo conectar a %s tras %d intentos: %w", c.addr, c.retries, lastErr)
}

func (c *ClienteNodo) EnviarEntrenar(msg MensajeEntrenar) (*MensajeRespuesta, error) {
	var lastErr error
	for intento := 0; intento < maxIntentosEnvio; intento++ {
		if c.conn == nil || c.enc == nil {
			if err := c.Conectar(); err != nil {
				lastErr = err
				continue
			}
		}
		if err := c.enc.Encode(msg); err != nil { // Encode añade \n
			c.cerrarConexion()
			lastErr = err
			continue
		}
		if !c.sc.Scan() {
			err := c.sc.Err()
			if err == nil {
				err = fmt.Errorf("cliente: conexión cerrada por %s", c.addr)
			}
			c.cerrarConexion()
			lastErr = err
			continue
		}
		var resp MensajeRespuesta
		if err := json.Unmarshal(c.sc.Bytes(), &resp); err != nil {
			// Respuesta malformada: el nodo persiste en el error, reconectar no ayuda.
			c.cerrarConexion()
			return nil, fmt.Errorf("cliente: respuesta malformada de %s: %w", c.addr, err)
		}
		return &resp, nil
	}
	return nil, fmt.Errorf("cliente: fallo persistente contra %s tras %d intentos: %w", c.addr, maxIntentosEnvio, lastErr)
}

func (c *ClienteNodo) Cerrar() {
	c.cerrarConexion()
}

func (c *ClienteNodo) cerrarConexion() {
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
		c.enc = nil
		c.sc = nil
	}
}
