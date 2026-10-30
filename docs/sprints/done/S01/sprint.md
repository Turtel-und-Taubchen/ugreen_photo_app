---
sprint: S01
status: done
branch: main
milestone: M1
---

# Sprint S01 — Fundament & Spike-Werkzeug

## Goal

Das Projektgerüst steht mit funktionierenden Verify-Befehlen und E2E-Harness, der Container läuft
abgesichert als Compose-Projekt mit Anmeldung, und das Spike-Werkzeug erzeugt den Testsatz samt
Protokoll, mit dem auf dem NAS geklärt wird, welche Metadatenfelder UGREEN Fotos liest.

## Stories (in build order)

<!-- Order = the order the build phase works through. Dependent stories go last. -->

- [x] 002 — Projektgerüst — Go-Backend, Svelte-Frontend, Verify-Befehle, E2E-Harness
- [x] 001 — Spike — welche Metadatenfelder liest UGREEN Fotos?
- [x] 003 — Deployment als Docker-Compose-Projekt auf UGOS Pro
- [x] 004 — Einfache Anmeldung für zwei Nutzer

## Notes

- Definition of Done: [BACKLOG.md](../../../BACKLOG.md#definition-of-done).
- 002 zuerst: Es schafft die Verify-Befehle, die `.claude/ai-scrum.md` schon nennt.
- **Nach dem Sprint (Nutzer):** Testsatz aus 001 nach `Sortiert/_pcprobe/` kopieren, in UGREEN
  Fotos prüfen, `PROTOKOLL.md` ausfüllen, Ergebnis in
  `docs/concepts/photocurator-metadaten-strategie.md` übertragen, Testordner wieder löschen.
  Muss vor dem Refinement von S03 erledigt sein, nicht vor S02.
- Kein Schreiben in echte Fotos in diesem Sprint — nur in generierte Testdateien.

## Regression gate

Lauf auf `ad4eb4d` (HEAD nach Story 004), je einmal, alle grün:

- `npm --prefix web run build && docker build --target runtime -t photocurator:dev .` — grün, <1 min
- `docker build --target test . && npm --prefix web run test` — grün, <1 min (Go-Test-Layer aus dem Cache, Vitest 7/7)
- `npm --prefix web run test:e2e` — grün, <1 min, 4/4 (auth, deployment, smoke ×2)
- `e2e-all`: `none` (nicht gesetzt)

Keine Fehler, keine Quarantäne-Einträge. Hinweis: Das Go-`test`-Stage kam aus dem Docker-Cache; die Go-Tests wurden in den Story-Gates je frisch gebaut und liefen dort grün.
