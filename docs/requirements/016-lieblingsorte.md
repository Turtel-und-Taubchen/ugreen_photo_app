---
id: 016
title: Lieblingsorte speichern und wiederverwenden
status: ready # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E4 Standorte
estimate: 2
sprint: S03
---

## Requirement

Orte, die immer wieder vorkommen (Zuhause, Eltern, Stammlokal), will ich nicht jedes Mal suchen,
sondern mit einem Tipp übernehmen.

## Acceptance Criteria

- [ ] **AC1** — Ein im Dialog „Ort setzen" gewählter Ort kann als Lieblingsort gespeichert
  werden; der Name ist dabei editierbar.
- [ ] **AC2** — Der Dialog listet die Lieblingsorte; ein Tipp übernimmt Koordinaten, Ortsname und
  Ortsteile und setzt den Marker.
- [ ] **AC3** — Lieblingsorte können umbenannt und gelöscht werden.
- [ ] **AC4** — Die Verwendung eines Lieblingsorts erzeugt keine Nominatim-Anfrage.

## Open Questions

- ~~Lieblingsorte gemeinsam für beide Personen oder pro Person?~~ answered → Decisions (Sprint)
- ~~Refinement S03 (2026-10-30) — blockiert durch die offene Frage in
  [015 › Open Questions](015-karte-und-batch-standort.md): gilt für 016 (B) Dialog „Ort setzen"
  wie geplant, (D) Seite `/orte` oder (A)/(C) Verschiebung mit 015?~~ answered → Decisions (Sprint)

## Decisions (Sprint)

- **(User)** Lieblingsorte gemeinsam oder pro Person: **gemeinsam** — eine Liste für beide Nutzer, beide können sie pflegen.
- **(User)** Spike abwarten: Ergebnis liegt vor; 015 komplett in S03, 016 wie geplant über den Dialog „Ort setzen"
- **E2E-Flow nur durch den Dialog „Ort setzen"** (Variante B), die `/orte`-Variante entfällt — AC1/AC2
  sprechen vom Dialog, und 015 liefert ihn samt Karte in S03 (Spike-Ergebnis in
  `docs/concepts/photocurator-metadaten-strategie.md`, Commit 0c8c0f1).
- **016 bleibt spike-unabhängig:** es schreibt nichts in Dateien, die Profil-Regel „Metadaten-Stories
  erst nach dem Spike" betrifft es nicht — die Ortsteile landen erst über „Übernehmen" aus 015 in Dateien.
- **015-Schnittstellen werden im Build nachgeprüft:** 015 wird parallel refined und vor 016 gebaut;
  Dateiname der Dialog-Komponente, Marker-Anbindung und E2E-Helfer zum Öffnen des Dialogs nimmt D3
  aus 015s fertigem Plan bzw. Code — 016 legt sie nicht fest, damit keine zweite, abweichende Annahme entsteht.
- **Kein `CLAUDE.md` im Repo** (geprüft 2026-10-30) — Leitplanken kommen aus Tech Spec und Profil;
  kein eigener Lane-Eintrag für den neuen Flow, er läuft im Story- und Sprint-Gate.
- **Ort der Oberfläche:** Lieblingsorte leben als Abschnitt „Lieblingsorte" in `PlacePicker.svelte` —
  der Dialog „Ort setzen" bettet genau diesen Picker ein (Entscheidung aus 014) und 015 legt auch die
  Karte in den Picker, so erscheinen sie im Dialog ohne Kopplung an Dialog-Dateien (Nebeneffekt:
  auch auf der Seite `/orte`, dort ohne Auswahl verwaltbar).
- **Übernehmen = derselbe Weg wie ein Suchtreffer:** eine gemeinsame `apply()`-Funktion im Picker
  setzt Breite/Länge/Ortsname/Ortsteile und ruft `emit()` — 015 entscheidet „Ein Zustand treibt
  Karte und Felder" (Suchtreffer zentrieren und setzen den Marker), also setzt derselbe Zustand auch
  für Lieblingsorte den Marker, ohne zweite Kopie des Punkts.
- **Ein Name:** der Name des Lieblingsorts ist der Ortsname, der beim Übernehmen eingesetzt wird
  (beim Speichern mit dem aktuellen Ortsnamen vorbelegt, editierbar) — AC2 nennt Koordinaten,
  Ortsname und Ortsteile als Inhalt, ein zweiter Anzeigename ist nicht verlangt.
- **Speichern nur mit gültigem Punkt**, Ortsteile dürfen leer sein — die Koordinaten sind der Kern
  eines Orts, Ortsteile gibt es nur nach Suchtreffer/Reverse, ein manueller Punkt soll trotzdem
  speicherbar sein.
- **Namen eindeutig** ohne Unterschied in Groß-/Kleinschreibung und Leerraum (sonst 409) — zwei
  „Zuhause" wären beim Ein-Tipp-Übernehmen nicht zu unterscheiden.
- **Speicher:** Tabelle `favorite_places` in `app.db` (Tech Spec §2.2) mit `AUTOINCREMENT`-IDs —
  nie wiederverwendete IDs verhindern, dass eine veraltete Liste der anderen Person einen neu
  angelegten, fremden Eintrag umbenennt oder löscht.
- **Keine Nutzerzuordnung** gespeichert (kein `created_by`) — die Liste ist gemeinsam (User-Entscheidung)
  und nichts würde die Spalte lesen.
- **API:** `GET`/`POST /api/places`, `PATCH`/`DELETE /api/places/{id}`; die Tech-Spec-Tabelle §6
  bekommt `PATCH` — §6 nennt GET/POST/DELETE, Umbenennen (AC3) braucht ein Update-Verb.
- **Validierung:** Name getrimmt, 1–200 Zeichen, keine Steuerzeichen; Ortsteile je ≤ 200 Zeichen,
  keine Steuerzeichen; Koordinaten endlich, Breite −90…90, Länge −180…180 — Name und Ortsteile
  landen später über ExifTool-Argumentdateien in Dateien (Tech Spec §9 weist Zeilenumbrüche ab).
- **Sortierung** alphabetisch nach normalisiertem Namen — stabil und auffindbar; Sortierung nach
  Nutzung verlangt kein AC.
- **Löschen zweistufig inline** („Löschen" → „Wirklich löschen"/„Abbrechen"), Umbenennen inline —
  schützt am Handy vor Fehltipps direkt neben dem Übernehmen-Knopf, ohne nativen Browser-Dialog.
- **Unabhängig von Nominatim:** Lieblingsorte bleiben nutzbar, wenn die Ortssuche deaktiviert oder
  ausgefallen ist, und weder Speichern noch Übernehmen löst Reverse-Geocoding aus — die Ortsteile
  werden mitgespeichert (AC4).
- **Paket:** `internal/geo/favorites.go` mit dem vorhandenen `geo.Parts` — Tech Spec §1 ordnet Orte
  `internal/geo` zu, ein eigenes Paket für eine Tabelle lohnt nicht.
- **Tests:** ein neuer E2E-Flow für Speichern/Übernehmen/Umbenennen/Löschen (ein Nutzerpfad); die
  gemeinsame Liste beider Nutzer prüft ein Go-API-Test mit zwei Sitzungen gegen einen Handler —
  gleiche Aussage ohne zweiten Browser-Login im Flow.

## Plan

Kleine CRUD-Funktion in drei Schichten; keine Dateischreibzugriffe, kein Nominatim.

1. **D1 — Speicher:** Migration `favorite_places` (ans Ende der append-only-Liste, nach den
   Migrationen von 009/010/013), `internal/geo/favorites.go` mit List/Create/Rename/Delete und
   Validierung; Go-Tests gegen `store.OpenAppDB(t.TempDir())`.
2. **D2 — API:** `internal/api/places.go` (`GET/POST /api/places`, `PATCH/DELETE /api/places/{id}`),
   `LibraryDeps.Places`, Verdrahtung in `cmd/photocurator/main.go`, Tech-Spec-Zeile;
   httptest mit `anna` und `ben`.
3. **D3 — Oberfläche + E2E:** `web/src/lib/places.ts`, Abschnitt „Lieblingsorte" und
   „Als Lieblingsort speichern" in `PlacePicker.svelte` (gemeinsames `apply()` mit Suchtreffern),
   ein Playwright-Flow gegen den Nominatim-Mock durch den Dialog „Ort setzen" aus 015.

Reihenfolge = Abhängigkeit (D1 → D2 → D3). D1/D2 hängen an nichts außer 014; D3 setzt die Karte im
`PlacePicker` aus 015 voraus (Marker).

Nicht Teil: Sortierung nach Nutzung, Lieblingsorte pro Person, Ortsvorschlag aus Zeitnachbarn (017),
Schreiben in Dateien (bleibt „Übernehmen" aus 015).

## Deliverables

- [ ] **D1 — Tabelle `favorite_places` + Speicherfunktionen in `internal/geo`.**
  Dateien: `internal/store/appdb.go`, `internal/geo/favorites.go` (neu),
  `internal/geo/favorites_test.go` (neu). Muster für einfache app.db-Funktionen:
  `internal/tags/definitions.go` (Funktionen mit `ctx` + `*sql.DB`).
  Migration **ans Ende** der Liste `migrations` in `internal/store/appdb.go` anhängen (append only:
  keine bestehende Zeile ändern; im selben Sprint hängen 009/010/013 eigene an — die eigene kommt
  danach):
  `CREATE TABLE favorite_places (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL,
  name_key TEXT NOT NULL UNIQUE, lat REAL NOT NULL, lon REAL NOT NULL, sublocation TEXT NOT NULL
  DEFAULT '', city TEXT NOT NULL DEFAULT '', state TEXT NOT NULL DEFAULT '', country TEXT NOT NULL
  DEFAULT '', country_code TEXT NOT NULL DEFAULT '', created_at INTEGER NOT NULL, updated_at INTEGER
  NOT NULL)` (AUTOINCREMENT: gelöschte IDs werden nie wieder vergeben).
  Typen in `favorites.go`: `type Favorite struct { ID int64; Name string; Lat, Lon float64;
  Parts Parts }` mit JSON-Tags `id`, `name`, `lat`, `lon`, `parts` — `Parts` ist das vorhandene `geo.Parts` aus `internal/geo/place.go` (camelCase-JSON
  `sublocation,city,state,country,countryCode`). Fehler: `ErrFavoriteExists`,
  `ErrFavoriteNotFound`, Eingabefehler als `*FavoriteInputError{Msg string}` (deutsche Meldung, wird
  per `errors.As` erkannt und unverändert angezeigt).
  Funktionen: `ListFavorites(ctx, db) ([]Favorite, error)` (sortiert nach `name_key`, dann `id`;
  leer → `[]Favorite{}`, nie `nil`); `CreateFavorite(ctx, db, f Favorite, now time.Time)
  (Favorite, error)`; `RenameFavorite(ctx, db, id int64, name string, now time.Time) (Favorite,
  error)`; `DeleteFavorite(ctx, db, id int64) error`.
  Normalisierung/Validierung (in Create und Rename): Name `strings.TrimSpace`, nicht leer, ≤ 200
  Runen, kein Zeichen mit `unicode.IsControl` → sonst `FavoriteInputError` („Bitte einen Namen mit
  höchstens 200 Zeichen ohne Zeilenumbruch eingeben."); `name_key` = Name klein, Leerraum-Folgen zu
  einem Leerzeichen (`strings.Join(strings.Fields(strings.ToLower(name)), " ")`); Ortsteile je
  getrimmt, ≤ 200 Runen, keine Steuerzeichen, `CountryCode` groß; Lat/Lon nicht NaN/Inf, Breite
  −90…90, Länge −180…180 → sonst `FavoriteInputError`. Gleicher `name_key` eines **anderen**
  Eintrags → `ErrFavoriteExists` (UNIQUE-Verletzung abfangen bzw. vorher prüfen, in einer
  Transaktion); Umbenennen auf denselben Schlüssel (nur Schreibweise geändert) ist erlaubt.
  Unbekannte ID bei Rename/Delete → `ErrFavoriteNotFound`.
  Tests in `internal/geo/favorites_test.go` (`store.OpenAppDB(t.TempDir())`):
  „favourite places are listed by name and keep their place parts",
  „a favourite name differing only in case and spaces is a duplicate",
  „a vanished favourite place cannot be renamed or deleted and its id is never reused",
  „favourite places with invalid coordinates or control characters are refused".

- [ ] **D2 — API `/api/places`.** Nutzt aus `internal/geo/favorites.go`: `Favorite`
  (`{id,name,lat,lon,parts{sublocation,city,state,country,countryCode}}`), `ListFavorites`,
  `CreateFavorite`, `RenameFavorite`, `DeleteFavorite`, `ErrFavoriteExists`, `ErrFavoriteNotFound`,
  `*FavoriteInputError{Msg}`.
  Dateien: `internal/api/places.go` (neu), `internal/api/places_test.go` (neu),
  `internal/api/scan.go` (Feld `Places *sql.DB` in `LibraryDeps`, Kommentar „Places serves
  /api/places; nil leaves the routes unregistered."), `internal/api/server.go` (`if lib.Places !=
  nil { registerPlaces(mux, lib.Places) }` neben den anderen), `cmd/photocurator/main.go`
  (`Places: db`, das app.db-Handle, das auch `Tags.App` bekommt),
  `docs/concepts/photocurator-tech-spec.md` (§6-Zeile `/api/places`: Methoden `GET/POST/PATCH/DELETE`,
  Pfade `/api/places`, `/api/places/{id}`). Muster: `internal/api/tags.go` (`MaxBytesReader` 64 KiB,
  `badRequest` aus `internal/api/photos.go`, 500 über das vorhandene `tagsFailed(w, what, err)` —
  keinen weiteren 500-Helfer anlegen). Routen hinter der bestehenden Session-/CSRF-Kette (keine
  neue Ausnahme):
  `GET /api/places` → 200 `{"places":[Favorite…]}`;
  `POST /api/places` Body `{name,lat,lon,parts}` → 201 `{"place":Favorite}`;
  `PATCH /api/places/{id}` Body `{name}` → 200 `{"place":Favorite}`;
  `DELETE /api/places/{id}` → 204.
  Fehler: kaputtes JSON → 400 „Ungültige Anfrage."; `FavoriteInputError` → 400 mit `Msg`; `{id}` keine
  positive Ganzzahl → 400; `ErrFavoriteNotFound` → 404 `{"error":"Lieblingsort nicht gefunden."}`;
  `ErrFavoriteExists` → 409 `{"error":"Es gibt schon einen Lieblingsort „<Name>“."}`. Keine
  Nutzerzuordnung — eine Liste für alle angemeldeten Nutzer.
  Tests in `internal/api/places_test.go` (eigene app.db per `store.OpenAppDB(t.TempDir())` als
  `LibraryDeps{Places: db}` an `newTestServerDeps`; Sitzungen über `s.signedIn(t, "anna")` /
  `"ben"`, Anfragen über `s.do(…)` wie in `internal/api/geo_test.go`):
  „favourite places are one list that both users maintain" (anna legt an → ben sieht ihn, benennt
  um → anna sieht den neuen Namen → ben löscht → annas Liste leer);
  „favourite place requests with bad input, duplicates or unknown ids are refused" (400, 409, 404).

- [ ] **D3 — Lieblingsorte im PlacePicker + E2E-Flow.** Nutzt die API aus D2:
  `GET /api/places` → `{places:[Favorite]}`, `POST /api/places` `{name,lat,lon,parts}` → 201
  `{place}`, `PATCH /api/places/{id}` `{name}` → `{place}`, `DELETE /api/places/{id}` → 204,
  Fehler `{error}` mit 400/404/409; `Favorite` = `{id,name,lat,lon,parts:PlaceParts}`.
  Dateien: `web/src/lib/places.ts` (neu; über den Wrapper `api()` aus `web/src/lib/api.ts`, der
  `X-Requested-With` setzt: `listFavorites()`, `createFavorite({name,lat,lon,parts})`,
  `renameFavorite(id,name)`, `deleteFavorite(id)`; Typ `Favorite` mit `PlaceParts` aus
  `web/src/lib/geo.ts`), `web/src/lib/PlacePicker.svelte`, die Dialog-Komponente „Ort setzen" aus
  015 (nur falls nötig, siehe Marker), `web/tests/e2e/favorite-places.spec.ts` (neu).
  Picker (Svelte 5, deutsch, Touch-Flächen ≥ 44 px, ab 360 px nutzbar):
  - **Übernehmen:** die bestehende `choose(place)` zu `apply({lat, lon, name, parts})` umbauen
    (setzt `lat`/`lon` als `String(…)`, `name`, `parts`, ruft `emit()`); Suchtreffer rufen
    `apply({lat, lon, name: place.label, parts})`, Lieblingsorte `apply(fav)` — **kein**
    `/api/geo/*`-Aufruf. Marker: die Karte im `PlacePicker` aus 015 folgt dem Picker-Zustand
    (Suchtreffer zentrieren die Karte und setzen den Marker); `apply` setzt genau diesen Zustand.
    Setzt 015 den Marker bei Suchtreffern über einen eigenen Aufruf, ruft `apply` ihn ebenso.
    Kartenklicks lösen laut 015 kein Reverse-Geocoding aus — `apply` auch nicht.
  - **Liste:** Abschnitt mit Überschrift „Lieblingsorte" **oberhalb** der Suche, `<ul role="list"
    aria-label="Lieblingsorte">`; je Eintrag ein Knopf mit dem Namen (+ `parts.city` klein) zum
    Übernehmen, dazu Knöpfe „Umbenennen" (`aria-label="„<Name>“ umbenennen"`) und „Löschen"
    (`aria-label="„<Name>“ löschen"`). Leer → „Noch keine Lieblingsorte.". Laden einmal beim
    Mounten (`listFavorites`); Fehler → `role="alert"` über das vorhandene `message()` im Picker
    (keine neue Kopie). Die Liste ist **nicht** von `enabled` abhängig (bleibt bei deaktivierter
    oder ausgefallener Ortssuche bedienbar).
  - **Umbenennen inline:** Feld „Neuer Name" (vorbelegt) + „Speichern"/„Abbrechen"; Fehler
    (409/400) → Alert mit Server-Meldung, Feld bleibt offen.
  - **Löschen zweistufig:** „Löschen" ersetzt die Knöpfe des Eintrags durch „Wirklich löschen"
    (`aria-label="„<Name>“ endgültig löschen"`) + „Abbrechen"; erst der zweite Tipp ruft
    `deleteFavorite`.
  - **Speichern:** im Abschnitt „Gewählter Ort" Knopf „Als Lieblingsort speichern", aktiv nur bei
    gültigem `point`; öffnet Feld „Name des Lieblingsorts" (vorbelegt mit dem aktuellen Ortsnamen
    `name`, editierbar) + „Speichern"/„Abbrechen" → `createFavorite({name, lat, lon, parts})`, neuer
    Eintrag erscheint sortiert in der Liste; Fehler → Alert, Feld bleibt offen.
  E2E `web/tests/e2e/favorite-places.spec.ts` › „favourite places are saved, reused without
  Nominatim, renamed and deleted" (Muster: `web/tests/e2e/places.spec.ts` — Mock unter
  `http://127.0.0.1:${NOMINATIM_MOCK_PORT}` aus `./compose`, `POST /__mock/reset`,
  `GET /__mock/requests`, `login(page, users.anna)` aus `./helpers/auth`, `test.setTimeout(300_000)`).
  Zu Beginn alle vorhandenen Lieblingsorte über `page.request` löschen (`GET /api/places`, dann
  `DELETE /api/places/{id}` mit Header `X-Requested-With: fetch`). OSM-Kacheln wie in 015: per
  `page.route` auf `tile.openstreetmap.org` mit festem leerem PNG beantworten (nie echte Kacheln).
  „Picker öffnen" heißt: ein Foto im Raster auswählen und den Dialog „Ort setzen" öffnen, genau wie
  im E2E-Flow von 015 (dessen Helfer wiederverwenden, sonst `helpers/grid.ts`/`helpers/library.ts`);
  „schließen" = Dialog ohne Übernehmen schließen (nichts wird in Dateien geschrieben). Marker-Prüfung
  wie im E2E-Flow von 015 (Leaflet-Marker in der Karte des Pickers sichtbar). 015 ist vor 016 gebaut:
  Dateiname der Dialog-Komponente, Marker-Anbindung und Öffnen-Helfer aus dem Code von 015 nehmen,
  nicht neu erfinden. Schritte:
  1. Picker öffnen, „Ritzenhofer Saalfelden" + Enter, ersten Treffer wählen; „Als Lieblingsort
     speichern" → Feld „Name des Lieblingsorts" hat „Ritzenhofer, Saalfelden"; auf „Stammhotel"
     ändern, „Speichern" → Liste „Lieblingsorte" enthält „Stammhotel".
  2. Picker schließen; `POST /__mock/reset`; ab jetzt Browser-Anfragen auf
     `/api/geo/` mitschreiben (`page.on('request')`); Picker öffnen; „Stammhotel" tippen →
     „Ortsname" = „Stammhotel", „Breite" = „47.4262", „Länge" = „12.8481", City „Saalfelden",
     Country „Österreich" (Selektor wie in `places.spec.ts`), Marker sichtbar; danach Mock-Protokoll
     leer **und** keine Browser-Anfrage auf `/api/geo/`.
  3. „„Stammhotel“ umbenennen" → „Hotel Ritzenhofer", „Speichern" → Liste zeigt den neuen Namen,
     nicht mehr den alten; „„Hotel Ritzenhofer“ löschen" → Eintrag noch da, „„Hotel Ritzenhofer“
     endgültig löschen" → Eintrag weg; Picker schließen und öffnen → „Noch keine Lieblingsorte."
     (Umbenennen und Löschen sind gespeichert).

## Model Hints

- Alle Ds Default-Tier — einfache CRUD-Funktion ohne Dateischreibzugriff, Nebenläufigkeit oder
  Nominatim-Verkehr; das Risiko „Übernehmen löst doch eine Nominatim-Anfrage aus" fängt der E2E-Flow
  am Mock-Protokoll.
- Review: → default

## Acceptance Tests

- AC1 → e2e `web/tests/e2e/favorite-places.spec.ts` › "favourite places are saved, reused without
  Nominatim, renamed and deleted" (Schritt 1: Speichern mit editiertem Namen) + unit
  `internal/api/places_test.go` › "favourite places are one list that both users maintain"
  (D1, D2, D3)
- AC2 → e2e `web/tests/e2e/favorite-places.spec.ts` › "favourite places are saved, reused without
  Nominatim, renamed and deleted" (Schritt 2: Liste im Picker, Tipp füllt Koordinaten, Ortsname,
  Ortsteile, Marker sichtbar) + unit `internal/geo/favorites_test.go` › "favourite places are listed
  by name and keep their place parts" (D1, D3)
- AC3 → e2e `web/tests/e2e/favorite-places.spec.ts` › "favourite places are saved, reused without
  Nominatim, renamed and deleted" (Schritt 3) + unit `internal/geo/favorites_test.go` › "a favourite
  name differing only in case and spaces is a duplicate", "a vanished favourite place cannot be
  renamed or deleted and its id is never reused", "favourite places with invalid coordinates or
  control characters are refused"; `internal/api/places_test.go` › "favourite places are one list
  that both users maintain", "favourite place requests with bad input, duplicates or unknown ids are
  refused" (D1, D2, D3)
- AC4 → e2e `web/tests/e2e/favorite-places.spec.ts` › "favourite places are saved, reused without
  Nominatim, renamed and deleted" (Schritt 2: Mock-Protokoll leer, keine Browser-Anfrage auf
  `/api/geo/`) (D3)
- Entscheidung (User) „gemeinsame Liste" → unit `internal/api/places_test.go` › "favourite places
  are one list that both users maintain" (D2)

## Done
