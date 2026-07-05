import { useState } from "react";
import { BrowserRouter, Routes, Route, Link, Navigate, useLocation } from "react-router-dom";
import { getToken, setToken, logout as apiLogout } from "./api";
import Login from "./pages/Login";
import Dashboard from "./pages/Dashboard";
import Admin from "./pages/Admin";

const S = {
  nav: { display:"flex",justifyContent:"space-between",alignItems:"center",padding:"16px 32px",
    background:"linear-gradient(135deg, #0d47a1, #1565c0)",color:"#fff",boxShadow:"0 2px 8px rgba(0,0,0,.15)" },
  logo: { fontSize:22, fontWeight:800, letterSpacing:-0.5, display:"flex",alignItems:"center",gap:8 },
  link: (active) => ({ color:"#fff",textDecoration:"none",padding:"8px 16px",borderRadius:6,
    background:active?"rgba(255,255,255,.15)":"transparent",fontWeight:500,fontSize:14 }),
  btn: { marginLeft:12,background:"#ef5350",color:"#fff",border:"none",padding:"6px 16px",
    borderRadius:6,fontWeight:600,cursor:"pointer",fontSize:13 },
  main: { maxWidth:960, margin:"32px auto", padding:"0 24px" },
};

function Layout() {
  const [auth, setAuth] = useState(!!getToken());
  const loc = useLocation();
  const handleLogin = (t) => { setToken(t); setAuth(true); };
  const handleLogout = () => { apiLogout(); setAuth(false); };

  return (
    <>
      <nav style={S.nav}>
        <Link to="/" style={{ textDecoration:"none",color:"#fff" }}>
          <span style={S.logo}><span style={{fontSize:26}}>🗺️</span> Alerta CABA</span>
        </Link>
        <div style={{ display:"flex",alignItems:"center",gap:0 }}>
          <Link to="/" style={S.link(loc.pathname==="/")}>Dashboard</Link>
          {auth && <Link to="/admin" style={S.link(loc.pathname==="/admin")}>Admin</Link>}
          {auth
            ? <button onClick={handleLogout} style={S.btn}>Salir</button>
            : <Link to="/login" style={S.link(loc.pathname==="/login")}>Ingresar</Link>}
        </div>
      </nav>
      <main style={S.main}>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/login" element={<Login onLogin={handleLogin} />} />
          <Route path="/admin" element={auth ? <Admin /> : <Navigate to="/login" />} />
        </Routes>
      </main>
    </>
  );
}

export default function App() {
  return <BrowserRouter><Layout /></BrowserRouter>;
}
