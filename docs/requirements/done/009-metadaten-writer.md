---
id: 009
title: Sicherer Metadaten-Writer mit Batch-Jobs
status: done # draft -> ready -> in-progress -> done
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

- [x] **AC1** — Das Feldmapping (Format × Domänenfeld → Metadatenfelder) entspricht der
  Strategie aus dem Spike-Dokument und liegt an genau einer Stelle im Code; für jedes Format
  (JPEG, HEIC, PNG, MP4, MOV) belegt ein Test an einer Fixture-Kopie, dass Tags, Ort und
  Prüfstatus in genau diese Felder geschrieben und wieder gelesen werden.
- [x] **AC2** — Nach dem Schreiben sind Inode, Besitzer und Rechte der Datei unverändert, und im
  Fotoordner liegen keine zusätzlichen Dateien (z. B. `_original`).
- [x] **AC3** — Die Bilddaten (ExifTool `ImageDataHash`) sind vor und nach dem Schreiben
  identisch; weicht der Read-back ab, wird die Datei aus dem Backup wiederhergestellt und als
  fehlgeschlagen gemeldet.
- [x] **AC4** — Das Änderungsdatum der Datei wird gemäß Konfiguration `PC_PRESERVE_MTIME`
  behandelt; der Standardwert entspricht dem Spike-Ergebnis (Entscheidung: Erkennung durch
  UGREEN hat Vorrang).
- [x] **AC5** — In einem Batch mit einer absichtlich defekten bzw. schreibgeschützten Datei
  werden alle anderen Dateien geschrieben; das Ergebnis zeigt „n erfolgreich, m fehlgeschlagen"
  und je fehlgeschlagener Datei Pfad und Fehlermeldung.
- [x] **AC6** — Es läuft höchstens ein Schreibjob gleichzeitig; weitere Jobs warten in einer
  Warteschlange. Fortschritt (x von y) ist in der Oberfläche sichtbar, auch nach Neuladen der
  Seite.
- [x] **AC7** — Dateien jünger als die Ruhezeit sowie Pfade außerhalb von `/photos` werden nicht
  geschrieben, sondern als übersprungen gemeldet.
- [x] **AC8** — Bereits vorhandene fremde Werte in Listenfeldern (z. B. Keywords anderer
  Programme) bleiben erhalten.
- [x] **AC9** — Nach Abschluss eines Jobs ist der Index für die betroffenen Dateien aktuell, und
  die Oberfläche zeigt den im Spike ermittelten Hinweis, wann UGREEN Fotos die Änderung zeigt.
- [x] **AC10** — Wird die App während eines Jobs beendet, ist der Job danach als abgebrochen
  markiert, und keine Datei ist halb geschrieben (jede Datei hat entweder alte oder neue Werte).

## Open Questions

- ~~Welche Felder je Format? → aus `photocurator-metadaten-strategie.md` übernehmen.~~ answered → Decisions (Sprint)
- ~~Standardwert `PC_PRESERVE_MTIME` → aus dem Erkennungstest des Spikes.~~ answered → Decisions (Sprint)
- ~~Spike-Ergebnis fehlt — wie geht es mit 009 weiter: (A) warten auf den Spike, (B) auf die
  vorläufige Strategie bauen, (C) 009 aus S03 nehmen?~~ answered → Decisions (Sprint)

## Decisions (Sprint)

- **(User)** Spike-Ergebnis fehlt (2026-10-30): **Option A — warten auf den Spike.** S03 pausiert; nach Ausfüllen von `photocurator-metadaten-strategie.md` werden 009/012/015 neu verfeinert.
- **(User)** Spike abwarten: Ergebnis liegt vor, Strategie-Dokument gilt
- **Ort je Format:** Bilder (JPEG, HEIC, PNG) `EXIF:GPSLatitude/-Ref`, `EXIF:GPSLongitude/-Ref`;
  Videos (MP4, MOV) `Keys:GPSCoordinates` **und** `UserData:GPSCoordinates` — das sind laut
  Spike die einzigen Felder, die UGREEN Fotos liest; `XMP-exif`-GPS in Bildern wird nicht gelesen
  und daher nicht geschrieben.
- **Ortstexte** (`XMP-photoshop:City/State/Country`, `XMP-iptcCore:Location`, Label als
  `XMP-iptcExt:LocationShownLocationName`) werden in allen Formaten eingebettet geschrieben, IPTC-IIM
  nicht — ohne Wirkung in UGREEN, aber nötig, damit der PhotoCurator-Index (`Parse`) Ortsteile und
  Label aus der Datei zurückliest; IPTC-IIM brächte nur Zeichensatz-/64-Byte-Risiko ohne Nutzen.
- **Tags** nur in `XMP-dc:Subject` (Blätter + Elternbegriffe) und `XMP-lr:HierarchicalSubject`
  (`Anlass|Geburtstag`), alle Formate eingebettet — der Spike zeigt keine Wirkung in UGREEN, und
  `EXIF:XPKeywords`/QuickTime-Keywords sind Einzelstrings, deren Schreiben die ganze Liste
  überschreiben würde (Strategie: nie ganze Listen überschreiben).
- **Tag entfernen** wirkt per `-=` auf `XMP-dc:Subject`, `XMP-lr:HierarchicalSubject` und (JPEG)
  `IPTC:Keywords`; liest der Read-back den Tag danach noch aus einem Einzelstring-Feld
  (`XPKeywords`, QuickTime), scheitert die Datei mit Grund — besser gemeldet als still ganze fremde
  Listen überschrieben.
- **Prüfstatus** nur `XMP-photocurator:ReviewState` + `ReviewedAt`, alle Formate eingebettet —
  unverändert aus der Strategie, nie in Keyword-Feldern.
- **Keine Sidecars, keine Dateien in `/photos`** — der Spike zeigt, dass UGREEN Video-Sidecars
  ignoriert.
- **`PC_PRESERVE_MTIME` Standard `true`** (Compose-Wert `"false"` → `"true"`, Tech Spec §3.3/§3.4/§8
  angepasst) — mit neuem Änderungsdatum verschwand die Datei im Spike aus dem UGREEN-Index, mit
  unverändertem wurde der neue Ort übernommen; bei `false` warnt die App beim Start im Log.
- **Hinweis nach Jobende (AC9):** „UGREEN Fotos zeigt nur den Ort an – nach ‚Aktualisieren' in UGREEN
  Fotos. Tags und Prüfstatus sind nur in PhotoCurator sichtbar." — wörtlich das Spike-Ergebnis
  (nur GPS wirkt, Übernahme nach Aktualisieren beobachtet, automatische Erkennung nicht gemessen).
- **Ruhezeit = `PC_SCAN_QUIET`** (Standard 5 min) — eine Ruhezeit für Lesen und Schreiben, wie
  Tech Spec §3.4 („weder gelesen noch geschrieben").
- **Vorab-Kopie je Datei** unter `<PC_BACKUP_DIR>/.inflight/<JJJJ-MM-TT>/<rel_path>` vor jedem
  Schreiben, nach Abschluss der Datei gelöscht — Quelle für die Read-back-Wiederherstellung (AC3)
  und für die Wiederherstellung nach Absturz (AC10), weil `-overwrite_original_in_place` die Datei
  nicht atomar zurückkopiert. 010 ergänzt die aufbewahrte Datums-Kopie, die Vorab-Kopie bleibt.
- **Wiederherstellen schreibt in place** (Original öffnen, kürzen, Inhalt der Kopie, `fsync`, mtime
  der Kopie) — ein Rename würde Inode, Besitzer und Rechte ändern (AC2).
- **Neustart:** übrig gebliebene Vorab-Kopien werden vor dem ersten Request zurückgeschrieben und
  gelöscht; laufende **und** wartende Jobs werden `aborted`, offene Dateien `skipped` mit Grund
  „abgebrochen" — nach einem Neustart läuft nichts unbemerkt los.
- **Feste Dateiliste beim Anlegen** des Jobs (`library.ResolveSelection`) — so ist „y" auch für
  wartende Jobs sichtbar, und der Job ändert genau, was der Nutzer ausgewählt hatte.
- **API-Aktion in 009 nur `tags.add`** — die übrigen Aktionen (Entfernen, Ort, Status) bringen
  012/015/019; `tags.add` braucht 009 für seinen E2E-Flow und 010 für seine Quelljobs.
- **GPS-Vergleich im Read-back mit Toleranz aus der Mapping-Tabelle** (Bilder 1e-6°, Videos 1e-4°)
  — ISO-6709 in QuickTime speichert weniger Stellen, ein exakter Vergleich ließe jedes Video
  scheitern.

## Plan

Pipeline nach Tech Spec §3.3, Felder nach „Entscheidung nach Spike". Reihenfolge:

1. **ExifTool-Schreibsitzung (D1):** `internal/exif/writer.go` — ein `-stay_open`-Prozess,
   `Write` mit `-overwrite_original_in_place`, `-P` nach Konfiguration, `-config`; `ImageHash`.
2. **Mapping (D2):** `internal/metadata/mapping.go` — die eine Tabelle Format × Domänenfeld →
   ExifTool-Tags + Operation; `Args`, `Expect`, `Matches`. Round-Trip je Format an Fixture-Kopien.
3. **Pipeline pro Datei (D3, hard):** `internal/writer/pipeline.go` + `inflight.go` —
   Vorbedingungen, Vorab-Kopie, alte Werte + Hash, Schreiben, Read-back + Hash, ggf.
   Wiederherstellen in place, Index-Zeile aktualisieren (`library.IndexFiles`), `Recover`.
4. **Jobs (D4):** Migration `jobs`/`job_items` in `internal/store/appdb.go`;
   `internal/writer/jobs.go` — Warteschlange, genau ein Läufer, Zähler pro Datei, Neustart-Regel.
5. **Konfiguration und Verdrahtung (D5):** `PC_PRESERVE_MTIME` in `internal/config`,
   `cmd/photocurator` (Sitzung, Pipeline, Queue, `Recover` vor dem Serve), Compose, README, Tech Spec.
6. **API (D6):** `POST /api/jobs` (`tags.add`), `GET /api/jobs`, `GET /api/jobs/{id}`.
7. **Oberfläche (D7):** Job-Status im Header (Fortschritt, Warteschlange, Ergebnis, Hinweis); E2E.

Kein Systems-Dokument betroffen (`docs/systems` existiert nicht).

## Deliverables

- **D1 — ExifTool-Schreibsitzung.** Neu `internal/exif/writer.go`: `type Writer struct{ConfigPath
  string; PreserveMtime bool}` mit einem `-stay_open True -@ -`-Prozess (lazy gestartet, bei Tod
  neu gestartet, Mutex: ein Aufruf zugleich, `{ready}`-Protokoll mit `-execute`). `Write(ctx,
  path string, args []string) error` hängt immer `-overwrite_original_in_place`, bei
  `PreserveMtime` `-P`, sowie `-config` an; jede Zeile mit `\r`/`\n` in Argument oder Pfad wird
  abgewiesen (wie `internal/exif/reader.go` Zeile 37–40); ExifTool-Fehler-/Warnzeilen „Error:" →
  Rückgabefehler mit dem Text. `ImageHash(ctx, path) (string, error)` liest `ImageDataHash` mit
  `-api ImageHashType=SHA256` über dieselbe Sitzung. `Close()` beendet den Prozess. Kein Shell-
  Aufruf. Tests `internal/exif/writer_test.go` (echtes ExifTool, `testsupport.CopyFixture`):
  `TestWriterRejectsLineBreaksInArguments`, `TestImageHashIgnoresAMetadataChange`
  (`library/2024/07/sample.jpg`: Hash vor = nach Schreiben eines Tags).
- **D2 — Feldmapping als eine Tabelle.** Neu `internal/metadata/mapping.go`: Formate `jpeg heic png
  mp4 mov` (Werte wie `photos.format`); `type Change struct{AddTags, RemoveTags []string; Location
  *Location; ReviewState string}` (`Location`: `Lat, Lon, Label, Sublocation, City, State,
  Country`); **eine** Tabelle Format × Domänenfeld (`tags`, `location`, `review`) → ExifTool-Tags,
  Operation und GPS-Toleranz, Inhalt exakt nach diesen Regeln: Tags `XMP-dc:Subject` (Blätter +
  Elternbegriffe, `+=`) und `XMP-lr:HierarchicalSubject` (`A|B`, `+=`), Entfernen `-=` dort und bei
  JPEG in `IPTC:Keywords` — ein Elternbegriff wird aus `dc:Subject` nur entfernt, wenn kein
  verbleibender Tag ihn braucht; Ort Bilder `EXIF:GPSLatitude/-Ref/GPSLongitude/-Ref`, Videos
  `Keys:GPSCoordinates` + `UserData:GPSCoordinates`, alle Formate `XMP-photoshop:City/State/Country`,
  `XMP-iptcCore:Location` (Sublocation), `XMP-iptcExt:LocationShownLocationName` (Label); Prüfstatus
  `XMP-photocurator:ReviewState` + `ReviewedAt`; nie `XPKeywords`, QuickTime-Keywords, IPTC-IIM-Ort,
  `XMP-exif`-GPS, Sidecars. Funktionen: `Args(format string, old Meta, c Change) ([]string,
  error)` (unbekanntes Format → Fehler), `Expect(old Meta, c Change) Meta`, `Matches(format
  string, got, want Meta) bool` (Toleranz aus der Tabelle: Bilder 1e-6°, Videos 1e-4°). Listenfelder
  nur mit `+=`/`-=`, nie `=`. `ReadTags` in `read.go` bleibt die Leseliste. Tests
  `internal/metadata/mapping_test.go`: unit `TestListFieldsAreNeverAssignedWhole` (alle Formate,
  Args nie `-<Listenfeld>=`), `TestEveryWrittenTagIsAlsoRead` (jeder Tag der Tabelle ist über
  `ReadTags`/Composite lesbar), `TestRemovingATagKeepsAParentAnotherTagStillNeeds`; integration mit
  D1-Writer an Kopien von `library/2024/07/sample.jpg`, `sample.heic`, `2024/08/screenshot.png`,
  `clip.mp4`, `short.mov`: `TestEveryFormatWritesAndReadsBackTagsPlaceAndReviewState` (Tag
  `Anlass/Geburtstag`, Ort, Status `done` schreiben → `Parse` der gelesenen Felder `Matches`
  `Expect`; die geschriebenen GPS-Felder des Formats tragen die Koordinaten; `ImageHash` vor =
  nach), `TestForeignKeywordsSurviveAddingATag` (`tagged.jpg`: nach `+Urlaub` sind `IPTC:Keywords`
  Familie/Sommer, `XPKeywords` `Sommer;Garten` und alte `dc:Subject` unverändert).
- **D3 — Sichere Pipeline pro Datei.** Neu `internal/writer/pipeline.go`, `internal/writer/
  inflight.go`. `type Pipeline struct{Root, BackupDir string; Exif *exif.Writer; Reader exif.Reader;
  Index *library.Scanner; Quiet time.Duration; Now func() time.Time; AfterWrite func(abs string)
  error /* Test-Hook */}`. `WriteFile(ctx, rel, c metadata.Change) Result{State ok|skipped|failed,
  Reason}`: (1) `filepath.Clean`, unter `Root` (Prüfung wie `internal/config/paths.go`), `Lstat`:
  reguläre Datei, kein Symlink, sonst `skipped` „außerhalb von /photos"; mtime jünger als `Quiet` →
  `skipped` „Ruhezeit"; (2) Vorab-Kopie nach `<BackupDir>/.inflight/<JJJJ-MM-TT>/<rel>` (Temp-Datei →
  `fsync` → Rename, mtime übernehmen; Fehler → `failed`, Datei unberührt); (3) alte Werte
  (`Reader` + `metadata.Parse`) und `ImageHash`; (4) `metadata.Args` → `Exif.Write`; (5) Read-back:
  `Matches(Expect(old,c))` und Hash gleich, sonst Wiederherstellen **in place** aus der Vorab-Kopie
  (Original `O_WRONLY|O_TRUNC`, kopieren, `fsync`, mtime der Kopie) → `failed` mit Grund;
  ExifTool-Fehler ebenso; (6) Vorab-Kopie löschen; (7) Index-Zeile über neues
  `library.(*Scanner).IndexFiles(ctx, rels)` in `internal/library/scan.go` (nutzt `indexBlock`, kein
  zweiter Indexpfad). `Recover(ctx) ([]string, error)`: jede Datei unter `.inflight/` in place
  zurückschreiben, löschen, Index aktualisieren. Nie eine Datei in `Root` anlegen. Tests
  `internal/writer/pipeline_test.go` (echtes ExifTool, Fixture-Kopien unter `t.TempDir()`):
  `TestWriteKeepsInodeOwnerModeAndLeavesNoExtraFiles` (`syscall.Stat_t` vor/nach; Ordnerinhalt
  `Root` vor = nach; `.inflight` leer), `TestReadBackMismatchRestoresFileAndReportsFailure`
  (`AfterWrite` ändert die Datei → Bytes = vorher, Inode gleich, `failed`),
  `TestMtimeFollowsPreserveMtimeSetting` (true: mtime gleich; false: neu),
  `TestFilesInQuietPeriodOrOutsidePhotosAreSkipped` (frische mtime; `../x.jpg`; Symlink nach außen),
  `TestIndexShowsWrittenValuesRightAfterTheWrite`,
  `TestRecoverRestoresAHalfWrittenFileFromItsInflightCopy` (`AfterWrite` kürzt die Datei auf die
  Hälfte und bricht ab → `Recover` → Bytes = Original, Inode gleich).
- **D4 — Job-Warteschlange mit Fortschritt.** Migration (anhängen) in `internal/store/appdb.go`:
  `jobs(id, username, action, payload JSON, status queued|running|done|aborted, total, done, ok,
  skipped, failed, created_at, started_at, finished_at)`, `job_items(job_id, rel_path, state
  pending|ok|skipped|failed, reason, updated_at)` + Index `job_items(job_id)`. Neu
  `internal/writer/jobs.go`: `Queue{DB, Pipeline}`; `Create(ctx, user, action string, c
  metadata.Change, rels []string) (id, error)`; ein Läufer-Goroutine, genau ein Job `running`,
  FIFO; nach jeder Datei `job_items` und Zähler in einer Transaktion; ein Dateifehler stoppt den
  Job nie; `Start(ctx)` ruft zuerst `Pipeline.Recover`, setzt `running`/`queued` → `aborted`, offene
  Items → `skipped` „abgebrochen". `List(ctx, n)`, `Get(ctx, id)`. Tests
  `internal/writer/jobs_test.go`: `TestBrokenFileFailsAloneAndTheRestIsWritten`
  (`2024/08/defekt.jpg` + fehlende Datei + zwei gute → „2 ok, 2 failed", je Pfad und Grund),
  `TestOnlyOneWriteJobRunsAtATimeAndOthersWait` (Hook blockiert Job 1 → Job 2 `queued`, nie zwei
  `running`; Zähler `done` steigt pro Datei), `TestRestartMarksJobsAbortedAndRestoresTheFileInFlight`
  (Abbruch per Hook mitten in Datei 2, neue Queue auf derselben DB → Job `aborted`, Datei 1 neue
  Werte, Datei 2 Original-Bytes, Rest `skipped`).
- **D5 — Konfiguration und Verdrahtung.** Neu `internal/config/writer.go`: `PC_PRESERVE_MTIME`
  (Standard `true`, `true|false`, sonst Startabbruch mit Meldung) wie `internal/config/scan.go`; Test
  `internal/config/writer_test.go` › `TestPreserveMtimeDefaultsToTrueAndRejectsInvalidValues`.
  `cmd/photocurator/main.go`: `exif.Writer` (Config aus `metadata.WriteConfig`), `writer.Pipeline`
  (Root = Fotos, BackupDir, `Quiet` = `PC_SCAN_QUIET`, `Index` = der vorhandene Scanner),
  `writer.Queue.Start` **vor** dem HTTP-Serve, `Writer.Close` beim Beenden, `slog.Warn` bei
  `PC_PRESERVE_MTIME=false`. `deploy/docker-compose.yml`: `PC_PRESERVE_MTIME: "true"`;
  `deploy/README.md`: Zeile zu `PC_PRESERVE_MTIME` wie die zu `PC_SCAN_QUIET` (warum `true`);
  `docs/concepts/photocurator-tech-spec.md` §3.3 Schritt 5, §3.4, §8: Standard `true` nach Spike.
  Läuft `deploy/deploy_test.go` gegen Compose-Werte, dort mitziehen.
- **D6 — Job-API.** Neu `internal/api/jobs.go`, Routing in `internal/api/server.go` (Abhängigkeit
  über `LibraryDeps` in `internal/api/scan.go`): `POST /api/jobs` `{selection, action, payload}` —
  nur `action: "tags.add"`, `payload.tags` je über `tags.ParsePath` (`internal/tags/path.go`),
  Auswahl über `sel.Validate` + `library.ResolveSelection` (Body-Limit wie
  `internal/api/selection.go`), Nutzer per `UserFromContext` → 202 `{id}`; unbekannte Aktion,
  leere Auswahl, ungültiger Tag → 400. `GET /api/jobs` → die letzten 20 Jobs, neueste zuerst
  (`id, action, status, total, done, ok, skipped, failed, created_at, finished_at`); `GET
  /api/jobs/{id}` zusätzlich `items` mit `rel_path, state, reason` (404 unbekannt). Hinter Session
  und CSRF wie alle Routen. Tests `internal/api/jobs_test.go`:
  `TestCreateJobResolvesTheSelectionAndQueuesIt`, `TestCreateJobRejectsUnknownActionAndBadTags`,
  `TestJobDetailListsEveryFailedFileWithReason`.
- **D7 — Job-Status in der Oberfläche.** Zuerst den zweimal vorhandenen Fehlertext-Helfer
  `message(e)` (`web/src/lib/PlacePicker.svelte:41`, `web/src/routes/Tags.svelte:16`) als
  `errorMessage` nach `web/src/lib/api.ts` ziehen und dort verwenden. Neu `web/src/lib/jobs.ts`
  (Typen, `fetchJobs`, `fetchJob`, `createJob`, `progressText`, `resultText`) +
  `web/src/lib/jobs.test.ts`; neu `web/src/lib/JobStatus.svelte` (Polling wie
  `web/src/lib/ScanStatus.svelte`, solange ein Job `queued`/`running`), eingebunden in
  `web/src/lib/AppHeader.svelte`: laufend „Schreibe x von y" + „n Jobs warten"; danach „n
  erfolgreich, m fehlgeschlagen" (+ „k übersprungen", falls > 0), aufklappbar je Datei Pfad und
  Grund, darunter wörtlich der Hinweis aus Decisions (Sprint). Zustand kommt nur vom Server →
  bleibt nach Neuladen. Vitest: „a running job reads 'Schreibe x von y'", „a finished job reads
  'n erfolgreich, m fehlgeschlagen'". E2E `web/tests/e2e/write-jobs.spec.ts`, Test „a write job
  reports its result per file and survives a reload": per `POST /api/jobs` (`tags.add`, Header
  `X-Requested-With`) Job über zwei gute Fixture-Fotos + `2024/08/defekt.jpg`; Seite neu laden →
  Job-Status sichtbar; Ergebnis „2 erfolgreich, 1 fehlgeschlagen", `2024/08/defekt.jpg` mit Grund,
  Hinweistext; `GET /api/photos/{id}` eines guten Fotos trägt den Tag.

## Model Hints

- D3 → deliverable-hard — der einzige Pfad, der Originale verändert: Wiederherstellung muss in
  place (Inode) und aus der Vorab-Kopie *dieses* Schreibens erfolgen, Hash- und Read-back-Prüfung
  müssen vor dem Löschen der Kopie greifen, und `Recover` muss halb geschriebene Dateien nach
  Absturz zurückholen — ein Reihenfolgefehler hier ist stiller Datenverlust.
- D2 bleibt Default: die Datenverlust-Falle (Listenfeld mit `=`) ist durch
  `TestListFieldsAreNeverAssignedWhole` über alle Formate und den Fremd-Keyword-Test festgenagelt.
- Review: → default — die plausiblen Fehlimplementierungen (Rename statt in-place-Restore,
  Wiederherstellen ohne Hash-Prüfung, Listen-Überschreiben, Video-GPS exakt verglichen) sind je
  durch einen benannten Test sichtbar.

## Acceptance Tests

- AC1 → integration `internal/metadata/mapping_test.go` ›
  `TestEveryFormatWritesAndReadsBackTagsPlaceAndReviewState` + unit `TestEveryWrittenTagIsAlsoRead` (D2)
- AC2 → integration `internal/writer/pipeline_test.go` ›
  `TestWriteKeepsInodeOwnerModeAndLeavesNoExtraFiles` (D3)
- AC3 → integration `internal/metadata/mapping_test.go` ›
  `TestEveryFormatWritesAndReadsBackTagsPlaceAndReviewState` (Hash je Format, D2) +
  `internal/writer/pipeline_test.go` › `TestReadBackMismatchRestoresFileAndReportsFailure` (D3)
- AC4 → integration `internal/writer/pipeline_test.go` › `TestMtimeFollowsPreserveMtimeSetting` (D3)
  + unit `internal/config/writer_test.go` › `TestPreserveMtimeDefaultsToTrueAndRejectsInvalidValues` (D5)
- AC5 → integration `internal/writer/jobs_test.go` › `TestBrokenFileFailsAloneAndTheRestIsWritten`
  (D4) + e2e `web/tests/e2e/write-jobs.spec.ts` › "a write job reports its result per file and
  survives a reload" (D7)
- AC6 → integration `internal/writer/jobs_test.go` › `TestOnlyOneWriteJobRunsAtATimeAndOthersWait`
  (D4) + unit `web/src/lib/jobs.test.ts` › "a running job reads 'Schreibe x von y'" (D7) + e2e
  `web/tests/e2e/write-jobs.spec.ts` › "a write job reports its result per file and survives a
  reload" (Neuladen, D7)
- AC7 → integration `internal/writer/pipeline_test.go` ›
  `TestFilesInQuietPeriodOrOutsidePhotosAreSkipped` (D3)
- AC8 → integration `internal/metadata/mapping_test.go` › `TestForeignKeywordsSurviveAddingATag` +
  unit `TestListFieldsAreNeverAssignedWhole` (D2)
- AC9 → integration `internal/writer/pipeline_test.go` ›
  `TestIndexShowsWrittenValuesRightAfterTheWrite` (D3) + e2e `web/tests/e2e/write-jobs.spec.ts` ›
  "a write job reports its result per file and survives a reload" (Hinweis, Tag im Detail, D7);
  manual residue: dass UGREEN Fotos den Ort nach „Aktualisieren" tatsächlich zeigt — proprietäre
  App auf dem NAS, nicht automatisierbar.
- AC10 → integration `internal/writer/jobs_test.go` ›
  `TestRestartMarksJobsAbortedAndRestoresTheFileInFlight` (D4) + `internal/writer/pipeline_test.go`
  › `TestRecoverRestoresAHalfWrittenFileFromItsInflightCopy` (D3)

Zusätzlich (Entscheidungen, keine eigenen AC): `TestWriterRejectsLineBreaksInArguments`,
`TestImageHashIgnoresAMetadataChange` (D1), `TestRemovingATagKeepsAParentAnotherTagStillNeeds` (D2),
`TestCreateJobResolvesTheSelectionAndQueuesIt`, `TestCreateJobRejectsUnknownActionAndBadTags`,
`TestJobDetailListsEveryFailedFileWithReason` (D6).

## Done

Schreibschicht gebaut: ein `-stay_open`-ExifTool-Writer, die eine Mapping-Tabelle (Format x Feld), die Pipeline je Datei
(Vorab-Kopie, Read-back mit Matches + ImageDataHash, Wiederherstellen in place, `Recover`), eine FIFO-Job-Queue mit
genau einem Läufer und persistentem Fortschritt (`jobs`/`job_items`), `PC_PRESERVE_MTIME` (Standard `true`),
`POST/GET /api/jobs` (`tags.add`) und der Job-Status im Header mit Hinweis. Ein schreibgeschütztes bzw. defektes File scheitert allein.

Change type: feat

Commit message: `009: safe metadata writer — mapping table, per-file pipeline with in-place restore, job queue + API + UI`

Decisions:
- Hook-abgebrochene Datei endet im Restart-Test als `failed` (Hook-Fehlertext), Rest `skipped`, Job `aborted`; fehlende Datei = `failed` "Datei nicht gefunden".
- Pre-Copy bleibt nur, wenn das Original abweicht und nicht zurückschreibbar ist; ist es byte-gleich zur Kopie (z. B. 0444), wird die Kopie verworfen (sonst blockiert sie die Datei dauerhaft).
- `library.FormatOf` exportiert (Format für `Args`/`Matches`); `Recover` und `Queue.Start` laufen vor dem Scan-Scheduler.
- E2E-Harness: `web/tests/e2e/compose.ts` fällt ohne `process.getuid` (Windows) auf UID/GID 0 zurück — Docker Desktop zeigt Bind-Mounts als root:644.
- Tech Spec: `job_items.old/new`-Journal entfernt (gehört zur Undo-Story 010); Standard `PC_PRESERVE_MTIME=true`.
- Bei read-only Dateien meldet ExifTool kein "Error:"; der Grund lautet "Read-back weicht ab" statt ExifTool-Text (akzeptiert, Datei bleibt unverändert).

Verification: narrow gate (kein `test-story`; `docker build --target test .` + `npm --prefix web run test` voll, `e2e-story`: `cd web && npx playwright test tests/e2e/write-jobs.spec.ts`), build, lint, typecheck grün; Go-Pakete alle `ok` (Testnamen nicht ausgegeben, Existenz per grep bestätigt, kein Skip). Review (default, 1 Stufe): FAIL -> 5 Befunde gefixt (read-only Stuck-State, Hash-Zweig-Test, Start-Reihenfolge, e2e-Grund, Tech-Spec) und neu verifiziert. Offene Low-Befunde ungefixt: Cancel unmittelbar nach erfolgreichem ExifTool-Aufruf (kein Aufrufer betroffen), DB-Fehler nach `running` lässt Job bis zum Neustart `running`, `FormatOf` dupliziert einen Lookup.
AC -> Test: AC1 TestEveryFormatWritesAndReadsBackTagsPlaceAndReviewState + TestEveryWrittenTagIsAlsoRead; AC2 TestWriteKeepsInodeOwnerModeAndLeavesNoExtraFiles; AC3 Mapping-Hash je Format + TestReadBackMismatchRestoresFileAndReportsFailure + TestHashMismatchRestoresFileEvenWhenTheReadBackMatches; AC4 TestMtimeFollowsPreserveMtimeSetting + TestPreserveMtimeDefaultsToTrueAndRejectsInvalidValues; AC5 TestBrokenFileFailsAloneAndTheRestIsWritten + TestReadOnlyFileInABatchFailsAloneAndTheRestIsWritten + e2e; AC6 TestOnlyOneWriteJobRunsAtATimeAndOthersWait + jobs.test.ts + e2e; AC7 TestFilesInQuietPeriodOrOutsidePhotosAreSkipped; AC8 TestForeignKeywordsSurviveAddingATag + TestListFieldsAreNeverAssignedWhole; AC9 TestIndexShowsWrittenValuesRightAfterTheWrite + e2e; AC10 TestRestartMarksJobsAbortedAndRestoresTheFileInFlight + TestRecoverRestoresAHalfWrittenFileFromItsInflightCopy — alle bestanden.
Manual residue: AC9 — dass UGREEN Fotos den Ort nach „Aktualisieren" zeigt (proprietäre App auf dem NAS).
Narrow gate only; der volle Regressionslauf ist Sache des Sprints.

tiers: D 7 / hard 1 · review default · cycles 1 · agents 12

- Regression gate S03: smoke.spec "non-root" regressed by the root fallback in e2e compose.ts; fixed in the following commit (named volumes + seed service, container runs as uid 1000).
