import { expect, test, type Page } from '@playwright/test';
import { users } from './helpers/auth';
import { bulkPlan } from './helpers/bulk-library';
import { waitForLibrary } from './helpers/library';

const MONTH_NAMES = [
  'Januar', 'Februar', 'März', 'April', 'Mai', 'Juni',
  'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember',
];
const monthLabel = (m: string) => `${MONTH_NAMES[Number(m.slice(5, 7)) - 1]} ${m.slice(0, 4)}`;
const dayLabel = (d: string) => `${Number(d.slice(8, 10))}. ${monthLabel(d.slice(0, 7))}`;

const TILE = '[data-testid="grid-tile"]';
const settle = { timeout: 60_000 };

async function takenOfTiles(page: Page): Promise<string[]> {
  return page.locator(`${TILE}[data-taken]`).evaluateAll((els) => els.map((e) => e.getAttribute('data-taken') ?? ''));
}

/** Number of tiles sharing the top edge of the topmost tile, i.e. the first photo row. */
async function firstRowTiles(page: Page): Promise<number> {
  await expect(page.locator(TILE).first()).toBeVisible(settle);
  return page.locator(TILE).evaluateAll((els) => {
    const tops = els.map((e) => Math.round(e.getBoundingClientRect().top));
    const min = Math.min(...tops);
    return tops.filter((t) => t === min).length;
  });
}

async function noHorizontalScroll(page: Page): Promise<void> {
  const overflow = await page.evaluate(() => {
    const scroller = document.querySelector('[data-testid="grid-scroller"]')!;
    return [
      document.documentElement.scrollWidth - document.documentElement.clientWidth,
      scroller.scrollWidth - scroller.clientWidth,
    ];
  });
  expect(overflow[0]).toBeLessThanOrEqual(0);
  expect(overflow[1]).toBeLessThanOrEqual(0);
}

test('the library grid groups 5,000 photos by month and day, jumps to a month and keeps the page light', async ({ page }) => {
  test.setTimeout(480_000);
  const plan = bulkPlan();
  const newest = plan.months[0];
  const [newestDay, newestDayCount] = [...newest.days][0];
  expect(newestDayCount).toBeGreaterThan(6);

  await page.setViewportSize({ width: 1440, height: 900 });
  await waitForLibrary(page.request);
  const groups = (await (await page.request.get('/api/photos/groups', { headers: { 'X-Requested-With': 'fetch' } })).json()) as {
    months: { month: string }[];
  };
  const oldest = groups.months[groups.months.length - 1].month;

  const photoRequests: string[] = [];
  page.on('request', (r) => {
    const url = new URL(r.url());
    if (url.pathname === '/api/photos') photoRequests.push(url.search);
  });

  // waitForLibrary signed in on the page's own context, so /login would bounce.
  await page.goto('/');
  await expect(page.getByTestId('current-user')).toHaveText(users.anna.name);

  await expect(page.getByRole('heading', { name: monthLabel(newest.month), level: 2 })).toBeVisible(settle);
  await expect(page.getByRole('heading', { name: dayLabel(newestDay), level: 3 })).toBeVisible();
  const counts = page.getByTestId('group-count');
  await expect(counts.nth(0)).toHaveText(String(newest.count));
  await expect(counts.nth(1)).toHaveText(String(newestDayCount));

  await expect.poll(async () => (await takenOfTiles(page)).length, settle).toBeGreaterThanOrEqual(newestDayCount);
  const taken = await takenOfTiles(page);
  expect(taken.every((t) => t.startsWith(newest.month))).toBe(true);
  expect(taken.slice(0, newestDayCount).every((t) => t.startsWith(newestDay))).toBe(true);
  expect([...taken].sort().reverse()).toEqual(taken);

  await page.getByRole('button', { name: 'Zeitraum' }).click();
  await page.getByRole('button', { name: 'März 2023', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'März 2023', level: 2 })).toBeInViewport(settle);
  await expect(page.locator(`${TILE}[data-taken^="2023-03"]`).first()).toBeInViewport(settle);

  const march = plan.months.find((m) => m.month === '2023-03')!;
  const marchHeader = page.getByRole('heading', { name: 'März 2023', level: 2 }).locator('xpath=..');
  await expect(marchHeader.getByTestId('group-count')).toHaveText(String(march.count));

  const requestsBefore = photoRequests.length;
  const scroller = page.getByTestId('grid-scroller');
  for (let step = 0; step < 400; step++) {
    const atEnd = await scroller.evaluate(async (el) => {
      el.scrollTop += el.clientHeight * 3;
      await new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)));
      return el.scrollTop + el.clientHeight >= el.scrollHeight - 1;
    });
    expect(await page.locator(TILE).count()).toBeLessThanOrEqual(300);
    if (atEnd) break;
  }
  expect(photoRequests.length).toBeGreaterThan(requestsBefore);

  await expect(page.locator(`${TILE}:not([data-id])`)).toHaveCount(0, settle);
  await expect.poll(async () => (await takenOfTiles(page)).some((t) => t.startsWith(oldest)), settle).toBe(true);
  expect(await page.locator(TILE).count()).toBeLessThanOrEqual(300);

  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/');
  expect(await firstRowTiles(page)).toBeGreaterThanOrEqual(3);
  await noHorizontalScroll(page);

  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto('/');
  expect(await firstRowTiles(page)).toBeGreaterThanOrEqual(6);
  await noHorizontalScroll(page);
});
