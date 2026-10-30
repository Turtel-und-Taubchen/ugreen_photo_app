# Sprint S03 — Review

## Überblick

Ziel: Tags und Orte lassen sich im Batch sicher in die Dateien schreiben — gemäß Spike-Ergebnis, mit Original-Backup, Fehlerprotokoll pro Datei und Rückgängig pro Job; Taggen wird schneller, Orte per Suche, Karte oder Lieblingsort.

| Story | Status | Commit |
| --- | --- | --- |
| [009](../../../requirements/done/009-metadaten-writer.md) Sicherer Metadaten-Writer mit Batch-Jobs | done | 73ba9c2 (+ Gate-Fix 024e351) |
| [010](../../../requirements/done/010-backup-und-rueckgaengig.md) Backup, Journal, Rückgängig pro Job | done | 33eda2e |
| [012](../../../requirements/done/012-batch-tagging.md) Tags im Batch zuweisen/entfernen | done | 465f5cc (+ Gate-Fix 79f3744) |
| [013](../../../requirements/done/013-tag-schnellauswahl-autocomplete.md) Zuletzt verwendete Tags, Autovervollständigung | done | 166ece5 |
| [015](../../../requirements/done/015-karte-und-batch-standort.md) Standort im Batch, Karte | done | 62bfad2 |
| [016](../../../requirements/done/016-lieblingsorte.md) Lieblingsorte | done | 2f4c8a9 |

Der Sprint war nach dem Refine pausiert, bis das Spike-Ergebnis vorlag (Entscheidung Nutzer: Option A, Spike abwarten); danach wurden 009, 010, 012, 015, 016 neu refined.

## Umgesetzte Stories

- **009:** Mapping-Tabelle (`internal/metadata`), ExifTool-Writer, Pipeline je Datei (Vorabkopie in `.inflight`, Read-back, Restore in place, Crash-Recovery), Job-Queue + API + Statusanzeige. `PC_PRESERVE_MTIME` Default `true` (Spike). Ort nur per GPS-Felder, keine Video-Sidecars.
- **010:** Backup der Originale mit Aufbewahrung, Änderungsjournal je Job, Rückgängig mit Konfliktprüfung, Seite `/jobs`, Betriebsanleitung.
- **012:** `tags.add`/`tags.remove` im Batch (Auswahl, Bereich, Tag, Monat), Skip vor Backup bei „bereits aktuell", Dialoge in der Auswahlleiste.
- **013:** Zuletzt verwendete Tags (gemeinsam), Autovervollständigung im TagPicker.
- **015:** `location.set`/`location.remove` (Modus alle/nur fehlende), Karte (Leaflet, lazy) im PlacePicker, Dialoge „Ort setzen/entfernen", Detailkarte.
- **016:** Lieblingsorte (`favorite_places`, `/api/places`), Anwendung ohne Nominatim-Aufruf.

## Findings & Entscheidungen

- Spike-Folge: UGREEN Fotos liest nur GPS; Tags/Beschreibungen sind dort wirkungslos, werden aber (dc:Subject + lr:HierarchicalSubject) geschrieben. Der Nutzen von Tags liegt in PhotoCurator selbst — Eingang für die Planung nach M5.
- Alle offenen Review-Findings sind gesammelt als Entwurf [023](../../../requirements/023-robustheit-aus-s03-review.md); dieses Review führt keine eigene Liste.
- Build-Dauern waren deutlich länger als in früheren Sprints (33–114 min je Story); 009 (114 min, ein Review-Zyklus mit fünf Findings, e2e-Harness unter Windows) war der Ausreißer.
- Der Gate-Lauf fand zwei Regressionen, die die Story-Gates nicht sehen konnten (beide in Test-Infrastruktur/Erwartung, nicht im Produktcode): Container lief im e2e als root (009) und ein veraltetes Write-Job-Szenario nach der Skip-Regel (012). Beide behoben.
- Flaky: `places.spec.ts` „place search finds a POI…" (Nominatim-Mock-Log im vollen Lauf zu lang, allein grün). Das Projekt hat noch keine Quarantäne (`e2e-quarantine: none`); als AC7 in 023 aufgenommen.
- Go-Testnamen sind im Docker-Output nicht sichtbar (kein `-v`): die AC-Zuordnung zu Go-Tests beruht auf grünen Paketen und vorhandenen Funktionen. → Follow-up in der Roadmap.
- Fortschritts-Trail: Zeitstempel monoton und plausibel, keine Auffälligkeiten.

## Blockiert / offen

Nichts blockiert. Kein Merge-Blocker.

## Regression-Gate

`build` grün, `test` grün (Go + vitest 55), `e2e` zunächst rot (3 von 19, 6,9 min): `places.spec.ts:25` flaky; `smoke.spec.ts:13` → Story 009, behoben in 024e351; `write-jobs.spec.ts:7` → Story 012, behoben in 79f3744. Fixes durch Wiederlauf der betroffenen Specs bestätigt, kein zweiter Voll-Lauf (Budgetregel). Details in [sprint.md](sprint.md#regression-gate).

## Akzeptanz

Alle Kriterien sind durch Tests belegt (Go-Tests, vitest, Playwright-E2E über die echte Oberfläche; Nominatim-Mock, Tile-Stub); die AC→Test-Zuordnung steht in den Done-Abschnitten der Story-Dateien (Links oben).

- Unterhalb der echten Oberfläche abgedeckt: einzelne Tastaturfälle im TagPicker (013: nur Unit-Logik, nicht E2E; AC4 in 023).
- **Manual residue:** 009 AC9 — ob UGREEN Fotos den geschriebenen Ort nach „Aktualisieren" tatsächlich zeigt (proprietäre App auf dem NAS). Siehe [testplan.md](testplan.md).

## Tier-Record

| Story | Ds | hard Ds | Review-Stufen | Review-Zyklen | Agents | Build (min) |
| --- | --- | --- | --- | --- | --- | --- |
| 009 | 7 | 1 | default | 1 | 12 | 114 |
| 010 | 7 | 1 | default | 1 | 11 | 70 |
| 012 | 4 | 1 | default | 1 | 7 | 63 |
| 013 | 4 | 0 | default | 1 | 6 | 46 |
| 015 | 5 | 0 | default | 1 | 10 | 71 |
| 016 | 3 | 0 | default | 1 | 5 | 33 |
| **Summe** | **30** | **3** | — | **6** | **51** | **397** |

Ein zweistufiges (hard) Review gab es in diesem Sprint nicht; ob es etwas gefunden hätte, was das Default-Review übersah, lässt sich daher nicht sagen. Die drei hard-Ds (009 D3, 010 D2, 012 D1) liefen ohne Beanstandung.
