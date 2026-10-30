# Sprint S01 — Review

## Überblick

Ziel: Projektgerüst mit Verify-Befehlen und E2E-Harness, abgesicherter Compose-Betrieb mit Anmeldung, Spike-Werkzeug samt Protokoll.

| Story | Status | Commit |
| --- | --- | --- |
| 002 Projektgerüst | done | f214d99 |
| 001 Spike-Werkzeug | done | e8f4539 |
| 003 Compose-Deployment | done | 3842f42 |
| 004 Anmeldung für zwei Nutzer | done | ad4eb4d |

## Umgesetzt

- **002:** Go-Backend (`serve`, `/api/health`, eingebettetes SPA), Svelte-5-Frontend, mehrstufiges Dockerfile, Fixtures, Playwright-Harness über `photocurator-e2e`.
- **001:** `photocurator probe generate|mutate` erzeugt den Testsatz (JPEG/HEIC/PNG/MP4/MOV, Sidecars, Combo, WhatsApp-Kontrolle) samt `PROTOKOLL.md`; Strategie-Dokument mit leerer Feldmatrix.
- **003:** `deploy/docker-compose.yml` + `.env.example` + deutsche Anleitung, Pfad-Startprüfung (Inbox-Schutz, Schreibbarkeit), `healthcheck`-Unterbefehl; E2E läuft gegen die Deploy-Compose.
- **004:** Argon2id + `users.conf` + `hash-password`, Sessions in `app.db`, Login-Limiter, Default-deny + CSRF-Header, Login-UI, E2E-Flow.

## Findings & Entscheidungen

Nutzer-Entscheidungen (003): Backup-Pfad `/volume1/docker/photocurator/backup`; HTTP im Heimnetz.

- Versionen in 002: Vite 7.3 (npm 10.9 stürzt bei Vite 8 ab), TypeScript 6.0 (typescript-eslint), Vitest 4 mit `--legacy-peer-deps`.
- 001: `UserData:Keywords` (Video) entfällt, ExifTool 13.25 kann es nicht schreiben; `libheif-plugin-x265` im Dockerfile ergänzt.
- `npm run test:e2e` baut `photocurator:dev` nicht mehr; vorher `build` aus dem Profil ausführen.
- Bewusst nicht behoben, als Follow-ups in der [Roadmap](../../../ROADMAP.md) geführt: Image-Größe ~978 MB, `x/sys/unix` ohne Build-Tag, Limiter-Map-Bereinigung, Session-Lookup für Assets, 360-px-Layout ungeprüft, Setup-Fehler überspringt Teardown, `test-support` im Profil nachziehen.
- Fortschrittsdatei: die ersten Zeilen (Orchestrator-Einträge bis Story-002-Start) tragen kein Datum, spätere Agent-Zeilen schon; Zeitstempel sind monoton, keiner liegt in der Zukunft.
- Go-Testnamen sind im Docker-`test`-Stage ohne `-v` nicht sichtbar; Nachweis je Story = Paket `ok` + Test per grep vorhanden.

## Blockiert / offen

Nichts blockiert. Offen für den Nutzer: Spike-Durchführung auf dem NAS (siehe unten), vor dem Refinement von S03.

## Regression-Gate

`build`, `test`, `e2e` je einmal auf `ad4eb4d`: grün (je <1 min, e2e 4/4). `e2e-all` ist `none`. Keine Fehler, keine Quarantäne (`e2e-quarantine: none` — das Projekt hat noch keine).

## Akzeptanz

Alle Kriterien durch Tests belegt (Details je Story im Done-Abschnitt):
- **001:** AC1–AC11 durch `internal/probe`-Tests (u. a. `TestGenerate_OneFilePerFormatAndField`, `TestProtocol_OneRowPerGeneratedFile`, `TestStrategyDoc_ListsEveryProbeField`).
- **002:** AC1–AC7 durch Build, `TestHealthReportsStatusAndVersion`, `TestNativeToolsAreAvailable`, `gitignore.test.ts`, E2E `smoke.spec.ts`.
- **003:** AC1–AC6 durch `deploy_test.go`, `paths_test.go`, `serve_test.go`, `readme_test.go`, E2E `deployment.spec.ts`.
- **004:** AC1–AC7 durch `internal/auth`-/`internal/api`-Tests und E2E `auth.spec.ts`.

**Manual residue:**
- 001 AC12: Testsatz in der proprietären UGREEN Fotos-App auf dem NAS prüfen (Nutzer, nach dem Sprint).
- 003 AC7: Lauf auf dem echten NAS mit UGOS Pro.

Unter der echten Oberfläche geprüft: AC5 (004) nur als Go-Test mit Fake-Clock, da die Sperre per E2E nicht abwartbar ist (bewusst, dokumentiert).

## Tier-Record

| Story | Ds | hard | Review | Zyklen | Agents | Build (min) |
| --- | --- | --- | --- | --- | --- | --- |
| 002 | 5 | 1 | default | 1 | 7 | 18 |
| 001 | 5 | 0 | default | 1 | 9 | 36 |
| 003 | 4 | 1 | default | 1 | 7 | 27 |
| 004 | 4 | 1 | default | 1 | 7 | 47 |
| **Summe** | 18 | 3 | 0 von 4 hard | 4 | 30 | 128 |

Kein Hard-Review in diesem Sprint, daher kein Vergleich möglich.

## Testplan

Kein `testplan.md`: Die beiden Manual-Residues (001 AC12, 003 AC7) stehen oben; sonst muss nichts von Hand geprüft werden.
Nach dem Sprint (Nutzer): Testsatz aus 001 nach `Sortiert/_pcprobe/` kopieren, in UGREEN Fotos prüfen, `PROTOKOLL.md` ausfüllen, Ergebnis ins Strategie-Dokument übertragen, Testordner löschen.
