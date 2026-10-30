import { api } from './api';
import type { PlaceParts } from './geo';

export interface Favorite {
  id: number;
  name: string;
  lat: number;
  lon: number;
  parts: PlaceParts;
}

export type NewFavorite = Omit<Favorite, 'id'>;

const collator = new Intl.Collator('de', { sensitivity: 'base' });

/** Case-insensitive name order; the input is not mutated. */
export function sortFavorites(list: Favorite[]): Favorite[] {
  return [...list].sort((a, b) => collator.compare(a.name, b.name) || a.id - b.id);
}

export async function listFavorites(): Promise<Favorite[]> {
  return sortFavorites((await api<{ places: Favorite[] | null }>('/api/places')).places ?? []);
}

export async function createFavorite(place: NewFavorite): Promise<Favorite> {
  return (await api<{ place: Favorite }>('/api/places', { method: 'POST', body: place })).place;
}

export async function renameFavorite(id: number, name: string): Promise<Favorite> {
  return (await api<{ place: Favorite }>(`/api/places/${id}`, { method: 'PATCH', body: { name } })).place;
}

export async function deleteFavorite(id: number): Promise<void> {
  await api(`/api/places/${id}`, { method: 'DELETE' });
}
