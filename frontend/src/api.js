const API = "http://localhost:8080";
let token = localStorage.getItem("token");

export const setToken = (t) => { token = t; localStorage.setItem("token", t); };
export const getToken = () => token;
export const logout = () => { token = null; localStorage.removeItem("token"); };

const auth = () => ({ Authorization: `Bearer ${token}` });

export async function login(email, password) {
  const r = await fetch(`${API}/login`, {
    method: "POST", headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  const data = await r.json();
  if (!r.ok) throw new Error(data.error);
  setToken(data.token);
  return data;
}

export async function registro(email, password) {
  const r = await fetch(`${API}/registro`, {
    method: "POST", headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  const d = await r.json();
  if (!r.ok) throw new Error(d.error);
  return d;
}

export async function predecir(hora, barrio_id, dia_semana) {
  const r = await fetch(`${API}/predecir?hora=${hora}&barrio_id=${barrio_id}&dia_semana=${dia_semana}`);
  const d = await r.json();
  if (!r.ok) throw new Error(d.error);
  return d;
}

export async function entrenar(epocas = 200) {
  const r = await fetch(`${API}/entrenar?epocas=${epocas}`, {
    method: "POST", headers: { ...auth() },
  });
  const d = await r.json();
  if (!r.ok) throw new Error(d.error);
  return d;
}

export async function metricas() {
  const r = await fetch(`${API}/metricas`, { headers: { ...auth() } });
  const d = await r.json();
  if (!r.ok) throw new Error(d.error);
  return d;
}

export async function prediccionesTodas(hora, diaSemana) {
  const r = await fetch(`${API}/predicciones?hora=${hora}&dia_semana=${diaSemana}`);
  const d = await r.json();
  if (!r.ok) throw new Error(d.error);
  return d;
}

export function wsUrl() {
  return `ws://localhost:8080/ws`;
}
