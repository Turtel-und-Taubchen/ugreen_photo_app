# Roadmap

As of: 2026-10-29. One screen: where the project stands, what was done recently, what comes
next. Detail lives where it is produced — sprint reviews, story files, concepts — and this file
links to it. Maintained by `/sprint`, `/concept` and `/roadmap`; the rules are in
`.claude/commands/roadmap.md`.

## Where we stand

- Planung des MVP steht (Konzept, Tech Spec, Backlog mit 21 Stories, Sprints S01–S05); noch nichts gebaut.
- Nächster Schritt: Planung prüfen, dann `/sprint S01`.
- Wartet auf den Nutzer: Freigabe der Planung; offene Punkte im [Konzept §11](concepts/photocurator.md#11-offene-punkte).

## Phase overview

| Phase | Goal | Milestones | Status |
| --- | --- | --- | --- |
| 1 — MVP | Batch-Tags, Batch-Orte und Arbeitslisten schreiben UGREEN-taugliche Metadaten sicher in die Dateien | 0/5 | ▶ **in progress** |
| 2 — Nach dem MVP | Komfort und Erweiterungen laut [Backlog](BACKLOG.md#nach-dem-mvp-vorgemerkt-noch-nicht-geschnitten) | 0/? | planned |

## Current phase: 1 — MVP

Concept: [concepts/photocurator.md](concepts/photocurator.md).

| M | Milestone | Status | Sprints | Note |
| --- | --- | --- | --- | --- |
| M1 | Fundament & Spike-Werkzeug | planned | [S01](sprints/S01/sprint.md) | Danach Spike-Durchführung auf dem NAS durch den Nutzer |
| M2 | Bibliothek sehen und auswählen | planned | [S02](sprints/S02/sprint.md) | Rein lesend, unabhängig vom Spike |
| M3 | Sicher schreiben und taggen | planned | [S03](sprints/S03/sprint.md) | Startet erst mit Spike-Ergebnis |
| M4 | Orte setzen, schneller taggen | planned | [S04](sprints/S04/sprint.md) | |
| M5 | Arbeitslisten und MVP-Abschluss | planned | [S05](sprints/S05/sprint.md) | |

## Open / unprioritised

Ideas and concepts that need a decision before they become work. One line each.

| Topic | State | Next step |
| --- | --- | --- |
| Batch-Beschreibung im MVP? | Offener Punkt 2 im [Konzept](concepts/photocurator.md#11-offene-punkte) | Entscheidung Nutzer |
| Nach-MVP-Themen (Timeline/Karte, KI-Tags, Fernzugriff, Duplikate, …) | vorgemerkt im [Backlog](BACKLOG.md#nach-dem-mvp-vorgemerkt-noch-nicht-geschnitten) | `/roadmap plan` nach M5 |

## Follow-ups worth doing

Small things a sprint surfaced that need no decision and no story yet — for the user while a
sprint runs, or for the next cut. One line each, with where it came from. Done items are
removed, not struck through. A line older than three sprints is promoted to a story or removed
by `/roadmap check`.

## History

Completed phases, one line per milestone; nothing else.

| Phase | M | Milestone | Done | Sprints |
| --- | --- | --- | --- | --- |
