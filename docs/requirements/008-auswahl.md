---
id: 008
title: Mehrfach-, Bereichs- und Gruppenauswahl
status: draft # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E1 Index & Ansicht
estimate: 3
sprint: S02
---

## Requirement

Für jede Batch-Aktion muss ich schnell viele Fotos auswählen können: einzeln, als Bereich, einen
ganzen Tag oder Monat — auch Fotos, die noch nicht geladen sind. Am Handy ohne Tastatur.

## Acceptance Criteria

- [ ] **AC1** — Im Auswahlmodus markiert ein Klick/Tippen ein Foto bzw. hebt die Markierung auf;
  ein Zähler zeigt „n ausgewählt".
- [ ] **AC2** — Am PC wählt Shift+Klick alle Fotos zwischen dem zuletzt markierten und dem
  angeklickten aus.
- [ ] **AC3** — Am Handy startet langes Drücken den Auswahlmodus; die Aktion „Bereich bis hier"
  wählt alle Fotos zwischen dem zuletzt markierten und dem aktuellen aus.
- [ ] **AC4** — „Alle auswählen" an einer Tages- oder Monatsüberschrift wählt alle Fotos dieser
  Gruppe aus, auch die noch nicht geladenen; der Zähler zeigt die richtige Gesamtzahl.
- [ ] **AC5** — Die Auswahl bleibt beim Scrollen und beim Öffnen/Schließen der Detailansicht
  erhalten; „Auswahl aufheben" leert sie.
- [ ] **AC6** — Die Auswahl wird an den Server als Filter plus ein-/ausgeschlossene IDs
  übertragen; eine Auswahl „ganzer Monat" mit 2.000 Fotos ergibt eine Anfrage mit höchstens den
  Ausnahmen als Einzel-IDs, und der Server löst sie in genau die erwarteten Dateien auf.

## Open Questions

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
