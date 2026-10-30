import { expect, test } from '@playwright/test';
import { login, users } from './helpers/auth';
import { EXPECTED_TOTAL } from './helpers/bulk-library';

interface Scan {
  state: string;
  finished_at: string | null;
  files: number;
  unreadable: { rel_path: string }[];
}

test('the library is read at start and on demand, unreadable files are flagged', async ({ page }) => {
  test.setTimeout(300_000);
  const settle = { timeout: 240_000 };
  const filesText = `${EXPECTED_TOTAL} Dateien`;
  await login(page, users.anna);
  const status = page.getByRole('status');
  await expect(status).toContainText(filesText, settle);

  const unreadable = page.getByRole('listitem').filter({ hasText: '2024/08/defekt.jpg' });
  await expect(unreadable).toContainText('nicht lesbar');
  await expect(page.getByRole('heading', { name: /Nicht lesbar \(\d+\)/ })).toBeVisible();

  const read = async () => (await (await page.request.get('/api/scan')).json()) as Scan;
  const before = await read();
  expect(before.files).toBe(EXPECTED_TOTAL);
  expect(before.finished_at).not.toBeNull();

  await page.getByRole('button', { name: 'Neu einlesen' }).click();

  // finished_at has second resolution, so a quick rescan is only seen by waiting for the new value.
  await expect
    .poll(async () => {
      const s = await read();
      return s.state === 'idle' ? s.finished_at : null;
    })
    .not.toBe(before.finished_at);
  await expect(page.getByRole('button', { name: 'Neu einlesen' })).toBeEnabled();
  await expect(status).toContainText(filesText);
  await expect(unreadable).toContainText('nicht lesbar');
});
