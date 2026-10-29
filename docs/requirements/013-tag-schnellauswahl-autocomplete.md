---
id: 013
title: Zuletzt verwendete Tags und Autovervollständigung
status: draft # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E3 Tags & Kategorien
estimate: 3
sprint: S04
---

## Requirement

Beim Taggen will ich nicht tippen müssen, was ich gerade eben schon verwendet habe, und bei
längeren Namen reichen ein paar Buchstaben — auch am Handy.

## Acceptance Criteria

- [ ] **AC1** — Der Tag-Dialog zeigt die zuletzt verwendeten Tags (Platzhalter: 8) als
  antippbare Chips, der zuletzt verwendete zuerst.
- [ ] **AC2** — Die Eingabe schlägt ab dem ersten Zeichen passende bekannte Tags vor —
  Teilstring auch innerhalb des Pfades, unabhängig von Groß-/Kleinschreibung und mit Umlauten
  (z. B. „öst" findet `Reise/Österreich`).
- [ ] **AC3** — Bei 1.000 bekannten Tags erscheinen die Vorschläge innerhalb von 100 ms nach der
  Eingabe (Messung im Test).
- [ ] **AC4** — Enter oder Tippen auf „neu anlegen" übernimmt einen noch unbekannten Begriff als
  neuen Tag; die Validierung aus Story 011 gilt.
- [ ] **AC5** — Bei 390 px Breite sind Chips und Vorschläge ohne Scrollen der Seite erreichbar
  und mindestens 44 px hoch.

## Open Questions

- „Zuletzt verwendet": gemeinsam für beide Personen oder pro Person?

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
