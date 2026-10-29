---
id: 005
title: Bibliotheks-Index aus den Dateien in Sortiert
status: draft # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E1 Index & Ansicht
estimate: 8
sprint: S02
---

## Requirement

PhotoCurator muss alle Fotos und Videos in `Sortiert` kennen — mit Aufnahmedatum, vorhandenen
Tags, Ort und Prüfstatus —, um sie schnell anzeigen und filtern zu können. Die Dateien sind die
Wahrheit: Der Index ist nur ein Cache, darf jederzeit gelöscht werden und wird dann vollständig
aus den Dateien neu aufgebaut. Neue Dateien von Phockup sollen von selbst auftauchen, ohne dass
halb verschobene Dateien gelesen werden. Details: [Tech Spec §2.1, §5](../concepts/photocurator-tech-spec.md).

## Acceptance Criteria

- [ ] **AC1** — Ein Erstscan erfasst rekursiv alle Dateien mit den Endungen jpg, jpeg, heic,
  heif, png, mp4, mov (Groß-/Kleinschreibung egal) unter `/photos`; andere Dateien und
  versteckte bzw. System-Ordner (z. B. `@eaDir`, `.*`) werden ignoriert.
- [ ] **AC2** — Für jede Datei stehen im Index: relativer Pfad, Ordner (JJJJ/MM), Typ und Format,
  Größe, Änderungszeit, Aufnahmedatum mit Quelle, Tags (aus allen Keyword-Feldern
  zusammengeführt, Hierarchie aus `HierarchicalSubject`), GPS, Ortsfelder und Prüfstatus.
- [ ] **AC3** — Das Aufnahmedatum wird in dieser Reihenfolge ermittelt: EXIF
  `DateTimeOriginal` → QuickTime `CreateDate` → Datum im Dateinamen (WhatsApp-Muster
  `IMG-JJJJMMTT-WA…`, `VID-JJJJMMTT-WA…`, `JJJJMMTT_hhmmss`) → Änderungszeit; die Quelle wird
  gespeichert.
- [ ] **AC4** — Ein Folgescan liest nur neue oder geänderte (Größe/Änderungszeit) Dateien mit
  ExifTool; entfernte Dateien verschwinden aus dem Index.
- [ ] **AC5** — Dateien, deren Änderungszeit jünger als die Ruhezeit ist (Platzhalter: 5 min),
  werden übersprungen und beim nächsten Scan erfasst.
- [ ] **AC6** — Der Scan läuft beim Start, periodisch (Platzhalter: alle 15 min) und auf Knopfdruck
  „Neu einlesen"; während des Scans zeigt die Oberfläche Status und Fortschritt.
- [ ] **AC7** — Nach Löschen von `index.db` und Neustart enthält der neu aufgebaute Index für die
  Fixture-Bibliothek dieselben Tags, Orte und Prüfstatus wie vorher.
- [ ] **AC8** — Eine nicht lesbare Datei bricht den Scan nicht ab; der Fehler wird protokolliert
  und die Datei in der Oberfläche als „nicht lesbar" gekennzeichnet.
- [ ] **AC9** — Der Scan verändert keine Datei unter `/photos` (Prüfsummen und Änderungszeiten
  der Fixture-Bibliothek sind vorher und nachher gleich).

## Open Questions

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
