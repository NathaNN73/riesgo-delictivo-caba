import { useState, useEffect } from "react";
import { predecir, prediccionesTodas } from "../api";
import MapView from "../components/MapView";

const BARRIOS = {
  0:"Agronomia",1:"Almagro",2:"Balvanera",3:"Barracas",4:"Belgrano",5:"Boca",6:"Boedo",
  7:"Caballito",8:"Chacarita",9:"Coghlan",10:"Colegiales",11:"Constitucion",12:"Flores",
  13:"Floresta",14:"Liniers",15:"Mataderos",16:"Monserrat",17:"Monte Castro",18:"Nueva Pompeya",
  19:"Nunez",20:"Palermo",21:"Parque Avellaneda",22:"Parque Chacabuco",23:"Parque Chas",
  24:"Parque Patricios",25:"Paternal",26:"Puerto Madero",27:"Recoleta",28:"Retiro",29:"Saavedra",
  30:"San Cristobal",31:"San Nicolas",32:"San Telmo",33:"Velez Sarsfield",34:"Versalles",
  35:"Villa Crespo",36:"Villa del Parque",37:"Villa Devoto",38:"Villa Gral. Mitre",
  39:"Villa Lugano",40:"Villa Luro",41:"Villa Ortuzar",42:"Villa Pueyrredon",43:"Villa Real",
  44:"Villa Riachuelo",45:"Villa Santa Rita",46:"Villa Soldati",47:"Villa Urquiza"
};
const DIAS = ["Lunes","Martes","Miercoles","Jueves","Viernes","Sabado","Domingo"];

const now = new Date();
const defaultH = now.getHours();
const defaultD = now.getDay();

const card = { background:"#fff", borderRadius:8, border:"1px solid #EAEAEA", overflow:"hidden" };
const label = { display:"block", marginBottom:4, fontSize:12, fontWeight:600, color:"#787774", textTransform:"uppercase", letterSpacing:"0.04em" };
const field = { display:"block", width:"100%", padding:"10px 12px", borderRadius:4, border:"1px solid #EAEAEA", fontSize:14, marginBottom:16, background:"#FBFBFA", outline:"none", color:"#111" };
const btn = { width:"100%", padding:"12px", background:"#111", color:"#fff", border:"none", borderRadius:4, fontSize:14, fontWeight:600, cursor:"pointer" };
const section = { fontSize:22,fontWeight:600,letterSpacing:"-0.02em",color:"#111",margin:"0 0 8px" };
const sub = { fontSize:15,color:"#787774",lineHeight:1.6,margin:"0 0 28px" };

const riskStyle = {
  alto:  { bg:"#FDEBEC", border:"#F4C2C2", text:"#9F2F2D", label:"Alto" },
  medio: { bg:"#FBF3DB", border:"#F4D98E", text:"#956400", label:"Medio" },
  bajo:  { bg:"#EDF3EC", border:"#B8D4B4", text:"#346538", label:"Bajo" },
};

export default function Dashboard() {
  const [h,setH]=useState(defaultH); const [b,setB]=useState(20); const [d,setD]=useState(defaultD);
  const [r,setR]=useState(null); const [e,setE]=useState("");
  const [allPred,setAllPred]=useState(null);
  const [sliderVal,setSliderVal]=useState(defaultH);

  useEffect(() => {
    prediccionesTodas(h,d).then(setAllPred).catch(()=>{});
  }, [h,d]);

  const q = async () => { setE(""); try { setR(await predecir(h,b,d)); } catch(ex) { setE(ex.message); } };
  const nivel = r ? riskStyle[r.nivel_riesgo] : null;

  return (
    <div style={{display:"grid",gap:28}}>

      {/* Header */}
      <header>
        <h1 style={section}>Centinela</h1>
        <p style={sub}>
          Consulta el nivel de riesgo delictivo de cada barrio de la Ciudad de Buenos Aires segun el dia y la hora.
          Los datos provienen del dataset abierto de delitos del Ministerio de Justicia y Seguridad (2016-2023).
        </p>
      </header>

      {/* Mapa */}
      <div style={{...card,padding:0,position:"relative"}}>
        <MapView predicciones={allPred} barrioId={b} onSelect={(id)=>{setB(id);q();}} />
        <div style={{position:"absolute",bottom:12,left:12,background:"#fff",border:"1px solid #EAEAEA",
          borderRadius:4,padding:"8px 12px",fontSize:11,color:"#787774",display:"flex",gap:12}}>
          <span><span style={{display:"inline-block",width:8,height:8,borderRadius:"50%",background:"#346538",marginRight:4,verticalAlign:"middle"}}></span> Bajo</span>
          <span><span style={{display:"inline-block",width:8,height:8,borderRadius:"50%",background:"#956400",marginRight:4,verticalAlign:"middle"}}></span> Medio</span>
          <span><span style={{display:"inline-block",width:8,height:8,borderRadius:"50%",background:"#9F2F2D",marginRight:4,verticalAlign:"middle"}}></span> Alto</span>
        </div>
      </div>

      {/* Formulario */}
      <div style={{...card,padding:28}}>
        <div style={{display:"grid",gridTemplateColumns:"1fr 1fr 1fr",gap:14}}>
          <div><label style={label}>Hora</label><input type="number" min={0} max={23} value={h} onChange={e=>{const v=+e.target.value; setH(v); setSliderVal(v);}} style={field} /></div>
          <div><label style={label}>Dia</label><select value={d} onChange={e=>setD(+e.target.value)} style={field}>{DIAS.map((n,i)=><option key={i} value={i}>{n}</option>)}</select></div>
          <div><label style={label}>Barrio</label><select value={b} onChange={e=>setB(+e.target.value)} style={field}>{Object.entries(BARRIOS).map(([id,n])=><option key={id} value={id}>{n}</option>)}</select></div>
        </div>

        {/* Slider horario */}
        {r && (
          <div style={{marginBottom:16}}>
            <label style={{...label,marginBottom:8}}>Evolucion del riesgo en {BARRIOS[b]}</label>
            <input type="range" min={0} max={23} value={sliderVal}
              onChange={e=>{const v=+e.target.value; setSliderVal(v); setH(v); q();}}
              style={{width:"100%",accentColor:"#111"}} />
            <div style={{display:"flex",justifyContent:"space-between",fontSize:11,color:"#BBB"}}>
              <span>00:00</span><span>06:00</span><span>12:00</span><span>18:00</span><span>23:00</span>
            </div>
          </div>
        )}

        <button onClick={q} style={btn}>Consultar</button>
        {e && <p style={{color:"#9F2F2D",fontSize:13,marginTop:12}}>{e}</p>}
        {r && (
          <div style={{marginTop:16,background:nivel.bg,border:`1px solid ${nivel.border}`,borderRadius:6,padding:20,textAlign:"center"}}>
            <p style={{fontSize:40,fontWeight:700,margin:0,color:nivel.text,letterSpacing:"-0.02em"}}>{(r.probabilidad*100).toFixed(1)}%</p>
            <p style={{fontSize:14,fontWeight:500,color:nivel.text,margin:"4px 0"}}>Riesgo {nivel.label} &middot; {BARRIOS[b]}</p>
            {r.desde_cache && <p style={{fontSize:11,color:"#787774",margin:0}}>resultado en cache</p>}
          </div>
        )}
      </div>

      {/* Como funciona */}
      <div style={{...card,padding:28}}>
        <h2 style={{...section,fontSize:16,marginBottom:12}}>Como funciona</h2>
        <p style={{fontSize:14,color:"#555",lineHeight:1.7,margin:0}}>
          Centinela utiliza un modelo de regresion logistica entrenado sobre mas de un millon de registros historicos
          de delitos en la Ciudad de Buenos Aires. El mapa se colorea segun el nivel de riesgo estimado para la
          combinacion de barrio, dia y hora seleccionada. Los colores indican la probabilidad de que una zona
          presente una actividad delictiva por encima del percentil 75 historico.
        </p>
      </div>

    </div>
  );
}
