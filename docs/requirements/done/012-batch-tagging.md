---
id: 012
title: Tags im Batch zuweisen und entfernen
status: done # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E3 Tags & Kategorien
estimate: 5
sprint: S03
---

## Requirement

Kernfunktion: ausgewählte Fotos in einem Schritt mit einem oder mehreren Tags versehen oder
Tags von ihnen entfernen — dauerhaft in den Dateien (Wiederauffindbarkeit, Interoperabilität,
PhotoCurator-eigener Index; laut Spike nutzt UGREEN Fotos Tags aus Dateien nicht).

## Acceptance Criteria

- [x] **AC1** — Für eine Auswahl fügt „Tags zuweisen" einen oder mehrere Tags hinzu; nach
  Abschluss des Jobs zeigen alle ausgewählten Fotos die Tags in der Oberfläche, und ein
  Read-back der Dateien bestätigt sie.
- [x] **AC2** — „Tags entfernen" listet die Tags der Auswahl mit der Anzahl betroffener Fotos;
  gewählte Tags werden aus allen ausgewählten Dateien und aus allen Feldern der
  Mapping-Tabelle aus 009 (sowie allen Keyword-Feldern, die PhotoCurator als Tag-Quelle liest)
  entfernt.
- [x] **AC3** — Zuweisen von `Anlass/Geburtstag` schreibt Pfad und Begriffe gemäß der
  Mapping-Tabelle aus 009; beim Entfernen bleibt ein Elternbegriff (`Anlass`) erhalten, solange ein
  anderer Pfad der Datei ihn noch braucht (z. B. `Anlass/Hochzeit`).
- [x] **AC4** — Fotos, die einen zuzuweisenden Tag schon tragen bzw. einen zu entfernenden nicht
  tragen, werden nicht neu geschrieben (Änderungszeit und Inhalt unverändert).
- [x] **AC5** — Ein neu eingegebener Tag, den es noch nicht gibt, wird beim Zuweisen angelegt.
- [x] **AC6** — Die Aktion funktioniert für Einzel-, Bereichs-, Tages- und Monatsauswahl
  (inkl. noch nicht geladener Fotos) sowie für Fotos und Videos.

## Open Questions

~~**Refinement S03 (2026-10-30) — BLOCKIERT: Spike-Ergebnis fehlt, 009 ist blockiert.**
(`photocurator-metadaten-strategie.md` nur Gerüst; Profil, Sprint-Eingangsbedingung und
S02-Review verbieten das Refinement ohne Ergebnis; 012 baut vollständig auf 009 auf.)~~
answered → Decisions (Sprint)

~~**Frage an den Nutzer:** (A) Warten auf den Spike · (B) Auf die vorläufige Strategie bauen ·
(C) 012 (mit 009/010) aus S03 nehmen.~~ answered → Decisions (Sprint)

## Decisions (Sprint)

- **(User)** Spike abwarten: Ergebnis liegt vor (`photocurator-metadaten-strategie.md`, Commit `0c8c0f1`).
- **Tags werden trotz Spike-Ergebnis geschrieben:** UGREEN Fotos liest nur GPS, Tags bleiben aber laut Strategie-Dokument („Entscheidung nach Spike") Teil der Datei — Nutzen liegt in PhotoCurator selbst, daher nur die Begründung im Requirement korrigiert, keine Scope-Änderung.
- **Schreibfelder = Mapping-Tabelle aus 009** (laut Spike mindestens `dc:Subject` + `lr:HierarchicalSubject`, gezielt `+=`/`-=`, nie ganze Listen überschreiben); 012 legt keine Felder selbst fest, weil 009 parallel refined wird und dort die Feldauswahl (inkl. ggf. Reduktion) entscheidet.
- **Schnittstellen zu 009 sind vorläufig benannt** (`POST /api/jobs`, Aktions-Registrierung im Writer, `internal/api/jobs.go`, Job-Fortschrittsanzeige); `/build` prüft Pfade und Namen gegen den finalen 009-Plan, weil 009 beim Refinement von 012 noch nicht fertig war.
- **E2E wählt einzeln `tagged.jpg` und `tagged.heic`**, keine Tages-/Monatsgruppe — die Bulk-Bibliothek belegt 2023–2024 (auch `2024/07`), Gruppen würden Hunderte Dateien schreiben, und `defekt.jpg` würde AC1 („alle ausgewählten") verfälschen; Format- und Auswahlbreite (AC6) beweist der Go-Integrationstest, die Auswahl-Serialisierung `selection.spec`.
- **`EXIF:XPKeywords` ist ein einzelner `;`-String, keine Liste:** Entfernen schreibt dort den Rest-String neu (nur dieses Feld, fremde Einträge bleiben), weil `-=` auf einen Teilstring nicht wirkt und sonst ein entfernter Tag nach dem Read-back wieder erscheint.
- **Keyword-Lesefelder werden eine exportierte Liste** (`metadata.KeywordTags`, heute inline in `metadata.Parse`), damit Entfernen und Lesen dieselben Felder kennen — eine Stelle statt zwei Kopien.

Vorab getroffen (unabhängig von der Feldauswahl des Spikes):

- **Feldagnostisch:** 012 nennt keine ExifTool-Felder, sondern nutzt ausschließlich die
  Mapping-Tabelle aus 009 (`internal/metadata`) — so fließt das Spike-Ergebnis weiter an genau
  einer Stelle ein (009 AC1).
- **Zwei Job-Aktionen** `tags.add` / `tags.remove` über `POST /api/jobs {selection, action, payload: {tags}}` (Tech Spec §6), Auswahl als `library.Selection`, beim Job-Start per `library.ResolveSelection` zur festen Dateiliste aufgelöst — damit gelten Tages-/Monatsgruppen und Bereiche inkl. nicht geladener Fotos (AC6) ohne eigenen Code.
- **Pfad → Feldwerte als reine Funktion** in `internal/metadata` (Gegenstück zu `metadata.Parse`): Pfad `Anlass/Geburtstag` ergibt Hierarchie-Wert `Anlass|Geburtstag` und Begriffe `Anlass`, `Geburtstag` — eine Stelle, die Lese- und Schreibrichtung konsistent hält.
- **Entfernen berechnet die Soll-Begriffe aus den verbleibenden Pfaden der Datei:** ein Begriff (Blatt oder Eltern) wird nur entfernt, wenn kein verbleibender Hierarchie-Pfad ihn als Segment enthält (AC3) — sonst entstünde nach dem Read-back ein „Geister-Tag" bzw. ginge ein gebrauchter Elternbegriff verloren.
- **Entfernt wird der exakte Pfad, kein Teilbaum:** die Entfernen-Liste zeigt die tatsächlichen Index-Tags der Auswahl, also wählt der Nutzer genau, was verschwindet.
- **Entfernen wirkt auf alle Keyword-Felder, die `metadata.Parse` als Tag-Quelle liest** (Obermenge der Schreibfelder, z. B. auch `Keys:Keywords` bei Videos) — sonst bliebe ein entfernter Tag nach dem Read-back sichtbar und AC2 wäre für den Nutzer nicht erfüllt.
- **„Trägt den Tag schon" heißt: alle Zielfelder des Mappings enthalten die Sollwerte** — die Änderungsmenge je Datei wird aus den beim Job frisch gelesenen Werten berechnet (nicht aus dem Index); ist sie leer, wird die Datei weder gesichert noch an ExifTool gegeben und als `skipped` („bereits aktuell") gemeldet (AC4). Eine Datei, die den Tag nur in einem Teil der Felder trägt, wird ergänzt — sonst wären die Felder untereinander inkonsistent und andere Programme sähen den Tag nicht.
- **Neue Tags beim Zuweisen:** jeder Tag läuft beim Job-Anlegen durch `tags.ParsePath` (ungültig → 400 mit Grund, kein Job) und `tags.Create` (idempotent) — er steht damit im Baum, auch wenn einzelne Dateien fehlschlagen (AC5, Validierung aus 011).
- **Entfernen-Liste über neuen Endpunkt** `POST /api/selection/tags {selection}` → `[{path, count}]` (Anzahl betroffener Fotos aus dem Index, `photo_tags` ⨝ aufgelöste Auswahl) — analog zum bestehenden `POST /api/selection/count`.
- **Oberfläche:** zwei Knöpfe „Tags zuweisen" / „Tags entfernen" in der bestehenden `SelectionBar.svelte`, je ein Dialog; der Zuweisen-Dialog bekommt ein schlichtes Eingabefeld mit Liste mehrerer Tags — Chips und Autovervollständigung ergänzt 013 in genau diesem Dialog.
- **Nachweis „Tags in der Oberfläche" (AC1)** über die Detailansicht (`data-testid="detail-tags"`), die einzige Stelle, die Tags je Foto zeigt; Read-back im E2E per `exiftool` im App-Container (`compose exec -T photocurator exiftool …`), weil ExifTool auf dem Entwicklungsrechner fehlt.
- **E2E schreibt nur eindeutige Test-Tags** (z. B. `E2E/Batch`) und entfernt sie im selben Flow wieder — die Fixture-Bibliothek wird von allen Specs geteilt, und `tags.spec`/`photo-detail.spec` zählen bzw. zeigen die vorhandenen Tags.

## Plan

Baut auf 009 (Writer, Job-Queue, Mapping-Tabelle in `internal/metadata`, `POST /api/jobs`),
011 (`tags.ParsePath`, `tags.Create`) und 008 (`library.Selection`, `ResolveSelection`,
`SelectionBar.svelte`). Alle 009-Namen sind vorläufig — `/build` gleicht sie mit dem finalen
009-Plan ab.

1. **D1 — Kern (rein, `internal/metadata`):** Pfad → Feldwerte, und je Datei die Änderungsmenge
   aus frisch gelesenen Feldern (Zuweisen/Entfernen, Eltern-Erhalt, „schon aktuell" = leer).
2. **D2 — Job-Aktionen `tags.add`/`tags.remove` (`internal/writer`):** nutzen D1 je Datei,
   leere Änderung → `skipped` ohne Backup/ExifTool; Integrationstest an Fixture-Kopien aller
   fünf Formate und aller Auswahlarten.
3. **D3 — API:** Tag-Payload-Validierung + `tags.Create` beim Job-Anlegen in `POST /api/jobs`;
   neuer `POST /api/selection/tags` → `[{path, count}]`.
4. **D4 — Oberfläche + E2E:** zwei Knöpfe in `SelectionBar.svelte`, zwei Dialoge, ein Flow
   „zuweisen → prüfen → entfernen → prüfen".

Reihenfolge D1 → D2 → D3 → D4; D3 kann nach D1 parallel zu D2 laufen.

## Deliverables

- [x] **D1 — Tag-Änderungsmenge je Datei (rein).** Dateien: `internal/metadata/tagwrite.go`
  (neu), `internal/metadata/tagwrite_test.go` (neu), `internal/metadata/read.go` (Keyword-Liste
  exportieren). Wiederverwenden: die Mapping-Tabelle aus Story 009 (in `internal/metadata`;
  welche Felder je Format Tags tragen — **nie eigene Feldnamen fest einbauen**) und
  `values`/`lookup` aus `read.go`.
  - `read.go`: die Inline-Liste in `Parse` (`XMP-dc:Subject`, `IPTC:Keywords`,
    `EXIF:XPKeywords`, `Keys:Keywords`, `UserData:Keywords`) als `var KeywordTags` exportieren,
    `Parse` nutzt sie; Verhalten unverändert (bestehende `read_test.go` bleibt grün).
  - `TagTerms(path string) (hier string, terms []string)`: `Anlass/Geburtstag` →
    `Anlass|Geburtstag`, `[Anlass, Geburtstag]`; Pfad ist bereits kanonisch (`tags.ParsePath`).
  - `TagChange(format string, fields map[string]any, add, remove []string)` → Änderung je Feld
    (hinzuzufügende / zu entfernende Werte; für `EXIF:XPKeywords` der neue Gesamt-String) in
    der Form, die der 009-Writer annimmt; leeres Ergebnis = Datei ist schon aktuell.
    - Zuweisen: je Zielfeld der Mapping-Tabelle nur die fehlenden Werte (Hierarchie-Feld: Pfad;
      Begriffs-Felder: alle Segmente). Fehlt ein Wert nur in einem Teil der Felder, wird nur dort
      ergänzt.
    - Entfernen: exakter Pfad, kein Teilbaum. Hierarchie-Wert entfernen; jeden Begriff des
      Pfades aus **allen** `KeywordTags`-Feldern entfernen, **außer** er ist Segment eines
      verbleibenden Hierarchie-Pfades der Datei oder selbst ein verbleibender flacher Tag
      (`Anlass` bleibt, wenn `Anlass|Hochzeit` bleibt). Ein flacher Tag (ohne Hierarchie) wird aus
      allen Keyword-Feldern entfernt. `XPKeywords`: Rest-String ohne die Begriffe, fremde Einträge
      in Reihenfolge erhalten.
    - Fremde Werte werden nie angefasst; nie ganze Listen ersetzen (außer `XPKeywords` wie oben).
  - Tests in `tagwrite_test.go` (table-driven, ExifTool-JSON-Maps wie in `read_test.go`):
    „assigning a hierarchical tag adds the path and every segment", „a file that already carries
    a tag in every field yields no change", „a tag present in only some fields is completed",
    „removing a path keeps a parent another path still needs", „removing a tag clears it from
    every keyword field Parse reads", „removing from XPKeywords keeps foreign entries",
    „removing a tag the file does not carry yields no change".

- [x] **D2 — Job-Aktionen `tags.add` / `tags.remove`.** Dateien: `internal/writer/tags.go` (neu),
  `internal/writer/tags_test.go` (neu), die Aktions-Registrierung des 009-Writers (Datei laut
  009-Plan, vermutlich `internal/writer/actions.go` oder `job.go`). Spiegeln: die Beispiel-/
  Test-Aktion aus 009 und deren Integrationstest-Gerüst (Fixture-Kopie in `t.TempDir()`,
  ExifTool im Docker-Test-Stage).
  - Payload `{tags: []string}` (kanonische Pfade). Je Datei: frisch gelesene Felder (nicht der
    Index) → `metadata.TagChange` (D1) → leer: Ergebnis `skipped` mit Grund „bereits aktuell",
    **kein** Backup, **kein** ExifTool-Schreibaufruf; sonst normaler 009-Ablauf (Backup →
    Schreiben → Read-back gegen die Sollwerte von D1 → Index).
  - Auswahl wird über 009 per `library.ResolveSelection` beim Job-Start aufgelöst — kein eigener
    Auflöse-Code.
  - Tests in `tags_test.go` (Fixtures aus `testdata/library`):
    „tagging a selection writes and reads back the tags on jpeg, heic, png, mp4 and mov",
    „removing Anlass/Geburtstag keeps Anlass while Anlass/Hochzeit remains and keeps foreign
    keywords" (Basis `tagged.jpg`: fremde `Familie`/`Sommer`/`Garten` bleiben),
    „a file that already carries the tag is skipped with mtime and bytes unchanged" (auch für
    Entfernen eines nicht getragenen Tags),
    „a tag job resolves single, range, day and month selections including files never listed".

- [x] **D3 — API: Tag-Jobs annehmen und Tags der Auswahl zählen.** Dateien: `internal/api/jobs.go`
  (aus 009), `internal/api/jobs_test.go`, `internal/api/selection.go`,
  `internal/api/selection_test.go`, `internal/library/tagquery.go` (+ Test in
  `internal/library/tagquery_test.go` falls vorhanden, sonst im API-Test). Spiegeln:
  `POST /api/selection/count` in `selection.go` (Body-Limit, `Validate`, 400/500-Muster) und die
  Testhelfer in `selection_test.go`/`tags_test.go`.
  - `POST /api/jobs` mit `action` `tags.add`/`tags.remove`: `payload.tags` nicht leer, jeder Pfad
    durch `tags.ParsePath` (Fehler → 400 mit der Meldung, kein Job); bei `tags.add` danach
    `tags.Create` je Pfad (idempotent) **vor** dem Einreihen, kanonische Pfade in den Job.
  - `POST /api/selection/tags {selection}` → `200 [{path, count}]`, sortiert nach Pfad; count =
    Anzahl Fotos der aufgelösten Auswahl mit genau diesem `tag_path` (`library.SelectionTagCounts`
    in `tagquery.go`, nutzt die Auswahl-SQL aus `selection.go`, kein Laden aller IDs in Go).
  - Tests: „assigning a new tag creates it in the tag tree even before the job runs",
    „an invalid tag path is rejected with its reason and creates no job",
    „selection tags lists each tag of the selection with its photo count" (inkl. 401 ohne Session).

- [x] **D4 — Oberfläche „Tags zuweisen" / „Tags entfernen" + E2E-Flow.** Dateien:
  `web/src/lib/SelectionBar.svelte`, `web/src/lib/TagAssignDialog.svelte` (neu),
  `web/src/lib/TagRemoveDialog.svelte` (neu), `web/src/lib/api.ts` (nur falls ein Helfer fehlt),
  `web/tests/e2e/batch-tagging.spec.ts` (neu). Die Fixture-mtimes sind bereits gealtert
  (`helpers/bulk-library.ts`), die Schreib-Ruhezeit aus 009 greift also nicht. Spiegeln: Fehleranzeige/`role="alert"` wie in
  `web/src/routes/Tags.svelte`, Auswahl-Body über `model.toRequest(state)` wie
  `selectionStore.svelte.ts`, Job-Fortschritt über die 009-Anzeige (nicht neu bauen).
  - Zwei Knöpfe in der Auswahlleiste (≥ 44 px). Zuweisen-Dialog: Eingabefeld + „Hinzufügen" baut
    eine Liste mehrerer Tags, „Zuweisen" → `POST /api/jobs` (`tags.add`); 400 → Meldung im
    Dialog. 013 setzt später `TagPicker` in genau diesen Dialog — Eingabe-Liste als eigenen Block
    halten. Entfernen-Dialog: lädt `POST /api/selection/tags`, zeigt `Pfad (n Fotos)` als
    Checkboxen, „Entfernen" → `tags.remove`.
  - E2E `web/tests/e2e/batch-tagging.spec.ts` › "assigning and removing tags on a selection
    shows in the detail view and in the files" — ein Flow (Test-Tags `E2E/Batch` und neu
    `E2E/Neu`, am Ende wieder entfernt, weil `tags.spec`/`photo-detail.spec` dieselbe Bibliothek
    nutzen): `tagged.jpg` und `tagged.heic` einzeln wählen (nie Gruppen: Bulk-Dateien liegen
    auch in 2024, `defekt.jpg` schlägt fehl) → beide Tags zuweisen → Job fertig abwarten → Detailansicht (`detail-tags`) zeigt sie bei jedem
    Foto → Read-back per `compose(['exec','-T','photocurator','exiftool', …])` →
    Tag-Seite zeigt `E2E/Neu` → Entfernen-Dialog zeigt `E2E/Batch (2 Fotos)` → entfernen →
    Detail und Read-back ohne beide Tags → Aufräumen der angelegten Definition `E2E` per
    `DELETE /api/tags` (011), damit keine leere Kategorie in anderen Specs auftaucht.

## Model Hints

- D1 → deliverable-hard: die Entfernen-Regel (Eltern-Erhalt über verbleibende Pfade, flache Tags, `XPKeywords` als String, nur fehlende Felder ergänzen) ist die Stelle, an der ein subtiler Fehler fremde Keywords löscht oder „Geister-Tags" nach dem Read-back hinterlässt — Datenverlust in Nutzerdateien.
- Review: → default — jede plausible Fehlimplementierung (Liste überschreiben, Eltern löschen, Index statt Datei als Basis) wird von den benannten D1/D2-Tests mit `tagged.jpg` und den Byte-/mtime-Vergleichen sichtbar.

## Acceptance Tests

- AC1 → e2e `web/tests/e2e/batch-tagging.spec.ts` › "assigning and removing tags on a selection shows in the detail view and in the files"; unit `internal/writer/tags_test.go` › "tagging a selection writes and reads back the tags on jpeg, heic, png, mp4 and mov"
- AC2 → e2e `web/tests/e2e/batch-tagging.spec.ts` › "assigning and removing tags on a selection shows in the detail view and in the files"; unit `internal/api/selection_test.go` › "selection tags lists each tag of the selection with its photo count"; unit `internal/metadata/tagwrite_test.go` › "removing a tag clears it from every keyword field Parse reads"
- AC3 → unit `internal/writer/tags_test.go` › "removing Anlass/Geburtstag keeps Anlass while Anlass/Hochzeit remains and keeps foreign keywords"; unit `internal/metadata/tagwrite_test.go` › "assigning a hierarchical tag adds the path and every segment", "removing a path keeps a parent another path still needs"
- AC4 → unit `internal/writer/tags_test.go` › "a file that already carries the tag is skipped with mtime and bytes unchanged"
- AC5 → unit `internal/api/jobs_test.go` › "assigning a new tag creates it in the tag tree even before the job runs"; e2e `web/tests/e2e/batch-tagging.spec.ts` › "assigning and removing tags on a selection shows in the detail view and in the files" (Tag-Seite zeigt `E2E/Neu`)
- AC6 → unit `internal/writer/tags_test.go` › "a tag job resolves single, range, day and month selections including files never listed", "tagging a selection writes and reads back the tags on jpeg, heic, png, mp4 and mov"; Auswahl-Serialisierung im UI ist durch `web/tests/e2e/selection.spec.ts` › "selecting photos one by one, as a range and as a whole month survives scrolling and the detail view" bewiesen; der E2E-Flow oben nutzt die Einzelauswahl über die echte Oberfläche.

## Done

**Zusammenfassung:** Tags lassen sich für eine Auswahl über die Auswahlleiste zuweisen und entfernen (`tags.add` / `tags.remove` über `POST /api/jobs`). Die Tag-Logik (`metadata.TagArgs`, aus `Args`) rechnet je Datei aus den frisch gelesenen Feldern nur die fehlenden bzw. vorhandenen Werte aus; ein Entfernen räumt alle Keyword-Felder (auch `XPKeywords` als Rest-String) und erhält benötigte Elternbegriffe. Die Pipeline überspringt bereits aktuelle Dateien vor Backup und ExifTool („bereits aktuell"). Neu: `POST /api/selection/tags`, Dialoge „Tags zuweisen"/„Tags entfernen".

Change type: feat

Commit: `012: batch tag assign/remove — fresh-read tag args, skip-before-backup, selection tag counts, assign/remove dialogs + e2e`

**Decisions (Build, Abweichungen vom Plantext, der 009/010 nicht kannte):**
- Statt `TagChange`/`internal/writer/tags.go`: `metadata.TagTerms`/`TagArgs` in `internal/metadata/tagwrite.go`; `Args(format, fields, c)` (ohne `old Meta`) delegiert den Tag-Teil dorthin — eine Tag-Implementierung. Keine neue Aktions-Registrierung nötig: die Aktion ist ein Label, `Change.AddTags/RemoveTags` trägt sie.
- Mapping-Tabelle: Remove-Regel nur noch `HierarchicalSubject`; Begriffe werden aus jedem `KeywordTags`-Feld entfernt (kein jpeg-Sonderfall).
- Pipeline: lesen + Args vor Backup/Vorab-Kopie; leere Args → `StateSkipped` „bereits aktuell" (gilt auch für Ort/Review-Änderungen und Undo ohne Rest).
- `registerJobs` bekommt `*TagDeps` (für `tags.Create`); Jobs-Routen nur mit `lib.Tags != nil` (in `main.go` immer gesetzt).
- UI: `createJob` feuert `JOB_CREATED_EVENT`, `JobStatus` pollt sofort (sonst bis zu 30 s Verzögerung); Eingabeliste als `TagListInput.svelte` für 013.
- Hinweis: der E2E-Lauf nutzt `photocurator:dev` (`--pull never`, Frontend per go:embed) — nach Web-Änderungen erst Image neu bauen.

**Verifikation (enger Gate):** `test-story` none → volles `docker build --target test .` + `npm --prefix web run test` (51 Tests), `lint`, `check`, `build` grün; `e2e-story`: `npx playwright test tests/e2e/batch-tagging.spec.ts` grün. Review 1 (default): PASS mit Findings; behoben: e2e-Read-back nach Entfernen jetzt beide Tags in allen Keyword-Feldern, Format-Test prüft Erhalt fremder Keywords, Duplikat-Test gelöscht, Flat-Tag-Fall ergänzt, Tech-Spec/Strategie-Doc angepasst; danach Gate erneut grün. Bewusst offen: Groß-/Kleinschreibungs-Varianten von Keywords werden beim Entfernen nicht erfasst (Edge Case); `/api/selection/tags` wiederholt den Decode/Validate-Block von `/count`; Dialog-CSS in drei Dateien. Volle Regression: Sprint-Gate.
AC → Test (alle gelaufen und grün; die `tagwrite_test`-Namen sind Subtests von `TestTagArgs`): AC1 e2e batch-tagging + writer „tagging a selection…"; AC2 e2e + api „selection tags lists…" + metadata „removing a tag clears it…"; AC3 writer „removing Anlass/Geburtstag…" + metadata „assigning a hierarchical…", „removing a path keeps a parent…"; AC4 writer „a file that already carries the tag is skipped…"; AC5 api „assigning a new tag creates it…" + e2e; AC6 writer „a tag job resolves single, range, day and month…" + Fünf-Formate-Test. Manual residue: keine.

tiers: D 4 / hard 1 · review default · cycles 1 · agents 7
