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

**Refinement S03 (2026-10-30) — BLOCKIERT: Spike-Ergebnis fehlt (gleiche Ursache wie 009).**
AC2 schreibt „Koordinaten, Ortsname und Ortsteile **gemäß Schreibstrategie**", und 015 nutzt den
Writer und die Jobs aus 009. `docs/concepts/photocurator-metadaten-strategie.md` enthält aber nur
das Gerüst: Feldmatrix ohne Beobachtungen, „Entscheidung nach Spike" komplett _offen_. Das Profil
(`.claude/ai-scrum.md`, Notes: „Stories that write metadata (009 ff.) must not be refined before
its result document exists") und die Eingangsbedingung in `sprints/S03/sprint.md` verbieten das
Refinement ohne dieses Ergebnis. Die Grundsatzfrage steht in
[009 › Open Questions](009-metadaten-writer.md) (Optionen A/B/C); für 015 folgt daraus:

**Frage an den Nutzer:** Wie geht es mit 015 weiter?

- **(A) bzw. (C) aus 009** — Warten auf den Spike bzw. 009 raus aus S03: 015 wandert mit in den
  nächsten Sprint und wird nach dem Spike-Ergebnis refined (es baut auf dem Writer aus 009 auf).
- **(B) aus 009** — vorläufige Strategie gilt: AC2 lautet dann „… gemäß **vorläufiger**
  Schreibstrategie (Tech Spec §3.2)"; die Felder je Format kommen allein aus dem Mapping von 009,
  015 legt keine eigenen fest. 015 wird refined, sobald 009 `ready` ist (Job-Schnittstelle
  `POST /api/jobs` mit Aktion „Ort setzen/entfernen" aus dessen Plan).
- **(D) 015 teilen** — der schreibfreie Teil läuft in S03: Karte im `PlacePicker` (Klick setzt
  Marker, Suchtreffer zentriert; zunächst auf der Seite `/orte`), kleine Karte in der
  Detailansicht, Lazy-Loading (AC1 ohne Dialog/„Übernehmen", AC5, AC6) als 015; Dialog „Ort
  setzen" und Schreiben (AC2–AC4) als neue Story nach dem Spike. Kombinierbar mit A oder C.

## Decisions (Sprint)

Unabhängig vom Spike-Ergebnis, gelten für jede Option oben:

- **Leaflet aus npm, nur per dynamischem `import()`:** ein Modul (z. B. `web/src/lib/map/`) kapselt
  Leaflet samt CSS und wird nur dynamisch geladen — Tech Spec §7 (Code-Splitting) und AC6; kein CDN,
  damit außer den Kacheln nichts von außen geladen wird (Tech Spec §9 „Ausgehend").
- **Detailkarte ebenfalls lazy:** die kleine Karte lädt dasselbe Modul erst, wenn ein Foto mit
  Koordinaten geöffnet wird — die Detailansicht liegt auf der Startseite, ein statischer Import
  brächte Leaflet ins Haupt-Bundle (AC6).
- **Kacheln:** `https://tile.openstreetmap.org/{z}/{x}/{y}.png` mit sichtbarer Attribution
  „© OpenStreetMap-Mitwirkende" — Tech Spec §10.
- **Kartenklick setzt nur den Punkt:** Marker + Breite/Länge im `PlacePicker`, kein automatisches
  Reverse-Geocoding; „Ort zum Punkt ermitteln" (014) bleibt der explizite Auslöser — keine
  Nominatim-Anfrage ohne ausdrückliche Nutzeraktion (R8), ein getippter Name wird nie überschrieben.
- **Ein Zustand treibt Karte und Felder:** Suchtreffer und von Hand eingegebene Koordinaten
  zentrieren die Karte und setzen den Marker, Kartenklick füllt die Felder — der `PickedPlace` aus
  014 bleibt die einzige Quelle, keine zweite Kopie des Punkts.
- **„Hat bereits einen Ort" = Koordinaten im Index (`gps_lat IS NOT NULL`):** dieselbe Definition
  wie die Arbeitsliste „ohne Ort" (Tech Spec §2.1, Story 018), damit „nur Fotos ohne Ort" und die
  Liste übereinstimmen.
- **„Übernehmen" erst mit Punkt und nicht leerem Ortsnamen:** die Anforderung verlangt Koordinaten
  **und** lesbaren Namen in den Dateien.
- **„Ort entfernen" überspringt Fotos ohne Ort:** sie werden nicht neu geschrieben (kein Backup,
  keine mtime-Änderung) — analog zu 012 AC4.
- **Detail-API liefert `lat`/`lon` in `location`:** `GET /api/photos/{id}` gibt heute nur Textfelder
  aus, der Marker in AC5 braucht die Koordinaten, die der Index (`gps_lat`/`gps_lon`) schon hat.
- **E2E ohne echte Kacheln:** Playwright `page.route` beantwortet `tile.openstreetmap.org` mit
  einem festen leeren PNG und zählt die Anfragen — Sprint-Notiz „nie den echten Dienst in Tests"
  gilt sinngemäß auch für OSM; Ortssuche weiter über den Nominatim-Mock aus 014.
- **AC6 per E2E gegen das echte Build:** Startseite lädt kein Skript mit Leaflet-Code, das Öffnen
  des Dialogs (bzw. der Detailkarte) lädt den Chunk nach — ein Vitest auf `web/dist` hinge von der
  Reihenfolge der Verify-Befehle ab, und gemessen wird so das, was der Browser wirklich lädt.

## Plan

_Nicht geschrieben — Refinement blockiert bis zur Antwort unter Open Questions._

## Deliverables

## Model Hints

## Acceptance Tests

## Done
