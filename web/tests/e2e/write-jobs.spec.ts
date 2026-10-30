import { expect, test } from "@playwright/test";
import { waitForLibrary } from "./helpers/library";
import { HEADERS, idsByPath } from "./helpers/photos";

const BAD = "2024/08/defekt.jpg";
// A tag no fixture carries, so every good file really changes (a file that is already up to date is skipped).
const TAG = "E2E/Schreibjob";

test("a write job reports its result per file and survives a reload", async ({
  page,
}) => {
  test.setTimeout(300_000);
  const request = page.request;
  await waitForLibrary(request);
  const ids = await idsByPath(request);
  const good = ["2024/07/sample.jpg", "2024/07/tagged.jpg"].map((p) =>
    ids.get(p)!,
  );
  const bad = ids.get(BAD)!;
  expect([...ids.keys()].sort()).toContain(BAD);

  try {
    const created = await request.post("/api/jobs", {
      headers: HEADERS,
      data: {
        selection: {
          filter: { groups: [], ranges: [] },
          include_ids: [...good, bad],
          exclude_ids: [],
        },
        action: "tags.add",
        payload: { tags: [TAG] },
      },
    });
    expect(created.status()).toBe(202);

    // waitForLibrary already logged this context in.
    await page.goto("/");
    await page.reload();
    const jobs = page.getByRole("region", { name: "Schreibjobs" });
    await expect(jobs).toContainText("2 erfolgreich, 1 fehlgeschlagen");
    await jobs.getByText("Details").click();
    // The separator is followed by the reason text.
    await expect(
      jobs.getByRole("listitem").filter({ hasText: BAD }),
    ).toContainText(/·\s*\S+/);
    await expect(jobs).toContainText(
      "UGREEN Fotos zeigt nur den Ort an – nach ‚Aktualisieren' in UGREEN Fotos. Tags und Prüfstatus sind nur in PhotoCurator sichtbar.",
    );

    const photo = (await (
      await request.get(`/api/photos/${good[0]}`, { headers: HEADERS })
    ).json()) as { tags: string[] };
    expect(photo.tags).toContain(TAG);

    // Writing the same tag again changes nothing: the file is skipped, not rewritten.
    const again = await request.post("/api/jobs", {
      headers: HEADERS,
      data: {
        selection: {
          filter: { groups: [], ranges: [] },
          include_ids: [good[0]],
          exclude_ids: [],
        },
        action: "tags.add",
        payload: { tags: [TAG] },
      },
    });
    expect(again.status()).toBe(202);
    await expect(jobs).toContainText("1 übersprungen", { timeout: 60_000 });
  } finally {
    for (const path of [TAG, "E2E"]) {
      await request.delete(`/api/tags?path=${encodeURIComponent(path)}`, {
        headers: HEADERS,
      });
    }
  }
});
