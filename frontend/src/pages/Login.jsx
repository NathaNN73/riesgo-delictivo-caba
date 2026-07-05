import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { login, registro } from "../api";

export default function Login({ onLogin }) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const navigate = useNavigate();

  const handle = async (fn, isLogin) => {
    setError("");
    try {
      const data = await fn(email, password);
      if (isLogin) onLogin(data.token);
      navigate("/admin");
    } catch (e) {
      setError(e.message);
    }
  };

  return (
    <div style={{ maxWidth: 400, margin: "40px auto" }}>
      <h2>Acceso</h2>
      <input placeholder="Email" value={email} onChange={e => setEmail(e.target.value)} style={inputStyle} />
      <input placeholder="Password" type="password" value={password} onChange={e => setPassword(e.target.value)} style={inputStyle} />
      {error && <p style={{ color: "red" }}>{error}</p>}
      <div style={{ display: "flex", gap: 10 }}>
        <button onClick={() => handle(login, true)} style={primaryBtn}>Ingresar</button>
        <button onClick={() => handle(registro, false)} style={secondaryBtn}>Registrarse</button>
      </div>
    </div>
  );
}

const inputStyle = { display: "block", width: "100%", padding: 8, marginBottom: 10, borderRadius: 4, border: "1px solid #ccc" };
const primaryBtn = { padding: "8px 20px", background: "#1a237e", color: "white", border: "none", borderRadius: 4, cursor: "pointer", flex: 1 };
const secondaryBtn = { ...primaryBtn, background: "#4caf50", flex: 1 };
