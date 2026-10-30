---
id: 014
title: Ortssuche und Reverse-Geocoding über Nominatim
status: ready # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E4 Standorte
estimate: 5
sprint: S02
---

## Requirement

Ich will Orte per Suche finden — auch konkrete Lokale und Hotels wie „Ritzenhofer Saalfelden" —
und für einen Kartenpunkt einen lesbaren Namen bekommen. Die Nominatim-Nutzungsrichtlinie wird
eingehalten: max. 1 Anfrage pro Sekunde, eigener User-Agent, Ergebnisse cachen, keine
Autovervollständigung. Details: [Tech Spec §10](../concepts/photocurator-tech-spec.md).

## Acceptance Criteria

- [ ] **AC1** — Eine Suche (ausgelöst per Enter oder Knopf, nicht bei jedem Tastendruck) zeigt
  bis zu 10 Treffer mit Name, Adresse und Art; mit der hinterlegten Nominatim-Testantwort für
  „Ritzenhofer Saalfelden" erscheint der POI als erster Treffer.
- [ ] **AC2** — Alle Anfragen an Nominatim gehen über den Server; bei 5 gleichzeitigen Suchen
  (auch von zwei Nutzern) liegen zwischen zwei ausgehenden Anfragen mindestens 1 s.
- [ ] **AC3** — Jede ausgehende Anfrage trägt den User-Agent
  `PhotoCurator/<version> (<Kontakt aus Konfiguration>)`; ohne konfigurierten Kontakt ist die
  Ortssuche deaktiviert und die Oberfläche sagt warum.
- [ ] **AC4** — Eine wiederholte gleiche Suche (Groß-/Kleinschreibung und Leerzeichen egal)
  wird aus dem Cache beantwortet, ohne ausgehende Anfrage; Cache-Einträge verfallen nach der
  Frist (Platzhalter: 90 Tage).
- [ ] **AC5** — Für einen Kartenpunkt liefert Reverse-Geocoding über denselben Limiter und Cache
  einen Ortsvorschlag.
- [ ] **AC6** — Aus einem Treffer entsteht ein lesbarer, editierbarer Ortsname nach dem Muster
  `<Name>, <Ort>` („Ritzenhofer, Saalfelden") sowie die Ortsteile Sublocation, City, State,
  Country.
- [ ] **AC7** — Ist Nominatim nicht erreichbar oder antwortet mit Fehler, erscheint eine
  verständliche Meldung; ein Ort kann trotzdem per Kartenklick und manuell eingegebenem Namen
  gesetzt werden.

## Open Questions

## Decisions (Sprint)

- **Oberfläche in S02:** eigene Seite `/orte` („Ortssuche", Link in der Kopfzeile) mit der
  wiederverwendbaren Komponente `PlacePicker.svelte`; 015 bettet dieselbe Komponente in den Dialog
  „Ort setzen" ein — Dialog und Auswahl, auf die er wirkt, gehören zu 015, 014 muss in S02 allein
  benutzbar und per E2E prüfbar sein.
- **Kartenpunkt ohne Karte:** In 014 ist der „Kartenpunkt" ein Koordinatenpaar (Felder Breite/Länge
  im Picker); 015 setzt denselben Picker-Zustand per Kartenklick — Leaflet, OSM-Kacheln und
  Lazy-Loading sind 015 AC1/AC6 und würden in E2E-Tests echte Kacheln laden.
- **„Ort gesetzt" (AC7) heißt in 014:** der Picker zeigt einen vollständigen „Gewählten Ort" (Punkt +
  von Hand eingegebener Name); in Dateien geschrieben wird erst in 015 — 014 schreibt laut
  Sprint-Notiz nichts in Dateien.
- **API:** `GET /api/geo/status`, `GET /api/geo/search?q=`, `GET /api/geo/reverse?lat=&lon=`
  liefern ein eigenes Treffer-Format statt rohem Nominatim-JSON — Pfade laut Tech Spec §6, und das
  Frontend bleibt vom Nominatim-Format unabhängig.
- **Basis-URL konfigurierbar:** `PC_NOMINATIM_URL` (Standard `https://nominatim.openstreetmap.org`),
  E2E zeigt auf einen Mock — Sprint-Notiz: nie den echten Dienst in Tests.
- **Limiter:** ein globales Tor pro Prozess, ≥ 1 s zwischen den Starts zweier ausgehender Anfragen
  (fehlgeschlagene zählen mit), nach dem Warten wird der Cache erneut geprüft, das Warten bricht mit
  dem Request-Kontext ab — erfüllt AC2 und vermeidet Doppelanfragen bei gleichzeitigen gleichen Suchen.
- **Cache-Schlüssel und Frist:** `search:de:<klein, getrimmt, Leerraum-Folgen zu einem Leerzeichen>`
  und `reverse:de:<lat>,<lon>` auf 5 Nachkommastellen (~1 m); Frist 90 Tage als Konstante,
  abgelaufene Einträge werden beim nächsten Abruf überschrieben; leere Trefferlisten werden
  gecacht, Fehler nicht — so wenig Last auf Nominatim wie möglich, ohne Ausfälle festzuschreiben.
- **Ortsname/Ortsteile:** Name = `name` (Fallback `namedetails.name`), Ort =
  `city|town|village|municipality`; Label `<Name>, <Ort>`, ohne Name oder Name = Ort → nur Ort, ohne
  Ort → nur Name; Sublocation = Name (wenn ≠ Ort), sonst `suburb|neighbourhood|hamlet` — Tech Spec
  §10 nennt `<name>, <city|town|village>`, der Rest folgt der IPTC-Bedeutung der Felder.
- **„Art" eines Treffers** = Nominatim-`type` unverändert (z. B. `hotel`) — keine
  Übersetzungstabelle im MVP, `accept-language` übersetzt Typen nicht.
- **Fehler:** Nominatim nicht erreichbar/Timeout 10 s/Status ≠ 200/kaputtes JSON → 502 mit
  verständlicher deutscher Meldung; leere/zu lange Suche (> 200 Zeichen) oder ungültige Koordinaten
  → 400 ohne ausgehende Anfrage — Eingabefehler kosten Nominatim nichts.
- **Deaktiviert (AC3):** ohne `PC_NOMINATIM_CONTACT` startet die App normal, `status` liefert
  `{enabled:false, reason}`, `search`/`reverse` antworten 503 ohne ausgehende Anfrage; die Anzeige
  des Grunds wird per Go-API-Test + Vitest (View-Model) geprüft, nicht per E2E — ein zweiter
  Container ohne Kontakt kostet einen kompletten Compose-Start pro Lauf, und die Anzeige ist keine
  Nutzeraktion.
- **Nominatim-Mock:** Node-HTTP-Server im Playwright-Global-Setup (Port 18471), vom Container über
  `host.docker.internal` (`extra_hosts: host-gateway`) erreichbar, mit Steuer-Endpunkten und
  handgeschriebenen jsonv2-Fixtures in `testdata/nominatim/` — Tests steuern den Fehlerfall und
  zählen Anfragen samt User-Agent, ohne Netz nach außen.
- **Ein E2E-Flow** für Suche, Cache, Reverse und Ausfall — alles derselbe Nutzerpfad (Seite
  `/orte`), jeder weitere Flow kostet jeden Lauf einen Start.

## Plan

Neues Paket `internal/geo` (Tech Spec §1/§10) hinter drei API-Routen, ein Picker im Frontend,
ein Nominatim-Mock für E2E. Reihenfolge = Abhängigkeit:

1. **D1 — Treffer-Mapping + Fixtures:** `internal/geo/place.go` wandelt Nominatim-jsonv2 in
   `Place` (Name, Adresse, Art, lat/lon, Label, Ortsteile); handgeschriebene jsonv2-Fixtures in
   `testdata/nominatim/`.
2. **D2 — Geocoder (hard):** Client (URL, Parameter, User-Agent, Timeout), globaler Limiter mit
   injizierbarer Uhr, Cache in `app.db` (Migration 2 `geocode_cache`), Normalisierung;
   `Search`/`Reverse` laufen beide über Cache → Limiter → Cache-Recheck → HTTP.
3. **D3 — Konfiguration + API:** `PC_NOMINATIM_CONTACT`/`PC_NOMINATIM_URL`, Handler
   `status`/`search`/`reverse` in `internal/api/geo.go`, eine Geocoder-Instanz in `serve`;
   httptest mit zwei Sitzungen und Upstream-Fake.
4. **D4 — Nominatim-Mock im E2E-Harness:** Mock-Server mit Steuer-Endpunkten, Start/Stopp in
   Global-Setup/-Teardown, Compose-Overlay setzt `PC_NOMINATIM_URL` + `extra_hosts`.
5. **D5 — Oberfläche + E2E-Flow:** `web/src/lib/geo.ts`, `PlacePicker.svelte`, Seite `/orte` +
   Kopfzeilen-Link, Vitest fürs View-Model, ein Playwright-Flow gegen den Mock.

Nicht Teil: Karte/Leaflet und Schreiben in Dateien (015), Lieblingsorte (016), Ortsvorschlag aus
Zeitnachbarn (017), Cache-Aufräumjob.

## Deliverables

- [ ] **D1 — Nominatim-Treffer → `Place` (Name, Label, Ortsteile) + Fixtures.**
  Dateien: `internal/geo/place.go`, `internal/geo/place_test.go`,
  `testdata/nominatim/search-ritzenhofer-saalfelden.json`, `testdata/nominatim/search-saalfelden.json`,
  `testdata/nominatim/reverse-saalfelden.json`. Fixtures sind **handgeschriebene** Antworten im
  Format von `format=jsonv2&addressdetails=1&namedetails=1` (nie vom echten Dienst holen):
  „Ritzenhofer Saalfelden" = Array mit 12 Treffern (für die 10er-Grenze), der erste ein POI mit
  `category:"tourism"`, `type:"hotel"`, `name:"Ritzenhofer"`, `namedetails.name:"Ritzenhofer"`,
  `address{road, town:"Saalfelden", state:"Salzburg", country:"Österreich", country_code:"at"}`,
  `display_name` mit voller Adresse, `lat`/`lon` in Saalfelden (≈ 47.426, 12.848);
  `search-saalfelden.json` = ein Treffer `type:"town"`, `name:"Saalfelden"`, `address.town:"Saalfelden"`;
  `reverse-saalfelden.json` = **ein Objekt** (kein Array), z. B. `type:"restaurant"`,
  `name:"Gasthof Beispiel"`, `address.town:"Saalfelden"`.
  Typen: `type Parts struct{ Sublocation, City, State, Country, CountryCode string }`,
  `type Place struct{ Name, Address, Kind string; Lat, Lon float64; Label string; Parts Parts }`;
  JSON-Tags camelCase (`name,address,kind,lat,lon,label,parts{sublocation,city,state,country,countryCode}`).
  `ParseSearch([]byte) ([]Place, error)` (höchstens 10, Reihenfolge erhalten) und
  `ParseReverse([]byte) (*Place, error)` (Antwort `{"error":"Unable to geocode"}` → `nil, nil`).
  jsonv2 liefert `lat`/`lon` als **Strings** → parsen. Regeln: Name = `name`, Fallback
  `namedetails.name`; Ort = erster nicht leerer Wert aus `address.city|town|village|municipality`;
  Label = `Name + ", " + Ort`; ohne Name oder Name == Ort → Ort; ohne Ort → Name; beides leer →
  erster Komma-Teil von `display_name`. Sublocation = Name, wenn ≠ Ort, sonst erster nicht leerer
  Wert aus `suburb|neighbourhood|hamlet`; City = Ort; State = `state`; Country = `country`;
  CountryCode = `strings.ToUpper(country_code)`; Address = `display_name`; Kind = `type`.
  Tests in `internal/geo/place_test.go`: „a POI hit becomes 'Ritzenhofer, Saalfelden' with its
  place parts", „a hit without its own name is labelled with the town alone",
  „search results keep Nominatim's order and stop at ten".

- [ ] **D2 — Geocoder: Client, globaler Limiter, Cache in `app.db`.** Nutzt `ParseSearch`/
  `ParseReverse` und `Place` aus `internal/geo/place.go`.
  Dateien: `internal/geo/geocoder.go`, `internal/geo/limiter.go`, `internal/geo/cache.go`,
  `internal/geo/geocoder_test.go`, `internal/geo/limiter_test.go`, `internal/store/appdb.go`
  (Migration an die append-only-Liste anhängen: `CREATE TABLE geocode_cache (key TEXT PRIMARY KEY,
  response TEXT NOT NULL, fetched_at INTEGER NOT NULL)`; Test-DB über `store.OpenAppDB(t.TempDir())`).
  API: `New(Options{BaseURL, Contact, Version string; DB *sql.DB; HTTP *http.Client;
  Now func() time.Time; Sleep func(context.Context, time.Duration) error}) *Geocoder`
  (Defaults: Client-Timeout 10 s, echte Uhr); `Search(ctx, q string) ([]Place, error)`,
  `Reverse(ctx, lat, lon float64) (*Place, error)`, `Enabled() bool` (= Contact nicht leer).
  Fehler: `ErrDisabled` (ohne Contact, keine Anfrage), `ErrUpstream` (Netzfehler, Timeout,
  Status ≠ 200, kaputtes JSON; gewrappt). Ausgehend:
  `GET {BaseURL}/search?q=…&format=jsonv2&addressdetails=1&namedetails=1&accept-language=de&limit=10`
  bzw. `GET {BaseURL}/reverse?lat=…&lon=…&format=jsonv2&addressdetails=1&namedetails=1&accept-language=de`,
  Header `User-Agent: PhotoCurator/<Version> (<Contact>)` exakt. Cache-Schlüssel: Suche
  `search:de:` + Anfrage kleingeschrieben, getrimmt, Leerraum-Folgen zu einem Leerzeichen
  (gesendet wird die getrimmte Anfrage); Reverse `fmt.Sprintf("reverse:de:%.5f,%.5f", lat, lon)`.
  Gespeichert wird der rohe Antwort-Body (beim Lesen über D1 geparst); ein Eintrag gilt, solange
  `now - fetched_at < CacheTTL` (Konstante 90 Tage), sonst neu holen und per Upsert überschreiben;
  leere Ergebnisse werden gecacht, Fehler nicht. Ablauf je Aufruf: Cache → `limiter.Wait(ctx)` →
  **Cache erneut** → HTTP → Cache schreiben. Limiter: **eine** Instanz pro Geocoder, Mutex +
  Zeitpunkt des letzten Starts; jeder Start frühestens 1 s nach dem vorigen, auch wenn der vorige
  fehlschlug; der Start-Zeitpunkt wird vor dem HTTP-Aufruf festgehalten; `Wait` respektiert `ctx`
  (Abbruch → kein Start, kein verbrauchter Slot).
  Tests (Upstream = `httptest.Server`, der Pfad, Query, User-Agent und den Fake-Zeitpunkt jeder
  Anfrage mitschreibt; Fake-Uhr, deren `Sleep` die Zeit vorstellt — keine echten Wartezeiten;
  `go test -race`): „five concurrent searches reach Nominatim at least one second apart",
  „a search asks Nominatim with jsonv2, address and name details, German and ten results and our
  user agent", „a repeated search differing only in case and spaces is answered from the cache",
  „a cached answer older than ninety days is fetched again", „reverse geocoding shares the limiter
  and the cache with search", „an unreachable or failing Nominatim yields an upstream error and
  caches nothing", „without a contact the geocoder sends nothing".

- [ ] **D3 — Konfiguration + API `/api/geo/*`.** Nutzt `geo.New`, `Search`, `Reverse`, `Enabled`,
  `ErrDisabled`, `ErrUpstream` aus `internal/geo`.
  Dateien: `internal/config/geo.go` (neu: `Geo{Contact, BaseURL}` aus `PC_NOMINATIM_CONTACT` und
  `PC_NOMINATIM_URL`, Muster wie `PathsFromEnv(getenv)` in `internal/config/paths.go`; Standard-URL
  `https://nominatim.openstreetmap.org`, abschließenden `/` entfernen; leerer Kontakt ist **kein**
  Startfehler), `internal/api/geo.go`, `internal/api/geo_test.go`, `internal/api/server.go`
  (Routen registrieren; `NewHandler` bekommt den Geocoder als zusätzliche Abhängigkeit),
  `cmd/photocurator/main.go` (eine Geocoder-Instanz mit `version.Version`, `db`, Config —
  bestehende Aufrufer von `NewHandler` in Tests anpassen), `deploy/README.md` (beim Eintrag
  `NOMINATIM_CONTACT`: ohne Kontakt ist die Ortssuche deaktiviert).
  Routen hinter der bestehenden Session-Prüfung (Default-deny, keine neue Ausnahme):
  `GET /api/geo/status` → `{"enabled":true}` bzw. `{"enabled":false,"reason":"Ortssuche
  deaktiviert: Es ist kein Nominatim-Kontakt konfiguriert (NOMINATIM_CONTACT)."}`;
  `GET /api/geo/search?q=` → `{"results":[Place…]}`; leer/nur Leerraum/> 200 Zeichen → 400
  `{"error":"…"}`; `GET /api/geo/reverse?lat=&lon=` → `{"place":Place|null}`; nicht parsebar,
  lat ∉ [-90,90] oder lon ∉ [-180,180] → 400. `ErrDisabled` → 503 `{"error":<reason wie oben>}`;
  `ErrUpstream` → 502 `{"error":"Ortssuche gerade nicht erreichbar. Du kannst den Ort per
  Koordinaten und Namen selbst setzen."}`; Request-Kontext an den Geocoder durchreichen.
  Tests (httptest, zwei angemeldete Sitzungen `anna`/`ben` gegen **einen** Handler, Upstream-Fake
  zählt Anfragen und Fake-Zeitpunkte, Fake-Uhr über `geo.Options`): „searches of two users share
  one limit of one request per second", „without a contact the place search reports why and sends
  nothing", „a Nominatim outage answers 502 with a readable message", „an empty search or invalid
  coordinates are refused without asking Nominatim".

- [ ] **D4 — Nominatim-Mock im E2E-Harness.** Dateien: `web/tests/e2e/nominatim-mock.ts` (neu),
  `web/tests/e2e/global-setup.ts`, `web/tests/e2e/global-teardown.ts`,
  `web/tests/e2e/compose.e2e.yml`, `web/tests/e2e/compose.ts` (Konstante
  `NOMINATIM_MOCK_PORT = 18471`). Mock = `node:http`-Server auf `0.0.0.0:18471` (nur so erreicht
  ihn der Container auch unter Linux; nur während des Laufs offen), gestartet in `globalSetup`
  **vor** `compose up`, gestoppt in `globalTeardown` (Handle in Modul-Variable oder
  `globalSetup` gibt eine Teardown-Funktion zurück; Teardown scheitert nicht, wenn der Mock nicht
  lief). Antworten aus den Fixtures `testdata/nominatim/*.json` (Pfad relativ wie `LIBRARY` in
  `global-setup.ts`): `GET /search` → Zuordnung über normalisiertes `q` (klein, getrimmt,
  Leerraum zusammengefasst): `ritzenhofer saalfelden` → `search-ritzenhofer-saalfelden.json`,
  `saalfelden` → `search-saalfelden.json`, sonst `[]`; `GET /reverse` → `reverse-saalfelden.json`.
  Jede `/search`- und `/reverse`-Anfrage wird protokolliert (`path`, `query`, `userAgent`, `at` ms).
  Steuerung (nur für Tests): `POST /__mock/reset` (Protokoll leeren, Modus `ok`),
  `POST /__mock/mode` mit `{"mode":"ok"|"error"}` (`error` → 503 auf `/search` und `/reverse`),
  `GET /__mock/requests` → Protokoll als JSON. Overlay `compose.e2e.yml`:
  `environment: PC_NOMINATIM_URL: http://host.docker.internal:18471` und
  `extra_hosts: ["host.docker.internal:host-gateway"]`. Abnahme: die bestehenden Flows
  (`auth`, `smoke`, `deployment`) laufen unverändert grün; das Verhalten des Mocks prüft der Flow
  aus D5.

- [ ] **D5 — Ortssuche-Oberfläche + E2E-Flow.** Nutzt die API `GET /api/geo/status|search|reverse`
  (Antworten: `{enabled, reason?}`, `{results:[Place]}`, `{place:Place|null}`, Fehler
  `{error}` mit 400/502/503; `Place` = `{name,address,kind,lat,lon,label,parts{sublocation,city,state,country,countryCode}}`)
  und den Mock auf `http://127.0.0.1:18471` mit `POST /__mock/reset`, `POST /__mock/mode`
  `{"mode":"ok"|"error"}`, `GET /__mock/requests` (`[{path,query,userAgent,at}]`).
  Dateien: `web/src/lib/geo.ts` (über den bestehenden Wrapper `web/src/lib/api.ts`:
  `fetchGeoStatus()`, `searchPlaces(q)`, `reversePlace(lat, lon)`; reine Funktion
  `geoNotice(status)` → `null` oder anzuzeigender Text = `reason`), `web/src/lib/geo.test.ts`,
  `web/src/lib/PlacePicker.svelte` (neu), `web/src/routes/Places.svelte` (neu, Seite „Ortssuche"),
  `web/src/App.svelte` (Route `/orte` mit dem dann vorhandenen Routing — Story 007 führt Ansichten
  ein; gibt es nur den Pfad-Schalter wie für `/login`, dort ergänzen), `web/src/lib/AppHeader.svelte`
  (Link „Ortssuche"), `web/tests/e2e/places.spec.ts` (neu).
  Picker (Svelte 5, deutsch, Touch-Flächen ≥ 44 px, ab 360 px nutzbar): Suchfeld (Label „Ort
  suchen") + Knopf „Suchen"; Anfrage **nur** bei Enter oder Knopf, nie bei Eingabe; Trefferliste
  (`role="list"`, max. 10) mit Name, Adresse, Art je Treffer; Klick auf einen Treffer füllt
  „Gewählter Ort": Feld „Ortsname" (editierbar, vorbelegt mit `label`), Felder „Breite"/„Länge",
  Anzeige Sublocation/City/State/Country. Knopf „Ort zum Punkt ermitteln" nutzt Breite/Länge →
  Reverse, Vorschlag füllt Ortsname und Ortsteile. Der Zustand Punkt + Name + Teile ist das
  Interface für Story 015 (dort setzt ein Kartenklick den Punkt): als Props/Callback von
  `PlacePicker` exportieren, z. B. `onchange(place)`. Fehler (400/502) → `role="alert"` mit der
  Server-Meldung; Breite/Länge und Ortsname bleiben eingebbar, „Gewählter Ort" zeigt Punkt und
  händischen Namen auch ohne Treffer. `status.enabled=false` → Suchfeld und Nominatim-Knöpfe
  gesperrt, Hinweis aus `geoNotice` sichtbar, manuelle Eingabe bleibt möglich.
  Vitest `web/src/lib/geo.test.ts` › „the place search notice explains why the search is disabled"
  (enabled → `null`; disabled → Text enthält „NOMINATIM_CONTACT").
  E2E `web/tests/e2e/places.spec.ts` › „place search finds a POI, names it and survives an outage"
  (zu Beginn `POST /__mock/reset`; Anmeldung mit `login(page, users.anna)` aus `helpers/auth.ts`):
  1. `/orte` öffnen, „Ritzenhofer Saalfelden" Zeichen für Zeichen tippen (`pressSequentially`) →
     Mock-Protokoll leer; Enter → genau eine `/search`-Anfrage mit `userAgent` =
     `PhotoCurator/<version> (e2e@example.invalid)` (Version aus `GET /api/health`); erster
     Treffer zeigt „Ritzenhofer", seine Adresse und „hotel"; höchstens 10 Treffer.
  2. Ersten Treffer wählen → „Ortsname" = „Ritzenhofer, Saalfelden", City „Saalfelden" und
     Country „Österreich" sichtbar; Namen ändern → Feld behält die Änderung.
  3. „  ritzenhofer   SAALFELDEN " per Knopf suchen → gleiche Treffer, Mock-Protokoll unverändert.
  4. Breite/Länge eines Punkts in Saalfelden eintragen, „Ort zum Punkt ermitteln" → Ortsname aus
     `reverse-saalfelden.json`; Protokoll enthält genau eine `/reverse`-Anfrage.
  5. Mock auf `error`, Suche nach nicht gecachtem „Zell am See" → Meldung „Ortssuche gerade nicht
     erreichbar…"; Breite/Länge + Ortsname „Hütte am Berg" von Hand → „Gewählter Ort" zeigt Punkt
     und Namen. Zum Schluss (auch bei Fehlschlag, `finally`/`afterEach`) Mock wieder auf `ok`.

## Model Hints

- D2 → deliverable-hard — der globale Limiter muss unter Nebenläufigkeit (Mutex, Kontext-Abbruch,
  fehlgeschlagene Anfragen zählen mit) zusammen mit dem Cache-Recheck nach dem Warten korrekt sein;
  ein subtiler Fehler (Zeitstempel nach statt vor dem Request, Slot bei Abbruch verbraucht,
  Doppelanfrage bei gleichzeitigen gleichen Suchen) verletzt die Nominatim-Richtlinie und riskiert
  die Sperre aus Risiko R8.
- Review: → default

## Acceptance Tests

- AC1 → e2e `web/tests/e2e/places.spec.ts` › "place search finds a POI, names it and survives an
  outage" (Schritt 1: keine Anfrage beim Tippen, Enter löst aus, POI als erster Treffer mit Name,
  Adresse, Art) + unit `internal/geo/place_test.go` › "search results keep Nominatim's order and
  stop at ten" (D1, D5)
- AC2 → unit `internal/geo/geocoder_test.go` › "five concurrent searches reach Nominatim at least
  one second apart"; `internal/api/geo_test.go` › "searches of two users share one limit of one
  request per second" (D2, D3) — Go-Test statt E2E: Abstand mit Fake-Uhr ohne Warten prüfbar,
  keine Nutzeraktion.
- AC3 → unit `internal/geo/geocoder_test.go` › "a search asks Nominatim with jsonv2, address and
  name details, German and ten results and our user agent", "without a contact the geocoder sends
  nothing"; `internal/api/geo_test.go` › "without a contact the place search reports why and sends
  nothing"; vitest `web/src/lib/geo.test.ts` › "the place search notice explains why the search is
  disabled"; e2e `web/tests/e2e/places.spec.ts` › "place search finds a POI, names it and survives
  an outage" (User-Agent am Mock) (D2, D3, D5) — Deaktiviert-Anzeige ohne E2E, siehe Entscheidung.
- AC4 → unit `internal/geo/geocoder_test.go` › "a repeated search differing only in case and
  spaces is answered from the cache", "a cached answer older than ninety days is fetched again";
  e2e `web/tests/e2e/places.spec.ts` › "place search finds a POI, names it and survives an outage"
  (Schritt 3) (D2, D5)
- AC5 → unit `internal/geo/geocoder_test.go` › "reverse geocoding shares the limiter and the cache
  with search"; e2e `web/tests/e2e/places.spec.ts` › "place search finds a POI, names it and
  survives an outage" (Schritt 4) (D2, D5)
- AC6 → unit `internal/geo/place_test.go` › "a POI hit becomes 'Ritzenhofer, Saalfelden' with its
  place parts", "a hit without its own name is labelled with the town alone"; e2e
  `web/tests/e2e/places.spec.ts` › "place search finds a POI, names it and survives an outage"
  (Schritt 2) (D1, D5)
- AC7 → unit `internal/geo/geocoder_test.go` › "an unreachable or failing Nominatim yields an
  upstream error and caches nothing"; `internal/api/geo_test.go` › "a Nominatim outage answers 502
  with a readable message"; e2e `web/tests/e2e/places.spec.ts` › "place search finds a POI, names
  it and survives an outage" (Schritt 5; Kartenklick = Koordinatenfelder bis Story 015, siehe
  Entscheidung) (D2, D3, D5)

## Done
