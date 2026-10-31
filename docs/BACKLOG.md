# Product Backlog — PhotoCurator

Stand: 2026-10-29. Übersicht über Epics und Stories. **Die Details (Anforderung,
Akzeptanzkriterien, offene Fragen) stehen in den Story-Dateien** unter
[requirements/](requirements/) — diese Datei wiederholt sie nicht. Status und Fortschritt stehen
in der [Roadmap](ROADMAP.md). Grundlage: [Konzept](concepts/photocurator.md),
[Tech Spec](concepts/photocurator-tech-spec.md).

Schätzung in Story Points (Fibonacci: 1, 2, 3, 5, 8). Bezugsgröße: 3 = klar umrissen, ein
Bereich (Backend oder UI), wenige Tests; 8 = mehrere Komponenten, echte Datei-I/O-Risiken.

## MVP

| Epic | Story | Titel | SP | Sprint |
| --- | --- | --- | --- | --- |
| E0 Fundament & Spike | [001](requirements/001-spike-ugreen-metadatenfelder.md) | Spike — welche Metadatenfelder liest UGREEN Fotos? | 5 | S01 |
| | [002](requirements/002-projektgeruest.md) | Projektgerüst, Verify-Befehle, E2E-Harness | 3 | S01 |
| | [003](requirements/003-docker-compose-deployment.md) | Deployment als Compose-Projekt auf UGOS Pro | 3 | S01 |
| | [004](requirements/004-anmeldung-zwei-nutzer.md) | Anmeldung für zwei Nutzer | 3 | S01 |
| E1 Index & Ansicht | [005](requirements/005-bibliotheks-index.md) | Bibliotheks-Index aus den Dateien | 8 | S02 |
| | [006](requirements/006-thumbnails.md) | Thumbnails mit Cache (inkl. HEIC, Video) | 5 | S02 |
| | [007](requirements/007-fotoraster-und-navigation.md) | Fotoraster, Navigation, Detailansicht | 5 | S02 |
| | [008](requirements/008-auswahl.md) | Mehrfach-, Bereichs-, Gruppenauswahl | 3 | S02 |
| E2 Sicheres Schreiben | [009](requirements/009-metadaten-writer.md) | Sicherer Metadaten-Writer mit Batch-Jobs | 8 | S03 |
| | [010](requirements/010-backup-und-rueckgaengig.md) | Backup, Journal, Rückgängig | 5 | S03 |
| E3 Tags & Kategorien | [011](requirements/011-kategorienbaum.md) | Hierarchische Kategorien | 3 | S02 |
| | [012](requirements/012-batch-tagging.md) | Tags im Batch zuweisen/entfernen | 5 | S03 |
| | [013](requirements/013-tag-schnellauswahl-autocomplete.md) | Zuletzt verwendet + Autovervollständigung | 3 | S03 |
| E4 Standorte | [014](requirements/014-ortssuche-nominatim.md) | Ortssuche/Reverse-Geocoding (Nominatim) | 5 | S02 |
| | [015](requirements/015-karte-und-batch-standort.md) | Standort im Batch per Suche/Karte | 5 | S03 |
| | [016](requirements/016-lieblingsorte.md) | Lieblingsorte | 2 | S03 |
| | [017](requirements/017-ortsvorschlag-zeitnachbarn.md) | Ortsvorschlag aus Zeitnachbarn | 5 | S04 |
| E5 Arbeitsliste | [018](requirements/018-arbeitslisten-filter.md) | Arbeitslisten-Filter | 5 | S04 |
| | [019](requirements/019-erledigt-markierung.md) | „Erledigt" / „nicht relevant" | 3 | S04 |
| | [020](requirements/020-fortschrittsanzeige.md) | Fortschrittsanzeige | 2 | S04 |
| E6 Betrieb | [021](requirements/021-mvp-haertung.md) | MVP-Härtung: Last, Ressourcen, Betriebshandbuch | 5 | S04 |

**Summe MVP: 91 SP** in 4 Sprints (S01 14 · S02 29 · S03 28 · S04 20).

### Abhängigkeiten

```
002 ─┬─► 001 (Spike-Werkzeug) ──► [Spike auf dem NAS, manuell] ──► 009 ─► 010, 012, 015, 019
     ├─► 003, 004
     └─► 005 ─► 006, 007 ─► 008 ─► 012, 015, 018
011 ─► 012 ─► 013          014 ─► 015 ─► 016, 017
```

Der Spike blockiert nur das Schreiben (ab S03). S02 ist rein lesend und überbrückt die Zeit, in
der der Spike auf dem NAS durchgeführt und ausgewertet wird.

## Nach dem MVP (vorgemerkt, noch nicht geschnitten)

| Epic | Thema | Grobe Größe | Notiz |
| --- | --- | --- | --- |
| E7 | Beschreibung im Batch setzen (`dc:description` u. a.) | 3 | Offener Punkt 2 im Konzept — evtl. ins MVP |
| E7 | Tags umbenennen/zusammenführen über alle Dateien | 8 | Massenschreiben, braucht bewährten Writer |
| E7 | Aufnahmedatum setzen (WhatsApp-Bilder ohne Datum) | 5 | Achtung: beeinflusst UGREEN-Zeitleiste. Arbeitsliste `Sortiert/unknown` (Phockup legt Dateien ohne EXIF-Datum dort ab); Vorschlag: Datum aus Dateiname (`IMG-JJJJMMTT-WA…`, „good enough“), Uhrzeit aus mtime bei gleichem Tag, sonst 00:00; `DateTimeOriginal` per Writer-Job mit Backup/Undo. Kein Verschieben durch PhotoCurator — Zurücklegen in die Inbox manuell, Phockup sortiert ein |
| E8 | Eigene Timeline- und Kartenansicht zum Durchstöbern | 13 | Laut Auftrag nicht im MVP |
| E9 | KI-Vorschläge für Tags | 13+ | Ressourcen auf dem NAS kritisch; ggf. extern |
| E10 | Zugriff von unterwegs | 5 | Sicherheitskonzept nötig (VPN/Reverse-Proxy/2FA) |
| E11 | Duplikaterkennung | 8 | Laut Auftrag nicht im MVP |
| E12 | Inotify statt periodischem Scan | 3 | Nur falls Scan-Last messbar stört (Story 021) |
| E12 | Videowiedergabe in PhotoCurator | 3 | HEVC im Browser problematisch |

## Definition of Done

### Pro Story

1. Jedes Akzeptanzkriterium ist durch einen benannten automatischen Test belegt und grün;
   Kriterien mit Nutzerinteraktion über die echte Oberfläche (Playwright). „Manual residue" nur
   für Prüfungen auf dem echten NAS bzw. in der UGREEN Fotos-App, mit Begründung.
2. `build`, `lint`, `typecheck`, `test` und die eigenen E2E-Tests (Profil `.claude/ai-scrum.md`)
   sind grün; Review durch einen frischen Agenten ist durch.
3. Keine Datei außerhalb von `/photos`, `/data`, `/backup` wird angefasst; in `/photos` entstehen
   keine Hilfsdateien. Tests arbeiten nur mit Kopien der Fixtures, nie mit echten Fotos.
4. Jeder Schreibzugriff auf Fotos läuft über den Writer (ab Story 009) — kein direkter
   ExifTool-Aufruf an anderer Stelle.
5. Neue Hintergrundarbeit hat eine begrenzte Parallelität und läuft mit niedriger Priorität.
6. Tech Spec bzw. Metadaten-Strategie sind aktualisiert, wenn die Story davon abweicht.
7. Doku auf Deutsch; Code, Bezeichner und Kommentare auf Englisch.
8. Story-Datei mit ausgefülltem `## Done`, nach `requirements/done/` verschoben, Zeile in
   `done/INDEX.md`.

### Pro Sprint

1. Alle Stories `done` oder als blockiert mit Grund im `review.md`.
2. Regression-Gate: vollständige `test`- und `e2e`-Suite grün auf dem Sprint-Branch.
3. Das Runtime-Image baut, `deploy/docker-compose.yml` ist gültig.
4. `review.md` mit Abnahme-Nachweis (welcher Test welches Kriterium belegt) und der Liste der
   manuellen Restprüfungen.

### MVP (Release)

1. S01–S04 abgeschlossen, Spike-Ergebnis eingearbeitet.
2. Auf dem DXP4800 Plus installiert, echte Bibliothek eingelesen.
3. Mindestens ein im Batch gesetzter Tag und Ort ist in UGREEN Fotos in Suche und einem
   bedingten Album sichtbar (Story 021, AC5).
4. Ressourcen-Messwerte aus Story 021 liegen innerhalb der Zielwerte.
