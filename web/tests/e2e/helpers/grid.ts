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

const TAGGED = '2024/07/tagged.jpg';

// The month picker lists folders (YYYY/MM), not dates: the fixtures without EXIF date sit in a 2022 group but in their 2024 folder.
export const folderLabel = (rel: string) => {
  const [year, month] = rel.split('/');
  return new Date(`${year}-${month}-15T12:00:00`).toLocaleDateString('de-DE', { month: 'long', year: 'numeric' });
};

/** Jumps to the photo's folder and returns its tile; the two files without EXIF date are dated by mtime (2022) and sit at the very end of the grid. */
export async function showTile(page: Page, id: number, rel: string): Promise<Locator> {
  await page.getByRole('button', { name: 'Zeitraum' }).click();
  await page.getByRole('button', { name: folderLabel(rel), exact: true }).click();
  return scrollToTile(page, id, rel === TAGGED ? 'down' : 'up');
}
