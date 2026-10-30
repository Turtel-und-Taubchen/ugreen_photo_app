---
id: 015
title: Standort im Batch setzen — per Suche oder Karte
status: draft # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E4 Standorte
estimate: 5
sprint: S03
---

## Requirement

Für die ausgewählten Fotos setze ich einen Ort — gefunden per Suche oder per Klick auf eine
Karte. Gespeichert werden Koordinaten und lesbarer Ortsname in den Dateien, sodass UGREEN Fotos
den Ort anzeigt und für Orts-Alben nutzen kann.

## Acceptance Criteria

- [ ] **AC1** — Der Dialog „Ort setzen" zeigt eine Karte (OSM mit Attribution); ein Klick setzt
  einen Marker, ein Suchtreffer (Story 014) zentriert die Karte und setzt den Marker.
- [ ] **AC2** — „Übernehmen" schreibt für alle ausgewählten Fotos Koordinaten, Ortsname und
  Ortsteile gemäß Schreibstrategie als einen Job (Fotos und Videos).
- [ ] **AC3** — Haben ausgewählte Fotos bereits einen Ort, fragt der Dialog vorher: „alle
  überschreiben" oder „nur Fotos ohne Ort"; die Wahl wirkt wie angegeben.
- [ ] **AC4** — „Ort entfernen" löscht Koordinaten und Ortsfelder aus den ausgewählten Dateien.
- [ ] **AC5** — Die Detailansicht zeigt Ortsname und eine kleine Karte mit Marker.
- [ ] **AC6** — Die Kartenbibliothek wird erst beim Öffnen des Dialogs geladen; das
  Haupt-Bundle der Startseite enthält sie nicht.

## Open Questions

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
