import { afterEach, describe, expect, it, vi } from 'vitest';
import { buildTagIndex, findExact, suggest } from './tagSuggest';
import type { TagNode } from './tags';

const node = (path: string, children: TagNode[] = []): TagNode => ({
  name: path.split('/').pop() as string,
  path,
  count: 0,
  children,
});

afterEach(() => vi.restoreAllMocks());

describe('tag suggestions', () => {
  it('suggestions match anywhere in the path regardless of case and umlauts', () => {
    const tree = [
      node('Reise', [node('Reise/Österreich')]),
      node('Anlass', [node('Anlass/Geburtstag')]),
      node('Ab', [node('Ab/Lebensgeburt')]),
    ];
    const index = buildTagIndex(tree);
    for (const q of ['öst', 'ÖST', 'reich']) expect(suggest(index, q)).toEqual(['Reise/Österreich']);
    expect(suggest(index, 'ost')).toEqual([]);
    expect(suggest(index, 'Öst')).toEqual(['Reise/Österreich']);
    expect(suggest(index, 'geb')).toEqual(['Anlass/Geburtstag', 'Ab/Lebensgeburt']);
    expect(suggest(index, 'geb', ['Anlass/Geburtstag'])).toEqual(['Ab/Lebensgeburt']);
    expect(suggest(index, '  ')).toEqual([]);
    const many = buildTagIndex(Array.from({ length: 20 }, (_, i) => node(`T/a${i}`)));
    expect(suggest(many, 'a')).toHaveLength(6);
    expect(findExact(index, ' reise/ÖSTERREICH ')).toBe('Reise/Österreich');
    expect(findExact(index, 'Reise/Öst')).toBeUndefined();
  });

  it('a keystroke over 1,000 tags lowercases only the query', () => {
    const tree = Array.from({ length: 1000 }, (_, i) => node(`Gruppe${i}/Österreich${i}`));
    const index = buildTagIndex(tree);
    const lower = vi.spyOn(String.prototype, 'toLocaleLowerCase');
    const norm = vi.spyOn(String.prototype, 'normalize');
    const result = suggest(index, 'öst');
    expect(lower.mock.calls.length).toBeLessThanOrEqual(1);
    expect(norm.mock.calls.length).toBeLessThanOrEqual(1);
    expect(result).toHaveLength(6);
    expect(result.every((p) => p.includes('Österreich'))).toBe(true);
  });
});
