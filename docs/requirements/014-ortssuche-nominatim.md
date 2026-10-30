---
id: 014
title: Ortssuche und Reverse-Geocoding über Nominatim
status: draft # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E4 Standorte
estimate: 5
sprint: S02
---

## Requirement

Ich will Orte per Suche finden — auch konkrete Lokale und Hotels wie „Ritzenhofer Saalfelden" —
und für einen Kartenpunkt einen lesbaren Namen bekommen. Die Nominatim-Nutzungsrichtlinie wird
eingehalten: max. 1 Anfrage pro Sekunde, eigener User-Agent, Ergebnisse cachen, keine
Autovervollständigung. Details: [Tech Spec §10](../concepts/photocurator-tech-spec.md).

## Acceptance Criteria

- [ ] **AC1** — Eine Suche (ausgelöst per Enter oder Knopf, nicht bei jedem Tastendruck) zeigt
  bis zu 10 Treffer mit Name, Adresse und Art; mit der hinterlegten Nominatim-Testantwort für
  „Ritzenhofer Saalfelden" erscheint der POI als erster Treffer.
- [ ] **AC2** — Alle Anfragen an Nominatim gehen über den Server; bei 5 gleichzeitigen Suchen
  (auch von zwei Nutzern) liegen zwischen zwei ausgehenden Anfragen mindestens 1 s.
- [ ] **AC3** — Jede ausgehende Anfrage trägt den User-Agent
  `PhotoCurator/<version> (<Kontakt aus Konfiguration>)`; ohne konfigurierten Kontakt ist die
  Ortssuche deaktiviert und die Oberfläche sagt warum.
- [ ] **AC4** — Eine wiederholte gleiche Suche (Groß-/Kleinschreibung und Leerzeichen egal)
  wird aus dem Cache beantwortet, ohne ausgehende Anfrage; Cache-Einträge verfallen nach der
  Frist (Platzhalter: 90 Tage).
- [ ] **AC5** — Für einen Kartenpunkt liefert Reverse-Geocoding über denselben Limiter und Cache
  einen Ortsvorschlag.
- [ ] **AC6** — Aus einem Treffer entsteht ein lesbarer, editierbarer Ortsname nach dem Muster
  `<Name>, <Ort>` („Ritzenhofer, Saalfelden") sowie die Ortsteile Sublocation, City, State,
  Country.
- [ ] **AC7** — Ist Nominatim nicht erreichbar oder antwortet mit Fehler, erscheint eine
  verständliche Meldung; ein Ort kann trotzdem per Kartenklick und manuell eingegebenem Namen
  gesetzt werden.

## Open Questions

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
