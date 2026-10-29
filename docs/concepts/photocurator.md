# PhotoCurator — Konzept

Status: **Draft** (Vision + Anforderungen, Stories für das MVP geschnitten, noch nichts gebaut).
Dieses Dokument legt fest, warum es PhotoCurator gibt, für wen, was das MVP leistet und was
bewusst nicht. Die technische Umsetzung steht in der
[Tech Spec](photocurator-tech-spec.md), die Arbeitspakete im [Backlog](../BACKLOG.md).

Es gibt noch kein `CLAUDE.md`; die Leitplanken dieses Konzepts und der Tech Spec gelten, bis
eins existiert.

---

## TL;DR

- **Vision:** Zwei Personen verschlagworten und verorten ihre Familienfotos auf dem UGREEN-NAS
  schnell und im Batch — und alles, was sie setzen, landet **in den Dateien** und ist in der
  UGREEN Fotos-App für Suche und bedingte Alben nutzbar.
- PhotoCurator ist ein **Werkzeug zum Abarbeiten**, kein Fotobetrachter. Angeschaut wird
  weiterhin in UGREEN Fotos.
- **Dateien sind die Wahrheit.** Tags, Kategorien, Ort und „Erledigt"-Status stehen in den
  Metadaten (ExifTool). Der Foto-Index ist ein Cache und jederzeit aus den Dateien neu baubar.
- **Größtes Risiko zuerst:** Ein Spike klärt, welche Felder UGREEN Fotos wirklich liest und
  wann es Änderungen erkennt. Daraus folgt die Schreibstrategie.
- **Sicheres Schreiben:** Änderungsjournal mit Rückgängig pro Batch + Kopie des Originals in
  einen Backup-Ordner außerhalb der Bibliothek, Fehler pro Datei statt Batch-Abbruch.
- **Ressourcenschonend auf dem NAS:** kleiner Go-Dienst, ExifTool im Dauerprozess,
  Thumbnails mit libvips/ffmpeg im Hintergrund mit begrenzter Parallelität.
- **MVP:** Batch-Tagging mit hierarchischen Kategorien, Batch-Standorte (Suche/Karte,
  Favoriten, Vorschläge aus zeitlich benachbarten Fotos), Arbeitslisten mit
  Erledigt-Markierung und Fortschrittsanzeige.
- **Offen:** Ergebnis des Spikes (Feldmatrix), Batch-Beschreibung im MVP ja/nein,
  Favoriten gemeinsam oder pro Person, genaue Grenzwerte (als Platzhalter markiert).

---

## 1. Vision

Die Fotobibliothek wächst automatisch: Handys laden in die Inbox, Phockup sortiert alle 15
Minuten nach `Sortiert/JJJJ/MM/`. Was fehlt, ist der Kontext — *wo* war das, *welcher Anlass*,
*wer*. UGREEN Fotos kann bedingte Alben und Suche, braucht dafür aber Metadaten, die heute
nicht in den Dateien stehen (WhatsApp-Bilder haben oft gar keine).

PhotoCurator soll sich anfühlen wie eine **Abarbeitungsliste**: „Zeig mir alles ohne Ort aus
dem Urlaub, markiere den Tag, Ort suchen, übernehmen, weiter." Wenige Klicks pro Batch, am
Handy auf dem Sofa genauso wie am PC. Der Fortschritt ist sichtbar, und was einmal erledigt
ist, taucht nicht wieder auf.

Die Arbeit ist **dauerhaft und app-unabhängig**: Wenn PhotoCurator morgen gelöscht wird,
bleiben alle Tags und Orte in den Dateien und in UGREEN Fotos erhalten.

## 2. Scope

### Zielgruppe

Zwei Personen (Nutzer und Partnerin) im Heimnetz, Browser am PC und Android-Handy. Keine
technischen Kenntnisse für die tägliche Nutzung nötig; Installation und Betrieb macht der
Nutzer selbst als Docker-Compose-Projekt in UGOS Pro.

### Problem

1. Tags/Kategorien und Orte lassen sich in UGREEN Fotos nicht effizient im Batch setzen (bzw.
   nicht dateibasiert und app-unabhängig).
2. Viele Fotos haben keinen Ort (Innenaufnahmen, WhatsApp, GPS aus) oder keinen Anlass.
3. Es ist nicht erkennbar, welche Fotos schon bearbeitet wurden und welche noch fehlen.

### In scope (v1 = MVP)

- **Batch-Tagging/Kategorisierung:** Mehrfach-, Bereichs-, Tages-/Monatsauswahl; Tags
  zuweisen und entfernen in einem Schritt; hierarchische Kategorien (`Anlass/Geburtstag`,
  `Reise/Österreich`, `Wir zwei`); zuletzt verwendete Tags, Autovervollständigung.
- **Batch-Standorte:** Ortssuche über OpenStreetMap/Nominatim inkl. POIs (Restaurants,
  Hotels), Klick auf Karte; gespeichert werden Koordinaten **und** lesbarer Ortsname;
  Lieblingsorte; Vorschlag des Orts zeitlich benachbarter Fotos mit einem Klick.
- **Arbeitslisten:** Filter „ohne Tags", „ohne Ort", „ohne beides", kombinierbar mit
  Zeitraum, Ordner und Dateityp; Markierung „erledigt / nicht relevant"; Fortschrittsanzeige.
- **Grundlagen:** Foto-Index über `Sortiert`, Thumbnails (JPEG, HEIC, PNG, MP4, MOV) mit Cache,
  mobiltaugliche Oberfläche, Anmeldung für zwei Personen, sicheres Schreiben mit Rückgängig,
  Deployment als Compose-Projekt.
- **Spike:** Welche Metadatenfelder liest UGREEN Fotos, wie erkennt es Änderungen.

### Bewusst nicht in v1

- Eigene Timeline- oder Kartenansicht zum Durchstöbern.
- KI-Vorschläge für Tags.
- Zugriff von unterwegs.
- Duplikaterkennung.
- Umbenennen/Zusammenführen von Tags, die schon in vielen Dateien stehen (massenhaftes
  Umschreiben).
- Aufnahmedatum korrigieren (z. B. für WhatsApp-Bilder ohne Datum).
- Videowiedergabe in PhotoCurator (Vorschaubild genügt; angeschaut wird in UGREEN Fotos).

> Begründung: Die vier erstgenannten Punkte hat der Auftraggeber explizit nach hinten
> gestellt. Tag-Umbenennen und Datumskorrektur schreiben massenhaft in Dateien und brauchen
> erst eine bewährte Schreibschicht; die Videowiedergabe dupliziert UGREEN Fotos.

### Non-goals (dauerhaft)

- **Kein Ersatz für UGREEN Fotos** als Betrachter oder Album-App. PhotoCurator pflegt
  Metadaten, UGREEN Fotos zeigt an.
- **Keine Änderung an UGOS oder an der UGREEN Fotos-App** (keine Datenbankzugriffe, keine
  Konfigurationsänderung, keine Plugins). Kommuniziert wird ausschließlich über die Dateien.
- **Keine Arbeit in der Inbox.** Die Inbox gehört Phockup.
- **Keine eigene, nicht aus Dateien rekonstruierbare Wahrheit über Fotos.** Foto-Metadaten
  leben in den Dateien; die App-Datenbank hält nur App-Zustand (siehe §5).
- **Kein Verschieben, Umbenennen oder Löschen von Fotos.** Ordnung ist Aufgabe von Phockup.

## 3. Getroffene Entscheidungen (aus dem Auftrag und der Rückfrage am 2026-10-29)

| Thema | Entscheidung | Begründung |
| --- | --- | --- |
| Source of Truth | Metadaten in den Dateien (ExifTool); Index ist Cache | Dauerhaft, app-unabhängig, UGREEN-tauglich (Auftrag) |
| Arbeitsbereich | Nur `/volume1/Gemeinsame Fotos/Sortiert/`, nie Inbox | Kein Konflikt mit Phockup (Auftrag) |
| Ziel-App | UGREEN Fotos (UGOS Pro „Photos") für Suche, bedingte Alben, Kategorisierung | Dort wird angeschaut (Auftrag) |
| Erstes Arbeitspaket | Spike zu den von UGREEN gelesenen Feldern | Wichtigstes Risiko (Auftrag) |
| Schreibstrategie im Zweifel | In mehrere kompatible Felder gleichzeitig schreiben | Auftrag |
| Backup/Rückgängig | Änderungsjournal (Rückgängig pro Batch) + einmalige Kopie des Originals in Backup-Ordner außerhalb der Bibliothek mit Aufbewahrungsfrist | Rückfrage: Schutz gegen beschädigte Dateien *und* bequemes Rückgängig |
| Änderungsdatum vs. Erkennung | Wenn UGREEN Änderungen nur über das Änderungsdatum erkennt, gewinnt die Erkennung; das Aufnahmedatum bleibt in EXIF | Rückfrage |
| „Erledigt"-Markierung | Eigenes, verstecktes XMP-Feld in der Datei | Rückfrage: überlebt Index-Neuaufbau, erscheint nicht als Tag |
| Technologie | Nach Performance und Ressourcenverbrauch auf dem NAS gewählt (Details: Tech Spec) | Rückfrage: an Claude delegiert mit diesem Kriterium |
| Nutzer | Zwei Konten, nur Heimnetz | Auftrag |
| Ortsdienst | Nominatim mit max. 1 Anfrage/s, eigenem User-Agent, Cache | Nutzungsrichtlinie (Auftrag) |
| Sprache | Doku Deutsch, Code und Kommentare Englisch | Auftrag |

## 4. Technologieentscheidungen

| Bereich | Wahl | Begründung |
| --- | --- | --- |
| Backend | Go, ein statisches Binary | Geringster RAM-/CPU-Bedarf im Leerlauf, kein Interpreter, kleines Image |
| Metadaten | ExifTool im `-stay_open`-Modus | De-facto-Standard, schreibt alle Kandidatenfelder inkl. HEIC und MP4/MOV; ein Dauerprozess statt Perl-Start pro Datei |
| Thumbnails | libvips (`vipsthumbnail`, inkl. HEIC über libheif) + ffmpeg für Videos | libvips ist deutlich speichersparender als ImageMagick/Pillow; Prozess pro Datei gibt Speicher sofort frei |
| Datenbank | SQLite (WAL), zwei Dateien: Index-Cache und App-Zustand | Kein separater DB-Server |
| Frontend | Svelte + Vite, statisch gebaut, vom Go-Binary ausgeliefert | Sehr kleines JS-Bundle, schnell am Handy, kein Node zur Laufzeit |
| Karte | Leaflet + OSM-Kacheln | Leichtgewichtig, OSM-konform mit Attribution |
| Tests | Go-Tests mit echtem ExifTool im Docker-Test-Stage, Vitest, Playwright | Gleiche native Werkzeuge wie in Produktion |

## 5. Begriffe & Modell

- **Bibliothek:** alle unterstützten Dateien unter `Sortiert/` (rekursiv).
- **Tag:** ein Begriff mit optionalem Pfad. `Wir zwei` ist ein Tag ohne Eltern,
  `Anlass/Geburtstag` ein Tag mit Elternkategorie `Anlass`. **Tags und Kategorien sind
  dasselbe Konzept** — eine Kategorie ist ein Tag, der Kinder hat.
- **Ort:** Koordinaten (Breite/Länge) + lesbarer Name, zerlegt in Teile
  (z. B. Sublocation „Ritzenhofer", City „Saalfelden", State „Salzburg", Country „Österreich").
- **Prüfstatus:** `offen` (Standard), `erledigt`, `nicht relevant` — verstecktes XMP-Feld.
- **Arbeitsliste:** gespeicherter oder Ad-hoc-Filter über die Bibliothek.
- **Job:** eine Batch-Änderung (z. B. „Tag X auf 240 Fotos"), mit Ergebnis pro Datei und
  Rückgängig.

Datenhaltung:

```
Dateien in Sortiert/  ──(ExifTool lesen)──►  index.db  (Cache, jederzeit löschbar)
        ▲                                        │
        │ (ExifTool schreiben, Backup, Journal)  ▼
     Writer-Job  ◄──────────  Web-UI  ──────►  app.db (App-Zustand: Nutzer-Sitzungen,
                                                leere Kategorien, Lieblingsorte, zuletzt
                                                verwendete Tags, Journal, Nominatim-Cache)
```

Der Auftrag fordert, dass „eine lokale Datenbank nur Index/Cache ist und jederzeit aus den
Dateien neu aufbaubar". Das gilt uneingeschränkt für alles, was **ein Foto beschreibt**
(`index.db`). App-Zustand, der kein Foto beschreibt (Lieblingsorte, Journal, Ortscache), kann
naturgemäß nicht aus Fotos rekonstruiert werden und liegt getrennt in `app.db`; sein Verlust
kostet Komfort, aber keine Foto-Metadaten.

## 6. Arbeitsablauf (Hauptpfad)

1. Arbeitsliste öffnen, z. B. „ohne Ort", Zeitraum Juli 2025.
2. Tagesgruppe „12. Juli" komplett auswählen.
3. „Ort setzen" → „Ritzenhofer Saalfelden" suchen → Treffer wählen → übernehmen.
4. Job läuft im Hintergrund, Fortschritt sichtbar; die Fotos verschwinden aus der Liste.
5. Gleiche Auswahl → Tag `Reise/Österreich` zuweisen.
6. Rest des Tages ist irrelevant → „nicht relevant".
7. Fortschritt: „1.240 von 3.500 Fotos haben einen Ort".
8. UGREEN Fotos übernimmt die Änderungen (automatisch oder nach Rescan — laut Spike).

## 7. Risiken

| # | Risiko | Auswirkung | Gegenmaßnahme |
| --- | --- | --- | --- |
| R1 | UGREEN Fotos liest Keywords/Kategorien gar nicht oder nur eingeschränkt | Kernnutzen (bedingte Alben) fällt weg | Spike als erste Story; vor Schreib-Stories auswerten; Ausweichfelder testen (Beschreibung, Ort); Entscheidung durch Auftraggeber, falls nichts greift |
| R2 | UGREEN erkennt Änderungen nur per manuellem Rescan oder sehr verzögert | Frust, „hat nicht funktioniert" | Spike misst Verhalten; UI zeigt nach Jobs den passenden Hinweis |
| R3 | ExifTool beschädigt eine Datei (v. a. HEIC/MOV) | Datenverlust | Original-Backup vor erstem Schreiben, Read-back-Prüfung, Prüfsumme der Bilddaten vor/nach |
| R4 | Schreiben ändert Besitzer/Rechte der Datei auf dem Share | UGOS-Rechte kaputt, UGREEN kann Datei nicht lesen | In-place-Schreiben (Inode bleibt), Container läuft mit UID/GID des Share-Nutzers |
| R5 | Konflikt mit Phockup oder UGREEN-Indexer | Halbe Dateien, doppelte Verarbeitung | Nur `Sortiert`; Ruhezeit für frisch geänderte Dateien; ein Schreibjob gleichzeitig |
| R6 | NAS-Last (Thumbnails von HEIC/Video bei Erstscan) | NAS träge, UGREEN-Indexer langsamer | Begrenzte Parallelität, niedrige Priorität, Ressourcenlimits im Compose |
| R7 | HEIC-Dekodierung nicht im Container verfügbar (libheif/HEVC) | Keine Thumbnails für Handyfotos | Im Projektgerüst prüfen (Test mit HEIC-Fixture) |
| R8 | Nominatim-Sperre wegen Richtlinienverstoß | Ortssuche fällt aus | Serverseitiger Limiter 1/s, User-Agent, Cache, **keine** Suche bei jedem Tastendruck |
| R9 | Videos: UGREEN liest eingebettete Metadaten anders als bei Fotos | Videos ohne Tags/Ort in UGREEN | Spike testet eingebettet und Sidecar getrennt |
| R10 | UGREEN Fotos schreibt selbst Metadaten in Dateien (z. B. beim Taggen in der App) | Gegenseitiges Überschreiben | Im Spike prüfen; falls ja, Regel festlegen |

## 8. Annahmen

| # | Annahme | Wird geprüft in |
| --- | --- | --- |
| A1 | UGREEN Fotos indexiert `Sortiert/` (bzw. den übergeordneten Ordner) | Spike (001) |
| A2 | Phockup verändert Dateien nach dem Verschieben nach `Sortiert` nicht mehr | Spike-Protokoll / Betrieb |
| A3 | Der Docker-Container kann mit einer UID/GID laufen, die Schreibrechte auf `Sortiert` hat, ohne UGOS-Rechte zu verändern | Deployment (003) |
| A4 | Ein Backup-Ordner außerhalb der von UGREEN Fotos indexierten Ordner ist verfügbar (Vorschlag: `/volume1/docker/photocurator/backup`) | Deployment (003) |
| A5 | Bibliotheksgröße im Bereich einiger Tausend bis ca. 50.000 Dateien | Härtung (021) |
| A6 | Das Heimnetz gilt als vertrauenswürdig; HTTP ohne TLS ist für den Zugriff akzeptabel | Offener Punkt 4 |
| A7 | WhatsApp-Bilder ohne EXIF-Datum tragen das Datum im Dateinamen (`IMG-JJJJMMTT-WA…`) | Index (005) |

## 9. Einbindung in die Umgebung

- **UGOS Pro:** Compose-Projekt; Volumes: `Sortiert` (rw), App-Daten (rw), Backup (rw). Kein
  Zugriff auf Inbox, Docker-Socket oder UGOS-Systemordner.
- **Phockup:** unabhängig; PhotoCurator ignoriert Dateien, deren Änderungszeit jünger als
  eine Ruhezeit ist, und liest neue Dateien beim nächsten Scan.
- **UGREEN Fotos:** liest die Dateien; PhotoCurator gibt nach Jobs den im Spike ermittelten
  Hinweis (z. B. „Änderungen erscheinen nach dem nächsten Index-Lauf").
- **Nominatim/OSM:** ausgehende HTTPS-Anfragen vom Container (Suche) bzw. Browser
  (Kartenkacheln).

## 10. Anforderungen

**Spike**
- PC-1 Ermittelt und dokumentiert, welche Keyword-, Hierarchie-, Beschreibungs- und Ortsfelder
  UGREEN Fotos je Format (JPEG, HEIC, PNG, MP4, MOV, Sidecar) in Suche, bedingten Alben und
  Kategorisierung verwendet.
- PC-2 Ermittelt, wie und wann UGREEN Fotos Änderungen an bereits indexierten Dateien erkennt.
- PC-3 Leitet daraus eine Schreibstrategie (Felder je Format) ab.

**Tagging**
- PC-10 Tags im Batch zuweisen und entfernen, für Einzel-, Bereichs-, Tages- und Monatsauswahl.
- PC-11 Hierarchische Kategorien anlegen und als Baum anzeigen.
- PC-12 Zuletzt verwendete Tags und Autovervollständigung.
- PC-13 Fremde, bereits vorhandene Keywords bleiben erhalten.

**Orte**
- PC-20 Ortssuche (inkl. POIs) über Nominatim; Klick auf Karte.
- PC-21 Speichert Koordinaten und lesbaren Ortsnamen in die Dateien.
- PC-22 Lieblingsorte speichern und wiederverwenden.
- PC-23 Vorschlag des Orts zeitlich benachbarter Fotos, Übernahme mit einem Klick.
- PC-24 Nominatim-Richtlinien: max. 1 Anfrage/s, eigener User-Agent, Cache.

**Arbeitslisten**
- PC-30 Filter ohne Tags / ohne Ort / ohne beides, kombinierbar mit Zeitraum, Ordner, Dateityp.
- PC-31 Markierung „erledigt" / „nicht relevant", in der Datei gespeichert.
- PC-32 Fortschrittsanzeige.

**Schreiben & Daten**
- PC-40 Metadaten werden mit ExifTool in die Dateien geschrieben; der Index ist aus den Dateien
  vollständig wieder aufbaubar.
- PC-41 Originale werden nicht beschädigt (Prüfung der Bilddaten), Backup vor erstem Schreiben.
- PC-42 Fehler pro Datei protokollieren, Batch läuft weiter.
- PC-43 Rückgängig pro Job.
- PC-44 Keine Dateien außerhalb von `Sortiert` (und dem eigenen Daten-/Backup-Ordner) werden
  verändert; keine Hilfsdateien im Fotoordner.

**Betrieb & UI**
- PC-50 Compose-Projekt auf UGOS Pro, Fotoordner als Volume.
- PC-51 Mobiltaugliche Oberfläche, flüssig bei mehreren Tausend Fotos.
- PC-52 Thumbnails mit Cache für JPEG, PNG, HEIC, MP4, MOV.
- PC-53 Anmeldung für zwei Nutzer, nur Heimnetz.
- PC-54 Ressourcenschonend: begrenzte Parallelität und Speicherlimits.

## 11. Offene Punkte

1. **Ergebnis des Spikes** (Feldmatrix, Erkennungsverhalten) — wird nach Sprint S01 auf dem
   NAS ermittelt und in `photocurator-metadaten-strategie.md` festgehalten. Blockiert das
   Refinement der Schreib-Stories (009 ff.), nicht Sprint S02.
2. **Batch-Beschreibung** (`dc:description` usw.) im MVP? Der Auftrag nennt Beschreibung als
   zu schreibendes Feld, aber nicht als Kern-Feature. Derzeit als Nach-MVP-Story vorgemerkt.
3. **Lieblingsorte und „zuletzt verwendet"**: gemeinsam für beide Personen oder pro Person?
   (Story 013/016.)
4. **HTTP im Heimnetz** ausreichend oder HTTPS über den UGOS-Reverse-Proxy?
5. **Platzhalter-Werte** (Ruhezeit 5 min, Scanintervall 15 min, Backup-Aufbewahrung 30 Tage,
   Zeitfenster Ortsvorschlag ±3 h, Speicherlimit Container) — Startwerte, in den Stories als
   Platzhalter markiert und konfigurierbar.
6. **Backup-Ordner:** genauer Pfad auf dem NAS und verfügbarer Speicherplatz.
