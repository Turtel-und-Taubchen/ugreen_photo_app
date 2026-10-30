---
sprint: S04
status: planned # planned | in-progress | done
branch: # set by /sprint
milestone: M4, M5
---

# Sprint S04 — Arbeitslisten und MVP-Abschluss

## Goal

Orte aus zeitlich benachbarten Fotos lassen sich mit einem Klick übernehmen; Arbeitslisten zeigen
gezielt, was noch fehlt, Fotos lassen sich als erledigt/nicht relevant abhaken, der Fortschritt
ist sichtbar — und das MVP ist auf dem NAS unter Last vermessen und mit Betriebshandbuch
einsatzbereit.

## Stories (in build order)

<!-- Order = the order the build phase works through. Dependent stories go last. -->

- [ ] 017 — Ortsvorschlag aus zeitlich benachbarten Fotos
- [ ] 019 — Fotos als „erledigt" oder „nicht relevant" markieren
- [ ] 018 — Arbeitslisten-Filter („Inbox zum Abarbeiten")
- [ ] 020 — Fortschrittsanzeige
- [ ] 021 — MVP-Härtung — Last, Ressourcen, Betriebshandbuch

## Notes

- Definition of Done inkl. MVP-Release-Kriterien: [BACKLOG.md](../../BACKLOG.md#definition-of-done).
- Am 2026-10-30 per `/roadmap plan` neu geschnitten: alter S05 hier aufgegangen, 017 aus dem alten
  S04 übernommen.
- 019 vor 018, weil der Filter das Ausblenden erledigter Fotos braucht.
- 021 zuletzt, weil die Härtung den vollständigen MVP vermisst.

## Regression gate

<!-- Filled by `/sprint` phase 2b: the commands run, minutes taken, result, commit, and a verdict per failure. -->
