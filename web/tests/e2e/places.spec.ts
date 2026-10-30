import { expect, test } from '@playwright/test';
import { NOMINATIM_MOCK_PORT } from './compose';
import { login, users } from './helpers/auth';

const MOCK = `http://127.0.0.1:${NOMINATIM_MOCK_PORT}`;

interface MockRequest {
  path: string;
  userAgent: string;
}

async function mockLog(): Promise<MockRequest[]> {
  return (await fetch(`${MOCK}/__mock/requests`)).json();
}

async function setMode(mode: 'ok' | 'error'): Promise<void> {
  await fetch(`${MOCK}/__mock/mode`, { method: 'POST', body: JSON.stringify({ mode }) });
}

test.afterEach(async () => {
  await setMode('ok');
});

test('place search finds a POI, names it and survives an outage', async ({ page }) => {
  test.setTimeout(300_000);
  await fetch(`${MOCK}/__mock/reset`, { method: 'POST' });
  await login(page, users.anna);
  const version = ((await (await page.request.get('/api/health')).json()) as { version: string }).version;

  // Typing never searches; Enter does, once.
  const searchRequests: string[] = [];
  page.on('request', (r) => {
    if (new URL(r.url()).pathname === '/api/geo/search') searchRequests.push(r.url());
  });
  await page.goto('/orte');
  const field = page.getByLabel('Ort suchen');
  await field.pressSequentially('Ritzenhofer Saalfelden');
  await expect(field).toHaveValue('Ritzenhofer Saalfelden');
  expect(searchRequests).toEqual([]);
  expect(await mockLog()).toEqual([]);
  await field.press('Enter');
  const hits = page.getByRole('list', { name: 'Treffer' }).getByRole('listitem');
  await expect(hits.first()).toContainText('Ritzenhofer');
  await expect(hits.first()).toContainText('Ritzenseestraße 12');
  await expect(hits.first()).toContainText('hotel');
  expect(await hits.count()).toBeLessThanOrEqual(10);
  const log = await mockLog();
  expect(log).toHaveLength(1);
  expect(log[0].path).toBe('/search');
  expect(log[0].userAgent).toBe(`PhotoCurator/${version} (e2e@example.invalid)`);
  const firstTexts = await hits.allInnerTexts();

  // Choosing fills the name and parts; edits stick.
  await hits.first().getByRole('button').click();
  const name = page.getByLabel('Ortsname');
  await expect(name).toHaveValue('Ritzenhofer, Saalfelden');
  const chosen = page.getByRole('region', { name: 'Gewählter Ort' });
  await expect(chosen.locator('dt:text-is("City") + dd')).toHaveText('Saalfelden');
  await expect(chosen.locator('dt:text-is("Country") + dd')).toHaveText('Österreich');
  await name.fill('Mein Hotel');
  await expect(name).toHaveValue('Mein Hotel');

  // A differently spelled query is served from the cache.
  await field.fill('  ritzenhofer   SAALFELDEN ');
  const searchButton = page.getByRole('button', { name: 'Suchen', exact: true });
  const [second] = await Promise.all([
    page.waitForResponse((r) => new URL(r.url()).pathname === '/api/geo/search'),
    searchButton.click(),
  ]);
  expect(second.ok()).toBe(true);
  await expect(searchButton).toBeEnabled();
  await expect(async () => expect(await hits.allInnerTexts()).toEqual(firstTexts)).toPass();
  expect(searchRequests).toHaveLength(2);
  expect(await mockLog()).toHaveLength(1);

  // Reverse lookup for a point.
  await page.getByLabel('Breite').fill('47.4259');
  await page.getByLabel('Länge').fill('12.8467');
  await page.getByRole('button', { name: 'Ort zum Punkt ermitteln' }).click();
  await expect(name).toHaveValue(/Gasthof Beispiel/);
  expect((await mockLog()).filter((r) => r.path === '/reverse')).toHaveLength(1);

  // Outage: the message shows and a manual point and name still work.
  await setMode('error');
  await field.fill('Zell am See');
  await field.press('Enter');
  await expect(page.getByRole('alert')).toContainText('Ortssuche gerade nicht erreichbar');
  await page.getByLabel('Breite').fill('47.3');
  await page.getByLabel('Länge').fill('12.79');
  await name.fill('Hütte am Berg');
  const summary = page.getByTestId('chosen-summary');
  await expect(summary).toContainText('Hütte am Berg');
  await expect(summary).toContainText('47.3');
  await expect(summary).toContainText('12.79');
});
