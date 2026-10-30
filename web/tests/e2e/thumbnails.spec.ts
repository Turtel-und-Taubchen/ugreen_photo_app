import { expect, test, type APIRequestContext, type Page } from '@playwright/test';
import { scrollToTile } from './helpers/grid';
import { waitForLibrary } from './helpers/library';
import { users } from './helpers/auth';

const LOADED = ['2024/07/sample.jpg', '2024/07/sample.heic', '2024/08/screenshot.png', '2024/08/clip.mp4', '2024/08/short.mov'];
const VIDEOS = ['2024/08/clip.mp4', '2024/08/short.mov'];
const BROKEN = '2024/08/defekt.jpg';
const HEADERS = { 'X-Requested-With': 'fetch' };

interface Item {
  id: number;
  relPath: string;
  mediaType: string;
  thumbKey: string;
}

// Fixtures are picked by path: the 5,000 bulk photos share the months and the grid is virtualized.
async function findItems(request: APIRequestContext, wanted: string[]): Promise<Map<string, Item>> {
  const groups = (await (await request.get('/api/photos/groups', { headers: HEADERS })).json()) as {
    months: { month: string }[];
  };
  const found = new Map<string, Item>();
  for (const { month } of groups.months) {
    let after: string | null = null;
    do {
      const q = new URLSearchParams({ month, limit: '200' });
      if (after) q.set('after', after);
      const page = (await (await request.get(`/api/photos?${q}`, { headers: HEADERS })).json()) as {
        items: Item[];
        next: string | null;
      };
      for (const item of page.items) if (wanted.includes(item.relPath)) found.set(item.relPath, item);
      after = page.next;
    } while (after);
    if (found.size === wanted.length) break;
  }
  return found;
}

test('thumbnails render for every format, mark broken files and are cached immutably', async ({ request }) => {
  // Rendering runs on a single background worker, so the first render of every file is slow.
  test.setTimeout(480_000);
  const settle = { timeout: 240_000 };
  await waitForLibrary(request);

  const items = await findItems(request, [...LOADED, BROKEN]);
  expect([...items.keys()].sort()).toEqual([...LOADED, BROKEN].sort());
  for (const rel of VIDEOS) expect(items.get(rel)?.mediaType).toBe('video');
  expect(items.get('2024/07/sample.jpg')?.mediaType).not.toBe('video');

  const thumb = (rel: string) => {
    const item = items.get(rel)!;
    return `/api/thumbs/${item.id}/s?k=${encodeURIComponent(item.thumbKey)}`;
  };
  // 202 means the render is queued; the final answer is either the image or a failure.
  const settled = async (rel: string) => {
    let status = 0;
    await expect
      .poll(async () => (status = (await request.get(thumb(rel), { headers: HEADERS })).status()), settle)
      .not.toBe(202);
    return status;
  };

  for (const rel of LOADED) {
    expect(await settled(rel)).toBe(200);
    const again = await request.get(thumb(rel), { headers: HEADERS });
    expect(again.headers()['content-type']).toMatch(/^image\//);
    expect(again.headers()['cache-control']).toContain('immutable');
  }
  expect(await settled(BROKEN)).not.toBe(200);
});

test('grid tiles show every format, mark broken files and reuse cached thumbnails', async ({ page }) => {
  // Rendering runs on a single background worker, so the first load of all tiles is slow.
  test.setTimeout(480_000);
  const settle = { timeout: 240_000 };
  await waitForLibrary(page.request);
  const items = await findItems(page.request, [...LOADED, BROKEN]);
  expect([...items.keys()].sort()).toEqual([...LOADED, BROKEN].sort());

  const cdp = await page.context().newCDPSession(page);
  await cdp.send('Network.enable');
  const thumbRequests = new Map<string, string>();
  const fromCache = new Set<string>();
  const responses = new Map<string, { status: number; cached: boolean }>();
  cdp.on('Network.requestWillBeSent', (e) => {
    if (e.request.url.includes('/api/thumbs/')) thumbRequests.set(e.requestId, e.request.url);
  });
  cdp.on('Network.requestServedFromCache', (e) => fromCache.add(e.requestId));
  cdp.on('Network.responseReceived', (e) => {
    if (thumbRequests.has(e.requestId)) {
      responses.set(e.requestId, { status: e.response.status, cached: Boolean(e.response.fromDiskCache) });
    }
  });

  // The grid re-renders while its months load, so a tile can detach mid-scroll: retry the whole approach.
  const tileOf = async (p: Page, rel: string) => {
    const id = items.get(rel)!.id;
    await expect(async () => {
      await p.getByTestId('grid-scroller').evaluate((el) => (el.scrollTop = 0));
      const tile = await scrollToTile(p, id, 'down');
      await tile.scrollIntoViewIfNeeded({ timeout: 5_000 });
    }).toPass({ timeout: 60_000 });
    return p.locator(`[data-testid="grid-tile"][data-id="${id}"]`);
  };
  const checkAll = async (withIcons: boolean) => {
    for (const rel of LOADED) {
      const tile = await tileOf(page, rel);
      await expect(tile.locator('[data-state="loaded"]')).toBeVisible(settle);
      await expect(tile.locator('[data-state="pending"]')).toHaveCount(0);
      if (withIcons) await expect(tile.getByLabel('Video')).toHaveCount(VIDEOS.includes(rel) ? 1 : 0);
    }
    const broken = (await tileOf(page, BROKEN)).locator('[data-state="failed"]');
    await expect(broken).toBeVisible(settle);
    await expect(broken).toHaveAccessibleName('Vorschau nicht verfügbar');
    await expect((await tileOf(page, BROKEN)).locator('[data-state="pending"]')).toHaveCount(0);
  };

  await page.goto('/');
  await expect(page.getByTestId('current-user')).toHaveText(users.anna.name);
  await checkAll(true);
  expect([...responses.values()].filter((r) => r.status === 200).length).toBeGreaterThanOrEqual(LOADED.length);

  responses.clear();
  fromCache.clear();
  thumbRequests.clear();

  // goto, not reload: reload revalidates every resource and would bypass the cache by design.
  await page.goto('/');
  await expect(page.getByTestId('current-user')).toHaveText(users.anna.name);
  await checkAll(false);

  // Scrolling to a fixture passes other tiles whose thumbnails were never served, so only the fixtures count.
  // A failed tile may ask again; a thumbnail that was already served must not hit the network.
  const served = LOADED.map((rel) => `/api/thumbs/${items.get(rel)!.id}/`);
  const refetched = [...responses]
    .filter(([id, r]) => r.status === 200 && !r.cached && !fromCache.has(id))
    .map(([id]) => thumbRequests.get(id))
    .filter((url) => served.some((part) => url?.includes(part)));
  expect(refetched).toEqual([]);
});
