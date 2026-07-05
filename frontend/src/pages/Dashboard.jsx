import { useState, useEffect } from "react";
import { predecir, prediccionesTodas } from "../api";
import MapView from "../components/MapView";

const BARRIOS = {
  0:"Agronomía",1:"Almagro",2:"Balvanera",3:"Barracas",4:"Belgrano",5:"Boca",6:"Boedo",
  7:"Caballito",8:"Chacarita",9:"Coghlan",10:"Colegiales",11:"Constitución",12:"Flores",
  13:"Floresta",14:"Liniers",15:"Mataderos",16:"Monserrat",17:"Monte Castro",18:"Nueva Pompeya",
  19:"Núñez",20:"Palermo",21:"Parque Avellaneda",22:"Parque Chacabuco",23:"Parque Chas",
  24:"Parque Patricios",25:"Paternal",26:"Puerto Madero",27:"Recoleta",28:"Retiro",29:"Saavedra",
  30:"San Cristóbal",31:"San Nicolás",32:"San Telmo",33:"Vélez Sarsfield",34:"Versalles",
  35:"Villa Crespo",36:"Villa del Parque",37:"Villa Devoto",38:"Villa Gral. Mitre",
  39:"Villa Lugano",40:"Villa Luro",41:"Villa Ortúzar",42:"Villa Pueyrredón",43:"Villa Real",
  44:"Villa Riachuelo",45:"Villa Santa Rita",46:"Villa Soldati",47:"Villa Urquiza"
};
const DIAS = ["Lunes","Martes","Miércoles","Jueves","Viernes","Sábado","Domingo"];

const card = { background:"#fff", borderRadius:12, padding:24, boxShadow:"0 2px 12px rgba(0,0,0,.06)" };
const label = { display:"block", marginBottom:4, fontWeight:600, fontSize:13, color:"#555" };
const field = { display:"block", width:"100%", padding:"10px 12px", borderRadius:8, border:"1px solid #ddd", fontSize:14, marginBottom:16, background:"#fafafa" };
const btn = { width:"100%", padding:"14px", background:"linear-gradient(135deg,#0d47a1,#1565c0)", color:"#fff",border:"none",borderRadius:8,fontSize:16,fontWeight:700,cursor:"pointer",marginTop:4 };
const risk = (nivel) => {
  if (nivel==="alto") return {bg:"#ffebee",border:"#ef5350",color:"#c62828",label:"ALTO",emoji:"🔴"};
  if (nivel==="medio") return {bg:"#fff3e0",border:"#ff9800",color:"#e65100",label:"MEDIO",emoji:"🟠"};
  return {bg:"#e8f5e9",border:"#66bb6a",color:"#2e7d32",label:"BAJO",emoji:"🟢"};
};

export default function Dashboard() {
  const [h,setH]=useState(22); const [b,setB]=useState(20); const [d,setD]=useState(5);
  const [r,setR]=useState(null); const [e,setE]=useState("");
  const [allPred,setAllPred]=useState(null);

  useEffect(() => {
    prediccionesTodas(h,d).then(setAllPred).catch(()=>{});
  }, [h,d]);

  const q = async () => { setE(""); try { setR(await predecir(h,b,d)); } catch(ex) { setE(ex.message); } };

  return (
    <div style={{display:"grid",gap:20}}>
      <div style={{...card,padding:8}}>
        <MapView predicciones={allPred} barrioId={b} onSelect={(id)=>{setB(id);q();}} />
        <div style={{display:"flex",justifyContent:"center",gap:16,padding:"8px 0 0",fontSize:12}}>
          <span><span style={{display:"inline-block",width:12,height:12,borderRadius:"50%",background:"#2e7d32",marginRight:4}}></span> Bajo (&lt;35%)</span>
          <span><span style={{display:"inline-block",width:12,height:12,borderRadius:"50%",background:"#ff9800",marginRight:4}}></span> Medio (35-60%)</span>
          <span><span style={{display:"inline-block",width:12,height:12,borderRadius:"50%",background:"#c62828",marginRight:4}}></span> Alto (&gt;60%)</span>
        </div>
      </div>
      <div style={card}>
        <h2 style={{fontSize:22,fontWeight:700,marginBottom:20}}>Consultar riesgo</h2>
        <div style={{display:"grid",gridTemplateColumns:"1fr 1fr 1fr",gap:16}}>
          <div><label style={label}>Hora</label><input type="number" min={0} max={23} value={h} onChange={e=>setH(+e.target.value)} style={field} /></div>
          <div><label style={label}>Día</label><select value={d} onChange={e=>setD(+e.target.value)} style={field}>{DIAS.map((n,i)=><option key={i} value={i}>{n}</option>)}</select></div>
          <div><label style={label}>Barrio</label><select value={b} onChange={e=>setB(+e.target.value)} style={field}>{Object.entries(BARRIOS).map(([id,n])=><option key={id} value={id}>{n}</option>)}</select></div>
        </div>
        <button onClick={q} style={btn}>🔍 Consultar</button>
        {e && <p style={{color:"#ef5350",marginTop:12,fontSize:14}}>{e}</p>}
        {r && (
          <div style={{marginTop:20,background:risk(r.nivel_riesgo).bg,border:`2px solid ${risk(r.nivel_riesgo).border}`,borderRadius:12,padding:24,textAlign:"center"}}>
            <p style={{fontSize:48,fontWeight:800,margin:0,color:risk(r.nivel_riesgo).color}}>{(r.probabilidad*100).toFixed(1)}%</p>
            <p style={{fontSize:18,fontWeight:600,color:risk(r.nivel_riesgo).color,margin:"4px 0"}}>{risk(r.nivel_riesgo).emoji} Riesgo {risk(r.nivel_riesgo).label} en {BARRIOS[b]||b}</p>
            <p style={{fontSize:12,color:"#888"}}>{r.desde_cache?"⚡ desde caché":"🧠 calculado"}</p>
          </div>
        )}
      </div>
    </div>
  );
}
