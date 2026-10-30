import { describe, expect, it } from 'vitest';
import { emptyParts } from './geo';
import { sortFavorites, type Favorite } from './places';

const fav = (id: number, name: string): Favorite => ({ id, name, lat: 0, lon: 0, parts: { ...emptyParts } });

describe('sortFavorites', () => {
  it('orders case-insensitively without mutating the input', () => {
    const input = [fav(1, 'zelt'), fav(2, 'Anker'), fav(3, 'berg')];
    expect(sortFavorites(input).map((f) => f.name)).toEqual(['Anker', 'berg', 'zelt']);
    expect(input[0].name).toBe('zelt');
  });
});
