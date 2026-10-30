import { expect, test, type Locator } from '@playwright/test';
import { NOMINATIM_MOCK_PORT } from './compose';
import { users } from './helpers/auth';
import { showTile } from './helpers/grid';
import { waitForLibrary } from './helpers/library';
import { stubTiles } from './helpers/map';
import { HEADERS, idsByPath } from './helpers/photos';

const MOCK = `http://127.0.0.1:${NOMINATIM_MOCK_PORT}`;
const SAMPLE = '2024/07/sample.jpg';

test('favourite places are saved, reused without Nominatim, renamed and deleted', async ({ page }) => {
  test.setTimeout(300_000);
  await waitForLibrary(page.request);
  const existing = (await (await page.request.get('/api/places', { headers: HEADERS })).json()) as { places: { id: number }[] | null };
  for (const p of existing.places ?? []) {
    await page.request.delete(`/api/places/${p.id}`, { headers: HEADERS });
  }
  const id = (await idsByPath(page.request)).get(SAMPLE)!;
  await stubTiles(page);

  async function openPicker(): Promise<Locator> {
    await page.goto('/');
    await expect(page.getByTestId('current-user')).toHaveText(users.anna.name);
    await page.getByRole('button', { name: 'Auswählen', exact: true }).click();
    await (await showTile(page, id, SAMPLE)).click();
    await page.getByRole('button', { name: 'Ort setzen' }).click();
    const dialog = page.getByRole('dialog', { name: 'Ort setzen' });
    await expect(dialog.getByRole('region', { name: 'Karte' })).toBeVisible();
    return dialog;
  }
  // Closing without Übernehmen writes nothing.
  async function closeDialog(dialog: Locator) {
    await dialog.getByRole('button', { name: 'Abbrechen', exact: true }).last().click();
    await expect(dialog).toBeHidden();
  }

  // Save a searched place as favourite under an own name.
  let dialog = await openPicker();
  await dialog.getByLabel('Ort suchen').fill('Ritzenhofer Saalfelden');
  await dialog.getByLabel('Ort suchen').press('Enter');
  await dialog.getByRole('list', { name: 'Treffer' }).getByRole('listitem').first().getByRole('button').click();
  await expect(page.locator('.pc-marker')).toBeVisible();
  await dialog.getByRole('button', { name: 'Als Lieblingsort speichern' }).click();
  const nameField = dialog.getByLabel('Name des Lieblingsorts');
  await expect(nameField).toHaveValue('Ritzenhofer, Saalfelden');
  await nameField.fill('Stammhotel');
  await dialog.getByRole('button', { name: 'Speichern', exact: true }).click();
  await expect(dialog.getByRole('list', { name: 'Lieblingsorte' })).toContainText('Stammhotel');
  await closeDialog(dialog);

  // Reuse needs no geo lookup at all.
  await fetch(`${MOCK}/__mock/reset`, { method: 'POST' });
  const geoRequests: string[] = [];
  page.on('request', (r) => {
    // The dialog's own status probe on opening is not a lookup.
    if (new URL(r.url()).pathname.startsWith('/api/geo/') && !r.url().endsWith('/api/geo/status')) geoRequests.push(r.url());
  });
  dialog = await openPicker();
  const favs = dialog.getByRole('list', { name: 'Lieblingsorte' });
  await favs.getByRole('button', { name: /^Stammhotel/ }).click();
  await expect(dialog.getByLabel('Ortsname')).toHaveValue('Stammhotel');
  await expect(dialog.getByLabel('Breite')).toHaveValue('47.4262');
  await expect(dialog.getByLabel('Länge')).toHaveValue('12.8481');
  await expect(dialog.locator('dt:text-is("City") + dd')).toHaveText('Saalfelden');
  await expect(dialog.locator('dt:text-is("Country") + dd')).toHaveText('Österreich');
  await expect(page.locator('.pc-marker')).toBeVisible();
  expect(((await (await fetch(`${MOCK}/__mock/requests`)).json()) as unknown[]).length).toBe(0);
  expect(geoRequests).toEqual([]);

  // Rename, then two-step delete.
  await dialog.getByRole('button', { name: '„Stammhotel“ umbenennen' }).click();
  await dialog.getByLabel('Neuer Name').fill('Hotel Ritzenhofer');
  await dialog.getByRole('button', { name: 'Speichern', exact: true }).click();
  await expect(favs).toContainText('Hotel Ritzenhofer');
  await expect(favs).not.toContainText('Stammhotel');

  await dialog.getByRole('button', { name: '„Hotel Ritzenhofer“ löschen' }).click();
  await expect(favs).toContainText('Hotel Ritzenhofer');
  await dialog.getByRole('button', { name: '„Hotel Ritzenhofer“ endgültig löschen' }).click();
  await expect(favs).not.toContainText('Hotel Ritzenhofer');
  await closeDialog(dialog);

  dialog = await openPicker();
  await expect(dialog.getByText('Noch keine Lieblingsorte.')).toBeVisible();
});
