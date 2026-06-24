// API HTTP del sistema
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"riesgo-delictivo/internal/cluster"
	"riesgo-delictivo/internal/dataset"
	"riesgo-delictivo/internal/loader"
	"riesgo-delictivo/internal/metrics"
	"riesgo-delictivo/internal/ml"
	"riesgo-delictivo/internal/store"
)

const comunaDefault = 1

type servidor struct {
	mu     sync.RWMutex // protege el acceso a modelo
	modelo *ml.LogReg   // modelo en memoria

	mongo *store.MongoStore
	redis *store.RedisStore
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func cargarModelo(redis *store.RedisStore, mongo *store.MongoStore) *ml.LogReg {
	if redis != nil {
		if pesos, ok := redis.GetModelo(); ok && len(pesos) > 0 {
			log.Printf("[api] modelo cargado desde Redis (%d pesos)", len(pesos))
			return &ml.LogReg{Pesos: pesos}
		}
	}
	if mongo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if pesos, _, err := mongo.CargarUltimoModelo(ctx); err == nil && len(pesos) > 0 {
			log.Printf("[api] modelo cargado desde MongoDB (%d pesos)", len(pesos))
			return &ml.LogReg{Pesos: pesos}
		}
	}
	log.Println("[api] no hay modelo entrenado")
	return nil
}

func main() {
	node1 := env("NODE1_ADDR", "127.0.0.1:9001")
	node2 := env("NODE2_ADDR", "127.0.0.1:9002")
	mongoURI := env("MONGO_URI", "mongodb://127.0.0.1:27017")
	redisAddr := env("REDIS_ADDR", "127.0.0.1:6379")
	datosPath := env("DATOS_PATH", "../data/datos_limpios.csv")
	port := env("PORT", "8080")

	// MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	mongo, err := store.NewMongoStore(ctx, mongoURI)
	cancel()
	if err != nil {
		log.Printf("[api] MongoDB unavailable: %v (continuando sin persistencia)", err)
	}

	// Redis
	redis, err := store.NewRedisStore(redisAddr)
	if err != nil {
		log.Printf("[api] Redis error: %v", err)
	}

	srv := &servidor{mongo: mongo, redis: redis}

	// Carga de modelo
	srv.modelo = cargarModelo(redis, mongo)

	mux := http.NewServeMux()
	mux.HandleFunc("/salud", srv.salud)
	mux.HandleFunc("/predecir", srv.predecir)
	mux.HandleFunc("/entrenar", srv.entrenar)

	logParams := func(k, v string) { log.Printf("[api] %s=%s", k, v) }
	logParams("NODE1_ADDR", node1)
	logParams("NODE2_ADDR", node2)
	logParams("DATOS_PATH", datosPath)

	addr := ":" + port
	log.Printf("[api] escuchando en %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("[api] servidor: %v", err)
	}
}

// --- handlers ---

func (s *servidor) salud(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *servidor) predecir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "método no permitido"})
		return
	}

	hora, _ := strconv.Atoi(r.URL.Query().Get("hora"))
	barrioID, _ := strconv.Atoi(r.URL.Query().Get("barrio_id"))
	diaSemana, _ := strconv.Atoi(r.URL.Query().Get("dia_semana"))

	// Leer modelo en memoria
	s.mu.RLock()
	modelo := s.modelo
	s.mu.RUnlock()
	if modelo == nil || len(modelo.Pesos) == 0 {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "modelo no entrenado"})
		return
	}

	// Intenta recuperar prediccion desde Redis
	if s.redis != nil {
		if prob, ok := s.redis.GetPrediccion(hora, barrioID, diaSemana); ok {
			if s.mongo != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				_ = s.mongo.RegistrarPrediccion(ctx, hora, barrioID, diaSemana, prob, true)
				cancel()
			}
			writeJSON(w, http.StatusOK, map[string]any{"probabilidad": prob, "desde_cache": true})
			return
		}
	}

	// Vector de features.
	feats := dataset.Features(hora, diaSemana, barrioID, comunaDefault, modelo.MaxBarrio)

	// Predicción.
	prob := modelo.Predecir(feats)

	// Cachear en Redis.
	if s.redis != nil {
		_ = s.redis.SetPrediccion(hora, barrioID, diaSemana, prob)
	}

	// Registrar en MongoDB.
	if s.mongo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = s.mongo.RegistrarPrediccion(ctx, hora, barrioID, diaSemana, prob, false)
		cancel()
	}

	// Respuesta.
	writeJSON(w, http.StatusOK, map[string]any{"probabilidad": prob, "desde_cache": false})
}

func (s *servidor) entrenar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "método no permitido"})
		return
	}

	epocas, _ := strconv.Atoi(r.URL.Query().Get("epocas"))
	if epocas <= 0 {
		epocas = 300
	}
	tasa, _ := strconv.ParseFloat(r.URL.Query().Get("tasa"), 64)
	if tasa <= 0 {
		tasa = 0.5
	}
	const l2 = 1e-4
	const checkpointInterval = 5
	datosPath := env("DATOS_PATH", "../data/datos_limpios.csv")
	node1 := env("NODE1_ADDR", "127.0.0.1:9001")
	node2 := env("NODE2_ADDR", "127.0.0.1:9002")

	log.Printf("[entrenar] inicio: epocas=%d tasa=%v l2=%v datos=%s", epocas, tasa, l2, datosPath)

	// Carga del csv
	res, err := loader.CargarConcurrente(datosPath, runtime.NumCPU())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cargando CSV: " + err.Error()})
		return
	}
	log.Printf("[entrenar] CSV cargado: %d filas válidas, %d inválidas", res.TotalLeidos, res.TotalInvalido)

	// Construcción del dataset y división en train/test
	ds := dataset.Construir(res, 42)
	train, test := ds.Dividir(0.2)
	log.Printf("[entrenar] dataset: %d train, %d test, %d feats", len(train), len(test), ds.NumFeats)

	// Coordinador con 2 nodos
	coord, err := cluster.NewCoordinador(node1, node2, tasa, l2, checkpointInterval, s.mongo, s.redis)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "coordinador: " + err.Error()})
		return
	}
	defer coord.Shutdown()

	// Entrenamiento distribuido
	callback := func(epoca int, costo float64) {
		if epoca%50 == 0 || epoca == epocas-1 {
			log.Printf("[entrenar] época %d costo=%f", epoca, costo)
		}
	}
	if err := coord.EntrenarDistribuido(train, ds.NumFeats, epocas, callback); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "entrenamiento: " + err.Error()})
		return
	}

	// Metadatos del modelo
	coord.Modelo.FeatNames = ds.FeatNames
	coord.Modelo.MaxBarrio = ds.MaxBarrio
	coord.Modelo.Umbral = ds.Umbral

	// Publicar el modelo entrenado para /predecir
	s.mu.Lock()
	s.modelo = coord.Modelo
	s.mu.Unlock()

	// Evaluación
	rep := metrics.Evaluar(coord.Modelo, test, 0.5)
	log.Printf("[entrenar] metrics: acc=%.4f prec=%.4f rec=%.4f f1=%.4f", rep.Accuracy, rep.Precision, rep.Recall, rep.F1)

	// Respuesta
	writeJSON(w, http.StatusOK, map[string]any{
		"epocas":     epocas,
		"tasa":       tasa,
		"num_feats":  ds.NumFeats,
		"train_size": len(train),
		"test_size":  len(test),
		"accuracy":   rep.Accuracy,
		"precision":  rep.Precision,
		"recall":     rep.Recall,
		"f1":         rep.F1,
		"tp":         rep.TP,
		"tn":         rep.TN,
		"fp":         rep.FP,
		"fn":         rep.FN,
	})
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("[api] error codificando respuesta: %v", err)
	}
}
