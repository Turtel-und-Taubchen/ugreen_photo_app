import { api } from './api';

export interface PlaceParts {
  sublocation: string;
  city: string;
  state: string;
  country: string;
  countryCode: string;
}

export interface Place {
  name: string;
  address: string;
  kind: string;
  lat: number;
  lon: number;
  label: string;
  parts: PlaceParts;
}

export interface GeoStatus {
  enabled: boolean;
  reason?: string;
}

export const emptyParts: PlaceParts = { sublocation: '', city: '', state: '', country: '', countryCode: '' };

export function fetchGeoStatus(): Promise<GeoStatus> {
  return api<GeoStatus>('/api/geo/status');
}

export async function searchPlaces(q: string): Promise<Place[]> {
  const res = await api<{ results: Place[] | null }>(`/api/geo/search?q=${encodeURIComponent(q)}`);
  return res.results ?? [];
}

export async function reversePlace(lat: number, lon: number): Promise<Place | null> {
  const res = await api<{ place: Place | null }>(`/api/geo/reverse?lat=${lat}&lon=${lon}`);
  return res.place ?? null;
}

/** The text to show when the place search is switched off, or null while it is available. */
export function geoNotice(status: GeoStatus | null): string | null {
  if (!status || status.enabled) return null;
  return status.reason ?? 'Die Ortssuche ist nicht verfügbar.';
}

/** Accepts a decimal comma; empty or non-numeric input yields null. */
export function parseCoordinate(text: string): number | null {
  const t = text.trim().replace(',', '.');
  if (t === '') return null;
  const n = Number(t);
  return Number.isFinite(n) ? n : null;
}
