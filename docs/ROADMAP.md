# Roadmap

As of: 2026-10-31. One screen: where the project stands, what was done recently, what comes
next. Detail lives where it is produced — sprint reviews, story files, concepts — and this file
links to it. Maintained by `/sprint`, `/concept` and `/roadmap`; the rules are in
`.claude/commands/roadmap.md`.

## Where we stand

- S03 fertig (2026-10-31): M3 komplett (Writer, Backup/Rückgängig, Batch-Tagging); dazu Tag-Schnellauswahl (013), Batch-Standort mit Karte (015) und Lieblingsorte (016); Regression-Gate grün nach zwei Fixes.
- Nächster Schritt: S04 (Ortsvorschlag 017, Arbeitslisten, Härtung) — M4 und M5.
- Wartet auf den Nutzer: Merge/Push (Commits liegen auf `main`), manueller Check AC9 in UGREEN Fotos ([Testplan](sprints/done/S03/testplan.md)), NAS-Lauf der Compose-Anleitung.

## Phase overview

| Phase | Goal | Milestones | Status |
| --- | --- | --- | --- |
| 1 — MVP | Batch-Tags, Batch-Orte und Arbeitslisten schreiben UGREEN-taugliche Metadaten sicher in die Dateien | 3/5 | ▶ **in progress** |
| 2 — Nach dem MVP | Komfort und Erweiterungen laut [Backlog](BACKLOG.md#nach-dem-mvp-vorgemerkt-noch-nicht-geschnitten) | 0/? | planned |

## Current phase: 1 — MVP

Concept: [concepts/photocurator.md](concepts/photocurator.md).

| M | Milestone | Status | Sprints | Note |
| --- | --- | --- | --- | --- |
| M1 | Fundament & Spike-Werkzeug | done 2026-10-29 | [S01](sprints/S01/review.md) | Danach Spike-Durchführung auf dem NAS durch den Nutzer |
| M2 | Bibliothek sehen und auswählen | done 2026-10-30 | [S02](sprints/S02/review.md) | |
| M3 | Sicher schreiben und taggen | done 2026-10-31 | [S02](sprints/S02/review.md), [S03](sprints/done/S03/review.md) | |
| M4 | Orte setzen, schneller taggen | in progress | [S02](sprints/S02/review.md), [S03](sprints/done/S03/review.md), [S04](sprints/S04/sprint.md) | 013–016 fertig; Ortsvorschlag (017) in S04 |
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
- `PC_SCAN_INTERVAL`/`PC_SCAN_QUIET` in `deploy/docker-compose.yml` + `deploy/.env.example` durchreichen (für Agenten gesperrt); Quelle: [S02 review](sprints/S02/review.md)
- Tech Spec: Endpunkttabelle um `POST /api/selection/count` ergänzen, §5 ExifTool-Leseweise prüfen; Quelle: [S02 review](sprints/S02/review.md)
- Verwaiste Thumbnail-Dateien in `/data/thumbs` aufräumen; Quelle: [S02 review](sprints/S02/review.md)
- `COALESCE(taken_at,'')` in Listen-/Nachbar-SQL nutzt keinen Index — in 021 mit großer Bibliothek messen; Quelle: [S02 review](sprints/S02/review.md)
- Go-Testnamen erscheinen im Docker-`test`-Stage nicht (kein `-v`) — AC→Test-Nachweis nur auf Paketebene; `-v`/JSON-Ausgabe prüfen; Quelle: [S03 review](sprints/done/S03/review.md)
- Flaky `places.spec.ts` „place search finds a POI…" im vollen E2E-Lauf; Projekt hat noch keine Quarantäne (`e2e-quarantine`); Quelle: [S03 review](sprints/done/S03/review.md)
- Robustheits-/Testlücken aus S03 als Entwurf [023](requirements/023-robustheit-aus-s03-review.md) — bei nächster Planung einplanen; Quelle: [S03 review](sprints/done/S03/review.md)
- Robustheits-/Testlücken aus S02 als Entwurf [022](requirements/022-robustheit-aus-s02-review.md) — bei nächster Planung einplanen; Quelle: [S02 review](sprints/S02/review.md)

## History

Completed phases, one line per milestone; nothing else.

| Phase | M | Milestone | Done | Sprints |
| --- | --- | --- | --- | --- |
