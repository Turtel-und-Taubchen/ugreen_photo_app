---
id: 009
title: Sicherer Metadaten-Writer mit Batch-Jobs
status: draft # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E2 Sicheres Schreiben
estimate: 8
sprint: S03
---

## Requirement

Alles, was PhotoCurator setzt (Tags, Ort, Prüfstatus), muss sicher in die Dateien geschrieben
werden — in die Felder, die UGREEN Fotos liest. Originale dürfen nicht beschädigt werden, ein
Fehler bei einer Datei darf nicht den ganzen Batch abbrechen, und es darf keine Konflikte mit
Phockup oder dem UGREEN-Indexer geben. Diese Story baut die Schreibschicht und die Job-Mechanik;
die Aktionen (Tags, Ort, Status) kommen in den Folgestories.

**Voraussetzung:** Das Ergebnis des Spikes (Story 001) steht in
`docs/concepts/photocurator-metadaten-strategie.md`. Ohne dieses Dokument wird diese Story nicht
refined. Ablauf: [Tech Spec §3.3](../concepts/photocurator-tech-spec.md).

## Acceptance Criteria

- [ ] **AC1** — Das Feldmapping (Format × Domänenfeld → Metadatenfelder) entspricht der
  Strategie aus dem Spike-Dokument und liegt an genau einer Stelle im Code; für jedes Format
  (JPEG, HEIC, PNG, MP4, MOV) belegt ein Test an einer Fixture-Kopie, dass Tags, Ort und
  Prüfstatus in genau diese Felder geschrieben und wieder gelesen werden.
- [ ] **AC2** — Nach dem Schreiben sind Inode, Besitzer und Rechte der Datei unverändert, und im
  Fotoordner liegen keine zusätzlichen Dateien (z. B. `_original`).
- [ ] **AC3** — Die Bilddaten (ExifTool `ImageDataHash`) sind vor und nach dem Schreiben
  identisch; weicht der Read-back ab, wird die Datei aus dem Backup wiederhergestellt und als
  fehlgeschlagen gemeldet.
- [ ] **AC4** — Das Änderungsdatum der Datei wird gemäß Konfiguration `PC_PRESERVE_MTIME`
  behandelt; der Standardwert entspricht dem Spike-Ergebnis (Entscheidung: Erkennung durch
  UGREEN hat Vorrang).
- [ ] **AC5** — In einem Batch mit einer absichtlich defekten bzw. schreibgeschützten Datei
  werden alle anderen Dateien geschrieben; das Ergebnis zeigt „n erfolgreich, m fehlgeschlagen"
  und je fehlgeschlagener Datei Pfad und Fehlermeldung.
- [ ] **AC6** — Es läuft höchstens ein Schreibjob gleichzeitig; weitere Jobs warten in einer
  Warteschlange. Fortschritt (x von y) ist in der Oberfläche sichtbar, auch nach Neuladen der
  Seite.
- [ ] **AC7** — Dateien jünger als die Ruhezeit sowie Pfade außerhalb von `/photos` werden nicht
  geschrieben, sondern als übersprungen gemeldet.
- [ ] **AC8** — Bereits vorhandene fremde Werte in Listenfeldern (z. B. Keywords anderer
  Programme) bleiben erhalten.
- [ ] **AC9** — Nach Abschluss eines Jobs ist der Index für die betroffenen Dateien aktuell, und
  die Oberfläche zeigt den im Spike ermittelten Hinweis, wann UGREEN Fotos die Änderung zeigt.
- [ ] **AC10** — Wird die App während eines Jobs beendet, ist der Job danach als abgebrochen
  markiert, und keine Datei ist halb geschrieben (jede Datei hat entweder alte oder neue Werte).

## Open Questions

- Welche Felder je Format? → aus `photocurator-metadaten-strategie.md` übernehmen.
- Standardwert `PC_PRESERVE_MTIME` → aus dem Erkennungstest des Spikes.

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
