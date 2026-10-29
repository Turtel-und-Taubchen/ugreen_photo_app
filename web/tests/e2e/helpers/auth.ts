import { expect, type Page } from '@playwright/test';

export const users = {
  anna: { name: 'anna', password: 'anna-e2e-pass' },
  ben: { name: 'ben', password: 'ben-e2e-pass' },
} as const;

export async function login(page: Page, user: { name: string; password: string }): Promise<void> {
  await page.goto('/login');
  await page.getByLabel('Name').fill(user.name);
  await page.getByLabel('Passwort').fill(user.password);
  await page.getByRole('button', { name: 'Anmelden' }).click();
  await expect(page.getByTestId('current-user')).toHaveText(user.name);
}
