---
sprint: S01
status: in-progress
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

- [ ] 002 — Projektgerüst — Go-Backend, Svelte-Frontend, Verify-Befehle, E2E-Harness
- [ ] 001 — Spike — welche Metadatenfelder liest UGREEN Fotos?
- [ ] 003 — Deployment als Docker-Compose-Projekt auf UGOS Pro
- [ ] 004 — Einfache Anmeldung für zwei Nutzer

## Notes

- Definition of Done: [BACKLOG.md](../../BACKLOG.md#definition-of-done).
- 002 zuerst: Es schafft die Verify-Befehle, die `.claude/ai-scrum.md` schon nennt.
- **Nach dem Sprint (Nutzer):** Testsatz aus 001 nach `Sortiert/_pcprobe/` kopieren, in UGREEN
  Fotos prüfen, `PROTOKOLL.md` ausfüllen, Ergebnis in
  `docs/concepts/photocurator-metadaten-strategie.md` übertragen, Testordner wieder löschen.
  Muss vor dem Refinement von S03 erledigt sein, nicht vor S02.
- Kein Schreiben in echte Fotos in diesem Sprint — nur in generierte Testdateien.

## Regression gate

<!-- Filled by `/sprint` phase 2b: the commands run, minutes taken, result, commit, and a verdict per failure. -->
