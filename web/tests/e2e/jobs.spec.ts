import { expect, test, type APIRequestContext } from '@playwright/test';
import { users } from './helpers/auth';
import { waitForLibrary } from './helpers/library';
import { HEADERS, idsByPath } from './helpers/photos';

const FIRST_TAG = 'Anlass/Rueckgaengig';
const SECOND_TAG = 'Anlass/Konflikt';
const FILES = ['2024/07/sample.jpg', '2024/07/tagged.jpg', '2024/08/20240815_143005.jpg'];

async function tagJob(request: APIRequestContext, ids: number[], tag: string) {
  const res = await request.post('/api/jobs', {
    headers: HEADERS,
    data: { selection: { filter: { groups: [], ranges: [] }, include_ids: ids, exclude_ids: [] }, action: 'tags.add', payload: { tags: [tag] } },
  });
  expect(res.status()).toBe(202);
  return ((await res.json()) as { id: number }).id;
}

async function tagsOf(request: APIRequestContext, id: number): Promise<string[]> {
  return ((await (await request.get(`/api/photos/${id}`, { headers: HEADERS })).json()) as { tags: string[] }).tags;
}

test('a job is undone from the job list', async ({ page }) => {
  test.setTimeout(300_000);
  const request = page.request;
  await waitForLibrary(request);
  const all = await idsByPath(request);
  const [a, b, conflicted] = FILES.map((p) => all.get(p)!);

  const first = await tagJob(request, [a, b, conflicted], FIRST_TAG);
  await expect.poll(async () => (await tagsOf(request, conflicted)).includes(FIRST_TAG)).toBe(true);
  const second = await tagJob(request, [conflicted], SECOND_TAG);
  await expect.poll(async () => (await tagsOf(request, conflicted)).includes(SECOND_TAG)).toBe(true);

  await page.goto('/jobs');
  const firstRow = page.locator(`[data-job-id="${first}"]`);
  const secondRow = page.locator(`[data-job-id="${second}"]`);
  await expect(firstRow).toContainText('Tags hinzufügen');
  await expect(firstRow).toContainText('3 Dateien');
  await expect(firstRow).toContainText('3 erfolgreich · 0 übersprungen · 0 fehlgeschlagen');
  await expect(firstRow.locator('time')).not.toBeEmpty();
  await expect(secondRow).toContainText('1 Datei');
  await expect(page.getByTestId('current-user')).toHaveText(users.anna.name);
  await expect(firstRow).toContainText(users.anna.name);
  await expect(secondRow).toContainText(users.anna.name);

  await firstRow.getByRole('button', { name: 'Rückgängig' }).click();
  await firstRow.getByRole('button', { name: 'Ja, rückgängig' }).click();

  const undoRow = page.locator('li[data-job-id]').filter({ hasText: 'Rückgängig: Tags hinzufügen' }).first();
  await expect(undoRow).toContainText('2 erfolgreich · 1 übersprungen');
  await undoRow.getByText('Ergebnis je Datei').click();
  await expect(undoRow.getByRole('listitem').filter({ hasText: FILES[2] })).toContainText('Konflikt');

  await expect(firstRow.getByRole('button', { name: 'Rückgängig' })).toHaveCount(0);
  expect(await tagsOf(request, a)).not.toContain(FIRST_TAG);
  expect(await tagsOf(request, b)).not.toContain(FIRST_TAG);
  expect(await tagsOf(request, conflicted)).toContain(FIRST_TAG);
});
