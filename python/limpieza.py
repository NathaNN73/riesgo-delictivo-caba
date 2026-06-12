"""
Output: data/datos_limpios.csv

Usage:
    python limpieza.py                  # descarga + limpieza de todos los años
    python limpieza.py --years 2022 2023  # descarga + especificos años
    python limpieza.py --no-download    # no descarga
"""

import argparse
import unicodedata
from pathlib import Path

import pandas as pd
import requests

BASE_URL = (
    "https://cdn.buenosaires.gob.ar/datosabiertos/datasets/"
    "ministerio-de-justicia-y-seguridad/delitos/delitos_{year}.csv"
)
YEARS = list(range(2016, 2024))  # 2016..2023

RAW_DIR = Path(__file__).resolve().parent.parent / "data" / "raw"
OUT_PATH = Path(__file__).resolve().parent.parent / "data" / "datos_limpios.csv"

# CABA bounding box
LAT_MIN, LAT_MAX = -34.71, -34.52
LON_MIN, LON_MAX = -58.54, -58.33

CRITICAL_COLS = ["fecha", "franja", "tipo", "barrio", "comuna", "latitud", "longitud"]
OUTPUT_COLS = [
    "anio", "mes", "dia_semana", "hora",
    "barrio_id", "comuna", "tipo",
    "uso_arma", "uso_moto", "latitud", "longitud",
]


# Descarga
def download_years(years: list[int]) -> list[Path]:
    RAW_DIR.mkdir(parents=True, exist_ok=True)
    paths = []
    for year in years:
        dest = RAW_DIR / f"delitos_{year}.csv"
        if not dest.exists():
            url = BASE_URL.format(year=year)
            print(f"[download] {url}")
            resp = requests.get(url, headers={"User-Agent": "Mozilla/5.0"}, timeout=300)
            resp.raise_for_status()
            dest.write_bytes(resp.content)
        paths.append(dest)
    return paths


# Helpers
def remove_accents(text: str) -> str:
    return "".join(
        c for c in unicodedata.normalize("NFD", text)
        if unicodedata.category(c) != "Mn"
    )


# Limpieza


def load_and_consolidate(paths: list[Path]) -> pd.DataFrame:
    frames = []
    for path in paths:
        df = pd.read_csv(path, dtype=str, keep_default_na=False, on_bad_lines="skip")
        df.columns = [c.strip().lower().replace("-", "_") for c in df.columns]
        frames.append(df)
        print(f"[read   ] {path.name}: {len(df):,} rows")
    df = pd.concat(frames, ignore_index=True)
    print(f"[consolidated] {len(df):,} rows total")
    return df


def drop_nulls(df: pd.DataFrame) -> pd.DataFrame:
    df = df.replace({"NULL": pd.NA, "null": pd.NA, "SD": pd.NA, "S/D": pd.NA, "": pd.NA})
    existing = [c for c in CRITICAL_COLS if c in df.columns]
    return df.dropna(subset=existing)


def parse_types(df: pd.DataFrame) -> pd.DataFrame:
    df["franja"] = pd.to_numeric(df["franja"], errors="coerce")
    df["latitud"] = pd.to_numeric(df["latitud"].str.replace(",", ".", regex=False), errors="coerce")
    df["longitud"] = pd.to_numeric(df["longitud"].str.replace(",", ".", regex=False), errors="coerce")
    df["fecha_dt"] = pd.to_datetime(df["fecha"], errors="coerce", format="mixed")
    df = df.dropna(subset=["franja", "latitud", "longitud", "fecha_dt"])

    df["hora"] = df["franja"].astype(int)
    df = df[(df["hora"] >= 0) & (df["hora"] <= 23)]
    df["dia_semana"] = df["fecha_dt"].dt.dayofweek  # 0=Monday..6=Sunday
    df["mes"] = df["fecha_dt"].dt.month
    df["anio"] = df["fecha_dt"].dt.year
    return df


def filter_spatial(df: pd.DataFrame) -> pd.DataFrame:
    return df[
        df["latitud"].between(LAT_MIN, LAT_MAX) &
        df["longitud"].between(LON_MIN, LON_MAX)
    ]


def normalize_text(df: pd.DataFrame) -> pd.DataFrame:
    for col in ["tipo", "subtipo", "barrio"]:
        if col in df.columns:
            df[col] = df[col].map(lambda s: remove_accents(str(s)).strip().upper())
    df["comuna"] = (
        df["comuna"].str.extract(r"(\d+)", expand=False).astype("Int64")
    )
    df = df.dropna(subset=["comuna"])
    df["barrio_id"] = df["barrio"].astype("category").cat.codes
    return df


def add_binary_flags(df: pd.DataFrame) -> pd.DataFrame:
    for col in ["uso_arma", "uso_moto"]:
        if col in df.columns:
            df[col] = (df[col].str.upper() == "SI").astype(int)
        else:
            df[col] = 0
    return df


def save_barrio_dictionary(df: pd.DataFrame) -> None:
    dic = df[["barrio_id", "barrio", "comuna"]].drop_duplicates().sort_values("barrio_id")
    dic.to_csv(OUT_PATH.parent / "diccionario_barrios.csv", index=False)


# Main pipeline
def clean(paths: list[Path]) -> pd.DataFrame:
    initial_count: int = 0

    df = load_and_consolidate(paths)
    initial_count = len(df)

    df = drop_nulls(df)
    df = parse_types(df)
    df = filter_spatial(df)
    df = normalize_text(df)
    save_barrio_dictionary(df)
    df = add_binary_flags(df)

    df_final = df[OUTPUT_COLS].reset_index(drop=True)
    retained_pct = 100 * len(df_final) / initial_count
    print(f"[clean  ] {initial_count:,} -> {len(df_final):,} rows "
          f"({initial_count - len(df_final):,} discarded, {retained_pct:.1f}% retained)")
    return df_final


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--years", nargs="+", type=int, default=YEARS)
    parser.add_argument("--no-download", action="store_true")
    args = parser.parse_args()

    if args.no_download:
        paths = sorted(RAW_DIR.glob("delitos_*.csv"))
    else:
        paths = download_years(args.years)

    df = clean(paths)
    OUT_PATH.parent.mkdir(parents=True, exist_ok=True)
    df.to_csv(OUT_PATH, index=False)
    print(f"[output ] {OUT_PATH} ({len(df):,} rows)")


if __name__ == "__main__":
    main()
