---
sprint: S02
status: done # planned | in-progress | done
branch: main
milestone: M2, M3, M4
---

# Sprint S02 — Bibliothek sehen und auswählen

## Goal

PhotoCurator liest die Bibliothek aus `Sortiert` (rein lesend), zeigt sie als schnelles,
mobiltaugliches Raster mit Thumbnails für alle Formate und erlaubt Einzel-, Bereichs- und
Gruppenauswahl — die Grundlage aller Batch-Aktionen. Dazu kommen die beiden spike-freien
Bausteine späterer Meilensteine: der Kategorienbaum (App-Zustand + Index) und die Ortssuche über
Nominatim.

## Stories (in build order)

<!-- Order = the order the build phase works through. Dependent stories go last. -->

- [x] 005 — Bibliotheks-Index aus den Dateien in Sortiert
- [x] 006 — Thumbnails mit Cache für Fotos und Videos
- [x] 007 — Fotoraster mit Gruppierung, Ordnernavigation und Detailansicht
- [x] 008 — Mehrfach-, Bereichs- und Gruppenauswahl
- [x] 011 — Hierarchische Kategorien definieren und als Baum anzeigen
- [x] 014 — Ortssuche und Reverse-Geocoding über Nominatim

## Notes

- Definition of Done: [BACKLOG.md](../../../BACKLOG.md#definition-of-done).
- Rein lesend — unabhängig vom Spike-Ergebnis. Läuft parallel zur Spike-Durchführung auf dem NAS.
- 011 und 014 am 2026-10-30 per `/roadmap plan` aus S03/S04 vorgezogen (beide schreiben nichts in
  Dateien). 011 nach 005, weil der Baum Tags aus dem Index zeigt.
- E2E-Tests für 014 nutzen einen Nominatim-Mock; nie den echten Dienst in Tests aufrufen.

## Regression gate

<!-- Filled by `/sprint` phase 2b: the commands run, minutes taken, result, commit, and a verdict per failure. -->

Lauf auf `05d6ab8` (HEAD nach 014), 2026-10-30:

| Befehl | Minuten | Ergebnis |
|---|---|---|
| `build` | < 1 | grün |
| `test` (Go im Docker-Cache, Vitest 45) | < 1 | grün |
| `lint` | < 1 | grün |
| `typecheck` | < 1 | grün |
| `e2e` (13 Flows) | 6,2 | grün |

`e2e-all`: none. Keine Fehler, keine Quarantäne-Einträge, kein unerwarteter Pass.

