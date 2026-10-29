---
id: 012
title: Tags im Batch zuweisen und entfernen
status: draft # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E3 Tags & Kategorien
estimate: 5
sprint: S03
---

## Requirement

Kernfunktion: ausgewählte Fotos in einem Schritt mit einem oder mehreren Tags versehen oder
Tags von ihnen entfernen — dauerhaft in den Dateien, sodass UGREEN Fotos sie für Suche und
bedingte Alben nutzen kann.

## Acceptance Criteria

- [ ] **AC1** — Für eine Auswahl fügt „Tags zuweisen" einen oder mehrere Tags hinzu; nach
  Abschluss des Jobs zeigen alle ausgewählten Fotos die Tags in der Oberfläche, und ein
  Read-back der Dateien bestätigt sie.
- [ ] **AC2** — „Tags entfernen" listet die Tags der Auswahl mit der Anzahl betroffener Fotos;
  gewählte Tags werden aus allen ausgewählten Dateien und aus allen Feldern der
  Schreibstrategie entfernt.
- [ ] **AC3** — Zuweisen von `Anlass/Geburtstag` schreibt Pfad und Begriffe gemäß
  Schreibstrategie; beim Entfernen bleibt ein Elternbegriff (`Anlass`) erhalten, solange ein
  anderer Pfad der Datei ihn noch braucht (z. B. `Anlass/Hochzeit`).
- [ ] **AC4** — Fotos, die einen zuzuweisenden Tag schon tragen bzw. einen zu entfernenden nicht
  tragen, werden nicht neu geschrieben (Änderungszeit und Inhalt unverändert).
- [ ] **AC5** — Ein neu eingegebener Tag, den es noch nicht gibt, wird beim Zuweisen angelegt.
- [ ] **AC6** — Die Aktion funktioniert für Einzel-, Bereichs-, Tages- und Monatsauswahl
  (inkl. noch nicht geladener Fotos) sowie für Fotos und Videos.

## Open Questions

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
