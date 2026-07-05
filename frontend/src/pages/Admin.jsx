import { useState, useEffect, useCallback } from "react";
import { entrenar, metricas, wsUrl } from "../api";

const card = { background:"#fff", borderRadius:12, padding:24, boxShadow:"0 2px 12px rgba(0,0,0,.06)" };
const btn = (d) => ({ width:"100%",padding:12,background:d?"#9e9e9e":"linear-gradient(135deg,#0d47a1,#1565c0)",
  color:"#fff",border:"none",borderRadius:8,fontSize:14,fontWeight:700,cursor:d?"not-allowed":"pointer" });
const input = { width:"100%",padding:"10px 12px",borderRadius:8,border:"1px solid #ddd",fontSize:14,marginBottom:10,background:"#fafafa" };

const metricRows = [
  ["Uptime", m => `${m.uptime_segundos.toFixed(0)}s`],
  ["MongoDB", m => m.mongo_conectado?"🟢":"🔴"],
  ["Redis", m => m.redis_conectado?"🟢":"🔴"],
  ["Entrenando", m => m.entrenando?"▶️ Sí":"⏸️ No"],
  ["Época", m => `${m.epoca_actual}/${m.epocas_totales}`],
  ["Predicciones", m => m.predicciones],
  ["Cache hits", m => m.cache_hits],
  ["Latencia prom", m => `${m.latencia_prom_ms.toFixed(1)}ms`],
];

const resultRows = (r) => [
  ["Train", r.train_size],["Test", r.test_size],
  ["Accuracy", (r.accuracy*100).toFixed(1)+"%"],
  ["Precision", (r.precision*100).toFixed(1)+"%"],
  ["Recall", (r.recall*100).toFixed(1)+"%"],
  ["F1", (r.f1*100).toFixed(1)+"%"],
  ["VP (TP)", r.tp],["VN (TN)", r.tn],
  ["FP", r.fp],["FN", r.fn],
];

export default function Admin() {
  const [m,setM]=useState(null); const [ws,setWs]=useState(null);
  const [loading,setLoading]=useState(false); const [ep,setEp]=useState(200); const [res,setRes]=useState(null);

  const load = useCallback(async ()=>{ try{setM(await metricas())}catch{}},[]);
  useEffect(()=>{ load(); const t=setInterval(load,3000); return ()=>clearInterval(t); },[load]);
  useEffect(()=>{ const w=new WebSocket(wsUrl()); w.onmessage=e=>setWs(JSON.parse(e.data)); return ()=>w.close(); },[]);

  const start = async ()=> { setLoading(true); setRes(null); try{setRes(await entrenar(ep)); load()}catch(e){alert(e.message)} setLoading(false); };
  const busy = loading || m?.entrenando;

  return (
    <div>
      <h2 style={{fontSize:24,fontWeight:700,marginBottom:20}}>Panel de Administración</h2>
      <div style={{display:"grid",gridTemplateColumns:"1fr 1fr",gap:20}}>
        <div style={card}>
          <h3 style={{fontSize:16,fontWeight:700,marginBottom:16}}>📊 Clúster</h3>
          {m && <table style={{width:"100%",borderCollapse:"collapse"}}><tbody>
            {metricRows.map(([k,fn])=><tr key={k}>
              <td style={{padding:"6px 0",fontWeight:600,color:"#555",fontSize:13}}>{k}</td>
              <td style={{textAlign:"right",fontSize:14}}>{fn(m)}</td></tr>)}
          </tbody></table>}
        </div>
        <div style={card}>
          <h3 style={{fontSize:16,fontWeight:700,marginBottom:16}}>🏋️ Entrenamiento</h3>
          <label style={{display:"block",marginBottom:4,fontWeight:600,fontSize:13,color:"#555"}}>Épocas</label>
          <input type="number" value={ep} onChange={e=>setEp(+e.target.value)} style={input} />
          <button onClick={start} disabled={busy} style={btn(busy)}>
            {busy?"Entrenando...":"Iniciar entrenamiento"}
          </button>
          {ws?.tipo==="progreso_entrenamiento" && (
            <div style={{marginTop:12,padding:12,background:"#e3f2fd",borderRadius:8,fontSize:13}}>
              Época {ws.epoca}/{ws.total} — Costo: {ws.costo.toFixed(4)}
            </div>
          )}
        </div>
      </div>
      {res && (
        <div style={{...card,marginTop:20}}>
          <h3 style={{fontSize:16,fontWeight:700,marginBottom:16}}>✅ Resultado del entrenamiento</h3>
          <div style={{display:"grid",gridTemplateColumns:"1fr 1fr",gap:16}}>
            <table style={{width:"100%",borderCollapse:"collapse"}}><tbody>
              {resultRows(res).slice(0,5).map(([k,v])=><tr key={k}>
                <td style={{padding:"6px 0",fontWeight:600,color:"#555"}}>{k}</td>
                <td style={{textAlign:"right",fontWeight:700}}>{v}</td></tr>)}
            </tbody></table>
            <table style={{width:"100%",borderCollapse:"collapse"}}><tbody>
              {resultRows(res).slice(5).map(([k,v])=><tr key={k}>
                <td style={{padding:"6px 0",fontWeight:600,color:"#555"}}>{k}</td>
                <td style={{textAlign:"right",fontWeight:700}}>{v}</td></tr>)}
            </tbody></table>
          </div>
        </div>
      )}
    </div>
  );
}
