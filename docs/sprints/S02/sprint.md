---
sprint: S02
status: planned # planned | in-progress | done
branch: # set by /sprint
milestone: M2
---

# Sprint S02 — Bibliothek sehen und auswählen

## Goal

PhotoCurator liest die Bibliothek aus `Sortiert` (rein lesend), zeigt sie als schnelles,
mobiltaugliches Raster mit Thumbnails für alle Formate und erlaubt Einzel-, Bereichs- und
Gruppenauswahl — die Grundlage aller Batch-Aktionen.

## Stories (in build order)

<!-- Order = the order the build phase works through. Dependent stories go last. -->

- [ ] 005 — Bibliotheks-Index aus den Dateien in Sortiert
- [ ] 006 — Thumbnails mit Cache für Fotos und Videos
- [ ] 007 — Fotoraster mit Gruppierung, Ordnernavigation und Detailansicht
- [ ] 008 — Mehrfach-, Bereichs- und Gruppenauswahl

## Notes

- Definition of Done: [BACKLOG.md](../../BACKLOG.md#definition-of-done).
- Rein lesend — unabhängig vom Spike-Ergebnis. Läuft parallel zur Spike-Durchführung auf dem NAS.

## Regression gate

<!-- Filled by `/sprint` phase 2b: the commands run, minutes taken, result, commit, and a verdict per failure. -->
