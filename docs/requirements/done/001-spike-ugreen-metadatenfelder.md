---
id: 001
title: Spike — welche Metadatenfelder liest UGREEN Fotos?
status: done # draft -> ready -> in-progress -> done
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

- [x] **AC1** — `photocurator probe generate --out <dir>` erzeugt für jedes Format (JPEG, HEIC,
  PNG, MP4, MOV) und jedes für dieses Format schreibbare Kandidatenfeld genau eine Testdatei,
  deren eindeutiger Marker (z. B. `pcprobe-heic-xmpsubject`) nur in diesem einen Feld steht.
- [x] **AC2** — Ein automatischer Read-back-Test weist für jede erzeugte Datei nach, dass ihr
  Marker im Zielfeld steht und in keinem anderen Kandidatenfeld.
- [x] **AC3** — Für Hierarchie-Felder enthält der Marker einen Pfad (z. B.
  `pcprobe|heic|hierarchical`), um zu sehen, wie UGREEN Hierarchien darstellt.
- [x] **AC4** — Für Ort gibt es je Format eine Datei nur mit GPS-Koordinaten (unterschiedliche,
  leicht erkennbare Orte je Feldvariante) und je eine nur mit Ortstext-Feldern.
- [x] **AC5** — Für MP4 und MOV gibt es zusätzlich Varianten ohne eingebettete Metadaten, aber
  mit Sidecar `<name>.xmp` bzw. `<name>.<ext>.xmp`.
- [x] **AC6** — Es gibt eine Datei mit allen Feldern gleichzeitig („combo") je Format sowie eine
  WhatsApp-artige JPEG ohne jegliche Metadaten.
- [x] **AC7** — Alle Testdateien sind generiert (keine privaten Inhalte), zeigen ihren Marker
  zusätzlich als Text im Bild bzw. Video und tragen ein festes, ungewöhnliches Aufnahmedatum
  (Platzhalter: 2001-01-01), damit sie in UGREEN Fotos leicht auffindbar sind.
- [x] **AC8** — Der Generator schreibt `PROTOKOLL.md` in den Ausgabeordner: eine Tabelle mit
  einer Zeile je Datei (Datei, Format, Feld, Marker) und leeren Spalten für die Beobachtungen
  (in Suche gefunden, als Tag/Kategorie sichtbar, in bedingtem Album nutzbar, Ort angezeigt,
  Zeit bis sichtbar).
- [x] **AC9** — `photocurator probe mutate --dir <dir>` ändert in bereits indexierten
  Testdateien den Marker in `XMP-dc:Subject` — einmal mit erhaltenem, einmal mit neuem
  Änderungsdatum — und das Protokoll enthält die Anleitung für den Erkennungstest (automatisch /
  nach manuellem Rescan / Verzögerung).
- [x] **AC10** — Das Protokoll enthält zusätzlich die Prüffragen: Welche Bedingungen bieten
  bedingte Alben überhaupt an? Schreibt UGREEN Fotos selbst Metadaten in Dateien (z. B. beim
  Taggen oder Favorisieren in der App)?
- [x] **AC11** — `docs/concepts/photocurator-metadaten-strategie.md` existiert mit der leeren
  Feldmatrix, der vorläufigen Strategie aus der Tech Spec und einem Abschnitt „Entscheidung
  nach Spike" (auszufüllen nach der Durchführung).
- [ ] **AC12** — Durchführung auf dem NAS und Eintrag der Beobachtungen. **manual residue:**
  proprietäre UGREEN Fotos-App auf dem NAS, nicht automatisierbar.

## Open Questions

Keine.

## Decisions (Sprint)

- **Paket `internal/probe`, Unterbefehl `probe generate|mutate` im Binary aus 002** — so sieht es
  die Tech Spec §1 vor, und das Werkzeug läuft im Runtime-Image mit genau den ExifTool/libvips/
  ffmpeg-Versionen, die später auch schreiben.
- **Probe ruft ExifTool direkt (einmalig pro Datei, Argumentdatei `-@`, kein Shell-Aufruf)** —
  der Writer (009) existiert noch nicht, und die Probe schreibt nur in selbst erzeugte Dateien,
  nie in `/photos`; die Argumentdatei hält die Sicherheitsregel aus Tech Spec §9 ein.
- **Feldmatrix ist eine feste Tabelle im Code (`internal/probe/matrix.go`)** — eine Stelle, aus
  der Generator, Read-back-Test, Protokoll und Doku-Konsistenztest lesen; „schreibbar" heißt:
  steht in der Tabelle (Kandidaten aus Tech Spec §3.1 je Format, siehe D1/D2).
- **IPTC-IIM und `EXIF:XPKeywords` nur für JPEG** — Tech Spec §3.1/§3.2 beschränkt sie so; HEIC
  hat kein IPTC-IIM, PNG-IPTC ist kein realistischer Schreibweg für 009.
- **Prüfstatus (`XMP-photocurator:*`) wird nicht geprüft** — der eigene Namespace ist für UGREEN
  unbekannt und wird bewusst nie für Suche/Alben verwendet (Tech Spec §3.2).
- **Ortstext: eine Datei je Feldfamilie** (XMP-photoshop, XMP-iptcCore Location, XMP-iptcExt
  LocationShown, IPTC-IIM nur JPEG), Werte innerhalb der Familie je Teilfeld eigener Marker — so
  ist erkennbar, welche Familie und welches Teilfeld UGREEN anzeigt.
- **GPS: festes Wahrzeichen je Feldvariante, unabhängig vom Format** (EXIF-GPS = Eiffelturm,
  XMP-exif-GPS = Kolosseum, `Keys:GPSCoordinates` = Sydney Opera House,
  `UserData:GPSCoordinates` = Freiheitsstatue, Combo = Taj Mahal, Sidecar = Machu Picchu) — der
  Ort in der UGREEN-Karte verrät die Variante auf einen Blick, das Format steht im Protokoll.
- **Dateinamen sind neutral (`pc001.jpg` …), nie der Marker** — sonst würde die UGREEN-Suche den
  Marker über den Dateinamen finden und das Ergebnis wäre wertlos.
- **Die WhatsApp-artige JPEG zeigt `pcprobe-jpeg-nometa` im Bild und dient als OCR-Kontrolle** —
  findet die Suche sie, liest UGREEN Bildtext, und Treffer zählen nur, wenn die Kontrolle nicht
  anschlägt (Hinweis im Protokoll).
- **Aufnahmedatum 2001-01-01 12:00 (lokal)** in `EXIF:DateTimeOriginal`/`CreateDate` (Bilder)
  bzw. QuickTime `CreateDate`/`MediaCreateDate`/`TrackCreateDate` = 11:00 UTC (Videos, auch die
  Sidecar-Varianten), dazu mtime aller Dateien = 2001-01-01 12:00; die WhatsApp-JPEG trägt das
  Datum nur über Namen (`IMG-20010101-WA0001.jpg`) und mtime — „ohne jegliche Metadaten" hat
  Vorrang vor AC7, das Datum bleibt trotzdem auffindbar.
- **Markertext im Bild wird in Go gerendert (`golang.org/x/image/font/basicfont`, hochskaliert)**
  — kein Font im read-only Runtime-Image nötig; Videos entstehen aus diesem Bild per ffmpeg.
- **HEIC wird per `vips heifsave` erzeugt; fehlt der HEVC-Encoder im Image, ergänzt D1 ihn im
  `Dockerfile` (test- und runtime-Stage)** — ohne HEIC-Testdatei fehlt das wichtigste
  Handy-Format im Spike.
- **Mutate arbeitet auf zwei eigenen JPEG-Dateien** (`pcmut-keep.jpg`, `pcmut-new.jpg`, Marker
  `pcprobe-jpeg-mutate-keep-v1` / `…-new-v1`) statt auf Matrix-Dateien — beide Varianten laufen
  gleichzeitig im selben Format, und AC1 („genau eine Datei je Feld") bleibt unberührt.
- **Mutate erhöht die Versionsnummer (`-v1` → `-v2` …) und fasst nur diese zwei Dateinamen mit
  `pcprobe-`-Marker an, sonst Abbruch** — wiederholbar und kann nie echte Fotos verändern.
- **`generate` verweigert einen nicht leeren Ausgabeordner** — schützt vor versehentlichem
  Erzeugen in einem echten Fotoordner.
- **CLI-Kriterien werden per Go-Integrationstest über den echten Befehlseinstieg (Argumentliste
  → Unterbefehl) im Docker-Stage `test` geprüft, nicht per Playwright** — die Probe hat keine
  Weboberfläche; der Befehlseinstieg mit echtem ExifTool/ffmpeg/libvips ist ihre echte
  Oberfläche.
- **Read-back prüft „in keinem anderen Feld" gegen den vollständigen ExifTool-Dump (`-a -G1`),
  nicht nur gegen die Kandidatenliste** — strenger als AC2 und fängt auch Nebeneffekte wie
  Feld-Synchronisation ab.
- **AC11 wird durch einen Konsistenztest belegt: das Strategie-Dokument nennt jede Zeile der
  Feldmatrix** — so kann Doku und Generator nicht auseinanderlaufen; das Dokument muss dafür im
  Docker-Build-Kontext liegen.

## Plan

Neues Paket `internal/probe` + Unterbefehl `probe` im Binary aus Story 002 (Voraussetzung:
002 ist gebaut; Tests laufen im Docker-Stage `test` mit echtem ExifTool/libvips/ffmpeg).

1. **Matrix + Bilder (D1):** Tabelle Format × Feld (Feldkey, ExifTool-Tags, Marker bzw. Ort),
   Rendern des Markertexts, Erzeugen der JPEG/HEIC/PNG-Dateien inkl. GPS-, Ortstext-, Combo- und
   WhatsApp-Datei, Datum + mtime; Read-back-Tests.
2. **Videos (D2):** MP4/MOV aus gerendertem Bild per ffmpeg, QuickTime- und XMP-Felder,
   Sidecar-Varianten `<name>.xmp` / `<name>.<ext>.xmp`; Read-back-Tests.
3. **Mutate (D3):** zwei eigene JPEGs im Generator, `probe mutate --dir` ersetzt den
   `XMP-dc:Subject`-Marker einmal mit `-P` (mtime bleibt), einmal ohne.
4. **Protokoll (D4):** `PROTOKOLL.md` im Ausgabeordner — Ablauf, Tabelle je Datei mit leeren
   Beobachtungsspalten, Ortsliste, OCR-Kontrolle, Erkennungstest-Anleitung, Prüffragen.
5. **Strategie-Dokument (D5):** `docs/concepts/photocurator-metadaten-strategie.md` mit leerer
   Feldmatrix, vorläufiger Strategie, „Entscheidung nach Spike" + Konsistenztest.

Reihenfolge D1 → D2 → D3 → D4 → D5. Kein Schreiben in echte Fotos; die Durchführung auf dem NAS
(AC12) macht der Nutzer nach dem Sprint.

## Deliverables

- [x] **D1 — Feldmatrix, Markerbild, Bild-Testdateien (JPEG/HEIC/PNG)**
  Dateien: `internal/probe/matrix.go`, `internal/probe/render.go`, `internal/probe/exiftool.go`,
  `internal/probe/generate.go`, `internal/probe/generate_test.go`, `internal/probe/matrix_test.go`,
  `cmd/photocurator/` (Unterbefehl `probe generate --out <dir>` an den bestehenden
  Befehlseinstieg aus 002 hängen), `go.mod`/`go.sum` (`golang.org/x/image`), ggf. `Dockerfile`.
  - `matrix.go`: Typ `Probe{File, Format, FieldKey, Tags []string, Marker, Place}`; Funktion
    `Probes() []Probe` liefert die feste Liste. Marker = `pcprobe-<format>-<fieldkey>`
    (klein, ASCII, ohne Leerzeichen); Hierarchie-Marker als Pfad `pcprobe|<format>|hierarchical`.
    Dateinamen neutral fortlaufend `pc001.<ext>` …, **nie** den Marker enthaltend.
  - Bildfelder (je Format eine Datei pro Feld): alle drei Formate: `xmpsubject`
    (`XMP-dc:Subject`), `hierarchical` (`XMP-lr:HierarchicalSubject`), `xmpdescription`
    (`XMP-dc:Description`), `exifdescription` (`EXIF:ImageDescription`), `gps-exif`
    (`EXIF:GPSLatitude/Ref`, `GPSLongitude/Ref`, Eiffelturm 48.8584, 2.2945), `gps-xmp`
    (`XMP-exif:GPSLatitude/Longitude`, Kolosseum 41.8902, 12.4922), `loc-photoshop`
    (`XMP-photoshop:City/State/Country`), `loc-iptccore` (`XMP-iptcCore:Location`),
    `loc-iptcext` (`XMP-iptcExt:LocationShown` City/ProvinceState/CountryName/LocationName),
    `combo`. Nur JPEG zusätzlich: `iptckeywords` (`IPTC:Keywords`, mit
    `IPTC:CodedCharacterSet=UTF8`), `xpkeywords` (`EXIF:XPKeywords`), `iptccaption`
    (`IPTC:Caption-Abstract`), `loc-iptc` (`IPTC:City/Sub-location/Province-State/
    Country-PrimaryLocationName`). Ortstext-Teilfelder tragen je eigenen Marker
    (`pcprobe-jpeg-loc-photoshop-city` …); GPS-Dateien tragen keinen Textmarker in Metadaten.
  - `combo`: alle Felder des Formats gleichzeitig, jedes mit eigenem Marker
    `pcprobe-<format>-combo-<fieldkey>` (Hierarchie `pcprobe|<format>|combo`), GPS = Taj Mahal
    (27.1751, 78.0421) in allen GPS-Varianten.
  - WhatsApp-artige JPEG `IMG-20010101-WA0001.jpg`: keine EXIF/XMP/IPTC (`-all=`), Pixel zeigen
    `pcprobe-jpeg-nometa` (OCR-Kontrolle).
  - `render.go`: zeichnet Marker (bzw. bei GPS-Dateien Feldkey + Ortsname) mit
    `basicfont.Face7x13` schwarz auf weiß, nearest-neighbour ×4 hochskaliert, 640×480; JPEG/PNG
    per Go-Stdlib, HEIC per `vips heifsave` aus einem temporären PNG. Fehlt der HEVC-Encoder
    im Image, im `Dockerfile` (Stages `test` und `runtime`) das Encoder-Paket ergänzen.
  - `exiftool.go`: ein ExifTool-Aufruf pro Datei mit Argumentdatei (`-@`), Werte zeilenweise,
    `-overwrite_original`, kein Shell-Aufruf; Lesen per `-j -a -G1 -n`.
  - Alle Dateien: `EXIF:DateTimeOriginal`/`EXIF:CreateDate` = `2001:01:01 12:00:00` (außer
    WhatsApp-JPEG), danach mtime = 2001-01-01 12:00 lokal (`os.Chtimes`).
  - `generate` bricht ab, wenn `--out` existiert und nicht leer ist.
  - Tests (`generate_test.go`, Integration, echtes ExifTool/libvips): siehe Acceptance Tests
    AC1, AC2 (Bilder), AC4, AC6, AC7; `matrix_test.go`: AC3. Der Read-back-Test liest den
    vollständigen Dump je Datei und prüft: Marker steht im Zielfeld und in **keinem** anderen
    Tag; GPS-Datei hat die Koordinaten (±1e-4) nur in ihrer Variante, keine Ortstext-Felder;
    Ortstext-Datei hat keine GPS-Tags. Der Pixeltest vergleicht das dekodierte Bild mit dem
    Render-Ergebnis des eigenen Markers (kleine mittlere Abweichung) und eines fremden Markers
    (große Abweichung).

- [x] **D2 — Video-Testdateien (MP4/MOV) inkl. Sidecar-Varianten**
  Dateien: `internal/probe/matrix.go` (Video-Zeilen ergänzen), `internal/probe/video.go`,
  `internal/probe/generate.go` (Video-Zweig aufrufen), `internal/probe/video_test.go`.
  Wiederverwenden: `render.go` (Markerbild), `exiftool.go` (Argumentdatei-Aufruf) und den
  Read-back-Helfer aus `generate_test.go` aus D1 — keine zweite Kopie davon.
  - Video aus dem gerenderten PNG: `ffmpeg -loop 1 -i frame.png -t 1 -r 10 -c:v libx264
    -pix_fmt yuv420p -map_metadata -1` → `.mp4` bzw. `.mov`, 640×480.
  - Videofelder (je Format MP4 und MOV eine Datei pro Feld): `xmpsubject`, `hierarchical`
    (Marker `pcprobe|<format>|hierarchical`), `xmpdescription`, `keys-keywords`
    (`Keys:Keywords`), `userdata-keywords` (`UserData:Keywords`), `keys-description`
    (`Keys:Description`), `gps-keys` (`Keys:GPSCoordinates`, Sydney Opera House −33.8568,
    151.2153), `gps-userdata` (`UserData:GPSCoordinates`, Freiheitsstatue 40.6892, −74.0445),
    `gps-xmp` (`XMP-exif:GPSLatitude/Longitude`, Kolosseum), `loc-photoshop`, `loc-iptccore`,
    `loc-iptcext` (wie Bilder), `combo` (alle gleichzeitig, eigene Marker
    `pcprobe-<format>-combo-<fieldkey>`, GPS Taj Mahal). Lehnt ExifTool ein Feld für ein Format
    ab, wird die Zeile mit Code-Kommentar entfernt und in der Story unter `## Done` vermerkt.
  - Sidecar-Varianten je Format: `sidecar-name` (Video `pcNNN.<ext>` + `pcNNN.xmp`) und
    `sidecar-ext` (`pcNNN.<ext>` + `pcNNN.<ext>.xmp`). Video ohne eingebettete Marker-/GPS-/
    Ortsfelder; das Sidecar (per `exiftool -o <sidecar>`) trägt `XMP-dc:Subject`,
    `XMP-lr:HierarchicalSubject`, `XMP-dc:Description`, `XMP-photoshop:City` mit Markern
    `pcprobe-<format>-<variant>-<fieldkey>` und XMP-GPS Machu Picchu (−13.1631, −72.5450).
  - Datum aller Videos: QuickTime `CreateDate`, `ModifyDate`, `TrackCreateDate`,
    `MediaCreateDate` = `2001:01:01 11:00:00` (UTC, `-api QuickTimeUTC`), mtime wie D1 (auch
    der Sidecars).
  - Tests (`video_test.go`): AC1 (eine Datei je Videofeld), AC2/AC4/AC6 (Read-back aller
    Videozeilen inkl. GPS, Ortstext, Combo), AC5, AC7 (Video: erstes Frame per ffmpeg
    extrahieren und wie in D1 mit dem Render-Ergebnis vergleichen; Datum prüfen).

- [x] **D3 — `probe mutate --dir <dir>` (Erkennungstest)**
  Dateien: `internal/probe/mutate.go`, `internal/probe/mutate_test.go`,
  `internal/probe/generate.go` (die zwei Mutate-Dateien mit erzeugen), `cmd/photocurator/`
  (Unterbefehl `probe mutate --dir`). Wiederverwenden: `exiftool.go`, `render.go`.
  - Generator legt zusätzlich `pcmut-keep.jpg` und `pcmut-new.jpg` an (JPEG, Datum/mtime wie
    D1, Pixeltext = Marker), `XMP-dc:Subject` = `pcprobe-jpeg-mutate-keep-v1` bzw.
    `pcprobe-jpeg-mutate-new-v1`. Sie gehören nicht zur Feldmatrix.
  - `mutate` liest den aktuellen Marker, erhöht die Version (`-v1` → `-v2` …) und ersetzt den
    Eintrag in `XMP-dc:Subject` (alten Wert entfernen, neuen setzen): `pcmut-keep.jpg` mit `-P`
    (mtime bleibt), `pcmut-new.jpg` ohne `-P` (mtime = jetzt). Gibt alte/neue Marker aus.
  - Bricht ohne Änderung ab, wenn eine der zwei Dateien fehlt oder ihr Marker nicht mit
    `pcprobe-jpeg-mutate-` beginnt. Fasst keine anderen Dateien an.
  - Test (`mutate_test.go`): generieren, mutieren → neue Marker stehen in `XMP-dc:Subject`, alte
    nicht mehr; mtime von keep unverändert, von new neuer als vorher; zweiter Lauf ergibt `-v3`;
    Ordner ohne Probe-Dateien → Fehler, Ordnerinhalt unverändert.

- [x] **D4 — `PROTOKOLL.md` im Ausgabeordner**
  Dateien: `internal/probe/protocol.go`, `internal/probe/protocol_test.go`,
  `internal/probe/generate.go` (am Ende `PROTOKOLL.md` schreiben). Eingabe: `Probes()` aus
  `matrix.go` plus die Mutate- und WhatsApp-Datei.
  - Inhalt (Deutsch): Zweck; Ablauf (Ordner nach `Sortiert/_pcprobe/` kopieren, Indexierung
    abwarten, je Zeile prüfen, danach Ordner löschen); Aufruf im Container
    (`docker run --rm -v "<ordner>:/out" photocurator:dev probe generate --out /out`).
  - Tabelle, eine Zeile je erzeugter Datei (auch Sidecar-Paar, Combo, WhatsApp, Mutate):
    Datei | Format | Feld | Marker (bzw. Ort) | in Suche gefunden | als Tag/Kategorie sichtbar |
    in bedingtem Album nutzbar | Ort angezeigt | Zeit bis sichtbar — letzte fünf Spalten leer.
  - Ortsliste (Wahrzeichen → GPS-Variante); Hinweis OCR-Kontrolle: findet die Suche
    `pcprobe-jpeg-nometa`, liest UGREEN Bildtext → Suchtreffer nur werten, wenn diese Kontrolle
    negativ ist; Hinweis: Hierarchie-Marker sowohl nach `pcprobe` als auch nach `hierarchical`
    suchen.
  - Abschnitt „Erkennungstest": nach Indexierung `probe mutate --dir` ausführen; je Datei
    notieren, ob der neue Marker automatisch, erst nach manuellem Rescan oder nach welcher
    Verzögerung gefunden wird (mit/ohne neues Änderungsdatum) — ergibt den Standard für
    `PC_PRESERVE_MTIME` (Story 009).
  - Abschnitt „Prüffragen": Welche Bedingungen bieten bedingte Alben überhaupt an? Schreibt
    UGREEN Fotos selbst Metadaten in Dateien (z. B. beim Taggen oder Favorisieren in der App)?
    — mit Anleitung: eine Probe-Datei in der App taggen/favorisieren, dann
    `exiftool -a -G1 <datei>` vergleichen.
  - Test (`protocol_test.go`): AC8, AC9 (Anleitungsteil), AC10.

- [x] **D5 — Strategie-Dokument `docs/concepts/photocurator-metadaten-strategie.md`**
  Dateien: `docs/concepts/photocurator-metadaten-strategie.md` (neu, Deutsch),
  `internal/probe/strategydoc_test.go`, ggf. `.dockerignore` (das Dokument muss im Build-Kontext
  des Stages `test` liegen).
  - Inhalt: Zweck und Verweis auf Story 001 und Tech Spec §3; **leere Feldmatrix** — Tabelle mit
    einer Zeile je Format × Feldkey aus `Probes()` (inkl. Sidecar-Varianten), Spalten Format |
    Feldkey | ExifTool-Tags | Suche | Tag/Kategorie | bedingtes Album | Ort angezeigt (leer);
    Abschnitt „Erkennung von Änderungen" (leer, mit den Fragen aus dem Protokoll);
    „Vorläufige Strategie" (Tech Spec §3.2 sinngemäß übernommen); „Entscheidung nach Spike"
    (Platzhalter: Felder je Format, Video-Sidecars ja/nein, `PC_PRESERVE_MTIME`-Standard,
    Regel falls UGREEN selbst schreibt).
  - Test (`strategydoc_test.go`): jede `(Format, Feldkey)`-Kombination aus `Probes()` steht in
    der Matrix-Tabelle des Dokuments, und die Überschrift „Entscheidung nach Spike" existiert.

## Model Hints

Alle Deliverables im Default-Tier — neuer, isolierter Code ohne Regressionsfläche; die
heikle Stelle (Feld-Isolation bei ExifTool) belegt der Read-back-Test gegen den vollständigen
Dump.

Review: → default — die naheliegende Fehlumsetzung (Marker im Dateinamen, Feld-Synchronisation)
ist durch benannte Tests abgedeckt.

## Acceptance Tests

Alle Go-Tests laufen im Docker-Stage `test` (echtes ExifTool/libvips/ffmpeg) über den echten
Befehlseinstieg `probe generate|mutate`; die Probe hat keine Weboberfläche, daher kein
Playwright-Flow.

- AC1 → integration `internal/probe/generate_test.go` › `TestGenerate_OneFilePerFormatAndField`
  ("generate writes exactly one file per format and candidate field") — D1, Video-Zeilen
  über dieselbe Matrix in `internal/probe/video_test.go` › `TestGenerateVideo_OneFilePerField` — D2
- AC2 → integration `internal/probe/generate_test.go` › `TestGenerate_MarkerOnlyInTargetField`
  — D1; `internal/probe/video_test.go` › `TestGenerateVideo_MarkerOnlyInTargetField` — D2
- AC3 → unit `internal/probe/matrix_test.go` › `TestProbes_HierarchyMarkersArePaths` (+ Read-back
  aus AC2 belegt, dass der Pfad so im Feld steht) — D1
- AC4 → integration `internal/probe/generate_test.go` ›
  `TestGenerate_GPSOnlyAndLocationTextOnlyFiles` (Bilder) — D1;
  `internal/probe/video_test.go` › `TestGenerateVideo_MarkerOnlyInTargetField` (GPS-/Ortszeilen
  der Videos) — D2
- AC5 → integration `internal/probe/video_test.go` ›
  `TestGenerateVideo_SidecarVariantsCarryMarkersOutsideTheVideo` — D2
- AC6 → integration `internal/probe/generate_test.go` ›
  `TestGenerate_ComboAndWhatsAppLikeFile` — D1; Video-Combo über
  `TestGenerateVideo_MarkerOnlyInTargetField` — D2
- AC7 → integration `internal/probe/generate_test.go` ›
  `TestGenerate_MarkerVisibleInPixelsDateFixedNameNeutral` — D1;
  `internal/probe/video_test.go` › `TestGenerateVideo_MarkerVisibleInFrameAndDateFixed` — D2
- AC8 → unit `internal/probe/protocol_test.go` › `TestProtocol_OneRowPerGeneratedFile` — D4
- AC9 → integration `internal/probe/mutate_test.go` ›
  `TestMutate_ReplacesSubjectKeepingAndRenewingMtime` — D3; Anleitung:
  `internal/probe/protocol_test.go` › `TestProtocol_ContainsDetectionGuideAndQuestions` — D4
- AC10 → unit `internal/probe/protocol_test.go` ›
  `TestProtocol_ContainsDetectionGuideAndQuestions` — D4
- AC11 → unit `internal/probe/strategydoc_test.go` › `TestStrategyDoc_ListsEveryProbeField` — D5
- AC12 → manual residue: Durchführung in der proprietären UGREEN Fotos-App auf dem NAS, nicht
  automatisierbar (Nutzer, nach dem Sprint).

## Done

Der Spike liefert das Werkzeug und das Protokoll: `photocurator probe generate --out <dir>` erzeugt
alle Test-Bilder/-Videos (JPEG/HEIC/PNG/MP4/MOV, Sidecar-Varianten, Combo, WhatsApp-Kontrolle,
zwei Mutate-JPEGs) samt `PROTOKOLL.md`; `probe mutate --dir <dir>` ändert den Marker für den
Erkennungstest. Das Strategie-Dokument mit leerer Feldmatrix liegt unter `docs/concepts/`.

**Commit:** `001: Spike-Werkzeug probe generate/mutate, PROTOKOLL.md, Metadaten-Strategie-Dokument`

**Verifikation (Narrow Gate, kein `test-story`/`e2e-story` nötig):** `build`, `test` (Docker-Stage
`test`, 17 Probe-Tests + web), `lint`, `typecheck` grün; kein e2e (CLI ohne Weboberfläche).
Review: 1 Zyklus (default), Findings behoben und neu verifiziert.
AC → Test (alle bestanden): AC1 `TestGenerate_OneFilePerFormatAndField` / `TestGenerateVideo_OneFilePerField`;
AC2 `TestGenerate_MarkerOnlyInTargetField` / `TestGenerateVideo_MarkerOnlyInTargetField`;
AC3 `TestProbes_HierarchyMarkersArePaths`; AC4 `TestGenerate_GPSOnlyAndLocationTextOnlyFiles` + Video-Test;
AC5 `TestGenerateVideo_SidecarVariantsCarryMarkersOutsideTheVideo`; AC6 `TestGenerate_ComboAndWhatsAppLikeFile` + Video-Test;
AC7 `TestGenerate_MarkerVisibleInPixelsDateFixedNameNeutral` / `TestGenerateVideo_MarkerVisibleInFrameAndDateFixed`;
AC8 `TestProtocol_OneRowPerGeneratedFile`; AC9 `TestMutate_ReplacesSubjectKeepingAndRenewingMtime` + `TestProtocol_ContainsDetectionGuideAndQuestions`;
AC10 `TestProtocol_ContainsDetectionGuideAndQuestions`; AC11 `TestStrategyDoc_ListsEveryProbeField`.
AC12: manual residue — Durchführung in der proprietären UGREEN Fotos-App auf dem NAS (Nutzer, nach dem Sprint).
Der volle Regressions-Gate (Sprint) steht aus.

**Decisions:**
- `UserData:Keywords` (Video) entfällt: ExifTool 13.25 kann das Feld weder in MP4 noch MOV schreiben (Code-Kommentar in `videoDefs()`); alle anderen Felder schreiben sauber.
- HEIC-Encoder: `libheif-plugin-x265` im `native`-Stage des Dockerfile ergänzt (test + runtime).
- IPTC-IIM City/Sub-location/Province-State sind auf 32 Bytes begrenzt; Combo-Marker von `loc-iptc` nutzen Kurzsuffixe (`city`, `sub`, `st`, `country`).
- `Probe.Place` ist ein Struct {Name, Lat, Lon}; XMP-GPS wird vorzeichenbehaftet geschrieben.
- ExifTool-Aufrufe übergeben Pfade nach `--`; Mutate ist nicht atomar (Fehler beim zweiten Schreiben lässt die erste Datei mutiert; Neustart zählt weiter).
- `.dockerignore`/`Dockerfile`: nur `docs/concepts/photocurator-metadaten-strategie.md` kommt in den Build-Kontext des Test-Stages.
- `testsupport` bot nichts Passendes für Probe-Dateien (nur `RepoRoot`/`CopyFixture`); Tests erzeugen in eigenen Temp-Ordnern.

tiers: D 5 / hard 0 · review default · cycles 1 · agents 9
