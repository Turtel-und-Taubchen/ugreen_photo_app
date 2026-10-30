---
id: 006
title: Thumbnails mit Cache für Fotos und Videos
status: done # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E1 Index & Ansicht
estimate: 5
sprint: S02
---

## Requirement

Damit ich Fotos auswählen kann, muss ich sie sehen — schnell, auch am Handy und auch für HEIC
und Videos. Thumbnails werden einmal erzeugt und zwischengespeichert, ohne das NAS lahmzulegen
und ohne etwas in den Fotoordner zu schreiben. Details: [Tech Spec §4](../concepts/photocurator-tech-spec.md).

## Acceptance Criteria

- [x] **AC1** — Für JPEG, PNG und HEIC entstehen ein Raster-Thumbnail (ca. 320 px) und eine
  Detailvorschau (ca. 1600 px) als WebP, mit korrekt angewandter EXIF-Ausrichtung.
- [x] **AC2** — Für MP4 und MOV entsteht das Vorschaubild aus einem Frame (bei ca. 1 s, bei
  kürzeren Videos aus dem ersten Frame); im Raster tragen Videos ein Video-Symbol.
- [x] **AC3** — Ein zweiter Abruf desselben Thumbnails erzeugt es nicht neu; ändert sich die
  Datei (Größe/Änderungszeit), wird es neu erzeugt.
- [x] **AC4** — Die Erzeugung läuft im Hintergrund mit begrenzter Parallelität (Platzhalter: 1
  Worker, niedrige Prozesspriorität); ein noch fehlendes, gerade sichtbares Thumbnail wird
  vorgezogen und bis dahin durch einen Platzhalter ersetzt.
- [x] **AC5** — Thumbnail-Antworten tragen langlebige Cache-Header; der Browser lädt ein bereits
  geladenes Thumbnail beim erneuten Scrollen nicht erneut vom Server.
- [x] **AC6** — Für eine defekte Datei erscheint ein Fehlerplatzhalter; die App läuft weiter.
- [x] **AC7** — Es wird nichts unter `/photos` angelegt oder verändert; der Cache liegt unter
  `/data`.

## Open Questions

- (keine offen — alle Detailfragen unten unter „Decisions (Sprint)" entschieden)
- Hinweis für das Sprint-Review: `CLAUDE.md` existiert im Repository nicht; Leitplanken kamen aus
  Profil, Tech Spec und Konzept.

## Decisions (Sprint)

- **Cache-Schlüssel** = `sha1(rel_path|size|mtime_ns)` aus der Index-Zeile, Pfad
  `/data/thumbs/<xx>/<key>-<s|l>.webp` — so steht es in Tech Spec §4, Dateiänderungen invalidieren
  ohne eigene Buchhaltung.
- **Größe/mtime kommen aus dem Index (005), nicht aus `stat` beim Abruf** — die URL trägt den
  Schlüssel und muss zur Liste passen; bis zum nächsten Scan gilt der indizierte Stand.
- **Nie hochskalieren** (`--size 320x320>` bzw. `1600x1600>`) — eine 1600-px-„Vorschau" aus einem
  kleinen Bild wäre nur größer, nicht besser.
- **Video-Frame:** erst `ffmpeg -ss 1`; liefert das keinen Frame (Video < 1 s), zweiter Versuch ohne
  `-ss` (Frame 0) — robuster als vorher die Dauer zu proben (spart einen ffprobe-Prozess).
- **Hintergrund erzeugt nur `s`**; `l` entsteht nur auf Anforderung (vorgezogen) — Ressourcenschonung
  laut Tech Spec, die Detailansicht ruft je ein Bild ab.
- **Antwort-Protokoll** `GET /api/thumbs/{id}/{s|l}?k=<key>`: fertig → 200 WebP; noch nicht fertig →
  202 + `Retry-After: 1` (Job vorgezogen); Erzeugung gescheitert → 422; unbekannte ID → 404 —
  ein blockierender Request würde bei 1 Worker die 6 Browser-Verbindungen und damit die übrige API
  verstopfen.
- **Nur bei passendem `k`** gibt es `Cache-Control: private, max-age=31536000, immutable`; fehlt `k`
  oder ist er veraltet, wird der aktuelle Stand mit `no-store` geliefert; 202/422 immer `no-store`
  — eine unveränderliche Antwort darf nur an genau diesen Inhalt gebunden sein.
- **Fehler werden negativ gecacht** (`<key>-<size>.failed`, leer) — eine defekte Datei soll nicht bei
  jedem Scrollen erneut ffmpeg/vips starten; eine Dateiänderung ändert den Schlüssel und damit den
  Retry.
- **Prozess-Zeitlimit 120 s** pro vips/ffmpeg-Aufruf, Abbruch zählt als Fehler — eine hängende Datei
  darf den einzigen Worker nicht dauerhaft blockieren.
- **Niedrige Priorität per `nice -n 10`** vor jedem Prozess (geht ohne `CAP_SYS_NICE`, passt zu
  `cap_drop: ALL`); Worker-Zahl 1 als Konstruktorparameter, kein neuer Env-Schalter (Platzhalter
  bleibt Code-Konstante bis 021 misst).
- **Atomar schreiben:** Ausgabe in eine Temp-Datei im Cache-Verzeichnis, dann `rename`; der
  Video-Zwischenframe liegt ebenfalls unter `/data/thumbs/tmp`, nie unter `/photos` und nicht im
  RAM-`/tmp`.
- **Frontend lädt per `fetch` + Blob-URL statt `<img src>`** — nur so unterscheidet der Client
  „in Arbeit" (202, erneut versuchen) von „defekt" (422, Fehlerplatzhalter); `fetch` nutzt den
  HTTP-Cache, AC5 bleibt erfüllt. Blob-URLs werden beim Entfernen der Kachel freigegeben.
- **„Sichtbar zuerst"** = die Kachel fordert erst an, wenn sie im Viewport ist (IntersectionObserver);
  jede Anforderung eines fehlenden Thumbnails zieht den Job vor die Hintergrund-Jobs.
- **Anzeigefläche bis 007:** 006 liefert eine schlichte, flache Vorschauliste auf der Startseite
  (erste Seite aus `GET /api/photos`, falls 005 den Endpunkt nicht schon anlegt, in Minimalform laut
  Tech Spec §6) — 007 ersetzt sie durch das virtualisierte, gruppierte Raster; ohne Fläche wäre
  keines der UI-Kriterien über die echte Oberfläche prüfbar.
- **Aufräumen verwaister Cache-Dateien** ist kein AC dieser Story und bleibt draußen (Follow-up fürs
  Review) — der Cache wächst nur bei geänderten Dateien.
- **Fixtures:** PNG, MP4 (2 s, Farbwechsel bei 1 s), MOV (0,5 s) und eine defekte `.jpg` werden
  committet (`testdata/library/2024/08/`), die EXIF-gedrehte JPEG wird im Test aus `sample.jpg`
  per ExifTool erzeugt — so prüfen Go- und E2E-Tests dieselben Dateien, und nichts Privates landet
  im Repo.

## Plan

Neues Paket `internal/thumbs` (Render → Queue/Service), ein API-Handler, eine Frontend-Kachel.
Baut auf 005 auf: Index-Zeile per ID (`rel_path`, `media_type`, `size`, `mtime_ns`) und
Scan-Abschluss als Auslöser für die Hintergrund-Erzeugung.

1. **D1 Render:** `Key()`/Cache-Pfade; Bilder über `nice -n 10 vipsthumbnail`, Videos über
   `nice -n 10 ffmpeg` (1 s, Fallback Frame 0) → vipsthumbnail; atomar ins Cache-Verzeichnis;
   Fehler → `.failed`-Marker. Fixtures + README. Integrationstests im Docker-Stage `test`.
2. **D2 Queue/Service:** Prioritäts-Queue (hoch = angefordert, niedrig = Hintergrund), 1 Worker,
   Deduplizierung je Schlüssel+Größe, Hochstufen bereits wartender Jobs, `Status(entry,size)` →
   ready/pending/failed, `EnqueueBackground(entries)`, sauberes Stoppen. Unit-Tests mit Fake-Renderer.
3. **D3 API + Verdrahtung:** `GET /api/thumbs/{id}/{s|l}?k=`, Header-Regeln laut Decisions;
   `serve` startet den Service mit `/data/thumbs`; nach jedem Scan `EnqueueBackground`;
   Listeneinträge tragen `mediaType` + `thumbKey` (ggf. Minimal-`GET /api/photos`).
4. **D4 Frontend + E2E:** `thumbLoader.ts` (Zustandsautomat fetch → loaded/pending/failed, Retry mit
   Backoff), `Thumbnail.svelte` (Platzhalter, Fehlerplatzhalter, Video-Symbol, lazy per
   IntersectionObserver, Blob-Freigabe), flache Vorschauliste auf der Startseite, ein E2E-Flow.

Reihenfolge D1 → D2 → D3 → D4. Kein Systems-Doc betroffen (`docs/systems/` ist leer).

## Deliverables

- [x] **D1 — Thumbnail-Erzeugung (Render) + Fixtures.** Neues Paket `internal/thumbs`:
  `key.go` — `Key(relPath string, size, mtimeNs int64) string` = hex `sha1("relPath|size|mtimeNs")`;
  `Path(cacheDir, key, size)` = `<cacheDir>/<key[0:2]>/<key>-<s|l>.webp`, Marker
  `…-<s|l>.failed`. `render.go` — `Renderer` mit `Render(ctx, absSrc, mediaType, size) error`:
  Bild → `nice -n 10 vipsthumbnail <src> --size 320x320> --export-profile srgb -o <abs-tmp>.webp[Q=75,strip]`
  (l: `1600x1600>`); **`-o` immer absolut**, sonst schreibt vipsthumbnail neben die Quelle.
  Video → `nice -n 10 ffmpeg -nostdin -ss 1 -i <src> -frames:v 1 <cacheDir>/tmp/<rand>.png`; ohne
  erzeugten Frame zweiter Lauf ohne `-ss`; dann vipsthumbnail wie oben, Temp-Frame löschen. Ausgabe
  erst in Temp-Datei im Zielordner, dann `os.Rename`. Fehler (Exit≠0, kein Output, Timeout 120 s
  per `exec.CommandContext`) → leerer `.failed`-Marker + Fehler zurück; `slog`-Warnung mit rel. Pfad.
  Kein Shell-Aufruf, Argumente als Slice. Muster für `exec.Command`: `internal/probe/video.go`.
  Fixtures in `testdata/library/2024/08/`: `sample.png` (aus `sample.jpg` konvertiert),
  `clip.mp4` (2 s, 320×240, Sekunde 0–1 rot, 1–2 blau, H.264), `short.mov` (0,5 s, rot), `broken.jpg`
  (wenige Bytes Nicht-JPEG); Erzeugungsrezept + Tabellenzeilen in `testdata/README.md` (Muster:
  bestehender Abschnitt dort). Falls 005-Tests die Fixture-Bibliothek abzählen, dort anpassen.
  Tests `internal/thumbs/render_test.go` (laufen im Docker-Stage `test`, Fixtures über
  `testsupport.CopyFixture(t, "library")`, Maße über `vipsheader`, Pixel über `vips getpoint`):
  JPEG/PNG/HEIC → s und l als WebP, längste Kante ≤ 320 bzw. ≤ 1600, nie größer als die Quelle;
  EXIF-gedrehte JPEG (Kopie von `sample.jpg` mit `exiftool -n -Orientation=6`) → Hochformat
  240×320; `clip.mp4` → überwiegend blau; `short.mov` → Thumbnail existiert; `broken.jpg` → Fehler +
  Marker; jeder gebaute `exec.Cmd` beginnt mit `nice -n 10` (Befehlsbau als eigene Funktion, damit
  prüfbar); danach für alle Fixtures: Dateiliste, Größen, mtimes und SHA-256 unter der
  Bibliothekskopie unverändert, alle neuen Dateien liegen unter dem Cache-Verzeichnis.

- [x] **D2 — Queue und Service mit Priorität.** `internal/thumbs/service.go` (+ `queue.go` nach
  Bedarf). `Entry{ID int64; RelPath, MediaType string; Size, MtimeNs int64}`;
  `NewService(cacheDir, photosDir string, r renderer, workers int)` mit Interface (umgesetzt als `RenderTo(ctx, absSrc, mediaType, size, dstWebp)`, siehe Done)
  `renderer interface{ Render(ctx, absSrc, mediaType, size string) error }` (D1s `Renderer` erfüllt
  es); `Start(ctx)`/`Stop()` (wartet auf laufenden Job, keine Goroutine bleibt hängen).
  `Status(e Entry, size string) State` → `Ready` (Datei existiert), `Failed` (Marker existiert),
  sonst reiht einen **hoch priorisierten** Job ein (bzw. stuft einen wartenden Hintergrund-Job hoch)
  und gibt `Pending` zurück. `EnqueueBackground(entries []Entry)` reiht für fehlende `s`-Thumbnails
  niedrig priorisierte Jobs ein. Dedupe je `(key,size)`, auch für gerade laufende Jobs. Hohe Jobs
  vor niedrigen, innerhalb einer Stufe LIFO für hoch (zuletzt gescrollt = sichtbar) und FIFO für
  niedrig. Pfadauflösung `filepath.Join(photosDir, filepath.FromSlash(rel))` mit Prüfung „liegt unter
  photosDir" (Tech Spec §9). Ein Render-Fehler beendet den Worker nie.
  Tests `internal/thumbs/service_test.go` mit Fake-Renderer (zählt Aufrufe, misst gleichzeitige
  Aufrufe, kann blockieren/scheitern), reine Unit-Tests ohne native Tools.

- [x] **D3 — API-Endpunkt und Verdrahtung.** `internal/api/thumbs.go`:
  `GET /api/thumbs/{id}/{size}` (size ∈ `s`,`l`, sonst 404), Eintrag per ID aus dem Index von 005
  (`internal/library`; fehlt dort eine Abfrage per ID, dort ergänzen), `thumbs.Status`:
  `Ready` → 200 `image/webp` (über `http.ServeFile`/`ServeContent`), `Cache-Control: private,
  max-age=31536000, immutable` **nur** wenn Query `k` == aktueller `Key`, sonst `no-store`;
  `Pending` → 202 JSON `{"state":"pending"}`, `Retry-After: 1`, `no-store`; `Failed` → 422 JSON
  `{"state":"failed"}`, `no-store`; unbekannte ID → 404 JSON. Route in `internal/api/server.go`
  registrieren (hinter `RequireSession`, wie die übrigen). Abhängigkeit über ein kleines Interface in
  `AuthDeps`-Art (z. B. `ThumbDeps{Lookup, Service}`), damit Tests ohne DB laufen.
  `cmd/photocurator/main.go` (`serve`): `<DataDir>/thumbs` anlegen, Service mit 1 Worker starten,
  beim Herunterfahren stoppen; nach jedem abgeschlossenen Scan (Hook von 005) `EnqueueBackground`
  mit allen Index-Einträgen. Listen-Einträge (`GET /api/photos` von 005; existiert er nicht, hier
  minimal anlegen: Keyset `(taken_at DESC, id DESC)`, `?after=<cursor>&limit=` ≤ 200, Antwort
  `{items:[{id, relPath, mediaType, takenAt, thumbKey}], next}`) tragen `mediaType` und `thumbKey`.
  Tests `internal/api/thumbs_test.go` mit Fake-Lookup + echtem `thumbs.Service` und Fake-Renderer:
  alle Statuscodes und Header oben; derselbe Abruf zweimal → genau ein Render; geänderte
  Größe/mtime im Fake-Index → neuer `thumbKey`, erneuter Render.

- [x] **D4 — Kachel im Frontend + E2E-Flow.** `web/src/lib/thumbLoader.ts`:
  `loadThumb(url, {fetchFn, signal, sleep})` → Zustände `pending | loaded(blobUrl) | failed`;
  200 → `URL.createObjectURL(blob)`; 202 → erneut nach `Retry-After` (Backoff bis 5 s, bis Abbruch
  per `signal`); 422/sonstige → `failed`; 401 → wie `api.ts` nach `/login`.
  `web/src/lib/Thumbnail.svelte`: Props `id`, `mediaType`, `thumbKey`, `size='s'`; URL
  `/api/thumbs/{id}/{size}?k={thumbKey}`; startet erst, wenn im Viewport (IntersectionObserver),
  bricht beim Verlassen/Zerstören ab und gibt die Blob-URL frei; zeigt Platzhalter
  (`data-state="pending"`, `aria-label="Vorschau wird erzeugt"`), Fehlerplatzhalter
  (`data-state="failed"`, Text/`aria-label` „Vorschau nicht verfügbar"), fertiges `<img>`
  (`data-state="loaded"`, `alt` = Dateiname), bei `mediaType==='video'` zusätzlich ein Video-Symbol
  mit `aria-label="Video"`. Flache Vorschauliste auf der Startseite (`web/src/App.svelte` bzw. neue
  `web/src/lib/PreviewList.svelte`): erste Seite `GET /api/photos`, je Eintrag eine `Thumbnail`-Kachel
  (`data-testid="thumb-<relPath>"`), CSS-Grid, ab 360 px benutzbar — wird in 007 ersetzt.
  Tests: `web/src/lib/thumbLoader.test.ts` (Vitest, Fake-fetch) und
  `web/tests/e2e/thumbnails.spec.ts` (genau ein Flow „start page previews every format, marks broken
  files and reuses cached thumbnails": Login → Kacheln für jpg/heic/png/mp4/mov `loaded`, mp4/mov mit
  Video-Symbol, `broken.jpg` mit Fehlerplatzhalter, `/api/health` weiter 200 → erneut `page.goto('/')` (kein `reload`, der Revalidierung auslösen kann) →
  keine Thumbnail-Antwort vom Server; Muster
  `web/tests/e2e/smoke.spec.ts`, Login über `helpers/auth.ts`). Für AC5 eine CDP-Session
  (`page.context().newCDPSession(page)`, `Network.enable`) und zählen, wie viele Antworten für die
  Thumbnail-URL **nicht** aus dem Cache kamen (`Network.responseReceived` ohne `fromDiskCache`
  und ohne vorheriges `Network.requestServedFromCache`); kein `page.route` (schaltet den HTTP-Cache
  ab). Fixture-Bibliothek wird wie bisher von `global-setup.ts` kopiert (enthält D1-Dateien).

## Model Hints

- D2 → deliverable-hard — einzige nebenläufige Logik der Story: Hochstufen eines bereits wartenden
  Jobs, Dedupe gegen den gerade laufenden Job und das Rennen zwischen `Status()` und
  Worker-Abschluss sind leicht so zu bauen, dass doppelt gerendert, ein Job verloren oder beim
  `Stop()` eine Goroutine hängen gelassen wird — und die übrigen Ds verlassen sich darauf.
- Review: → default

## Acceptance Tests

- AC1 → integration `internal/thumbs/render_test.go` › "images become 320 and 1600 px webp previews
  without upscaling" und › "an exif-rotated jpeg yields a portrait thumbnail"
- AC2 → integration `internal/thumbs/render_test.go` › "a video preview is the frame at one second"
  und › "a video shorter than one second uses its first frame"; Video-Symbol → e2e
  `web/tests/e2e/thumbnails.spec.ts` › "start page previews every format, marks broken files and reuses cached thumbnails"
- AC3 → unit `internal/api/thumbs_test.go` › "a second request serves the cached thumbnail without rendering"
  und › "a changed size or mtime renders a new thumbnail under a new key"
- AC4 → unit `internal/thumbs/service_test.go` › "never more renders at once than workers" und
  › "a requested thumbnail jumps ahead of background jobs"; `nice` → integration
  `internal/thumbs/render_test.go` › "renderer processes run at nice 10" (jeder vom Renderer gebaute
  `exec.Cmd` startet mit `nice -n 10`);
  Platzhalter → unit `web/src/lib/thumbLoader.test.ts` › "a pending thumbnail retries until it is ready"
  und e2e `web/tests/e2e/thumbnails.spec.ts` › "start page previews every format, marks broken files and reuses cached thumbnails"
  (jede sichtbare Kachel endet in `loaded` oder `failed`; kein Assert auf das flüchtige `pending`).
- AC5 → unit `internal/api/thumbs_test.go` › "only a request with the current key is cacheable forever";
  e2e `web/tests/e2e/thumbnails.spec.ts` › "start page previews every format, marks broken files and reuses cached thumbnails" (Schritt: erneut `page.goto('/')` statt Reload, Kacheln
  erneut sichtbar → 0 Thumbnail-Antworten vom Server laut CDP)
- AC6 → integration `internal/thumbs/render_test.go` › "a broken file fails and leaves a failure marker";
  unit `internal/thumbs/service_test.go` › "a failed render does not stop the worker";
  e2e `web/tests/e2e/thumbnails.spec.ts` › "start page previews every format, marks broken files and reuses cached thumbnails"
  (Fehlerplatzhalter für `broken.jpg`, übrige Kacheln geladen, `/api/health` danach 200).
- AC7 → integration `internal/thumbs/render_test.go` › "rendering leaves the library untouched and writes only into the cache"

## Done

Neues Paket `internal/thumbs` (Key/Render, Prioritäts-Queue mit 1 Worker), `GET /api/thumbs/{id}/{s|l}` mit den Cache-Header-Regeln, `GET /api/photos` (Keyset, neu angelegt), Hintergrund-Erzeugung nach jedem Scan, Kachel + flache Vorschauliste im Frontend, ein E2E-Flow.

Commit message: `006: thumbnails — vips/ffmpeg render, priority queue, /api/thumbs with immutable cache, preview tiles + E2E`

Verification (narrow gate, test-story none → full `test`): build, lint, typecheck, `docker build --target test .` + vitest (14) und `e2e-story` `npx playwright test tests/e2e/thumbnails.spec.ts` grün (nach Review-Fix erneut e2e + test/lint/check). Review: 1 Zyklus, PASS, 4 Findings behoben. AC → Test wie oben unter Acceptance Tests, alle gelaufen und grün; kein manual residue.

Decisions:
- Renderer-API: `Renderer{CacheDir, Root}.RenderTo(ctx, src, mediaType, size, dst)` statt `Render(...)`; das Service-Interface verlangt `RenderTo` (Ziel = `thumbs.Path`), so kennt der Fake dasselbe Ziel.
- Fixtures: vorhandene `screenshot.png`/`defekt.jpg` wiederverwendet; `clip.mp4` neu erzeugt (2 s rot/blau), `short.mov` neu → Fixture-Bibliothek 10 statt 9 Medien (005-Tests/E2E angepasst, 259→260).
- 005 hatte kein `GET /api/photos`: hier angelegt (opaker base64-Cursor, limit ≤ 200), mit DB-Test in `internal/library/photos_test.go`.
- Service merkt Fehlschläge zusätzlich im Speicher (bis Neustart); `.failed`-Marker eines durch `Stop()` abgebrochenen Renders wird entfernt.
- Bewusst offen: nice-Test prüft nur die Befehlsbau-Funktionen; `COALESCE(taken_at,'')` nutzt den Index nicht (Perf, bei 007/021 messen); Queue-Slots wachsen pro Poll während langem Render (begrenzt); Symlinks nicht geprüft; Cache-Aufräumen bleibt Follow-up.

tiers: D 4 / hard 1 · review default · cycles 1 · agents 8
