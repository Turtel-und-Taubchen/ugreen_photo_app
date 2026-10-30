import { expect, test, type APIRequestContext, type Page } from '@playwright/test';
import { users } from './helpers/auth';
import { bulkPlan } from './helpers/bulk-library';
import { openTile } from './helpers/grid';
import { waitForLibrary } from './helpers/library';

interface Item {
  id: number;
  relPath: string;
}

const HEADERS = { 'X-Requested-With': 'fetch' };
const settle = { timeout: 60_000 };

async function listMonth(request: APIRequestContext, month: string): Promise<Item[]> {
  const items: Item[] = [];
  let after: string | null = null;
  do {
    const q = new URLSearchParams({ month, limit: '500' });
    if (after) q.set('after', after);
    const body = (await (await request.get(`/api/photos?${q}`, { headers: HEADERS })).json()) as { items: Item[]; next: string | null };
    items.push(...body.items);
    after = body.next;
  } while (after);
  return items;
}

async function findByPath(request: APIRequestContext, relPath: string): Promise<{ items: Item[]; index: number }> {
  const groups = (await (await request.get('/api/photos/groups', { headers: HEADERS })).json()) as { months: { month: string }[] };
  for (const { month } of groups.months) {
    const items = await listMonth(request, month);
    const index = items.findIndex((i) => i.relPath === relPath);
    if (index >= 0) return { items, index };
  }
  throw new Error(`${relPath} is not indexed`);
}

async function swipe(page: Page, dx: number, dy = 0): Promise<void> {
  await page.getByTestId('detail-stage').evaluate(
    (el, [mx, my]) => {
      const fire = (type: string, x: number, y: number) =>
        el.dispatchEvent(new PointerEvent(type, { pointerId: 7, pointerType: 'touch', clientX: x, clientY: y, bubbles: true }));
      fire('pointerdown', 200, 300);
      fire('pointermove', 200 + mx / 2, 300 + my / 2);
      fire('pointerup', 200 + mx, 300 + my);
    },
    [dx, dy],
  );
}

test("the detail view shows a photo's facts and steps through photos with arrow keys and swipes", async ({ page }) => {
  test.setTimeout(480_000);
  await page.setViewportSize({ width: 390, height: 844 });
  await waitForLibrary(page.request);
  await page.goto('/');
  await expect(page.getByTestId('current-user')).toHaveText(users.anna.name);

  const detail = page.getByTestId('photo-detail');
  const path = page.getByTestId('detail-path');
  const scroller = page.getByTestId('grid-scroller');

  // The newest photo is a bulk file: dated by its name, no review state, no newer neighbour.
  const newestItems = await listMonth(page.request, bulkPlan().months[0].month);
  await page.locator('[data-testid="grid-tile"][data-id]').first().click(settle);
  await expect(detail).toBeVisible();
  await expect(path).toHaveText(newestItems[0].relPath);
  await expect(page.getByText('Datum aus dem Dateinamen')).toBeVisible();
  await expect(page.getByTestId('detail-state')).toHaveText('offen');
  await expect(page.getByTestId('detail-tags')).toHaveText('keine Tags');
  await expect(page.getByTestId('detail-place')).toHaveText('kein Ort');
  await page.keyboard.press('ArrowLeft');
  await expect(path).toHaveText(newestItems[0].relPath);
  await page.keyboard.press('ArrowRight');
  await expect(path).toHaveText(newestItems[1].relPath);
  await page.keyboard.press('ArrowLeft');
  await expect(path).toHaveText(newestItems[0].relPath);
  await page.keyboard.press('Escape');
  await expect(detail).toBeHidden();

  // The tagged fixture, opened mid-month so both neighbours exist.
  const tagged = await findByPath(page.request, '2024/07/tagged.jpg');
  expect(tagged.index).toBeGreaterThan(0);
  expect(tagged.index).toBeLessThan(tagged.items.length - 1);
  const newer = tagged.items[tagged.index - 1];
  const older = tagged.items[tagged.index + 1];
  await page.getByRole('button', { name: 'Zeitraum' }).click();
  await page.getByRole('button', { name: 'Juli 2024', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Juli 2024', level: 2 })).toBeInViewport(settle);
  await openTile(page, tagged.items[tagged.index].id, 'down');
  const scrolledTo = await scroller.evaluate((el) => el.scrollTop);

  await expect(path).toHaveText('2024/07/tagged.jpg');
  await expect(page.getByTestId('detail-state')).toHaveText('erledigt');
  await expect(page.getByTestId('detail-place')).toHaveText('Stadtpark, Graz');
  await expect(page.getByTestId('detail-tags')).toContainText('Geburtstag');
  await expect(page.getByTestId('detail-taken')).toContainText('14. Juli 2024');
  await expect(page.getByText(/Datum aus/)).toHaveCount(0);
  const img = detail.locator('img[data-state="loaded"]');
  await expect(img).toBeVisible(settle);
  expect(await img.evaluate((el: HTMLImageElement) => el.naturalWidth)).toBeGreaterThan(0);

  await page.keyboard.press('ArrowRight');
  await expect(path).toHaveText(older.relPath);
  await page.keyboard.press('ArrowLeft');
  await expect(path).toHaveText('2024/07/tagged.jpg');
  await swipe(page, -80, 10);
  await expect(path).toHaveText(older.relPath);
  await swipe(page, 80, -10);
  await expect(path).toHaveText('2024/07/tagged.jpg');
  await swipe(page, 80, 5);
  await expect(path).toHaveText(newer.relPath);
  // Too short and too steep: neither steps.
  await swipe(page, 30);
  await swipe(page, 60, 90);
  await expect(path).toHaveText(newer.relPath);

  await page.keyboard.press('Escape');
  await expect(detail).toBeHidden();
  expect(await scroller.evaluate((el) => el.scrollTop)).toBe(scrolledTo);

  // Browser back closes the view as well.
  await openTile(page, tagged.items[tagged.index].id, 'down');
  await expect(detail).toBeVisible();
  await page.goBack();
  await expect(detail).toBeHidden();

  // A fixture without metadata is dated by its modification time.
  const sample = await findByPath(page.request, '2024/07/sample.jpg');
  await openTile(page, sample.items[sample.index].id, 'up');
  await expect(path).toHaveText('2024/07/sample.jpg');
  await expect(page.getByText('Datum aus der Änderungszeit')).toBeVisible();
  await expect(page.getByTestId('detail-state')).toHaveText('offen');
  await page.getByRole('button', { name: 'Schließen' }).click();
  await expect(detail).toBeHidden();
});
