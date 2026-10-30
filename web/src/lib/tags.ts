import { api } from './api';

export interface TagNode {
  name: string;
  path: string;
  count: number;
  children: TagNode[];
}

const collator = new Intl.Collator('de');

/** Siblings are ordered by German collation at every level; the input is not mutated. */
export function sortTree(nodes: TagNode[]): TagNode[] {
  return [...nodes]
    .sort((a, b) => collator.compare(a.name, b.name))
    .map((n) => ({ ...n, children: sortTree(n.children ?? []) }));
}

export async function fetchTags(): Promise<TagNode[]> {
  const res = await api<{ tags: TagNode[] | null }>('/api/tags');
  return sortTree(res.tags ?? []);
}

export async function createTag(path: string): Promise<void> {
  await api('/api/tags', { method: 'POST', body: { path } });
}

export async function deleteTag(path: string): Promise<void> {
  await api(`/api/tags?path=${encodeURIComponent(path)}`, { method: 'DELETE' });
}
