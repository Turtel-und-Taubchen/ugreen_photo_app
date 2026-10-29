---
id: 001
title: Spike — welche Metadatenfelder liest UGREEN Fotos?
status: draft # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E0 Fundament & Spike
estimate: 5
sprint: S01
---

## Requirement

Bevor PhotoCurator irgendetwas in Fotos schreibt, müssen wir wissen, welche Metadatenfelder die
UGREEN Fotos-App (UGOS Pro „Photos") tatsächlich ausliest und für Suche, bedingte Alben und
Kategorisierung verwendet — je Format (JPEG, HEIC, PNG, MP4, MOV) und für Videos eingebettet vs.
XMP-Sidecar. Außerdem: wie und wann erkennt die App Änderungen an bereits indexierten Dateien?

Diese Story liefert das **Werkzeug und das Protokoll** für den Test auf dem NAS. Die
Durchführung in UGREEN Fotos ist Handarbeit (proprietäre App) und erfolgt durch den Nutzer nach
dem Sprint; das Ergebnis wird in `docs/concepts/photocurator-metadaten-strategie.md`
festgehalten und ist Voraussetzung für das Refinement von Story 009.

Kandidatenfelder siehe [Tech Spec §3.1](../concepts/photocurator-tech-spec.md).

## Acceptance Criteria

- [ ] **AC1** — `photocurator probe generate --out <dir>` erzeugt für jedes Format (JPEG, HEIC,
  PNG, MP4, MOV) und jedes für dieses Format schreibbare Kandidatenfeld genau eine Testdatei,
  deren eindeutiger Marker (z. B. `pcprobe-heic-xmpsubject`) nur in diesem einen Feld steht.
- [ ] **AC2** — Ein automatischer Read-back-Test weist für jede erzeugte Datei nach, dass ihr
  Marker im Zielfeld steht und in keinem anderen Kandidatenfeld.
- [ ] **AC3** — Für Hierarchie-Felder enthält der Marker einen Pfad (z. B.
  `pcprobe|heic|hierarchical`), um zu sehen, wie UGREEN Hierarchien darstellt.
- [ ] **AC4** — Für Ort gibt es je Format eine Datei nur mit GPS-Koordinaten (unterschiedliche,
  leicht erkennbare Orte je Feldvariante) und je eine nur mit Ortstext-Feldern.
- [ ] **AC5** — Für MP4 und MOV gibt es zusätzlich Varianten ohne eingebettete Metadaten, aber
  mit Sidecar `<name>.xmp` bzw. `<name>.<ext>.xmp`.
- [ ] **AC6** — Es gibt eine Datei mit allen Feldern gleichzeitig („combo") je Format sowie eine
  WhatsApp-artige JPEG ohne jegliche Metadaten.
- [ ] **AC7** — Alle Testdateien sind generiert (keine privaten Inhalte), zeigen ihren Marker
  zusätzlich als Text im Bild bzw. Video und tragen ein festes, ungewöhnliches Aufnahmedatum
  (Platzhalter: 2001-01-01), damit sie in UGREEN Fotos leicht auffindbar sind.
- [ ] **AC8** — Der Generator schreibt `PROTOKOLL.md` in den Ausgabeordner: eine Tabelle mit
  einer Zeile je Datei (Datei, Format, Feld, Marker) und leeren Spalten für die Beobachtungen
  (in Suche gefunden, als Tag/Kategorie sichtbar, in bedingtem Album nutzbar, Ort angezeigt,
  Zeit bis sichtbar).
- [ ] **AC9** — `photocurator probe mutate --dir <dir>` ändert in bereits indexierten
  Testdateien den Marker in `XMP-dc:Subject` — einmal mit erhaltenem, einmal mit neuem
  Änderungsdatum — und das Protokoll enthält die Anleitung für den Erkennungstest (automatisch /
  nach manuellem Rescan / Verzögerung).
- [ ] **AC10** — Das Protokoll enthält zusätzlich die Prüffragen: Welche Bedingungen bieten
  bedingte Alben überhaupt an? Schreibt UGREEN Fotos selbst Metadaten in Dateien (z. B. beim
  Taggen oder Favorisieren in der App)?
- [ ] **AC11** — `docs/concepts/photocurator-metadaten-strategie.md` existiert mit der leeren
  Feldmatrix, der vorläufigen Strategie aus der Tech Spec und einem Abschnitt „Entscheidung
  nach Spike" (auszufüllen nach der Durchführung).
- [ ] **AC12** — Durchführung auf dem NAS und Eintrag der Beobachtungen. **manual residue:**
  proprietäre UGREEN Fotos-App auf dem NAS, nicht automatisierbar.

## Open Questions

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
