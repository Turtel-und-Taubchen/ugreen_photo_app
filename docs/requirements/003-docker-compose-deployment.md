---
id: 003
title: Deployment als Docker-Compose-Projekt auf UGOS Pro
status: draft # draft -> ready -> in-progress -> done
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

- [ ] **AC1** — `deploy/docker-compose.yml` bindet genau drei Host-Pfade ein: `Sortiert` →
  `/photos`, App-Daten → `/data`, Backup → `/backup`; kein Inbox-Pfad, kein Docker-Socket
  (automatischer Test, der die Compose-Datei parst).
- [ ] **AC2** — Der Dienst läuft mit `user: ${PUID}:${PGID}`, `read_only: true`,
  `cap_drop: [ALL]`, `no-new-privileges` und gesetzten Grenzen für Speicher und CPU (Test
  wie AC1).
- [ ] **AC3** — Die App startet nicht und meldet einen verständlichen Fehler, wenn `/photos`
  fehlt, nicht beschreibbar ist oder der konfigurierte Fotopfad `Inbox` enthält; ebenso wenn
  `/data` oder `/backup` nicht beschreibbar sind.
- [ ] **AC4** — Der Compose-Dienst hat einen Healthcheck; im E2E-Lauf meldet Docker den
  Container als `healthy`.
- [ ] **AC5** — `deploy/.env.example` enthält alle Variablen (PUID, PGID, Pfade, Port,
  Nominatim-Kontakt, Zeitzone) mit Kommentaren.
- [ ] **AC6** — `deploy/README.md` (Deutsch) beschreibt die Installation in UGOS Pro Schritt für
  Schritt: Ordner anlegen, passenden Nutzer/UID ermitteln, Rechte, `.env`, Projekt anlegen,
  erster Aufruf, Update.
- [ ] **AC7** — Installation auf dem echten DXP4800 Plus. **manual residue:** echtes NAS mit
  UGOS Pro nötig.

## Open Questions

- Genauer Pfad des Backup-Ordners auf dem NAS und ob er außerhalb aller von UGREEN Fotos
  indexierten Ordner liegt (Vorschlag: `/volume1/docker/photocurator/backup`).
- HTTP im Heimnetz ausreichend oder HTTPS über den UGOS-Reverse-Proxy?

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
