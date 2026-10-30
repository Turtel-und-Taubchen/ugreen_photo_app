import { expect, type APIRequestContext } from '@playwright/test';
import { EXPECTED_TOTAL } from './bulk-library';
import { users } from './auth';

/** Logs in on the request context and waits until the first scan has indexed the whole library. */
export async function waitForLibrary(request: APIRequestContext): Promise<void> {
  const login = await request.post('/api/login', {
    headers: { 'X-Requested-With': 'fetch' },
    data: { username: users.anna.name, password: users.anna.password },
  });
  expect(login.ok()).toBe(true);
  await expect
    .poll(
      async () => {
        const r = await request.get('/api/photos/groups', { headers: { 'X-Requested-With': 'fetch' } });
        return r.ok() ? ((await r.json()) as { total: number }).total : -1;
      },
      { timeout: 240_000, intervals: [1000, 2000, 5000] },
    )
    .toBe(EXPECTED_TOTAL);
}
