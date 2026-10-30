# Roadmap

As of: 2026-10-30. One screen: where the project stands, what was done recently, what comes
next. Detail lives where it is produced — sprint reviews, story files, concepts — and this file
links to it. Maintained by `/sprint`, `/concept` and `/roadmap`; the rules are in
`.claude/commands/roadmap.md`.

## Where we stand

- S01 (M1) ist gebaut: Projektgerüst, Spike-Werkzeug, Compose-Deployment, Anmeldung; Regression-Gate grün.
- S02 läuft (Refinement gestartet); am 2026-10-30 neu geschnitten: 011 + 014 vorgezogen, alter S05 in S04 aufgegangen — MVP in S02–S04.
- Nächster Schritt: S02 fertig bauen; parallel Spike auf dem NAS durchführen (Eingangsbedingung für S03).
- Wartet auf den Nutzer: Spike-Protokoll, NAS-Lauf der Compose-Anleitung, Push/Merge.

## Phase overview

| Phase | Goal | Milestones | Status |
| --- | --- | --- | --- |
| 1 — MVP | Batch-Tags, Batch-Orte und Arbeitslisten schreiben UGREEN-taugliche Metadaten sicher in die Dateien | 1/5 | ▶ **in progress** |
| 2 — Nach dem MVP | Komfort und Erweiterungen laut [Backlog](BACKLOG.md#nach-dem-mvp-vorgemerkt-noch-nicht-geschnitten) | 0/? | planned |

## Current phase: 1 — MVP

Concept: [concepts/photocurator.md](concepts/photocurator.md).

| M | Milestone | Status | Sprints | Note |
| --- | --- | --- | --- | --- |
| M1 | Fundament & Spike-Werkzeug | done 2026-10-29 | [S01](sprints/S01/review.md) | Danach Spike-Durchführung auf dem NAS durch den Nutzer |
| M2 | Bibliothek sehen und auswählen | in progress | [S02](sprints/S02/sprint.md) | Rein lesend, unabhängig vom Spike |
| M3 | Sicher schreiben und taggen | planned | [S02](sprints/S02/sprint.md), [S03](sprints/S03/sprint.md) | Kategorienbaum (011) in S02; Rest startet erst mit Spike-Ergebnis |
| M4 | Orte setzen, schneller taggen | planned | [S02](sprints/S02/sprint.md), [S03](sprints/S03/sprint.md), [S04](sprints/S04/sprint.md) | Ortssuche (014) in S02, Ortsvorschlag (017) in S04 |
| M5 | Arbeitslisten und MVP-Abschluss | planned | [S04](sprints/S04/sprint.md) | |

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

- Image `photocurator:dev` ist ~978 MB (ffmpeg-Abhängigkeiten) — gegen NAS-Ziele prüfen; Quelle: [S01 review](sprints/S01/review.md)
- `internal/config` nutzt `x/sys/unix` ohne Build-Tag, Go baut nur im Docker; Quelle: [S01 review](sprints/S01/review.md)
- Login-Limiter räumt seine Map nur bei erneutem Zugriff derselben IP auf; Quelle: [S01 review](sprints/S01/review.md)
- Session-Lookup läuft auch für statische Assets; Quelle: [S01 review](sprints/S01/review.md)
- 360-px-Layout der Login-Seite visuell prüfen; Quelle: [S01 review](sprints/S01/review.md)
- E2E-Setup-Fehler überspringt Teardown (nächster Lauf räumt auf); Quelle: [S01 review](sprints/S01/review.md)
- `test-support` im Profil auf `internal/testsupport/` setzen; Quelle: [S01 review](sprints/S01/review.md)

## History

Completed phases, one line per milestone; nothing else.

| Phase | M | Milestone | Done | Sprints |
| --- | --- | --- | --- | --- |
