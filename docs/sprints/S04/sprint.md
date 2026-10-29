---
sprint: S04
status: planned # planned | in-progress | done
branch: # set by /sprint
milestone: M4
---

# Sprint S04 — Orte setzen und schneller taggen

## Goal

Orte lassen sich per Suche (inkl. Lokale/Hotels) oder Karte im Batch setzen, Lieblingsorte
wiederverwenden und Orte aus zeitlich benachbarten Fotos mit einem Klick übernehmen; Taggen wird
durch zuletzt verwendete Tags und Autovervollständigung schneller.

## Stories (in build order)

<!-- Order = the order the build phase works through. Dependent stories go last. -->

- [ ] 013 — Zuletzt verwendete Tags und Autovervollständigung
- [ ] 014 — Ortssuche und Reverse-Geocoding über Nominatim
- [ ] 015 — Standort im Batch setzen — per Suche oder Karte
- [ ] 016 — Lieblingsorte speichern und wiederverwenden
- [ ] 017 — Ortsvorschlag aus zeitlich benachbarten Fotos

## Notes

- Definition of Done: [BACKLOG.md](../../BACKLOG.md#definition-of-done).
- E2E-Tests nutzen einen Nominatim-Mock; nie den echten Dienst in Tests aufrufen.

## Regression gate

<!-- Filled by `/sprint` phase 2b: the commands run, minutes taken, result, commit, and a verdict per failure. -->
