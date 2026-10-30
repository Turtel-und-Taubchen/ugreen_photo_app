---
id: 013
title: Zuletzt verwendete Tags und Autovervollständigung
status: done # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E3 Tags & Kategorien
estimate: 3
sprint: S03
---

## Requirement

Beim Taggen will ich nicht tippen müssen, was ich gerade eben schon verwendet habe, und bei
längeren Namen reichen ein paar Buchstaben — auch am Handy.

## Acceptance Criteria

- [x] **AC1** — Der Tag-Dialog zeigt die zuletzt verwendeten Tags (Platzhalter: 8) als
  antippbare Chips, der zuletzt verwendete zuerst.
- [x] **AC2** — Die Eingabe schlägt ab dem ersten Zeichen passende bekannte Tags vor —
  Teilstring auch innerhalb des Pfades, unabhängig von Groß-/Kleinschreibung und mit Umlauten
  (z. B. „öst" findet `Reise/Österreich`).
- [x] **AC3** — Bei 1.000 bekannten Tags erscheinen die Vorschläge ohne Wartezeit: Sie entstehen
  im Browser aus der beim Öffnen des Dialogs einmal geladenen Tag-Liste, Tippen löst keine
  Server-Anfrage aus, und ein Tastendruck normalisiert nur die Eingabe, nicht die 1.000 Tags
  (Nachweis per Zählung im Test, nicht per Stoppuhr). *(Refine: umformuliert von „innerhalb von
  100 ms (Messung im Test)" — siehe Decisions.)*
- [x] **AC4** — Enter oder Tippen auf „neu anlegen" übernimmt einen noch unbekannten Begriff als
  neuen Tag; die Validierung aus Story 011 gilt.
- [x] **AC5** — Bei 390 px Breite sind Chips und Vorschläge ohne Scrollen der Seite erreichbar
  und mindestens 44 px hoch.

## Open Questions

- ~~„Zuletzt verwendet": gemeinsam für beide Personen oder pro Person?~~ answered → Decisions (Sprint)

## Decisions (Sprint)

- **(User)** „Zuletzt verwendet" gemeinsam oder pro Person: **gemeinsam** — eine globale Liste für beide Nutzer.
- **AC3 wird per Zählung statt Stoppuhr bewiesen** (Kriterium umformuliert, Absicht unverändert) — `refine.md` verlangt Kosten-Nachweise für Geschwindigkeit, und das Projekt hat keine Perf-Lane, in der ein 100-ms-Budget warnen statt blockieren könnte.
- **Vorschläge entstehen im Client** aus `GET /api/tags` (Baum aus Index + Definitionen, Story 011), einmal beim Öffnen des Dialogs geladen und flach gemacht — keine Anfrage pro Tastendruck, kein neuer Such-Endpunkt; 1.000 Pfade sind für den Browser klein.
- **Abgleich = Teilstring auf dem ganzen Pfad nach `normalize('NFC')` + `toLocaleLowerCase('de')`**, ohne Akzent-Faltung („ost" findet `Österreich` nicht) — das AC-Beispiel verlangt nur Groß/Klein-Unabhängigkeit mit Umlauten; NFC fängt zerlegte Umlaute (macOS-Keywords) ab.
- **Reihenfolge der Vorschläge:** zuerst Pfade, bei denen ein Segment mit der Eingabe beginnt, dann übrige Treffer; innerhalb jeder Gruppe `Intl.Collator('de')` auf dem Pfad; höchstens 6 Vorschläge, bereits gewählte Tags ausgeblendet — so bleibt die Liste bei 390 × 844 ohne Seiten-Scrollen sichtbar (AC5).
- **„Neu anlegen" erscheint nur, wenn kein bekannter Pfad der Eingabe (getrimmt) normalisiert gleicht;** gleicht einer, übernimmt Enter diesen bestehenden Tag — verhindert Fast-Duplikate, die sich nur in Groß/Klein unterscheiden (Identität in 011 bleibt exakt).
- **Enter:** mit per Pfeiltaste markiertem Vorschlag → dieser Vorschlag; sonst bekannter Tag (siehe oben) oder neu anlegen — kein automatisches Vormarkieren, damit Enter einen neuen Begriff nicht still durch einen Vorschlag ersetzt (AC4).
- **„Neu anlegen" ruft `POST /api/tags` auf** (011, idempotent) und übernimmt den kanonischen Pfad aus der Antwort; ein 400 zeigt die Server-Erklärung als `role="alert"` — Validierung lebt nur in Go (011), und der Nutzer sieht den Fehler sofort im Dialog, nicht erst am Job.
- **Zuletzt verwendet wird serverseitig beim Annehmen eines „Tags zuweisen"-Jobs (012) erfasst**, nicht vom Client gemeldet — eine Quelle, keine verlierbare Zweitanfrage, und beide Nutzer sehen dieselbe Liste.
- **Nur Zuweisen zählt als Verwendung, Entfernen nicht;** die Chips stehen nur im Zuweisen-Dialog — der Entfernen-Dialog listet ohnehin die Tags der Auswahl (012 AC2).
- **Speicher `recent_tags(path PRIMARY KEY, seq)` in `app.db`, global, auf 8 Zeilen beschnitten;** `seq` ist eine streng steigende Zahl statt Zeitstempel, damit mehrere Tags eines Jobs eindeutig geordnet sind — der letzte im Job-Payload steht vorn.
- **Gelöschte Kategorien werden aus der Liste nicht gefiltert** — ein Chip ist eine Abkürzung für einen Begriff, und Zuweisen legt ihn ohnehin wieder an (012 AC5).
- **Antippen eines Chips oder Vorschlags fügt den Tag der Auswahl des Dialogs hinzu** (Toggle bei Chips), es weist nicht sofort zu — 012 weist einen oder mehrere Tags in einem Job zu.
- **Der Fehlertext-Helfer wird jetzt zusammengeführt** (`errorMessage` in `web/src/lib/api.ts`) — die Form existiert schon dreimal (Login, Tags, PlacePicker; Follow-up aus S02), eine vierte Kopie wäre ein Refine-Fehler.
- **Prüf-Viewport 390 × 844 mit `hasTouch`** — dieselbe Handy-Größe wie die bestehenden E2E-Flows (`selection.spec.ts`).

## Plan

Baut auf 011 (`GET/POST /api/tags`, `tags.ParsePath`) und 012 (Dialog „Tags zuweisen",
Job-Anlage über `POST /api/jobs`) auf. Schreibt selbst nichts in Dateien.

1. **D1 — Speicher (Go):** Migration `recent_tags` in `app.db`, `tags.TouchRecent` /
   `tags.ListRecent` in `internal/tags/recent.go`.
2. **D2 — Erfassen + API (Go):** `GET /api/tags/recent`; der Job-Handler aus 012 ruft
   `TouchRecent` nach dem Annehmen eines Zuweisen-Jobs.
3. **D3 — Vorschlagslogik (TS):** `web/src/lib/tagSuggest.ts` — Baum flach machen, Index einmal
   normalisieren, `suggest(query)`; `fetchRecentTags()` in `tags.ts`; Zähl-Test für AC3;
   `errorMessage`-Helfer in `api.ts` zusammenführen (3 Kopien).
4. **D4 — Oberfläche + E2E:** `web/src/lib/TagPicker.svelte` (Chips, Combobox, „neu anlegen"),
   eingebaut in den Zuweisen-Dialog aus 012; ein E2E-Flow bei 390 × 844.

Reihenfolge D1 → D2 → D3 → D4 (D3 ist unabhängig von D1/D2 und darf parallel laufen).
`docs/systems/` ist leer — kein Systems-Doc zu pflegen.

## Deliverables

- [x] **D1 — `recent_tags` in `app.db` mit Erfassen und Auflisten.** Dateien:
  `internal/store/appdb.go` (Migration als nächsten Eintrag an `migrations` **anhängen**, nie einen
  bestehenden ändern), neu `internal/tags/recent.go`, neu `internal/tags/recent_test.go`. Vorbild
  für DB-Zugriff und Tests: `internal/tags/definitions.go` / `definitions_test.go`.
  - Migration: `CREATE TABLE recent_tags (path TEXT PRIMARY KEY, seq INTEGER NOT NULL)` — global,
    ohne Nutzerspalte (beide Nutzer teilen eine Liste).
  - `const RecentLimit = 8`.
  - `TouchRecent(ctx, db *sql.DB, paths []string) error`: in **einer** Transaktion jeden Pfad in
    Payload-Reihenfolge upserten (`INSERT … ON CONFLICT(path) DO UPDATE SET seq = excluded.seq`) mit
    `seq = (SELECT COALESCE(MAX(seq), 0) FROM recent_tags) + 1`, d. h. jeder Pfad bekommt eine
    streng größere Zahl als der vorige; danach alles außer den `RecentLimit` höchsten `seq`
    löschen. Pfade kommen bereits kanonisch (aus `tags.ParsePath`); leere Liste = no-op.
  - `ListRecent(ctx, db) ([]string, error)`: `ORDER BY seq DESC LIMIT RecentLimit`; leer → `[]`.
  - Test `"the most recently used tag comes first and only eight are kept"`
    (`TestTheMostRecentlyUsedTagComesFirstAndOnlyEightAreKept`): 10 Pfade einzeln berühren →
    8 Einträge, neuester vorn; einen alten erneut berühren → er steht vorn, kein Duplikat;
    ein Aufruf mit `[A, B]` → `B` vor `A`. Der bestehende Migrationstest in
    `internal/store/appdb_test.go` bleibt grün.

- [x] **D2 — `GET /api/tags/recent` und Erfassen beim Zuweisen-Job.** Dateien:
  `internal/api/tags.go` (Route in `registerTags`, nutzt `TagDeps.App`), `internal/api/tags_test.go`,
  der Handler, der in Story 012 den Job „Tags zuweisen" annimmt (`POST /api/jobs`; Datei und
  Aktionsname stehen in 012 unter `## Deliverables`/`## Done`, vermutlich `internal/api/jobs.go`),
  ggf. die Verdrahtung in `internal/api/server.go` / `cmd/photocurator/main.go`, falls der
  Job-Handler `app.db` noch nicht kennt. Nutzt `tags.TouchRecent/ListRecent` (D1).
  - `GET /api/tags/recent` → 200 `{"tags": ["Wir zwei", "Reise/Österreich", …]}` (neuester zuerst,
    leere Liste als `[]`), hinter der bestehenden Session-/CSRF-Schicht wie `GET /api/tags`.
  - Im 012-Handler: **nachdem** ein Job der Aktion „Tags zuweisen" angenommen wurde (2xx), mit
    den kanonischen Tag-Pfaden des Jobs in Payload-Reihenfolge `tags.TouchRecent` aufrufen. Ein
    Fehler dabei wird mit `slog` geloggt und **nicht** an den Client gegeben (der Job läuft
    schon). „Tags entfernen" und andere Aktionen berühren die Liste nicht.
  - Test `"assigning tags makes them recent for both users"`
    (`TestAssigningTagsMakesThemRecentForBothUsers`) über `NewHandler` (Session-Muster aus
    `server_test.go`/`tags_test.go`, Job-Aufbau aus den 012-Tests): Nutzer A legt einen
    Zuweisen-Job mit `[Reise/Österreich, Wir zwei]` an → Nutzer B bekommt von
    `GET /api/tags/recent` `["Wir zwei", "Reise/Österreich"]`; ein Entfernen-Job mit einem
    anderen Tag ändert die Liste nicht.

- [x] **D3 — Vorschlagslogik im Client und Fehler-Helfer.** Dateien: neu
  `web/src/lib/tagSuggest.ts`, neu `web/src/lib/tagSuggest.test.ts`, `web/src/lib/tags.ts`
  (`fetchRecentTags()` ergänzen, `createTag` erweitern; Typ `TagNode` und `fetchTags()` dort
  wiederverwenden), `web/src/lib/api.ts`, `web/src/routes/Tags.svelte`,
  `web/src/lib/PlacePicker.svelte`, `web/src/routes/Login.svelte`.
  - `fetchRecentTags(): Promise<string[]>` → `api<{tags: string[] | null}>('/api/tags/recent')`,
    `null` → `[]`.
  - `createTag(path)` gibt den kanonischen Pfad aus der Antwort zurück (`{path}` bei 201/200);
    der bestehende Aufrufer in `Tags.svelte` ignoriert ihn weiter.
  - Neu in `api.ts`: `errorMessage(e: unknown, fallback = 'Anfrage fehlgeschlagen'): string` =
    `e instanceof ApiError && e.body?.error ? e.body.error : fallback`. Die lokalen
    `message()`-Kopien in `Tags.svelte` und `PlacePicker.svelte` sowie den gleichen
    Inline-Ausdruck in `Login.svelte` (Fallback `'Anmeldung fehlgeschlagen.'`) dadurch ersetzen;
    hat Story 012 eine weitere Kopie angelegt, auch diese. Verhalten unverändert — die
    bestehenden E2E-Flows `tags.spec.ts`, `places.spec.ts`, `auth.spec.ts` bleiben grün.
  - `buildTagIndex(tree: TagNode[]): TagIndex` — alle Knotenpfade (auch Eltern wie `Anlass`)
    flach, jeder Pfad **einmal** mit `normalize('NFC')` + `toLocaleLowerCase('de')` vorberechnet,
    zusätzlich die normalisierten Segment-Anfänge.
  - `suggest(index, query, exclude: string[] = [], limit = 6): string[]` — `query` wird getrimmt
    und genau einmal normalisiert; leer → `[]`. Treffer = normalisierter Pfad enthält die
    normalisierte Eingabe. Gruppe 1: ein Segment beginnt mit der Eingabe; Gruppe 2: übrige.
    Innerhalb jeder Gruppe `Intl.Collator('de')` auf dem Original-Pfad. Pfade aus `exclude`
    (schon gewählt) fallen weg. Keine Akzent-Faltung.
  - `findExact(index, input): string | undefined` — bekannter Pfad, der der getrimmten Eingabe
    normalisiert gleicht (für „neu anlegen" ja/nein und Enter).
  - Tests: `"suggestions match anywhere in the path regardless of case and umlauts"` („öst",
    „ÖST", „reich" finden `Reise/Österreich`; „ost" nicht; zerlegtes `Öst` findet ihn;
    „geb" listet `Anlass/Geburtstag` vor `Wir/Lebensgeburt` (Segmentanfang vor Teilstring);
    gewählte Pfade fallen weg; höchstens 6) und
    `"a keystroke over 1,000 tags lowercases only the query"`: Index aus 1.000 Pfaden bauen, dann
    `vi.spyOn(String.prototype, 'toLocaleLowerCase')` **und** `vi.spyOn(String.prototype,
    'normalize')` setzen und `suggest(index, 'öst')` aufrufen → jeder Spy ≤ 1 Aufruf; Ergebnis
    korrekt. (Der Spy auf dem Prototyp fängt auch Inline-Aufrufe — ein Zähler nur auf einer
    eigenen Hilfsfunktion wäre umgehbar.) Kein Zeitbudget im Test.

- [x] **D4 — `TagPicker` im Zuweisen-Dialog und E2E-Flow.** Dateien: neu
  `web/src/lib/TagPicker.svelte`, der Zuweisen-Dialog aus Story 012 (Datei siehe 012
  `## Deliverables`/`## Done`; dessen freies Tag-Eingabefeld wird durch `TagPicker` ersetzt), neu
  `web/tests/e2e/tag-quick-select.spec.ts`. Nutzt aus `web/src/lib/tags.ts` `fetchTags`,
  `fetchRecentTags` und `createTag` (gibt den kanonischen Pfad zurück), aus
  `web/src/lib/tagSuggest.ts` `buildTagIndex(tree)`, `suggest(index, query, exclude)` (höchstens
  6, schon sortiert) und `findExact(index, input)` (bekannter Pfad, der der Eingabe Groß/Klein-
  unabhängig gleicht, sonst `undefined`), aus `web/src/lib/api.ts` `errorMessage(e)` für
  Fehlertexte.
  - `TagPicker` (Props: `selected: string[]` bindbar): lädt beim Mounten **einmal** Baum
    (`fetchTags`) und Recents (`fetchRecentTags`), baut den Index einmal; danach keine Anfragen
    beim Tippen.
  - Oben „Zuletzt verwendet": Recents als Chips (`<button aria-pressed>`), Reihenfolge wie vom
    Server, Antippen schaltet den Tag in `selected` ein/aus. Keine Recents → Abschnitt fehlt.
  - Darunter Eingabe mit Label „Tag suchen oder neu anlegen" als ARIA-Combobox
    (`role="combobox"`, `aria-expanded`, `aria-controls`, `aria-activedescendant`); Liste
    `role="listbox"` mit `role="option"` je Vorschlag aus `suggest(index, query, selected)`, ab
    dem ersten Zeichen. Pfeil hoch/runter markiert, Escape schließt. Ohne `findExact`-Treffer
    und bei nicht leerer Eingabe zusätzlich eine Option „„<Eingabe>“ neu anlegen".
  - Enter: markierte Option → übernehmen; sonst `findExact`-Treffer → diesen übernehmen; sonst wie
    „neu anlegen". Übernehmen = Pfad an `selected` anhängen (Reihenfolge der Wahl), Eingabe leeren.
    „Neu anlegen" = `createTag(input)`; Erfolg → zurückgegebenen kanonischen Pfad übernehmen und
    in den Index aufnehmen (Index neu bauen ist bei einem Treffer pro Anlegen in Ordnung);
    400 → `errorMessage(e)` als `role="alert"` unter dem Feld, Eingabe bleibt stehen.
  - Gewählte Tags als Chips mit Entfernen-Knopf („<Pfad> entfernen") — sie gehen als Tag-Liste
    in den Job des 012-Dialogs (in dieser Reihenfolge).
  - Layout ab 360 px: Chips umbrechen (`flex-wrap`), alle Chips, Optionen und Knöpfe
    `min-height: 44px`; die Vorschlagsliste liegt im Fluss direkt unter dem Feld (kein
    Overlay, das aus dem Viewport ragt), höchstens 6 Vorschläge + „neu anlegen".
  - E2E-Flow `"the tag dialog offers recent tags and suggestions and creates new tags on a phone"`
    mit `test.use({ viewport: { width: 390, height: 844 }, hasTouch: true })` (Vorbild:
    `web/tests/e2e/selection.spec.ts`; Login über `helpers/auth.ts`; Auswahl und Öffnen des
    Dialogs wie im E2E-Flow von 012): per `page.request.post('/api/tags', …)` mit Header
    `X-Requested-With` `Reise/Österreich` anlegen → ein Foto wählen, „Tags zuweisen" öffnen →
    Request-Zähler (`page.on('request')` auf `/api/`) nullen → `öst` tippen → Option
    `Reise/Österreich` sichtbar, Zähler weiterhin 0 → Option antippen → `Wir zwei` tippen
    (vorher sicherstellen, dass es nicht existiert, sonst eindeutigen Namen wählen) + Enter →
    gewählt → `Neu|Tag` tippen, „neu anlegen" antippen → Alert nennt `|` → zuweisen, Job-Ende
    abwarten (wie 012) → Dialog erneut öffnen → erster Chip `Wir zwei`, zweiter
    `Reise/Österreich` → für jeden Chip und, nach `rei` tippen, jede Option: `boundingBox()`
    liegt vollständig in 0…844 und ist ≥ 44 hoch, und `window.scrollY === 0` → Chip
    `Reise/Österreich` antippen → `aria-pressed="true"`.

## Model Hints

Kein `deliverable-hard`: die heikelsten Punkte (Reihenfolge per `seq`, Aufrufzählung pro
Tastendruck, Combobox-Enter-Regeln) sind klein und durch benannte Tests fixiert.

Review: → default

## Acceptance Tests

- AC1 → unit `internal/tags/recent_test.go` › "the most recently used tag comes first and only eight are kept"
- AC1 → unit `internal/api/tags_test.go` › "assigning tags makes them recent for both users"
- AC1 → e2e `web/tests/e2e/tag-quick-select.spec.ts` › "the tag dialog offers recent tags and suggestions and creates new tags on a phone"
- AC2 → unit `web/src/lib/tagSuggest.test.ts` › "suggestions match anywhere in the path regardless of case and umlauts"
- AC2 → e2e `web/tests/e2e/tag-quick-select.spec.ts` › "the tag dialog offers recent tags and suggestions and creates new tags on a phone"
- AC3 → unit `web/src/lib/tagSuggest.test.ts` › "a keystroke over 1,000 tags lowercases only the query"
- AC3 → e2e `web/tests/e2e/tag-quick-select.spec.ts` › "the tag dialog offers recent tags and suggestions and creates new tags on a phone" (keine `/api/`-Anfrage beim Tippen)
- AC4 → e2e `web/tests/e2e/tag-quick-select.spec.ts` › "the tag dialog offers recent tags and suggestions and creates new tags on a phone"
- AC4 → unit `internal/tags/path_test.go` › "invalid tag names are rejected with a reason" (bestehend, 011)
- AC5 → e2e `web/tests/e2e/tag-quick-select.spec.ts` › "the tag dialog offers recent tags and suggestions and creates new tags on a phone"

Go-Tests tragen den Satz als Funktionsnamen in CamelCase (Muster aus `internal/store/appdb_test.go`);
Vitest- und Playwright-Tests tragen ihn wörtlich.

Abdeckung: AC1 → D1+D2+D4 · AC2 → D3+D4 · AC3 → D3+D4 · AC4 → D4 (+011) · AC5 → D4.

## Done

Zuletzt verwendete Tags (global, serverseitig beim Zuweisen-Job erfasst) und Autovervollständigung im Zuweisen-Dialog: `recent_tags`, `GET /api/tags/recent`, clientseitige Vorschlagslogik, `TagPicker` ersetzt `TagListInput`.

Change type: feat

Commit: `013: recent tags + tag autocomplete — recent_tags store/API, client-side suggest, TagPicker in assign dialog + e2e`

Verification (narrow gate; `test-story: none` → full `test`): build, `docker build --target test` + vitest (54), lint, `svelte-check` grün; e2e `npx playwright test tests/e2e/tag-quick-select.spec.ts` grün. Review 1 (default): UNCLEAR → Test-Lücke (Segmentanfang-Reihenfolge war tautologisch) behoben, Unit-Lauf grün. Voller Gate offen (Sprint).
AC → Test: AC1 → recent_test.go, tags_test.go (Go-Pakete `ok`, Einzelnamen ohne `-v` nicht gelistet), e2e · AC2/AC3 → tagSuggest.test.ts, e2e (Request-Zähler 0) · AC4 → e2e + path_test.go (011) · AC5 → e2e. Kein manual residue.

Decisions:
- 012 hatte `errorMessage` schon zusammengeführt; D3 ergänzte nur den `fallback`-Parameter und Login.svelte.
- `TagListInput.svelte` gelöscht (nur der Dialog nutzte ihn); `batch-tagging.spec.ts` auf die Combobox umgestellt, nicht abgeschwächt.
- Neu angelegter Tag kommt flach in den Index (Eltern erst nach erneutem Öffnen vorgeschlagen); nicht geändert.
- Offen: das e2e lässt zugewiesene Tags (`Wir zwei <ts>`, `Reise/Österreich`) an `tagged.jpg` zurück (Löschen von Tags mit Fotos wird abgelehnt) — der Sprint-Gate zeigt, ob andere Flows darüber stolpern. Nicht verifiziert: Pfeiltasten/Escape/Enter-auf-bekannten-Tag im UI (nur Unit-Logik).
- e2e-Image: der Stack nutzt `photocurator:dev`; vor e2e neu bauen (`build`-Befehl).

tiers: D 4 / hard 0 · review default · cycles 1 · agents 6
