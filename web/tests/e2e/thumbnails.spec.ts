import { expect, test } from '@playwright/test';
import { login, users } from './helpers/auth';

const LOADED = ['2024/07/sample.jpg', '2024/07/sample.heic', '2024/08/screenshot.png', '2024/08/clip.mp4', '2024/08/short.mov'];
const VIDEOS = ['2024/08/clip.mp4', '2024/08/short.mov'];
const BROKEN = '2024/08/defekt.jpg';

test('start page previews every format, marks broken files and reuses cached thumbnails', async ({ page }) => {
  // Rendering runs on a single background worker, so the first load of all tiles is slow.
  test.setTimeout(300_000);
  const settle = { timeout: 240_000 };

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

  await login(page, users.anna);

  const tile = (rel: string) => page.getByTestId(`thumb-${rel}`);
  for (const rel of LOADED) await expect(tile(rel).locator('[data-state="loaded"]')).toBeVisible(settle);
  for (const rel of VIDEOS) await expect(tile(rel).getByLabel('Video')).toBeVisible();
  await expect(tile('2024/07/sample.jpg').getByLabel('Video')).toHaveCount(0);

  const broken = tile(BROKEN).locator('[data-state="failed"]');
  await expect(broken).toBeVisible(settle);
  await expect(broken).toHaveAccessibleName('Vorschau nicht verfügbar');

  // Everything is finished: no tile is still waiting for a render.
  await expect(page.locator('[data-state="pending"]')).toHaveCount(0, settle);
  expect([...responses.values()].filter((r) => r.status === 200).length).toBeGreaterThanOrEqual(LOADED.length);
  expect((await page.request.get('/api/health')).status()).toBe(200);

  responses.clear();
  fromCache.clear();
  thumbRequests.clear();

  // goto, not reload: reload revalidates every resource and would bypass the cache by design.
  await page.goto('/');
  for (const rel of LOADED) await expect(tile(rel).locator('[data-state="loaded"]')).toBeVisible(settle);
  await expect(tile(BROKEN).locator('[data-state="failed"]')).toBeVisible(settle);

  // A failed tile may ask again; a thumbnail that was already served must not hit the network.
  const refetched = [...responses]
    .filter(([id, r]) => r.status === 200 && !r.cached && !fromCache.has(id))
    .map(([id]) => thumbRequests.get(id));
  expect(refetched).toEqual([]);
  expect([...responses.values()].some((r) => r.status === 200)).toBe(true);
});
