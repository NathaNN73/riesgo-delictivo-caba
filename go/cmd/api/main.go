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
	"strings"
	"sync"
	"time"

	"riesgo-delictivo/internal/auth"
	"riesgo-delictivo/internal/cluster"
	"riesgo-delictivo/internal/dataset"
	"riesgo-delictivo/internal/loader"
	"riesgo-delictivo/internal/metrics"
	"riesgo-delictivo/internal/ml"
	"riesgo-delictivo/internal/store"
)

type servidor struct {
	mu           sync.RWMutex
	modelo       *ml.LogReg
	comunaBarrio map[int]int
	entrenando   bool

	jwtSecret []byte
	tokenTTL  time.Duration

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
	jwtSecret := []byte(env("JWT_SECRET", "x027Y5qNzxTUfV1vlCZVX5P0oTIsdbsuGeCqYaEnQ7F"))
	tokenTTL := 24 * time.Hour

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

	srv := &servidor{
		mongo:     mongo,
		redis:     redis,
		jwtSecret: jwtSecret,
		tokenTTL:  tokenTTL,
	}

	// Carga de modelo
	srv.modelo = cargarModelo(redis, mongo)

	mux := http.NewServeMux()
	mux.HandleFunc("/salud", srv.salud)
	mux.HandleFunc("/predecir", srv.predecir)
	mux.HandleFunc("/registro", srv.registro)
	mux.HandleFunc("/login", srv.login)
	mux.HandleFunc("/entrenar", srv.autenticar(srv.entrenar))

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

// --- auth ---

func (s *servidor) autenticar(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "token requerido"})
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if _, err := auth.ValidateToken(token, s.jwtSecret); err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "token inválido o expirado"})
			return
		}
		next(w, r)
	}
}

func (s *servidor) registro(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "método no permitido"})
		return
	}
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" || body.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email y password requeridos"})
		return
	}
	if s.mongo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "MongoDB no disponible"})
		return
	}
	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "error al procesar contraseña"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := s.mongo.RegistrarUsuario(ctx, body.Email, hash); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"mensaje": "usuario registrado"})
}

func (s *servidor) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "método no permitido"})
		return
	}
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" || body.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email y password requeridos"})
		return
	}
	if s.mongo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "MongoDB no disponible"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	u, err := s.mongo.BuscarUsuario(ctx, body.Email)
	if err != nil || u == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "credenciales inválidas"})
		return
	}
	if !auth.CheckPassword(body.Password, u.Password) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "credenciales inválidas"})
		return
	}
	token, err := auth.GenerateToken(u.Email, s.jwtSecret, s.tokenTTL)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "error al generar token"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (s *servidor) predecir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "método no permitido"})
		return
	}

	hora, _ := strconv.Atoi(r.URL.Query().Get("hora"))
	barrioID, _ := strconv.Atoi(r.URL.Query().Get("barrio_id"))
	diaSemana, _ := strconv.Atoi(r.URL.Query().Get("dia_semana"))

	if hora < 0 || hora > 23 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hora debe estar entre 0 y 23"})
		return
	}
	if barrioID < 0 || barrioID > 47 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "barrio_id debe estar entre 0 y 47"})
		return
	}
	if diaSemana < 0 || diaSemana > 6 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "dia_semana debe estar entre 0 y 6"})
		return
	}

	umbral, _ := strconv.ParseFloat(env("UMBRAL_DEFAULT", "0.35"), 64)
	if umbral <= 0 {
		umbral = 0.35
	}

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
			alto := 0
			if prob >= umbral {
				alto = 1
			}
			if s.mongo != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				_ = s.mongo.RegistrarPrediccion(ctx, hora, barrioID, diaSemana, prob, true)
				cancel()
			}
			writeJSON(w, http.StatusOK, map[string]any{"probabilidad": prob, "alto_riesgo": alto, "desde_cache": true})
			return
		}
	}

	// Vector de features.
	comuna := s.comunaBarrio[barrioID]
	if comuna == 0 {
		comuna = 1
	}
	feats := dataset.Features(hora, diaSemana, barrioID, comuna, modelo.MaxBarrio)

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
	alto := 0
	if prob >= umbral {
		alto = 1
	}
	writeJSON(w, http.StatusOK, map[string]any{"probabilidad": prob, "alto_riesgo": alto, "desde_cache": false})
}

func (s *servidor) entrenar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "método no permitido"})
		return
	}

	s.mu.Lock()
	if s.entrenando {
		s.mu.Unlock()
		writeJSON(w, http.StatusConflict, map[string]string{"error": "entrenamiento en curso"})
		return
	}
	s.entrenando = true
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.entrenando = false; s.mu.Unlock() }()

	epocas, _ := strconv.Atoi(r.URL.Query().Get("epocas"))
	if epocas <= 0 {
		epocas, _ = strconv.Atoi(env("EPOCAS_DEFAULT", "300"))
	}
	if epocas <= 0 {
		epocas = 300
	}
	tasa, _ := strconv.ParseFloat(r.URL.Query().Get("tasa"), 64)
	if tasa <= 0 {
		tasa, _ = strconv.ParseFloat(env("TASA_DEFAULT", "0.5"), 64)
	}
	if tasa <= 0 {
		tasa = 0.5
	}
	umbral, _ := strconv.ParseFloat(r.URL.Query().Get("umbral"), 64)
	if umbral <= 0 {
		umbral, _ = strconv.ParseFloat(env("UMBRAL_DEFAULT", "0.35"), 64)
	}
	if umbral <= 0 {
		umbral = 0.35
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

	// Guardar mapeo barrio→comuna para /predecir
	s.mu.Lock()
	s.comunaBarrio = res.ComunaBarrio
	s.mu.Unlock()

	// Construcción del dataset y división en train/test
	ds := dataset.Construir(res, 42)
	train, test := ds.Dividir(0.2)
	log.Printf("[entrenar] dataset: %d train, %d test, %d feats", len(train), len(test), ds.NumFeats)

	// Exportar celdas etiquetadas para documentación
	if err := ds.ExportarCSV("/data/dataset_celdas.csv", res); err != nil {
		log.Printf("[entrenar] exportando celdas: %v", err)
	} else {
		log.Printf("[entrenar] celdas exportadas a /data/dataset_celdas.csv")
	}

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
	rep := metrics.Evaluar(coord.Modelo, test, umbral)
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
