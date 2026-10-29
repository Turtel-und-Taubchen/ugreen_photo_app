---
id: 004
title: Einfache Anmeldung für zwei Nutzer
status: draft # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E0 Fundament & Spike
estimate: 3
sprint: S01
---

## Requirement

Nur wir zwei sollen PhotoCurator benutzen können — auch im Heimnetz soll nicht jedes Gerät
Fotos verändern dürfen. Die Anmeldung soll einfach sein und am Handy lange angemeldet bleiben.
Nutzer werden in einer Konfigurationsdatei gepflegt, eine Nutzerverwaltung im UI gibt es nicht.

## Acceptance Criteria

- [ ] **AC1** — Ohne Anmeldung leitet jede Seite auf den Login um, und jeder API-Aufruf außer
  Login und Health liefert 401.
- [ ] **AC2** — Beide in `users.conf` konfigurierten Nutzer können sich anmelden; bei falschen
  Daten erscheint eine Fehlermeldung, die nicht verrät, ob Name oder Passwort falsch war.
- [ ] **AC3** — `photocurator hash-password` erzeugt einen Argon2id-Hash für `users.conf`;
  Klartext-Passwörter werden nie gespeichert.
- [ ] **AC4** — Das Session-Cookie ist `HttpOnly` und `SameSite=Strict`; eine Sitzung bleibt
  bis zum Ablauf (Platzhalter: 30 Tage) gültig, Abmelden beendet sie sofort.
- [ ] **AC5** — Nach zu vielen Fehlversuchen (Platzhalter: 5 pro Minute und IP) antwortet der
  Login mit 429, bis die Sperrzeit abgelaufen ist.
- [ ] **AC6** — Der angemeldete Nutzername ist in der Oberfläche sichtbar.
- [ ] **AC7** — Schreibende API-Aufrufe ohne Header `X-Requested-With` werden abgewiesen
  (CSRF-Schutz).

## Open Questions

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
