import { expect, test } from '@playwright/test';
import { login, users } from './helpers/auth';

test('the tag page shows file tags and empty categories as a tree and guards deletion', async ({ page }) => {
  test.setTimeout(300_000);
  await login(page, users.anna);
  await page.getByRole('link', { name: 'Tags' }).click();
  await expect(page.getByRole('heading', { name: 'Tags' })).toBeVisible();

  // The first scan may still be running: reload until the file tags show up.
  const birthday = page.getByText(/^Geburtstag \(\d+\)$/);
  await expect(async () => {
    await page.reload();
    await expect(birthday).toBeVisible({ timeout: 3_000 });
  }).toPass({ timeout: 240_000, intervals: [2_000, 5_000] });
  await expect(page.getByText(/^Anlass \(\d+\)$/)).toBeVisible();
  await expect(birthday).not.toHaveText('Geburtstag (0)');

  const field = page.getByLabel('Neuer Tag');
  await field.fill('Ausflug/Wandern');
  await page.getByRole('button', { name: 'Anlegen' }).click();
  await expect(page.getByText('Ausflug (0)')).toBeVisible();
  await expect(page.getByText('Wandern (0)')).toBeVisible();
  await expect(field).toHaveValue('');

  await field.fill('Ausflug/Wan|dern');
  await field.press('Enter');
  await expect(page.getByRole('alert')).toContainText('|');

  await page.getByRole('button', { name: 'Wandern löschen' }).click();
  await expect(page.getByText('Wandern (0)')).toHaveCount(0);
  await expect(page.getByText('Ausflug (0)')).toBeVisible();

  await expect(page.getByRole('button', { name: 'Geburtstag löschen' })).toBeDisabled();
  await expect(page.getByText(/Tags entfernen/).first()).toBeVisible();

  await page.reload();
  await expect(page.getByText('Ausflug (0)')).toBeVisible();
});
