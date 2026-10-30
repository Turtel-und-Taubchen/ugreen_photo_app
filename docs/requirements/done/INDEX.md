# Story history

One line per finished story, appended by `/build` when it moves the file here.
Format: `- NNN — <title> · <sprint or —> · <one-sentence result>`.

The full texts live next to this file. This index is the fast overview — do not turn it into a
second roadmap.
- 002 — Projektgerüst · S01 · Go-Backend, Svelte-Frontend, Docker-Stages und E2E-Harness laufen mit den Profil-Verify-Befehlen
- 001 — Spike — welche Metadatenfelder liest UGREEN Fotos? · S01 · probe generate/mutate erzeugt Testdateien und PROTOKOLL.md, Strategie-Dokument mit leerer Feldmatrix; NAS-Durchführung (AC12) manuell
- 003 — Deployment als Docker-Compose-Projekt auf UGOS Pro · S01 · Compose/.env/README, Pfad-Startprüfung, healthcheck und E2E gegen Deploy-Compose laufen; NAS-Installation (AC7) manuell
- 004 — Einfache Anmeldung für zwei Nutzer · S01 · Argon2id-Login mit Sessions, Limiter, Default-deny und CSRF-Schutz, Login-UI und E2E-Flow
- 005 — Bibliotheks-Index aus den Dateien in Sortiert · S02 · Scanner + index.db aus den Dateien, Scheduler, /api/scan, Scan-Anzeige mit E2E
- 006 — Thumbnails mit Cache für Fotos und Videos · S02 · WebP-Thumbnails per vips/ffmpeg mit Prioritäts-Queue, immutable Cache-Header und Vorschaukacheln.
- 007 — Fotoraster mit Gruppierung, Ordnernavigation und Detailansicht · S02 · virtualisiertes Raster nach Monat/Tag, Zeitraum-Navigation, Detailansicht mit Blättern und Lese-API
- 008 — Mehrfach-, Bereichs- und Gruppenauswahl · S02 · serverseitige Auflösung (Filter + ein-/ausgeschlossene IDs, /api/selection/count), Auswahlmodus im Raster mit Shift+Klick, Monats-/Tagesauswahl und langem Drücken
- 011 — Hierarchische Kategorien definieren und als Baum anzeigen · S02 · Tag-Pfade validieren, Baum mit Zählung, /api/tags und Seite „Tags“ mit Anlegen/Löschen-Sperre
- 014 — Ortssuche und Reverse-Geocoding über Nominatim · S02 · Suche/Reverse über /api/geo/* mit 1-s-Limiter und 90-Tage-Cache, Seite /orte mit PlacePicker, E2E gegen Nominatim-Mock
- 009 — Sicherer Metadaten-Writer mit Batch-Jobs · S03 · ExifTool-Writer, eine Mapping-Tabelle, Pipeline mit Vorab-Kopie/Read-back/Restore in place, FIFO-Jobs mit Fortschritt, /api/jobs (tags.add) und Job-Status im Header
- 010 — Backup der Originale, Änderungsjournal und Rückgängig pro Job · S03 · Backup mit Frist, Journal (Read-back-Werte), Undo-Job mit Konfliktprüfung, Seite /jobs, Betriebshandbuch
