---
id: 003
title: Deployment als Docker-Compose-Projekt auf UGOS Pro
status: done # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E0 Fundament & Spike
estimate: 3
sprint: S01
---

## Requirement

Als Nutzer möchte ich PhotoCurator als „Projekt" in UGOS Pro (Docker Compose) installieren,
mit dem Ordner `Sortiert` als Volume, ohne dass die App Zugriff auf die Inbox, auf UGOS oder auf
andere Ordner bekommt, und ohne dass sie das NAS über Gebühr belastet.

## Acceptance Criteria

- [x] **AC1** — `deploy/docker-compose.yml` bindet genau drei Host-Pfade ein: `Sortiert` →
  `/photos`, App-Daten → `/data`, Backup → `/backup`; kein Inbox-Pfad, kein Docker-Socket
  (automatischer Test, der die Compose-Datei parst).
- [x] **AC2** — Der Dienst läuft mit `user: ${PUID}:${PGID}`, `read_only: true`,
  `cap_drop: [ALL]`, `no-new-privileges` und gesetzten Grenzen für Speicher und CPU (Test
  wie AC1).
- [x] **AC3** — Die App startet nicht und meldet einen verständlichen Fehler, wenn `/photos`
  fehlt, nicht beschreibbar ist oder der konfigurierte Fotopfad `Inbox` enthält; ebenso wenn
  `/data` oder `/backup` nicht beschreibbar sind.
- [x] **AC4** — Der Compose-Dienst hat einen Healthcheck; im E2E-Lauf meldet Docker den
  Container als `healthy`.
- [x] **AC5** — `deploy/.env.example` enthält alle Variablen (PUID, PGID, Pfade, Port,
  Nominatim-Kontakt, Zeitzone) mit Kommentaren.
- [x] **AC6** — `deploy/README.md` (Deutsch) beschreibt die Installation in UGOS Pro Schritt für
  Schritt: Ordner anlegen, passenden Nutzer/UID ermitteln, Rechte, `.env`, Projekt anlegen,
  erster Aufruf, Update.
- [ ] **AC7** — Installation auf dem echten DXP4800 Plus. **manual residue:** echtes NAS mit
  UGOS Pro nötig.

## Open Questions

- ~~Genauer Pfad des Backup-Ordners auf dem NAS und ob er außerhalb aller von UGREEN Fotos indexierten Ordner liegt (Vorschlag: `/volume1/docker/photocurator/backup`).~~ answered → Decisions (Sprint)
- ~~HTTP im Heimnetz ausreichend oder HTTPS über den UGOS-Reverse-Proxy?~~ answered → Decisions (Sprint)

## Decisions (Sprint)

- **(User)** Backup-Pfad: `/volume1/docker/photocurator/backup`.
- **(User)** Transport: HTTP im Heimnetz, kein HTTPS/Reverse-Proxy in diesem Sprint.
- Host-Pfade, Port, Bind-IP, PUID/PGID, TZ und Nominatim-Kontakt kommen als `${VAR}` aus `.env`;
  die Compose-Datei enthält keine festen NAS-Pfade — nur so kann das E2E dieselbe Datei mit
  Fixture-Ordnern starten und AC5 („alle Variablen") ist prüfbar.
- Compose-Datei hat nur `image: ${PC_IMAGE:-photocurator:latest}`, kein `build:` — auf dem NAS
  liegt kein Quellcode; das Image wird am PC gebaut und per `docker save`/`docker load` (SSH)
  übertragen, so beschrieben im README.
- Ressourcengrenzen als `mem_limit: 768m` und `cpus: 2.0` (Startwerte der Tech Spec §8, gemessen
  wird erst in Story 021).
- Schreibbarkeit wird mit `access(2)` (`unix.Access(path, W_OK)`) geprüft, nicht über
  Modus-Bits und nicht über eine Probedatei — erkennt `:ro`-Mounts und ACLs von UGOS und legt
  keine Hilfsdateien in `/photos` an (DoD 3).
- Inbox-Schutz: Start wird abgelehnt, wenn der Fotopfad im Container **oder** der durchgereichte
  Host-Pfad (`PC_PHOTOS_HOST_PATH=${PHOTOS_PATH}`) ein Segment `Inbox` (ohne Groß/Klein) enthält
  oder `/photos` direkt einen Unterordner `Inbox` hat — der Container sieht den Host-Pfad sonst
  nie, und der wahrscheinlichste Fehler ist, `Gemeinsame Fotos` statt `Sortiert` einzubinden.
- Startfehler-Meldungen auf Englisch (Code-Sprache), aber mit Pfad, Ursache und Abhilfe
  (z. B. PUID/PGID-Rechte); Exit-Code 1 — sichtbar im UGOS-Container-Log.
- Healthcheck über Unterbefehl `photocurator healthcheck` (GET auf `/api/health` aus 002), weil
  das schlanke Runtime-Image kein curl/wget hat; Werte `interval 30s, timeout 5s, retries 3,
  start_period 20s` (kein `start_interval`, das ältere Docker-Versionen auf UGOS evtl. nicht kennen).
- Das E2E-Harness aus 002 startet künftig `deploy/docker-compose.yml` + Override
  `web/tests/e2e/compose.e2e.yml` (Image `photocurator:dev`, kurzes Healthcheck-Intervall) statt
  einer eigenen Compose-Datei — so prüft jeder E2E-Lauf die echte Deploy-Datei.
- Strukturtests in Go (`deploy/`-Paket, `gopkg.in/yaml.v3`), weil sie im Docker-Stage `test`
  laufen, das ohnehin alle Backend-Tests ausführt; `deploy/` darf nicht in `.dockerignore` stehen.
- Der Docker-Stage `test` führt `go test` als Nicht-root-Nutzer aus — als root greifen
  Dateirechte nicht und die „nicht beschreibbar"-Tests wären wertlos; zugleich näher an der
  Produktion.
- `PC_USERS_FILE` kommt nicht in diese Story — die Variable gehört Story 004, die sie in die
  Compose-Datei ergänzt.
- Keine Router-Freigabe: README weist darauf hin, den Port nur im Heimnetz zu nutzen
  (folgt aus der HTTP-Entscheidung).

## Plan

1. **Compose + `.env.example`** (`deploy/`): ein Dienst `photocurator`, genau drei Volumes
   (`${PHOTOS_PATH}`→`/photos`, `${DATA_PATH}`→`/data`, `${BACKUP_PATH}`→`/backup`),
   `user: "${PUID}:${PGID}"`, `read_only`, `tmpfs: [/tmp]`, `cap_drop: [ALL]`,
   `no-new-privileges:true`, Limits, Port `${BIND_IP:-0.0.0.0}:${PORT:-8470}:8080`,
   `restart: unless-stopped`, Healthcheck. Go-Strukturtest parst Compose und `.env.example`.
2. **Startprüfung** (`internal/config`): Pfade aus Env (`PC_PHOTOS_DIR`, `PC_DATA_DIR`,
   `PC_BACKUP_DIR`, `PC_PHOTOS_HOST_PATH`), `ValidatePaths()` vor dem Listen in `serve`;
   Test-Stage im `Dockerfile` läuft als Nicht-root.
3. **Healthcheck + E2E**: Unterbefehl `healthcheck`; E2E-Harness auf Deploy-Compose + Override
   umstellen; E2E-Test liest `docker inspect` → `healthy`.
4. **README** (Deutsch) Schritt für Schritt + Doku-Test; Tech Spec §8 an die Variablen-Fassung
   anpassen.

Reihenfolge D1 → D2 → D3 → D4. Setzt voraus, dass 002 (Gerüst, `/api/health`, E2E-Harness)
fertig ist. AC7 bleibt manual residue (echtes NAS).

## Deliverables

- **D1 — Compose-Datei, `.env.example`, Strukturtest.**
  Dateien: `deploy/docker-compose.yml`, `deploy/.env.example`, `deploy/doc.go` (nur
  `package deploy` + Paketkommentar, damit `go build ./...` kein Test-only-Paket meldet),
  `deploy/deploy_test.go`, `go.mod`/`go.sum` (`gopkg.in/yaml.v3`), ggf. `.dockerignore`
  (`deploy/` muss im Build-Kontext des Stage `test` sein). Vorlage: Tech Spec §8
  (`docs/concepts/photocurator-tech-spec.md`), aber alle Host-Werte als `${VAR}`:
  `image: ${PC_IMAGE:-photocurator:latest}` (kein `build:`), `container_name: photocurator`,
  `restart: unless-stopped`, `user: "${PUID}:${PGID}"`, Port `"${BIND_IP:-0.0.0.0}:${PORT:-8470}:8080"`,
  environment `TZ: ${TZ}`, `PC_NOMINATIM_CONTACT: ${NOMINATIM_CONTACT}`,
  `PC_PHOTOS_HOST_PATH: ${PHOTOS_PATH}`, `PC_PRESERVE_MTIME: "false"`; volumes genau
  `${PHOTOS_PATH}:/photos`, `${DATA_PATH}:/data`, `${BACKUP_PATH}:/backup`; `read_only: true`,
  `tmpfs: [/tmp]`, `cap_drop: [ALL]`, `security_opt: ["no-new-privileges:true"]`,
  `mem_limit: 768m`, `cpus: 2.0`, `healthcheck: test ["CMD", "/app/photocurator", "healthcheck"],
  interval 30s, timeout 5s, retries 3, start_period 20s` (Binary-Pfad an das Runtime-Image aus
  002 anpassen). Kein `privileged`, kein `network_mode: host`, kein `pid: host`, keine `devices`.
  `.env.example`: jede Variable mit vorangestellter `#`-Kommentarzeile (Deutsch), Beispielwerte
  `PHOTOS_PATH=/volume1/Gemeinsame Fotos/Sortiert`, `DATA_PATH=/volume1/docker/photocurator/data`,
  `BACKUP_PATH=/volume1/docker/photocurator/backup`, `PUID`, `PGID`, `PORT=8470`, `BIND_IP=0.0.0.0`,
  `TZ=Europe/Vienna`, `NOMINATIM_CONTACT`, `PC_IMAGE`.
  Tests in `deploy/deploy_test.go` (YAML roh parsen, nicht interpolieren):
  `TestComposeMountsOnlyPhotosDataBackup` (genau drei Volumes mit Zielen `/photos`, `/data`,
  `/backup`; keine Quelle enthält `docker.sock` oder `Inbox`; kein `privileged`/Host-Netz/-PID/
  `devices`), `TestComposeServiceIsHardened` (user `${PUID}:${PGID}`, `read_only: true`,
  `cap_drop` = `[ALL]`, `no-new-privileges:true`, `mem_limit` und `cpus` gesetzt, Healthcheck
  vorhanden), `TestEnvExampleDocumentsEveryComposeVariable` (jede `${VAR}` der Compose-Datei steht
  in `.env.example`, jede Zeile `VAR=` hat direkt darüber einen Kommentar; Pflichtliste PUID,
  PGID, PHOTOS_PATH, DATA_PATH, BACKUP_PATH, PORT, NOMINATIM_CONTACT, TZ; kein Beispielpfad
  enthält `Inbox`).

- **D2 — Startprüfung der Pfade.**
  Dateien: `internal/config/paths.go` (+ ggf. Erweiterung des Config-Ladens aus 002),
  `internal/config/paths_test.go`, `cmd/photocurator` (`serve` ruft die Prüfung vor dem
  Listen auf; Test in `cmd/photocurator/serve_test.go` o. ä.), `Dockerfile` (Stage `test`:
  `go test` als Nicht-root-Nutzer, z. B. `USER 65532` mit `GOCACHE=/tmp/gocache`; Modul-Download
  davor als root).
  Verhalten: Env `PC_PHOTOS_DIR` (Default `/photos`), `PC_DATA_DIR` (`/data`), `PC_BACKUP_DIR`
  (`/backup`), `PC_PHOTOS_HOST_PATH` (optional). `ValidatePaths(cfg) error` liefert einen Fehler,
  wenn: Fotopfad fehlt oder kein Verzeichnis ist; Fotopfad nicht beschreibbar; Fotopfad oder
  Host-Pfad ein Pfadsegment `inbox` (case-insensitive) enthält; `/photos` direkt einen
  Unterordner `Inbox` (case-insensitive) hat; Daten- oder Backup-Pfad fehlt oder nicht
  beschreibbar. Schreibbarkeit **nur** über `unix.Access(path, unix.W_OK)` (`golang.org/x/sys/unix`)
  — keine Modus-Bit-Auswertung, keine Probedatei. Alle Verstöße auf einmal melden (gesammelt),
  jede Meldung nennt Pfad, Ursache und Abhilfe (z. B. „check that PUID/PGID may write to …").
  `serve` loggt die Meldung und endet mit Exit-Code 1, ohne den HTTP-Server zu starten.
  Tests: `TestValidatePathsRejectsMissingPhotosDir`, `TestValidatePathsRejectsReadOnlyDirs`
  (Fotos, Daten, Backup per `chmod 0555`; schlägt fehl statt zu überspringen, wenn als root
  ausgeführt), `TestValidatePathsRejectsInboxPaths` (Container-Pfad, Host-Pfad, Unterordner
  `Inbox`), `TestValidatePathsAcceptsWritableLibrary`, und in `cmd/photocurator`
  `TestServeRefusesToStartWithInvalidPaths` (Startfunktion gibt Fehler zurück, Port wird nicht
  geöffnet).

- **D3 — Healthcheck-Befehl und E2E gegen die Deploy-Compose-Datei.**
  Dateien: `cmd/photocurator` (Unterbefehl `healthcheck`: GET
  `http://127.0.0.1:<Listen-Port>/api/health`, Timeout 3 s, Exit 0 bei 200, sonst 1; Test
  `cmd/photocurator/healthcheck_test.go` › `TestHealthcheckExitCodeFollowsHealthEndpoint` mit
  `httptest`), `web/tests/e2e/compose.e2e.yml`
  (Override: `image: photocurator:dev`, `healthcheck.interval: 2s`, `start_period: 5s`),
  das E2E-Global-Setup/-Teardown aus 002 (startet jetzt
  `docker compose -p photocurator-e2e -f ../deploy/docker-compose.yml -f tests/e2e/compose.e2e.yml up -d --wait`
  mit Env `PHOTOS_PATH`/`DATA_PATH`/`BACKUP_PATH` = temporäre Ordner, `PUID`/`PGID` =
  `process.getuid?.() ?? 1000` bzw. `getgid`, `PORT` = E2E-Port, `TZ`, `NOMINATIM_CONTACT=e2e@example.invalid`;
  eine bisherige eigene E2E-Compose-Datei aus 002 entfällt), `web/tests/e2e/deployment.spec.ts`.
  Test: `"the compose service reports healthy"` — liest
  `docker inspect --format {{.State.Health.Status}}` des Containers im Projekt
  `photocurator-e2e` und erwartet `healthy`. Bestehender Smoke-Test aus 002 muss weiter grün sein;
  `e2e-cleanup` (`docker compose -p photocurator-e2e down -v`) muss das Projekt weiter entfernen.

- **D4 — Installationsanleitung und Tech-Spec-Abgleich.**
  Dateien: `deploy/README.md` (Deutsch), `deploy/readme_test.go`,
  `docs/concepts/photocurator-tech-spec.md` (§8: Compose-Auszug auf `${VAR}`-Fassung, kein
  `build:`, Healthcheck-Werte, `PC_PHOTOS_HOST_PATH`; §9 Netzwerk: HTTP-Entscheidung).
  README-Abschnitte (Überschriften genau so): `## Ordner anlegen` (data/backup unter
  `/volume1/docker/photocurator/`, Backup außerhalb der von UGREEN Fotos indexierten Ordner),
  `## Nutzer und UID ermitteln` (UGOS-Nutzer mit Schreibrecht auf `Gemeinsame Fotos`, `id <name>`
  per SSH), `## Rechte`, `## .env anlegen` (aus `.env.example`), `## Image übertragen`
  (`docker build --target runtime`, `docker save`/`docker load`), `## Projekt anlegen` (UGOS
  Pro Docker → Projekt, Compose einfügen), `## Erster Aufruf` (`http://<NAS-IP>:<PORT>`, nur
  Heimnetz, Port nicht im Router freigeben; Startfehler im Container-Log), `## Update`.
  Test `TestInstallGuideCoversEveryStep` (alle Überschriften vorhanden, jede Variable aus
  `.env.example` wird im README erwähnt).

## Model Hints

- D3 → deliverable-hard — stellt das E2E-Harness aus 002 auf die produktive Compose-Datei um:
  falsche Env-Weitergabe, UID/GID-Rechte auf den Temp-Ordnern oder ein Healthcheck-Timing, das
  `--wait` hängen lässt, brechen jeden künftigen E2E-Lauf aller Stories.
- Review: → default

## Acceptance Tests

- AC1 → unit `deploy/deploy_test.go` › `TestComposeMountsOnlyPhotosDataBackup`
- AC2 → unit `deploy/deploy_test.go` › `TestComposeServiceIsHardened`
- AC3 → unit `internal/config/paths_test.go` › `TestValidatePathsRejectsMissingPhotosDir`,
  `TestValidatePathsRejectsReadOnlyDirs`, `TestValidatePathsRejectsInboxPaths`,
  `TestValidatePathsAcceptsWritableLibrary`; unit `cmd/photocurator/serve_test.go` ›
  `TestServeRefusesToStartWithInvalidPaths`
- AC4 → e2e `tests/e2e/deployment.spec.ts` (relativ zu `web/`) › "the compose service reports healthy";
  Healthcheck-Logik zusätzlich unit `cmd/photocurator/healthcheck_test.go` ›
  `TestHealthcheckExitCodeFollowsHealthEndpoint`
- AC5 → unit `deploy/deploy_test.go` › `TestEnvExampleDocumentsEveryComposeVariable`
- AC6 → unit `deploy/readme_test.go` › `TestInstallGuideCoversEveryStep`
- AC7 → manual residue: Installation auf dem echten DXP4800 Plus mit UGOS Pro — Hardware und
  proprietäre UGOS-Docker-Oberfläche sind nicht automatisierbar.

Abdeckung: AC1/AC2/AC5 → D1 · AC3 → D2 · AC4 → D1 (Healthcheck-Eintrag) + D3 · AC6 → D4 ·
AC7 → manual residue (README aus D4 ist die Anleitung dafür).

## Done

Compose-Datei, `.env.example`, Pfad-Startprüfung (`internal/config`, Exit 1), `healthcheck`-Unterbefehl, E2E-Harness auf die Deploy-Compose umgestellt und deutsche Installationsanleitung samt Tech-Spec-Abgleich.

**Commit:** `003: Docker-Compose-Deployment — deploy/compose+env+README, Pfad-Startprüfung, healthcheck, E2E gegen Deploy-Compose`

**Verifikation (narrow gate):** `npm --prefix web run build && docker build --target runtime`, `docker build --target test` (alle Pakete ok), `--target lint`, `npm --prefix web run test`, `npm --prefix web run check`, `cd web && npx playwright test tests/e2e/deployment.spec.ts` — alle grün. Review (default, 1 Durchgang): PASS, keine Muss-Fixes.
- AC1 → `TestComposeMountsOnlyPhotosDataBackup` ok · AC2 → `TestComposeServiceIsHardened` ok · AC3 → `paths_test.go` (4 Tests) + `TestServeRefusesToStartWithInvalidPaths` ok · AC4 → e2e "the compose service reports healthy" ok + `TestHealthcheckExitCodeFollowsHealthEndpoint` ok · AC5 → `TestEnvExampleDocumentsEveryComposeVariable` ok · AC6 → `TestInstallGuideCoversEveryStep` ok. (Go-Tests: Paket-`ok` im Stage `test`, nicht `-v`.)
- AC7: manual residue — echtes NAS mit UGOS Pro.

**Decisions:** Stage `test` läuft als USER 65532 (`chmod a+rX` auf Quellen/Module). E2E-Override benennt den Container `photocurator-e2e`, `compose.ts` setzt alle `${VAR}` explizit (BIND_IP=127.0.0.1, PORT=18470), `up --pull never --wait --wait-timeout 90`; E2E-Temp-Ordner 0777 (PUID = Host-UID). `npm run test:e2e` baut das Image nicht mehr — vorher `build` aus dem Profil.
**Offene Findings (bewusst nicht behoben):** `internal/config` nutzt `x/sys/unix` ohne Build-Tag (Go läuft nur im Docker, nicht auf Windows); `checkStart` ist dünner Wrapper mit leicht redundanter Assertion; README-Test verlangt LF-Zeilenenden; Healthcheck-Werte in `TestComposeServiceIsHardened` nur auf Vorhandensein geprüft.

tiers: D 4 / hard 1 · review default · cycles 1 · agents 7
