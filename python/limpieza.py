"""
CC65 - PC3 | Limpieza y preparación del dataset "Delitos" (Ciudad de Buenos Aires)
==================================================================================
Descarga los CSV anuales (2016-2023) del portal Buenos Aires Data, los consolida
y aplica el pipeline de limpieza descrito en la sección 2.1 del informe.

Salida: data/datos_limpios.csv  -> entrada del cargador concurrente en Go.

Uso:
    python limpieza.py                  # descarga + limpia todos los años
    python limpieza.py --anios 2022 2023  # solo algunos años (pruebas rápidas)
    python limpieza.py --sin-descarga   # usa los CSV ya presentes en data/raw/
"""

import argparse
import unicodedata
from pathlib import Path

import pandas as pd
import requests

BASE_URL = (
    "https://cdn.buenosaires.gob.ar/datosabiertos/datasets/"
    "ministerio-de-justicia-y-seguridad/delitos/delitos_{anio}.csv"
)
ANIOS = list(range(2016, 2024))  # 2016..2023 -> ~1.1M registros

RAW_DIR = Path(__file__).resolve().parent.parent / "data" / "raw"
OUT_PATH = Path(__file__).resolve().parent.parent / "data" / "datos_limpios.csv"

# Bounding box aproximado de la Ciudad Autónoma de Buenos Aires
LAT_MIN, LAT_MAX = -34.71, -34.52
LON_MIN, LON_MAX = -58.54, -58.33

COLS_CRITICAS = ["fecha", "franja", "tipo", "barrio", "comuna", "latitud", "longitud"]


def descargar(anios: list[int]) -> list[Path]:
    RAW_DIR.mkdir(parents=True, exist_ok=True)
    rutas = []
    for anio in anios:
        destino = RAW_DIR / f"delitos_{anio}.csv"
        if not destino.exists():
            url = BASE_URL.format(anio=anio)
            print(f"[descarga] {url}")
            r = requests.get(url, headers={"User-Agent": "Mozilla/5.0"}, timeout=300)
            r.raise_for_status()
            destino.write_bytes(r.content)
        rutas.append(destino)
    return rutas


def sin_tildes(texto: str) -> str:
    return "".join(
        c for c in unicodedata.normalize("NFD", texto) if unicodedata.category(c) != "Mn"
    )


def limpiar(rutas: list[Path]) -> pd.DataFrame:
    frames = []
    for ruta in rutas:
        df = pd.read_csv(ruta, dtype=str, keep_default_na=False, on_bad_lines="skip")
        df.columns = [c.strip().lower().replace("-", "_") for c in df.columns]
        frames.append(df)
        print(f"[lectura ] {ruta.name}: {len(df):,} filas")
    df = pd.concat(frames, ignore_index=True)
    total_inicial = len(df)
    print(f"[consolidado] {total_inicial:,} filas")

    # 1) Normalizar marcadores de nulo del portal: "NULL", "SD" (sin dato), vacío
    df = df.replace({"NULL": pd.NA, "null": pd.NA, "SD": pd.NA, "S/D": pd.NA, "": pd.NA})

    # 2) Eliminar filas con nulos en campos críticos
    df = df.dropna(subset=[c for c in COLS_CRITICAS if c in df.columns])

    # 3) Tipos numéricos + fecha
    df["franja"] = pd.to_numeric(df["franja"], errors="coerce")
    df["latitud"] = pd.to_numeric(df["latitud"].str.replace(",", ".", regex=False), errors="coerce")
    df["longitud"] = pd.to_numeric(df["longitud"].str.replace(",", ".", regex=False), errors="coerce")
    df["fecha_dt"] = pd.to_datetime(df["fecha"], errors="coerce", format="mixed")
    df = df.dropna(subset=["franja", "latitud", "longitud", "fecha_dt"])

    # 4) Variables temporales derivadas
    df["hora"] = df["franja"].astype(int)
    df = df[(df["hora"] >= 0) & (df["hora"] <= 23)]
    df["dia_semana"] = df["fecha_dt"].dt.dayofweek  # 0=lunes .. 6=domingo
    df["mes"] = df["fecha_dt"].dt.month
    df["anio"] = df["fecha_dt"].dt.year

    # 5) Outliers espaciales: (0,0) y fuera del bounding box de CABA
    df = df[
        df["latitud"].between(LAT_MIN, LAT_MAX) & df["longitud"].between(LON_MIN, LON_MAX)
    ]

    # 6) Normalización de textos y codificación de zona a enteros
    for col in ["tipo", "subtipo", "barrio"]:
        if col in df.columns:
            df[col] = df[col].map(lambda s: sin_tildes(str(s)).strip().upper())
    df["comuna"] = (
        df["comuna"].str.extract(r"(\d+)", expand=False).astype("Int64")
    )
    df = df.dropna(subset=["comuna"])
    df["barrio_id"] = df["barrio"].astype("category").cat.codes

    # Diccionario barrio_id -> barrio (lo usará la API/Frontend en PC4/TB2)
    dic = df[["barrio_id", "barrio", "comuna"]].drop_duplicates().sort_values("barrio_id")
    dic.to_csv(OUT_PATH.parent / "diccionario_barrios.csv", index=False)

    # 7) Flags binarias
    for col in ["uso_arma", "uso_moto"]:
        if col in df.columns:
            df[col] = (df[col].str.upper() == "SI").astype(int)
        else:
            df[col] = 0

    columnas_salida = [
        "anio", "mes", "dia_semana", "hora",
        "barrio_id", "comuna", "tipo",
        "uso_arma", "uso_moto", "latitud", "longitud",
    ]
    df_final = df[columnas_salida].reset_index(drop=True)
    print(f"[limpieza] {total_inicial:,} -> {len(df_final):,} filas "
          f"({total_inicial - len(df_final):,} descartadas, "
          f"{100 * len(df_final) / total_inicial:.1f}% retenido)")
    return df_final


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--anios", nargs="+", type=int, default=ANIOS)
    parser.add_argument("--sin-descarga", action="store_true")
    args = parser.parse_args()

    if args.sin_descarga:
        rutas = sorted(RAW_DIR.glob("delitos_*.csv"))
    else:
        rutas = descargar(args.anios)

    df = limpiar(rutas)
    OUT_PATH.parent.mkdir(parents=True, exist_ok=True)
    df.to_csv(OUT_PATH, index=False)
    print(f"[salida  ] {OUT_PATH} ({len(df):,} filas)")


if __name__ == "__main__":
    main()
