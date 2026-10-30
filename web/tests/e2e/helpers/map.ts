import type { Page } from '@playwright/test';

// 1x1 transparent PNG.
const PNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==',
  'base64',
);

/** Serves a fixed tile for every OSM tile request; `count()` is the number of tiles requested. */
export async function stubTiles(page: Page): Promise<{ count: () => number }> {
  let n = 0;
  await page.route('https://tile.openstreetmap.org/**', (route) => {
    n++;
    return route.fulfill({ status: 200, contentType: 'image/png', body: PNG });
  });
  return { count: () => n };
}

/**
 * Tracks every script loaded from now on (also chunks loaded lazily later) and reports
 * whether any body carries Leaflet's signature. Call before navigating.
 */
export function watchLeaflet(page: Page): () => Promise<boolean> {
  const bodies: Promise<string>[] = [];
  page.on('response', (r) => {
    if (r.request().resourceType() === 'script') bodies.push(r.text().catch(() => ''));
  });
  return async () => (await Promise.all(bodies)).some((b) => b.includes('leafletjs.com'));
}
