import { useState, useEffect, useCallback } from "react";
import { entrenar, metricas, wsUrl } from "../api";

const card = { background:"#fff", borderRadius:8, border:"1px solid #EAEAEA", padding:24 };
const btn = (d) => ({ width:"100%",padding:10,background:d?"#BBB":"#111",
  color:"#fff",border:"none",borderRadius:4,fontSize:13,fontWeight:600,cursor:d?"not-allowed":"pointer" });
const input = { width:"100%",padding:"8px 10px",borderRadius:4,border:"1px solid #EAEAEA",fontSize:13,marginBottom:10,background:"#FBFBFA",outline:"none",color:"#111" };
const heading = { fontSize:16,fontWeight:600,margin:"0 0 20px",color:"#111" };

const h3 = { fontSize:13,fontWeight:600,margin:"0 0 14px",color:"#111",textTransform:"uppercase",letterSpacing:"0.04em" };
const th = { padding:"5px 0",fontSize:12,color:"#787774",fontWeight:600,textAlign:"left" };
const td = { padding:"5px 0",fontSize:13,textAlign:"right",fontWeight:500,color:"#111" };

function Row({ label, value }) {
  return <tr><td style={th}>{label}</td><td style={td}>{value}</td></tr>;
}

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
      <h2 style={heading}>Panel</h2>
      <div style={{display:"grid",gridTemplateColumns:"1fr 1fr 1fr",gap:20}}>
        <div style={card}>
          <h3 style={h3}>Cluster</h3>
          {m && <table style={{width:"100%"}}><tbody>
            <Row label="Uptime" value={`${m.uptime_segundos.toFixed(0)}s`} />
            <Row label="MongoDB" value={m.mongo_conectado?"Conectado":"Desconectado"} />
            <Row label="Redis" value={m.redis_conectado?"Conectado":"Desconectado"} />
            <Row label="Estado" value={m.entrenando?"Entrenando":"En espera"} />
            <Row label="Epoca" value={`${m.epoca_actual}/${m.epocas_totales}`} />
            <Row label="Predicciones" value={m.predicciones} />
            <Row label="Cache hits" value={m.cache_hits} />
            <Row label="Latencia prom" value={`${m.latencia_prom_ms.toFixed(1)} ms`} />
          </tbody></table>}
        </div>
        <div style={card}>
          <h3 style={h3}>Modelo</h3>
          {m?.modelo_entrenado ? <table style={{width:"100%"}}><tbody>
            <Row label="Train size" value={m.train_size} />
            <Row label="Test size" value={m.test_size} />
            <Row label="Accuracy" value={`${(m.ultima_accuracy*100).toFixed(1)}%`} />
            <Row label="Precision" value={`${(m.ultima_precision*100).toFixed(1)}%`} />
            <Row label="Recall" value={`${(m.ultimo_recall*100).toFixed(1)}%`} />
            <Row label="F1" value={`${(m.ultimo_f1*100).toFixed(1)}%`} />
          </tbody></table> : <p style={{fontSize:13,color:"#787774",margin:0}}>No hay modelo entrenado.</p>}
        </div>
        <div style={card}>
          <h3 style={h3}>Entrenamiento</h3>
          <label style={{display:"block",marginBottom:4,fontSize:12,fontWeight:600,color:"#787774"}}>Epocas</label>
          <input type="number" value={ep} onChange={e=>setEp(+e.target.value)} style={input} />
          <button onClick={start} disabled={busy} style={btn(busy)}>
            {busy?"Entrenando":"Iniciar entrenamiento"}
          </button>
          {ws?.tipo==="progreso_entrenamiento" && (
            <div style={{marginTop:10,padding:10,background:"#FBFBFA",borderRadius:4,fontSize:12,color:"#555"}}>
              Epoca {ws.epoca}/{ws.total} &middot; Costo: {ws.costo.toFixed(4)}
            </div>
          )}
        </div>
      </div>
      {res && (
        <div style={{...card,marginTop:20}}>
          <h3 style={h3}>Resultado</h3>
          <div style={{display:"grid",gridTemplateColumns:"1fr 1fr",gap:24}}>
            <table style={{width:"100%"}}><tbody>
              <Row label="Train size" value={res.train_size} />
              <Row label="Test size" value={res.test_size} />
              <Row label="Accuracy" value={`${(res.accuracy*100).toFixed(1)}%`} />
              <Row label="Precision" value={`${(res.precision*100).toFixed(1)}%`} />
              <Row label="Recall" value={`${(res.recall*100).toFixed(1)}%`} />
              <Row label="F1" value={`${(res.f1*100).toFixed(1)}%`} />
            </tbody></table>
            <table style={{width:"100%"}}><tbody>
              <Row label="Verdaderos positivos" value={res.tp} />
              <Row label="Verdaderos negativos" value={res.tn} />
              <Row label="Falsos positivos" value={res.fp} />
              <Row label="Falsos negativos" value={res.fn} />
            </tbody></table>
          </div>
        </div>
      )}
    </div>
  );
}
