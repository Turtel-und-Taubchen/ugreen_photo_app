---
id: 007
title: Fotoraster mit Gruppierung, Ordnernavigation und Detailansicht
status: done # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E1 Index & Ansicht
estimate: 5
sprint: S02
---

## Requirement

Ich möchte die Bibliothek als Raster sehen, nach Tagen und Monaten gruppiert, schnell zu einem
Monat springen und ein Foto groß ansehen können — am PC und am Handy, auch bei mehreren Tausend
Fotos flüssig. Das Raster ist die Grundlage für Auswahl und Arbeitslisten, keine
Betrachter-Galerie (das bleibt UGREEN Fotos).

## Acceptance Criteria

- [x] **AC1** — Die Startansicht zeigt alle Fotos und Videos nach Aufnahmedatum absteigend,
  gruppiert nach Monat und Tag, jeweils mit Überschrift und Anzahl.
- [x] **AC2** — Eine Navigation nach Jahr/Monat (aus der Ordnerstruktur JJJJ/MM) springt zur
  entsprechenden Gruppe.
- [x] **AC3** — Bei 5.000 Einträgen in der Fixture-Datenbank enthält die Seite zu jedem Zeitpunkt
  höchstens eine begrenzte Zahl Bildelemente (Platzhalter: 300); beim Scrollen werden weitere
  Einträge nachgeladen.
- [x] **AC4** — Ein Klick bzw. Tippen außerhalb des Auswahlmodus öffnet eine Detailansicht mit
  großer Vorschau, Aufnahmedatum (mit Hinweis, wenn es nur aus Dateiname/Änderungszeit stammt),
  Tags, Ortsname, Prüfstatus und Dateipfad.
- [x] **AC5** — In der Detailansicht wechseln Pfeiltasten bzw. Wischen zum vorigen/nächsten Foto.
- [x] **AC6** — Bei 390 px Breite zeigt das Raster mindestens 3 Spalten ohne horizontales
  Scrollen; bei 1440 px Breite mindestens 6 Spalten.

## Open Questions

## Decisions (Sprint)

Keine `(User)`-Vorgaben; alle Punkte vom Refinement entschieden (Grundlage: Tech Spec §2.1, §4,
§6, §7, Konzept PC-51, Stories 005/006/008). Ein `CLAUDE.md` existiert im Repo nicht.

- **007 besitzt die Lese-API** `GET /api/photos/groups`, `GET /api/photos`, `GET /api/photos/{id}`
  (Tech Spec §6) samt Abfragen in `internal/library`; hat 005 schon eine Listen-Abfrage, wird sie
  erweitert statt verdoppelt — 005 baut den Index, 007 ist der erste Leser fürs Raster.
- **Gruppiert wird nach Aufnahmedatum** (`taken_at`, Monat = `JJJJ-MM`, Tag = `JJJJ-MM-TT`),
  absteigend nach `(taken_at, id)` — AC1 verlangt Aufnahmedatum, und `taken_at` ist dank
  mtime-Rückfall (005 AC3) nie leer.
- **Die Navigation (AC2) listet die Ordner `JJJJ/MM`** aus dem Index (andere Ordnernamen werden
  ignoriert) und springt zur Monatsgruppe gleichen Monats, fehlt diese, zur nächstälteren — so
  gilt „aus der Ordnerstruktur" wörtlich, und bei Phockup-Sortierung fallen Ordner und
  Aufnahmemonat ohnehin zusammen.
- **Nachladen per Keyset innerhalb eines Monats** (`?month=JJJJ-MM&after=<cursor>&limit≤500`,
  Standard 200), nie `OFFSET` — Tech Spec §2.1; der Sprung zu einem Monat braucht so keinen
  Rückwärts-Cursor.
- **Das Layout entsteht vollständig aus `/api/photos/groups`** (Anzahl je Tag), Fotodaten kommen
  seitenweise; noch nicht geladene Kacheln sind Platzhalter ohne `<img>` — so stimmen Scrollhöhe
  und Sprungziel, bevor Daten da sind, und 008 kann Gruppen ohne geladene Fotos auswählen.
- **„Bildelement" (AC3) = Rasterkachel** (`data-testid="grid-tile"`, mit oder ohne `<img>`); die
  Obergrenze 300 ist die Konstante `MAX_TILES` — Platzhalterwert aus der Story, so bleibt er
  änderbar.
- **Spalten:** `max(3, floor(Breite / 120 px))`, quadratische Kacheln; würde ein Bildschirm voll
  mehr als 200 Kacheln zeigen (sehr große Monitore), werden die Kacheln größer statt mehr — so
  hält AC3 auch auf 4K, und AC6 (3 bzw. 6 Spalten) bleibt erfüllt.
- **Geladene Seiten werden nicht verworfen** (nur das DOM ist virtualisiert) — die JSON-Zeilen
  sind klein (einige 10 kB je 1.000 Fotos), Verwerfen brächte nur Nachlade-Flackern.
- **Detailansicht als Überlagerung über dem Raster** mit eigenem History-Eintrag (Zurück am
  Handy schließt sie, Esc und ×-Knopf auch) — Scrollposition und spätere Auswahl (008 AC5)
  bleiben so erhalten.
- **Vorheriges/nächstes Foto liefert der Server** (`prev_id`/`next_id` als Keyset-Nachbarn in
  Rasterreihenfolge) — unabhängig davon, welche Seiten der Client gerade geladen hat.
- **Pfeil links / Wischen nach rechts = vorheriges (neueres) Foto, Pfeil rechts / Wischen nach
  links = nächstes (älteres)**; am Ende passiert nichts — gleiche Richtung wie das Raster.
- **Wischen über Pointer Events** (waagrecht ≥ 50 px, überwiegend horizontal) — funktioniert mit
  Finger und Maus und ist im E2E mit echten Ereignissen am Element prüfbar.
- **Videos werden in der Detailansicht nicht abgespielt**, sie zeigen das Vorschaubild (`l`) mit
  Video-Symbol — Abspielen bleibt UGREEN Fotos (Requirement: keine Betrachter-Galerie).
- **Texte:** Datumsquelle `filename` → „Datum aus dem Dateinamen", `mtime` → „Datum aus der
  Änderungszeit", `exif`/`quicktime` ohne Hinweis; Prüfstatus `open/done/irrelevant` →
  „offen/erledigt/nicht relevant"; leer → „keine Tags"/„kein Ort" — deutsche Oberfläche wie Login.
- **Ein Klick auf eine Kachel öffnet in 007 immer die Detailansicht** — den Auswahlmodus gibt es
  erst mit 008, das die Weiche „außerhalb des Auswahlmodus" einbaut.
- **Kacheln nutzen die Thumbnail-Darstellung aus 006** (URL mit Cache-Schlüssel, Platzhalter,
  Fehlerplatzhalter, Video-Symbol) — keine zweite Thumbnail-Komponente.
- **Die 5.000-Einträge-Fixture entsteht im E2E-Setup** aus Kopien der metadatenfreien
  `testdata/library/2024/07/sample.jpg` mit WhatsApp-Dateinamen (`IMG-JJJJMMTT-WA…`) über 24
  Monate in `JJJJ/MM`-Ordnern, Änderungszeit in der Vergangenheit (Ruhezeit aus 005) — echte
  Dateien statt vorbefüllter `index.db`, weil der Scan fehlende Dateien aus dem Index entfernt;
  nichts davon wird eingecheckt.
- **Nach Abschluss eines Scans lädt das Raster die Gruppen neu**, sofern 005 den Scanstatus im
  Frontend bereitstellt — sonst bliebe „Neu einlesen" ohne sichtbare Wirkung.

## Plan

Baut auf 005 (`index.db` mit `photos`/`photo_tags`, Scanstatus) und 006
(`/api/thumbs/{id}/{s|l}`, Thumbnail-Kachel) auf. Reihenfolge D1 → D2 → D3 → D4.

1. **Backend (D1):** `internal/library` bekommt drei Abfragen über `index.db`: Gruppen
   (Monate → Tage mit Anzahl, Ordner `JJJJ/MM` mit Anzahl, Gesamtzahl), Seite (Keyset
   `(taken_at DESC, id DESC)` innerhalb eines Monats) und Detail (Felder + Tags +
   `prev_id`/`next_id`). `internal/api/photos.go` stellt sie als JSON-Routen hinter der
   bestehenden Session-Prüfung bereit.
2. **Rasterlogik (D2):** reine TS-Module ohne DOM — `layout.ts` (Zeilenmodell aus Gruppen +
   Spaltenzahl, Fenster für eine Scrollposition mit Kachel-Obergrenze, Versatz eines Monats) und
   `loader.ts` (Seiten je Monat, Cursor, Deduplizierung, max. 2 parallele Anfragen), mit Vitest.
3. **Raster-UI (D3):** `Library.svelte` als Startansicht, `PhotoGrid.svelte` (absolut
   positionierte Zeilen, nur das Fenster im DOM), `MonthNav.svelte` (Knopf „Zeitraum" → Jahre →
   Monate). Das E2E-Setup erzeugt die 5.000-Dateien-Bibliothek; Flow „Raster" prüft AC1/2/3/6.
4. **Detail (D4):** `PhotoDetail.svelte` als Überlagerung mit großer Vorschau, Datum + Quelle,
   Tags, Ort, Prüfstatus, Pfad; Pfeiltasten, Wischen, Esc, Zurück. Flow „Detail" prüft AC4/5.

Betroffen: `internal/library/`, `internal/api/`, `web/src/lib/grid/`, `web/src/lib/detail/`,
`web/src/routes/Library.svelte`, `web/src/App.svelte`, `web/tests/e2e/`.

Risiken: Der Scan von 5.000 Dateien im E2E über einen Windows-Bind-Mount dauert (Warte-Helfer
mit großzügigem Timeout); bestehende 005/006-E2E-Prüfungen, die die Bibliotheksgröße zählen,
müssen nach Pfad statt nach Gesamtzahl prüfen.

## Deliverables

- **D1 — Lese-API für Raster und Detail.** Go-Abfragen in `internal/library/query.go` (neu, oder
  die Listen-Abfrage erweitern, die 005 dort angelegt hat — nicht doppeln) über die `index.db`
  aus 005 (`photos`, `photo_tags`, Index `photos_taken`):
  `Groups(ctx) → {total, months:[{month:"2024-07", count, days:[{day:"2024-07-12", count}]}],
  folders:[{folder:"2024/07", count}]}` (Monat/Tag = Präfix von `taken_at`, absteigend; nur
  Ordner im Muster `JJJJ/MM`);
  `Page(ctx, month, cursor, limit) → {items:[{id, taken_at, media_type, …Thumbnail-Felder}],
  next}` — Keyset `(taken_at, id)` absteigend, Bedingung `taken_at` im Monat, nie `OFFSET`;
  Cursor opak (z. B. base64 von `taken_at|id`), `limit` Standard 200, Maximum 500, ungültige
  Parameter → 400; die Thumbnail-Felder sind die, die 006 für die Thumbnail-URL mit
  Cache-Schlüssel braucht (dessen Helfer nutzen);
  `Detail(ctx, id) → {id, rel_path, folder, media_type, format, taken_at, taken_source, tags[],
  location{label, sublocation, city, state, country}, review_state, read_error, prev_id, next_id}`
  — Nachbarn per Keyset über die ganze Bibliothek in Rasterreihenfolge, `null` an den Enden;
  unbekannte ID → 404.
  Routen in `internal/api/photos.go` (`GET /api/photos/groups`, `GET /api/photos`,
  `GET /api/photos/{id}`), registriert in `internal/api/server.go` (bzw. dort, wo 005 die
  Index-Abhängigkeit in den Handler reicht), hinter der bestehenden Session-Prüfung.
  Tests: `internal/library/query_test.go` mit per Code befüllter Test-`index.db` (Hilfen aus
  `internal/testsupport` bzw. aus 005 nutzen) und `internal/api/photos_test.go` (Statuscodes,
  JSON-Form, 401 ohne Session). Hat 005/006 ein Systemdokument unter `docs/systems/` für
  Index/API angelegt, die drei Routen dort ergänzen.
- **D2 — Rasterlogik als reine Module.** `web/src/lib/photos.ts` (Typen + Aufrufe der drei Routen
  über `api()` aus `web/src/lib/api.ts`), `web/src/lib/grid/layout.ts`,
  `web/src/lib/grid/loader.ts`, Tests `web/src/lib/grid/layout.test.ts` und
  `web/src/lib/grid/loader.test.ts` (Vitest, kein DOM).
  `layout.ts`: `columnsFor(width, viewportHeight)` = `max(3, floor(width/120))`, aber so
  begrenzt, dass ein Bildschirm voll höchstens 200 Kacheln zeigt (dann größere Kacheln);
  `buildRows(groups, columns, tileSize)` → Zeilenliste (Monatskopf 48 px, Tageskopf 36 px,
  Fotozeile = Kachelgröße) mit `top`, Gesamthöhe und je Fotozeile Monat + Index-Bereich im Monat
  (Tage beginnen immer eine neue Zeile, letzte Teilzeile eines Tages korrekt);
  `windowAt(rows, scrollTop, viewportHeight)` → sichtbare Zeilen plus Überhang (bis ½ Bildschirm
  je Seite), gekürzt so, dass nie mehr als `MAX_TILES = 300` Kacheln enthalten sind;
  `offsetOfMonth(rows, month)` und `monthForFolder(groups, "2023/03")` (gleicher Monat, sonst
  nächstälterer, sonst `null`).
  `loader.ts`: Cache je Monat (Array nach Index im Monat); `ensure(month, uptoIndex)` lädt Seiten
  über den Cursor nacheinander, bis der Index abgedeckt ist; eine laufende Anfrage je Monat,
  höchstens 2 Monate parallel; Fehler lassen Platzhalter stehen und werden beim nächsten
  `ensure` erneut versucht; `reset()` für den Neuaufbau nach einem Scan.
- **D3 — Rasteransicht mit Monatsnavigation.** `web/src/routes/Library.svelte` (Startansicht
  statt des bisherigen Platzhalters in `web/src/App.svelte`; Überschrift „PhotoCurator" und
  „Version …" bleiben sichtbar, z. B. im `AppHeader`, damit `smoke.spec.ts` grün bleibt);
  `web/src/lib/grid/PhotoGrid.svelte` (Scrollcontainer mit Gesamthöhe aus `buildRows`, nur die
  `windowAt`-Zeilen im DOM, absolut positioniert; Monatskopf als Überschrift „Juli 2024",
  Tageskopf „12. Juli 2024", Anzahl jeweils in `data-testid="group-count"`; Kachel
  `data-testid="grid-tile"` mit `data-taken` und `data-id`, Darstellung über die Thumbnail-Kachel
  aus 006 — ohne Daten ein Platzhalter ohne `<img>`; kein horizontales Scrollen ab 360 px; Klick
  ruft `onOpen(id)`; lädt die Gruppen neu, wenn der Scanstatus aus 005 von „läuft" auf „fertig"
  wechselt); `web/src/lib/grid/MonthNav.svelte` (Knopf „Zeitraum" ≥ 44 px öffnet die Liste
  Jahr → Monat als Knöpfe „März 2023" aus `folders`; Klick scrollt zu
  `offsetOfMonth(monthForFolder(…))`).
  E2E-Fixture: `web/tests/e2e/helpers/bulk-library.ts`, aufgerufen in
  `web/tests/e2e/global-setup.ts` nach dem Kopieren der Bibliothek, erzeugt 5.000 Kopien von
  `testdata/library/2024/07/sample.jpg` als `JJJJ/MM/IMG-JJJJMMTT-WA####.jpg` über
  2023-01 … 2024-12 (ungleich verteilt, auch Tage mit mehr als einer Fotozeile), `utimes` einen
  Tag zurück, und liefert die erwarteten Monats-/Tageszahlen (deterministisch, ohne Zufall).
  `web/tests/e2e/helpers/library.ts` › `waitForLibrary(request)` meldet sich an und wartet
  (bis 240 s), bis `total` aus `/api/photos/groups` der erwarteten Dateizahl entspricht. Prüfen
  005/006-Specs die Gesamtgröße der Bibliothek, auf Pfad-/Dateiauswahl umstellen.
  Test: `web/tests/e2e/library-grid.spec.ts` › "the library grid groups 5,000 photos by month and
  day, jumps to a month and keeps the page light" — bei 1440×900: erste Monats-/Tagesköpfe und
  Anzahlen gegen die Generatorzahlen, `data-taken` absteigend; Sprung über „Zeitraum" →
  „März 2023": Monatskopf im Viewport, erste Kachel aus 2023-03; in Schritten bis ans Ende
  scrollen: nach jedem Schritt ≤ 300 `grid-tile`, neue `/api/photos`-Anfragen, Kacheln des
  ältesten Monats tragen Daten; Viewport 390×844 → erste Fotozeile ≥ 3 Kacheln, 1440×900 → ≥ 6,
  `scrollWidth ≤ clientWidth` in beiden.
- **D4 — Detailansicht mit Blättern.** `web/src/lib/detail/PhotoDetail.svelte` (Überlagerung,
  `role="dialog"`; große Vorschau über die `l`-Thumbnail-URL aus 006, bei Videos mit
  Video-Symbol und ohne Abspielen; Aufnahmedatum de-DE mit Hinweis „Datum aus dem Dateinamen" /
  „Datum aus der Änderungszeit" je `taken_source`, bei `exif`/`quicktime` ohne Hinweis; Tags als
  Pfade oder „keine Tags"; Ort `location.label` oder „kein Ort"; Prüfstatus
  „offen"/„erledigt"/„nicht relevant"; `rel_path`; ArrowLeft/ArrowRight → `prev_id`/`next_id`,
  Pointer-Wischen ≥ 50 px waagrecht (nach links = nächstes, nach rechts = vorheriges), Esc und
  ×-Knopf schließen; Öffnen legt einen History-Eintrag an, `popstate` schließt), eingebunden in
  `web/src/routes/Library.svelte` (Kachel-Klick öffnet, Schließen lässt die Scrollposition stehen).
  Fixture: eine Datei der Testbibliothek mit Keywords/`HierarchicalSubject` und Ortsnamen — die
  aus 005 nutzen, wenn vorhanden (siehe `testdata/README.md`); sonst
  `testdata/library/2023/05/tagged.jpg` als Kopie von `sample.jpg` mit ExifTool in einem
  Wegwerf-Container nach den Lesefeldern aus 005 beschreiben und in `testdata/README.md`
  dokumentieren.
  Test: `web/tests/e2e/photo-detail.spec.ts` › "the detail view shows a photo's facts and steps
  through photos with arrow keys and swipes" — getaggte Datei öffnen: Vorschau-`img` geladen,
  Tags, Ort, „offen", Pfad sichtbar; eine Bulk-Datei zeigt „Datum aus dem Dateinamen",
  `2024/07/sample.jpg` „Datum aus der Änderungszeit"; ArrowRight/ArrowLeft wechseln den Pfad
  gemäß Rasterreihenfolge; Wischen per `pointerdown`/`pointermove`/`pointerup`
  (`pointerType: 'touch'`) im Viewport 390×844 wechselt ebenso; Esc schließt, das Raster steht
  an gleicher Stelle. Vorher `waitForLibrary` aus D3.

## Model Hints

- D2 → deliverable-hard — Zeilenmodell, Fenster und Monatsversatz bestimmen Sprungziel (AC2),
  Kachel-Obergrenze (AC3) und die Index-Bereiche, auf denen 008 Bereichs- und Gruppenauswahl über
  nicht geladene Fotos aufbaut; ein Off-by-one an Tagesgrenzen oder in der letzten Teilzeile
  besteht den E2E-Glückspfad und fällt erst in 008 auf.
- Review: → default — die naheliegenden Abkürzungen (alle Kacheln im DOM, `OFFSET`-Paginierung,
  Nachbarn nur aus geladenen Seiten) fängt die Kachelzählung im E2E bzw. sind im Diff sichtbar.

## Acceptance Tests

- AC1 → e2e `web/tests/e2e/library-grid.spec.ts` › "the library grid groups 5,000 photos by
  month and day, jumps to a month and keeps the page light" + unit
  `internal/library/query_test.go` › "groups count photos and videos per month and day, newest
  first" (D1, D3)
- AC2 → e2e `web/tests/e2e/library-grid.spec.ts` › "the library grid groups 5,000 photos by
  month and day, jumps to a month and keeps the page light" + unit
  `web/src/lib/grid/layout.test.ts` › "a folder jumps to its month, or the next older month when
  that month is empty" (D1, D2, D3)
- AC3 → e2e `web/tests/e2e/library-grid.spec.ts` › "the library grid groups 5,000 photos by
  month and day, jumps to a month and keeps the page light" + unit
  `web/src/lib/grid/layout.test.ts` › "no window ever holds more than 300 tiles, on any screen
  size"; `web/src/lib/grid/loader.test.ts` › "scrolling deep into a month loads its pages in
  cursor order, once each"; `internal/library/query_test.go` › "pages follow the keyset cursor
  without gaps or repeats" (D1, D2, D3)
- AC4 → e2e `web/tests/e2e/photo-detail.spec.ts` › "the detail view shows a photo's facts and
  steps through photos with arrow keys and swipes" + unit `internal/library/query_test.go` ›
  "the detail carries tags, place, review state, date source and path" (D1, D4)
- AC5 → e2e `web/tests/e2e/photo-detail.spec.ts` › "the detail view shows a photo's facts and
  steps through photos with arrow keys and swipes" + unit `internal/library/query_test.go` ›
  "previous and next follow the grid order across months and stop at the ends" (D1, D4)
- AC6 → e2e `web/tests/e2e/library-grid.spec.ts` › "the library grid groups 5,000 photos by
  month and day, jumps to a month and keeps the page light" + unit
  `web/src/lib/grid/layout.test.ts` › "a phone gets at least 3 columns and a desktop at least 6"
  (D2, D3)

Zwei neue E2E-Flows, je einer pro Nutzerpfad (Raster, Detail). Coverage-Gate: AC1 D1+D3 ·
AC2 D1+D2+D3 · AC3 D1+D2+D3 · AC4 D1+D4 · AC5 D1+D4 · AC6 D2+D3 — jedes Kriterium hat
Deliverable und Test, keine manual residue.

## Done

Raster nach Monat/Tag (virtualisiert, max. 300 Kacheln im DOM), Zeitraum-Navigation aus `JJJJ/MM`, Detailansicht als Überlagerung mit Blättern per Pfeiltasten/Wischen; Lese-API `/api/photos/groups`, `/api/photos` (erweitert um `month`), `/api/photos/{id}`.

**Commit message:** `007: Fotoraster mit Gruppierung, Monatsnavigation und Detailansicht + Lese-API`

**Verification (narrow gate):** `npm --prefix web run build` + runtime-Image, `docker build --target test .` + `npm --prefix web run test` (33 Vitest), `docker build --target lint .` + `web run lint`, `web run check`: grün. e2e (`e2e-story`, `npx playwright test library-grid.spec.ts photo-detail.spec.ts`): 2/2 grün; nach Review-Fix `thumbnails.spec.ts` + `photo-detail.spec.ts` grün (Fix-Agent). Full gate (`e2e-all`/Gesamtlauf) steht aus (Sprint).
AC → Test: AC1/2/3/6 → e2e library-grid + query_test/layout.test/loader.test; AC4/5 → e2e photo-detail + query_test; alle ran + passed, keine manual residue.

**Decisions:**
- Items von `/api/photos` bleiben camelCase (006-Vertrag), Groups/Detail snake_case wie spezifiziert; Undatierte Dateien zählen in `total`, aber in keinem Monat; im Detail-Blättern gelten sie als älteste.
- `PreviewList.svelte` entfernt; `AppHeader` trägt h1 + Version; `ScanStatus` bekam `onStatus` für Neuladen nach Scan.
- Bulk-Fixture startet bei WA1001 (keine Kollision mit `IMG-20240812-WA0001.jpg`), Fixture-mtime 2022-06; `library-scan.spec` erwartet 5010 Dateien.
- Review-Befund: `thumbnails.spec.ts` war auf API-Ebene abgeschwächt — Browser-Test (loaded/Video-Icon/failed/Cache-Reuse über das Raster) wiederhergestellt, Helfer `helpers/grid.ts`; Tie-Test für Nachbarn ergänzt.
- Bewusst offen: Neighbour-SQL nutzt `COALESCE(taken_at,'')` (kein Index, ok bei 5.000); neuer Thumbnail-Browser-Test dauert ~4,7 min (Single-Worker-Render der Fixtures); Detail-Tag-Assertion prüft nur „Geburtstag".

tiers: D 4 / hard 1 · review default · cycles 1 · agents 7
