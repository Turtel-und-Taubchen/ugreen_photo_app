---
id: 008
title: Mehrfach-, Bereichs- und Gruppenauswahl
status: ready # draft -> ready -> in-progress -> done
created: 2026-10-29
epic: E1 Index & Ansicht
estimate: 3
sprint: S02
---

## Requirement

Für jede Batch-Aktion muss ich schnell viele Fotos auswählen können: einzeln, als Bereich, einen
ganzen Tag oder Monat — auch Fotos, die noch nicht geladen sind. Am Handy ohne Tastatur.

## Acceptance Criteria

- [ ] **AC1** — Im Auswahlmodus markiert ein Klick/Tippen ein Foto bzw. hebt die Markierung auf;
  ein Zähler zeigt „n ausgewählt".
- [ ] **AC2** — Am PC wählt Shift+Klick alle Fotos zwischen dem zuletzt markierten und dem
  angeklickten aus.
- [ ] **AC3** — Am Handy startet langes Drücken den Auswahlmodus; die Aktion „Bereich bis hier"
  wählt alle Fotos zwischen dem zuletzt markierten und dem aktuellen aus.
- [ ] **AC4** — „Alle auswählen" an einer Tages- oder Monatsüberschrift wählt alle Fotos dieser
  Gruppe aus, auch die noch nicht geladenen; der Zähler zeigt die richtige Gesamtzahl.
- [ ] **AC5** — Die Auswahl bleibt beim Scrollen und beim Öffnen/Schließen der Detailansicht
  erhalten; „Auswahl aufheben" leert sie.
- [ ] **AC6** — Die Auswahl wird an den Server als Filter plus ein-/ausgeschlossene IDs
  übertragen; eine Auswahl „ganzer Monat" mit 2.000 Fotos ergibt eine Anfrage mit höchstens den
  Ausnahmen als Einzel-IDs, und der Server löst sie in genau die erwarteten Dateien auf.

## Open Questions

## Decisions (Sprint)

- Übertragungsformat ist `{ filter: { groups, ranges }, include_ids, exclude_ids }` mit
  Auflösung `(filter ∪ include_ids) \ exclude_ids`; ein leerer Filter trifft **nichts** — so
  bleibt die Form aus Tech Spec §6 erhalten, und eine leere Auswahl kann nie zur ganzen Bibliothek
  werden.
- `groups` sind Monats- (`JJJJ-MM`) bzw. Tagesschlüssel (`JJJJ-MM-TT`) auf `taken_at`, gebildet mit
  demselben Go-Helfer wie die Gruppen von 007 — sonst zählen Raster-Überschrift und Auswahl
  verschieden.
- Ein Bereich (Shift+Klick, „Bereich bis hier") wird als Keyset-Intervall `[von_id, bis_id]` in der
  Rastersortierung `(taken_at DESC, id DESC)` übertragen und vom Server aufgelöst — nur so umfasst er
  auch noch nicht geladene Fotos dazwischen, ohne IDs aufzuzählen.
- Der Zähler „n ausgewählt" kommt vom Server (`POST /api/selection/count`, dieselbe Auflösung wie
  später beim Job-Start), entprellt, nur die jüngste Antwort zählt — der Client kann Bereiche über
  ungeladene Fotos nicht selbst zählen, und eine einzige Wahrheit verhindert abweichende Zahlen.
- `POST` statt `GET` für die Zählung, weil Ausnahmelisten Tausende IDs lang sein dürfen
  (URL-Länge); den CSRF-Header setzt `api()` ohnehin.
- Der Auswahlmodus startet über den Knopf „Auswählen", langes Drücken, Shift+Klick oder „Alle
  auswählen" an einer Überschrift — alle Einstiege der AC plus ein expliziter Knopf für die Maus
  ohne Shift.
- Im Auswahlmodus schaltet Tippen die Markierung um; die Detailansicht öffnet dann ein eigener
  Knopf „Details öffnen" auf der Kachel (≥ 44 px) — AC5 verlangt die Detailansicht bei bestehender
  Auswahl, und Tippen ist im Modus belegt.
- „Auswahl aufheben" leert die Auswahl **und** beendet den Auswahlmodus — übliches Muster, ein
  leerer Modus hat keinen Zweck.
- Überschriften bieten nur „Alle auswählen" (bei vollständig gewählter Gruppe deaktiviert), kein
  Gruppen-Abwählen — AC4 verlangt nur das Auswählen; Abwählen eines Tages in einem gewählten Monat
  bräuchte negative Gruppenklauseln.
- Langes Drücken = Zeiger ≥ 500 ms gedrückt, ≤ 10 px bewegt, jeder Zeigertyp; der folgende Klick
  und `contextmenu` der Kachel werden unterdrückt. „Bereich bis hier" erscheint als Aktion, wenn im
  Auswahlmodus lange auf ein Foto gedrückt wird, ohne Anker deaktiviert — so bleibt Tippen =
  Umschalten eindeutig.
- Anker („zuletzt markiert") ist das zuletzt **an**geschaltete Foto (Tippen, Bereichsziel, langes
  Drücken); Abwählen verschiebt ihn nicht — gängiges Galerie-Verhalten und wörtlich AC2/AC3.
- Die Auswahl lebt im Speicher der Seite (Store außerhalb der Kacheln), nicht über Neuladen hinweg
  — AC5 verlangt nur Scrollen und Detailansicht.
- Server-Grenzen: je ID-Liste ≤ 10.000, Gruppen + Bereiche ≤ 500, Body ≤ 1 MiB; falsches Format oder
  unbekannte Bereichs-ID → 400 — Schutz des NAS vor unbegrenzten Anfragen.
- Ein Ansichtsfilter (Arbeitslisten, 018) gehört nicht zu 008; er kommt später als zusätzliche
  UND-Bedingung in denselben `filter` — S02 hat noch keinen Ansichtsfilter.
- Es gibt kein `CLAUDE.md` und keine Systemdoku unter `docs/systems/`; Leitplanken kommen aus Profil
  und Tech Spec, eine Doku-D entfällt.

## Plan

Baut auf 005 (Tabelle `photos` in `index.db`, Paket `internal/library`) und 007 (virtualisiertes
Raster, Gruppen-API, Detailansicht, große E2E-Fixture) auf. Die Namen unten sind die erwarteten;
hat 005/007 im Repo andere Namen, gelten diese.

1. **Server-Auflösung (D1):** `internal/library/selection.go` — Typ `Selection`, `Validate`, ein
   gemeinsamer SQL-WHERE-Baustein, `ResolveSelection` (IDs + `rel_path` in Rastersortierung; nutzt
   später der Job-Start in 012) und `CountSelection` aus demselben Baustein.
2. **API (D2):** `POST /api/selection/count` → `{count}` in `internal/api/selection.go`, registriert
   in `internal/api/server.go` (hinter Session + CSRF wie alles).
3. **Auswahlmodell (D3):** reines TS-Modul `web/src/lib/selection.ts` — umschalten, Anker, Bereich,
   Gruppe, leeren, `isSelected`, `toRequest()`; normalisiert (Gruppe wählen entfernt überflüssige
   Ein-/Ausschlüsse darin). Vitest.
4. **UI Desktop (D4):** Auswahlmodus im Raster von 007 — Leiste mit Server-Zähler und „Auswahl
   aufheben", Knopf „Auswählen", Klick, Shift+Klick, „Alle auswählen" an Überschriften, „Details
   öffnen"; Zustand im Store, damit die Virtualisierung ihn nicht verliert. E2E-Flow Desktop.
5. **UI Handy (D5):** langes Drücken als Svelte-Action + „Bereich bis hier". E2E-Flow Handy.

Reihenfolge: D1 → D2 → D3 → D4 → D5 (D3 hängt nicht an D1/D2).

## Deliverables

- [ ] **D1 — Auswahl-Auflösung im Server** (`internal/library/selection.go`,
  `internal/library/selection_test.go`). Typ `Selection` mit JSON-Form
  `{"filter":{"groups":["2024-07","2024-07-15"],"ranges":[[fromID,toID]]},"include_ids":[…],"exclude_ids":[…]}`.
  Semantik: Treffer = (Foto in einer Gruppe **oder** in einem Bereich **oder** in `include_ids`)
  **und nicht** in `exclude_ids`; leerer Filter ohne Includes → leeres Ergebnis (nie „alles").
  Gruppe = Monats- (`JJJJ-MM`) oder Tagesschlüssel (`JJJJ-MM-TT`) auf `taken_at`; den Schlüssel
  bildet **derselbe** Helfer/SQL-Ausdruck, mit dem 007s `/api/photos/groups` gruppiert (steht er
  bei 007 inline: in eine exportierte Funktion in `internal/library` herausziehen und von beiden
  nutzen — keine zweite Kopie). Bereich = beide Endpunkte per ID nachschlagen, alle Fotos mit
  `(taken_at, id)` zwischen beiden einschließlich, Reihenfolge der Endpunkte egal; unbekannte ID →
  Validierungsfehler. `Validate`: Schlüsselformat, je ID-Liste ≤ 10.000, Gruppen + Bereiche ≤ 500.
  Ein parametrisierter WHERE-Baustein (ID-Listen über `json_each(?)` oder temporäre Tabelle, nie
  Werte per String-Verkettung) für `ResolveSelection(ctx, db, sel) ([]Resolved{ID, RelPath}, error)`
  (Sortierung `taken_at DESC, id DESC`) und `CountSelection(ctx, db, sel) (int, error)`. Tests gegen
  eine temporäre `index.db` (Schema/Öffnen aus 005) mit 2.000 Fotos im Juli 2024 plus Randfotos
  2024-06-30T23:59:59 und 2024-08-01T00:00:00: „a whole month of 2000 photos minus three exceptions
  resolves to exactly those files", „a day group and an include id from another month add up
  without duplicates", „a range between two ids covers every photo between them in grid order,
  whichever end comes first", „an empty selection resolves to nothing", „an excluded id wins over
  its group and over include_ids", „count and resolve agree", „malformed group keys, unknown range
  ids and oversized lists are rejected".
- [ ] **D2 — Endpunkt `POST /api/selection/count`** (`internal/api/selection.go`,
  `internal/api/selection_test.go`, Registrierung in `internal/api/server.go`; Handler-Muster und
  `writeJSON` wie in `internal/api/auth.go`). Body = `library.Selection`, Antwort
  `200 {"count": n}` aus `CountSelection`; Fehler aus `Validate` oder ungültiges JSON →
  `400 {"error": …}`; Body per `http.MaxBytesReader` auf 1 MiB begrenzt. Session und CSRF kommen von
  der bestehenden Middleware (nichts Eigenes bauen). Tests: „counting a month selection with
  exclusions answers the resolved count", „an invalid selection answers 400".
- [ ] **D3 — Auswahlmodell im Frontend** (`web/src/lib/selection.ts`,
  `web/src/lib/selection.test.ts`). Reines TypeScript, kein Svelte, kein Fetch. Foto-Referenz
  `{id, takenAt}`; Sortierung wie im Raster (`takenAt` absteigend, dann `id` absteigend). Zustand:
  `groups` (Monat/Tag), `ranges` (Endpunkt-Referenzen), `include`, `exclude` (Referenzen, damit die
  Gruppenzugehörigkeit prüfbar ist), `anchor`, `active` (Auswahlmodus). Funktionen: `toggle(ref)`
  (gewählt → aus `include` entfernen bzw. in `exclude`; nicht gewählt → aus `exclude` entfernen bzw.
  in `include`; Anschalten setzt den Anker und `active`), `selectRange(ref)` (Bereich Anker→ref;
  ohne Anker wie Anschalten; Ziel wird Anker; entfernt `exclude` im Bereich), `selectGroup(key)`
  (entfernt `exclude`/`include` innerhalb der Gruppe; Tag in schon gewähltem Monat ist No-op),
  `isGroupFullySelected(key)`, `isSelected(ref)`, `clear()` (leert, `active = false`),
  `toRequest()` → genau die JSON-Form von `POST /api/selection/count`:
  `{filter:{groups:string[], ranges:[number,number][]}, include_ids:number[], exclude_ids:number[]}`
  (Bereiche nur als ID-Paare). Tests: „toggling a photo twice leaves nothing selected",
  „deselecting a photo inside a selected month becomes a single exclusion", „a whole month is sent
  as one group, not as ids", „selecting a group drops the exceptions inside it", „a range runs from
  the last marked photo, not from the last deselected", „clear empties the selection and leaves
  selection mode".
- [ ] **D4 — Auswahlmodus im Raster (Desktop)** — Dateien: Raster-, Kachel- und
  Überschrift-Komponenten aus 007 (erwartet unter `web/src/lib/grid/`), neu
  `web/src/lib/selectionStore.svelte.ts` (Svelte-5-`$state` um das Modell aus
  `web/src/lib/selection.ts` + Server-Zählung), neu `web/src/lib/SelectionBar.svelte`, neu
  `web/tests/e2e/selection.spec.ts` (Login über `web/tests/e2e/helpers/auth.ts`). Verhalten: Knopf
  „Auswählen" startet den Modus; im Modus schaltet Klick/Tippen eine Kachel um (sichtbar markiert,
  `aria-pressed`), Shift+Klick ruft `selectRange` (auch außerhalb des Modus, startet ihn); jede
  Überschrift (Monat und Tag) hat „Alle auswählen" (≥ 44 px, deaktiviert bei
  `isGroupFullySelected`), startet den Modus. Die Leiste zeigt „n ausgewählt" — n vom Server: nach
  jeder Änderung entprellt (~150 ms) `api('/api/selection/count', {method: 'POST', body:
  toRequest()})` aus `web/src/lib/api.ts`, Antworten älter als die letzte Anfrage verwerfen — und
  „Auswahl aufheben" (`clear`, Leiste verschwindet). Im Modus trägt jede Kachel „Details öffnen"
  (≥ 44 px), das die Detailansicht von 007 öffnet; außerhalb des Modus öffnet Klick wie bisher.
  Markierung wird aus dem Store abgeleitet, nie in der Kachel gehalten (Virtualisierung baut
  Kacheln neu). Die E2E nutzt die große Fixture von 007 und einen Monat mit mehr Fotos als das
  DOM-Limit (Platzhalter 300); hat 007s Fixture keinen solchen Monat, wird ihr Generator so
  angepasst, dass einer ≥ 400 Fotos hat. Zahlen liest der Test aus der Überschrift, nicht
  hartkodiert. Flow „selecting photos one by one, as a range and as a whole month survives
  scrolling and the detail view": „Auswählen"; Kacheln T1, T2 (Rasterreihenfolge) klicken → „2 ausgewählt";
  T2 erneut → „1 ausgewählt" (Anker bleibt T2); Shift+Klick auf T5 → T2..T5 + T1 = „5 ausgewählt";
  „Auswahl aufheben"; „Alle
  auswählen" am großen Monat (vorher prüfen: Kacheln der Gruppe im DOM < Gruppenzahl) → Zähler =
  Gruppenzahl; eine Kachel abwählen und den Request per `page.waitForRequest` abfangen:
  `filter.groups` = [Monat], `include_ids` leer, `exclude_ids` Länge 1, Zähler = Gruppenzahl − 1;
  ans Ende scrollen und zurück, Detail öffnen/schließen → Zähler und Markierung unverändert;
  „Auswahl aufheben" → Leiste weg.
- [ ] **D5 — Langes Drücken und „Bereich bis hier" (Handy)** (neu `web/src/lib/longpress.ts` als
  Svelte-Action + `web/src/lib/longpress.test.ts`, Kachel-Komponente aus 007 (erwartet unter
  `web/src/lib/grid/`), `web/src/lib/SelectionBar.svelte` bzw. ein kleines Aktionsmenü daneben,
  `web/tests/e2e/selection.spec.ts`). Action: Zeiger (jeder `pointerType`) ≥ 500 ms gedrückt und
  ≤ 10 px bewegt → Callback; der direkt folgende `click` wird unterdrückt, `contextmenu` auf Kacheln
  per `preventDefault`. Außerhalb des Auswahlmodus: langes Drücken startet ihn und schaltet das Foto
  an (`toggle` im Store aus `web/src/lib/selectionStore.svelte.ts`, wird Anker). Im Modus: langes
  Drücken öffnet ein Aktionsmenü mit „Bereich bis hier" (ruft `selectRange` des Stores; ohne Anker
  deaktiviert). Unit: „a press held 500 ms fires once and swallows the following click", „moving
  more than 10 px cancels the long press". E2E in einem `test.describe` mit
  `test.use({ viewport: { width: 390, height: 844 }, hasTouch: true })`, langes Drücken per
  `page.mouse.down()` / 600 ms warten / `page.mouse.up()`: Flow „on a phone a long press starts
  selection and 'Bereich bis hier' selects the range": langes Drücken auf Kachel T1 → Modus, „1
  ausgewählt"; langes Drücken auf T4 (Rasterreihenfolge) → „Bereich bis hier" → T1..T4 = „4
  ausgewählt".

## Model Hints

- Keine `deliverable-hard`: D1 ist abgegrenzte SQL-Logik mit dichten Tests, D3 reine Funktionen,
  D4/D5 nageln die E2E-Flows über die echte Oberfläche fest.
- Review: → default — die naheliegenden falschen Umsetzungen (leere Auswahl = alles, Monat als
  2.000 IDs übertragen, Bereich nur über geladene Kacheln, anderer Gruppenschlüssel als das Raster)
  sind je eigener Test in D1/D3/D4.

## Acceptance Tests

- AC1 → e2e `web/tests/e2e/selection.spec.ts` › "selecting photos one by one, as a range and as a
  whole month survives scrolling and the detail view" + unit `web/src/lib/selection.test.ts` ›
  "toggling a photo twice leaves nothing selected" (D3, D4)
- AC2 → e2e `web/tests/e2e/selection.spec.ts` › "selecting photos one by one, as a range and as a
  whole month survives scrolling and the detail view" (Shift+Klick) + unit
  `web/src/lib/selection.test.ts` › "a range runs from the last marked photo, not from the last
  deselected"; `internal/library/selection_test.go` › "a range between two ids covers every photo
  between them in grid order, whichever end comes first" (D1, D3, D4)
- AC3 → e2e `web/tests/e2e/selection.spec.ts` › "on a phone a long press starts selection and
  'Bereich bis hier' selects the range" + unit `web/src/lib/longpress.test.ts` › "a press held
  500 ms fires once and swallows the following click", "moving more than 10 px cancels the long
  press" (D5)
- AC4 → e2e `web/tests/e2e/selection.spec.ts` › "selecting photos one by one, as a range and as a
  whole month survives scrolling and the detail view" (Monat mit ungeladenen Fotos, Zähler =
  Gruppenzahl) + unit `internal/library/selection_test.go` › "a day group and an include id from
  another month add up without duplicates" (D1, D4)
- AC5 → e2e `web/tests/e2e/selection.spec.ts` › "selecting photos one by one, as a range and as a
  whole month survives scrolling and the detail view" (Scrollen, Detail, „Auswahl aufheben") + unit
  `web/src/lib/selection.test.ts` › "clear empties the selection and leaves selection mode" (D3, D4)
- AC6 → unit `internal/library/selection_test.go` › "a whole month of 2000 photos minus three
  exceptions resolves to exactly those files", "an empty selection resolves to nothing", "an
  excluded id wins over its group and over include_ids", "count and resolve agree";
  `internal/api/selection_test.go` › "counting a month selection with exclusions answers the
  resolved count"; `web/src/lib/selection.test.ts` › "a whole month is sent as one group, not as
  ids", "deselecting a photo inside a selected month becomes a single exclusion"; e2e
  `web/tests/e2e/selection.spec.ts` › "selecting photos one by one, as a range and as a whole month
  survives scrolling and the detail view" (abgefangener Request) (D1, D2, D3, D4)

## Done
