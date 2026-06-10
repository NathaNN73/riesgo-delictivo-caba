# Sistema concurrente de predicción de riesgo delictivo por zona y franja horaria (CABA)

CC65 – Programación Concurrente y Distribuida | Entregable 1 (PC3)

Predice el nivel de riesgo delictivo de cada **barrio** de la Ciudad de Buenos Aires
para cada **hora del día** y **día de la semana**, a partir del dataset abierto
"Delitos" (Ministerio de Justicia y Seguridad, https://data.buenosaires.gob.ar/dataset/delitos),
con más de 1,1 millones de registros (2016–2023).

## Estructura (preparada para PC4 y TB2)

```
python/limpieza.py        Limpieza de datos (PC3, no concurrente por diseño)
go/internal/loader        Carga concurrente del CSV (fan-out/fan-in con channels)
go/internal/dataset       Celdas (barrio, hora, día), target P75 y features
go/internal/ml            Regresión logística con entrenamiento paralelo
go/internal/metrics       Accuracy, precision, recall, F1
go/cmd/trainer            Entrada del PC3 (carga + entrenamiento + model.json)
go/cmd/api                PC4: API REST/WS coordinadora del clúster (placeholder)
go/cmd/node               PC4: nodo de cómputo ML por TCP (placeholder)
docker-compose.yml        Despliegue (trainer hoy; api/nodos/mongo/redis/UI en PC4-TB2)
```

## Uso

```bash
# 1. Limpieza (descarga 2016–2023 y genera data/datos_limpios.csv)
cd python && pip install -r requirements.txt && python limpieza.py

# 2. Entrenamiento concurrente
cd ../go && go run ./cmd/trainer -datos ../data/datos_limpios.csv -epocas 300

# Verificación de ausencia de condiciones de carrera
go run -race ./cmd/trainer -datos ../data/datos_limpios.csv -epocas 60

# Con Docker
docker compose run --rm trainer
```

## Diseño concurrente (resumen)

1. **Carga**: un productor lee bloques de 10 000 filas → channel → pool de
   workers (uno por CPU) parsea y cuenta delitos por celda en mapas locales →
   channel → agregador fusiona (fan-out/fan-in, sin memoria compartida).
2. **Entrenamiento**: descenso de gradiente síncrono con paralelismo de datos;
   cada worker calcula el gradiente parcial de su shard y el coordinador agrega
   y actualiza pesos en un único punto (sin locks, verificado con `-race`).
3. En PC4, los workers de entrenamiento pasan a ser **nodos remotos por TCP** y
   `model.json` se convierte en el insumo de la **API de predicciones** (<100 ms).

## Git Flow

Ramas: `main` (releases), `develop`, `feature/*`. Ver historial de commits como
evidencia en el informe.
