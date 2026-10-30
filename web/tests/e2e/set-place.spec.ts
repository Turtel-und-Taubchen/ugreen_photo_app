import { expect, test } from '@playwright/test';
import { compose, NOMINATIM_MOCK_PORT } from './compose';
import { users } from './helpers/auth';
import { showTile } from './helpers/grid';
import { waitForLibrary } from './helpers/library';
import { stubTiles, watchLeaflet } from './helpers/map';
import { HEADERS, idsByPath } from './helpers/photos';

const MOCK = `http://127.0.0.1:${NOMINATIM_MOCK_PORT}`;
const settle = { timeout: 60_000 };

const SAMPLE = '2024/07/sample.jpg';
const MOVIE = '2024/08/short.mov';
const TAGGED = '2024/07/tagged.jpg';

interface Job {
  id: number;
  status: string;
}

const mockLogLength = async (): Promise<number> => ((await (await fetch(`${MOCK}/__mock/requests`)).json()) as unknown[]).length;

// Files are only readable through the app container: it is the one with exiftool.
const exif = (relPath: string, ...args: string[]): string =>
  compose(['exec', '-T', 'photocurator', 'exiftool', '-s3', ...args, `/photos/${relPath}`]).trim();

const coordinates = (relPath: string): number[] =>
  exif(relPath, '-n', '-GPSLatitude', '-GPSLongitude').split(/\s+/).map(Number);

test('places are set for a selection from search or map, asked before overwriting and removed again', async ({ page }) => {
  test.setTimeout(420_000);
  await waitForLibrary(page.request);
  await fetch(`${MOCK}/__mock/reset`, { method: 'POST' });
  const ids = await idsByPath(page.request);
  const facts = new Map<string, number>();
  for (const rel of [SAMPLE, MOVIE, TAGGED]) {
    const id = ids.get(rel)!;
    facts.set(rel, id);
  }
  expect(exif(SAMPLE, '-GPSLatitude'), 'fixture starts without GPS').toBe('');
  expect(exif(MOVIE, '-Keys:GPSCoordinates'), 'fixture starts without GPS').toBe('');
  const taggedBefore = coordinates(TAGGED);
  expect(taggedBefore[0]).toBeCloseTo(47.0707, 3);
  expect(taggedBefore[1]).toBeCloseTo(15.4395, 3);

  const tiles = await stubTiles(page);
  const leafletLoaded = watchLeaflet(page);
  const counter = page.getByTestId('selection-count');
  const detail = page.getByTestId('photo-detail');

  async function latestJobId(): Promise<number> {
    const jobs = (await (await page.request.get('/api/jobs', { headers: HEADERS })).json()) as Job[];
    return Math.max(0, ...jobs.map((j) => j.id));
  }

  async function jobFinishedAfter(previous: number): Promise<void> {
    await expect
      .poll(async () => {
        const jobs = (await (await page.request.get('/api/jobs', { headers: HEADERS })).json()) as Job[];
        const fresh = jobs.filter((j) => j.id > previous);
        return fresh.length > 0 && fresh.every((j) => j.status === 'done' || j.status === 'aborted');
      }, settle)
      .toBe(true);
  }

  async function select(rels: string[]) {
    await page.goto('/');
    await expect(page.getByTestId('current-user')).toHaveText(users.anna.name);
    await page.getByRole('button', { name: 'Auswählen', exact: true }).click();
    for (const rel of rels) {
      await (await showTile(page, facts.get(rel)!, rel)).click();
    }
    await expect(counter).toHaveText(`${rels.length} ausgewählt`);
  }

  async function expectPlaces(expected: Record<string, string>) {
    await page.goto('/');
    for (const [rel, text] of Object.entries(expected)) {
      await (await showTile(page, facts.get(rel)!, rel)).click();
      await expect(page.getByTestId('detail-path')).toHaveText(rel);
      await expect(page.getByTestId('detail-place')).toHaveText(text, settle);
      await page.keyboard.press('Escape');
      await expect(detail).toBeHidden();
    }
  }

  await page.goto('/');
  await expect(page.getByTestId('current-user')).toHaveText(users.anna.name);
  expect(await leafletLoaded()).toBe(false);

  // Search hit; the one photo that already has a place is left alone.
  await select([SAMPLE, MOVIE, TAGGED]);
  await page.getByRole('button', { name: 'Ort setzen' }).click();
  const dialog = page.getByRole('dialog', { name: 'Ort setzen' });
  await expect(dialog.getByRole('region', { name: 'Karte' })).toBeVisible();
  await expect(dialog.getByRole('link', { name: '© OpenStreetMap-Mitwirkende' })).toBeVisible();
  await expect.poll(tiles.count).toBeGreaterThan(0);
  expect(await leafletLoaded()).toBe(true);
  await dialog.getByLabel('Ort suchen').fill('Ritzenhofer Saalfelden');
  await dialog.getByLabel('Ort suchen').press('Enter');
  await dialog.getByRole('list', { name: 'Treffer' }).getByRole('listitem').first().getByRole('button').click();
  await expect(page.locator('.pc-marker')).toBeVisible();
  let before = await latestJobId();
  await dialog.getByRole('button', { name: 'Übernehmen' }).click();
  await expect(dialog).toContainText('1 von 3 ausgewählten Fotos haben bereits einen Ort.');
  await dialog.getByRole('button', { name: 'Nur Fotos ohne Ort' }).click();
  await expect(dialog).toBeHidden();
  await jobFinishedAfter(before);

  await expectPlaces({ [SAMPLE]: 'Ritzenhofer, Saalfelden', [MOVIE]: 'Ritzenhofer, Saalfelden', [TAGGED]: 'Stadtpark, Graz' });
  const [lat, lon] = coordinates(SAMPLE);
  expect(lat).toBeCloseTo(47.4262, 2);
  expect(lon).toBeCloseTo(12.8481, 2);
  expect(exif(MOVIE, '-Keys:GPSCoordinates')).not.toBe('');
  const taggedAfter = coordinates(TAGGED);
  expect(taggedAfter[0]).toBeCloseTo(taggedBefore[0], 4);
  expect(taggedAfter[1]).toBeCloseTo(taggedBefore[1], 4);

  // Map click with a typed name needs no lookup; both photos have a place now, so overwriting is asked.
  await select([SAMPLE, MOVIE]);
  await page.getByRole('button', { name: 'Ort setzen' }).click();
  const second = page.getByRole('dialog', { name: 'Ort setzen' });
  const map = second.getByRole('region', { name: 'Karte' });
  await expect(map).toBeVisible();
  const lookups = await mockLogLength();
  await map.click({ position: { x: 150, y: 120 } });
  await expect(page.locator('.pc-marker')).toBeVisible();
  await expect(second.getByLabel('Breite')).not.toHaveValue('');
  await expect(second.getByLabel('Länge')).not.toHaveValue('');
  await expect(second.getByRole('button', { name: 'Übernehmen' })).toBeDisabled();
  await second.getByLabel('Ortsname').fill('E2E Kartenpunkt');
  expect(await mockLogLength()).toBe(lookups);
  before = await latestJobId();
  await second.getByRole('button', { name: 'Übernehmen' }).click();
  await expect(second).toContainText('2 von 2 ausgewählten Fotos haben bereits einen Ort.');
  await second.getByRole('button', { name: 'Alle überschreiben' }).click();
  await expect(second).toBeHidden();
  await jobFinishedAfter(before);
  await expectPlaces({ [SAMPLE]: 'E2E Kartenpunkt', [MOVIE]: 'E2E Kartenpunkt' });

  // Removing restores the fixtures' start state.
  await select([SAMPLE, MOVIE]);
  await page.getByRole('button', { name: 'Ort entfernen' }).click();
  const confirm = page.getByRole('dialog', { name: 'Ort aus 2 Fotos entfernen?' });
  before = await latestJobId();
  await confirm.getByRole('button', { name: 'Entfernen' }).click();
  await expect(confirm).toBeHidden();
  await jobFinishedAfter(before);
  await expectPlaces({ [SAMPLE]: 'kein Ort', [MOVIE]: 'kein Ort' });
  for (const rel of [SAMPLE, MOVIE]) {
    expect(exif(rel, '-GPS:all', '-Keys:GPSCoordinates', '-City', '-State', '-Country', '-Sub-location', '-Location'), `${rel} place fields`).toBe('');
  }
});
