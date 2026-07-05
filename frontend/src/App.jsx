import { useState } from "react";
import { BrowserRouter, Routes, Route, Link, Navigate, useLocation } from "react-router-dom";
import { getToken, setToken, logout as apiLogout } from "./api";
import Login from "./pages/Login";
import Dashboard from "./pages/Dashboard";
import Admin from "./pages/Admin";

const S = {
  nav: { display:"flex",justifyContent:"space-between",alignItems:"center",padding:"14px 32px",
    background:"#111",color:"#fff" },
  logo: { fontSize:17, fontWeight:600, letterSpacing:"-0.02em", color:"#fff" },
  link: (active) => ({ color:active?"#fff":"#999",textDecoration:"none",padding:"6px 14px",
    fontSize:14,fontWeight:500 }),
  btn: { marginLeft:12,background:"transparent",color:"#999",border:"1px solid #333",padding:"5px 14px",
    borderRadius:4,fontSize:13,cursor:"pointer" },
  footer: { textAlign:"center",padding:"24px",fontSize:12,color:"#666",background:"#111",marginTop:60 },
};

function Layout() {
  const [auth, setAuth] = useState(!!getToken());
  const loc = useLocation();
  const handleLogin = (t) => { setToken(t); setAuth(true); };
  const handleLogout = () => { apiLogout(); setAuth(false); };

  return (
    <div style={{minHeight:"100vh",display:"flex",flexDirection:"column"}}>
      <nav style={S.nav}>
        <Link to="/"><span style={S.logo}>Centinela</span></Link>
        <div style={{ display:"flex",alignItems:"center" }}>
          <Link to="/" style={S.link(loc.pathname==="/")}>Mapa</Link>
          {auth && <Link to="/admin" style={S.link(loc.pathname==="/admin")}>Panel</Link>}
          {auth
            ? <button onClick={handleLogout} style={S.btn}>Cerrar sesion</button>
            : <Link to="/login" style={S.link(loc.pathname==="/login")}>Ingresar</Link>}
        </div>
      </nav>
      <main style={{ flex:1, maxWidth:1040, margin:"0 auto", padding:"40px 24px 0", width:"100%" }}>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/login" element={<Login onLogin={handleLogin} />} />
          <Route path="/admin" element={auth ? <Admin /> : <Navigate to="/login" />} />
        </Routes>
      </main>
      <footer style={S.footer}>
        Datos: Ministerio de Justicia y Seguridad, GCBA (2016-2023)
      </footer>
    </div>
  );
}

export default function App() {
  return <BrowserRouter><Layout /></BrowserRouter>;
}
