---
id: 002
title: Projektgerüst — Go-Backend, Svelte-Frontend, Verify-Befehle, E2E-Harness
status: draft # draft -> ready -> in-progress -> done
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

- [ ] **AC1** — Der `build`-Befehl aus `.claude/ai-scrum.md` läuft auf einem frischen Checkout
  fehlerfrei durch und erzeugt das Runtime-Image `photocurator:dev`.
- [ ] **AC2** — Die Befehle `test`, `lint` und `typecheck` aus dem Profil laufen grün, mit
  mindestens einem echten Test in Go und einem in `web/`.
- [ ] **AC3** — Ein Test im Docker-Stage `test` belegt, dass ExifTool, `vipsthumbnail` und
  ffmpeg verfügbar sind und dass libvips eine HEIC-Fixture-Datei dekodieren kann.
- [ ] **AC4** — Das Go-Binary liefert das eingebettete Frontend unter `/` aus, und
  `GET /api/health` antwortet mit 200 und JSON inkl. Versionsnummer.
- [ ] **AC5** — `npm --prefix web run test:e2e` startet die App über
  `docker compose -p photocurator-e2e` mit einer Fixture-Bibliothek (temporäre Kopie aus
  `testdata/`) und ein Smoke-Test öffnet die Startseite; danach wird das Compose-Projekt
  wieder entfernt.
- [ ] **AC6** — `.gitignore` deckt `node_modules/`, `web/dist/`, `test-results/`,
  `playwright-report/`, lokale Datenordner (`data/`, `*.db`) und `.env` ab.
- [ ] **AC7** — Das Runtime-Image läuft als Nicht-root-Nutzer und enthält keine
  Build-Werkzeuge (Go, Node).

## Open Questions

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
