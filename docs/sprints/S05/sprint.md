---
sprint: S05
status: planned # planned | in-progress | done
branch: # set by /sprint
milestone: M5
---

# Sprint S05 — Arbeitslisten und MVP-Abschluss

## Goal

Arbeitslisten zeigen gezielt, was noch fehlt, Fotos lassen sich als erledigt/nicht relevant
abhaken, der Fortschritt ist sichtbar — und das MVP ist auf dem NAS unter Last vermessen und
mit Betriebshandbuch einsatzbereit.

## Stories (in build order)

<!-- Order = the order the build phase works through. Dependent stories go last. -->

- [ ] 019 — Fotos als „erledigt" oder „nicht relevant" markieren
- [ ] 018 — Arbeitslisten-Filter („Inbox zum Abarbeiten")
- [ ] 020 — Fortschrittsanzeige
- [ ] 021 — MVP-Härtung — Last, Ressourcen, Betriebshandbuch

## Notes

- Definition of Done inkl. MVP-Release-Kriterien: [BACKLOG.md](../../BACKLOG.md#definition-of-done).
- 019 vor 018, weil der Filter das Ausblenden erledigter Fotos braucht.

## Regression gate

<!-- Filled by `/sprint` phase 2b: the commands run, minutes taken, result, commit, and a verdict per failure. -->
