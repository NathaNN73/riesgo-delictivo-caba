# Centinela — Prediccion de riesgo delictivo en CABA

Predice el nivel de riesgo delictivo de cada **barrio** de la Ciudad de Buenos Aires
para cada **hora del dia** y **dia de la semana**, usando el dataset abierto
"Delitos" (Ministerio de Justicia y Seguridad) con mas de 1 millon de registros (2016-2023).

## Estructura

```
python/limpieza.py            Limpieza y preparacion del dataset
go/internal/loader            Carga concurrente del CSV (fan-out/fan-in)
go/internal/dataset           Features y etiquetado (percentil 75)
go/internal/ml                Regresion logistica con entrenamiento paralelo
go/internal/metrics           Accuracy, precision, recall, F1
go/cmd/trainer                Entrenamiento local
go/cmd/api                    API HTTP: predicciones, entrenamiento distribuido, JWT
go/cmd/node                   Nodo de computo TCP para cluster distribuido
go/internal/cluster           Protocolo TCP, cliente, coordinador distribuido
go/internal/store             MongoDB (persistencia) y Redis (cache RESP)
go/internal/auth              Autenticacion JWT con bcrypt
go/internal/ws                WebSocket para monitoreo en tiempo real
frontend/                     SPA React con mapa interactivo (MapLibre GL)
docker-compose.yml            6 servicios: api, node1, node2, mongo, redis, frontend
```

## Uso

```bash
# 1. Limpieza (descarga los CSV y genera data/datos_limpios.csv)
cd python && pip install -r requirements.txt && python limpieza.py

# 2. Entrenamiento local
cd ../go && go build -o trainer.exe ./cmd/trainer
./trainer.exe -datos ../data/datos_limpios.csv -epocas 300

# 3. Benchmark de speedup
./benchmark.ps1 -Epochs 10000
python ../python/graficar.py

# 4. Desplegar con Docker (todos los servicios)
docker compose up --build

# 5. Endpoints disponibles
curl http://localhost:8080/salud
curl "http://localhost:8080/predecir?hora=22&barrio_id=10&dia_semana=5"
curl -X POST http://localhost:8080/registro -H "Content-Type: application/json" -d '{"email":"admin@test.com","password":"123456"}'
curl -X POST http://localhost:8080/login -H "Content-Type: application/json" -d '{"email":"admin@test.com","password":"123456"}'
curl -X POST "http://localhost:8080/entrenar?epocas=100" -H "Authorization: Bearer TOKEN"
curl http://localhost:8080/metricas -H "Authorization: Bearer TOKEN"
curl "http://localhost:8080/predicciones?hora=22&dia_semana=5"

# 6. Frontend
# Abrir http://localhost:4200 en el navegador
```

## Diseno concurrente y distribuido

- **Carga**: productor lee bloques de 10 000 filas, N workers parsean en paralelo
  y cuentan delitos por celda en mapas locales, agregador fusiona. Sin memoria compartida.
- **Entrenamiento local**: descenso de gradiente sincrono con paralelismo de datos.
  Cada worker calcula el gradiente parcial de su shard, el coordinador agrega y
  actualiza pesos. Verificado con `go run -race`.
- **Entrenamiento distribuido**: topologia estrella con 2 nodos TCP. El coordinador (API)
  reparte shards a los nodos, recolecta gradientes y actualiza pesos. Checkpoints cada
  5 epocas en MongoDB y Redis.
- **API HTTP**: endpoints REST para prediccion (<100ms), entrenamiento, metricas del
  cluster, autenticacion JWT y WebSocket para monitoreo en tiempo real.
- **Frontend**: SPA en React con mapa de CABA coloreado por nivel de riesgo (bajo/medio/alto).
  48 barrios con prediccion en tiempo real y slider horario.

## Resultados del benchmark

Con 10 000 epocas sobre ~1M de registros en una maquina de 8 nucleos fisicos (16 logicos):

| Workers | Tiempo (ms) | SpeedUp |
|---------|-------------|---------|
| 1       | 4108        | 1.00x   |
| 2       | 2441        | 1.68x   |
| 4       | 1430        | 2.87x   |
| 8       | 1220        | 3.37x   |
| 12      | 1179        | 3.48x   |
| 16      | 1122        | 3.66x   |
| 24      | 1096        | 3.75x   |
| 32      | 1115        | 3.69x   |
| 48      | 1160        | 3.54x   |
| 64      | 1158        | 3.55x   |
| 96      | 1184        | 3.47x   |
| 128     | 1272        | 3.23x   |

SpeedUp maximo: **3.75x** con 24 workers. Con mas de 24 workers el rendimiento
se degrada por sobresuscripcion de goroutines.

## Evaluacion experimental

| Metrica | Valor |
|---------|-------|
| Latencia p95 /predecir | 2 ms |
| Latencia /predicciones (48 barrios) | 0.2 ms |
| Throughput | 668 qps |
| Entrenamiento local (10000 epocas) | ~4 s |
| Entrenamiento distribuido (10000 epocas) | ~87 s |

## Tecnologias

Go 1.22 · Python 3.13 · React · MapLibre GL · MongoDB · Redis · Docker · JWT · WebSocket
