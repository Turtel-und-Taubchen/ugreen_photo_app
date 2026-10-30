---
id: 005
title: Bibliotheks-Index aus den Dateien in Sortiert
status: done # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E1 Index & Ansicht
estimate: 8
sprint: S02
---

## Requirement

PhotoCurator muss alle Fotos und Videos in `Sortiert` kennen — mit Aufnahmedatum, vorhandenen
Tags, Ort und Prüfstatus —, um sie schnell anzeigen und filtern zu können. Die Dateien sind die
Wahrheit: Der Index ist nur ein Cache, darf jederzeit gelöscht werden und wird dann vollständig
aus den Dateien neu aufgebaut. Neue Dateien von Phockup sollen von selbst auftauchen, ohne dass
halb verschobene Dateien gelesen werden. Details: [Tech Spec §2.1, §5](../concepts/photocurator-tech-spec.md).

## Acceptance Criteria

- [x] **AC1** — Ein Erstscan erfasst rekursiv alle Dateien mit den Endungen jpg, jpeg, heic,
  heif, png, mp4, mov (Groß-/Kleinschreibung egal) unter `/photos`; andere Dateien und
  versteckte bzw. System-Ordner (z. B. `@eaDir`, `.*`) werden ignoriert.
- [x] **AC2** — Für jede Datei stehen im Index: relativer Pfad, Ordner (JJJJ/MM), Typ und Format,
  Größe, Änderungszeit, Aufnahmedatum mit Quelle, Tags (aus allen Keyword-Feldern
  zusammengeführt, Hierarchie aus `HierarchicalSubject`), GPS, Ortsfelder und Prüfstatus.
- [x] **AC3** — Das Aufnahmedatum wird in dieser Reihenfolge ermittelt: EXIF
  `DateTimeOriginal` → QuickTime `CreateDate` → Datum im Dateinamen (WhatsApp-Muster
  `IMG-JJJJMMTT-WA…`, `VID-JJJJMMTT-WA…`, `JJJJMMTT_hhmmss`) → Änderungszeit; die Quelle wird
  gespeichert.
- [x] **AC4** — Ein Folgescan liest nur neue oder geänderte (Größe/Änderungszeit) Dateien mit
  ExifTool; entfernte Dateien verschwinden aus dem Index.
- [x] **AC5** — Dateien, deren Änderungszeit jünger als die Ruhezeit ist (Platzhalter: 5 min),
  werden übersprungen und beim nächsten Scan erfasst.
- [x] **AC6** — Der Scan läuft beim Start, periodisch (Platzhalter: alle 15 min) und auf Knopfdruck
  „Neu einlesen"; während des Scans zeigt die Oberfläche Status und Fortschritt.
- [x] **AC7** — Nach Löschen von `index.db` und Neustart enthält der neu aufgebaute Index für die
  Fixture-Bibliothek dieselben Tags, Orte und Prüfstatus wie vorher.
- [x] **AC8** — Eine nicht lesbare Datei bricht den Scan nicht ab; der Fehler wird protokolliert
  und die Datei in der Oberfläche als „nicht lesbar" gekennzeichnet.
- [x] **AC9** — Der Scan verändert keine Datei unter `/photos` (Prüfsummen und Änderungszeiten
  der Fixture-Bibliothek sind vorher und nachher gleich).

## Open Questions

## Decisions (Sprint)

- ExifTool liest pro Block (~200 Dateien) in **einem** Aufruf (`-@ argfile`), kein
  `-stay_open`-Dauerprozess fürs Lesen — ein Perl-Start pro 200 Dateien kostet ~1–2 ms je Datei,
  spart den Leerlauf-RAM eines Dauerprozesses und das fehleranfällige stay_open-Protokoll
  (Tech Spec §5 wird entsprechend ergänzt; der Schreibprozess in 009 ist davon unberührt).
- Kein `-struct` beim Lesen; Ortsfelder kommen flach (`LocationShownLocationName` usw.), weil
  die flachen Namen 1:1 auf die Mapping-Tabelle passen und kein Struktur-Parsing brauchen.
- `-fast2` nur, wenn der Integrationstest mit dem Video-Fixture (moov-Atom hinter mdat, wie
  ffmpeg es standardmäßig schreibt) Datum, GPS und Keywords weiterhin liest; sonst ohne — Korrektheit
  vor einer Mikrooptimierung.
- Eigener XMP-Namespace für den Prüfstatus: Präfix `photocurator`, URI
  `http://ns.photocurator.invalid/1.0/`, Tags `ReviewState` (`open`/`done`/`irrelevant`) und
  `ReviewedAt`; als eingebettete ExifTool-Konfiguration in `internal/metadata` — die URI steht
  dauerhaft in Dateien, die reservierte TLD `.invalid` beansprucht keine fremde Domain, und 019
  schreibt dann mit genau dieser Definition.
- Fehlender oder unbekannter Prüfstatus-Wert → `open`, weil Arbeitslisten (018) unbekannte Werte
  sonst nie anzeigen würden.
- QuickTime-`CreateDate` gilt als UTC und wird in die Container-Zeitzone (`TZ`) umgerechnet;
  dafür bettet das Binary `time/tzdata` ein, weil `debian:trixie-slim` keine tzdata garantiert.
  `taken_at` wird ohne Zonenangabe als lokale Zeit `YYYY-MM-DDTHH:MM:SS` gespeichert (Tech Spec §2.1).
- Datum aus WhatsApp-Namen (`IMG-/VID-JJJJMMTT-WA…`) ohne Uhrzeit → `T00:00:00`; Muster
  `JJJJMMTT_hhmmss` wird überall im Namen erkannt, aber nur mit gültigem Datum/Uhrzeit.
- `folder` = Elternordner relativ zu `/photos` (in einer normalen Bibliothek genau `JJJJ/MM`),
  damit auch Dateien außerhalb der JJJJ/MM-Struktur einen eindeutigen Ordner haben.
- Ignoriert werden Ordner und Dateien, deren Name mit `.` oder `@` beginnt, sowie `#recycle`;
  Symlinks werden nicht verfolgt (Tech Spec §9).
- Dateien mit `read_error` werden bei **jedem** Scan erneut gelesen (sonst bliebe ein
  vorübergehender Rechtefehler für immer kleben); es sind wenige.
- Löschen aus dem Index nur für Pfade, deren Ordner vollständig gelesen wurde; ein nicht lesbarer
  Unterordner oder eine Datei in der Ruhezeit behält ihre bestehende Index-Zeile — ein
  I/O-Fehler darf nie Tausende Einträge verwerfen.
- Ruhezeit-Datei, die schon im Index steht (gerade geändert): alte Zeile bleibt bis zum nächsten
  Scan nach Ablauf der Ruhezeit, weil halb geschriebene Dateien nicht gelesen werden dürfen.
- `index.db` hat eine Schema-Version (`PRAGMA user_version`); passt sie nicht zum Build, wird die
  Datei verworfen und neu aufgebaut statt migriert — sie ist ein Cache.
- Konfiguration `PC_SCAN_INTERVAL` (Standard `15m`, Minimum `1m`) und `PC_SCAN_QUIET`
  (Standard `5m`, Minimum `0s`), ungültig → Start verweigert; die E2E-Compose setzt
  `PC_SCAN_QUIET=0s`, weil das Global-Setup die Fixtures mit frischer mtime kopiert.
- Ein „Neu einlesen" während eines laufenden Scans startet keinen zweiten (Antwort 202 mit dem
  laufenden Status); ein Intervall-Tick während eines Scans entfällt — nie zwei Scans parallel.
- Fortschritt „x von y": y = Zahl der in diesem Scan zu lesenden (neuen/geänderten/fehlerhaften)
  Dateien, weil nur das ExifTool-Lesen Zeit kostet.
- Kennzeichnung „nicht lesbar" in 005 im Scan-Bereich der Startseite (Liste, max. 50 Einträge +
  Gesamtzahl); Raster/Detailansicht kommen erst mit 007, das `read_error` aus dem Index übernimmt.
- Die Scan-Statusanzeige ist eine eigene Komponente unter der Kopfzeile, damit 007 sie über dem
  Raster weiterverwenden kann; sie fragt `GET /api/scan` alle 2 s während eines Scans, sonst alle 30 s ab.
- `photocurator reindex` (Tech Spec §1) ist nicht Teil von 005 — AC7 verlangt nur Löschen von
  `index.db` + Neustart.
- Fixture-Bibliothek wird um Dateien mit Metadaten, Dateinamen-Datum, Video, PNG, eine defekte
  Datei und eine Nicht-Mediendatei erweitert (Liste in D1); versteckte/System-Ordner legen die
  Tests selbst in Temp-Verzeichnissen an, weil Git/Docker-Kontext solche Pfade unzuverlässig
  transportieren. Die Bibliothek hat danach **9** Mediendateien (relevant für 006/007-E2E).
- `CLAUDE.md` existiert im Repository nicht; Leitplanken kommen aus Profil (`.claude/ai-scrum.md`)
  und Tech Spec.

## Plan

Rein lesend; baut auf `internal/config`, `internal/store` (SQLite, modernc), `internal/api`
(Default-deny + CSRF) und der E2E-Compose aus S01 auf. Reihenfolge = Abhängigkeit:

1. **D1 — Fixtures + Namespace:** ExifTool-Konfiguration für `XMP-photocurator`
   (`internal/metadata`, eingebettet) und erweiterte `testdata/library` mit bekannten Metadaten.
2. **D2 — ExifTool-Leser** (`internal/exif`): ein Aufruf pro Block, Ergebnis pro Pfad,
   Fehler pro Datei, nur lesende Optionen.
3. **D3 — Abbildung Datei → Domäne** (`internal/metadata` Lese-Mapping, `internal/library`
   Datumsermittlung): reine Funktionen, tabellengetestet.
4. **D4 — `index.db` + Scanner** (hard): Schema §2.1, Walk-Filter, Delta, Ruhezeit,
   sicheres Löschen, Lesefehler, Fortschritt; Integrationstests mit echtem ExifTool für
   AC1/2/4/5/7/8/9.
5. **D5 — Scheduler + Konfiguration:** Start, Intervall, manueller Auslöser, nie parallel.
6. **D6 — HTTP + Verdrahtung:** `GET/POST /api/scan`, `serve` öffnet `index.db` und startet den
   Scheduler, Doku der neuen Variablen, E2E-Compose `PC_SCAN_QUIET=0s`.
7. **D7 — Oberfläche + E2E-Flow:** Scan-Statusanzeige mit „Neu einlesen", Fortschritt und
   Liste „nicht lesbar"; ein Playwright-Flow.

Nicht Teil: Foto-Liste/Raster-API (007), Thumbnails (006), `reindex`-Kommando, Schreiben.

## Deliverables

- [x] **D1 — ExifTool-Namespace `photocurator` + Fixture-Bibliothek mit Metadaten.**
  Dateien: `internal/metadata/photocurator.config` (ExifTool-Konfiguration, Perl-Syntax:
  `%Image::ExifTool::UserDefined` mit neuem XMP-Namespace, Präfix `photocurator`, URI
  `http://ns.photocurator.invalid/1.0/`, Tags `ReviewState` (string) und `ReviewedAt` (date)),
  `internal/metadata/config.go` (`//go:embed photocurator.config`; `func WriteConfig(dir string) (path string, err error)`
  schreibt die Datei in ein Temp-Verzeichnis — das Root-FS des Containers ist read-only, `/tmp`
  ist tmpfs), Test `internal/metadata/config_test.go` › „the photocurator namespace round-trips
  the review state" (Kopie von `testdata/library/2024/07/sample.jpg` via
  `testsupport.CopyFixture`, `exiftool -config <cfg> -XMP-photocurator:ReviewState=done`, dann mit
  `-config` lesen → `done`; läuft im Docker-Stage `test`, dort gibt es ExifTool).
  Neue Fixtures unter `testdata/library/` (klein, selbst erzeugt, keine privaten Inhalte; Basis
  ist das vorhandene `sample.jpg`/`sample.heic`, Video per ffmpeg, PNG per vips; Werte exakt so):
  - `2024/07/tagged.jpg`: `DateTimeOriginal=2024:07:14 10:00:00`; `XMP-dc:Subject` = Anlass,
    Geburtstag, Familie; `XMP-lr:HierarchicalSubject` = `Anlass|Geburtstag`; `IPTC:Keywords` =
    Familie, Sommer (mit `IPTC:CodedCharacterSet=UTF8`); `EXIF:XPKeywords` = `Sommer;Garten`;
    GPS 47.0707 N / 15.4395 E; `XMP-photoshop:City=Graz`, `State=Steiermark`,
    `Country=Österreich`; `XMP-iptcCore:Location=Stadtpark`;
    `XMP-iptcExt:LocationShownLocationName=Stadtpark, Graz`; `XMP-photocurator:ReviewState=done`.
  - `2024/07/tagged.heic`: `DateTimeOriginal=2024:07:20 18:30:00`; `XMP-dc:Subject` = Reise;
    `XMP-lr:HierarchicalSubject` = `Reise|Österreich`; GPS 47.2692 N / 11.4041 E;
    `XMP-photoshop:City=Innsbruck`; `XMP-photocurator:ReviewState=irrelevant`.
  - `2024/08/screenshot.png`: nur `XMP-dc:Subject` = Notizen (kein Datum → Quelle mtime).
  - `2024/08/IMG-20240812-WA0001.jpg`: ohne Metadaten (Kopie von `sample.jpg`) → Datum aus Name.
  - `2024/08/20240815_143005.jpg`: ohne Metadaten → Datum+Zeit aus Name.
  - `2024/08/clip.mp4`: 1 s, 64×64, ffmpeg ohne `faststart` (moov hinter mdat);
    `QuickTime:CreateDate=2024:08:20 08:00:00` (UTC); `XMP-dc:Subject` = Video;
    `Keys:GPSCoordinates` = 48.2082 16.3738.
  - `2024/08/defekt.jpg`: ~1 KB Zufallsbytes mit `.jpg`-Endung; muss bei `exiftool -j` einen
    `Error`-Eintrag liefern (beim Erzeugen prüfen).
  - `2024/08/notizen.txt`: kurzer Text (muss ignoriert werden).
  `testdata/README.md`: Tabelle um alle neuen Dateien ergänzen (Inhalt, Herkunft) und das
  Erzeugungs-Skript (Wegwerf-Container `debian:trixie-slim` mit `libimage-exiftool-perl`,
  `libvips-tools`, `ffmpeg`, `-config internal/metadata/photocurator.config`) dokumentieren,
  wie es dort für `sample.*` schon steht. Danach hat die Bibliothek genau 9 Mediendateien.
  `web/tests/repo/gitignore.test.ts` listet versionierte Fixtures — prüfen, ob die neuen Dateien
  dort ergänzt werden müssen, und ggf. ergänzen.

- [x] **D2 — ExifTool-Leser `internal/exif`.**
  Dateien: `internal/exif/reader.go`, `internal/exif/reader_test.go`. API:
  `type Reader struct{ ConfigPath string; Tags []string }` (Tags = gezielte Tag-Liste; der Aufrufer
  befüllt sie, D4 mit `metadata.ReadTags`),
  `func (r Reader) Read(ctx context.Context, paths []string) (map[string]Record, error)`,
  `type Record struct{ Fields map[string]any; Err string }` (Schlüssel `Gruppe:Tag`, z. B.
  `XMP-dc:Subject`, `ExifIFD:DateTimeOriginal`, `QuickTime:CreateDate`, `Composite:GPSLatitude`).
  Ein `exec.CommandContext("exiftool", "-config", cfg, "-@", argfile)` pro Aufruf (kein
  `-stay_open`, keine Shell); Argumentdatei (Muster: `internal/probe/exiftool.go` `writeTagsWith`)
  mit `-j -n -G1 -a`, je Eintrag aus `Tags` eine Zeile `-<Tag>`, optional `-fast2`
  (siehe unten), `--`, dann je Datei eine Zeile mit dem absoluten Pfad. Pfade mit Zeilenumbruch
  → sofort `Record{Err: "unsupported file name"}` ohne ExifTool. Zuordnung der JSON-Objekte über
  `SourceFile`, **nie** über die Reihenfolge; eine angefragte Datei ohne JSON-Objekt (z. B.
  „File not found"/Rechtefehler, steht nur auf stderr) bekommt `Err` aus der passenden
  stderr-Zeile bzw. „not readable"; ein Objekt mit `Error`-Feld → `Err` = dessen Text. Exit-Code
  ≠ 0 bei teilweisem Erfolg ist kein Gesamtfehler; Gesamtfehler nur, wenn ExifTool nicht startet
  oder kein gültiges JSON liefert. Nur lesende Optionen — keine `-overwrite*`, keine `-o`, keine
  Zuweisung `=`. `-fast2` nur übernehmen, wenn der Test mit `clip.mp4` weiterhin `CreateDate`,
  GPS und `XMP-dc:Subject` liefert. Tech Spec §5 (`docs/concepts/photocurator-tech-spec.md`):
  Satz ergänzen „Lesen: ein ExifTool-Aufruf je Block, kein Dauerprozess" und §1-Grafik
  „ExifTool-Pool: 1 Leseprozess…" auf „ExifTool: Lesen je Block, 1 Schreibprozess `-stay_open`"
  anpassen. Tests (echtes ExifTool, Fixture-Kopien via `testsupport.CopyFixture(t, "library")`):
  „a batch returns one record per file keyed by path, with the error of an unreadable file"
  (inkl. `defekt.jpg`, einer per `chmod 000` unlesbaren Kopie und einem Pfad mit Leerzeichen
  und führendem `-`), „the video fixture yields its QuickTime date, GPS and subject".

- [x] **D3 — Abbildung ExifTool-Record → Domäne + Aufnahmedatum.**
  Dateien: `internal/metadata/read.go`, `internal/metadata/read_test.go`,
  `internal/library/takendate.go`, `internal/library/takendate_test.go`. Reine Funktionen, keine
  ExifTool-Aufrufe in den Tests (Records als Go-Literale).
  `metadata.ReadTags []string` = alle zu lesenden Tags (eine Tabelle, Tech Spec §3.2, „Mapping
  an genau einer Stelle"): Keywords `XMP-dc:Subject`, `XMP-lr:HierarchicalSubject`,
  `IPTC:Keywords`, `EXIF:XPKeywords`, `Keys:Keywords`, `UserData:Keywords`; Datum
  `EXIF:DateTimeOriginal`, `QuickTime:CreateDate`; GPS `Composite:GPSLatitude`,
  `Composite:GPSLongitude`, `XMP-exif:GPSLatitude`, `XMP-exif:GPSLongitude`; Ort
  `XMP-iptcCore:Location`, `IPTC:Sub-location`, `XMP-photoshop:City`, `IPTC:City`,
  `XMP-iptcExt:LocationShownCity`, `XMP-photoshop:State`, `IPTC:Province-State`,
  `XMP-photoshop:Country`, `IPTC:Country-PrimaryLocationName`, `XMP-iptcExt:LocationShownLocationName`;
  `XMP-photocurator:ReviewState`. Hinweis: mit `-G1` meldet ExifTool EXIF-Tags unter
  `IFD0`/`ExifIFD`/`GPS` — das Lookup muss diese Gruppen für `EXIF:` akzeptieren (Muster:
  `Entry.Matches` in `internal/probe/exiftool.go`, nicht kopieren, sondern als kleine Funktion
  in `internal/metadata` neu und knapp).
  `metadata.Parse(fields map[string]any) Meta` mit
  `Meta{Tags []string; Lat, Lon *float64; Sublocation, City, State, Country, Label string; ReviewState string; DateTimeOriginal, QuickTimeCreate string}`:
  Tags = Pfade aus `HierarchicalSubject` (`|` → `/`), plus alle Werte der übrigen Keyword-Felder
  (`XPKeywords` an `;` getrennt, Listen oder Einzelwerte, getrimmt, leere verworfen), die nicht
  schon als Segment eines Hierarchie-Pfads vorkommen; dedupliziert, sortiert. Für das
  `tagged.jpg`-Beispiel (Subject Anlass/Geburtstag/Familie, Hierarchie `Anlass|Geburtstag`,
  IPTC Familie/Sommer, XP `Sommer;Garten`) ergibt das genau
  `[Anlass/Geburtstag Familie Garten Sommer]`. Ortsfelder: erstes nicht-leeres Feld in der
  obigen Reihenfolge; GPS: Composite vor XMP-exif, beide Koordinaten nötig. ReviewState:
  `open`/`done`/`irrelevant`, alles andere/fehlend → `open`.
  `library.TakenDate(m metadata.Meta, fileName string, mtime time.Time, loc *time.Location) (takenAt, source string)`:
  `DateTimeOriginal` (`YYYY:MM:DD HH:MM:SS`, Nullwerte `0000:…` ignorieren) → `exif`;
  `QuickTimeCreate` als UTC → in `loc` → `quicktime`; Dateiname `IMG-JJJJMMTT-WA…`/`VID-JJJJMMTT-WA…`
  (→ `T00:00:00`) bzw. `JJJJMMTT_hhmmss` irgendwo im Namen, nur gültige Kalenderwerte →
  `filename`; sonst mtime in `loc` → `mtime`. Format immer `YYYY-MM-DDTHH:MM:SS`.
  Tests: „tags merge all keyword fields with the hierarchy taking precedence",
  „place, GPS and review state come from the first filled field; unknown states read as open",
  „the taken date falls back from EXIF to QuickTime to the file name to the modification time"
  (Tabellenfälle inkl. UTC→`Europe/Vienna` im Sommer, Nulldatum, ungültiges `20241341_250000`).

- [x] **D4 — `index.db` und Scanner.** Baut auf D2 (`exif.Reader`), D3 (`metadata.Parse`,
  `metadata.ReadTags`, `library.TakenDate`) und D1 (Fixtures, `metadata.WriteConfig`).
  Dateien: `internal/store/indexdb.go` + `internal/store/indexdb_test.go`
  (`OpenIndexDB(dataDir) (*sql.DB, error)` öffnet `<dataDir>/index.db`; Schema exakt Tech Spec
  §2.1 — Tabellen `photos`, `photo_tags`, drei Indizes, `PRAGMA foreign_keys=ON`; Schema-Version
  per `PRAGMA user_version`; stimmt sie nicht mit dem Build überein, Datei schließen, löschen
  (inkl. `-wal`/`-shm`) und neu anlegen; DSN-Aufbau aus `internal/store/appdb.go` in eine kleine
  gemeinsame Funktion `openSQLite(path)` in derselben Package auslagern und in beiden nutzen),
  `internal/library/walk.go`, `internal/library/scan.go`, `internal/library/index.go`,
  `internal/library/scan_test.go`.
  Scanner: `type Scanner struct{ Root string; DB *sql.DB; Reader interface{ Read(ctx, []string) (map[string]exif.Record, error) }; Quiet time.Duration; Now func() time.Time; Loc *time.Location }`,
  `func (s *Scanner) Run(ctx context.Context, progress func(done, total int)) (Result, error)`.
  Ablauf: (1) `filepath.WalkDir` über `Root`: nur reguläre Dateien (keine Symlinks), Endungen
  `jpg jpeg heic heif png mp4 mov` case-insensitiv (`format` jpeg/heic/png/mp4/mov,
  `media_type` image/video); Ordner und Dateien mit Namen `.*`, `@*` oder `#recycle` überspringen;
  Ordner, die nicht gelesen werden konnten, merken. (2) Mit Index vergleichen: neu, geändert
  (`size` oder `mtime_ns` anders) oder `read_error IS NOT NULL` → zu lesen; mtime jünger als
  `Quiet` → überspringen (bestehende Zeile bleibt unverändert, neue Datei wird nicht angelegt).
  (3) In Blöcken zu 200 an den Reader, je Block eine Transaktion: Zeile per Upsert über
  `rel_path` (`/`-getrennt, relativ zu `Root`), `folder` = Elternordner relativ, Tags in
  `photo_tags` ersetzen, `taken_at`/`taken_source` aus `TakenDate`, `read_error` = `Record.Err`
  oder NULL (bei Fehler trotzdem Zeile mit Datum aus Name/mtime, ohne Tags/Ort), `scanned_at`;
  nach jedem Block `progress(done, total)` mit total = Zahl der zu lesenden Dateien. (4) Zeilen
  löschen, deren Datei nicht mehr gesehen wurde — **nur** für Pfade außerhalb nicht lesbarer
  Ordner. Kein Schreibzugriff auf `Root`: Dateien nur per `WalkDir`/`Lstat`, keine
  `os.OpenFile` mit Schreib-Flags. `ctx`-Abbruch beendet nach dem laufenden Block.
  `index.go`: `Count(db) (int, error)`, `Unreadable(db, limit int) ([]Unreadable, int, error)`
  (`rel_path`, `read_error`, Gesamtzahl) für D6.
  Tests in `scan_test.go` mit echtem ExifTool (`exif.Reader` mit `metadata.WriteConfig`) auf
  `testsupport.CopyFixture(t, "library")`; da die Kopie frische mtimes hat, `Quiet: 0` bzw.
  `os.Chtimes` in die Vergangenheit; versteckte/System-Einträge (`.hidden/x.jpg`,
  `2024/07/@eaDir/sample.jpg/thumb.jpg`, `._a.jpg`, `#recycle/y.jpg`, `a.MOV`, `b.JpEg`, `c.gif`,
  Symlink auf eine Datei) legt der Test im Temp-Baum selbst an. Ein zählender Wrapper um den
  Reader belegt, welche Pfade gelesen wurden. Tests:
  „the first scan indexes every media file below the root and nothing else",
  „every indexed file carries path, folder, type, size, mtime, taken date with source, tags,
  GPS, place and review state" (konkrete Erwartungswerte aus D1 für `tagged.jpg`,
  `tagged.heic`, `clip.mp4` inkl. `quicktime`-Quelle in `Europe/Vienna` = `2024-08-20T10:00:00`,
  die beiden Dateinamen-Fixtures mit Quelle `filename`, `screenshot.png` mit `mtime`),
  „a second scan reads only new or changed files and drops removed ones",
  „files younger than the quiet period are skipped until they settle",
  „a rebuilt index after deleting index.db holds the same tags, places and review states"
  (vergleicht mit den **konkreten** D1-Werten, nicht nur vorher == nachher),
  „an unreadable file is recorded and does not stop the scan" (`defekt.jpg` + `chmod 000`-Datei;
  alle anderen Dateien sind danach im Index),
  „a folder that cannot be read keeps its files in the index" (`chmod 000` auf einen
  Unterordner nach dem ersten Scan),
  „scanning leaves every file below the root unchanged" (SHA-256, mtime und Verzeichnisliste
  vor/nach zwei Scans gleich), „progress counts up to the number of files to read".
  `internal/store/indexdb_test.go`: „an index with a foreign schema version is rebuilt empty".

- [x] **D5 — Scan-Scheduler und Konfiguration.**
  Dateien: `internal/config/scan.go`, `internal/config/scan_test.go`,
  `internal/library/scheduler.go`, `internal/library/scheduler_test.go`.
  `config.ScanFromEnv(get func(string) string) (Scan, error)` mit
  `Scan{Interval, Quiet time.Duration}` aus `PC_SCAN_INTERVAL` (Standard `15m`, Minimum `1m`)
  und `PC_SCAN_QUIET` (Standard `5m`, Minimum `0s`), Format `time.ParseDuration`; ungültig →
  Fehler mit Variablenname. `library.Scheduler`:
  `NewScheduler(run func(ctx context.Context, progress func(done, total int)) (Result, error), interval time.Duration, tick func(time.Duration) <-chan time.Time)`;
  `Run(ctx)` scannt sofort (Auslöser `start`), danach bei jedem Tick (`periodic`) und bei
  `Trigger()` (`manual`; nicht blockierend, Kanal mit Puffer 1); nie zwei Scans gleichzeitig —
  Tick während eines Scans entfällt, `Trigger()` während eines Scans ist ein No-op und liefert
  `false`. `Status()` (mutex-geschützt):
  `Status{State string /* idle|running */; Trigger string; Done, Total int; StartedAt, FinishedAt time.Time; LastError string}`.
  Ein Scan-Fehler wird mit `slog` protokolliert und in `LastError` gesetzt; der Scheduler läuft
  weiter. Tests: „scan settings default to a 15 minute interval and a 5 minute quiet period and
  reject invalid values", „scans run at start, on every interval tick and on demand, never two at
  once" (Fake-`run`, der auf einen Kanal wartet; Fake-`tick`).

- [x] **D6 — HTTP-Endpunkte und Verdrahtung in `serve`.** Baut auf D4/D5.
  Dateien: `internal/api/scan.go`, `internal/api/scan_test.go`, `internal/api/server.go`,
  `cmd/photocurator/main.go`, `web/tests/e2e/compose.e2e.yml`, `deploy/README.md`.
  `api.NewHandler(static, deps AuthDeps, lib LibraryDeps)` — neuer Parameter
  `LibraryDeps{Scan ScanControl}` (006/007 ergänzen dort ihre Felder) mit
  `ScanControl interface{ Status() library.Status; Trigger() bool; Count() (int, error); Unreadable(limit int) ([]library.Unreadable, int, error) }`;
  bestehende Aufrufer/Tests von `NewHandler` anpassen (Nil-taugliche Fake-Implementierung für
  Auth-Tests). Routen hinter der bestehenden Session-/CSRF-Schicht:
  `GET /api/scan` → 200
  `{"state","trigger","done","total","started_at","finished_at","last_error","files","unreadable":[{"rel_path","error"}],"unreadable_count"}`
  (Zeiten RFC 3339 oder `null`, `unreadable` max. 50); `POST /api/scan` → `Trigger()`, 202 mit
  demselben JSON. `serve`: `import _ "time/tzdata"`; `config.ScanFromEnv` in `checkStart`
  (Start verweigert bei Fehler); `store.OpenIndexDB(paths.DataDir)`; `metadata.WriteConfig` in
  `os.TempDir()`; Scanner + Scheduler mit `time.After` als Tick in einer Goroutine mit dem
  Signal-Context, beim Shutdown beenden. `compose.e2e.yml`: `PC_SCAN_QUIET: "0s"` unter
  `environment`. `deploy/README.md`: neue optionale Variablen `PC_SCAN_INTERVAL`/`PC_SCAN_QUIET`
  mit Standard und Bedeutung (Ruhezeit schützt vor halb verschobenen Phockup-Dateien), Hinweis
  „`index.db` löschen + Neustart baut den Index neu auf". Test:
  „POST /api/scan starts a scan and GET /api/scan reports progress and unreadable files"
  (httptest mit angemeldeter Session wie in `internal/api/auth_test.go`, Fake-`ScanControl` im
  Zustand `running` 3/10; ohne `X-Requested-With` → 403, ohne Session → 401).

- [x] **D7 — Scan-Anzeige in der Oberfläche + E2E-Flow.** Baut auf D6 (JSON oben).
  Dateien: `web/src/lib/scan.ts`, `web/src/lib/scan.test.ts`, `web/src/lib/ScanStatus.svelte`,
  `web/src/App.svelte`, `web/tests/e2e/library-scan.spec.ts`.
  `scan.ts`: Typ `ScanStatus` (JSON aus D6), `fetchScan()`/`startScan()` über `api()` aus
  `web/src/lib/api.ts`, `scanStatusText(s)` → läuft: „Einlesen läuft … {done} von {total}";
  fertig: „{files} Dateien · zuletzt eingelesen {HH:MM}"; nie gelaufen: „Noch nicht eingelesen";
  `last_error` → zusätzlicher Hinweis. `ScanStatus.svelte` (unter `<AppHeader />` in der
  Hauptansicht von `App.svelte`, bestehende Überschrift und Versionsanzeige bleiben — der Smoke-Test
  prüft sie): Statuszeile mit `role="status"`, Knopf „Neu einlesen" (≥ 44 px, während eines Scans
  deaktiviert), `<progress>` während eines Scans, bei `unreadable_count > 0` ein Abschnitt
  „Nicht lesbar ({n})" mit Liste `rel_path` + Text „nicht lesbar" + Fehler; Polling alle 2 s
  während `running`, sonst alle 30 s; responsiv ab 360 px. Tests:
  Vitest `web/src/lib/scan.test.ts` › „a running scan reads as progress x of y, a finished one as
  file count and time"; Playwright `web/tests/e2e/library-scan.spec.ts` › „the library is read
  at start and on demand, unreadable files are flagged": anmelden (`helpers/auth.ts`), ohne Klick
  erscheint „9 Dateien" (Startscan; `notizen.txt` zählt nicht), `2024/08/defekt.jpg` steht mit
  „nicht lesbar" in der Liste; Klick „Neu einlesen" → Status wechselt und endet mit neuer
  Zeitangabe bzw. `finished_at` (per `GET /api/scan` vor/nach vergleichen), weiterhin 9 Dateien.

## Model Hints

- D4 → deliverable-hard: Die Lösch- und Delta-Logik ist die Stelle, an der ein plausibler Fehler
  still Daten verwirft — ein nicht lesbarer Unterordner oder eine Datei in der Ruhezeit darf
  keine bestehenden Index-Zeilen entfernen, und das Delta über `(size, mtime_ns, read_error)`
  muss mit Upsert, Tag-Ersetzung und blockweisen Transaktionen über drei Pakete (store, exif,
  metadata) zusammenpassen.
- Review: → default

## Acceptance Tests

- AC1 → unit `internal/library/scan_test.go` › "the first scan indexes every media file below the root and nothing else"
- AC2 → unit `internal/library/scan_test.go` › "every indexed file carries path, folder, type, size, mtime, taken date with source, tags, GPS, place and review state"; Tag-Zusammenführung zusätzlich `internal/metadata/read_test.go` › "tags merge all keyword fields with the hierarchy taking precedence" und › "place, GPS and review state come from the first filled field; unknown states read as open"
- AC3 → unit `internal/library/takendate_test.go` › "the taken date falls back from EXIF to QuickTime to the file name to the modification time" (plus die Quellen-Erwartungen im AC2-Test für die echten Fixtures)
- AC4 → unit `internal/library/scan_test.go` › "a second scan reads only new or changed files and drops removed ones"
- AC5 → unit `internal/library/scan_test.go` › "files younger than the quiet period are skipped until they settle"
- AC6 → e2e `tests/e2e/library-scan.spec.ts` › "the library is read at start and on demand, unreadable files are flagged" (Start + Knopf + Status über die echte Oberfläche); Intervall und Nicht-Parallelität: unit `internal/library/scheduler_test.go` › "scans run at start, on every interval tick and on demand, never two at once"; Fortschrittsanzeige: unit `web/src/lib/scan.test.ts` › "a running scan reads as progress x of y, a finished one as file count and time" und `internal/library/scan_test.go` › "progress counts up to the number of files to read" — ein laufender Scan der 9-Dateien-Fixture ist im E2E zu kurz, um den Zwischenstand stabil zu sehen.
- AC7 → unit `internal/library/scan_test.go` › "a rebuilt index after deleting index.db holds the same tags, places and review states"
- AC8 → unit `internal/library/scan_test.go` › "an unreadable file is recorded and does not stop the scan"; Kennzeichnung in der Oberfläche: e2e `tests/e2e/library-scan.spec.ts` › "the library is read at start and on demand, unreadable files are flagged"
- AC9 → unit `internal/library/scan_test.go` › "scanning leaves every file below the root unchanged"

## Done

**Zusammenfassung:** ExifTool-Namespace `photocurator` + 7 neue Fixtures (9 Mediendateien), `internal/exif`-Leser (ein Aufruf je Block), Mapping/Aufnahmedatum, `index.db` (Schema §2.1, `user_version`) mit Scanner (Delta, Ruhezeit, sicheres Löschen, Lesefehler), Scheduler + `PC_SCAN_*`, `GET/POST /api/scan`, Scan-Anzeige mit „Neu einlesen" und E2E-Flow.

**Commit:** `005: Bibliotheks-Index — exif-Leser, index.db-Scanner, Scheduler, /api/scan, Scan-Anzeige + E2E`

**Verifikation (Schmales Gate; `test-story` = none → volles `test`):** `build`, `lint`, `typecheck`, `test` (`docker build --target test . && npm --prefix web run test`), `e2e-story` (`npx playwright test tests/e2e/library-scan.spec.ts`) grün; nach den Review-Fixes `test` + `lint` erneut grün. Review (default, 1 Zyklus): PASS AC1–AC9.
AC → Test (alle in der Quelle vorhanden und gelaufen; Go-Namen = abgeleitete CamelCase-Funktionsnamen der Sätze, Go-Lauf ohne `-v`, Nachweis = Paket-`ok`): AC1–AC5, AC7–AC9 `internal/library/scan_test.go`/`takendate_test.go`, AC2 zusätzlich `internal/metadata/read_test.go`; AC6 `library-scan.spec.ts` (e2e, grün), `scheduler_test.go`, `scan.test.ts`, `scan_test.go` (Fortschritt); AC8 zusätzlich e2e. Kein `manual residue`.

**Decisions / Abweichungen:**
- `-fast2` nicht übernommen: es verlor bei `clip.mp4` `CreateDate` und GPS; Tech Spec §5 nennt „kein -fast2".
- Typ heißt `library.UnreadableFile` (Go erlaubt Typ und Funktion `Unreadable` nicht im selben Paket).
- `tagged.heic` hat Tags `[Reise/Österreich]` (flaches `Reise` ist Segment der Hierarchie und entfällt, wie spezifiziert).
- Scanner löst `Root` per `EvalSymlinks` auf (ein Symlink-Root darf den Index nicht leeren); mtime mehr als `Quiet` in der Zukunft gilt als „gesetzt" (falscher Zeitstempel blockiert nicht ewig; Test vorhanden); begonnener Block läuft bei Abbruch zu Ende, danach kein Löschen.
- `clip.mp4`: ExifTool verschiebt `moov` beim Schreiben nach vorn; Fixture per Perl-Skript (steht in `testdata/README.md`) wieder mit `moov` hinter `mdat` erzeugt.
- `PC_SCAN_*` stehen nur als `environment:`-Einträge der Compose-Datei dokumentiert, nicht in `.env` (`deploy/.env.example` war für den Agenten gesperrt und der Test verlangt dort jede Compose-Variable); Weiterleitung in `deploy/docker-compose.yml` ist offener Punkt.
- `openSQLite` setzt `foreign_keys(1)` jetzt auch für `app.db` (dort keine FKs).
- Offen/Review ungefixt (kosmetisch): ein beschädigtes, nicht-SQLite-`index.db` verhindert den Start statt neu aufgebaut zu werden; Scheduler meldet kurz `idle` zwischen `Trigger()` und Schleife (UI pollt dann einmal 30 s); `finished_at` hat Sekundenauflösung (E2E pollt bis Wert abweicht).

tiers: D 7 / hard 1 · review default · cycles 1 · agents 11
