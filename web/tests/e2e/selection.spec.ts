import { expect, test, type Locator } from '@playwright/test';
import { users } from './helpers/auth';
import { bulkPlan } from './helpers/bulk-library';
import { waitForLibrary } from './helpers/library';

const MONTH_NAMES = [
  'Januar', 'Februar', 'März', 'April', 'Mai', 'Juni',
  'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember',
];
const monthLabel = (m: string) => `${MONTH_NAMES[Number(m.slice(5, 7)) - 1]} ${m.slice(0, 4)}`;
const settle = { timeout: 60_000 };

test('selecting photos one by one, as a range and as a whole month survives scrolling and the detail view', async ({ page }) => {
  test.setTimeout(480_000);
  await waitForLibrary(page.request);
  await page.goto('/');
  await expect(page.getByTestId('current-user')).toHaveText(users.anna.name);

  const big = bulkPlan().months.reduce((a, b) => (b.count > a.count ? b : a));
  const label = monthLabel(big.month);
  const heading = page.getByRole('heading', { name: label, level: 2 });
  const head = page.locator('.head').filter({ has: heading });
  const monthTiles = page.locator(`[data-testid="grid-tile"][data-taken^="${big.month}"]`);
  const counter = page.getByTestId('selection-count');
  const bar = page.getByRole('region', { name: 'Auswahl' });
  const scroller = page.getByTestId('grid-scroller');

  async function showBigMonth(): Promise<void> {
    await page.getByRole('button', { name: 'Zeitraum' }).click();
    await page.getByRole('button', { name: label, exact: true }).click();
    await expect(heading).toBeInViewport(settle);
    await expect(monthTiles.first()).toBeVisible(settle);
  }
  const pressed = (tile: Locator) => tile.getAttribute('aria-pressed');

  await showBigMonth();
  // The count is read from the heading, not hardcoded.
  const total = Number(await head.getByTestId('group-count').innerText());
  expect(total).toBe(big.count);
  expect(total).toBeGreaterThanOrEqual(400);

  await page.getByRole('button', { name: 'Auswählen', exact: true }).click();
  await expect(counter).toHaveText('0 ausgewählt');

  const [t1, t2, , , t5] = [0, 1, 2, 3, 4].map((i) => monthTiles.nth(i));
  await t1.click();
  await t2.click();
  await expect(counter).toHaveText('2 ausgewählt');
  await expect(t1).toHaveAttribute('aria-pressed', 'true');
  await t2.click();
  await expect(counter).toHaveText('1 ausgewählt');
  await expect(t2).toHaveAttribute('aria-pressed', 'false');

  // The anchor stays on T2, so the range runs T2..T5 and T1 is still included.
  await t5.click({ modifiers: ['Shift'] });
  await expect(counter).toHaveText('5 ausgewählt');
  for (let i = 0; i < 5; i++) await expect(monthTiles.nth(i)).toHaveAttribute('aria-pressed', 'true');

  await page.getByRole('button', { name: 'Auswahl aufheben' }).click();
  await expect(bar).toBeHidden();

  // The month does not fit into the DOM, so the selection cannot live in the tiles.
  expect(await monthTiles.count()).toBeLessThan(total);
  await head.getByRole('button', { name: 'Alle auswählen' }).click();
  await expect(counter).toHaveText(`${total} ausgewählt`);
  await expect(head.getByRole('button', { name: 'Alle auswählen' })).toBeDisabled();

  const victim = monthTiles.nth(3);
  const victimId = await victim.getAttribute('data-id');
  const request = page.waitForRequest((r) => {
    if (!r.url().endsWith('/api/selection/count')) return false;
    const body = r.postDataJSON() as { exclude_ids: number[] };
    return body.exclude_ids.length === 1;
  });
  await victim.click();
  const body = (await request).postDataJSON() as {
    filter: { groups: string[] };
    include_ids: number[];
    exclude_ids: number[];
  };
  expect(body.filter.groups).toEqual([big.month]);
  expect(body.include_ids).toEqual([]);
  expect(body.exclude_ids).toHaveLength(1);
  await expect(counter).toHaveText(`${total - 1} ausgewählt`);

  // Scroll far away and back: virtualization rebuilds the tiles, the marks come from the store.
  await scroller.evaluate((el) => (el.scrollTop = el.scrollHeight));
  await expect(monthTiles).toHaveCount(0, settle);
  await showBigMonth();
  await expect(page.locator(`[data-testid="grid-tile"][data-id="${victimId}"]`)).toHaveAttribute('aria-pressed', 'false');
  await expect(monthTiles.first()).toHaveAttribute('aria-pressed', 'true');
  expect(await pressed(monthTiles.nth(4))).toBe('true');

  await page.getByRole('button', { name: 'Details öffnen' }).first().click();
  await expect(page.getByTestId('photo-detail')).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(page.getByTestId('photo-detail')).toBeHidden();
  await expect(counter).toHaveText(`${total - 1} ausgewählt`);
  await expect(monthTiles.first()).toHaveAttribute('aria-pressed', 'true');

  await page.getByRole('button', { name: 'Auswahl aufheben' }).click();
  await expect(bar).toBeHidden();
});

test.describe('on a phone', () => {
  test.use({ viewport: { width: 390, height: 844 }, hasTouch: true });

  test("on a phone a long press starts selection and 'Bereich bis hier' selects the range", async ({ page }) => {
    test.setTimeout(240_000);
    await waitForLibrary(page.request);
    await page.goto('/');
    await expect(page.getByTestId('current-user')).toHaveText(users.anna.name);

    const big = bulkPlan().months.reduce((a, b) => (b.count > a.count ? b : a));
    const label = monthLabel(big.month);
    const monthTiles = page.locator(`[data-testid="grid-tile"][data-taken^="${big.month}"]`);
    const counter = page.getByTestId('selection-count');

    await page.getByRole('button', { name: 'Zeitraum' }).click();
    await page.getByRole('button', { name: label, exact: true }).click();
    await expect(monthTiles.first()).toBeVisible(settle);

    async function longPress(tile: Locator): Promise<void> {
      await tile.hover();
      await page.mouse.down();
      await page.waitForTimeout(600);
      await page.mouse.up();
    }

    const [t1, , , t4] = [0, 1, 2, 3].map((i) => monthTiles.nth(i));
    await longPress(t1);
    await expect(counter).toHaveText('1 ausgewählt');
    await expect(t1).toHaveAttribute('aria-pressed', 'true');

    await longPress(t4);
    const item = page.getByRole('menuitem', { name: 'Bereich bis hier' });
    await expect(item).toBeEnabled();
    expect((await item.boundingBox())!.height).toBeGreaterThanOrEqual(44);
    await item.click();
    await expect(counter).toHaveText('4 ausgewählt');
    for (let i = 0; i < 4; i++) await expect(monthTiles.nth(i)).toHaveAttribute('aria-pressed', 'true');
    await expect(item).toBeHidden();
  });
});
