import { useEffect, useRef } from "react";
import maplibregl from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";

const CENTRO = [-58.45, -34.61];

const BARRIO_COORDS = {
  0:[-58.48,-34.59],1:[-58.43,-34.61],2:[-58.40,-34.61],3:[-58.38,-34.64],
  4:[-58.46,-34.56],5:[-58.36,-34.63],6:[-58.42,-34.63],7:[-58.44,-34.62],
  8:[-58.45,-34.59],9:[-58.47,-34.56],10:[-58.45,-34.57],11:[-58.39,-34.63],
  12:[-58.46,-34.63],13:[-58.47,-34.63],14:[-58.51,-34.64],15:[-58.50,-34.66],
  16:[-58.38,-34.61],17:[-58.47,-34.62],18:[-58.41,-34.65],19:[-58.46,-34.54],
  20:[-58.43,-34.58],21:[-58.48,-34.64],22:[-58.44,-34.63],23:[-58.46,-34.59],
  24:[-58.40,-34.64],25:[-58.46,-34.60],26:[-58.36,-34.61],27:[-58.39,-34.59],
  28:[-58.38,-34.60],29:[-58.48,-34.55],30:[-58.40,-34.62],31:[-58.38,-34.60],
  32:[-58.37,-34.62],33:[-58.48,-34.62],34:[-58.51,-34.62],35:[-58.44,-34.60],
  36:[-58.47,-34.60],37:[-58.50,-34.60],38:[-58.47,-34.59],39:[-58.47,-34.67],
  40:[-58.49,-34.62],41:[-58.46,-34.58],42:[-58.47,-34.57],43:[-58.50,-34.63],
  44:[-58.43,-34.68],45:[-58.47,-34.61],46:[-58.45,-34.67],47:[-58.47,-34.55],
};

function colorForRisk(prob) {
  if (prob >= 0.6) return "#9F2F2D";
  if (prob >= 0.35) return "#956400";
  return "#346538";
}

export default function MapView({ predicciones, barrioId, onSelect }) {
  const mapRef = useRef(null);
  const mapObj = useRef(null);
  const markersRef = useRef([]);

  useEffect(() => {
    if (mapObj.current) return;
    const map = new maplibregl.Map({
      container: mapRef.current, style: "https://basemaps.cartocdn.com/gl/positron-gl-style/style.json",
      center: CENTRO, zoom: 12,
    });
    map.addControl(new maplibregl.NavigationControl());
    mapObj.current = map;
  }, []);

  useEffect(() => {
    const map = mapObj.current;
    if (!map || !predicciones) return;
    markersRef.current.forEach(m => m.remove());
    markersRef.current = [];

    Object.entries(BARRIO_COORDS).forEach(([id, coords]) => {
      const prob = predicciones[id] || 0;
      const el = document.createElement("div");
      el.style.cssText = `width:14px;height:14px;border-radius:50%;background:${colorForRisk(prob)};
        border:2px solid rgba(255,255,255,.85);box-shadow:0 1px 3px rgba(0,0,0,.15);cursor:pointer;transition:width .15s,height .15s,margin .15s`;
      el.dataset.originalSize = "14";
      el.onmouseenter = () => { el.style.width = "20px"; el.style.height = "20px"; el.style.margin = "-3px 0 0 -3px"; };
      el.onmouseleave = () => { el.style.width = "14px"; el.style.height = "14px"; el.style.margin = "0"; };
      el.onclick = () => onSelect && onSelect(parseInt(id));
      const marker = new maplibregl.Marker({ element: el }).setLngLat(coords).addTo(map);
      markersRef.current.push(marker);
    });
  }, [predicciones, onSelect]);

  useEffect(() => {
    if (!mapObj.current) return;
    const coords = BARRIO_COORDS[barrioId] || CENTRO;
    mapObj.current.flyTo({ center: coords, zoom: 14, duration: 800 });
  }, [barrioId]);

  return <div ref={mapRef} style={{ width:"100%", height:420 }} />;
}
