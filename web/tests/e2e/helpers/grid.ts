import type { Locator, Page } from '@playwright/test';

/** Scrolls the virtualized grid in one direction until the tile of `id` is rendered, and returns it. */
export async function scrollToTile(page: Page, id: number, direction: 'down' | 'up'): Promise<Locator> {
  const scroller = page.getByTestId('grid-scroller');
  const tile = page.locator(`[data-testid="grid-tile"][data-id="${id}"]`);
  if (direction === 'up') await scroller.evaluate((el) => (el.scrollTop = el.scrollHeight));
  for (let step = 0; step < 600 && (await tile.count()) === 0; step++) {
    await scroller.evaluate((el, dir) => el.scrollBy(0, dir * el.clientHeight * 0.7), direction === 'down' ? 1 : -1);
    await page.waitForTimeout(40);
  }
  return tile;
}

/** Brings the tile of `id` into view and clicks it. */
export async function openTile(page: Page, id: number, direction: 'down' | 'up'): Promise<void> {
  await (await scrollToTile(page, id, direction)).click();
}
