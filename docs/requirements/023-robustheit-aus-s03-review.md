---
id: 023
title: Robustheit und Testlücken aus dem S03-Review
status: draft # draft -> ready -> in-progress -> done
created: 2026-10-31
---

## Requirement

Das Review von Sprint S03 hat im Schreibpfad (Writer, Jobs, Tags, Orte, Lieblingsorte) Kanten
gefunden, die bewusst nicht im Sprint behoben wurden. Sie sollen gesammelt geschlossen werden,
bevor das MVP abgeschlossen wird (spätestens mit 021).
Quelle: [S03 review](../sprints/done/S03/review.md).

## Acceptance Criteria

- [ ] **AC1** — `POST /api/places` ohne `lat`/`lon` wird abgewiesen statt als Koordinate 0,0 gespeichert (016).
- [ ] **AC2** — Ein DB-Fehler nach dem Status `running` lässt einen Job nicht bis zum nächsten Neustart auf `running` stehen; ein Abbruch des Kontexts direkt nach einem erfolgreichen ExifTool-Aufruf ist durch einen Test belegt (009).
- [ ] **AC3** — Das Entfernen eines Tags erfasst auch abweichende Groß-/Kleinschreibung desselben Keywords in den Keyword-Feldern (012).
- [ ] **AC4** — Pfeiltasten, Escape und Enter auf einem bekannten Tag in anderer Schreibweise im TagPicker sind per E2E belegt; ein neu angelegtes Tag schlägt seine Elternpfade ohne erneutes Öffnen des Dialogs vor (013).
- [ ] **AC5** — Umbenennen/Löschen eines Lieblingsorts, der serverseitig nicht mehr existiert (404), entfernt den Eintrag aus der Liste; die 200-Zeichen-Grenze des Ortsnamens ist durch einen Test belegt; Server- und Client-Sortierung der Lieblingsorte stimmen bei Umlauten überein (016).
- [ ] **AC6** — `RemovePlaceDialog` zeigt die aktuelle statt einer Momentaufnahme der Auswahlzahl (015).
- [ ] **AC7** — `places.spec.ts` „place search finds a POI…" ist im vollen E2E-Lauf stabil (kein Zählerfehler des Nominatim-Mock-Logs durch Vorgänger-Specs).

## Open Questions

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests
