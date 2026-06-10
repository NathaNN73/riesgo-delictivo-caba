"""
Genera gráficos del benchmark de entrenamiento concurrente.
Entrada: go/benchmark_results.csv
Salida:  3 gráficos PNG en la carpeta actual.
"""

import pandas as pd
import matplotlib.pyplot as plt
import matplotlib.ticker as ticker
import numpy as np
from pathlib import Path

# ── Configuración ──────────────────────────────────────────────
CSV = Path(__file__).resolve().parent.parent / "go" / "benchmark_results.csv"
OUT_DIR = Path(__file__).resolve().parent
plt.rcParams.update({
    "figure.dpi": 150,
    "font.size": 11,
    "axes.titlesize": 14,
    "axes.labelsize": 12,
})

# ── Carga de datos ─────────────────────────────────────────────
df = pd.read_csv(CSV)

# Descartar primera ejecución de cada worker (cold start: JIT, cache fría)
df_clean = df[df["Run"] > 1].copy()
# Reemplazar "N/A" por NaN en columnas de tiempo
for col in ["CargaMs", "EntrenaMs"]:
    df_clean[col] = pd.to_numeric(df_clean[col], errors="coerce")

# Agrupar por workers
agg = df_clean.groupby("Workers").agg(
    TotalMean=("TotalMs", "mean"),
    TotalMin=("TotalMs", "min"),
    TotalMax=("TotalMs", "max"),
    CargaMean=("CargaMs", "mean"),
    EntrenaMean=("EntrenaMs", "mean"),
).reset_index()

# Tiempo base (1 worker)
t1 = agg.loc[agg["Workers"] == 1, "TotalMean"].values[0]
agg["SpeedUp"] = t1 / agg["TotalMean"]
agg["Eficiencia"] = agg["SpeedUp"] / agg["Workers"]
agg["RestoMs"] = agg["TotalMean"] - agg["CargaMean"] - agg["EntrenaMean"]

WORKERS = agg["Workers"].values

# ── Gráfico 1: SpeedUp + Eficiencia ────────────────────────────
fig1, ax1a = plt.subplots(figsize=(10, 6))

# SpeedUp (eje izquierdo)
color_speed = "#2196F3"
ax1a.plot(WORKERS, agg["SpeedUp"], "o-", color=color_speed, linewidth=2,
          markersize=8, label="SpeedUp real (T₁ / Tₙ)")
ax1a.plot(WORKERS, WORKERS, "--", color="#90CAF9", linewidth=1.5,
          label="SpeedUp ideal (lineal)")
ax1a.set_xlabel("Workers")
ax1a.set_ylabel("SpeedUp", color=color_speed)
ax1a.tick_params(axis="y", labelcolor=color_speed)
ax1a.set_ylim(0, max(WORKERS) + 2)
ax1a.legend(loc="upper left")
ax1a.grid(True, alpha=0.3)

# Eficiencia (eje derecho)
ax1b = ax1a.twinx()
color_eff = "#FF5722"
ax1b.plot(WORKERS, agg["Eficiencia"] * 100, "s--", color=color_eff,
          linewidth=1.5, markersize=7, label="Eficiencia (SpeedUp / Workers)")
ax1b.set_ylabel("Eficiencia (%)", color=color_eff)
ax1b.tick_params(axis="y", labelcolor=color_eff)
ax1b.set_ylim(0, 110)
ax1b.legend(loc="upper right")

# Anotar valores sobre los puntos
for w, s, e in zip(WORKERS, agg["SpeedUp"], agg["Eficiencia"]):
    ax1a.annotate(f"{s:.2f}×", (w, s), textcoords="offset points",
                  xytext=(0, 12), ha="center", fontsize=8, color=color_speed)

ax1a.set_xticks(WORKERS)
ax1a.set_title(f"SpeedUp y Eficiencia — 10 000 épocas, {agg['TotalMean'].iloc[0]:.0f} ms base\n"
               f"CPUs lógicos: 16 | Núcleos físicos: 8",
               fontweight="bold")
fig1.tight_layout()
fig1.savefig(OUT_DIR / "grafico_speedup.png", bbox_inches="tight")
print(f"[OK] {OUT_DIR / 'grafico_speedup.png'}")


# ── Gráfico 2: Tiempo total con barras min/max ─────────────────
fig2, ax2 = plt.subplots(figsize=(10, 6))

colors = ["#4CAF50" if w <= 16 else "#FF9800" for w in WORKERS]
bars = ax2.bar(WORKERS.astype(str), agg["TotalMean"], color=colors,
               edgecolor="white", linewidth=0.8)

# Barras de error min/max
yerr_low = agg["TotalMean"] - agg["TotalMin"]
yerr_high = agg["TotalMax"] - agg["TotalMean"]
ax2.errorbar(range(len(WORKERS)), agg["TotalMean"],
             yerr=[yerr_low, yerr_high], fmt="none",
             ecolor="black", capsize=5, linewidth=1.2)

# Anotar speedup sobre cada barra
for i, (w, val, s) in enumerate(zip(WORKERS, agg["TotalMean"], agg["SpeedUp"])):
    ax2.text(i, val + 30, f"{s:.1f}×", ha="center", fontsize=9,
             fontweight="bold", color="#333333")

ax2.set_ylabel("Tiempo total promedio (ms)")
ax2.set_xlabel("Workers")
ax2.set_title("Tiempo de ejecución total por cantidad de workers\n"
              "Barras = promedio de 4 ejecuciones | Líneas = rango [min, max]",
              fontweight="bold")
ax2.grid(axis="y", alpha=0.3)

# Leyenda de colores
from matplotlib.patches import Patch
legend_elements = [
    Patch(facecolor="#4CAF50", label="Óptimo (≤ 16 = CPUs lógicos)"),
    Patch(facecolor="#FF9800", label="Sobre-suscripción (> 16)"),
]
ax2.legend(handles=legend_elements, loc="upper right")
fig2.tight_layout()
fig2.savefig(OUT_DIR / "grafico_tiempos.png", bbox_inches="tight")
print(f"[OK] {OUT_DIR / 'grafico_tiempos.png'}")


# ── Gráfico 3: Desglose del tiempo (stacked) ───────────────────
fig3, ax3 = plt.subplots(figsize=(10, 6))

width = 0.55
bottom_carga = np.zeros(len(WORKERS))
bottom_entrena = agg["CargaMean"].values

ax3.bar(WORKERS.astype(str), agg["CargaMean"], width, label="Carga CSV",
        color="#607D8B", edgecolor="white")
ax3.bar(WORKERS.astype(str), agg["EntrenaMean"], width, bottom=bottom_carga,
        label="Entrenamiento", color="#03A9F4", edgecolor="white")
ax3.bar(WORKERS.astype(str), agg["RestoMs"], width,
        bottom=bottom_carga + agg["EntrenaMean"].values,
        label="Dataset + Evaluación + Guardado", color="#CDDC39",
        edgecolor="white")

# Porcentajes sobre cada segmento
for i, w in enumerate(WORKERS):
    total = agg["TotalMean"].iloc[i]
    carga_pct = agg["CargaMean"].iloc[i] / total * 100
    entrena_pct = agg["EntrenaMean"].iloc[i] / total * 100
    if carga_pct > 8:
        ax3.text(i, agg["CargaMean"].iloc[i] / 2, f"{carga_pct:.0f}%",
                 ha="center", va="center", fontsize=8, fontweight="bold",
                 color="white")
    if entrena_pct > 8:
        ax3.text(i, agg["CargaMean"].iloc[i] + agg["EntrenaMean"].iloc[i] / 2,
                 f"{entrena_pct:.0f}%", ha="center", va="center", fontsize=8,
                 fontweight="bold", color="white")

ax3.set_ylabel("Tiempo (ms)")
ax3.set_xlabel("Workers")
ax3.set_title("Desglose del tiempo de ejecución por etapa\n"
              "Muestra cómo la carga (secuencial) crece en proporción al aumentar workers",
              fontweight="bold")
ax3.legend(loc="upper right")
ax3.grid(axis="y", alpha=0.3)
fig3.tight_layout()
fig3.savefig(OUT_DIR / "grafico_desglose.png", bbox_inches="tight")
print(f"[OK] {OUT_DIR / 'grafico_desglose.png'}")

# ── Tabla resumen en consola ───────────────────────────────────
print("\n" + "=" * 78)
print(f"{'Workers':<10}{'Promedio':>10}{'Mín':>10}{'Máx':>10}{'SpeedUp':>10}{'Eficiencia':>12}")
print("-" * 78)
for _, row in agg.iterrows():
    print(f"{int(row['Workers']):<10}{row['TotalMean']:>8.0f} ms{row['TotalMin']:>8.0f} ms"
          f"{row['TotalMax']:>8.0f} ms{row['SpeedUp']:>8.2f}×{row['Eficiencia']*100:>10.1f}%")
print("=" * 78)
print(f"\nLey de Amdahl estimada: fraccion secuencial ~ {agg['RestoMs'].iloc[0] / t1 * 100:.0f}%")
print(f"SpeedUp maximo teorico (1/f_seq) = {1 / (agg['RestoMs'].iloc[0] / t1):.1f}x")
print(f"SpeedUp real obtenido = {agg['SpeedUp'].max():.2f}x")
print("\nListo. Los graficos estan en la carpeta python/")
