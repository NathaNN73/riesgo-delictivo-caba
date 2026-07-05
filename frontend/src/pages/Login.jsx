import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { login, registro } from "../api";

const card = { background:"#fff", borderRadius:12, padding:32, boxShadow:"0 2px 12px rgba(0,0,0,.06)", maxWidth:420, margin:"40px auto" };
const field = { display:"block", width:"100%", padding:"12px", borderRadius:8, border:"1px solid #ddd", fontSize:14, marginBottom:14, background:"#fafafa" };
const btn1 = { flex:1, padding:"12px", border:"none", borderRadius:8, fontSize:15, fontWeight:700, cursor:"pointer",
  background:"linear-gradient(135deg,#0d47a1,#1565c0)", color:"#fff" };
const btn2 = { ...btn1, background:"linear-gradient(135deg,#2e7d32,#43a047)" };

export default function Login({ onLogin }) {
  const [email, setEmail] = useState(""); const [pass, setPass] = useState("");
  const [error, setError] = useState(""); const nav = useNavigate();

  const go = async (fn, isLogin) => {
    setError("");
    try { const d = await fn(email, pass); if (isLogin) onLogin(d.token); nav("/admin"); }
    catch (e) { setError(e.message); }
  };

  return (
    <div style={card}>
      <h2 style={{fontSize:24,fontWeight:700,marginBottom:20}}>Acceso</h2>
      <input placeholder="Email" value={email} onChange={e=>setEmail(e.target.value)} style={field} />
      <input placeholder="Contraseña" type="password" value={pass} onChange={e=>setPass(e.target.value)} style={field} />
      {error && <p style={{color:"#ef5350",fontSize:14,marginBottom:10}}>{error}</p>}
      <div style={{display:"flex",gap:10}}>
        <button onClick={()=>go(login,true)} style={btn1}>Ingresar</button>
        <button onClick={()=>go(registro,false)} style={btn2}>Registrarse</button>
      </div>
    </div>
  );
}
