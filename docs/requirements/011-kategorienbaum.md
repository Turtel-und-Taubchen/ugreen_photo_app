---
id: 011
title: Hierarchische Kategorien definieren und als Baum anzeigen
status: draft # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E3 Tags & Kategorien
estimate: 3
sprint: S03
---

## Requirement

Ich möchte meine eigene Ordnung festlegen — z. B. `Anlass/Geburtstag`, `Reise/Österreich`,
`Wir zwei` — und sie als Baum sehen, auch bevor ein Foto die Kategorie trägt. Tags und
Kategorien sind dasselbe Konzept: Eine Kategorie ist ein Tag mit Kindern.

## Acceptance Criteria

- [ ] **AC1** — Ein neuer Tag kann mit Pfad angelegt werden (`Anlass/Geburtstag`); fehlende
  Elternknoten entstehen automatisch.
- [ ] **AC2** — Die Tag-Verwaltung zeigt alle bekannten Tags als Baum — aus den Dateien (Index)
  und den angelegten, noch leeren Kategorien — jeweils mit Anzahl der Fotos.
- [ ] **AC3** — Ein Tag ohne Fotos kann gelöscht werden; für einen Tag mit Fotos ist Löschen
  gesperrt, mit dem Hinweis, ihn zuerst über „Tags entfernen" von den Fotos zu lösen.
- [ ] **AC4** — Ungültige Namen werden mit Erklärung abgewiesen: leere Segmente, Zeichen, die in
  den Zielfeldern als Trenner dienen (`|`, `;`, `,`, `/` innerhalb eines Segments), und Segmente
  über der Längengrenze der Zielfelder (IPTC: 64 Byte).
- [ ] **AC5** — Nach einem Index-Neuaufbau sind alle in Dateien vorhandenen Tags wieder im Baum;
  leere Kategorien bleiben erhalten (App-Zustand).

## Open Questions

## Decisions (Sprint)

## Plan

## Deliverables

## Model Hints

## Acceptance Tests

## Done
