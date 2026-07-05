import { useState } from "react";
import { BrowserRouter, Routes, Route, Link, Navigate, useNavigate } from "react-router-dom";
import { getToken, setToken, logout as apiLogout } from "./api";
import Login from "./pages/Login";
import Dashboard from "./pages/Dashboard";
import Admin from "./pages/Admin";

export default function App() {
  const [auth, setAuth] = useState(!!getToken());

  const handleLogin = (token) => { setToken(token); setAuth(true); };
  const handleLogout = () => { apiLogout(); setAuth(false); };

  return (
    <BrowserRouter>
      <nav style={navStyle}>
        <span style={{ fontWeight: "bold", fontSize: 18 }}>Alerta CABA</span>
        <div>
          <Link to="/" style={linkStyle}>Dashboard</Link>
          {auth && <Link to="/admin" style={linkStyle}>Admin</Link>}
          {auth ? (
            <button onClick={handleLogout} style={btnStyle}>Salir</button>
          ) : (
            <Link to="/login" style={linkStyle}>Login</Link>
          )}
        </div>
      </nav>
      <main style={{ maxWidth: 900, margin: "20px auto", padding: "0 20px" }}>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/login" element={<Login onLogin={handleLogin} />} />
          <Route path="/admin" element={auth ? <Admin /> : <Navigate to="/login" />} />
        </Routes>
      </main>
    </BrowserRouter>
  );
}

const navStyle = {
  display: "flex", justifyContent: "space-between", alignItems: "center",
  padding: "12px 24px", background: "#1a237e", color: "white",
};
const linkStyle = { color: "white", marginLeft: 16, textDecoration: "none" };
const btnStyle = { marginLeft: 16, background: "#c62828", color: "white", border: "none", padding: "4px 12px", borderRadius: 4, cursor: "pointer" };
