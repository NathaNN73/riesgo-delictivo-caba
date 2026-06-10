# Prediccion de riesgo delictivo por zona y franja horaria (CABA)

Predice el nivel de riesgo delictivo de cada **barrio** de la Ciudad de Buenos Aires
para cada **hora del dia** y **dia de la semana**, usando el dataset abierto
"Delitos" (Ministerio de Justicia y Seguridad) con mas de 1 millon de registros (2016-2023).

## Estructura

```
python/limpieza.py          Limpieza y preparacion del dataset
go/internal/loader          Carga concurrente del CSV (fan-out/fan-in)
go/internal/dataset         Features y etiquetado (percentil 75)
go/internal/ml              Regresion logistica con entrenamiento paralelo
go/internal/metrics         Accuracy, precision, recall, F1
go/cmd/trainer              Entrenamiento y exportacion del modelo
```

## Uso

```bash
# 1. Limpieza (descarga los CSV y genera data/datos_limpios.csv)
cd python && pip install -r requirements.txt && python limpieza.py

# 2. Entrenamiento
cd ../go && go build -o trainer.exe ./cmd/trainer
./trainer.exe -datos ../data/datos_limpios.csv -epocas 300

# 3. Verificar ausencia de condiciones de carrera
go run -race ./cmd/trainer -datos ../data/datos_limpios.csv -epocas 60

# 4. Benchmark de speedup
./benchmark.ps1 -Epochs 10000
python ../python/graficar.py
```

## Diseño concurrente

- **Carga**: productor lee bloques de 10 000 filas, N workers parsean en paralelo
  y cuentan delitos por celda en mapas locales, agregador fusiona. Sin memoria compartida.
- **Entrenamiento**: descenso de gradiente sincrono con paralelismo de datos.
  Cada worker calcula el gradiente parcial de su shard, el coordinador agrega y
  actualiza pesos. Verificado con `-race`.

## Resultados del benchmark

Con 10 000 epocas sobre ~1M de registros en una maquina de 8 nucleos fisicos (16 logicos):

| Workers | Tiempo (ms) | SpeedUp |
|---------|-------------|---------|
| 1       | 4108        | 1.00x   |
| 4       | 1430        | 2.87x   |
| 8       | 1220        | 3.37x   |
| 16      | 1122        | 3.66x   |
| 24      | 1096        | 3.75x   |
| 32      | 1115        | 3.69x   |
| 64      | 1158        | 3.55x   |
| 128     | 1272        | 3.23x   |

SpeedUp maximo: **3.75x** con 24 workers. Con mas de 24 workers el rendimiento
se degrada por sobresuscripcion de goroutines.
