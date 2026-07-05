import { useState } from "react";
import { predecir } from "../api";

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

export default function Dashboard() {
  const [hora, setHora] = useState(22);
  const [barrio, setBarrio] = useState(20);
  const [dia, setDia] = useState(5);
  const [result, setResult] = useState(null);
  const [error, setError] = useState("");

  const consultar = async () => {
    setError(""); setResult(null);
    try {
      const r = await predecir(hora, barrio, dia);
      setResult(r);
    } catch (e) {
      setError(e.message);
    }
  };

  return (
    <div style={{ maxWidth: 500, margin: "40px auto" }}>
      <h2>Consultar riesgo delictivo</h2>
      <label>Hora (0-23)</label>
      <input type="number" min={0} max={23} value={hora} onChange={e => setHora(+e.target.value)} style={inputStyle} />

      <label>Barrio</label>
      <select value={barrio} onChange={e => setBarrio(+e.target.value)} style={inputStyle}>
        {Object.entries(BARRIOS).map(([id, name]) => (
          <option key={id} value={id}>{name}</option>
        ))}
      </select>

      <label>Día</label>
      <select value={dia} onChange={e => setDia(+e.target.value)} style={inputStyle}>
        {DIAS.map((d, i) => <option key={i} value={i}>{d}</option>)}
      </select>

      <button onClick={consultar} style={{ ...primaryBtn, width: "100%", marginTop: 10 }}>
        Consultar
      </button>

      {error && <p style={{ color: "red", marginTop: 10 }}>{error}</p>}

      {result && (
        <div style={{ marginTop: 20, padding: 16, background: result.alto_riesgo ? "#ffebee" : "#e8f5e9", borderRadius: 8 }}>
          <p style={{ fontSize: 24, fontWeight: "bold", margin: 0 }}>
            {(result.probabilidad * 100).toFixed(1)}%
          </p>
          <p style={{ margin: "4px 0", color: result.alto_riesgo ? "#c62828" : "#2e7d32" }}>
            Riesgo {result.alto_riesgo ? "ALTO" : "BAJO"} {result.desde_cache ? "(cache)" : ""}
          </p>
        </div>
      )}
    </div>
  );
}

const inputStyle = { display: "block", width: "100%", padding: 8, marginBottom: 10, borderRadius: 4, border: "1px solid #ccc" };
const primaryBtn = { padding: "8px 20px", background: "#1a237e", color: "white", border: "none", borderRadius: 4, cursor: "pointer" };
