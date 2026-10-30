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
- `NOMINATIM_CONTACT`: Kontaktadresse für die Nominatim-Nutzungsrichtlinie
- `PHOTOS_PATH`: Pfad zu `Sortiert` (nur dieser Ordner, nie die Inbox)
- `DATA_PATH`: Datenordner
- `BACKUP_PATH`: Backupordner
- `PORT`: Port im Heimnetz (Standard 8470)
- `BIND_IP`: optionale Bindung an die LAN-IP des NAS
- `PC_IMAGE`: Image-Name und Tag (Standard `photocurator:latest`)
- `PC_SCAN_INTERVAL` (optional, Standard `15m`, mindestens `1m`): Abstand zwischen zwei automatischen Bibliotheks-Scans; bei Bedarf als Eintrag unter `environment:` der Compose-Datei setzen, nicht in `.env`
- `PC_SCAN_QUIET` (optional, Standard `5m`, ebenfalls unter `environment:` der Compose-Datei): Ruhezeit; Dateien, die jünger als diese Zeit geändert wurden, werden erst im nächsten Scan erfasst. Das schützt vor halb verschobenen Phockup-Dateien.

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

## Update

Baue und übertrage das neue Image wie unter "Image übertragen", dann starte das Projekt in UGOS neu (Projekt stoppen und starten). Daten und Backup bleiben erhalten.
