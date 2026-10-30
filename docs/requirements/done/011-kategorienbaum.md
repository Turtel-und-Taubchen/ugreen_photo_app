---
id: 011
title: Hierarchische Kategorien definieren und als Baum anzeigen
status: done
created: 2026-10-29
epic: E3 Tags & Kategorien
estimate: 3
sprint: S02
---

## Requirement

Ich möchte meine eigene Ordnung festlegen — z. B. `Anlass/Geburtstag`, `Reise/Österreich`,
`Wir zwei` — und sie als Baum sehen, auch bevor ein Foto die Kategorie trägt. Tags und
Kategorien sind dasselbe Konzept: Eine Kategorie ist ein Tag mit Kindern.

## Acceptance Criteria

- [x] **AC1** — Ein neuer Tag kann mit Pfad angelegt werden (`Anlass/Geburtstag`); fehlende
  Elternknoten entstehen automatisch.
- [x] **AC2** — Die Tag-Verwaltung zeigt alle bekannten Tags als Baum — aus den Dateien (Index)
  und den angelegten, noch leeren Kategorien — jeweils mit Anzahl der Fotos.
- [x] **AC3** — Ein Tag ohne Fotos kann gelöscht werden; für einen Tag mit Fotos ist Löschen
  gesperrt, mit dem Hinweis, ihn zuerst über „Tags entfernen" von den Fotos zu lösen.
- [x] **AC4** — Ungültige Namen werden mit Erklärung abgewiesen: leere Segmente, Zeichen, die in
  den Zielfeldern als Trenner dienen (`|`, `;`, `,`, `/` innerhalb eines Segments), und Segmente
  über der Längengrenze der Zielfelder (IPTC: 64 Byte).
- [x] **AC5** — Nach einem Index-Neuaufbau sind alle in Dateien vorhandenen Tags wieder im Baum;
  leere Kategorien bleiben erhalten (App-Zustand).

## Open Questions

Keine offen.

## Decisions (Sprint)

- **Segmente werden getrimmt**, ein nach dem Trimmen leeres Segment (auch `Anlass//Geburtstag`, führender/abschließender `/`) wird abgewiesen — unsichtbare Leerzeichen erzeugen sonst scheinbar doppelte Tags.
- **Verbotene Zeichen in einem Segment:** `|` (Trenner in `HierarchicalSubject`), `;` (Trenner in `XPKeywords`), `,` (Listentrenner von Keyword-Feldern) und Steuerzeichen inkl. Zeilenumbruch (Tech Spec §9: Zeilenumbrüche in ExifTool-Argumentdateien werden abgewiesen); `/` ist der Pfadtrenner und kann darum nie innerhalb eines Segments stehen.
- **Längengrenze 64 Byte UTF-8 je Segment**, keine Grenze für den Gesamtpfad — `IPTC:Keywords` erhält laut Schreibstrategie nur Einzelbegriffe, `HierarchicalSubject` (XMP) hat keine Grenze.
- **Identität ist der exakte String:** keine Unicode-Normalisierung, keine Groß-/Kleinschreibungs-Faltung — Keywords in den Dateien sind ebenso exakt, und es braucht keine neue Abhängigkeit (`golang.org/x/text`).
- **Anlegen eines schon bekannten Pfads ist idempotent** (200, kein Fehler) — der Nutzer will den Knoten haben, und er existiert.
- **Jeder Knoten ist eine eigene Zeile in `tag_definitions`**, auch automatisch angelegte Eltern — so überlebt `Anlass` das Löschen von `Anlass/Geburtstag`, wie der Nutzer ihn angelegt gesehen hat.
- **Kategorien sind global für beide Nutzer** — die Tags stehen in gemeinsamen Dateien.
- **Anzahl = verschiedene Fotos, die den Tag oder einen Nachfahren tragen** — so zeigt `Anlass` den Umfang der Kategorie, ein Foto mit zwei Kindern zählt einmal, und die Löschsperre folgt derselben Zahl.
- **Löschen nur bei Anzahl 0 (Teilbaum)**; entfernt den Knoten samt leerer Nachfahren-Definitionen nur aus `app.db`, nie aus Dateien; der Server prüft selbst (409), die UI-Sperre ist nur Komfort.
- **Tags aus dem Index werden unverändert angezeigt**, auch wenn sie die Regeln aus AC4 verletzen — die Validierung gilt nur für neue Eingaben, fremde Keywords bleiben erhalten (PC-13).
- **Validierungsregeln leben nur in Go** (`tags.ParsePath`); die UI zeigt die Fehlermeldung des Servers — eine Quelle der Wahrheit, die 012/013 wiederverwenden.
- **API:** `GET /api/tags` (Baum), `POST /api/tags` `{path}`, `DELETE /api/tags?path=…` — Tech Spec §6 nennt `/api/tags`; der Pfad enthält `/` und steht darum im Query-Parameter.
- **UI:** eigene Seite `/tags` („Tags") mit Link im Header; Sortierung im Client mit `Intl.Collator('de')` — Umlaute stehen dann bei ihrem Grundbuchstaben.
- **Index-Abfragen ohne `LIKE`:** Teilbaum-Abgleich über `substr` — `_` und `%` sind gültige Tag-Zeichen und dürfen nicht als Platzhalter wirken.
- **AC5 wird per Go-Integrationstest bewiesen, nicht per E2E** — es ist eine Systemeigenschaft ohne Nutzeraktion; ein E2E-Neuaufbau bräuchte einen Container-Neustart mitten im Lauf.
- **E2E nutzt eine getaggte Fixture aus Story 005**; nur wenn 005 keine Datei mit `XMP-lr:HierarchicalSubject` liefert, legt D4 eine an — zusätzliche Dateien in `testdata/library` verändern Zählungen der Stories 005–008.
- Hinweis: Im Repository gibt es (noch) keine `CLAUDE.md`; die Leitplanken stammen aus `.claude/ai-scrum.md` (Notes) und der Tech Spec.

## Plan

Baut auf dem Index aus 005 auf (`index.db`, Tabelle `photo_tags(photo_id, tag_path)`, Pfade mit
`/`, Paket `internal/library`). Schreibt nichts in Dateien.

1. **D1 — Domäne (`internal/tags`):** `ParsePath` (Trimmen, Segmentregeln, 64 Byte, Fehlertexte
   auf Deutsch) und `BuildTree` (Vereinigung aus Index-Pfaden und Definitionen, implizite Eltern,
   Teilbaum-Zählung verschiedener Fotos). Reine Funktionen, Tabellentests.
2. **D2 — Persistenz:** Migration `tag_definitions` in `app.db` (angehängt), Anlegen (mit Eltern,
   eine Transaktion), Auflisten, Teilbaum-Löschen; in `internal/library` zwei Index-Abfragen:
   Paare `(photo_id, tag_path)` sortiert nach `photo_id` und Teilbaum-Anzahl (ohne `LIKE`).
3. **D3 — API:** `GET/POST/DELETE /api/tags`, Verdrahtung in `NewHandler` und `cmd/photocurator`;
   400 mit Erklärung, 409 bei Fotos; Integrationstest „Baum überlebt Index-Neuaufbau" mit echtem
   ExifTool.
4. **D4 — Oberfläche:** Seite `/tags` mit Baum, Anzahl, Anlegen-Feld, Löschen-Knopf (gesperrt +
   Hinweis), Link im Header; ein E2E-Flow über alle Nutzerkriterien.

Reihenfolge D1 → D2 → D3 → D4. Systems-Doc: falls 005 unter `docs/systems/` ein Index-Doc
angelegt hat, ergänzt D2 dort die Tag-Abfragen und `tag_definitions`.

## Deliverables

- [x] **D1 — Tag-Pfade validieren und Baum bauen (`internal/tags`).** Neue Dateien
  `internal/tags/path.go`, `internal/tags/tree.go`, Tests `internal/tags/path_test.go`,
  `internal/tags/tree_test.go`. Reines Go, keine DB.
  - `ParsePath(input string) (string, error)`: an `/` splitten, jedes Segment mit
    `strings.TrimSpace` trimmen, mit `/` wieder zusammensetzen (kanonische Form:
    `" Anlass / Geburtstag "` → `Anlass/Geburtstag`). Abweisen mit konkreter deutscher Erklärung
    (nennt Segment und Grund): leere Eingabe; leeres Segment (auch durch führenden,
    abschließenden oder doppelten `/`); Segment enthält `|`, `;` oder `,` (Meldung nennt das
    Zeichen und dass es in den Metadatenfeldern als Trenner dient); Segment enthält ein
    Steuerzeichen (`unicode.IsControl`, z. B. Zeilenumbruch, Tab); Segment länger als 64 Byte
    UTF-8 (Meldung „höchstens 64 Byte (IPTC)"; Umlaute zählen 2 Byte). Keine
    Unicode-Normalisierung, keine Case-Faltung. Exportiere `MaxSegmentBytes = 64`.
  - `BuildTree(defs []string, pairs func(yield func(photoID int64, path string) bool)) []Node`
    mit `Node{Name, Path string; Count int; Children []Node}`: Knoten = alle Pfade aus `defs`
    und aus den Paaren plus alle Vorfahren. `Count` = Anzahl **verschiedener** Fotos, die den
    Knoten oder einen Nachfahren tragen. Die Paare kommen nach `photo_id` sortiert; zähle pro
    Knoten mit „zuletzt gezählte photo_id" statt Mengen (O(Zeilen), wenig Speicher). Pfade aus
    dem Index werden **nicht** validiert, nur an `/` gesplittet (leere Segmente überspringen).
    Kinder in Byte-Reihenfolge (der Client sortiert für die Anzeige).
  - Tests: Tabellentest `"invalid tag names are rejected with a reason"` (leer, `Anlass//X`,
    `/Anlass`, `A|B`, `A;B`, `A,B`, Zeilenumbruch, 65-Byte-Segment, 33×`ä` = 66 Byte; gültig:
    `Wir zwei`, `Reise/Österreich`, genau 64 Byte, Trimmen);
    `"a photo carrying two tags under one parent counts once for the parent"`;
    `"an empty definition appears with count zero and its parents"`.

- [x] **D2 — `tag_definitions` in `app.db` und Tag-Abfragen auf dem Index.** Dateien:
  `internal/store/appdb.go` (Migration als nächsten Eintrag an `migrations` **anhängen**, nie einen
  bestehenden ändern), `internal/store/appdb_test.go`, neu `internal/tags/definitions.go` +
  `internal/tags/definitions_test.go`, neu `internal/library/tagquery.go` +
  `internal/library/tagquery_test.go` (Paket und Index-Öffnen aus Story 005 wiederverwenden,
  keinen zweiten Index-Öffner bauen).
  - Migration: `CREATE TABLE tag_definitions (path TEXT PRIMARY KEY, created_at INTEGER NOT NULL)`
    — global, ohne Nutzerspalte.
  - `tags.Create(ctx, db *sql.DB, path string) (created bool, err error)`: `path` ist bereits
    kanonisch (aus `ParsePath`); legt Pfad und alle Vorfahren in **einer** Transaktion mit
    `INSERT OR IGNORE` an; `created=false`, wenn der Pfad schon existierte.
    `tags.List(ctx, db) ([]string, error)`. `tags.DeleteSubtree(ctx, db, path) (int64, error)`
    löscht `path` und alle Zeilen darunter mit
    `path = ? OR substr(path, 1, length(?) + 1) = ? || '/'` — **kein `LIKE`** (`_`/`%` sind
    gültige Tag-Zeichen).
  - `library.TagPairs(ctx, yield)` streamt `SELECT photo_id, tag_path FROM photo_tags ORDER BY
    photo_id` (Signatur passend zu `tags.BuildTree`); `library.TagSubtreeCount(ctx, path) (int, error)`
    = `COUNT(DISTINCT photo_id)` mit demselben `substr`-Abgleich statt `LIKE`.
  - Falls Story 005 ein Systems-Doc unter `docs/systems/` für den Index angelegt hat: dort einen
    kurzen Absatz zu `tag_definitions` und den beiden Abfragen ergänzen.
  - Tests: `"creating a path also creates its missing parents"` (`Anlass/Geburtstag` → Zeilen
    `Anlass` und `Anlass/Geburtstag`; zweites Anlegen → `created=false`);
    `"deleting a category removes its empty descendants but not its parent or a lookalike"`
    (`A`, `A/B`, `A/B/C`, `A_B` angelegt; Löschen von `A/B` lässt `A` und `A_B` stehen);
    `"subtree counts treat underscore and percent literally"` (Index-Zeilen `A_b/x`, `Axb/y`,
    `A%/z` → Anzahl für `A_b` ist 1). Der bestehende Migrationstest in `appdb_test.go` bleibt grün.

- [x] **D3 — API `/api/tags` und Verdrahtung.** Dateien: neu `internal/api/tags.go`,
  `internal/api/tags_test.go`; `internal/api/server.go` (Routen in `NewHandler`; Abhängigkeiten
  analog `AuthDeps` übergeben), `cmd/photocurator/main.go` (App-DB und Index-Handle übergeben).
  Nutzt `tags.ParsePath/Create/List/DeleteSubtree/BuildTree` und
  `library.TagPairs/TagSubtreeCount` (D1/D2).
  - `GET /api/tags` → 200 `{"tags": [Node…]}` (JSON-Felder `name`, `path`, `count`, `children`;
    leere Kinder als `[]`).
  - `POST /api/tags` Body `{"path": "…"}` → `ParsePath`; Fehler → 400 `{"error": "<Erklärung>"}`;
    neu → 201 `{"path": "<kanonisch>"}`; schon vorhanden → 200 dasselbe.
  - `DELETE /api/tags?path=…` → `TagSubtreeCount > 0` → 409
    `{"error": "Dieser Tag ist noch N Fotos zugeordnet. Entferne ihn zuerst über „Tags entfernen“ von den Fotos.", "count": N}`;
    Pfad weder definiert noch im Index → 404; sonst `DeleteSubtree` → 204. Der Server prüft
    immer selbst, unabhängig von der UI. Dateien unter `/photos` werden nie berührt.
  - Alle Routen hinter der bestehenden Session- und CSRF-Schicht (nichts als offen eintragen).
  - Tests über `NewHandler` mit angemeldeter Session (Muster aus `server_test.go`/`auth_test.go`):
    `"deleting a tag that photos carry is refused"` (409, Definition bleibt),
    `"an empty category can be deleted"` (204, danach nicht mehr im Baum),
    `"an invalid name is rejected with its reason"` (400, Text enthält `|`).
    Integrationstest mit echtem ExifTool (läuft in `docker build --target test`)
    `"the tag tree survives an index rebuild"`: Fixture-Kopie (`testsupport.CopyFixture`) mit der
    getaggten Fixture aus Story 005 (`XMP-lr:HierarchicalSubject`), Scan mit dem Scanner aus 005 in
    ein frisches `index.db`, `Anlass/Geburtstag` per POST anlegen, `GET /api/tags` merken,
    `index.db` löschen, neu scannen → `GET /api/tags` ist identisch (Datei-Tag mit Anzahl ≥ 1,
    `Anlass/Geburtstag` mit Anzahl 0).

- [x] **D4 — Seite „Tags" mit Baum, Anlegen und Löschen.** Dateien: neu
  `web/src/routes/Tags.svelte`, neu `web/src/lib/tags.ts` (Aufrufe über `api()` aus
  `web/src/lib/api.ts`, Typ `TagNode`, Sortierung mit `Intl.Collator('de')`), `web/src/App.svelte`
  (Route `/tags` im Routing, das Story 007 hinterlassen hat; ohne Router wie bisher über
  `location.pathname`), `web/src/lib/AppHeader.svelte` (Link „Tags", ≥ 44 px), neu
  `web/tests/e2e/tags.spec.ts` (Login über `web/tests/e2e/helpers/auth.ts`).
  - API: `GET /api/tags` → `{tags: TagNode[]}` mit `{name, path, count, children}`;
    `POST /api/tags` `{path}` → 201/200, 400 `{error}`; `DELETE /api/tags?path=<encodeURIComponent>`
    → 204, 409 `{error, count}`.
  - Seite: Überschrift „Tags"; Eingabefeld mit Label „Neuer Tag" + Knopf „Anlegen" (auch Enter);
    Fehlertext des Servers (400) als `role="alert"` unter dem Feld; Baum als verschachtelte Liste
    mit Einrückung, jeder Knoten zeigt Name und Anzahl (`Geburtstag (0)`), Kinder aufgeklappt.
    Löschen-Knopf je Knoten mit zugänglichem Namen „<Name> löschen": bei Anzahl 0 aktiv (danach
    Baum neu laden), bei Anzahl > 0 `disabled` und daneben sichtbar der Hinweis „Zuerst über
    „Tags entfernen“ von den Fotos lösen". Eine 409-Antwort (Wettlauf) zeigt deren Text. Ab 360 px
    Breite nutzbar, Bedienelemente ≥ 44 px.
  - Fixture: der E2E-Lauf kopiert `testdata/library`. Nutze die getaggte Fixture aus Story 005.
    Nur falls dort keine Datei `XMP-lr:HierarchicalSubject` trägt: lege
    `testdata/library/2023/05/tagged.jpg` an (Kopie von `sample.jpg`, im Wegwerf-Container mit
    ExifTool `-XMP-lr:HierarchicalSubject=Reise|Österreich -XMP-dc:Subject=Reise
    -XMP-dc:Subject=Österreich`), dokumentiere sie in `testdata/README.md` und lass danach die
    komplette `e2e`-Suite laufen (andere Stories zählen Fixture-Dateien).
  - E2E-Flow `"the tag page shows file tags and empty categories as a tree and guards deletion"`:
    anmelden → Link „Tags" → Datei-Tag der Fixture mit Anzahl ≥ 1 sichtbar (auf Scan-Ende warten:
    `expect(...).toBeVisible({ timeout })`, ggf. Reload) → `Anlass/Geburtstag` anlegen → `Anlass`
    mit Kind `Geburtstag (0)` sichtbar → `Anlass/Ge|burtstag` anlegen → Alert nennt `|` →
    „Geburtstag löschen" → Knoten weg, `Anlass` bleibt → Löschen-Knopf des Datei-Tags deaktiviert,
    Hinweis mit „Tags entfernen" sichtbar → Seite neu laden: `Anlass (0)` noch da.

## Model Hints

Kein `deliverable-hard`: die heikelste Stelle (Teilbaum-Zählung ohne `LIKE`, idempotentes
Anlegen mit Eltern) ist klein und durch benannte Tests fixiert.

Review: → default

## Acceptance Tests

- AC1 → unit `internal/tags/definitions_test.go` › "creating a path also creates its missing parents"
- AC1 → e2e `web/tests/e2e/tags.spec.ts` › "the tag page shows file tags and empty categories as a tree and guards deletion"
- AC2 → e2e `web/tests/e2e/tags.spec.ts` › "the tag page shows file tags and empty categories as a tree and guards deletion"
- AC2 → unit `internal/tags/tree_test.go` › "a photo carrying two tags under one parent counts once for the parent"
- AC2 → unit `internal/library/tagquery_test.go` › "subtree counts treat underscore and percent literally"
- AC3 → e2e `web/tests/e2e/tags.spec.ts` › "the tag page shows file tags and empty categories as a tree and guards deletion"
- AC3 → unit `internal/api/tags_test.go` › "deleting a tag that photos carry is refused"
- AC3 → unit `internal/tags/definitions_test.go` › "deleting a category removes its empty descendants but not its parent or a lookalike"
- AC4 → unit `internal/tags/path_test.go` › "invalid tag names are rejected with a reason"
- AC4 → e2e `web/tests/e2e/tags.spec.ts` › "the tag page shows file tags and empty categories as a tree and guards deletion"
- AC5 → integration `internal/api/tags_test.go` › "the tag tree survives an index rebuild"

Go-Tests tragen den Satz als Funktionsnamen in CamelCase (Muster aus `internal/store/appdb_test.go`,
z. B. `TestCreatingAPathAlsoCreatesItsMissingParents`); Playwright-Tests tragen ihn wörtlich.

Abdeckung: AC1 → D1+D2+D3+D4 · AC2 → D1+D2+D3+D4 · AC3 → D2+D3+D4 · AC4 → D1+D3+D4 · AC5 → D2+D3.

## Done

Tag-Pfade (`tags.ParsePath`), Baum mit Teilbaum-Zählung (`tags.BuildTree`), `tag_definitions` in `app.db`, `GET/POST/DELETE /api/tags` und Seite `/tags` mit Anlegen/Löschen/Sperre sind umgesetzt. Schreibt nichts in Dateien.

Commit: `011: Kategorienbaum — tags.ParsePath/BuildTree, tag_definitions, /api/tags, Seite Tags + E2E`

Verifikation (enges Gate): `npm --prefix web run build` + `docker build --target runtime` grün; `docker build --target test .` (inkl. ExifTool) + `npm --prefix web run test` grün; `docker build --target lint .` grün (nach gofmt-Zeilenenden-Fix an appdb.go) + `npm --prefix web run lint` grün; `npm --prefix web run check` grün; `cd web && npx playwright test tests/e2e/tags.spec.ts` grün. Review (default): PASS, keine Blocker.
AC → Test (alle gelaufen und grün, Go auf Paketebene des Docker-`test`-Stage): AC1 `TestCreatingAPathAlsoCreatesItsMissingParents` + e2e; AC2 `TestAPhotoCarryingTwoTagsUnderOneParentCountsOnceForTheParent`, `TestSubtreeCountsTreatUnderscoreAndPercentLiterally` + e2e; AC3 `TestDeletingATagThatPhotosCarryIsRefused`, `TestDeletingACategoryRemovesItsEmptyDescendantsButNotItsParentOrALookalike` + e2e; AC4 `TestInvalidTagNamesAreRejectedWithAReason` + e2e; AC5 `TestTheTagTreeSurvivesAnIndexRebuild`. Kein manual residue. Volles Regressions-Gate steht aus (Sprint).

Decisions:
- `tagged.jpg` (005) trägt bereits `Anlass/Geburtstag`; E2E und AC5-Test legen darum `Ausflug/Wandern` als leere Kategorie an, statt des im Plan genannten Pfads. Keine neue Fixture.
- `library.TagPairs(ctx, db)` liefert `(seq, errFn)` — `errFn()` nach `BuildTree` prüft Lesefehler; `TagSubtreeCount(ctx, db, path)` nimmt `db` wie die übrigen Library-Funktionen.
- DELETE liefert 404 nur, wenn der Pfad nicht in `tag_definitions` steht (bei Anzahl 0 kann ein Index-Knoten nicht existieren).
- Tag-Routen hängen als optionales `LibraryDeps.Tags *TagDeps{App, Index}` in der Session-/CSRF-Schicht.
- Header erhielt zusätzlich einen Link „Fotos" (Rückweg zur Bibliothek); beide Links ≥ 44 px.
- Go-Dateien auf LF normalisiert (Working-Tree-CRLF ließ `gofmt -l` an der Raw-String-Migration scheitern).
- Bewusst offen (Review, unkritisch): `unicode.IsControl` fängt unsichtbare Cf-Zeichen (U+200B) nicht; ungültiges UTF-8 wird nicht abgewiesen.

tiers: D 4 / hard 0 · review default · cycles 0 · agents 6
