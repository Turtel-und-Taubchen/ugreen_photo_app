---
id: 012
title: Tags im Batch zuweisen und entfernen
status: draft # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E3 Tags & Kategorien
estimate: 5
sprint: S03
---

## Requirement

Kernfunktion: ausgewählte Fotos in einem Schritt mit einem oder mehreren Tags versehen oder
Tags von ihnen entfernen — dauerhaft in den Dateien, sodass UGREEN Fotos sie für Suche und
bedingte Alben nutzen kann.

## Acceptance Criteria

- [ ] **AC1** — Für eine Auswahl fügt „Tags zuweisen" einen oder mehrere Tags hinzu; nach
  Abschluss des Jobs zeigen alle ausgewählten Fotos die Tags in der Oberfläche, und ein
  Read-back der Dateien bestätigt sie.
- [ ] **AC2** — „Tags entfernen" listet die Tags der Auswahl mit der Anzahl betroffener Fotos;
  gewählte Tags werden aus allen ausgewählten Dateien und aus allen Feldern der
  Schreibstrategie entfernt.
- [ ] **AC3** — Zuweisen von `Anlass/Geburtstag` schreibt Pfad und Begriffe gemäß
  Schreibstrategie; beim Entfernen bleibt ein Elternbegriff (`Anlass`) erhalten, solange ein
  anderer Pfad der Datei ihn noch braucht (z. B. `Anlass/Hochzeit`).
- [ ] **AC4** — Fotos, die einen zuzuweisenden Tag schon tragen bzw. einen zu entfernenden nicht
  tragen, werden nicht neu geschrieben (Änderungszeit und Inhalt unverändert).
- [ ] **AC5** — Ein neu eingegebener Tag, den es noch nicht gibt, wird beim Zuweisen angelegt.
- [ ] **AC6** — Die Aktion funktioniert für Einzel-, Bereichs-, Tages- und Monatsauswahl
  (inkl. noch nicht geladener Fotos) sowie für Fotos und Videos.

## Open Questions

**Refinement S03 (2026-10-30) — BLOCKIERT: Spike-Ergebnis fehlt, 009 ist blockiert.**
`docs/concepts/photocurator-metadaten-strategie.md` enthält nur das Gerüst aus Story 001
(Beobachtungsspalten leer, „Entscheidung nach Spike" viermal _offen_). Das Profil
(`.claude/ai-scrum.md`, Notes: „Stories that write metadata (009 ff.) must not be refined before
its result document exists"), die Eingangsbedingung in `sprints/S03/sprint.md` und das S02-Review
(„Eingangsbedingung für 009/012 in S03") verbieten das Refinement ohne dieses Ergebnis. Dazu
baut 012 vollständig auf Writer, Job-Queue und Feldmapping aus 009 auf, das aus demselben Grund
als `draft` mit Frage an den Nutzer zurückgeht — ohne 009 ist 012 nicht baubar.

**Frage an den Nutzer** (dieselbe Weiche wie in 009; die Antwort dort gilt hier mit):

- **(A) Warten auf den Spike** — Spike auf dem NAS durchführen, Strategie-Dokument füllen, dann
  009 und 012 neu refinen. 012 ruht in S03. *Entspricht den Projektregeln.*
- **(B) Auf die vorläufige Strategie bauen** — du hebst die Vorbedingung für 009 ff. ausdrücklich
  auf. Dann schreibt 012 über die eine Mapping-Tabelle aus 009 (vorläufig Tech Spec §3.2:
  Bilder `dc:Subject` Blätter + Eltern, `lr:HierarchicalSubject` Pfade, JPEG zusätzlich
  `IPTC:Keywords`/`XPKeywords`; Videos nur XMP eingebettet), AC3 „gemäß Schreibstrategie" heißt
  „gemäß dieser Tabelle"; die Entscheidungen unten gelten unverändert, und 012 kann direkt nach
  dem Refinement von 009 fertig refined werden. Risiko wie in 009: ggf. zweiter Schreiblauf,
  wenn der Spike andere Felder verlangt.
- **(C) 012 (mit 009/010) aus S03 nehmen** — in den nächsten Sprint nach dem Spike.

## Decisions (Sprint)

Vorab getroffen (unabhängig von der Feldauswahl des Spikes; gelten bei (A) und (B)):

- **Feldagnostisch:** 012 nennt keine ExifTool-Felder, sondern nutzt ausschließlich die
  Mapping-Tabelle aus 009 (`internal/metadata`) — so fließt das Spike-Ergebnis weiter an genau
  einer Stelle ein (009 AC1).
- **Zwei Job-Aktionen** `tags.add` / `tags.remove` über `POST /api/jobs {selection, action, payload: {tags}}` (Tech Spec §6), Auswahl als `library.Selection`, beim Job-Start per `library.ResolveSelection` zur festen Dateiliste aufgelöst — damit gelten Tages-/Monatsgruppen und Bereiche inkl. nicht geladener Fotos (AC6) ohne eigenen Code.
- **Pfad → Feldwerte als reine Funktion** in `internal/metadata` (Gegenstück zu `metadata.Parse`): Pfad `Anlass/Geburtstag` ergibt Hierarchie-Wert `Anlass|Geburtstag` und Begriffe `Anlass`, `Geburtstag` — eine Stelle, die Lese- und Schreibrichtung konsistent hält.
- **Entfernen berechnet die Soll-Begriffe aus den verbleibenden Pfaden der Datei:** ein Begriff (Blatt oder Eltern) wird nur entfernt, wenn kein verbleibender Hierarchie-Pfad ihn als Segment enthält (AC3) — sonst entstünde nach dem Read-back ein „Geister-Tag" bzw. ginge ein gebrauchter Elternbegriff verloren.
- **Entfernt wird der exakte Pfad, kein Teilbaum:** die Entfernen-Liste zeigt die tatsächlichen Index-Tags der Auswahl, also wählt der Nutzer genau, was verschwindet.
- **Entfernen wirkt auf alle Keyword-Felder, die `metadata.Parse` als Tag-Quelle liest** (Obermenge der Schreibfelder, z. B. auch `Keys:Keywords` bei Videos) — sonst bliebe ein entfernter Tag nach dem Read-back sichtbar und AC2 wäre für den Nutzer nicht erfüllt.
- **„Trägt den Tag schon" heißt: alle Zielfelder des Mappings enthalten die Sollwerte** — die Änderungsmenge je Datei wird aus den beim Job frisch gelesenen Werten berechnet (nicht aus dem Index); ist sie leer, wird die Datei weder gesichert noch an ExifTool gegeben und als `skipped` („bereits aktuell") gemeldet (AC4). Eine Datei, die den Tag nur in einem Teil der Felder trägt, wird ergänzt — genau das ist der Fall, in dem UGREEN ihn sonst evtl. nicht sieht.
- **Neue Tags beim Zuweisen:** jeder Tag läuft beim Job-Anlegen durch `tags.ParsePath` (ungültig → 400 mit Grund, kein Job) und `tags.Create` (idempotent) — er steht damit im Baum, auch wenn einzelne Dateien fehlschlagen (AC5, Validierung aus 011).
- **Entfernen-Liste über neuen Endpunkt** `POST /api/selection/tags {selection}` → `[{path, count}]` (Anzahl betroffener Fotos aus dem Index, `photo_tags` ⨝ aufgelöste Auswahl) — analog zum bestehenden `POST /api/selection/count`.
- **Oberfläche:** zwei Knöpfe „Tags zuweisen" / „Tags entfernen" in der bestehenden `SelectionBar.svelte`, je ein Dialog; der Zuweisen-Dialog bekommt ein schlichtes Eingabefeld mit Liste mehrerer Tags — Chips und Autovervollständigung ergänzt 013 in genau diesem Dialog.
- **Nachweis „Tags in der Oberfläche" (AC1)** über die Detailansicht (`data-testid="detail-tags"`), die einzige Stelle, die Tags je Foto zeigt; Read-back im E2E per `exiftool` im App-Container (`compose exec -T photocurator exiftool …`), weil ExifTool auf dem Entwicklungsrechner fehlt.
- **E2E schreibt nur eindeutige Test-Tags** (z. B. `E2E/Batch`) und entfernt sie im selben Flow wieder — die Fixture-Bibliothek wird von allen Specs geteilt, und `tags.spec`/`photo-detail.spec` zählen bzw. zeigen die vorhandenen Tags.

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
