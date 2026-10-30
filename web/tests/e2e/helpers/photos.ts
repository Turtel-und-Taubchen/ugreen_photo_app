import type { APIRequestContext } from '@playwright/test';

export const HEADERS = { 'X-Requested-With': 'fetch' };

interface Item {
  id: number;
  relPath: string;
}

// Capture months follow the EXIF/mtime date, not the folder, so every month is listed.
export async function idsByPath(request: APIRequestContext): Promise<Map<string, number>> {
  const groups = (await (await request.get('/api/photos/groups', { headers: HEADERS })).json()) as { months: { month: string }[] };
  const ids = new Map<string, number>();
  for (const { month } of groups.months) {
    const body = (await (await request.get(`/api/photos?month=${month}&limit=500`, { headers: HEADERS })).json()) as { items: Item[] };
    for (const i of body.items) ids.set(i.relPath, i.id);
  }
  return ids;
}
