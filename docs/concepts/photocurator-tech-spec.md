# PhotoCurator — Tech Spec

Status: **Draft** (vor Implementierung). Ergänzt das [Konzept](photocurator.md) um Architektur,
Datenmodell, Metadaten-Strategie, Deployment und Sicherheit. Die endgültige Feldauswahl folgt
aus dem Spike (Story 001) und wird in `photocurator-metadaten-strategie.md` festgehalten; diese
Spec beschreibt den Rahmen und die vorläufige Strategie.

Leitkriterium für alle technischen Entscheidungen: **Performance und geringer
Ressourcenverbrauch auf dem NAS** (Pentium Gold 8505, 8 GB RAM, geteilt mit UGOS, Phockup und
dem UGREEN-Indexer).

---

## 1. Architektur

```
 Browser (PC / Android)
   │  HTTP (Heimnetz), Session-Cookie
   ▼
┌─────────────────────────── Container „photocurator" ───────────────────────────┐
│  Go-Binary (ein Prozess)                                                       │
│   ├─ HTTP-Server: REST-API + statisches Svelte-Frontend (go:embed)             │
│   ├─ Auth: Sessions, Login-Rate-Limit                                          │
│   ├─ Scanner: Dateisystem-Walk, Delta über (Pfad, Größe, mtime)                │
│   ├─ ExifTool: Lesen je Block, 1 Schreibprozess `-stay_open`                    │
│   ├─ Thumbnailer: Queue, N Worker → vipsthumbnail / ffmpeg (nice 10)           │
│   ├─ Writer: serielle Job-Queue → Backup → ExifTool → Read-back → Index         │
│   ├─ Geocoder: Nominatim-Proxy, globaler Limiter 1/s, Cache                    │
│   └─ Scheduler: periodischer Scan, Backup-Aufräumen                            │
│  SQLite: /data/index.db (Cache) · /data/app.db (App-Zustand)                   │
└────────────────────────────────────────────────────────────────────────────────┘
   │ rw                         │ rw                        │ rw
 /photos  = …/Sortiert       /data (DB, Thumbnail-Cache)   /backup (Original-Kopien)
```

**Ein Prozess, ein Container.** Kein Redis, kein separater DB-Server, keine Node-Laufzeit.
ExifTool (Perl) schreibt über einen Dauerprozess, weil ein Perl-Start pro Datei auf dem Pentium
~150–300 ms kostet. Lesen: ein ExifTool-Aufruf je Block, kein Dauerprozess.

### Komponenten (Go-Pakete, vorläufig)

| Paket | Aufgabe |
| --- | --- |
| `cmd/photocurator` | Einstieg: `serve`, `hash-password`, `probe generate|mutate` (Spike), `reindex` |
| `internal/config` | Konfiguration aus Umgebungsvariablen, Validierung der Pfade |
| `internal/auth` | Nutzer, Passwort-Hashes, Sessions, Rate-Limit |
| `internal/library` | Scanner, Datums-Ermittlung, Index-Abfragen, Filter |
| `internal/exif` | ExifTool-Prozessverwaltung, Lesen (`-json`), Schreiben (Argumentdateien) |
| `internal/metadata` | **Feldmapping**: Domänenmodell (Tags, Ort, Prüfstatus) ↔ ExifTool-Tags je Format |
| `internal/writer` | Jobs, Backup, Journal, Read-back, Rückgängig |
| `internal/thumbs` | Thumbnail-Queue und -Cache |
| `internal/geo` | Nominatim-Client, Limiter, Cache, Ortsvorschläge |
| `internal/api` | HTTP-Handler |
| `web/` | Svelte-Frontend |

## 2. Datenmodell

### 2.1 `index.db` — Cache, jederzeit löschbar

```sql
CREATE TABLE photos (
  id            INTEGER PRIMARY KEY,
  rel_path      TEXT NOT NULL UNIQUE,      -- relative to /photos, '/' separated
  folder        TEXT NOT NULL,             -- e.g. '2025/07'
  media_type    TEXT NOT NULL,             -- 'image' | 'video'
  format        TEXT NOT NULL,             -- 'jpeg' | 'heic' | 'png' | 'mp4' | 'mov'
  size          INTEGER NOT NULL,
  mtime_ns      INTEGER NOT NULL,
  taken_at      TEXT,                      -- ISO 8601, local time as stored in file
  taken_source  TEXT NOT NULL,             -- 'exif' | 'quicktime' | 'filename' | 'mtime'
  gps_lat       REAL, gps_lon REAL,
  loc_sublocation TEXT, loc_city TEXT, loc_state TEXT, loc_country TEXT,
  loc_label     TEXT,                      -- human readable name as written
  review_state  TEXT NOT NULL DEFAULT 'open', -- 'open' | 'done' | 'irrelevant'
  read_error    TEXT,                      -- last ExifTool read error, NULL if fine
  scanned_at    TEXT NOT NULL
);
CREATE TABLE photo_tags (
  photo_id INTEGER NOT NULL REFERENCES photos(id) ON DELETE CASCADE,
  tag_path TEXT NOT NULL,                  -- 'Anlass/Geburtstag'
  PRIMARY KEY (photo_id, tag_path)
);
CREATE INDEX photos_taken ON photos(taken_at DESC, id DESC);
CREATE INDEX photos_folder ON photos(folder);
CREATE INDEX photo_tags_tag ON photo_tags(tag_path);
```

Arbeitslisten sind reine Abfragen (`NOT EXISTS photo_tags`, `gps_lat IS NULL`,
`review_state = 'open'`). Paginierung per Keyset `(taken_at, id)`, nie `OFFSET`.

### 2.2 `app.db` — App-Zustand (sichern)

| Tabelle | Inhalt |
| --- | --- |
| `sessions` | Session-ID (Hash), Nutzer, Ablauf |
| `tag_definitions` | Kategorien ohne Fotos (leere Knoten); Tags mit Fotos kommen aus dem Index |
| `recent_tags` | zuletzt verwendete Tags (Nutzer oder global — offener Punkt) |
| `favorite_places` | Name, Koordinaten, Ortsteile |
| `jobs` | Job-ID, Nutzer, Art, Parameter, Status, Zähler, Zeitstempel |
| `job_items` | Job, Datei, Ergebnis (`ok`/`skipped`/`failed`), Fehlertext, alte/neue Werte (JSON) — **das Journal** |
| `backups` | Datei, Backup-Pfad, Zeitpunkt (für Aufbewahrung und „schon gesichert?") |
| `geocode_cache` | Anfrage-Schlüssel, Antwort-JSON, Zeitpunkt |
| `dismissed_suggestions` | abgelehnte Ortsvorschläge pro Foto |

Nutzerkonten selbst stehen **nicht** in der DB, sondern in der Konfiguration (2 Nutzer,
Passwort-Hashes) — keine Nutzerverwaltung nötig.

### 2.3 Domänenmodell ↔ Datei

| Domäne | Bedeutung |
| --- | --- |
| `tags: []string` | Pfade mit `/` als Trenner im UI, z. B. `Anlass/Geburtstag` |
| `location` | `lat`, `lon`, `label`, `sublocation`, `city`, `state`, `country`, `countryCode` |
| `reviewState` | `open` / `done` / `irrelevant` |

## 3. Metadaten-Strategie

### 3.1 Kandidatenfelder (werden im Spike geprüft)

| Zweck | Kandidaten |
| --- | --- |
| Keywords | `XMP-dc:Subject`, `IPTC:Keywords`, `EXIF:XPKeywords` (nur JPEG/TIFF-artige), QuickTime `Keys:Keywords` / `UserData:Keywords` (Video) |
| Hierarchie | `XMP-lr:HierarchicalSubject` (`Anlass|Geburtstag`) |
| Beschreibung | `XMP-dc:Description`, `EXIF:ImageDescription`, `IPTC:Caption-Abstract`, QuickTime `Keys:Description` |
| Ort (Koordinaten) | `EXIF:GPSLatitude/Ref`, `GPSLongitude/Ref` (Bilder), `Keys:GPSCoordinates`, `UserData:GPSCoordinates` (Video, ISO 6709), `XMP-exif:GPSLatitude/Longitude` |
| Ort (Text) | `XMP-photoshop:City/State/Country`, `XMP-iptcCore:Location` (Sublocation), `XMP-iptcExt:LocationShown`, `IPTC:City/Sub-location/Province-State/Country-PrimaryLocationName` |
| Prüfstatus | eigener Namespace `XMP-photocurator:ReviewState` (+ `ReviewedAt`) über ExifTool-Konfigurationsdatei |
| Sidecar | `<name>.xmp` und `<name>.<ext>.xmp` (nur Videos, nur falls der Spike Bedarf zeigt) |

### 3.2 Vorläufige Schreibstrategie (bis zum Spike-Ergebnis)

„Im Zweifel mehrfach schreiben":

- **Tags, Bilder:** `XMP-dc:Subject` = alle Blätter **und** Elternbegriffe (Lightroom-Konvention:
  `Anlass`, `Geburtstag`); `XMP-lr:HierarchicalSubject` = vollständige Pfade
  (`Anlass|Geburtstag`); `IPTC:Keywords` = wie `dc:Subject` (nur JPEG; mit
  `IPTC:CodedCharacterSet=UTF8`, max. 64 Byte je Eintrag); `EXIF:XPKeywords` = wie
  `dc:Subject`, `;`-getrennt (nur JPEG).
- **Tags, Videos:** `XMP-dc:Subject` + `XMP-lr:HierarchicalSubject` eingebettet;
  QuickTime-Keywords nur, falls der Spike zeigt, dass UGREEN sie liest.
- **Ort:** GPS-Koordinaten (Bilder EXIF, Videos `Keys:GPSCoordinates` + `UserData:GPSCoordinates`)
  + Textfelder in XMP-photoshop/iptcCore und (JPEG) IPTC. `loc_label` wird zusätzlich als
  `XMP-iptcExt:LocationShownLocationName` geschrieben.
- **Prüfstatus:** nur `XMP-photocurator:*`, nie in Keyword-Feldern.
- **Bestehende fremde Werte bleiben erhalten:** Schreiben erfolgt als gezieltes Hinzufügen
  (`-XMP-dc:Subject+=…`) bzw. Entfernen (`-=`), nie als Überschreiben ganzer Listen.
- **Lesen:** Tags werden aus allen Keyword-Feldern zusammengeführt; Hierarchie hat Vorrang
  (`HierarchicalSubject` → Pfade, übrige Feldwerte, die nicht bereits als Teil eines Pfades
  vorkommen, als Tags ohne Eltern).

Das Mapping ist **eine Tabelle in `internal/metadata`** (Format × Domänenfeld → ExifTool-Tags),
sodass das Spike-Ergebnis an genau einer Stelle eingespielt wird.

### 3.3 Sicheres Schreiben

Ablauf pro Datei in einem Job:

1. Vorbedingungen: Pfad liegt unter `/photos`, Datei existiert, `mtime` älter als Ruhezeit
   (Platzhalter 5 min), Datei steht im Index mit unveränderter Größe/mtime (sonst erst neu
   lesen).
2. **Backup**, falls in der Aufbewahrungsfrist noch keine Kopie existiert:
   `/backup/<JJJJ-MM-TT>/<rel_path>` (Kopie, dann `fsync`). Schlägt das fehl → Datei `failed`,
   keine Änderung.
3. Alte Werte der betroffenen Felder lesen → Journal (`job_items.old`).
4. **Bilddaten-Prüfsumme** vorher: ExifTool `ImageDataHash` (SHA-256).
5. Schreiben mit ExifTool: `-overwrite_original_in_place` (Inode, Besitzer und Rechte bleiben;
   keine `_original`-Dateien), `-P` nur wenn `PRESERVE_MTIME=true` (Standard `false`,
   endgültig nach Spike), `-config photocurator.config`, Werte über Argumentdatei
   (keine Shell, kein Quoting-Risiko).
6. **Read-back:** Felder erneut lesen, mit Sollwerten vergleichen; `ImageDataHash` muss
   gleich sein. Abweichung → `failed` + automatische Wiederherstellung aus dem Backup.
7. Index-Zeile aktualisieren, `job_items.new` speichern.

Ein Fehler betrifft immer nur die eine Datei; der Job läuft weiter. Es läuft **genau ein
Schreibjob** gleichzeitig (Queue). Ein Neustart während eines Jobs markiert ihn als
abgebrochen; bereits verarbeitete Dateien sind im Journal.

**Rückgängig** schreibt die alten Werte aus dem Journal zurück — nur für Dateien, deren
aktuelle Werte noch den neuen Werten des Jobs entsprechen; andere werden übersprungen und
gemeldet. **Wiederherstellen** einer Datei aus `/backup` ist ein dokumentierter manueller
Schritt (Betriebshandbuch). Backups älter als die Aufbewahrung (Platzhalter 30 Tage) werden
täglich gelöscht.

### 3.4 Konflikte mit Phockup und UGREEN

- Phockup schreibt nur durch Verschieben aus der Inbox nach `Sortiert`. Dateien unter der
  Ruhezeit werden weder gelesen noch geschrieben.
- UGREEN-Indexer: Schreiben in-place ist kurz; ob UGREEN per mtime, Inotify oder periodisch
  erkennt, liefert der Spike. Standard: mtime ändert sich (Entscheidung „Erkennung gewinnt").
- PhotoCurator legt **keine Dateien** in `/photos` an (keine Thumbnails, keine Sidecars — außer
  der Spike erzwingt Video-Sidecars; dann gesondert entscheiden).

## 4. Thumbnails

- Größen: `s` 320 px (Raster), `l` 1600 px (Detail), WebP, Qualität ~75.
- Bilder: `vipsthumbnail <file> --size 320x320 --export-profile srgb -o <cache>.webp[Q=75]`
  (berücksichtigt EXIF-Orientierung; HEIC über libheif).
- Videos: `ffmpeg -ss 1 -i <file> -frames:v 1` (bei kürzeren Videos Frame 0) → vipsthumbnail.
- Cache: `/data/thumbs/<xx>/<sha1(rel_path|size|mtime)>-<size>.webp` — Änderungen an der Datei
  invalidieren automatisch; verwaiste Dateien räumt ein Aufräumlauf weg.
- Queue mit Priorität: sichtbare Thumbnails zuerst; Hintergrund-Erzeugung nach dem Scan.
  Worker-Zahl Platzhalter 1, Prozesse mit `nice -n 10`.
- Auslieferung: `Cache-Control: private, max-age=31536000, immutable`, URL enthält den
  Cache-Schlüssel.

## 5. Scan & Index

- Walk über `/photos`, Endungen `jpg jpeg heic heif png mp4 mov` (Groß/Klein egal), versteckte
  Ordner und `@eaDir`-artige Systemordner ignoriert.
- Delta: neue oder geänderte (Größe/mtime) Dateien werden in Blöcken von ~200 an den
  ExifTool-Aufruf gegeben (`-json -n -G1 -a`, kein `-fast2` (Videos mit moov hinter mdat verlieren sonst Datum und GPS), + gezielte Tag-Liste).
- Gelöschte Dateien werden aus dem Index entfernt.
- Aufnahmedatum: `DateTimeOriginal` → `CreateDate` (QuickTime, UTC-Korrektur) → Dateiname
  (`IMG-JJJJMMTT-WA…`, `VID-…`, `JJJJMMTT_hhmmss`) → mtime; die Quelle wird gespeichert.
- Auslöser: Start, periodisch (Platzhalter 15 min), Knopf „Neu einlesen", nach jedem Job für
  die betroffenen Dateien.
- `photocurator reindex` bzw. Löschen von `index.db` baut alles neu aus den Dateien auf.

## 6. API (Auszug)

| Methode | Pfad | Zweck |
| --- | --- | --- |
| GET | `/api/health` | Lebenszeichen + Version (offen, für den Container-Healthcheck) |
| POST | `/api/login` | Anmeldung: `{username, password}` → `{username}` + Session-Cookie; Fehlschlag immer 401 mit identischer Meldung; gesperrte IP → 429 + `Retry-After` |
| POST | `/api/logout` | Session serverseitig löschen, Cookie mit `Max-Age` ≤ 0 → 204 |
| GET | `/api/me` | Angemeldeter Nutzer: `{username}` |
| GET | `/api/photos?filter…&after=<cursor>` | Liste mit Keyset-Cursor |
| GET | `/api/photos/groups?filter…` | Tages-/Monatsgruppen mit Anzahl |
| GET | `/api/photos/{id}` | Details |
| GET | `/api/thumbs/{id}/{s|l}` | Thumbnail |
| GET | `/api/tags` | Tag-Baum mit Anzahl |
| POST | `/api/jobs` | Batch-Job anlegen: `{selection, action, payload}` |
| GET | `/api/jobs`, `/api/jobs/{id}` | Status, Ergebnis pro Datei |
| POST | `/api/jobs/{id}/undo` | Rückgängig |
| GET | `/api/geo/search?q=`, `/api/geo/reverse?lat&lon` | Nominatim-Proxy |
| GET/POST/DELETE | `/api/places` | Lieblingsorte |
| GET | `/api/stats?filter…` | Fortschritt |
| POST | `/api/scan` | Neu einlesen |

**Default-deny:** Jede Route liegt hinter der Session-Prüfung. Ohne gültige Session offen sind
nur `POST /api/login`, `GET /api/health`, `GET /login` (liefert die SPA-Hülle) und `GET` auf
Dateien, die im eingebetteten Frontend wirklich existieren — außer `index.html`, die
App-Hülle unter `/` ist geschützt. Der Abgleich ist exakt auf den gesendeten Pfad; nicht
kanonische Pfade (`..`, `//`, `.`, abschließender `/`, abweichendes Percent-Encoding) sind nie
offen. Ohne Session: `/api/*` → 401 JSON, Seiten-`GET`/`HEAD` → 302 `/login`, andere Methoden
auf Seitenpfaden → 401 JSON (kein Redirect, damit kein Client einen Schreibversuch als `GET`
wiederholt). Angemeldet: `GET /login` → 302 `/`. Danach prüft die CSRF-Schicht: schreibende
Methoden (nicht `GET`/`HEAD`/`OPTIONS`) unter `/api/` ohne `X-Requested-With` → 403, auch für
`/api/login`; ohne Session gewinnt die 401.

**Auswahl** wird als `{ filter, include_ids, exclude_ids }` übertragen, damit „ganzer Monat"
oder „alle Treffer" ohne Tausende IDs funktioniert; der Server löst sie beim Job-Start in eine
feste Dateiliste auf.

## 7. Frontend

- Svelte + Vite, TypeScript, statisch gebaut, im Go-Binary eingebettet.
- Virtualisiertes Raster (nur sichtbare Zeilen im DOM), gruppiert nach Monat/Tag.
- Auswahlmodell: Tippen (Einzeln), Shift+Klick bzw. „Bereich bis hier" am Handy, Gruppenkopf
  „alle auswählen".
- Leaflet für Karten, nur im Ort-Dialog geladen (Code-Splitting).
- Responsiv ab 360 px Breite, Bedienelemente ≥ 44 px Touch-Fläche.

## 8. Deployment

`deploy/docker-compose.yml` (Auszug, Werte sind Startwerte):

```yaml
services:
  photocurator:
    image: ${PC_IMAGE:-photocurator:latest}
    container_name: photocurator
    restart: unless-stopped
    user: "${PUID}:${PGID}"
    ports:
      - "${BIND_IP:-0.0.0.0}:${PORT:-8470}:8080"
    environment:
      TZ: ${TZ}
      PC_NOMINATIM_CONTACT: ${NOMINATIM_CONTACT}
      PC_PHOTOS_HOST_PATH: ${PHOTOS_PATH}
      PC_PRESERVE_MTIME: "false"
    volumes:
      - ${PHOTOS_PATH}:/photos
      - ${DATA_PATH}:/data
      - ${BACKUP_PATH}:/backup
    read_only: true
    tmpfs: [/tmp]
    cap_drop: [ALL]
    security_opt: ["no-new-privileges:true"]
    mem_limit: 768m
    cpus: 2.0
    healthcheck:
      test: ["CMD", "/app/photocurator", "healthcheck"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 20s
```

- Image: Multi-Stage — `web` (Node, baut Frontend) → `build` (Go) → `test`/`lint` (Go +
  ExifTool + libvips + ffmpeg, für die Verify-Befehle) → `runtime` (Debian slim mit
  `libimage-exiftool-perl`, `libvips-tools`, `libheif`-Plugins, `ffmpeg`, `tini`).
- `PUID/PGID` = UGOS-Nutzer mit Schreibrecht auf `Gemeinsame Fotos`.
- Die Zeitzone (`TZ`) ist ein Platzhalter und gehört in `.env`.
- Kein `build:` im Compose-Projekt: Das Image wird auf dem Entwicklungsrechner gebaut und per `docker save`/`docker load` auf das NAS übertragen (siehe `deploy/README.md`).
- Der Healthcheck ruft `photocurator healthcheck` auf (Intervall 30 s, Timeout 5 s, 3 Versuche, Startphase 20 s).
- `PC_PHOTOS_HOST_PATH` trägt den Host-Pfad von `PHOTOS_PATH`; die App nutzt ihn beim Start für den Inbox-Schutz (Host-Pfad darf keine `Inbox` enthalten). Gemountet wird nur `Sortiert`, nie die Inbox.
- Die App prüft beim Start: `/photos` existiert, ist beschreibbar, Pfad enthält nicht
  `Inbox`; `/backup` und `/data` beschreibbar; sonst Abbruch mit klarer Meldung.
- Ressourcen-Zielwerte (Platzhalter, gemessen in Story 021): Leerlauf < 150 MB RAM, Scan
  < 500 MB, Container-Limit 768 MB.

## 9. Sicherheit

| Thema | Maßnahme |
| --- | --- |
| Anmeldung | Nutzer in `users.conf` (Pfad `PC_USERS_FILE`, Standard `/data/users.conf`): eine Zeile `name:<argon2id-PHC>` pro Nutzer, `#`-Kommentare und Leerzeilen erlaubt, Namen case-sensitiv; Klartext wird beim Start abgewiesen; Hash erzeugen mit `photocurator hash-password`. Unbekannte Namen werden gegen einen Dummy-Hash geprüft, sodass jeder Versuch genau eine Argon2-Prüfung kostet; Passwörter werden nie geloggt. Eine Session, deren Nutzer nicht mehr in `users.conf` steht, ist ungültig |
| Session | Cookie `pc_session`: `HttpOnly`, `SameSite=Strict`, `Path=/`, `Max-Age` = 30 Tage (gleitend, Erneuerung höchstens einmal pro Tag); **ohne `Secure`**, weil die App per HTTP im Heimnetz läuft. Zufällige 256-bit-ID, in `app.db` nur als SHA-256 gespeichert |
| Zugriff | Default-deny für alle Routen, Ausnahmen siehe §6 |
| Brute Force | Rate-Limit pro IP aus der TCP-Gegenstelle (`X-Forwarded-For` wird ignoriert): 5 Fehlversuche in 60 s → 60 s Sperre, auch mit richtigem Passwort 429 + `Retry-After`; Zähler nur im Speicher |
| CSRF | `SameSite=Strict` + alle schreibenden Requests nur mit JSON und Header `X-Requested-With` |
| Pfade | Alle Dateizugriffe über ID → `rel_path` aus dem Index; Auflösung mit `filepath.Clean` + Prüfung „unter /photos"; Symlinks nicht folgen |
| ExifTool-Injection | Argumentdateien (`-@`), Werte zeilenweise, Zeilenumbrüche in Werten abgewiesen; kein Shell-Aufruf |
| Container | Nicht-root, read-only rootfs, `cap_drop: ALL`, `no-new-privileges`, kein Docker-Socket, keine Inbox, Ressourcenlimits |
| Netzwerk | Nur Heimnetz (Port nicht im Router freigeben); optional Bindung an LAN-IP; Entscheidung: HTTP im Heimnetz, vorerst kein HTTPS und kein Reverse-Proxy; Port niemals im Router freigeben |
| Ausgehend | Nur Nominatim (Server) und OSM-Kacheln (Browser) |
| Datenschutz | Keine Telemetrie; Nominatim erhält nur Suchbegriffe bzw. Koordinaten |

## 10. Nominatim & Karten

- Serverseitiger Proxy, **globaler** Token-Bucket 1 Anfrage/s (alle Nutzer zusammen).
- `User-Agent: PhotoCurator/<version> (<NOMINATIM_CONTACT>)`.
- Suche nur bei Absenden (Enter/Knopf), **keine** Autovervollständigung gegen Nominatim
  (Richtlinie verbietet Autocomplete).
- Cache in `app.db` (Platzhalter 90 Tage), Schlüssel = normalisierte Anfrage + Sprache.
- Parameter: `format=jsonv2&addressdetails=1&namedetails=1&accept-language=de&limit=10`.
- Ortsname: `<name>, <city|town|village>`; Ortsteile aus `address`.
- Kartenkacheln: `tile.openstreetmap.org` mit Attribution; geringe Last bei zwei Nutzern.

## 11. Teststrategie

| Ebene | Werkzeug | Inhalt |
| --- | --- | --- |
| Unit | `go test` | Mapping, Datumsermittlung, Filter, Limiter, Auswahlauflösung |
| Integration | `go test` im Docker-Stage `test` mit echtem ExifTool/libvips/ffmpeg | Schreiben/Lesen je Format an Fixture-Kopien, Bilddaten-Hash, Rückgängig, Index-Neuaufbau |
| Frontend | Vitest | Auswahlmodell, Komponentenlogik |
| E2E | Playwright gegen `docker compose -p photocurator-e2e` mit Fixture-Bibliothek (temporäre Kopie) und gemocktem Nominatim | Alle Nutzer-Akzeptanzkriterien |
| Manuell (Rest) | UGREEN Fotos auf dem echten NAS | Nur, was die proprietäre App betrifft |

Fixtures: kleine, selbst erzeugte Dateien (keine privaten Fotos) in `testdata/`, je Format,
mit und ohne Metadaten, plus WhatsApp-artige Datei ohne EXIF.
