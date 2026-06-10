"""
Grafico dedicado: SpeedUp + Eficiencia con escala adecuada.
Para que se vea bien la curva real sin que la ideal (lineal) domine.
"""

import pandas as pd
import matplotlib.pyplot as plt
import matplotlib.ticker as ticker
import numpy as np
from pathlib import Path

CSV = Path(__file__).resolve().parent.parent / "go" / "benchmark_results.csv"
OUT_DIR = Path(__file__).resolve().parent

plt.rcParams.update({"figure.dpi": 150, "font.size": 11})

# -- Carga --
df = pd.read_csv(CSV)
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

WORKERS = agg["Workers"].values

# -- Grafico --
fig, ax1 = plt.subplots(figsize=(12, 7))

# === SpeedUp (eje izquierdo) ===
color_speed = "#1565C0"
line1, = ax1.plot(WORKERS, agg["SpeedUp"], "D-", color=color_speed,
                  linewidth=2.2, markersize=8, markerfacecolor="white",
                  markeredgewidth=2, label="SpeedUp real")

# Ideal lineal: solo hasta donde llega el speedup real, punteado fino
ax1.plot(WORKERS, WORKERS, "--", color="#B0BEC5", linewidth=1, alpha=0.6,
         label="SpeedUp ideal (lineal)")

ax1.set_xlabel("Workers")
ax1.set_ylabel("SpeedUp", color=color_speed, fontsize=13)
ax1.tick_params(axis="y", labelcolor=color_speed)
ax1.set_ylim(0, max(agg["SpeedUp"]) + 1.5)
ax1.yaxis.set_major_locator(ticker.MultipleLocator(0.5))
ax1.grid(True, alpha=0.25)

# Anotar speedup sobre cada punto
for w, s in zip(WORKERS, agg["SpeedUp"]):
    offset_y = 16 if w < 32 else -16  # alternar arriba/abajo para no chocar
    ax1.annotate(f"{s:.2f}", (w, s), textcoords="offset points",
                 xytext=(0, offset_y), ha="center", fontsize=7.5,
                 fontweight="bold", color=color_speed)

# === Eficiencia (eje derecho) ===
ax2 = ax1.twinx()
color_eff = "#BF360C"
line2, = ax2.plot(WORKERS, agg["Eficiencia"] * 100, "s--", color=color_eff,
                  linewidth=2, markersize=7, markerfacecolor="white",
                  markeredgewidth=1.5, label="Eficiencia")
ax2.set_ylabel("Eficiencia (%)", color=color_eff, fontsize=13)
ax2.tick_params(axis="y", labelcolor=color_eff)
ax2.set_ylim(0, 110)

# Anotar eficiencia
for w, e in zip(WORKERS, agg["Eficiencia"]):
    ax2.annotate(f"{e*100:.0f}%", (w, e * 100), textcoords="offset points",
                 xytext=(0, 10), ha="center", fontsize=7, color=color_eff)

# === Leyenda unificada ===
lines = [line1, line2]
labels = ["SpeedUp real", "Eficiencia"]
ax1.legend(lines, labels, loc="center right", fontsize=10)

# === Eje X: logaritmico pero con etiquetas exactas ===
ax1.set_xscale("log", base=2)
ax1.set_xticks(WORKERS)
ax1.set_xticklabels([str(w) for w in WORKERS])
ax1.set_xlim(0.9, max(WORKERS) * 1.15)
ax1.xaxis.set_minor_formatter(ticker.NullFormatter())

# === Zonas coloreadas de fondo ===
ax1.axvspan(0.9, 16, alpha=0.04, color="green", label="_escala")
ax1.axvspan(16, 24, alpha=0.04, color="orange", label="_meseta")
ax1.axvspan(24, max(WORKERS) * 1.15, alpha=0.06, color="red", label="_degradacion")

# Etiquetas de zona
for x_start, x_end, label, y_pos in [
    (1.3, 16, "Escala", 4.0),
    (16, 24, "Meseta", 4.0),
    (28, 128, "Degradacion", 2.0),
]:
    x_mid = np.sqrt(x_start * x_end)  # media geometrica en log
    ax1.text(x_mid, y_pos, label, ha="center", fontsize=8.5,
             fontstyle="italic", color="#555555",
             bbox=dict(boxstyle="round,pad=0.2", facecolor="white",
                       edgecolor="#CCCCCC", alpha=0.85))

ax1.set_title("SpeedUp y Eficiencia", fontweight="bold", fontsize=14)

fig.tight_layout()
fig.savefig(OUT_DIR / "grafico_speedup_v2.png", bbox_inches="tight")
print(f"[OK] {OUT_DIR / 'grafico_speedup_v2.png'}")

# -- Resumen --
print(f"\nPeak speedup: {agg['SpeedUp'].max():.2f}x con {int(agg.loc[agg['SpeedUp'].idxmax(), 'Workers'])} workers")
print(f"128 workers es mas lento que 8: {agg[agg['Workers'] == 128]['SpeedUp'].values[0]:.2f}x vs {agg[agg['Workers'] == 8]['SpeedUp'].values[0]:.2f}x")
