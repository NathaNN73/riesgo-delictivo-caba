import { useState, useEffect, useCallback } from "react";
import { entrenar, metricas, wsUrl } from "../api";

export default function Admin() {
  const [m, setM] = useState(null);
  const [wsMsg, setWsMsg] = useState(null);
  const [entrenando, setEntrenando] = useState(false);
  const [epocas, setEpocas] = useState(200);

  const cargar = useCallback(async () => {
    try { setM(await metricas()); } catch {}
  }, []);

  useEffect(() => { cargar(); const id = setInterval(cargar, 3000); return () => clearInterval(id); }, [cargar]);

  useEffect(() => {
    const ws = new WebSocket(wsUrl());
    ws.onmessage = (e) => setWsMsg(JSON.parse(e.data));
    return () => ws.close();
  }, []);

  const iniciar = async () => {
    setEntrenando(true);
    try { await entrenar(epocas); cargar(); } catch (e) { alert(e.message); }
    setEntrenando(false);
  };

  return (
    <div>
      <h2>Panel de Administración</h2>

      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16 }}>
        <Panel titulo="Clúster">
          {m && <Metricas m={m} />}
        </Panel>

        <Panel titulo="Entrenamiento">
          <label>Épocas</label>
          <input type="number" value={epocas} onChange={e => setEpocas(+e.target.value)}
            style={{ width: "100%", padding: 6, marginBottom: 8, borderRadius: 4, border: "1px solid #ccc" }} />
          <button onClick={iniciar} disabled={entrenando || m?.entrenando}
            style={btnStyle(entrenando || m?.entrenando)}>
            {entrenando || m?.entrenando ? "Entrenando..." : "Iniciar entrenamiento"}
          </button>
          {wsMsg?.tipo === "progreso_entrenamiento" && (
            <div style={{ marginTop: 8, padding: 8, background: "#e3f2fd", borderRadius: 4 }}>
              Época {wsMsg.epoca}/{wsMsg.total} — Costo: {wsMsg.costo.toFixed(4)}
            </div>
          )}
        </Panel>
      </div>
    </div>
  );
}

function Panel({ titulo, children }) {
  return (
    <div style={{ padding: 16, background: "#f5f5f5", borderRadius: 8 }}>
      <h3 style={{ margin: "0 0 12px 0" }}>{titulo}</h3>
      {children}
    </div>
  );
}

function Metricas({ m }) {
  const rows = [
    ["Uptime", `${m.uptime_segundos.toFixed(0)}s`],
    ["MongoDB", m.mongo_conectado ? "✅" : "❌"],
    ["Redis", m.redis_conectado ? "✅" : "❌"],
    ["Entrenando", m.entrenando ? "Sí" : "No"],
    ["Época", `${m.epoca_actual}/${m.epocas_totales}`],
    ["Predicciones", m.predicciones],
    ["Cache hits", m.cache_hits],
    ["Latencia prom", `${m.latencia_prom_ms.toFixed(1)}ms`],
  ];
  return (
    <table style={{ width: "100%" }}>
      <tbody>
        {rows.map(([k, v]) => (
          <tr key={k}><td style={{ padding: 4, fontWeight: "bold" }}>{k}</td><td>{v}</td></tr>
        ))}
      </tbody>
    </table>
  );
}

function btnStyle(disabled) {
  return {
    width: "100%", padding: 10,
    background: disabled ? "#9e9e9e" : "#1a237e",
    color: "white", border: "none", borderRadius: 4, cursor: disabled ? "not-allowed" : "pointer",
  };
}
