---
id: 002
title: Projektgerüst — Go-Backend, Svelte-Frontend, Verify-Befehle, E2E-Harness
status: done # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E0 Fundament & Spike
estimate: 3
sprint: S01
---

## Requirement

Als Entwickler brauche ich ein lauffähiges Grundgerüst gemäß
[Tech Spec](../concepts/photocurator-tech-spec.md), damit alle weiteren Stories auf denselben
Build-, Test- und Prüfbefehlen aufsetzen. Die Verify-Befehle in `.claude/ai-scrum.md` sind
bereits festgelegt und müssen danach exakt so funktionieren. Go, ExifTool, libvips und ffmpeg
sind auf dem Entwicklungsrechner nicht installiert — Backend-Build und -Tests laufen in
Docker-Build-Stages; Node ist lokal vorhanden.

## Acceptance Criteria

- [x] **AC1** — Der `build`-Befehl aus `.claude/ai-scrum.md` läuft auf einem frischen Checkout
  fehlerfrei durch und erzeugt das Runtime-Image `photocurator:dev`.
- [x] **AC2** — Die Befehle `test`, `lint` und `typecheck` aus dem Profil laufen grün, mit
  mindestens einem echten Test in Go und einem in `web/`.
- [x] **AC3** — Ein Test im Docker-Stage `test` belegt, dass ExifTool, `vipsthumbnail` und
  ffmpeg verfügbar sind und dass libvips eine HEIC-Fixture-Datei dekodieren kann.
- [x] **AC4** — Das Go-Binary liefert das eingebettete Frontend unter `/` aus, und
  `GET /api/health` antwortet mit 200 und JSON inkl. Versionsnummer.
- [x] **AC5** — `npm --prefix web run test:e2e` startet die App über
  `docker compose -p photocurator-e2e` mit einer Fixture-Bibliothek (temporäre Kopie aus
  `testdata/`) und ein Smoke-Test öffnet die Startseite; danach wird das Compose-Projekt
  wieder entfernt.
- [x] **AC6** — `.gitignore` deckt `node_modules/`, `web/dist/`, `test-results/`,
  `playwright-report/`, lokale Datenordner (`data/`, `*.db`) und `.env` ab.
- [x] **AC7** — Das Runtime-Image läuft als Nicht-root-Nutzer und enthält keine
  Build-Werkzeuge (Go, Node).

## Open Questions

## Decisions (Sprint)

- **Frischer Checkout** = Clone + einmalig `npm --prefix web ci` (steht in der neuen
  `README.md`), weil die Verify-Befehle im Profil fix sind und ein `npm ci` in jedem Lauf die
  Iteration bremst.
- **Frontend in Docker selbst gebaut:** Das Dockerfile hat eine eigene `web`-Stage (Node 24
  LTS, `npm ci && npm run build`); das lokale `npm run build` ist nur der schnelle
  Frontend-Check — so hängt das Image nie von lokalen Artefakten ab (Tech Spec §8).
- **Gemeinsame Basis-Stage `native`** (debian:trixie-slim + ExifTool, libvips-tools,
  libheif-Decoder-Plugin, ffmpeg, tini): `runtime` und `test` erben beide davon, damit die in
  AC3 getesteten Werkzeuge exakt die des Runtime-Images sind.
- **Go-Modulpfad `photocurator`**, kein Hosting-Pfad, weil das Modul nie importiert wird.
- **Embed-Paket `web/embed.go`** (`package web`, `//go:embed all:dist`), weil `go:embed` keine
  Elternverzeichnisse referenzieren kann; die Go-Stages kopieren `dist` aus der `web`-Stage.
- **`CGO_ENABLED=0`**, statisches Binary; die SQLite-Treiberwahl entscheidet Story 005.
- **CLI ohne Framework** (`os.Args`-Switch, `flag` je Unterbefehl); 002 liefert nur `serve`
  (Standard), weitere Unterbefehle (`probe`, `hash-password`, `healthcheck`) kommen mit
  001/004/003 — minimale Abhängigkeiten.
- **Listen-Adresse** über `PC_LISTEN`, Standard `:8080` (passt zu Tech Spec §8 `8470:8080`).
- **Version** `internal/version.Version`, Standard `0.1.0-dev`, per Build-Arg `VERSION` über
  `-ldflags -X` überschreibbar; Health liefert `{"status":"ok","version":"…"}`.
- **SPA-Fallback:** unbekannte Nicht-`/api`-Pfade liefern `index.html`, unbekannte
  `/api/*`-Pfade 404 JSON — damit spätere Client-Routen (Login, Raster) ohne Serverumbau gehen.
- **Svelte 5 + Vite + TypeScript ohne SvelteKit**, weil das Frontend ein eingebettetes
  statisches SPA ist (Tech Spec §7) und SvelteKit nur Adapter-Komplexität bringt.
- **Frontend-Lint = ESLint** (flat config, `typescript-eslint`, `eslint-plugin-svelte`), kein
  Prettier; **Go-Lint = `gofmt -l` (muss leer sein) + `go vet`**, kein golangci-lint — kleinste
  Werkzeugfläche, später erweiterbar.
- **Tests im Docker-Stage `test`** schlagen fehl statt zu überspringen, wenn ein Werkzeug fehlt,
  weil AC3 genau das belegen soll.
- **Fixtures:** `testdata/library/2024/07/` (Struktur wie `Sortiert`, JJJJ/MM) mit einer
  generierten JPEG- und HEIC-Datei (je ≤ 50 KB, keine privaten Inhalte); der Erzeugungsbefehl
  steht in `testdata/README.md`. Dieselbe Bibliothek speist AC3 und die E2E-Kopie.
- **E2E-Compose-Datei eigenständig** (`web/tests/e2e/compose.e2e.yml`, `build:` mit
  `target: runtime`, `image: photocurator:dev`, `up -d --build`), damit der E2E-Lauf immer den
  aktuellen Code testet; 003 ergänzt dort den Healthcheck. Bereitschaft = Polling auf
  `/api/health` (max. 60 s), weil es in 002 noch keinen Docker-Healthcheck gibt.
- **E2E-Port** `127.0.0.1:18470` → 8080, damit eine Dev-Instanz auf 8470 nicht kollidiert.
- **Aufräumen doppelt:** Global-Setup ruft zuerst `down -v` (Reste eines abgebrochenen Laufs),
  Global-Teardown ruft `down -v`, löscht das Temp-Verzeichnis und prüft danach, dass
  `docker compose -p photocurator-e2e ps -aq` leer ist (sonst Fehler) — so ist AC5 „danach
  entfernt" durch den Lauf selbst belegt.
- **Playwright nur Chromium**, `pretest:e2e` = `playwright install chromium` (idempotent), damit
  der E2E-Befehl auf einem frischen Rechner läuft.
- **`.gitignore`-Datenordner verankert (`/data/`)**, damit kein Quellordner namens `data` (z. B.
  `web/src/lib/data/`) verschluckt wird; `deploy/.env.example` bleibt ausdrücklich versioniert.
- **Nicht-root-Nutzer** im Image: `photocurator`, UID/GID 10001; 003 überschreibt per
  `user: ${PUID}:${PGID}`. Binary unter `/app/photocurator` (Tech Spec §8 Healthcheck-Pfad),
  `ENTRYPOINT ["tini","--","/app/photocurator"]`, `CMD ["serve"]`.

## Plan

Greenfield: noch kein Code im Repo. Reihenfolge so, dass jeder Schritt auf dem vorigen prüfbar ist.

1. **Frontend (D1):** `web/` mit Svelte 5 + Vite + TS. Startseite zeigt „PhotoCurator" und die
   Version aus `GET /api/health`. Scripts `build`, `test` (Vitest), `lint` (ESLint),
   `check` (svelte-check). Vite-Dev-Proxy `/api` → `localhost:8080`.
2. **Repo-Hygiene (D2):** `.gitignore`, `.dockerignore`, kurze `README.md` (Entwicklung:
   Voraussetzungen, `npm --prefix web ci`, Verify-Befehle) + Vitest-Test über `git check-ignore`.
3. **Go-Backend (D3):** `go.mod`, `cmd/photocurator` (`serve`), `internal/version`,
   `internal/api` (Health + eingebettetes SPA mit Fallback), `web/embed.go`, Unit-Tests mit
   `httptest` + `fstest.MapFS`.
4. **Dockerfile (D4):** Stages `web` → `build` (Go) → `native` (Basis mit Werkzeugen) →
   `test` (native + Go-Toolchain, `go test ./...`) / `lint` (gofmt + vet) / `runtime`
   (native + Binary, Nicht-root). Fixture-Bibliothek in `testdata/`,
   `internal/testsupport` mit Fixture-Helfer und dem Werkzeug-/HEIC-Test.
5. **E2E-Harness (D5):** Playwright-Konfiguration, Global-Setup/-Teardown mit
   `docker compose -p photocurator-e2e`, temporäre Kopie von `testdata/library`, Smoke-Spec
   (Startseite + Container-Eigenschaften).

Danach laufen `build`, `test`, `lint`, `typecheck`, `e2e` exakt wie im Profil.

Neue Dateien (Überblick): `web/**`, `.gitignore`, `.dockerignore`, `README.md`, `go.mod`,
`cmd/photocurator/main.go`, `internal/{version,api,testsupport}/**`, `Dockerfile`,
`testdata/**`. Keine bestehende Datei wird geändert. Kein `docs/systems`-Dokument betroffen.

## Deliverables

- [x] **D1 — Svelte-Frontend-Gerüst.** Neues Verzeichnis `web/`: `package.json` +
  `package-lock.json` (Svelte 5, Vite, TypeScript, Vitest, ESLint mit `typescript-eslint` + `eslint-plugin-svelte`, `svelte-check`), Scripts
  `build` (`vite build` → `web/dist`), `test` (`vitest run`), `lint` (`eslint .`),
  `check` (`svelte-check --tsconfig ./tsconfig.json`); `vite.config.ts` (Svelte-Plugin, Dev-Proxy
  `/api` → `http://localhost:8080`, Vitest `include: ['src/**/*.test.ts','tests/repo/**/*.test.ts']`),
  `svelte.config.js`, `tsconfig.json`, `eslint.config.js`, `index.html`, `src/main.ts`,
  `src/App.svelte` (Überschrift `PhotoCurator`, darunter `Version <v>` aus Health; bei Fehler
  ein Hinweis „Server nicht erreichbar"), `src/lib/health.ts`
  (`fetchHealth(fetchFn = fetch): Promise<{status: string; version: string}>`, wirft bei
  Nicht-200). Test `src/lib/health.test.ts`: "reads status and version from the health
  endpoint", "rejects a non-200 health response" (mit injiziertem `fetchFn`). Englische
  Bezeichner, UI-Texte deutsch. Abnahme: `npm --prefix web ci` und danach `run build`, `run test`,
  `run lint`, `run check` grün.
- [x] **D2 — Repo-Hygiene.** `.gitignore` (Repo-Wurzel): `node_modules/`, `web/dist/`,
  `test-results/`, `playwright-report/`, `/data/`, `*.db`, `.env` (nicht `.env.example`);
  `.dockerignore`: `.git`, `**/node_modules`, `web/dist`, `**/test-results`,
  `**/playwright-report`, `data`, `*.db`, `.env`, `docs`, `.claude`; `README.md` (Deutsch,
  ≤ 30 Zeilen): Voraussetzungen (Node, Docker), einmalig `npm --prefix web ci`, die fünf
  Verify-Befehle aus `.claude/ai-scrum.md`, Hinweis dass Go-Build/-Tests nur in Docker laufen.
  Test `web/tests/repo/gitignore.test.ts` › "git ignores build output, test artefacts, local
  data and secrets": ruft `git check-ignore -q --no-index <pfad>` (cwd = Repo-Wurzel) für
  `node_modules/x`, `web/node_modules/x`, `web/dist/index.html`, `web/test-results/x`,
  `web/playwright-report/index.html`, `data/index.db`, `foo.db`, `.env`, `deploy/.env` → alle
  ignoriert; und › "git keeps sources, fixtures and the env example": `deploy/.env.example`,
  `testdata/library/2024/07/sample.heic`, `web/src/lib/data/x.ts` → nicht ignoriert.
- [x] **D3 — Go-Backend-Gerüst.** `go.mod` (Modul `photocurator`, aktuelle Go-Version, keine
  Fremdabhängigkeiten), `web/embed.go` (`package web`; `//go:embed all:dist`; `var Dist embed.FS`),
  `internal/version/version.go` (`var Version = "0.1.0-dev"`), `internal/api/server.go`
  (`NewHandler(static fs.FS) http.Handler`: `GET /api/health` → 200,
  `Content-Type: application/json`, `{"status":"ok","version":version.Version}`; unbekannte
  `/api/*` → 404 JSON `{"error":"not found"}`; sonst statische Datei aus `static`, fehlende
  Datei → `index.html`), `cmd/photocurator/main.go` (Unterbefehl `serve` = Standard, `fs.Sub(web.Dist,"dist")`,
  Adresse aus `PC_LISTEN` Standard `:8080`, Log einer Startzeile, sauberes Beenden auf
  SIGTERM/SIGINT via `http.Server.Shutdown`; unbekannter Unterbefehl → Exit 2 mit Usage).
  Tests `internal/api/server_test.go` (`httptest` + `fstest.MapFS`):
  `TestHealthReportsStatusAndVersion`, `TestRootServesEmbeddedIndex`,
  `TestUnknownClientRouteFallsBackToIndex`, `TestUnknownAPIRouteReturnsJSON404`. Lokal ist kein
  Go installiert — verifiziert wird mit D4 im Docker-Stage `test`; bis dahin reicht
  `docker run --rm -v <repo>:/src -w /src golang:<ver> go vet ./internal/... ./cmd/...` mit
  einem Platzhalter-`web/dist/index.html` (nicht committen; `web/dist` ist ignoriert).
- [x] **D4 — Dockerfile, native Werkzeuge, Fixtures.** `Dockerfile` (Repo-Wurzel) mit Stages:
  `web` (`node:24-slim`, `npm ci`, `npm run build` in `/src/web`); `gobase`
  (`golang:<aktuell>-trixie`, `go.mod` kopieren, Quellen kopieren, `COPY --from=web
  /src/web/dist ./web/dist`); `build` (FROM gobase, `CGO_ENABLED=0 go build -trimpath
  -ldflags "-s -w -X photocurator/internal/version.Version=${VERSION}"`, `ARG VERSION=0.1.0-dev`);
  `lint` (FROM gobase: `test -z "$(gofmt -l .)"` mit Ausgabe der Dateien bei Fehler, `go vet ./...`);
  `native` (`debian:trixie-slim`, `apt-get install --no-install-recommends
  libimage-exiftool-perl libvips-tools libheif-plugin-libde265 ffmpeg tini ca-certificates`,
  apt-Listen löschen); `test` (FROM native, `COPY --from=gobase /usr/local/go /usr/local/go`,
  PATH, Quellen + `web/dist` aus gobase, `go test ./...`); `runtime` (FROM native, Nutzer
  `photocurator` 10001:10001, `COPY --from=build` → `/app/photocurator`, `USER 10001:10001`,
  `EXPOSE 8080`, `ENTRYPOINT ["tini","--","/app/photocurator"]`, `CMD ["serve"]`). Nur `test`
  und `gobase`/`build`/`lint` enthalten Go, nur `web` enthält Node. Fixtures:
  `testdata/library/2024/07/sample.jpg` und `sample.heic` (generiert, z. B. Farbfläche mit
  Text, ≤ 50 KB; HEIC einmalig in einem Wegwerf-Container mit `libheif-plugin-x265` erzeugt),
  `testdata/README.md` (Deutsch, Herkunft + exakter Erzeugungsbefehl). Helfer
  `internal/testsupport/testsupport.go`: `RepoRoot(t)` (sucht `go.mod` aufwärts),
  `CopyFixture(t, rel) string` (kopiert nach `t.TempDir()`, Tests arbeiten nie auf dem
  Original). Test `internal/testsupport/tools_test.go`: `TestNativeToolsAreAvailable`
  (`exiftool -ver`, `vipsthumbnail --vips-version`, `ffmpeg -version` laufen mit Exit 0) und
  `TestLibvipsDecodesHEICFixture` (`vipsthumbnail <kopie>.heic --size 64x64 -o <tmp>/t.webp`,
  danach `vipsheader` → Breite und Höhe > 0); fehlende Werkzeuge = Fehler, kein Skip.
  Abnahme: `build`, `test`, `lint` aus dem Profil grün; `docker run --rm --entrypoint id
  photocurator:dev -u` ≠ 0.
- [x] **D5 — E2E-Harness.** In `web/`: devDependency `@playwright/test`, Scripts
  `test:e2e` (`playwright test`) und `pretest:e2e` (`playwright install chromium`);
  `playwright.config.ts` (`testDir: tests/e2e`, nur Chromium, `baseURL http://127.0.0.1:18470`,
  `workers: 1`, `globalSetup`/`globalTeardown`); `tests/e2e/compose.e2e.yml` (Dienst
  `photocurator`, `build: {context: ../../.., target: runtime}`, `image: photocurator:dev`,
  Port `127.0.0.1:18470:8080`, Volumes `${PC_E2E_ROOT}/photos:/photos`,
  `${PC_E2E_ROOT}/data:/data`, `${PC_E2E_ROOT}/backup:/backup`);
  `tests/e2e/compose.ts` (gemeinsamer Helfer: Projektname `photocurator-e2e`, Compose-Dateipfad,
  `compose(args)` per `execFileSync('docker', …)`, kein Shell-String); `tests/e2e/global-setup.ts`
  (`down -v`; Temp-Verzeichnis via `fs.mkdtemp(os.tmpdir())`, rekursive Kopie von
  `testdata/library` nach `<tmp>/photos`, leere `data`/`backup`; Pfad in
  `process.env.PC_E2E_ROOT` und Datei `test-results/.e2e-root` für den Teardown;
  `up -d --build`; Polling `GET /api/health` bis 200, max. 60 s, sonst `compose logs` ausgeben und
  Fehler); `tests/e2e/global-teardown.ts` (`down -v`, Temp-Verzeichnis löschen, dann
  `ps -aq` muss leer sein, sonst `throw`); `tests/e2e/smoke.spec.ts` mit
  › "the start page shows the app name and version" (öffnet `/`, sieht Überschrift
  „PhotoCurator" und den Text `Version 0.1.0-dev`, der aus `/api/health` stammt) und
  › "the app container runs as non-root without build tools" (`compose exec -T photocurator
  sh -c 'id -u; command -v go node npm; test -d /usr/local/go && echo has-go'` → UID ≠ 0, keine
  weitere Ausgabe). Windows-Pfade (Docker Desktop) müssen funktionieren. Abnahme:
  `npm --prefix web run test:e2e` grün, danach `docker compose -p photocurator-e2e ps -aq` leer.

## Model Hints

- D4 → deliverable-hard — Risiko: Stage-Aufbau mit gemeinsamer `native`-Basis, HEIC-Dekodierung
  hängt am richtigen libheif-Plugin unter trixie und die HEIC-Fixture braucht einen separaten
  Encoder; eine falsche Paket-/Stage-Zusammensetzung lässt `test` grün, während `runtime` die
  Werkzeuge nicht hat oder Go/Node enthält.
- D1, D2, D3, D5 → default.
- Review: → default

## Acceptance Tests

- AC1 → Verify-Befehl `build` aus dem Profil (Exit 0, erzeugt `photocurator:dev`) im
  Story-Gate; der E2E-Lauf startet genau dieses Image (`tests/e2e/smoke.spec.ts` ›
  "the start page shows the app name and version").
- AC2 → Verify-Befehle `test`, `lint`, `typecheck` grün; echte Tests: Go
  `internal/api/server_test.go` › `TestHealthReportsStatusAndVersion`, web
  `web/src/lib/health.test.ts` › "reads status and version from the health endpoint".
- AC3 → integration (Docker-Stage `test`) `internal/testsupport/tools_test.go` ›
  `TestNativeToolsAreAvailable` und › `TestLibvipsDecodesHEICFixture`.
- AC4 → unit `internal/api/server_test.go` › `TestRootServesEmbeddedIndex` und ›
  `TestHealthReportsStatusAndVersion`; durch das echte Binary: e2e `tests/e2e/smoke.spec.ts` ›
  "the start page shows the app name and version".
- AC5 → e2e `tests/e2e/smoke.spec.ts` › "the start page shows the app name and version"
  (Start über Compose-Projekt mit Fixture-Kopie); Entfernen belegt durch die Prüfung in
  `web/tests/e2e/global-teardown.ts` (leeres `ps -aq`, sonst schlägt der Lauf fehl).
- AC6 → unit `web/tests/repo/gitignore.test.ts` › "git ignores build output, test artefacts,
  local data and secrets".
- AC7 → e2e `tests/e2e/smoke.spec.ts` › "the app container runs as non-root without build
  tools".

Coverage: AC1 → D4 (+D1 Build-Script) · AC2 → D1, D3, D4 · AC3 → D4 · AC4 → D1, D3 ·
AC5 → D5 · AC6 → D2 · AC7 → D4 (Image), D5 (Test).

## Done

**Zusammenfassung:** Go-Backend (serve, /api/health, eingebettetes SPA mit Fallback), Svelte-5-Frontend, mehrstufiges Dockerfile (web/gobase/build/lint/native/test/runtime, Nicht-root 10001), Fixtures + Testsupport, Playwright-E2E-Harness über `docker compose -p photocurator-e2e` mit Aufräum-Prüfung.

**Commit:** `002: Projektgerüst — Go-Backend, Svelte-Frontend, Dockerfile-Stages, E2E-Harness`

**Verifikation (enges Gate):** `build`, `test`, `lint`, `typecheck` grün; e2e per `cd web && npx playwright test tests/e2e/smoke.spec.ts` grün (2/2), danach `ps -aq` leer. Review 1 (default): PASS. AC → Test (alle bestanden): AC1 build + Image `photocurator:dev`; AC2 `TestHealthReportsStatusAndVersion`, health.test.ts; AC3 `TestNativeToolsAreAvailable`, `TestLibvipsDecodesHEICFixture`; AC4 `TestRootServesEmbeddedIndex` + smoke; AC5 smoke "start page" + Teardown-Prüfung; AC6 gitignore.test.ts; AC7 smoke "non-root". Kein manual residue. Volles Gate steht beim Sprint aus.

**Decisions:**
- Vite 7.3 statt 8, TypeScript 6.0, Svelte-Plugin 6: npm 10.9 stürzt bei Vite 8 ab, typescript-eslint akzeptiert TS < 6.1; Vitest 4 per `--legacy-peer-deps` installiert, Lockfile läuft mit schlichtem `npm ci`.
- Go 1.26 (`golang:1.26-trixie`); Health-URL `/api/health`.
- Review-Fix: Test um `data/x.json` ergänzt (prüft `/data/`-Regel unabhängig von `*.db`).
- Offen, nicht behoben (dokumentiert): Setup-Fehler überspringt Teardown (nächster Lauf räumt per `down -v` auf); Teststärke `TestRootServesEmbeddedIndex` nutzt MapFS, Embed-Verdrahtung beweist nur e2e; Image `photocurator:dev` ~978 MB (ffmpeg-Abhängigkeiten) — gegen NAS-Ziele prüfen; `test-support` im Profil kann auf `internal/testsupport/` zeigen.

tiers: D 5 / hard 1 · review default · cycles 1 · agents 7
