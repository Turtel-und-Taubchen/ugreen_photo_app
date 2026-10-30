---
id: 021
title: MVP-Härtung — Last, Ressourcen, Betriebshandbuch
status: draft # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E6 Betrieb
estimate: 5
sprint: S04
---

## Requirement

Bevor wir PhotoCurator mit der echten Bibliothek nutzen, muss belegt sein, dass es auf dem NAS
ressourcenschonend und flüssig läuft, und es muss beschrieben sein, wie man es betreibt,
aktualisiert und im Notfall wiederherstellt.

## Acceptance Criteria

- [ ] **AC1** — Ein Lasttest mit 5.000 generierten Dateien (gemischte Formate) läuft im Container
  mit den Compose-Grenzwerten: Erstscan-Dauer, Spitzen-RAM und Thumbnail-Durchsatz werden
  gemessen und in `docs/systems/photocurator-betrieb.md` dokumentiert; der Container wird dabei
  nicht wegen Speichermangels beendet.
- [ ] **AC2** — Der RAM-Verbrauch im Leerlauf liegt unter dem Zielwert (Platzhalter: 150 MB).
- [ ] **AC3** — Mit 5.000 Einträgen antwortet die Rasterliste (eine Seite) im 95. Perzentil in
  unter 200 ms (Platzhalter).
- [ ] **AC4** — Das Betriebshandbuch beschreibt: Update, Sicherung von `/data/app.db`,
  Wiederherstellung einer Datei aus `/backup`, Index neu aufbauen, Logs ansehen, Deinstallation
  ohne Rückstände in `Sortiert`.
- [ ] **AC5** — Abnahme auf dem echten NAS: echte Bibliothek eingelesen, ein Tag und ein Ort im
  Batch gesetzt, in UGREEN Fotos in Suche und einem bedingten Album sichtbar. **manual residue:**
  echtes NAS und proprietäre UGREEN Fotos-App.

## Open Questions

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
