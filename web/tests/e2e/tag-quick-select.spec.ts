import { expect, test, type Page } from '@playwright/test';
import { users } from './helpers/auth';
import { scrollToTile } from './helpers/grid';
import { waitForLibrary } from './helpers/library';
import { HEADERS, idsByPath } from './helpers/photos';

test.use({ viewport: { width: 390, height: 844 }, hasTouch: true });

const FILE = '2024/07/tagged.jpg';
const PARENT = 'Reise/Österreich';
const settle = { timeout: 60_000 };

const monthLabel = (takenAt: string) =>
  new Date(`${takenAt.slice(0, 7)}-15T12:00:00`).toLocaleDateString('de-DE', { month: 'long', year: 'numeric' });

async function openAssignDialog(page: Page, id: number, takenAt: string) {
  await page.goto('/');
  await expect(page.getByTestId('current-user')).toHaveText(users.anna.name);
  await page.getByRole('button', { name: 'Auswählen', exact: true }).click();
  await page.getByRole('button', { name: 'Zeitraum' }).click();
  await page.getByRole('button', { name: monthLabel(takenAt), exact: true }).click();
  await (await scrollToTile(page, id, 'down')).click();
  await expect(page.getByTestId('selection-count')).toHaveText('1 ausgewählt');
  await page.getByRole('button', { name: 'Tags zuweisen' }).click();
  return page.getByRole('dialog', { name: 'Tags zuweisen' });
}

test('the tag dialog offers recent tags and suggestions and creates new tags on a phone', async ({ page }) => {
  test.setTimeout(420_000);
  await waitForLibrary(page.request);
  const id = (await idsByPath(page.request)).get(FILE)!;
  const facts = (await (await page.request.get(`/api/photos/${id}`, { headers: HEADERS })).json()) as { taken_at: string };
  const fresh = `Wir zwei ${Date.now()}`;

  const created = await page.request.post('/api/tags', { data: { path: PARENT }, headers: { 'X-Requested-With': 'fetch' } });
  expect(created.ok()).toBe(true);

  try {
    let assign = await openAssignDialog(page, id, facts.taken_at);
    const combo = assign.getByRole('combobox', { name: 'Tag suchen oder neu anlegen' });
    await expect(combo).toBeVisible();

    let requests = 0;
    page.on('request', (r) => {
      if (new URL(r.url()).pathname.startsWith('/api/')) requests++;
    });

    await combo.fill('öst');
    const option = assign.getByRole('option', { name: PARENT });
    await expect(option).toBeVisible();
    expect(requests).toBe(0);
    await option.getByRole('button').tap();
    await expect(assign.getByRole('button', { name: `${PARENT} entfernen` })).toBeVisible();

    await combo.fill(fresh);
    await combo.press('Enter');
    await expect(assign.getByRole('button', { name: `${fresh} entfernen` })).toBeVisible();

    await combo.fill('Neu|Tag');
    await assign.getByRole('option', { name: /neu anlegen/ }).getByRole('button').tap();
    await expect(assign.getByRole('alert')).toContainText('|');
    await expect(combo).toHaveValue('Neu|Tag');
    await combo.fill('');

    await assign.getByRole('button', { name: 'Zuweisen' }).click();
    await expect(assign).toBeHidden();
    await expect(page.getByRole('region', { name: 'Schreibjobs' })).toContainText('1 erfolgreich', settle);

    assign = await openAssignDialog(page, id, facts.taken_at);
    const chips = assign.getByRole('region', { name: 'Zuletzt verwendet' }).getByRole('button');
    await expect(chips.first()).toHaveText(fresh);
    await expect(chips.nth(1)).toHaveText(PARENT);

    const inView = async (loc: ReturnType<Page['locator']>) => {
      const box = (await loc.boundingBox())!;
      expect(box.y).toBeGreaterThanOrEqual(0);
      expect(box.y + box.height).toBeLessThanOrEqual(844);
      expect(box.height).toBeGreaterThanOrEqual(44);
    };
    for (const chip of await chips.all()) await inView(chip);
    await assign.getByRole('combobox', { name: 'Tag suchen oder neu anlegen' }).fill('rei');
    const options = assign.getByRole('option');
    await expect(options.first()).toBeVisible();
    for (const o of await options.all()) await inView(o);
    expect(await page.evaluate(() => window.scrollY)).toBe(0);

    const parentChip = chips.filter({ hasText: PARENT });
    await parentChip.tap();
    await expect(parentChip).toHaveAttribute('aria-pressed', 'true');
  } finally {
    for (const path of [`${PARENT}`, 'Reise', fresh]) {
      await page.request.delete(`/api/tags?path=${encodeURIComponent(path)}`, { headers: HEADERS });
    }
  }
});
