---
id: 022
title: Robustheit und Testlücken aus dem S02-Review
status: draft # draft -> ready -> in-progress -> done
created: 2026-10-30
---

## Requirement

Das Review von Sprint S02 hat in Index, Auswahl und Kategorien Kanten gefunden, die bewusst
nicht im Sprint behoben wurden. Keine davon ist im Alltag sichtbar, jede kann aber zu einem
falschen Zustand oder einer unnötigen Startblockade führen. Sie sollen gesammelt geschlossen
werden, bevor das MVP abgeschlossen wird (spätestens mit 021).
Quelle: [S02 review](../sprints/S02/review.md).

## Acceptance Criteria

- [ ] **AC1** — Ist `index.db` keine gültige SQLite-Datei, startet die App trotzdem und baut den Index neu auf, statt den Start abzubrechen (005).
- [ ] **AC2** — Der Auswahlzähler zeigt nach einem fehlgeschlagenen Zähl-Request keinen veralteten Wert als gültig an, und das Verwerfen veralteter Antworten ist durch einen Test belegt (008).
- [ ] **AC3** — Auswahl mit Fotos ohne Aufnahmedatum (`takenAt = null`) ist durch einen Test belegt; der Client überschreitet die Server-Grenze von 500 Gruppen plus Bereichen nicht stillschweigend (008).
- [ ] **AC4** — Tag-Namen mit unsichtbaren Formatzeichen (Unicode Cf, z. B. U+200B) oder ungültigem UTF-8 werden mit einer Begründung abgewiesen (011).
- [ ] **AC5** — Das Menü der Auswahlleiste (`role="menu"`) hat eine Tastatur-Fokusführung (008).

## Open Questions

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests
