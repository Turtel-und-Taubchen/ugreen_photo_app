# PhotoCurator auf dem UGREEN-NAS installieren

Diese Anleitung beschreibt die Installation per UGOS Pro Docker. Gemountet wird nur der Ordner `Sortiert` der Fotobibliothek, niemals die Inbox.

## Ordner anlegen

Lege per SSH oder Dateimanager die Ordner für Daten und Backup an:

- `/volume1/docker/photocurator/data` (Datenbanken und Thumbnails)
- `/volume1/docker/photocurator/backup` (Sicherungen vor Schreibzugriffen)

Das Backup liegt bewusst außerhalb der Ordner, die UGREEN Fotos indiziert, damit Sicherungskopien nicht als Fotos erscheinen.

## Nutzer und UID ermitteln

Lege in UGOS einen Nutzer an (oder nimm einen vorhandenen), der Schreibrecht auf `Gemeinsame Fotos` hat. Ermittle UID und GID per SSH:

```
id <name>
```

Die Werte `uid` und `gid` werden als `PUID` und `PGID` eingetragen.

## Rechte

Der Nutzer braucht Schreibrecht auf `Gemeinsame Fotos/Sortiert` sowie auf die Ordner `data` und `backup`. Prüfe sie mit `ls -ln` und passe sie bei Bedarf mit `chown -R <uid>:<gid> /volume1/docker/photocurator` an. Die Inbox wird nicht eingebunden und braucht keine Rechte für die App.

## .env anlegen

Kopiere `.env.example` nach `.env` und trage die Werte ein:

- `PUID`, `PGID`: UID und GID des Nutzers aus dem vorigen Schritt
- `TZ`: Zeitzone, z. B. `Europe/Vienna`
- `NOMINATIM_CONTACT`: Kontaktadresse für die Nominatim-Nutzungsrichtlinie (ohne Kontakt ist die Ortssuche deaktiviert)
- `PHOTOS_PATH`: Pfad zu `Sortiert` (nur dieser Ordner, nie die Inbox)
- `DATA_PATH`: Datenordner
- `BACKUP_PATH`: Backupordner
- `PORT`: Port im Heimnetz (Standard 8470)
- `BIND_IP`: optionale Bindung an die LAN-IP des NAS
- `PC_IMAGE`: Image-Name und Tag (Standard `photocurator:latest`)
- `PC_SCAN_INTERVAL` (optional, Standard `15m`, mindestens `1m`): Abstand zwischen zwei automatischen Bibliotheks-Scans; bei Bedarf als Eintrag unter `environment:` der Compose-Datei setzen, nicht in `.env`
- `PC_SCAN_QUIET` (optional, Standard `5m`, ebenfalls unter `environment:` der Compose-Datei): Ruhezeit; Dateien, die jünger als diese Zeit geändert wurden, werden erst im nächsten Scan erfasst. Das schützt vor halb verschobenen Phockup-Dateien.
- `PC_PRESERVE_MTIME` (optional, Standard `true`, erlaubt `true`/`false`, ebenfalls unter `environment:` der Compose-Datei): `true` lässt die Änderungszeit einer Datei beim Metadaten-Schreiben unverändert. Im Spike verschwand eine Datei mit geänderter mtime aus dem UGREEN-Index, mit unveränderter mtime wurde der neue Speicherort übernommen. `false` nur setzen, wenn du das bewusst in Kauf nimmst; beim Start erscheint dann eine Warnung.

Beim Start prüft die App, dass der Fotopfad nicht `Inbox` enthält.

Die App legt den Bibliotheksindex als `index.db` im Datenordner an. Löschen von `index.db` und ein Neustart bauen den Index neu auf.

## Nutzer anlegen

Die App kennt nur die Nutzer aus `users.conf` im Datenordner (`/volume1/docker/photocurator/data/users.conf`, im Container `/data/users.conf`). Ohne diese Datei startet sie nicht. Pro Nutzer eine Zeile `name:hash`; Zeilen mit `#` und Leerzeilen werden ignoriert, Namen unterscheiden Groß- und Kleinschreibung.

Erzeuge den Hash (das Passwort wird von der Eingabe gelesen und erscheint nicht in der Ausgabe):

```
docker run --rm -i photocurator:latest hash-password
```

Trage die ausgegebene Zeile mit dem Namen davor in `users.conf` ein, z. B. `anna:$argon2id$v=19$...`. Klartext-Passwörter lehnt die App beim Start ab. Starte danach den Container neu, damit die Datei neu gelesen wird.

## Image übertragen

Das Compose-Projekt baut nichts selbst. Baue das Image auf dem Entwicklungsrechner und übertrage es:

```
docker build --target runtime -t photocurator:latest .
docker save photocurator:latest | ssh <nutzer>@<NAS-IP> docker load
```

## Projekt anlegen

In UGOS Pro: Docker, Projekt, neues Projekt. Füge den Inhalt von `docker-compose.yml` ein und hinterlege die Werte aus `.env` (Umgebungsdatei im selben Ordner). Starte das Projekt.

## Erster Aufruf

Öffne `http://<NAS-IP>:<PORT>` im Heimnetz. Die App ist nur für das Heimnetz gedacht; richte keine Portweiterleitung im Router ein. Startfehler (z. B. nicht beschreibbarer Ordner, Inbox im Pfad) erscheinen im Container-Log.

## Datei aus dem Backup wiederherstellen

Vor dem ersten Schreiben einer Datei legt die App eine Kopie unter `/volume1/docker/photocurator/backup/<JJJJ-MM-TT>/<relativer Pfad>` ab (im Container `/backup`; der relative Pfad entspricht dem Pfad innerhalb von `Sortiert`). Zum Wiederherstellen kopierst du die Sicherung per SSH über die Datei in `Sortiert`:

```
cp --preserve=timestamps /volume1/docker/photocurator/backup/<JJJJ-MM-TT>/<relativer Pfad> "<PHOTOS_PATH>/<relativer Pfad>"
```

`--preserve=timestamps` ist Pflicht: Eine Datei mit neuem Änderungsdatum verschwindet aus UGREEN Fotos, deshalb muss die ursprüngliche Änderungszeit erhalten bleiben. Klicke danach in der App auf „Neu einlesen“, damit der Index die wiederhergestellte Datei übernimmt.

- Aufbewahrung: Backups werden `PC_BACKUP_RETENTION_DAYS` Tage aufbewahrt (Standard 30). Ältere Backups löscht die App täglich. Die Variable steht nicht in `.env.example`; setze sie nur dann als Eintrag unter `environment:` der Compose-Datei, wenn der Standard geändert werden soll.
- Pro Datei und Aufbewahrungszeitraum wird nur einmal gesichert; das Backup enthält den Zustand vor dem ersten Schreiben, nicht jeden Zwischenstand.
- Der Ordner `.inflight/` im Backupordner enthält temporäre Daten zur Wiederherstellung nach einem Absturz. Er ist kein Backup, stelle nichts daraus von Hand wieder her.

## App-Datenbank sichern

Die App-Datenbank (`app.db`, u. a. Jobs und Einstellungen) läuft im WAL-Modus; eine Kopie im laufenden Betrieb kann inkonsistent sein. So sicherst du sie:

1. Stoppe den Container (Projekt in UGOS stoppen).
2. Kopiere `app.db`, `app.db-wal` und `app.db-shm` aus `/volume1/docker/photocurator/data` an einen sicheren Ort (nicht alle drei Dateien sind immer vorhanden; kopiere, was da ist).
3. Starte den Container wieder.

`index.db` und die Thumbnails musst du nicht sichern, sie entstehen beim Scannen neu.

## Update

Baue und übertrage das neue Image wie unter "Image übertragen", dann starte das Projekt in UGOS neu (Projekt stoppen und starten). Daten und Backup bleiben erhalten.
