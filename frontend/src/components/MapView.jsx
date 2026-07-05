import { useEffect, useRef } from "react";
import maplibregl from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";

// Coordenadas aproximadas del centro de cada barrio de CABA
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

export default function MapView({ barrioId, onSelect }) {
  const mapRef = useRef(null);
  const mapObj = useRef(null);

  useEffect(() => {
    if (mapObj.current) return;
    mapObj.current = new maplibregl.Map({
      container: mapRef.current,
      style: "https://basemaps.cartocdn.com/gl/positron-gl-style/style.json",
      center: [-58.45, -34.61],
      zoom: 12,
    });
    mapObj.current.addControl(new maplibregl.NavigationControl());
  }, []);

  useEffect(() => {
    if (!mapObj.current) return;
    const coords = BARRIO_COORDS[barrioId] || [-58.45, -34.61];
    mapObj.current.flyTo({ center: coords, zoom: 14, duration: 800 });
  }, [barrioId]);

  return <div ref={mapRef} style={{ width:"100%", height:320, borderRadius:12, overflow:"hidden" }} />;
}
