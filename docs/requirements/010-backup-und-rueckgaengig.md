---
id: 010
title: Backup der Originale, Änderungsjournal und Rückgängig pro Job
status: draft # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E2 Sicheres Schreiben
estimate: 5
sprint: S03
---

## Requirement

Wenn ich mich vertan habe („falscher Ort für 300 Fotos"), will ich den ganzen Job mit einem
Klick rückgängig machen. Und falls doch einmal eine Datei Schaden nimmt, soll eine Kopie des
Originals existieren — außerhalb der Fotobibliothek, damit UGREEN Fotos keine Duplikate zeigt.
Entschieden: Journal + einmalige Original-Kopie mit Aufbewahrungsfrist.

## Acceptance Criteria

- [ ] **AC1** — Vor dem ersten Schreiben auf eine Datei liegt eine Kopie unter
  `/backup/<JJJJ-MM-TT>/<relativer Pfad>`; existiert innerhalb der Aufbewahrungsfrist schon eine
  Kopie, wird keine zweite angelegt.
- [ ] **AC2** — Schlägt das Backup fehl (z. B. kein Platz), wird die Datei nicht verändert und im
  Job als fehlgeschlagen mit Grund gemeldet.
- [ ] **AC3** — Für jede geschriebene Datei speichert das Journal Job, Nutzer, Zeitpunkt sowie die
  alten und neuen Werte der betroffenen Felder.
- [ ] **AC4** — „Rückgängig" an einem Job stellt für alle seine Dateien die alten Werte wieder
  her; der Rückgängig-Vorgang ist selbst ein Job mit Ergebnis pro Datei.
- [ ] **AC5** — Dateien, deren Werte seit dem Job erneut geändert wurden, werden beim
  Rückgängigmachen übersprungen und als Konflikt gemeldet.
- [ ] **AC6** — Backups älter als die Aufbewahrungsfrist (Platzhalter: 30 Tage) werden täglich
  gelöscht; jüngere bleiben.
- [ ] **AC7** — Eine Jobliste zeigt die letzten Jobs mit Nutzer, Zeitpunkt, Aktion, Anzahl
  Dateien und Ergebnis.
- [ ] **AC8** — `deploy/README.md` beschreibt, wie eine Datei manuell aus `/backup`
  wiederhergestellt wird und wie `/data/app.db` gesichert wird.

## Open Questions

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
