import { expect, test, type Page } from '@playwright/test';
import { compose } from './compose';
import { users } from './helpers/auth';
import { scrollToTile } from './helpers/grid';
import { waitForLibrary } from './helpers/library';
import { HEADERS, idsByPath } from './helpers/photos';

const FILES = ['2024/07/tagged.jpg', '2024/07/tagged.heic'];
const BATCH = 'E2E/Batch';
const NEU = 'E2E/Neu';
const settle = { timeout: 60_000 };

interface PhotoFacts {
  taken_at: string;
  tags: string[];
}

const monthLabel = (takenAt: string) =>
  new Date(`${takenAt.slice(0, 7)}-15T12:00:00`).toLocaleDateString('de-DE', { month: 'long', year: 'numeric' });

async function showTile(page: Page, id: number, takenAt: string) {
  await page.getByRole('button', { name: 'Zeitraum' }).click();
  await page.getByRole('button', { name: monthLabel(takenAt), exact: true }).click();
  return scrollToTile(page, id, 'down');
}

// Files are written through the app container: it is the only place with exiftool.
function fileTags(relPath: string): string {
  return compose(['exec', '-T', 'photocurator', 'exiftool', '-s3', '-XMP-lr:HierarchicalSubject', `/photos/${relPath}`]);
}

// Every keyword field the app reads, as exiftool returns it for the fixture (absent ones print nothing).
const KEYWORD_FIELDS = ['XMP-dc:Subject', 'XMP-lr:HierarchicalSubject', 'IPTC:Keywords', 'EXIF:XPKeywords', 'Keys:Keywords', 'UserData:Keywords'];

function keywordFields(relPath: string): Record<string, string> {
  const out: Record<string, string> = {};
  for (const f of KEYWORD_FIELDS) {
    out[f] = compose(['exec', '-T', 'photocurator', 'exiftool', '-s3', '-sep', ';', `-${f}`, `/photos/${relPath}`]);
  }
  return out;
}

const hasTerm = (value: string, term: string) => value.split(/[;|\n]/).some((t) => t.trim() === term);

test('assigning and removing tags on a selection shows in the detail view and in the files', async ({ page }) => {
  test.setTimeout(420_000);
  await waitForLibrary(page.request);
  const ids = await idsByPath(page.request);
  const photos = FILES.map((relPath) => ({ relPath, id: ids.get(relPath)! }));
  const facts = new Map<number, PhotoFacts>();
  for (const p of photos) {
    facts.set(p.id, (await (await page.request.get(`/api/photos/${p.id}`, { headers: HEADERS })).json()) as PhotoFacts);
  }

  const counter = page.getByTestId('selection-count');
  const detail = page.getByTestId('photo-detail');
  const result = page.getByRole('region', { name: 'Schreibjobs' });

  async function selectBoth() {
    await page.goto('/');
    await expect(page.getByTestId('current-user')).toHaveText(users.anna.name);
    await page.getByRole('button', { name: 'Auswählen', exact: true }).click();
    for (const p of photos) await (await showTile(page, p.id, facts.get(p.id)!.taken_at)).click();
    await expect(counter).toHaveText('2 ausgewählt');
  }

  try {
    await selectBoth();
    await page.getByRole('button', { name: 'Tags zuweisen' }).click();
    const assign = page.getByRole('dialog', { name: 'Tags zuweisen' });
    for (const tag of [BATCH, NEU]) {
      await assign.getByRole('combobox', { name: 'Tag suchen oder neu anlegen' }).fill(tag);
      await page.keyboard.press('Enter');
      await expect(assign.getByRole('button', { name: `${tag} entfernen` })).toBeVisible();
    }
    await assign.getByRole('button', { name: 'Zuweisen' }).click();
    await expect(assign).toBeHidden();
    await expect(result).toContainText('2 erfolgreich', settle);

    for (const p of photos) {
      await expect.poll(async () => (await (await page.request.get(`/api/photos/${p.id}`, { headers: HEADERS })).json()).tags, settle).toEqual(
        expect.arrayContaining([BATCH, NEU]),
      );
      const written = fileTags(p.relPath);
      expect(written).toContain('E2E|Batch');
      expect(written).toContain('E2E|Neu');
      const subjects = keywordFields(p.relPath)['XMP-dc:Subject'];
      for (const term of ['E2E', 'Batch', 'Neu']) expect(hasTerm(subjects, term), `dc:Subject has ${term}`).toBe(true);
    }

    await page.goto('/');
    for (const p of photos) {
      await (await showTile(page, p.id, facts.get(p.id)!.taken_at)).click();
      await expect(page.getByTestId('detail-path')).toHaveText(p.relPath);
      await expect(page.getByTestId('detail-tags')).toContainText(BATCH);
      await expect(page.getByTestId('detail-tags')).toContainText(NEU);
      await page.keyboard.press('Escape');
      await expect(detail).toBeHidden();
    }

    await page.getByRole('link', { name: 'Tags' }).click();
    await expect(page.getByText(/^Neu \(\d+\)$/)).toBeVisible();

    await page.goto('/');
    await selectBoth();
    await page.getByRole('button', { name: 'Tags entfernen' }).click();
    const remove = page.getByRole('dialog', { name: 'Tags entfernen' });
    await expect(remove.getByLabel('E2E/Batch (2 Fotos)')).toBeVisible();
    await expect(remove.getByRole('button', { name: 'Entfernen' })).toBeDisabled();
    await remove.getByLabel('E2E/Batch (2 Fotos)').check();
    await remove.getByLabel('E2E/Neu (2 Fotos)').check();
    await remove.getByRole('button', { name: 'Entfernen' }).click();
    await expect(remove).toBeHidden();

    for (const p of photos) {
      await expect.poll(async () => (await (await page.request.get(`/api/photos/${p.id}`, { headers: HEADERS })).json()).tags, settle).not.toEqual(
        expect.arrayContaining([BATCH]),
      );
      await expect.poll(async () => (await (await page.request.get(`/api/photos/${p.id}`, { headers: HEADERS })).json()).tags, settle).not.toEqual(
        expect.arrayContaining([NEU]),
      );
      expect(fileTags(p.relPath)).not.toContain('E2E|');
      for (const [field, value] of Object.entries(keywordFields(p.relPath))) {
        for (const term of ['E2E', 'Batch', 'Neu']) expect(hasTerm(value, term), `${field} still has ${term}`).toBe(false);
      }
    }
    await page.goto('/');
    for (const p of photos) {
      await (await showTile(page, p.id, facts.get(p.id)!.taken_at)).click();
      await expect(page.getByTestId('detail-path')).toHaveText(p.relPath);
      await expect(page.getByTestId('detail-tags')).not.toContainText('E2E');
      await page.keyboard.press('Escape');
      await expect(detail).toBeHidden();
    }
  } finally {
    // The created definition would otherwise leave an empty category for the other specs.
    for (const path of [BATCH, NEU, 'E2E']) {
      await page.request.delete(`/api/tags?path=${encodeURIComponent(path)}`, { headers: HEADERS });
    }
  }
});
