---
id: 010
title: Backup der Originale, Änderungsjournal und Rückgängig pro Job
status: done # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E2 Sicheres Schreiben
estimate: 5
sprint: S03
---

## Requirement

Wenn ich mich vertan habe („falscher Ort für 300 Fotos"), will ich den ganzen Job mit einem
Klick rückgängig machen. Und falls doch einmal eine Datei Schaden nimmt, soll eine Kopie des
Originals existieren — außerhalb der Fotobibliothek, damit UGREEN Fotos keine Duplikate zeigt.
Entschieden: Journal + einmalige Original-Kopie mit Aufbewahrungsfrist.

## Acceptance Criteria

- [x] **AC1** — Vor dem ersten Schreiben auf eine Datei liegt eine Kopie unter
  `/backup/<JJJJ-MM-TT>/<relativer Pfad>`; existiert innerhalb der Aufbewahrungsfrist schon eine
  Kopie, wird keine zweite angelegt.
- [x] **AC2** — Schlägt das Backup fehl (z. B. kein Platz), wird die Datei nicht verändert und im
  Job als fehlgeschlagen mit Grund gemeldet.
- [x] **AC3** — Für jede geschriebene Datei speichert das Journal Job, Nutzer, Zeitpunkt sowie die
  alten und neuen Werte der betroffenen Felder.
- [x] **AC4** — „Rückgängig" an einem Job stellt für alle seine Dateien die alten Werte wieder
  her; der Rückgängig-Vorgang ist selbst ein Job mit Ergebnis pro Datei.
- [x] **AC5** — Dateien, deren Werte seit dem Job erneut geändert wurden, werden beim
  Rückgängigmachen übersprungen und als Konflikt gemeldet.
- [x] **AC6** — Backups älter als die Aufbewahrungsfrist (Platzhalter: 30 Tage) werden täglich
  gelöscht; jüngere bleiben.
- [x] **AC7** — Eine Jobliste zeigt die letzten Jobs mit Nutzer, Zeitpunkt, Aktion, Anzahl
  Dateien und Ergebnis.
- [x] **AC8** — `deploy/README.md` beschreibt, wie eine Datei manuell aus `/backup`
  wiederhergestellt wird und wie `/data/app.db` gesichert wird.

## Open Questions

- ~~**Freigabe hängt an 009 (Spike-Ergebnis fehlt).** 010 baut auf Writer, Job-Queue und
  Read-back aus 009; 009 ist im Refinement S03 blockiert, weil
  `docs/concepts/photocurator-metadaten-strategie.md` kein Spike-Ergebnis enthält. Die
  Profilregel („Stories, die Metadaten schreiben (009 ff.), nicht vor dem Ergebnisdokument
  refinen") erfasst formal auch 010 (Rückgängig schreibt Metadaten). Der Plan unten ist deshalb
  bewusst **feldunabhängig** geschrieben (Journal und Rückgängig arbeiten auf Domänenebene über
  das Mapping aus 009, keine ExifTool-Feldnamen in 010) und bleibt bei jeder Antwort gültig.
  Entscheidung folgt der Frage in [[009-metadaten-writer]]:
  (A) Warten auf den Spike · (B) Vorbedingung für 009 aufgehoben · (C) 009 aus S03.~~
  answered → Decisions (Sprint)

## Decisions (Sprint)

- **(User)** Spike abwarten: Ergebnis liegt vor (`photocurator-metadaten-strategie.md`, Commit
  0c8c0f1) — damit ist die Profil-Vorbedingung für Schreib-Stories erfüllt und 010 kann `ready`
  werden.
- Der feldunabhängige Plan bleibt nach dem Spike unverändert gültig — das Spike-Ergebnis betrifft
  nur das Mapping in `internal/metadata` (009), nicht Journal, Backup oder Rückgängig auf
  Domänenebene.
- Das Journal behält das volle `location`-Objekt (inkl. Ortstexte), obwohl UGREEN nur GPS liest —
  laut Spike werden die Ortstexte weiterhin geschrieben, also muss Rückgängig sie auch
  zurückdrehen.
- Gesichert wird nur die Mediendatei selbst, keine `.xmp`-Sidecars — der Spike legt fest, dass
  keine Sidecars geschrieben werden.
- `PC_PRESERVE_MTIME` (Standard laut Spike `true`) gehört zu 009; Rückgängig erbt die Behandlung,
  weil es durch dieselbe Pipeline schreibt — keine eigene mtime-Logik in 010.
- D7 betont `cp --preserve=timestamps` ausdrücklich — laut Spike verschwindet eine Datei mit
  neuem Änderungsdatum aus UGREEN Fotos.
- Die Konfliktprüfung bleibt nötig, obwohl UGREEN laut Spike nichts in Dateien schreibt —
  Fremdänderungen durch andere Werkzeuge sind weiterhin möglich.
- 009 wird parallel refined und hat noch keinen Plan; Pfade und Schnittstellen in 010 folgen der
  Tech Spec (§2 Module, §4 Tabellen `jobs`/`job_items`/`backups`, §6 API) und dem aktuellen
  009-Text — **`/build` gleicht die genauen Pfade (Pipeline-Datei, Job-Handler, `web/src/lib/jobs.ts`,
  Read-back-Test-Hook, E2E-Flow) gegen den finalen 009-Plan ab**, weil 010 erst nach 009 gebaut
  wird.
- Die Fehlertext-Funktion `message()` existiert schon zweimal (`web/src/routes/Tags.svelte`,
  `web/src/lib/PlacePicker.svelte`); D6 extrahiert sie nach `web/src/lib/api.ts` als
  `errorMessage`, falls 009 das nicht schon getan hat — eine dritte Kopie wäre ein Refine-Fehler.
- Journal und Rückgängig arbeiten auf Domänenebene (`tags`, `location`, `reviewState`) über das
  Mapping aus 009 — so hängt 010 nicht davon ab, welche Felder der Spike je Format festlegt.
- Betroffene Felder sind nur die Domänenfelder, die die Aktion des Jobs ändert — ein Ort-Job
  kollidiert nicht mit einer späteren Tag-Änderung derselben Datei.
- `new` im Journal sind die per Read-back gelesenen Werte, nicht die angeforderten — sonst wäre
  jede Datei mit gerundeten GPS-Werten beim Rückgängigmachen ein falscher Konflikt.
- Die Konfliktprüfung liest die aktuellen Werte aus der Datei (ExifTool), nicht aus dem Index —
  eine noch nicht eingelesene Fremdänderung muss als Konflikt zählen.
- Rückgängig umfasst nur die `ok`-Dateien des Quelljobs und läuft als normaler Job (Aktion
  `undo`, Verweis auf den Quelljob) durch dieselbe Queue und Pipeline inkl. Backup-Regel,
  Read-back und Journal — kein zweiter Schreibpfad.
- Tags werden als Mengendifferenz alt/neu gezielt hinzugefügt bzw. entfernt; leerer alter Ort
  bzw. Prüfstatus heißt „Felder löschen" — fremde Werte bleiben so erhalten (009 AC8).
- Rückgängig nur für abgeschlossene oder abgebrochene Jobs; ein zweites Rückgängig desselben
  Jobs gibt 409 — verhindert doppelt eingereihte Undo-Jobs bei Doppelklick.
- Beide Nutzer sehen alle Jobs und dürfen jeden rückgängig machen — zwei Personen in einem
  Haushalt, die Jobliste zeigt ohnehin, wer was getan hat.
- Backup-Datum ist das lokale Datum (Container-`TZ`) beim Anlegen; „Kopie existiert" heißt
  `backups`-Zeile innerhalb der Frist **und** Datei vorhanden — eine von Hand gelöschte Kopie
  wird neu angelegt.
- Backup wird als Temp-Datei geschrieben, `fsync`, dann per Rename an den Zielpfad; mtime und
  Rechte des Originals bleiben — keine halbe Kopie gilt je als Backup, und eine manuelle
  Wiederherstellung bringt exakt das Original zurück.
- Die Read-back-Wiederherstellung aus 009 nutzt den Stand **unmittelbar vor diesem
  Schreibvorgang**: das eben angelegte Backup, sonst eine temporäre Vorab-Kopie unter
  `/backup/.inflight/<JJJJ-MM-TT>/<rel_path>` — die ältere Fristkopie würde spätere Jobs still
  zurückdrehen (Tech Spec §3.3 Schritt 2 vs. 6).
- Aufbewahrung konfigurierbar über `PC_BACKUP_RETENTION_DAYS` (Standard 30), nicht in
  Compose/`.env.example` (für Agenten gesperrt, Standard genügt) — dokumentiert in der README.
- Aufräumen beim Start und alle 24 h: löscht Datumsordner `<JJJJ-MM-TT>` (auch unter
  `.inflight/`) vor heute − Frist samt `backups`-Zeilen, nie während eines Schreibjobs, andere
  Einträge in `/backup` bleiben — eine Stichtagsfunktion für Aufräumen und „schon gesichert?",
  damit beide Regeln nie auseinanderlaufen.
- Das Journal wird nicht aufgeräumt — wenige Bytes je Datei, und Rückgängig braucht keine
  Backups, nur Metadaten.
- Jobliste als eigene Seite `/jobs` (Header-Link „Jobs"), die letzten 50 Jobs, neueste zuerst,
  Ergebnis pro Datei aufklappbar, Rückgängig mit Bestätigung auf der Seite — wie `/tags` und
  `/orte` als eigene Route.
- Im E2E-Test entstehen die Quelljobs per `POST /api/jobs` wie im E2E-Flow von 009 — die
  Tagging-Oberfläche kommt erst mit 012; die geprüften Nutzeraktionen (Liste, Rückgängig) laufen
  über die Oberfläche.
- `app.db` sichern: Container stoppen, `app.db` samt `-wal`/`-shm` kopieren, starten — WAL-Modus,
  eine Kopie im laufenden Betrieb kann inkonsistent sein.

## Plan

Baut auf 009 (`internal/writer`: Job-Queue, Pipeline Tech Spec §3.3, Read-back, `jobs`/
`job_items` in `app.db`, `POST /api/jobs`, `GET /api/jobs[/{id}]`). Reihenfolge:

1. **Backup-Speicher (D1):** `internal/writer/backup.go` — `Ensure(relPath)` legt
   `/backup/<JJJJ-MM-TT>/<rel_path>` an, falls keine Kopie innerhalb der Frist; `Prune(now)`.
   Migration `backups` in `internal/store/appdb.go`, `PC_BACKUP_RETENTION_DAYS` in
   `internal/config`, Aufräum-Schleife (Start + 24 h) in `cmd/photocurator`.
2. **Backup in der Pipeline (D2, hard):** Schritt 2 vor dem ersten Schreiben; Fehler → Datei
   `failed` mit Grund, unverändert. Read-back-Restore auf „Stand vor diesem Schreiben".
3. **Journal (D3):** `job_items` erhält alte Werte (vor dem Schreiben) und neue Werte (Read-back)
   der betroffenen Domänenfelder als JSON plus `written_at`; Job und Nutzer über `jobs`.
4. **Rückgängig-Kern (D4):** `internal/writer/undo.go` — Undo-Job aus Journal, Konfliktprüfung
   gegen Datei, Rückschreiben über die normale Pipeline; Löschen von Ort/Prüfstatus im Mapping
   (`internal/metadata`), falls 009 es nicht kann.
5. **API (D5):** `POST /api/jobs/{id}/undo`, Jobliste mit Nutzer, Zeit, Aktion, Anzahl, Ergebnis.
6. **Oberfläche (D6):** Fehlertext-Helfer nach `web/src/lib/api.ts` extrahieren; Seite `/jobs`
   mit Liste, Ergebnis pro Datei, Rückgängig; E2E-Flow.
7. **Betriebshandbuch (D7):** `deploy/README.md` — Datei aus `/backup` wiederherstellen,
   `app.db` sichern, `PC_BACKUP_RETENTION_DAYS`.

Kein Systems-Dokument betroffen (`docs/systems` existiert nicht).

## Deliverables

- **D1 — Backup-Speicher mit Frist und Aufräumen.** Neu `internal/writer/backup.go`:
  `Ensure(relPath string) (path string, created bool, err error)` kopiert
  `/photos/<rel_path>` nach `<PC_BACKUP_DIR>/<JJJJ-MM-TT>/<rel_path>` (lokales Datum), wenn in
  `backups` keine Zeile für `rel_path` innerhalb der Frist steht **oder** deren Datei fehlt.
  Kopie: Temp-Datei im Zielordner → `fsync` → Rename; mtime und Rechte des Originals übernehmen;
  bei Fehler Temp-Datei entfernen und Fehler zurückgeben. Zielpfad mit `filepath.Clean` gegen
  `PC_BACKUP_DIR` prüfen (wie die `/photos`-Prüfung in `internal/config/paths.go`).
  `Prune(now)` löscht Datumsordner `<JJJJ-MM-TT>` direkt unter `PC_BACKUP_DIR` und unter
  `.inflight/`, deren Datum vor `now − Frist` liegt, samt `backups`-Zeilen; andere Einträge
  bleiben. **Eine** Stichtagsfunktion `cutoff(now)` für `Ensure` und `Prune`. Uhr injizierbar.
  Migration (anhängen, nie ändern) in `internal/store/appdb.go`: `backups(rel_path, backup_path,
  created_at)` + Index auf `rel_path`. `internal/config`: `PC_BACKUP_RETENTION_DAYS`
  (Standard 30, ungültig/≤0 → Startabbruch mit Meldung) wie `internal/config/scan.go`, mit Test.
  `cmd/photocurator`: Aufräumen beim Start und alle 24 h, nie gleichzeitig mit einem
  Schreibjob (über die Serialisierung der Job-Queue aus 009). Tests in
  `internal/writer/backup_test.go`: `TestFirstWriteLeavesDatedBackupCopy`,
  `TestSecondBackupWithinRetentionIsSkipped`, `TestPruneRemovesOnlyBackupsOlderThanRetention`.
- **D2 — Backup in der Schreib-Pipeline und Restore auf den Stand vor dem Schreiben.** In der
  Pipeline pro Datei aus 009 (`internal/writer`) vor dem ersten ExifTool-Schreiben
  `Ensure` aus D1 aufrufen (Tech Spec §3.3 Schritt 2). Fehler → Datei `failed` mit Grund
  `Backup fehlgeschlagen: <Fehler>`, kein Schreibzugriff, Job läuft weiter. Read-back-Restore
  (009 AC3) stellt den Stand **unmittelbar vor diesem Schreiben** her: hat `Ensure` die Kopie
  eben angelegt, diese; sonst vorher eine Vorab-Kopie nach
  `<PC_BACKUP_DIR>/.inflight/<JJJJ-MM-TT>/<rel_path>` (selbe Kopierfunktion wie D1), nach
  Abschluss der Datei löschen; nach Absturz übrig gebliebene räumt `Prune` auf. Nie aus einer
  älteren Fristkopie wiederherstellen. Keine Datei in `/photos` anlegen. Tests in
  `internal/writer/backup_pipeline_test.go` gegen echte ExifTool-Fixture-Kopien:
  `TestFailedBackupLeavesFileUntouchedAndFailsWithReason` (Backup-Fehler erzwingen, indem an
  `<PC_BACKUP_DIR>/<heutiges Datum>` eine normale Datei liegt — wirkt auch als root; Datei-Bytes
  vorher = nachher, Ergebnis `failed` mit Grund, übrige Dateien des Jobs `ok`),
  `TestReadBackFailureRestoresStateBeforeThisWrite` (Job 1 schreibt Tag A → Backup entsteht;
  Job 2 mit erzwungenem Read-back-Fehler über den Test-Hook aus 009 → Datei trägt danach Tag A,
  nicht den Originalzustand), dazu im ersten Backup-Test aus D1 der Pipeline-Fall: nach einem Job
  ist die Backup-Datei byte-gleich mit dem Original vor dem Job.
- **D3 — Journal mit alten und neuen Werten.** `job_items` (aus 009) um `old_values`,
  `new_values` (JSON) und `written_at` erweitern — neue Migration in
  `internal/store/appdb.go`, falls 009 die Spalten nicht schon angelegt hat. Inhalt: nur die
  Domänenfelder, die die Aktion ändert (`tags` als sortierte Liste, `location` als Objekt
  `lat, lon, label, sublocation, city, state, country, countryCode`, `reviewState`), leer =
  `null`. `old` = vor dem Schreiben gelesen (Schritt 3), `new` = Read-back (Schritt 6). Typ und
  (De-)Serialisierung an einer Stelle: `internal/writer/journal.go`. Job und Nutzer kommen aus
  `jobs`. Test `internal/writer/journal_test.go`:
  `TestJournalKeepsJobUserTimeAndOldAndNewValues` (Job auf zwei Fixture-Dateien mit
  vorhandenem fremden Tag; Journal-Zeilen tragen Job-ID, Nutzer, Zeitpunkt, alte und neue
  Werte; GPS-`new` = gelesener Wert).
- **D4 — Rückgängig-Kern.** Neu `internal/writer/undo.go`: `Undo(sourceJobID, user)` legt
  einen Job `action: "undo"` mit `undo_of = sourceJobID` an (Spalte per Migration), nur für
  abgeschlossene/abgebrochene Quelljobs, nur mit deren `ok`-Dateien; zweites Undo desselben
  Jobs → Fehler `ErrAlreadyUndone`, laufender/wartender Quelljob → `ErrJobActive`. Pro Datei in
  der Queue: aktuelle Werte der betroffenen Felder **aus der Datei** lesen, normalisieren wie
  das Journal (D3), mit `new_values` vergleichen; Abweichung → `skipped`, Grund
  `Konflikt: seit dem Job geändert`; sonst alte Werte über die normale Pipeline schreiben (Backup,
  Read-back, Journal): Tags als Mengendifferenz (`+=`/`-=`), leerer alter Ort/Prüfstatus = Felder
  löschen. Fehlt dem Mapping in `internal/metadata` das Löschen von Ort/Prüfstatus, dort
  ergänzen (aus der einen Mapping-Tabelle abgeleitet, keine zweite Feldliste). Tests in
  `internal/writer/undo_test.go` mit echtem ExifTool: `TestUndoRestoresOldValuesOfEveryFile`
  (Tag-Job und Ort-Job auf Dateien ohne Ort → nach Undo alte Tags, fremder Tag noch da, kein
  Ort; Ergebnis pro Datei `ok`), `TestUndoSkipsFileChangedSinceJobAsConflict` (eine Datei nach dem
  Job direkt per ExifTool geändert, Index nicht neu eingelesen → `skipped` mit Konflikt, andere
  `ok`), `TestChangeToUntouchedFieldIsNoConflict` (nach Ort-Job nur Tag geändert → Undo `ok`).
- **D5 — API für Jobliste und Rückgängig.** In den Job-Handlern aus 009 (`internal/api`, z. B.
  `jobs.go`; Routing in `internal/api/server.go`): `POST /api/jobs/{id}/undo` → 202
  `{id}` des Undo-Jobs, 404 unbekannt, 409 bei `ErrAlreadyUndone`/`ErrJobActive`.
  `GET /api/jobs` liefert die letzten 50 Jobs neueste zuerst mit `user`, `created_at`, `action`,
  `undo_of`, `undone_by`, `total`, `ok`, `skipped`, `failed`, `status`; `GET /api/jobs/{id}`
  zusätzlich pro Datei `rel_path`, Ergebnis, Grund. Hinter Session und CSRF wie alle Routen.
  Tests in `internal/api/jobs_test.go`: `TestUndoEndpointQueuesUndoJob`,
  `TestUndoEndpointRejectsActiveOrAlreadyUndoneJob`, `TestJobListShowsUserTimeActionCountAndResult`.
- **D6 — Seite „Jobs" mit Rückgängig.** Neu `web/src/routes/Jobs.svelte`, Route `/jobs` in
  `web/src/App.svelte` (wie `/tags`, `/orte`), Link „Jobs" in `web/src/lib/AppHeader.svelte`,
  API-Aufrufe in `web/src/lib/jobs.ts` (erweitern, falls 009 sie angelegt hat). Fehlertexte:
  die Funktion `message(e)` steht heute doppelt in `web/src/routes/Tags.svelte` und
  `web/src/lib/PlacePicker.svelte` (`ApiError` mit `body.error`, sonst „Anfrage
  fehlgeschlagen") — zuerst als `export function errorMessage(e: unknown): string` nach
  `web/src/lib/api.ts` ziehen und beide Stellen darauf umstellen (entfällt, falls 009 das schon
  getan hat), dann in `Jobs.svelte` verwenden. Liste: Nutzer, Zeitpunkt,
  Aktion (deutsche Bezeichnung, Undo-Jobs als „Rückgängig: <Aktion>"), Anzahl Dateien,
  Ergebnis „n erfolgreich · m übersprungen · k fehlgeschlagen"; aufklappbar Ergebnis pro Datei
  mit Grund; Knopf „Rückgängig" (≥ 44 px) mit Bestätigung auf der Seite, ausgeblendet bei
  laufenden oder bereits rückgängig gemachten Jobs; Liste aktualisiert sich, bis der Undo-Job
  fertig ist. E2E `web/tests/e2e/jobs.spec.ts`, Test „a job is undone from the job list": per
  `POST /api/jobs` wie im 009-E2E-Flow Tag-Job auf drei Fixture-Dateien, dann zweiter Job auf
  eine davon; Seite `/jobs` zeigt beide mit Nutzer, Zeit, Aktion, Anzahl, Ergebnis; Rückgängig am
  ersten Job → Undo-Job erscheint mit 2 erfolgreich, 1 übersprungen, die Datei mit Grund
  „Konflikt"; Detail-API bzw. Tags der zwei Dateien ohne den Tag.
- **D7 — Betriebshandbuch.** `deploy/README.md`: neue Abschnitte
  `## Datei aus dem Backup wiederherstellen` (Kopie unter
  `/volume1/docker/photocurator/backup/<JJJJ-MM-TT>/<relativer Pfad>` finden, mit
  `cp --preserve=timestamps` über die Datei in `Sortiert` kopieren — mit dem Hinweis, dass eine
  Datei mit neuem Änderungsdatum aus UGREEN Fotos verschwindet (Spike-Ergebnis) —, danach in der
  App „Neu einlesen"; Hinweis auf Frist `PC_BACKUP_RETENTION_DAYS`, Standard 30 Tage) und
  `## App-Datenbank sichern` (Container stoppen, `app.db`, `app.db-wal`, `app.db-shm` aus
  `/volume1/docker/photocurator/data` kopieren, starten; `index.db` und Thumbnails müssen nicht
  gesichert werden). `deploy/readme_test.go`: beide Überschriften in die Liste von
  `TestInstallGuideCoversEveryStep` aufnehmen.

## Model Hints

- D2 → deliverable-hard — greift in die Schreib-Pipeline aus 009 ein: der Read-back-Restore
  darf nach Einführung der „nur einmal je Frist"-Kopie nie auf eine ältere Fristkopie fallen
  (würde spätere Jobs still zurückdrehen), und ein Backup-Fehler muss vor jedem Schreibzugriff
  greifen.
- Review: → default — die Fallen (Konflikt gegen Index statt Datei, Vergleich mit Soll- statt
  Read-back-Werten, Restore aus alter Kopie) sind je durch einen benannten Test festgenagelt.

## Acceptance Tests

- AC1 → integration `internal/writer/backup_test.go` › `TestFirstWriteLeavesDatedBackupCopy`,
  `TestSecondBackupWithinRetentionIsSkipped` (D1, Pipeline-Fall D2)
- AC2 → integration `internal/writer/backup_pipeline_test.go` ›
  `TestFailedBackupLeavesFileUntouchedAndFailsWithReason` (D2)
- AC3 → integration `internal/writer/journal_test.go` ›
  `TestJournalKeepsJobUserTimeAndOldAndNewValues` (D3)
- AC4 → e2e `web/tests/e2e/jobs.spec.ts` › "a job is undone from the job list" (D6) +
  integration `internal/writer/undo_test.go` › `TestUndoRestoresOldValuesOfEveryFile` (D4) +
  `internal/api/jobs_test.go` › `TestUndoEndpointQueuesUndoJob` (D5)
- AC5 → integration `internal/writer/undo_test.go` › `TestUndoSkipsFileChangedSinceJobAsConflict`,
  `TestChangeToUntouchedFieldIsNoConflict` (D4) + Schritt im e2e-Flow
  "a job is undone from the job list" (D6)
- AC6 → unit `internal/writer/backup_test.go` › `TestPruneRemovesOnlyBackupsOlderThanRetention`
  (D1)
- AC7 → e2e `web/tests/e2e/jobs.spec.ts` › "a job is undone from the job list" (D6) + unit
  `internal/api/jobs_test.go` › `TestJobListShowsUserTimeActionCountAndResult` (D5)
- AC8 → unit `deploy/readme_test.go` › `TestInstallGuideCoversEveryStep` (erweitert, D7)

Zusätzlich (Entscheidungen, keine eigenen AC): `TestReadBackFailureRestoresStateBeforeThisWrite`
(D2), `TestUndoEndpointRejectsActiveOrAlreadyUndoneJob` (D5).

## Done

Backup vor dem ersten Schreiben (`internal/writer/backup.go`, Frist `PC_BACKUP_RETENTION_DAYS`, tägliches Aufräumen über `Queue.Exclusive`), Journal mit alten/neuen Werten (`journal.go`, per Read-back), Rückgängig als normaler Job `undo` mit Konfliktprüfung gegen die Datei (`undo.go`, Löschen von Ort/Prüfstatus im Mapping), API `POST /api/jobs/{id}/undo` + Liste (50), Seite `/jobs`, Betriebshandbuch.

Change type: feat

Commit message: `010: backup of originals with retention, change journal, per-job undo with conflict check, jobs page + ops guide`

Decisions:
- Abgleich mit dem echten 009-Code: die `.inflight`-Vorab-Kopie entsteht dort bei jeder Datei und bleibt unverändert (Crash-Recovery); der Restore nutzt immer sie, nie die Fristkopie. `Backups.Ensure` läuft davor (nach Skip-/Pending-Prüfung), ein Backup-Fehler stoppt vor jedem Schreibzugriff. Der Plan-Zweig „sonst Vorab-Kopie" entfällt damit; `errorMessage` existierte schon (009).
- `backups.backup_path` relativ zu `/backup`; Prune und „schon gesichert?" teilen `cutoff(now)`; Aufräumen über `Queue.Exclusive` (Mutex um `runNext`).
- Undo-Payload `{"undo_of": id}`, Änderung je Datei zur Laufzeit aus dem Journal; Unique-Index auf `jobs.undo_of` + `undoMu` machen Doppel-Undo atomar. Neu `metadata.Change.ClearLocation/ClearReviewState` aus der einen Mapping-Tabelle; Review-State „open" wird als Feld löschen zurückgedreht (Parse liest fehlend als open).
- Alter Ort ohne Koordinaten ist nicht wiederherstellbar: Datei `skipped` mit eigenem Grund.
- Review-Befunde ungefixt (bewusst): `Ensure` auch bei No-op-Schreibvorgang (harmlos); zwei Uhren (`Backups.Now`/`Pipeline.Now`, in Produktion identisch); Lesefenster zwischen Konfliktprüfung und `WriteFile` (serielle Queue); Test des Mutex-Zugriffs von `runNext` nur indirekt; `resultSummary`/`resultText` bleiben getrennt (unterschiedlicher Text). Gefixt: README-Test schärfer, Journal-Test, e2e-Nutzer, Tech Spec §2.2/§3.3.
- D1–D7 stehen im Story-File ohne Checkboxen, daher keine Häkchen.

Verification: narrow gate — kein `test-story`: voller `docker build --target test .` + `npm --prefix web run test` (51 Tests), `e2e-story`: `cd web && npx playwright test tests/e2e/jobs.spec.ts` (nach Fix zusammen mit write-jobs.spec.ts), build, lint, typecheck grün. Go-Testnamen werden nicht ausgegeben (Existenz per Quelltext, Pakete `ok`). Review default, 1 Stufe: PASS mit Test-Stärke-Befunden, gefixt und neu verifiziert.
AC -> Test: AC1 backup_test.go TestFirstWriteLeavesDatedBackupCopy + TestSecondBackupWithinRetentionIsSkipped (+ backup_pipeline_test.go Pipeline-Fall); AC2 TestFailedBackupLeavesFileUntouchedAndFailsWithReason; AC3 TestJournalKeepsJobUserTimeAndOldAndNewValues; AC4 e2e "a job is undone from the job list" + TestUndoRestoresOldValuesOfEveryFile + TestUndoEndpointQueuesUndoJob; AC5 TestUndoSkipsFileChangedSinceJobAsConflict + TestChangeToUntouchedFieldIsNoConflict + e2e; AC6 TestPruneRemovesOnlyBackupsOlderThanRetention; AC7 e2e + TestJobListShowsUserTimeActionCountAndResult; AC8 TestInstallGuideCoversEveryStep (erweitert); Extras TestReadBackFailureRestoresStateBeforeThisWrite, TestUndoEndpointRejectsActiveOrAlreadyUndoneJob — alle bestanden. Manual residue: keine.
Narrow gate only; der volle Regressionslauf ist Sache des Sprints.

tiers: D 7 / hard 1 · review default · cycles 1 · agents 11
