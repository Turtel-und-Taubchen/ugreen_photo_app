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

**Refinement S03 (2026-10-30) — BLOCKIERT: Spike-Ergebnis fehlt.**
`docs/concepts/photocurator-metadaten-strategie.md` enthält nur das Gerüst aus Story 001: die
Beobachtungsspalten der Feldmatrix (Suche, Tag/Kategorie, bedingtes Album, Ort angezeigt) und
die Tabelle „Erkennung von Änderungen" sind leer, unter „Entscheidung nach Spike" stehen alle
vier Punkte auf _offen_ (Felder je Format, Video-Sidecars, Default `PC_PRESERVE_MTIME`, Regel
falls UGREEN selbst schreibt). Die beiden Fragen oben lassen sich daraus also **nicht**
beantworten; AC1 („entspricht der Strategie aus dem Spike-Dokument"), AC4 („Standardwert
entspricht dem Spike-Ergebnis") und AC9 („im Spike ermittelter Hinweis") haben keine
Faktenbasis. Die Story selbst, das Profil (`.claude/ai-scrum.md`, Notes) und die
Eingangsbedingung in `sprints/S03/sprint.md` verbieten das Refinement ohne dieses Ergebnis.

**Frage an den Nutzer:** Wie geht es mit 009 weiter?

- **(A) Warten auf den Spike** — du führst `probe generate`/`probe mutate` auf dem NAS durch,
  füllst `PROTOKOLL.md` aus und trägst Matrix, Erkennungstest und „Entscheidung nach Spike"
  ins Strategie-Dokument ein; danach wird 009 neu refined. Folge: 009 und die darauf
  aufbauenden Schreib-Stories (010, 012, 015) ruhen in S03. *Entspricht den eigenen Regeln
  des Projekts.*
- **(B) Auf die vorläufige Strategie bauen** — du hebst die Vorbedingung für 009 ausdrücklich
  auf. Dann gilt: Mapping = „Vorläufige Strategie" (Tech Spec §3.2: mehrfach schreiben,
  Video ohne QuickTime-Keywords, keine Sidecars) als eine Tabelle in `internal/metadata`;
  `PC_PRESERVE_MTIME` Default `false` (Tech Spec §3.3/§3.4 „Erkennung gewinnt", bereits so in
  `deploy/docker-compose.yml`); AC9-Hinweis als vorläufiger Text („UGREEN Fotos zeigt die
  Änderung nach der nächsten Indexierung"). AC1/AC4/AC9 werden auf „vorläufige Strategie"
  umformuliert, und eine Folgestory „Spike-Ergebnis ins Mapping einspielen" wird angelegt.
  Risiko: geschriebene Felder, die UGREEN nicht liest, bzw. fehlende Felder, die es liest —
  ggf. ein zweiter Schreiblauf über bereits getaggte Dateien.
- **(C) 009 aus S03 nehmen** — S03 läuft nur mit den Teilen weiter, die nicht in Dateien
  schreiben; 009 ff. wandern in den nächsten Sprint nach dem Spike.

## Decisions (Sprint)

- **(User)** Spike-Ergebnis fehlt (2026-10-30): **Option A — warten auf den Spike.** S03 pausiert; nach Ausfüllen von `photocurator-metadaten-strategie.md` werden 009/012/015 neu verfeinert.

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
