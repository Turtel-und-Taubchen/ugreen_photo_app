# Sprint S02 — Review

Stand: 2026-10-30 · Branch: `main` · Meilensteine: M2 (komplett), M3 (011), M4 (014)

## Überblick

**Ziel:** PhotoCurator liest die Bibliothek aus `Sortiert` rein lesend, zeigt sie als schnelles,
mobiltaugliches Raster mit Thumbnails und erlaubt Einzel-, Bereichs- und Gruppenauswahl; dazu
Kategorienbaum und Ortssuche.

| Story | Status | Commit |
| --- | --- | --- |
| 005 — Bibliotheks-Index | done | `9ab0a13` exif-Leser, index.db-Scanner, Scheduler, /api/scan, Scan-Anzeige |
| 006 — Thumbnails | done | `432d9cc` vips/ffmpeg-Render, Prioritäts-Queue, /api/thumbs mit immutable Cache |
| 007 — Fotoraster & Navigation | done | `7f38979` Raster mit Gruppierung, Monatsnavigation, Detailansicht, Lese-API |
| 008 — Auswahl | done | `f1c9dbb` Auflösung + /api/selection/count, Auswahlmodell, Desktop + Handy |
| 011 — Kategorienbaum | done | `4fb8d71` ParsePath/BuildTree, tag_definitions, /api/tags, Seite Tags |
| 014 — Ortssuche | done | `05d6ab8` Nominatim-Client, Limiter, geocode_cache, /api/geo/*, Seite /orte |

## Umgesetzte Stories

- **005:** Scan von `/photos` mit ExifTool (ein Aufruf je ~200 Dateien, kein `-stay_open`, kein `-fast2`) in `index.db`; Delta über Größe/mtime, Ruhezeit, sicheres Löschen nur in vollständig gelesenen Ordnern, Aufnahmedatum-Rückfallkette inkl. WhatsApp-Namen. Scheduler (Start, Intervall, „Neu einlesen"), `GET/POST /api/scan`, Scan-Status mit „nicht lesbar"-Liste auf der Startseite.
- **006:** WebP-Thumbnails 320/1600 px über `nice -n 10 vipsthumbnail`, Video-Frame bei 1 s über ffmpeg; Cache-Key aus Pfad/Größe/mtime, atomar geschrieben, Fehler-Marker. Ein Worker mit Vorziehen angefragter Bilder; `/api/thumbs` mit `immutable` nur bei passendem Key, 202 solange in Arbeit.
- **007:** Startansicht als virtualisiertes Raster, gruppiert nach Monat/Tag, ≤ 300 Kacheln im DOM, Keyset-Nachladen pro Monat, Sprung über `JJJJ/MM`. Detail-Overlay mit Tags, Ort, Prüfstatus, Datumsquelle, Pfeil-/Wischnavigation.
- **008:** Auswahl als `{filter:{groups,ranges}, include_ids, exclude_ids}`, serverseitig aufgelöst (deckt nicht geladene Fotos ab); Zähler vom Server. Desktop: Shift+Klick, „Alle auswählen" am Gruppenkopf; Handy: langes Drücken, „Bereich bis hier".
- **011:** Kategorien als Pfade (`Anlass/Geburtstag`) mit Validierung in Go, `tag_definitions` in app.db, Baum aus Index-Tags + Definitionen mit Fotozahl je Teilbaum; Löschen nur leerer Kategorien (409 sonst). Seite `/tags`.
- **014:** Nominatim-Suche und Reverse-Geocoding mit globalem 1-s-Limiter, 90-Tage-Cache in app.db, eigenem `Place`-Format und Ortsnamen `<Name>, <Ort>`; ohne `PC_NOMINATIM_CONTACT` deaktiviert mit Begründung. Seite `/orte` mit wiederverwendbarem `PlacePicker` (für 015). E2E gegen einen Nominatim-Mock.

## Erkenntnisse & Entscheidungen

Für die nächste Planung relevant (Details in den `## Decisions (Sprint)`-Abschnitten der Stories):

- **Schnittstellen, auf die M3/M4 bauen:** `api.NewHandler` mit `LibraryDeps` (Felder je Story), `library.ResolveSelection` für 012, `tags.ParsePath` für 012/013, `PlacePicker` mit `onchange(PickedPlace|null)` für 015. Migrationen in app.db: 2 = `tag_definitions`, 3 = `geocode_cache`.
- **Fixture-Bibliothek:** 10 Mediendateien in `testdata/library`; das E2E-Setup erzeugt zusätzlich 5.000 Fotos (Monat 2023-03 mit ~540). Tests, die Dateien zählen, müssen das mitziehen.
- **Zeilenenden:** Go-Dateien müssen LF sein, sonst scheitert `gofmt -l` in der Lint-Stufe (in 007 und 011 aufgetreten). `.gitattributes` kam mit 005.
- **Go-Testnachweis:** Das Dockerfile ruft `go test` ohne `-v`; die Builds belegten Go-Tests deshalb nur paketweise („ok"), nicht pro Testname.
- **Zugriffssperre:** `deploy/.env.example` ist für Agenten gesperrt, daher wurden `PC_SCAN_INTERVAL`/`PC_SCAN_QUIET` nicht in die Deploy-Compose durchgereicht → Follow-up in der Roadmap.
- **CLAUDE.md fehlt** im Repo; die Agenten nutzten Profil und Tech Spec als Leitplanken.
- **Bewusst nicht behobene Review-Funde** sind in [022 — Robustheit und Testlücken](../../../requirements/022-robustheit-aus-s02-review.md) (Story-Entwurf) bzw. als Follow-ups in der [Roadmap](../../../ROADMAP.md#follow-ups-worth-doing) gelandet. Als nicht handlungsbedürftig eingestuft: der nice-Test prüft nur den Befehlsbau, die Fake-Uhr in 014 unterscheidet den Startzeitpunkt nicht, `PickedPlace` statt `Place` (bewusste Schnittstelle), Detail-Tag-Assertion prüft nur einen Tag, Scheduler meldet kurz `idle` nach `Trigger()`, `finished_at` mit Sekundenauflösung.

## Blockiert / offen

Keine blockierte Story. Offen beim Nutzer: Spike auf dem NAS (Eingangsbedingung für 009/012 in S03).

## Regression Gate

Lauf auf `05d6ab8`: `build`, `test`, `lint`, `typecheck` je < 1 min grün; `e2e` 13 Flows grün in 6,2 min (davon 4,8 min der Thumbnail-Rastertest). `e2e-all` ist `none`. Keine Fehler, keine Quarantäne.

## Abnahme

Alle Kriterien sind durch automatisierte Tests belegt. **Manual residue: keines.**

| Story | Kriterium | Beleg |
| --- | --- | --- |
| 005 | AC1 rekursiver Scan, Ignorierte | `internal/library/scan_test.go` (first scan indexes every media file…) |
| 005 | AC2 Felder je Datei | `scan_test.go`, `internal/metadata/read_test.go` |
| 005 | AC3 Datum-Rückfallkette | `internal/library/takendate_test.go` |
| 005 | AC4 Delta-Folgescan | `scan_test.go` (second scan reads only new or changed files) |
| 005 | AC5 Ruhezeit | `scan_test.go` (files younger than the quiet period…) |
| 005 | AC6 Start/Intervall/Knopf/Status | e2e `library-scan.spec.ts`; `scheduler_test.go`; `web/src/lib/scan.test.ts` |
| 005 | AC7 Neuaufbau nach Löschen | `scan_test.go` (rebuilt index … holds the same tags…) |
| 005 | AC8 defekte Datei „nicht lesbar" | `scan_test.go` + e2e `library-scan.spec.ts` |
| 005 | AC9 Scan ändert nichts | `scan_test.go` (leaves every file … unchanged) |
| 006 | AC1 WebP 320/1600, Rotation | `internal/thumbs/render_test.go` |
| 006 | AC2 Video-Frame, Video-Symbol | `render_test.go`; e2e `thumbnails.spec.ts` |
| 006 | AC3 Cache-Reuse/Neuerzeugung | `internal/api/thumbs_test.go` |
| 006 | AC4 1 Worker, nice, Vorziehen, Platzhalter | `service_test.go`, `render_test.go`, `thumbLoader.test.ts`; e2e `thumbnails.spec.ts` |
| 006 | AC5 langlebige Cache-Header | `thumbs_test.go`; e2e `thumbnails.spec.ts` (CDP, 0 Server-Antworten) |
| 006 | AC6 Fehlerplatzhalter | `render_test.go`, `service_test.go`; e2e `thumbnails.spec.ts` |
| 006 | AC7 nichts unter /photos | `render_test.go` (leaves the library untouched) |
| 007 | AC1 Gruppierung + Anzahl | e2e `library-grid.spec.ts`; `query_test.go` |
| 007 | AC2 Sprung nach JJJJ/MM | e2e `library-grid.spec.ts`; `grid/layout.test.ts` |
| 007 | AC3 ≤ 300 Kacheln, Nachladen | e2e `library-grid.spec.ts`; `layout.test.ts`, `loader.test.ts`, `query_test.go` |
| 007 | AC4 Detailansicht | e2e `photo-detail.spec.ts`; `query_test.go` |
| 007 | AC5 Pfeile/Wischen | e2e `photo-detail.spec.ts`; `query_test.go` |
| 007 | AC6 ≥ 3 / ≥ 6 Spalten | e2e `library-grid.spec.ts`; `layout.test.ts` |
| 008 | AC1 Klick markiert, Zähler | e2e `selection.spec.ts` (Desktop); `selection.test.ts` |
| 008 | AC2 Shift+Klick | e2e `selection.spec.ts`; `selection.test.ts`; `internal/library/selection_test.go` |
| 008 | AC3 langes Drücken, „Bereich bis hier" | e2e `selection.spec.ts` (Handy); `longpress.test.ts` |
| 008 | AC4 „Alle auswählen" inkl. ungeladener | e2e `selection.spec.ts`; `selection_test.go` |
| 008 | AC5 Auswahl bleibt / aufheben | e2e `selection.spec.ts`; `selection.test.ts` |
| 008 | AC6 Server-Auflösung | `internal/library/selection_test.go`, `internal/api/selection_test.go`, `selection.test.ts`; e2e (abgefangener Request) |
| 011 | AC1 Pfad anlegen, Eltern auto | `internal/tags/definitions_test.go`; e2e `tags.spec.ts` |
| 011 | AC2 Baum + Anzahl | e2e `tags.spec.ts`; `tree_test.go`, `tagquery_test.go` |
| 011 | AC3 Löschsperre | e2e `tags.spec.ts`; `internal/api/tags_test.go`, `definitions_test.go` |
| 011 | AC4 ungültige Namen | `internal/tags/path_test.go`; e2e `tags.spec.ts` |
| 011 | AC5 Baum nach Index-Neuaufbau | `internal/api/tags_test.go` (tag tree survives an index rebuild) |
| 014 | AC1 Suche, ≤ 10, POI zuerst | e2e `places.spec.ts`; `internal/geo/place_test.go` |
| 014 | AC2 ≥ 1 s Abstand | `internal/geo/geocoder_test.go`; `internal/api/geo_test.go` |
| 014 | AC3 User-Agent, deaktiviert ohne Kontakt | `geocoder_test.go`, `geo_test.go`, `web/src/lib/geo.test.ts`; e2e (UA am Mock) |
| 014 | AC4 Cache 90 Tage | `geocoder_test.go`; e2e `places.spec.ts` |
| 014 | AC5 Reverse über Limiter + Cache | `geocoder_test.go`; e2e `places.spec.ts` |
| 014 | AC6 Ortsname + Ortsteile | `place_test.go`; e2e `places.spec.ts` |
| 014 | AC7 Ausfall, manueller Ort | `geocoder_test.go`, `geo_test.go`; e2e `places.spec.ts` |

**Unterhalb der echten Oberfläche belegt** (alle ohne eigene Nutzeraktion oder mit begründetem
Ersatz): 005 AC6 Intervall/Zwischenstand (Fixture-Scan zu kurz für E2E → Go + Vitest); 006 AC4
Worker/nice/Queue (Prozessinterna); 011 AC5 (E2E bräuchte Container-Neustart); 014 AC3
Deaktiviert-Anzeige (zweiter Container pro Lauf zu teuer → Go-API + Vitest); 014 AC7 Kartenpunkt
als Koordinatenfelder bis 015 die Karte bringt.

## Tier-Bilanz

| Story | Ds | hard Ds | Review-Stufen | Zyklen | Agenten | Build-Minuten |
| --- | --- | --- | --- | --- | --- | --- |
| 005 | 7 | 1 | default | 1 | 11 | 98 |
| 006 | 4 | 1 | default | 1 | 8 | 48 |
| 007 | 4 | 1 | default | 1 | 7 | 73 |
| 008 | 5 | 0 | default | 0 | 7 | 75 |
| 011 | 4 | 0 | default | 0 | 6 | 36 |
| 014 | 5 | 1 | default | 1 | 9 | 44 |
| **Summe** | **29** | **4** | **0 × hard** | **4** | **48** | **374** |

Kein Story-Review lief in der zweiten (hard) Stufe; die Frage, ob sie etwas gefunden hätte, stellt
sich in diesem Sprint nicht. Refine: 6 Stories parallel in 6 min.
