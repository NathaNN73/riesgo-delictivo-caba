"""
Genera 2 graficos del benchmark de entrenamiento concurrente.
Entrada: go/benchmark_results.csv
Salida:  PNG en la carpeta actual.
"""

import pandas as pd
import matplotlib.pyplot as plt
import numpy as np
from pathlib import Path

# -- Configuracion --
CSV = Path(__file__).resolve().parent.parent / "go" / "benchmark_results.csv"
OUT_DIR = Path(__file__).resolve().parent
plt.rcParams.update({
    "figure.dpi": 150,
    "font.size": 11,
    "axes.titlesize": 13,
    "axes.labelsize": 11,
})

# -- Carga de datos --
df = pd.read_csv(CSV)

# Descartar run 1 de cada worker (cold start)
df_clean = df[df["Run"] > 1].copy()
for col in ["CargaMs", "EntrenaMs"]:
    df_clean[col] = pd.to_numeric(df_clean[col], errors="coerce")

agg = df_clean.groupby("Workers").agg(
    TotalMean=("TotalMs", "mean"),
    TotalMin=("TotalMs", "min"),
    TotalMax=("TotalMs", "max"),
    CargaMean=("CargaMs", "mean"),
    EntrenaMean=("EntrenaMs", "mean"),
).reset_index()

t1 = agg.loc[agg["Workers"] == 1, "TotalMean"].values[0]
agg["SpeedUp"] = t1 / agg["TotalMean"]
agg["Eficiencia"] = agg["SpeedUp"] / agg["Workers"]
agg["RestoMs"] = agg["TotalMean"] - agg["CargaMean"] - agg["EntrenaMean"]

WORKERS = agg["Workers"].values

# ================================================================
# Grafico 1: Tiempo total con barras min/max
# ================================================================
fig2, ax2 = plt.subplots(figsize=(10, 6))

colors = ["#4CAF50" if w <= 16 else "#FF9800" for w in WORKERS]
ax2.bar(WORKERS.astype(str), agg["TotalMean"], color=colors,
        edgecolor="white", linewidth=0.8)

yerr_low = agg["TotalMean"] - agg["TotalMin"]
yerr_high = agg["TotalMax"] - agg["TotalMean"]
ax2.errorbar(range(len(WORKERS)), agg["TotalMean"],
             yerr=[yerr_low, yerr_high], fmt="none",
             ecolor="#333333", capsize=5, linewidth=1.2)

for i, (val, s) in enumerate(zip(agg["TotalMean"], agg["SpeedUp"])):
    ax2.text(i, val + 35, f"{s:.1f}x", ha="center", fontsize=8.5,
             fontweight="bold", color="#333333")

ax2.set_ylabel("Tiempo total (ms)")
ax2.set_xlabel("Workers")
ax2.set_title("Tiempo total de ejecucion", fontweight="bold")
ax2.grid(axis="y", alpha=0.3)

from matplotlib.patches import Patch
legend_elements = [
    Patch(facecolor="#4CAF50", label="<= 16 (CPUs logicos)"),
    Patch(facecolor="#FF9800", label="> 16 (sobre-suscripcion)"),
]
ax2.legend(handles=legend_elements, loc="upper right")
fig2.tight_layout()
fig2.savefig(OUT_DIR / "grafico_tiempos.png", bbox_inches="tight")
print(f"[OK] {OUT_DIR / 'grafico_tiempos.png'}")


# ================================================================
# Grafico 2: Desglose por etapa (stacked)
# ================================================================
fig3, ax3 = plt.subplots(figsize=(10, 6))

width = 0.55
bottom_entrena = agg["CargaMean"].values

ax3.bar(WORKERS.astype(str), agg["CargaMean"], width, label="Carga CSV",
        color="#607D8B", edgecolor="white")
ax3.bar(WORKERS.astype(str), agg["EntrenaMean"], width, bottom=bottom_entrena,
        label="Entrenamiento", color="#03A9F4", edgecolor="white")
ax3.bar(WORKERS.astype(str), agg["RestoMs"], width,
        bottom=bottom_entrena + agg["EntrenaMean"].values,
        label="Dataset + Eval + Guardado", color="#CDDC39", edgecolor="white")

for i in range(len(WORKERS)):
    total = agg["TotalMean"].iloc[i]
    carga_pct = agg["CargaMean"].iloc[i] / total * 100
    entrena_pct = agg["EntrenaMean"].iloc[i] / total * 100
    if carga_pct > 7:
        ax3.text(i, agg["CargaMean"].iloc[i] / 2, f"{carga_pct:.0f}%",
                 ha="center", va="center", fontsize=7.5, fontweight="bold", color="white")
    if entrena_pct > 7:
        ax3.text(i, agg["CargaMean"].iloc[i] + agg["EntrenaMean"].iloc[i] / 2,
                 f"{entrena_pct:.0f}%", ha="center", va="center", fontsize=7.5,
                 fontweight="bold", color="white")

ax3.set_ylabel("Tiempo (ms)")
ax3.set_xlabel("Workers")
ax3.set_title("Desglose de tiempo por etapa", fontweight="bold")
ax3.legend(loc="upper right")
ax3.grid(axis="y", alpha=0.3)
fig3.tight_layout()
fig3.savefig(OUT_DIR / "grafico_desglose.png", bbox_inches="tight")
print(f"[OK] {OUT_DIR / 'grafico_desglose.png'}")


# ================================================================
# Tabla resumen en consola
# ================================================================
print("\n" + "=" * 76)
print(f"{'Workers':<10}{'Promedio':>10}{'Min':>10}{'Max':>10}{'SpeedUp':>10}{'Eficiencia':>12}")
print("-" * 76)
for _, row in agg.iterrows():
    print(f"{int(row['Workers']):<10}{row['TotalMean']:>8.0f} ms{row['TotalMin']:>8.0f} ms"
          f"{row['TotalMax']:>8.0f} ms{row['SpeedUp']:>8.2f}x{row['Eficiencia']*100:>10.1f}%")
print("=" * 76)
f_seq = agg["RestoMs"].iloc[0] / t1
print(f"\nFraccion secuencial (Amdahl): ~{f_seq * 100:.0f}%")
print(f"SpeedUp maximo teorico: {1 / f_seq:.1f}x")
print(f"SpeedUp real obtenido:   {agg['SpeedUp'].max():.2f}x")
print("\nListo. Los graficos estan en la carpeta python/")
