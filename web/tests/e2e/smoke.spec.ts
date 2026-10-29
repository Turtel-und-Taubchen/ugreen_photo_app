import { expect, test } from '@playwright/test';
import { compose } from './compose';
import { login, users } from './helpers/auth';

test('the start page shows the app name and version', async ({ page, request }) => {
  const health = (await (await request.get('/api/health')).json()) as { version: string };
  await login(page, users.anna);
  await expect(page.getByRole('heading', { name: 'PhotoCurator' })).toBeVisible();
  await expect(page.getByText(`Version ${health.version}`)).toBeVisible();
  await expect(page.getByText('Version 0.1.0-dev')).toBeVisible();
});

test('the app container runs as non-root without build tools', () => {
  const out = compose([
    'exec', '-T', 'photocurator', 'sh', '-c',
    'id -u; command -v go node npm; test -d /usr/local/go && echo has-go; true',
  ]);
  const lines = out.split(/\r?\n/).filter((l) => l !== '');
  expect(Number(lines[0])).toBeGreaterThan(0);
  expect(lines).toHaveLength(1);
});
