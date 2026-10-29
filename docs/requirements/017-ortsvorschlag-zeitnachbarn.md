---
id: 017
title: Ortsvorschlag aus zeitlich benachbarten Fotos
status: draft # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E4 Standorte
estimate: 5
sprint: S04
---

## Requirement

Viele Fotos ohne Ort sind am selben Tag entstanden wie Fotos mit Ort (Handy-GPS vs. WhatsApp
oder Kamera ohne GPS). PhotoCurator soll für solche Fotos den Ort der zeitlich nächsten Fotos
vorschlagen, und ich übernehme ihn mit einem Klick — einzeln oder für viele auf einmal.

## Acceptance Criteria

- [ ] **AC1** — Für ein Foto ohne Ort wird der Ort des zeitlich nächsten Fotos mit Ort
  vorgeschlagen, sofern dieses am selben Kalendertag und innerhalb des Zeitfensters
  (Platzhalter: ±3 h) liegt.
- [ ] **AC2** — Detailansicht und Arbeitsliste zeigen den Vorschlag mit Herkunft (z. B.
  „Vermutlich: Ritzenhofer, Saalfelden — vom Foto um 14:32"); ein Klick übernimmt ihn als Job.
- [ ] **AC3** — „Vorschläge übernehmen" für eine Auswahl setzt in einem Job für jedes Foto mit
  Vorschlag dessen jeweiligen Ort; Fotos ohne Vorschlag bleiben unverändert und werden gezählt.
- [ ] **AC4** — Fotos, deren Aufnahmedatum nur aus der Änderungszeit stammt, bekommen keinen
  Vorschlag; bei Datum aus dem Dateinamen (nur tagesgenau) nur dann, wenn alle Fotos mit Ort an
  diesem Tag denselben Ort tragen.
- [ ] **AC5** — „Vorschlag ablehnen" blendet den Vorschlag für dieses Foto dauerhaft aus.

## Open Questions

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
