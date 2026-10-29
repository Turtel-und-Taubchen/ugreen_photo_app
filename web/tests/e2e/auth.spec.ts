import { expect, test, type Page } from '@playwright/test';
import { login, users } from './helpers/auth';

async function expectLoginRedirects(page: Page) {
  for (const path of ['/', '/albums']) {
    await page.goto(path);
    await expect(page).toHaveURL(/\/login$/);
  }
}

test('two users sign in, see their name and sign out', async ({ page, context }) => {
  await expectLoginRedirects(page);

  const failedMessage = 'Anmeldung fehlgeschlagen. Name oder Passwort falsch.';
  const attempt = async (name: string, password: string) => {
    await page.goto('/login'); // fresh page: no stale alert from an earlier attempt
    await expect(page.getByRole('alert')).toHaveCount(0);
    await page.getByLabel('Name').fill(name);
    await page.getByLabel('Passwort').fill(password);
    await page.getByRole('button', { name: 'Anmelden' }).click();
    await expect(page.getByRole('alert')).toHaveText(failedMessage);
    return (await page.getByRole('alert').textContent())?.trim();
  };
  const wrongPassword = await attempt('anna', 'falsch');
  const unknownName = await attempt('niemand', 'falsch');
  expect(unknownName).toBe(wrongPassword);

  for (const user of [users.anna, users.ben]) {
    await login(page, user);
    const cookie = (await context.cookies()).find((c) => c.name === 'pc_session');
    expect(cookie?.httpOnly).toBe(true);
    expect(cookie?.sameSite).toBe('Strict');
    await page.getByRole('button', { name: 'Abmelden' }).click();
    await expect(page).toHaveURL(/\/login$/);
    await page.goto('/');
    await expect(page).toHaveURL(/\/login$/);
  }
});
