import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { login, registro } from "../api";

const card = { background:"#fff", borderRadius:8, border:"1px solid #EAEAEA", padding:28, maxWidth:400, margin:"40px auto" };
const field = { display:"block", width:"100%", padding:"10px 12px", borderRadius:4, border:"1px solid #EAEAEA", fontSize:14, marginBottom:14, background:"#FBFBFA", outline:"none", color:"#111" };
const btn1 = { flex:1, padding:"10px", border:"none", borderRadius:4, fontSize:14, fontWeight:600, cursor:"pointer",
  background:"#111", color:"#fff" };
const btn2 = { ...btn1, background:"#fff", color:"#111", border:"1px solid #EAEAEA" };
const heading = { fontSize:16,fontWeight:600,margin:"0 0 20px",color:"#111" };

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
      <h2 style={heading}>Acceso</h2>
      <input placeholder="Email" value={email} onChange={e=>setEmail(e.target.value)} style={field} />
      <input placeholder="Contrasena" type="password" value={pass} onChange={e=>setPass(e.target.value)} style={field} />
      {error && <p style={{color:"#9F2F2D",fontSize:13,marginBottom:12}}>{error}</p>}
      <div style={{display:"flex",gap:10}}>
        <button onClick={()=>go(login,true)} style={btn1}>Ingresar</button>
        <button onClick={()=>go(registro,false)} style={btn2}>Registrarse</button>
      </div>
    </div>
  );
}
