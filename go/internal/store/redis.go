// Cache Redis vía protocolo RESP sobre TCP puro (sin librería externa).
package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
)

const ttlPrediccion = 3600

const claveModelo = "modelo:latest"

type RedisStore struct {
	addr      string
	conn      net.Conn
	rd        *bufio.Reader
	available bool // false si la conexión falló o cayó
}

func NewRedisStore(addr string) (*RedisStore, error) {
	s := &RedisStore{addr: addr}
	if err := s.conectar(); err != nil {
		log.Printf("[redis] unavailable en %s: %v (degradando a no-cache)", addr, err)
		return s, nil
	}
	return s, nil
}

func (s *RedisStore) conectar() error {
	conn, err := net.DialTimeout("tcp", s.addr, 2*time.Second)
	if err != nil {
		s.available = false
		return err
	}
	s.conn = conn
	s.rd = bufio.NewReader(conn)
	s.available = true
	return nil
}

func (s *RedisStore) Close() error {
	s.available = false
	if s.conn != nil {
		err := s.conn.Close()
		s.conn = nil
		s.rd = nil
		return err
	}
	return nil
}

// Disponible indica si Redis está accesible.
func (s *RedisStore) Disponible() bool { return s.available }

// GetPrediccion recupera la probabilidad cacheada para la celda (hora,barrioID,diaSemana).
func (s *RedisStore) GetPrediccion(hora, barrioID, diaSemana int) (float64, bool) {
	key := clavePrediccion(hora, barrioID, diaSemana)
	v, ok := s.getString(key)
	if !ok || v == "" {
		return 0, false
	}
	prob, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, false
	}
	return prob, true
}

// SetPrediccion cachea la probabilidad para la celda (hora,barrioID,diaSemana) 
func (s *RedisStore) SetPrediccion(hora, barrioID, diaSemana int, prob float64) error {
	if !s.available {
		return nil
	}
	key := clavePrediccion(hora, barrioID, diaSemana)
	val := strconv.FormatFloat(prob, 'f', -1, 64)
	return s.cmdSimple("SETEX", key, strconv.Itoa(ttlPrediccion), val)
}

// GetModelo recupera los pesos del modelo cacheado bajo "modelo:latest".
func (s *RedisStore) GetModelo() ([]float64, bool) {
	v, ok := s.getString(claveModelo)
	if !ok {
		return nil, false
	}
	var pesos []float64
	if err := json.Unmarshal([]byte(v), &pesos); err != nil {
		return nil, false
	}
	return pesos, true
}

// SetModelo cachea los pesos del modelo como JSON bajo "modelo:latest".
func (s *RedisStore) SetModelo(pesos []float64) error {
	if !s.available {
		return nil
	}
	data, err := json.Marshal(pesos)
	if err != nil {
		return err
	}
	return s.cmdSimple("SET", claveModelo, string(data))
}

// --- helpers RESP ---

// clavePrediccion genera "pred:{hora}:{barrioID}:{diaSemana}".
func clavePrediccion(hora, barrioID, diaSemana int) string {
	return fmt.Sprintf("pred:%d:%d:%d", hora, barrioID, diaSemana)
}

// getString ejecuta GET y devuelve (valor, true) si existe, ("", false) si nil/miss o error.
func (s *RedisStore) getString(key string) (string, bool) {
	if !s.available {
		return "", false
	}
	if err := s.writeArray("GET", key); err != nil {
		s.degradar()
		return "", false
	}
	resp, err := s.readReply()
	if err != nil {
		s.degradar()
		return "", false
	}
	r, ok := resp.(bulkString)
	if !ok || r.nil {
		return "", false
	}
	return r.value, true
}

// cmdSimple envía un comando cuya respuesta esperada es +OK (SET/SETEX).
func (s *RedisStore) cmdSimple(args ...string) error {
	if err := s.writeArray(args...); err != nil {
		s.degradar()
		return nil 
	}
	resp, err := s.readReply()
	if err != nil {
		s.degradar()
		return nil
	}
	if _, ok := resp.(simpleString); ok {
		return nil
	}
	if _, ok := resp.(errorReply); ok {
		s.degradar()
		return nil
	}
	return nil
}

func (s *RedisStore) degradar() {
	s.available = false
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
		s.rd = nil
	}
}

// writeArray escribe un comando RESP: *N\r\n$len\r\narg\r\n ...
func (s *RedisStore) writeArray(args ...string) error {
	if s.conn == nil {
		return fmt.Errorf("conexión cerrada")
	}
	var b []byte
	b = append(b, '*')
	b = strconv.AppendInt(b, int64(len(args)), 10)
	b = append(b, '\r', '\n')
	for _, a := range args {
		b = append(b, '$')
		b = strconv.AppendInt(b, int64(len(a)), 10)
		b = append(b, '\r', '\n')
		b = append(b, a...)
		b = append(b, '\r', '\n')
	}
	_, err := s.conn.Write(b)
	return err
}

// tipos de respuesta RESP.
type simpleString struct{ value string }
type errorReply struct{ value string }
type integerReply struct{ value int64 }
type bulkString struct {
	value string
	nil   bool
}

// readReply parsea una única respuesta RESP del lector.
func (s *RedisStore) readReply() (interface{}, error) {
	line, err := s.rd.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = trimCRLF(line)
	if len(line) == 0 {
		return nil, fmt.Errorf("respuesta vacía")
	}
	switch line[0] {
	case '+': // simple string
		return simpleString{value: line[1:]}, nil
	case '-': // error
		return errorReply{value: line[1:]}, nil
	case ':': // integer
		n, err := strconv.ParseInt(line[1:], 10, 64)
		if err != nil {
			return nil, err
		}
		return integerReply{value: n}, nil
	case '$': // bulk string
		n, err := strconv.Atoi(line[1:])
		if err != nil {
			return nil, err
		}
		if n < 0 {
			return bulkString{nil: true}, nil // $-1\r\n
		}
		buf := make([]byte, n)
		if _, err := readFull(s.rd, buf); err != nil {
			return nil, err
		}
		// descartar CRLF final
		if _, err := s.rd.Discard(2); err != nil {
			return nil, err
		}
		return bulkString{value: string(buf)}, nil
	default:
		return nil, fmt.Errorf("tipo RESP no soportado: %q", line)
	}
}

func readFull(rd *bufio.Reader, buf []byte) (int, error) {
	n := 0
	for n < len(buf) {
		k, err := rd.Read(buf[n:])
		n += k
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func trimCRLF(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
