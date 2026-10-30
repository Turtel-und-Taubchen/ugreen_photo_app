---
id: 019
title: Fotos als „erledigt" oder „nicht relevant" markieren
status: draft # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E5 Arbeitsliste
estimate: 3
sprint: S04
---

## Requirement

Manche Fotos brauchen keinen Ort oder keine Tags (Screenshots, Belege, Ausschuss). Ich will sie
bewusst abhaken, damit sie nicht immer wieder in der Arbeitsliste auftauchen. Entschieden: Der
Status steht in einem eigenen, versteckten XMP-Feld in der Datei — er überlebt einen
Index-Neuaufbau und erscheint in UGREEN Fotos nicht als Tag.

## Acceptance Criteria

- [ ] **AC1** — Für eine Auswahl setzen die Aktionen „Erledigt", „Nicht relevant" und
  „Zurücksetzen" den Prüfstatus als Job; die Oberfläche zeigt den Status am Foto.
- [ ] **AC2** — Der Status steht in der Datei im PhotoCurator-XMP-Namespace (Status +
  Zeitpunkt) — für JPEG, HEIC, PNG, MP4 und MOV.
- [ ] **AC3** — Keyword-, Beschreibungs- und Ortsfelder der Datei sind nach dem Markieren
  unverändert.
- [ ] **AC4** — Nach Löschen von `index.db` und Neuaufbau tragen die markierten Fotos wieder
  ihren Status.
- [ ] **AC5** — Markieren ist wie jeder Job rückgängig zu machen (Story 010).

## Open Questions

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
