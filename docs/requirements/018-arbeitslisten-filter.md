---
id: 018
title: Arbeitslisten-Filter („Inbox zum Abarbeiten")
status: draft # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E5 Arbeitsliste
estimate: 5
sprint: S05
---

## Requirement

Ich will gezielt sehen, was noch zu tun ist: Fotos ohne Tags, ohne Ort oder ohne beides —
eingegrenzt auf einen Zeitraum, einen Ordner oder einen Dateityp. Was ich erledigt habe,
verschwindet aus der Liste.

## Acceptance Criteria

- [ ] **AC1** — Es gibt die Filter „ohne Tags", „ohne Ort" und „ohne beides"; jeder liefert genau
  die passenden Fotos der Fixture-Bibliothek.
- [ ] **AC2** — Die Filter sind kombinierbar mit Zeitraum (von/bis), Ordner (JJJJ oder JJJJ/MM)
  und Dateityp (Foto/Video bzw. einzelnes Format).
- [ ] **AC3** — Als „erledigt" oder „nicht relevant" markierte Fotos sind in Arbeitslisten
  standardmäßig ausgeblendet und per Schalter einblendbar.
- [ ] **AC4** — Der Filter steht in der URL; Neuladen, Teilen des Links und die Zurück-Taste
  stellen ihn wieder her. Die Trefferzahl wird angezeigt.
- [ ] **AC5** — Nach einem Job verschwinden Fotos, die den Filter nicht mehr erfüllen, aus der
  Liste, ohne dass die Seite neu geladen werden muss.
- [ ] **AC6** — „Alle Treffer auswählen" wählt alle Fotos des Filters aus, auch nicht geladene.
- [ ] **AC7** — Bei 10.000 Einträgen im Index antwortet die Filterabfrage in unter 300 ms
  (Platzhalter; Messung im Test).

## Open Questions

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
