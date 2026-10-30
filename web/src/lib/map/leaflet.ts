// The only module that imports Leaflet. Callers must reach it through a dynamic
// import() so the library stays out of the main bundle.
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';

export interface LatLon {
  lat: number;
  lon: number;
}

export interface MapHandle {
  setMarker(p: LatLon | null): void;
  focus(p: LatLon, zoom: number): void;
  remove(): void;
}

const DEFAULT_VIEW: [number, number] = [47.5, 13.5];
const DEFAULT_ZOOM = 6;

export function createMap(
  el: HTMLElement,
  opts: { point: LatLon | null; onpick?: (p: LatLon) => void },
): MapHandle {
  const map = L.map(el, { attributionControl: false }).setView(
    opts.point ? [opts.point.lat, opts.point.lon] : DEFAULT_VIEW,
    opts.point ? 15 : DEFAULT_ZOOM,
  );
  L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', { maxZoom: 19 }).addTo(map);
  const icon = L.divIcon({ className: 'pc-marker', iconSize: [24, 24], iconAnchor: [12, 12] });
  let marker: L.Marker | null = null;

  function setMarker(p: LatLon | null) {
    if (!p) {
      marker?.remove();
      marker = null;
      return;
    }
    if (marker) marker.setLatLng([p.lat, p.lon]);
    else marker = L.marker([p.lat, p.lon], { icon, interactive: false, keyboard: false }).addTo(map);
  }
  setMarker(opts.point);

  const onpick = opts.onpick;
  if (onpick) {
    map.on('click', (e: L.LeafletMouseEvent) => {
      // wrap(): panning across the antimeridian yields longitudes outside ±180.
      const ll = e.latlng.wrap();
      const p = { lat: Number(ll.lat.toFixed(6)), lon: Number(ll.lng.toFixed(6)) };
      setMarker(p);
      onpick(p);
    });
  }

  return {
    setMarker,
    focus(p, zoom) {
      setMarker(p);
      map.setView([p.lat, p.lon], zoom);
    },
    remove: () => map.remove(),
  };
}
