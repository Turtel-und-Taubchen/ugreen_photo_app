# Roadmap

As of: 2026-10-30. One screen: where the project stands, what was done recently, what comes
next. Detail lives where it is produced — sprint reviews, story files, concepts — and this file
links to it. Maintained by `/sprint`, `/concept` and `/roadmap`; the rules are in
`.claude/commands/roadmap.md`.

## Where we stand

- S02 fertig (2026-10-30): M2 komplett (Index, Thumbnails, Raster, Auswahl), dazu Kategorienbaum (011) und Ortssuche (014); Regression-Gate grün.
- Nächster Schritt: S03 (Metadaten-Writer, Backup, Batch-Tagging) — startet mit dem Spike-Ergebnis.
- Wartet auf den Nutzer: Spike-Protokoll vom NAS, NAS-Lauf der Compose-Anleitung, Push.

## Phase overview

| Phase | Goal | Milestones | Status |
| --- | --- | --- | --- |
| 1 — MVP | Batch-Tags, Batch-Orte und Arbeitslisten schreiben UGREEN-taugliche Metadaten sicher in die Dateien | 2/5 | ▶ **in progress** |
| 2 — Nach dem MVP | Komfort und Erweiterungen laut [Backlog](BACKLOG.md#nach-dem-mvp-vorgemerkt-noch-nicht-geschnitten) | 0/? | planned |

## Current phase: 1 — MVP

Concept: [concepts/photocurator.md](concepts/photocurator.md).

| M | Milestone | Status | Sprints | Note |
| --- | --- | --- | --- | --- |
| M1 | Fundament & Spike-Werkzeug | done 2026-10-29 | [S01](sprints/S01/review.md) | Danach Spike-Durchführung auf dem NAS durch den Nutzer |
| M2 | Bibliothek sehen und auswählen | done 2026-10-30 | [S02](sprints/S02/review.md) | |
| M3 | Sicher schreiben und taggen | planned | [S02](sprints/S02/review.md), [S03](sprints/S03/sprint.md) | 011 fertig in S02; Rest startet erst mit Spike-Ergebnis |
| M4 | Orte setzen, schneller taggen | planned | [S02](sprints/S02/review.md), [S03](sprints/S03/sprint.md), [S04](sprints/S04/sprint.md) | 014 fertig in S02; Ortsvorschlag (017) in S04 |
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
- `message()`-Fehlerhelfer im Web dreifach kopiert (Login, Tags, PlacePicker) — zusammenführen; Quelle: [S02 review](sprints/S02/review.md)
- Robustheits-/Testlücken aus S02 als Entwurf [022](requirements/022-robustheit-aus-s02-review.md) — bei nächster Planung einplanen; Quelle: [S02 review](sprints/S02/review.md)

## History

Completed phases, one line per milestone; nothing else.

| Phase | M | Milestone | Done | Sprints |
| --- | --- | --- | --- | --- |
