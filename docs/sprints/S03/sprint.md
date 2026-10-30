---
sprint: S03
status: planned # planned | in-progress | done
branch: # set by /sprint
milestone: M3, M4
---

# Sprint S03 — Sicher schreiben: Tags und Orte

## Goal

Tags und Orte lassen sich im Batch in die Dateien schreiben — gemäß Spike-Ergebnis, mit
Original-Backup, Fehlerprotokoll pro Datei und Rückgängig pro Job. Orte kommen per Suche, Karte
oder Lieblingsort; Taggen wird durch zuletzt verwendete Tags und Autovervollständigung schneller.

## Stories (in build order)

<!-- Order = the order the build phase works through. Dependent stories go last. -->

- [ ] 009 — Sicherer Metadaten-Writer mit Batch-Jobs
- [ ] 010 — Backup der Originale, Änderungsjournal und Rückgängig pro Job
- [ ] 012 — Tags im Batch zuweisen und entfernen
- [ ] 013 — Zuletzt verwendete Tags und Autovervollständigung
- [ ] 015 — Standort im Batch setzen — per Suche oder Karte
- [ ] 016 — Lieblingsorte speichern und wiederverwenden

## Notes

- Definition of Done: [BACKLOG.md](../../BACKLOG.md#definition-of-done).
- **Eingangsbedingung:** `docs/concepts/photocurator-metadaten-strategie.md` enthält das
  Spike-Ergebnis. Fehlt es, startet dieser Sprint nicht.
- Für 009 ist im Refinement `deliverable-hard` für Mapping und Read-back-Prüfung zu erwägen
  (Datenverlust-Risiko).
- Am 2026-10-30 per `/roadmap plan` neu geschnitten: 011 nach S02 vorgezogen; 013, 015, 016 aus
  dem alten S04 übernommen. 011 (S02) und 014 (S02) sind Voraussetzung für 012 bzw. 015.
- E2E-Tests nutzen einen Nominatim-Mock; nie den echten Dienst in Tests aufrufen.

## Regression gate

<!-- Filled by `/sprint` phase 2b: the commands run, minutes taken, result, commit, and a verdict per failure. -->
