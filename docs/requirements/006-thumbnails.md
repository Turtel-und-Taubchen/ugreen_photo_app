---
id: 006
title: Thumbnails mit Cache für Fotos und Videos
status: draft # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E1 Index & Ansicht
estimate: 5
sprint: S02
---

## Requirement

Damit ich Fotos auswählen kann, muss ich sie sehen — schnell, auch am Handy und auch für HEIC
und Videos. Thumbnails werden einmal erzeugt und zwischengespeichert, ohne das NAS lahmzulegen
und ohne etwas in den Fotoordner zu schreiben. Details: [Tech Spec §4](../concepts/photocurator-tech-spec.md).

## Acceptance Criteria

- [ ] **AC1** — Für JPEG, PNG und HEIC entstehen ein Raster-Thumbnail (ca. 320 px) und eine
  Detailvorschau (ca. 1600 px) als WebP, mit korrekt angewandter EXIF-Ausrichtung.
- [ ] **AC2** — Für MP4 und MOV entsteht das Vorschaubild aus einem Frame (bei ca. 1 s, bei
  kürzeren Videos aus dem ersten Frame); im Raster tragen Videos ein Video-Symbol.
- [ ] **AC3** — Ein zweiter Abruf desselben Thumbnails erzeugt es nicht neu; ändert sich die
  Datei (Größe/Änderungszeit), wird es neu erzeugt.
- [ ] **AC4** — Die Erzeugung läuft im Hintergrund mit begrenzter Parallelität (Platzhalter: 1
  Worker, niedrige Prozesspriorität); ein noch fehlendes, gerade sichtbares Thumbnail wird
  vorgezogen und bis dahin durch einen Platzhalter ersetzt.
- [ ] **AC5** — Thumbnail-Antworten tragen langlebige Cache-Header; der Browser lädt ein bereits
  geladenes Thumbnail beim erneuten Scrollen nicht erneut vom Server.
- [ ] **AC6** — Für eine defekte Datei erscheint ein Fehlerplatzhalter; die App läuft weiter.
- [ ] **AC7** — Es wird nichts unter `/photos` angelegt oder verändert; der Cache liegt unter
  `/data`.

## Open Questions

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
