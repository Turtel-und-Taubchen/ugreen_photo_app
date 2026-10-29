---
id: 007
title: Fotoraster mit Gruppierung, Ordnernavigation und Detailansicht
status: draft # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E1 Index & Ansicht
estimate: 5
sprint: S02
---

## Requirement

Ich möchte die Bibliothek als Raster sehen, nach Tagen und Monaten gruppiert, schnell zu einem
Monat springen und ein Foto groß ansehen können — am PC und am Handy, auch bei mehreren Tausend
Fotos flüssig. Das Raster ist die Grundlage für Auswahl und Arbeitslisten, keine
Betrachter-Galerie (das bleibt UGREEN Fotos).

## Acceptance Criteria

- [ ] **AC1** — Die Startansicht zeigt alle Fotos und Videos nach Aufnahmedatum absteigend,
  gruppiert nach Monat und Tag, jeweils mit Überschrift und Anzahl.
- [ ] **AC2** — Eine Navigation nach Jahr/Monat (aus der Ordnerstruktur JJJJ/MM) springt zur
  entsprechenden Gruppe.
- [ ] **AC3** — Bei 5.000 Einträgen in der Fixture-Datenbank enthält die Seite zu jedem Zeitpunkt
  höchstens eine begrenzte Zahl Bildelemente (Platzhalter: 300); beim Scrollen werden weitere
  Einträge nachgeladen.
- [ ] **AC4** — Ein Klick bzw. Tippen außerhalb des Auswahlmodus öffnet eine Detailansicht mit
  großer Vorschau, Aufnahmedatum (mit Hinweis, wenn es nur aus Dateiname/Änderungszeit stammt),
  Tags, Ortsname, Prüfstatus und Dateipfad.
- [ ] **AC5** — In der Detailansicht wechseln Pfeiltasten bzw. Wischen zum vorigen/nächsten Foto.
- [ ] **AC6** — Bei 390 px Breite zeigt das Raster mindestens 3 Spalten ohne horizontales
  Scrollen; bei 1440 px Breite mindestens 6 Spalten.

## Open Questions

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
