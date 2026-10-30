---
id: 015
title: Standort im Batch setzen — per Suche oder Karte
status: done # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E4 Standorte
estimate: 5
sprint: S03
---

## Requirement

Für die ausgewählten Fotos setze ich einen Ort — gefunden per Suche oder per Klick auf eine
Karte. Gespeichert werden Koordinaten und lesbarer Ortsname in den Dateien, sodass UGREEN Fotos
den Ort anzeigt und für Orts-Alben nutzen kann.

## Acceptance Criteria

- [x] **AC1** — Der Dialog „Ort setzen" zeigt eine Karte (OSM mit Attribution); ein Klick setzt
  einen Marker, ein Suchtreffer (Story 014) zentriert die Karte und setzt den Marker.
- [x] **AC2** — „Übernehmen" schreibt für alle ausgewählten Fotos Koordinaten, Ortsname und
  Ortsteile gemäß Schreibstrategie als einen Job (Fotos und Videos).
- [x] **AC3** — Haben ausgewählte Fotos bereits einen Ort, fragt der Dialog vorher: „alle
  überschreiben" oder „nur Fotos ohne Ort"; die Wahl wirkt wie angegeben.
- [x] **AC4** — „Ort entfernen" löscht Koordinaten und Ortsfelder aus den ausgewählten Dateien.
- [x] **AC5** — Die Detailansicht zeigt Ortsname und eine kleine Karte mit Marker.
- [x] **AC6** — Die Kartenbibliothek wird erst beim Öffnen des Dialogs geladen; das
  Haupt-Bundle der Startseite enthält sie nicht.

## Open Questions

- ~~**Refinement S03 (2026-10-30) — BLOCKIERT: Spike-Ergebnis fehlt (gleiche Ursache wie 009).**
  Frage an den Nutzer: Wie geht es mit 015 weiter? (A)/(C) aus 009 warten bzw. verschieben ·
  (B) vorläufige Strategie · (D) 015 teilen (schreibfreier Teil in S03, Dialog + Schreiben
  später).~~ answered → Decisions (Sprint)

## Decisions (Sprint)

- **(User)** Spike abwarten: Ergebnis liegt vor, 015 bleibt vollständig in S03
- **Spike-Folgen für 015:** In der App wirkt nur der GPS-Punkt (Bilder `EXIF:GPS*`, Videos
  `Keys:`/`UserData:GPSCoordinates`); Ortsname und Ortsteile werden laut Strategie-Dokument
  trotzdem als Zusatzinformation geschrieben — AC2 bleibt damit wörtlich erfüllbar, und
  PhotoCurators eigener Index und Lieblingsorte (016) brauchen den Namen.
- **Feldagnostisch:** 015 nennt keine ExifTool-Felder, sondern schreibt und löscht das
  Domänenfeld `location` ausschließlich über die Mapping-Tabelle aus 009 (`internal/metadata`)
  — das Spike-Ergebnis fließt an genau einer Stelle ein (009 AC1).
- **Schnittstellen aus 009/010/012 werden im Build nachgeprüft:** 009 wird parallel refined;
  Pfade (Aktions-Registrierung, Pipeline, `web/src/lib/jobs.ts`, Fortschrittsanzeige, E2E-Muster)
  folgen der Tech Spec §3.3/§6 und dem 010-Plan (`internal/writer`) — **`/build` gleicht sie gegen
  den finalen 009-Plan und den Code von 012 ab**, das vor 015 gebaut wird.
- **Zwei Job-Aktionen `location.set` / `location.remove`** über `POST /api/jobs {selection,
  action, payload}` (Tech Spec §6) — wie `tags.add`/`tags.remove` aus 012, damit Auswahlauflösung,
  Queue, Backup, Journal und Rückgängig (010) ohne eigenen Code gelten.
- **Modus als Payload (`mode: "all" | "missing"`), entschieden je Datei auf den frisch gelesenen
  Werten:** „hat schon einen Ort" heißt „Koordinaten vorhanden" — dieselbe Definition wie
  `gps_lat IS NOT NULL` der Arbeitsliste „ohne Ort" (Tech Spec §2.1, 018), aber gegen die Datei
  geprüft, damit eine noch nicht eingelesene Fremdänderung nicht überschrieben wird.
- **Übersprungene Dateien werden nicht angefasst** (kein Backup, kein ExifTool, mtime/Inhalt
  gleich) und als `skipped` mit Grund gemeldet: „hat bereits einen Ort" bzw. „kein Ort
  vorhanden" — analog zu 012 AC4.
- **„Ort entfernen" löscht alle `location`-Felder des Mappings** (auch Textfelder ohne
  Koordinaten); übersprungen wird nur eine Datei, in der keines dieser Felder einen Wert hat.
- **Rückfrage aus dem Index, Wirkung aus der Datei:** der Dialog zählt Fotos mit Ort über die
  erweiterte `POST /api/selection/count` (`{count, located}`) — eine Abfrage statt eines neuen
  Endpunkts; die verbindliche Entscheidung trifft der Job (vorige Entscheidung).
- **Gemeinsame Validierung `internal/geo/validate.go`:** Breite/Länge endlich und im Bereich,
  Ortsname getrimmt 1–200 Zeichen, Ortsteile ≤ 200, keine Steuerzeichen — die Werte landen über
  ExifTool-Argumentdateien in Dateien (Tech Spec §9); 016 (Lieblingsorte) hat dieselben Regeln und
  soll beim Build diese Funktionen nutzen statt einer zweiten Kopie.
- **Leaflet aus npm, nur per dynamischem `import()`:** ein Modul `web/src/lib/map/leaflet.ts`
  kapselt Leaflet samt CSS und wird nur dynamisch geladen — Tech Spec §7 und AC6; kein CDN, damit
  außer den Kacheln nichts von außen geladen wird (Tech Spec §9 „Ausgehend").
- **Eine Kartenkomponente `PlaceMap.svelte` für Picker und Detail:** interaktiv mit `onpick`,
  ohne `onpick` nur Anzeige — keine zweite Kartenkopie für AC5.
- **Detailkarte ebenfalls lazy:** sie lädt dasselbe Modul erst, wenn ein Foto mit Koordinaten
  geöffnet wird — die Detailansicht liegt auf der Startseite, ein statischer Import brächte
  Leaflet ins Haupt-Bundle (AC6).
- **Marker als `L.divIcon`** (CSS-Kreis, Klasse `pc-marker`) statt Leaflets Bild-Icon — dessen
  Bildpfade brechen beim Vite-Bundling, und der Marker bleibt ohne Zusatz-Assets prüfbar.
- **Kacheln:** `https://tile.openstreetmap.org/{z}/{x}/{y}.png` mit sichtbarer Attribution
  „© OpenStreetMap-Mitwirkende" — Tech Spec §10.
- **Kartenklick setzt nur den Punkt:** Marker + Breite/Länge (6 Nachkommastellen) im
  `PlacePicker`, kein automatisches Reverse-Geocoding; „Ort zum Punkt ermitteln" (014) bleibt der
  explizite Auslöser — keine Nominatim-Anfrage ohne Nutzeraktion, ein getippter Name wird nie
  überschrieben.
- **Ein Zustand treibt Karte und Felder:** die Karte folgt dem `point` des Pickers (aus den
  Feldern Breite/Länge), Kartenklick füllt die Felder; Suchtreffer und Lieblingsorte (016) setzen
  den Punkt über eine gemeinsame `apply({lat, lon, name, parts})` — so erscheinen 016s Liste und
  Marker im Dialog, ohne dass der Dialog davon weiß.
- **Zentrieren nur bei Punkten von außen:** stammt der Punkt nicht aus dem eigenen Kartenklick
  (Suchtreffer, Lieblingsort, getippte Koordinaten), zentriert die Karte auf Zoom 15; ein Klick
  setzt nur den Marker — sonst springt die Karte unter dem Finger.
- **Der Dialog bettet den `PlacePicker` ein** (Entscheidung aus 014), der die Karte enthält —
  damit hat auch `/orte` die Karte (Nebeneffekt, ohne eigenen Code).
- **„Übernehmen" erst mit Punkt und nicht leerem Ortsnamen:** die Anforderung verlangt
  Koordinaten **und** lesbaren Namen in den Dateien.
- **„Ort entfernen" mit Bestätigung** („Ort aus n Fotos entfernen?") — ein Fehltipp am Handy
  träfe sonst sofort viele Dateien; Rückgängig (010) bleibt zusätzlich.
- **Detail-API liefert `lat`/`lon` in `location`:** `GET /api/photos/{id}` gibt heute nur
  Textfelder aus, der Marker in AC5 braucht die Koordinaten, die der Index schon hat.
- **E2E ohne echte Kacheln:** ein Helfer `web/tests/e2e/helpers/map.ts` beantwortet
  `tile.openstreetmap.org` per `page.route` mit einem festen leeren PNG — „nie den echten Dienst in
  Tests" gilt sinngemäß für OSM; Ortssuche über den Nominatim-Mock aus 014. Ein Helfer statt
  Kopien in drei Specs (+ 016).
- **AC6 per E2E gegen das echte Build:** geprüft werden die Skripte, die der Browser wirklich
  lädt, auf die Leaflet-Signatur `leafletjs.com` (Attribution-Präfix im Leaflet-Code) — ein Vitest
  auf `web/dist` hinge von der Reihenfolge der Verify-Befehle ab; eigener Code enthält die
  Zeichenkette nicht.
- **E2E schreibt nur auf Dateien ohne Ort und hinterlässt sie ohne Ort** (`sample.jpg`,
  `short.mov`), `tagged.jpg` wird nur als „hat schon einen Ort" übersprungen — die
  Fixture-Bibliothek wird von allen Specs geteilt (wie 012).
- **Keine neuen E2E-Flows für Picker und Detail:** die Karte im Picker ist ein Schritt des
  bestehenden `places.spec.ts`-Flows, die Detailkarte einer von `photo-detail.spec.ts`; neu ist nur
  der Flow „Ort setzen" (ein neuer Nutzerpfad).
- **Kein `CLAUDE.md` im Repo** (geprüft 2026-10-30) und kein Systems-Dokument (`docs/systems` leer)
  — Leitplanken aus Tech Spec und Profil; die Tech Spec bekommt die geänderten §6/§7-Zeilen.

## Plan

Baut auf 009 (Writer, Job-Queue, Mapping `location`, `POST /api/jobs`, Fortschritt), 010
(Backup/Journal, Löschen von Ort im Mapping) und 012 (Muster der Job-Aktionen und Dialoge, vorher
gebaut). Reihenfolge:

1. **D1 — Job-Aktionen `location.set`/`location.remove`** (Backend): Validierung in
   `internal/geo/validate.go`, Aktion je Datei auf frisch gelesenen Werten (Modus, Skip-Regeln),
   Schreiben/Löschen über Mapping-Tabelle; Integrationstests je Format mit echtem ExifTool.
2. **D2 — API-Ergänzungen:** `POST /api/selection/count` → `{count, located}`;
   `GET /api/photos/{id}` → `location.lat/lon`; Tech Spec §6.
3. **D3 — Karte im `PlacePicker`:** Leaflet-Abhängigkeit, Lazy-Modul, `PlaceMap.svelte`, `apply()`
   im Picker, E2E-Kartenhelfer; Schritte im bestehenden `/orte`-Flow; Tech Spec §7.
4. **D4 — Dialog „Ort setzen" + „Ort entfernen":** Knöpfe in der `SelectionBar`, Rückfrage,
   Jobs anlegen; neuer E2E-Flow.
5. **D5 — Detailkarte:** `PhotoDetail.svelte` mit Ortsname + nicht-interaktiver `PlaceMap`;
   Schritt im bestehenden Detail-Flow.

D1 → D2 unabhängig voneinander; D3 vor D4 und D5; D4 braucht D1–D3, D5 braucht D2/D3.
Nicht Teil: Lieblingsorte (016), Ortsvorschlag (017), Arbeitsliste „ohne Ort" (018).

## Deliverables

- [x] **D1 — Job-Aktionen „Ort setzen" und „Ort entfernen" (Backend).**
  Dateien: `internal/geo/validate.go` (neu), `internal/geo/validate_test.go` (neu), die Datei der
  Ort-Aktion im Writer-Paket aus 009 (vorgesehen `internal/writer/location.go`, neu) und
  `internal/writer/location_test.go` (neu), die Stelle, an der 012 `tags.add`/`tags.remove`
  registriert und deren Payload beim Job-Anlegen prüft (Muster: genau diese Aktionen; Pfade aus
  dem Code von 009/012 übernehmen, nicht neu erfinden).
  - `validate.go`: `ValidatePoint(lat, lon float64) error` (endlich, −90…90 / −180…180) und
    `ValidateLocationText(name string, parts Parts) (string, Parts, error)` (Name getrimmt, 1–200
    Runen; Ortsteile je getrimmt ≤ 200 Runen; kein `unicode.IsControl`; `CountryCode` groß) mit
    `*LocationInputError{Msg}` (deutsche Meldung). `Parts` = vorhandenes `geo.Parts`.
  - Aktion `location.set`, Payload `{location:{lat,lon,label,sublocation,city,state,country,
    countryCode}, mode:"all"|"missing"}`: beim Job-Anlegen validieren (Fehler → 400 mit `Msg`,
    kein Job; unbekannter `mode` → 400). Je Datei auf den in der Pipeline frisch gelesenen Werten:
    `mode=missing` und Datei hat Koordinaten → `skipped` „hat bereits einen Ort" ohne Backup und
    ohne ExifTool-Schreiben; sonst Domänenfeld `location` über die Mapping-Tabelle aus 009 setzen
    (alle Felder ersetzen, leere Ortsteile löschen das jeweilige Feld).
  - Aktion `location.remove`, Payload `{}`: alle `location`-Felder des Mappings löschen (das
    Löschen hat 010 D4 im Mapping ergänzt; fehlt es, dort aus der einen Tabelle ableiten); Datei
    ohne Wert in irgendeinem dieser Felder → `skipped` „kein Ort vorhanden", unberührt.
  - Journal (`location` alt/neu) und Read-back liefert die Pipeline aus 009/010 — kein eigener Code.
  Tests (echtes ExifTool, Fixture-Kopien wie in den 009-Tests):
  `internal/writer/location_test.go` › `TestSetLocationWritesMappedFieldsForEveryFormat`
  (JPEG, HEIC, PNG, MP4, MOV in **einem** Job; Read-back: Koordinaten innerhalb der
  Mapping-Toleranz, Name und Ortsteile; gesetzte Felder = Mapping-Felder für `location` je Format),
  `TestSetLocationModeMissingSkipsLocatedFilesAndModeAllOverwrites` (Datei mit/ohne GPS; bei
  `missing` bleibt die GPS-Datei byte- und mtime-gleich und ist `skipped`, bei `all` trägt sie den
  neuen Ort), `TestRemoveLocationClearsCoordinatesAndPlaceFieldsAndSkipsFilesWithout` (danach kein
  `location`-Feld mehr lesbar, auch nicht bei Videos; Datei ohne Ort `skipped` und unverändert);
  `internal/geo/validate_test.go` › `TestLocationInputWithBadCoordinatesNamesOrControlCharsIsRefused`.

- [x] **D2 — API: Anzahl mit Ort in der Auswahl, Koordinaten in der Detailansicht.**
  Dateien: `internal/api/selection.go` (Antwort `{"count": n, "located": k}`; `located` = Fotos
  der aufgelösten Auswahl mit `gps_lat IS NOT NULL`), die Zählfunktion in `internal/library`
  neben `CountSelection` (eine Abfrage mit beiden Zählern, z. B. `COUNT(*)`, `COUNT(gps_lat)`;
  `CountSelection` darauf umstellen statt zweiter Auflösung), `internal/library/query.go`
  (Detail-Abfrage um `gps_lat, gps_lon`), `internal/api/photos.go` (`placeJSON` um `lat`, `lon`
  als `*float64`, JSON `null` ohne Koordinaten), Tests in `internal/api/selection_test.go` und
  `internal/api/photos_test.go` (vorhandene Dateien bzw. Muster dort), Tech Spec §6 (Zeile
  `POST /api/selection/count` → `{count, located}` ergänzen). Bestehende Aufrufer von `count`
  (`web/src/lib/selectionStore.svelte.ts`) bleiben unverändert gültig.
  Tests: `internal/api/selection_test.go` › `TestSelectionCountReportsPhotosWithLocation`
  (Auswahl mit `tagged.jpg` + `sample.jpg` → `count 2, located 1`);
  `internal/api/photos_test.go` › `TestPhotoDetailIncludesCoordinates` (Foto mit GPS → `lat`/`lon`
  gesetzt, ohne → `null`).

- [x] **D3 — Karte im `PlacePicker` (lazy geladen) + E2E-Kartenhelfer.**
  Dateien: `web/package.json`/`package-lock.json` (`leaflet` ^1.9, dev `@types/leaflet`),
  `web/src/lib/map/leaflet.ts` (neu; **einzige** Stelle mit `import 'leaflet'` und
  `import 'leaflet/dist/leaflet.css'`, exportiert z. B. `createMap(el, opts)`; nur per
  `await import('./map/leaflet')` erreichbar — kein statischer Import, auch kein `import type`, der
  Werte zieht), `web/src/lib/map/PlaceMap.svelte` (neu), `web/src/lib/PlacePicker.svelte`,
  `web/tests/e2e/helpers/map.ts` (neu), `web/tests/e2e/places.spec.ts`,
  `docs/concepts/photocurator-tech-spec.md` (§7: „Leaflet nur per dynamischem Import — Dialog
  ‚Ort setzen', Seite `/orte`, Detailkarte").
  - `PlaceMap.svelte` (Svelte 5): Props `point: {lat, lon} | null`, `onpick?: (p) => void`,
    `height = 240`. Lädt in `onMount` das Modul dynamisch, Kacheln
    `https://tile.openstreetmap.org/{z}/{x}/{y}.png`, Attribution „© OpenStreetMap-Mitwirkende"
    (Link auf openstreetmap.org/copyright), Container mit `role="region"` und `aria-label="Karte"`.
    Marker = `L.divIcon({className: 'pc-marker'})` (CSS-Kreis ≥ 20 px). Ohne Punkt: Ansicht
    Mitteleuropa (47.5, 13.5, Zoom 6). Mit `onpick`: Klick → `onpick({lat, lon})` auf 6
    Nachkommastellen gerundet, Marker an die Stelle, **kein** Zentrieren. Punktwechsel von außen
    (≠ letzter eigener Klick) → Marker + `setView(point, 15)`. Ohne `onpick`: keine
    Klickbehandlung, nur Marker (für D5). `onDestroy` → `map.remove()`.
  - `PlacePicker.svelte`: `choose(place)` zu `apply({lat, lon, name, parts})` umbauen (setzt
    `lat`/`lon` als `String(…)`, `name`, `parts`, ruft `emit()`); Suchtreffer rufen
    `apply({lat, lon, name: place.label, parts: place.parts})`. Die Karte sitzt im Abschnitt
    „Gewählter Ort" über den Koordinatenfeldern, `<PlaceMap point={point} onpick={…}>`; der
    Klick setzt nur `lat`/`lon` und ruft `emit()` (Name/Ortsteile bleiben, kein `/api/geo/*`).
    Die Karte ist **nicht** von `enabled` abhängig. Oberhalb der Suche bleibt Platz für den
    Abschnitt „Lieblingsorte" aus 016 (nichts dafür vorbauen).
  - `helpers/map.ts`: `stubTiles(page)` (`page.route('https://tile.openstreetmap.org/**')` →
    festes 1×1-PNG, gibt Zähler zurück) und `watchLeaflet(page)` (sammelt JS-Antworten, liefert
    `() => Promise<boolean>`: enthält ein geladenes Skript `leafletjs.com`).
  E2E: den bestehenden Test in `web/tests/e2e/places.spec.ts` umbenennen in „place search finds a
  POI, names it, shows it on the map and survives an outage" und erweitern: vor dem ersten
  `goto('/orte')` die Startseite `/` laden → `watchLeaflet` false; auf `/orte` Karte mit
  Attribution sichtbar, `watchLeaflet` true, Kachel-Zähler > 0; nach Wahl des Suchtreffers ist
  `.pc-marker` sichtbar und die Karte auf den Treffer zentriert (Marker nahe Kartenmitte); Klick auf
  eine andere Kartenstelle → Breite/Länge ändern sich, „Ortsname" bleibt „Mein Hotel" (bzw. der
  zuletzt getippte Name), Mock-Protokoll ohne neue Anfrage.

- [x] **D4 — Dialog „Ort setzen" und „Ort entfernen" an der Auswahl + E2E-Flow.**
  Dateien: `web/src/lib/SetPlaceDialog.svelte` (neu), `web/src/lib/SelectionBar.svelte` (Knöpfe
  „Ort setzen", „Ort entfernen" neben den Tag-Knöpfen aus 012, ≥ 44 px), die Dialog-Hülle der
  012-Dialoge (wiederverwenden; existiert die Form dort zweimal ohne gemeinsame Hülle, zuerst nach
  `web/src/lib/Dialog.svelte` extrahieren), `web/src/lib/jobs.ts` (Job anlegen aus 009/012
  nutzen, ggf. typisierte Payloads ergänzen), Zählaufruf `POST /api/selection/count` mit
  `toRequest()` aus `web/src/lib/selection.ts` (vorhandene Funktion im Store erweitern statt
  zweiter Kopie), `web/tests/e2e/set-place.spec.ts` (neu).
  - Dialog „Ort setzen" (Überschrift genau so): Geo-Status wie `web/src/routes/Places.svelte`
    laden, `<PlacePicker enabled={…} onchange={…}>` einbetten. „Übernehmen" aktiv nur mit Punkt
    und nicht leerem Ortsnamen. Beim Tippen: `{count, located}` holen; `located > 0` → im Dialog
    „k von n ausgewählten Fotos haben bereits einen Ort." mit „Alle überschreiben" /
    „Nur Fotos ohne Ort" / „Abbrechen"; sonst direkt. Dann `POST /api/jobs {selection:
    toRequest(…), action: 'location.set', payload: {location: {lat, lon, label: name, …parts},
    mode}}` → Dialog schließt, Fortschritt/Ergebnis über die Job-Anzeige aus 009. Fehler →
    `role="alert"` mit Server-Meldung über `errorMessage` aus `web/src/lib/api.ts` (von 010
    extrahiert), Dialog bleibt offen. „Abbrechen"/Schließen schreibt nichts.
  - „Ort entfernen": Bestätigung „Ort aus n Fotos entfernen?" (n = Auswahlanzahl) mit
    „Entfernen"/„Abbrechen" → Job `location.remove`.
  E2E `web/tests/e2e/set-place.spec.ts` › „places are set for a selection from search or map,
  asked before overwriting and removed again" (Muster: `places.spec.ts` für Mock-Reset/-Protokoll,
  `helpers/auth.ts`, `helpers/grid.ts`/`library.ts`, Auswahl wie `selection.spec.ts`, Warten auf
  Job-Ende und Read-back per `compose exec -T photocurator exiftool …` wie im E2E-Flow von 012;
  `stubTiles`/`watchLeaflet` aus D3; `test.setTimeout(300_000)`). Schritte:
  1. Startseite geladen, `watchLeaflet` false; `sample.jpg`, `short.mov`, `tagged.jpg` auswählen,
     „Ort setzen" → Karte mit Attribution sichtbar, `watchLeaflet` true.
  2. „Ritzenhofer Saalfelden" + Enter, ersten Treffer wählen → `.pc-marker` sichtbar;
     „Übernehmen" → Rückfrage „1 von 3 …" → „Nur Fotos ohne Ort" → Job fertig: Detail von
     `sample.jpg` und `short.mov` zeigt „Ritzenhofer, Saalfelden", `tagged.jpg` weiter „Stadtpark,
     Graz"; Read-back: `sample.jpg` EXIF-GPS ≈ 47.4262/12.8481, `short.mov`
     `Keys:GPSCoordinates` gesetzt, `tagged.jpg` GPS unverändert 47.0707/15.4395.
  3. Auswahl `sample.jpg` + `short.mov`, „Ort setzen", in die Karte klicken → Marker, Breite/Länge
     gefüllt, Ortsname „E2E Kartenpunkt" tippen; Mock-Protokoll ohne neue Anfrage; „Übernehmen"
     → Rückfrage „2 von 2 …" → „Alle überschreiben" → beide Details zeigen „E2E Kartenpunkt".
  4. Dieselbe Auswahl, „Ort entfernen" → Bestätigung „Ort aus 2 Fotos entfernen?" → „Entfernen"
     → beide Details „kein Ort"; Read-back ohne GPS- und Ortsfelder (Ausgangszustand der
     Fixtures wiederhergestellt).

- [x] **D5 — Ortsname und kleine Karte in der Detailansicht.**
  Dateien: `web/src/lib/detail/PhotoDetail.svelte`, der Detail-Typ in `web/src/lib/photos.ts`
  (`location.lat/lon: number | null` aus D2), `web/tests/e2e/photo-detail.spec.ts`. Unter
  `data-testid="detail-place"` (Ortsname bzw. „kein Ort" wie bisher) bei vorhandenen Koordinaten
  `<PlaceMap point={{lat, lon}} height={160}>` aus D3 **ohne** `onpick` (Leaflet lädt erst dann
  dynamisch); ohne Koordinaten keine Karte. Beim Blättern folgt der Marker dem Foto (Punktwechsel
  von außen → zentriert). Kachel-Stub über `stubTiles` aus `helpers/map.ts`.
  E2E: bestehender Test in `photo-detail.spec.ts` › „the detail view shows a photo's facts and
  steps through photos with arrow keys and swipes" erweitern: beim Foto ohne Ort (Schritt mit
  „kein Ort") keine Karte und `watchLeaflet` false; bei `tagged.jpg` Ortsname „Stadtpark, Graz"
  (Wert aus dem Index) und Karte mit `.pc-marker` sichtbar.

## Model Hints

- Alle Ds Default-Tier — 015 ist Anbindung an die Pipeline aus 009/010 und an das
  Aktions-/Dialogmuster aus 012; die Fallen (Modus gegen Index statt Datei, Skip trotzdem
  geschrieben, Leaflet statisch importiert, Kartenklick löst Reverse aus) sind je durch einen
  benannten Test oder E2E-Schritt festgenagelt.
- Review: → default

## Acceptance Tests

- AC1 → e2e `web/tests/e2e/set-place.spec.ts` › "places are set for a selection from search or
  map, asked before overwriting and removed again" (Schritte 1–3: Dialog mit Karte und
  Attribution, Treffer setzt Marker, Klick setzt Marker und Punkt) (D4) + e2e
  `web/tests/e2e/places.spec.ts` › "place search finds a POI, names it, shows it on the map and
  survives an outage" (Zentrieren auf den Treffer, Klick ohne Nominatim-Anfrage) (D3)
- AC2 → e2e `web/tests/e2e/set-place.spec.ts` › "places are set for a selection from search or
  map, asked before overwriting and removed again" (Schritt 2: ein Job, Foto + Video, Read-back)
  (D4) + integration `internal/writer/location_test.go` › `TestSetLocationWritesMappedFieldsForEveryFormat`
  (D1) + unit `internal/geo/validate_test.go` ›
  `TestLocationInputWithBadCoordinatesNamesOrControlCharsIsRefused` (D1)
- AC3 → e2e `web/tests/e2e/set-place.spec.ts` › "places are set for a selection from search or
  map, asked before overwriting and removed again" (Schritte 2 und 3: beide Antworten) (D4) +
  integration `internal/writer/location_test.go` ›
  `TestSetLocationModeMissingSkipsLocatedFilesAndModeAllOverwrites` (D1) + unit
  `internal/api/selection_test.go` › `TestSelectionCountReportsPhotosWithLocation` (D2)
- AC4 → e2e `web/tests/e2e/set-place.spec.ts` › "places are set for a selection from search or
  map, asked before overwriting and removed again" (Schritt 4) (D4) + integration
  `internal/writer/location_test.go` ›
  `TestRemoveLocationClearsCoordinatesAndPlaceFieldsAndSkipsFilesWithout` (D1)
- AC5 → e2e `web/tests/e2e/photo-detail.spec.ts` › "the detail view shows a photo's facts and
  steps through photos with arrow keys and swipes" (Ortsname + Karte mit Marker) (D5) + unit
  `internal/api/photos_test.go` › `TestPhotoDetailIncludesCoordinates` (D2)
- AC6 → e2e `web/tests/e2e/set-place.spec.ts` › "places are set for a selection from search or
  map, asked before overwriting and removed again" (Schritt 1: Startseite ohne Leaflet, Dialog lädt
  es) (D4) + e2e `web/tests/e2e/places.spec.ts` › "place search finds a POI, names it, shows it on
  the map and survives an outage" (D3) + e2e `web/tests/e2e/photo-detail.spec.ts` › "the detail view
  shows a photo's facts and steps through photos with arrow keys and swipes" (Detail ohne Ort lädt
  kein Leaflet) (D5)
- Manuelle Restprüfung (Profil-Notiz, proprietäre App): dass UGREEN Fotos den gesetzten Ort
  anzeigt, ist durch den Spike für genau diese Felder belegt — kein zusätzlicher manueller Schritt
  für 015.

## Done

Zusammenfassung: Aktionen `location.set`/`location.remove` (Modus `all`/`missing` je Datei auf frisch gelesenen Werten, Skip vor Backup), `POST /api/selection/count` → `{count, located}`, Detail-API mit `lat`/`lon`, Leaflet nur per dynamischem Import (`PlaceMap` in Picker und Detailansicht), Dialoge „Ort setzen"/„Ort entfernen" an der Auswahl, gemeinsame `Dialog.svelte`-Hülle.
Change type: feat
Commit message: `015: Standort im Batch — location.set/remove (Modus, Skip vor Backup), Karte (lazy Leaflet) im PlacePicker, Dialog Ort setzen/entfernen, Detailkarte + e2e`

Verifikation (enge Gate): `npm --prefix web run build` + `docker build --target runtime` (Image neu vor e2e), `docker build --target test .` + vitest (54), `docker build --target lint .` + eslint, `svelte-check` grün; e2e `cd web && npx playwright test tests/e2e/set-place.spec.ts tests/e2e/places.spec.ts tests/e2e/photo-detail.spec.ts` grün (Nominatim-Mock, OSM-Kacheln gestubbt). Review 1 (Sonnet): PASS mit Befunden, bestätigte behoben (vakuose Backup-Assertion, fehlende lat/lon-Pflicht → 0,0, Library-Test für gps-Lesen, lastPick-Reset, Tech-Spec §7), danach Gate erneut grün.
AC → Test (verifiziert): AC1/AC6 set-place + places + photo-detail e2e (bestanden); AC2/AC3/AC4 e2e set-place + Go-Integrationstests `internal/writer/location_test.go` (laufen im `test`-Stage, nur Paketebene `ok` sichtbar, Testnamen nicht einzeln im Output), `TestLocationInput…`, `TestSelectionCountReportsPhotosWithLocation`; AC5 photo-detail e2e + `TestPhotoDetailIncludesCoordinates` + `TestPhotoDetailReadsCoordinatesFromTheIndex`. Keine manual residue.

Decisions:
- Modus/Skip-Regeln liegen in `Pipeline.WriteFile` über `metadata.Change.LocationMode` (kein `internal/writer/location.go`); Setzen sendet immer `ClearLocation` + `Location`, damit leere Ortsteile ältere Werte löschen. `mode=all` mit identischem Ort schreibt trotzdem (Backup) — harmlos.
- `countryCode` wird validiert/normalisiert, aber nicht geschrieben (Mapping hat kein Feld).
- Die Anzahl `located` wird beim Klick auf „Übernehmen" geholt (nicht beim Tippen) — gleiche Wirkung, weniger Anfragen.
- `Dialog.svelte` extrahiert (012-Dialoge nutzen sie); `selectionStore.countSelection()` liefert `{count, located}`.
- E2E: Monatsliste zeigt Ordner (`YYYY/MM`), `showTile` baut das Label aus dem Pfad; Karten-Klick abseits der Zoom-Buttons.
- Offen/nicht behoben: `RemovePlaceDialog` nutzt `selection.count` als Momentwert; Hilfsfunktionen `showTile`/`select` in set-place.spec könnten für 016 nach helpers/ wandern.
tiers: D 5 / hard 0 · review default · cycles 1 · agents 10
