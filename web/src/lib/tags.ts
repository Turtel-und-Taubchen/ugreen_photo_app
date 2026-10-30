import { api } from './api';
import type { SelectionRequest } from './selection';

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

export interface SelectionTag {
  path: string;
  count: number;
}

/** Tags carried by the selected photos, each with the number of selected photos that have it. */
export async function fetchSelectionTags(selection: SelectionRequest): Promise<SelectionTag[]> {
  return (await api<SelectionTag[] | null>('/api/selection/tags', { method: 'POST', body: selection })) ?? [];
}
