import type { TagNode } from './tags';

interface Entry {
  path: string;
  norm: string;
  segments: string[];
}

export interface TagIndex {
  entries: Entry[];
}

const collator = new Intl.Collator('de');

function normalize(s: string): string {
  return s.normalize('NFC').toLocaleLowerCase('de');
}

/** Every node path (parents included) is normalized once here, never per keystroke. */
export function buildTagIndex(tree: TagNode[]): TagIndex {
  const entries: Entry[] = [];
  const walk = (nodes: TagNode[]) => {
    for (const n of nodes) {
      const norm = normalize(n.path);
      entries.push({ path: n.path, norm, segments: norm.split('/') });
      walk(n.children ?? []);
    }
  };
  walk(tree);
  return { entries };
}

/** Substring matches; paths with a segment starting with the query come first. No accent folding. */
export function suggest(index: TagIndex, query: string, exclude: string[] = [], limit = 6): string[] {
  const q = normalize(query.trim());
  if (!q) return [];
  const skip = new Set(exclude);
  const starts: string[] = [];
  const rest: string[] = [];
  for (const e of index.entries) {
    if (skip.has(e.path) || !e.norm.includes(q)) continue;
    (e.segments.some((s) => s.startsWith(q)) ? starts : rest).push(e.path);
  }
  starts.sort(collator.compare);
  rest.sort(collator.compare);
  return [...starts, ...rest].slice(0, limit);
}

/** The known path equal to the trimmed input after normalization. */
export function findExact(index: TagIndex, input: string): string | undefined {
  const q = normalize(input.trim());
  return q ? index.entries.find((e) => e.norm === q)?.path : undefined;
}
