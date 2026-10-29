---
sprint: S03
status: planned # planned | in-progress | done
branch: # set by /sprint
milestone: M3
---

# Sprint S03 — Sicher schreiben und taggen

## Goal

Tags und hierarchische Kategorien lassen sich im Batch zuweisen und entfernen; geschrieben wird
gemäß Spike-Ergebnis sicher in die Dateien — mit Original-Backup, Fehlerprotokoll pro Datei und
Rückgängig pro Job.

## Stories (in build order)

<!-- Order = the order the build phase works through. Dependent stories go last. -->

- [ ] 009 — Sicherer Metadaten-Writer mit Batch-Jobs
- [ ] 010 — Backup der Originale, Änderungsjournal und Rückgängig pro Job
- [ ] 011 — Hierarchische Kategorien definieren und als Baum anzeigen
- [ ] 012 — Tags im Batch zuweisen und entfernen

## Notes

- Definition of Done: [BACKLOG.md](../../BACKLOG.md#definition-of-done).
- **Eingangsbedingung:** `docs/concepts/photocurator-metadaten-strategie.md` enthält das
  Spike-Ergebnis. Fehlt es, startet dieser Sprint nicht.
- Für 009 ist im Refinement `deliverable-hard` für Mapping und Read-back-Prüfung zu erwägen
  (Datenverlust-Risiko).

## Regression gate

<!-- Filled by `/sprint` phase 2b: the commands run, minutes taken, result, commit, and a verdict per failure. -->
