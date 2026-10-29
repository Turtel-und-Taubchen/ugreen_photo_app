---
id: 004
title: Einfache Anmeldung für zwei Nutzer
status: done # draft -> ready -> in-progress -> done
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

- [x] **AC1** — Ohne Anmeldung leitet jede Seite auf den Login um, und jeder API-Aufruf außer
  Login und Health liefert 401.
- [x] **AC2** — Beide in `users.conf` konfigurierten Nutzer können sich anmelden; bei falschen
  Daten erscheint eine Fehlermeldung, die nicht verrät, ob Name oder Passwort falsch war.
- [x] **AC3** — `photocurator hash-password` erzeugt einen Argon2id-Hash für `users.conf`;
  Klartext-Passwörter werden nie gespeichert.
- [x] **AC4** — Das Session-Cookie ist `HttpOnly` und `SameSite=Strict`; eine Sitzung bleibt
  bis zum Ablauf (Platzhalter: 30 Tage) gültig, Abmelden beendet sie sofort.
- [x] **AC5** — Nach zu vielen Fehlversuchen (Platzhalter: 5 pro Minute und IP) antwortet der
  Login mit 429, bis die Sperrzeit abgelaufen ist.
- [x] **AC6** — Der angemeldete Nutzername ist in der Oberfläche sichtbar.
- [x] **AC7** — Schreibende API-Aufrufe ohne Header `X-Requested-With` werden abgewiesen
  (CSRF-Schutz).

## Open Questions

## Decisions (Sprint)

Keine `(User)`-Vorgaben; alle Punkte vom Refinement entschieden (Grundlage: Tech Spec §1, §2.2,
§6, §9, Story 003 `(User)` „HTTP im Heimnetz, kein Reverse-Proxy"). Ein `CLAUDE.md` existiert im
Repo nicht — Leitplanken kommen aus Tech Spec, Profil-Notes und BACKLOG-DoD.

- **`users.conf`-Format:** eine Zeile `name:$argon2id$…` (PHC-String), `#`-Kommentare und
  Leerzeilen erlaubt — einfachstes Format, das man auf dem NAS mit einem Editor pflegen kann.
- **Nutzeranzahl:** mindestens 1, keine Obergrenze erzwungen — „zwei Nutzer" ist der Einsatzfall,
  eine harte 2 wäre nur zusätzliche Fehlerquelle.
- **Fehlende/ungültige `users.conf` stoppt den Start** mit klarer Meldung (Datei fehlt, keine
  Nutzer, Zeile ohne Argon2id-Hash, doppelter Name) — gleiche Linie wie Story 003 AC3, und so
  kann kein Klartext-Passwort versehentlich in Betrieb gehen.
- **Pfad:** `PC_USERS_FILE`, Standard `/data/users.conf` — wie in Tech Spec §8 vorgesehen.
- **Argon2id-Parameter:** m = 19 MiB, t = 2, p = 1 (OWASP-Minimum), Parameter stehen im Hash und
  werden beim Prüfen daraus gelesen — Ressourcenschonung auf dem NAS hat Vorrang, zwei Nutzer
  brauchen kein teureres Profil.
- **`hash-password` liest das Passwort von stdin** (ohne Echo am Terminal, Pipe erlaubt) und gibt
  nur den Hash aus — kein Passwort in der Shell-History oder Prozessliste.
- **Benutzernamen sind case-sensitiv**, das Eingabefeld hat `autocapitalize="off"` — exakter
  Vergleich ist einfach, das Handy-Großschreiben wird im UI verhindert.
- **Unbekannter Name prüft trotzdem gegen einen Dummy-Hash** — sonst verrät die Antwortzeit, ob
  der Name existiert (AC2 gilt auch für Timing).
- **Sessions:** 256-bit-Zufalls-ID im Cookie `pc_session`, in `app.db` nur als SHA-256 gespeichert
  (Tabelle `sessions`: `id_hash`, `username`, `expires_at`) — wie Tech Spec §2.2/§9.
- **SQLite-Treiber `modernc.org/sqlite`** (reines Go, kein CGO) — `app.db` entsteht mit dieser
  Story als erste DB; kein C-Toolchain-Bedarf im Build-Stage. Story 005 darf für `index.db`
  begründet abweichen.
- **Ablauf gleitend:** 30 Tage (Platzhalter, Konstante in `internal/auth`) ab letzter
  Verlängerung; eine Sitzung wird höchstens einmal pro 24 h verlängert (Cookie `Max-Age` neu) —
  „am Handy lange angemeldet" ohne Schreibzugriff pro Request.
- **Sitzung eines aus `users.conf` entfernten Nutzers ist ungültig** — Entfernen aus der Datei
  plus Neustart muss den Zugang sicher beenden.
- **Cookie ohne `Secure`:** Transport ist HTTP im Heimnetz (Story 003, `(User)`), ein
  `Secure`-Cookie käme nie zurück; `Path=/`, `HttpOnly`, `SameSite=Strict`.
- **Abmelden** (`POST /api/logout`) löscht die Zeile in `app.db` und das Cookie — „sofort beenden"
  heißt serverseitig, nicht nur im Browser.
- **Rate-Limit:** nur Fehlversuche zählen, Schlüssel = IP aus `RemoteAddr` (X-Forwarded-For wird
  ignoriert, es gibt keinen Proxy — sonst wäre das Limit per Header umgehbar); ab dem 5.
  Fehlversuch innerhalb 60 s ist die IP 60 s gesperrt (Platzhalter), in der Sperre liefert jeder
  Login 429 mit `Retry-After`, auch mit richtigem Passwort; Erfolg setzt den Zähler zurück; Zustand
  nur im Speicher (Neustart hebt Sperre auf — für Heimnetz ausreichend).
- **Default-deny:** die Auth-Middleware umschließt den ganzen Server; offen sind nur
  `POST /api/login`, `GET /api/health`, `GET /login` und echte Dateien des eingebetteten
  Frontends (außer `index.html`) — neue Routen späterer Stories sind automatisch geschützt.
- **Seiten-Umleitung serverseitig** (302 → `/login`) für jede nicht-API-Navigation ohne Sitzung,
  zusätzlich leitet das Frontend bei 401 auf `/login` um — funktioniert auch bei Deep-Links und
  abgelaufenen Sitzungen im offenen Tab.
- **Reihenfolge 401 vor 403:** ohne Sitzung liefert jede geschützte API 401 (AC1), mit Sitzung
  prüft die CSRF-Regel; `POST /api/login` selbst verlangt den Header ebenfalls (403 ohne).
- **CSRF-Regel:** jede API-Anfrage mit Methode ≠ GET/HEAD/OPTIONS ohne `X-Requested-With`
  → 403; kein zusätzlicher Content-Type-Zwang — `SameSite=Strict` + Custom-Header (erzwingt
  Preflight) reicht laut Tech Spec §9 für den Schutz.
- **`GET /api/me`** liefert `{ "username": … }` — Quelle für die Namensanzeige (AC6) und den
  Sitzungscheck des Frontends.
- **Fehlermeldung:** einheitlich „Anmeldung fehlgeschlagen. Name oder Passwort falsch." (HTTP 401,
  gleicher Body für unbekannten Namen und falsches Passwort).
- **E2E-Nutzer:** `testdata/e2e/users.conf` mit zwei Testnutzern (`anna`, `ben`) und
  Hashes bekannter Testpasswörter; die Passwörter stehen nur im E2E-Helper — Fixture-Konvention
  aus Tech Spec §11, keine echten Zugangsdaten im Repo.

## Plan

Setzt auf dem Gerüst aus 002 (Go-Server, eingebettetes Frontend, `/api/health`, E2E-Compose) und
der Konfiguration aus 003 (Datenordner `/data`) auf. Reihenfolge = Abhängigkeit:

1. **D1 — Nutzer & Passwörter (Kern + CLI):** `internal/auth` bekommt Argon2id-Hash/Verify und
   den `users.conf`-Loader; `photocurator hash-password`; Start bricht ohne gültige
   `users.conf` ab. Weil der Start ab hier eine `users.conf` braucht, liefert D1 auch die
   E2E-Fixture und trägt sie in die E2E-Compose ein, plus Abschnitt „Nutzer anlegen" in
   `deploy/README.md`.
2. **D2 — Sitzungen & Sperre (Kern):** `app.db` (SQLite, Migration `sessions`), Session-Store
   mit gleitendem Ablauf, Fehlversuchs-Limiter. Beide mit injizierbarer Uhr, rein unit-getestet.
3. **D3 — HTTP-Schicht (hard):** Handler `login`/`logout`/`me`, Default-deny-Middleware
   (401 / Seiten-302), CSRF-Middleware (403), Verdrahtung im Server aus 002, Tech Spec §9
   nachziehen. httptest-Tests für AC1/4/5/7 und die Negativfälle (Timing, XFF, Hash in DB).
4. **D4 — Oberfläche + E2E-Flow:** Login-Seite, Kopfzeile mit Nutzername + Abmelden,
   API-Wrapper (Header `X-Requested-With`, 401 → `/login`); ein Playwright-Flow über den
   echten Pfad; bestehender Smoke-Test aus 002 meldet sich vorher an.

Nicht Teil: HTTPS, Nutzerverwaltung im UI, Passwort ändern im UI, persistente Sperrliste.

## Deliverables

- **D1 — Argon2id, `users.conf`, `hash-password`, Startprüfung.**
  Dateien: `internal/auth/password.go` (`HashPassword(pw) (string, error)`,
  `VerifyPassword(phc, pw) (bool, error)`; PHC-Format `$argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>`,
  16-Byte-Salt, 32-Byte-Key, `golang.org/x/crypto/argon2`, Vergleich mit
  `crypto/subtle.ConstantTimeCompare`, Parameter beim Verify aus dem String lesen),
  `internal/auth/users.go` (`LoadUsers(path) (Users, error)`: Zeilen `name:<phc>`, `#`-Kommentare,
  Leerzeilen; Fehler mit Zeilennummer bei fehlender Datei, 0 Nutzern, Wert ohne
  `$argon2id$`-Präfix, doppeltem Namen, leerem Namen), Tests `internal/auth/password_test.go`,
  `internal/auth/users_test.go`; `cmd/photocurator` — Subcommand `hash-password` (Passwort von
  stdin; am Terminal ohne Echo via `golang.org/x/term`, sonst erste Zeile der Pipe; Ausgabe nur
  der Hash + Zeilenumbruch; leeres Passwort → Fehler, Exit ≠ 0) plus Test
  `cmd/photocurator/hash_password_test.go`; `serve` lädt `PC_USERS_FILE` (Standard
  `/data/users.conf`, in `internal/config`) beim Start und bricht mit klarer Meldung ab.
  `testdata/e2e/users.conf` mit `anna` / `ben` (Passwörter `anna-e2e-pass` / `ben-e2e-pass`,
  Hashes mit dem neuen Subcommand erzeugt) und in der E2E-Compose-Datei aus Story 002 einbinden
  (Datei finden über `web/playwright.config.ts` bzw. dessen Global-Setup; Mount read-only +
  `PC_USERS_FILE`). Go-Tests, die `serve` starten, bekommen eine Test-`users.conf`.
  `deploy/README.md` (aus Story 003): Abschnitt „Nutzer anlegen" (Hash erzeugen per
  `docker run --rm -i photocurator:latest hash-password`, Zeile in `/data/users.conf`,
  Container neu starten). Tests: „a hash verifies the right password and rejects a wrong one",
  „hash-password prints an argon2id hash that never contains the password",
  „a users.conf line with a plaintext password stops the start" (deckt auch fehlende Datei /
  0 Nutzer / doppelter Name als Tabellenfälle).

- **D2 — `app.db`, Session-Store, Fehlversuchs-Limiter.**
  Dateien: `internal/store/appdb.go` (`OpenAppDB(dataDir)` öffnet `<dataDir>/app.db` mit
  `modernc.org/sqlite`, WAL, einfache versionierte Migration; Migration 1 =
  `CREATE TABLE sessions (id_hash TEXT PRIMARY KEY, username TEXT NOT NULL, expires_at INTEGER NOT NULL, renewed_at INTEGER NOT NULL)`),
  `internal/auth/sessions.go` (`SessionStore` mit `Create(user) (token, expires)`,
  `Lookup(token) (user, newExpires, ok)`, `Delete(token)`; Token = 32 Byte `crypto/rand`,
  base64url; gespeichert nur `sha256(token)` hex; TTL 30 Tage, Verlängerung höchstens alle 24 h,
  abgelaufene Zeilen werden beim Lookup bzw. bei `Create` gelöscht; Uhr als `func() time.Time`
  injizierbar), `internal/auth/ratelimit.go` (`LoginLimiter`: `Allowed(ip) (ok, retryAfter)`,
  `Fail(ip)`, `Success(ip)`; 5 Fehlversuche in 60 s → 60 s Sperre; Map mit Mutex, alte Einträge
  beim Zugriff aufräumen; Uhr injizierbar), Tests `internal/store/appdb_test.go`,
  `internal/auth/sessions_test.go`, `internal/auth/ratelimit_test.go` (temporäres Verzeichnis
  statt `/data`). Tests: „a session stays valid until 30 days without use and is gone after",
  „using a session at most once a day pushes its expiry forward",
  „the database stores only the hash of the session token",
  „the sixth attempt after five failures within a minute is refused until the lockout ends",
  „a successful login clears the failure count".

- **D3 — HTTP: Login/Logout/Me, Default-deny, CSRF.** Baut auf D1/D2 (`internal/auth`,
  `internal/store`) und dem Server aus Story 002 (Router/Server-Datei in `internal/api`, dort
  liegt `GET /api/health`). Dateien: `internal/api/auth.go` — `POST /api/login` (JSON
  `{username, password}`; Limiter prüfen → 429 + `Retry-After`; unbekannter Name → Verify gegen
  festen Dummy-Hash, damit immer genau eine Argon2-Prüfung läuft; Fehler → 401
  `{"error":"Anmeldung fehlgeschlagen. Name oder Passwort falsch."}` identisch für beide Fälle;
  Erfolg → Session + Cookie `pc_session` mit `HttpOnly`, `SameSite=Strict`, `Path=/`,
  `Max-Age`=TTL, ohne `Secure`), `POST /api/logout` (Session löschen, Cookie mit `Max-Age=-1`),
  `GET /api/me` → `{"username"}`; Passwort nie loggen. `internal/api/middleware.go` —
  `RequireSession` um den gesamten Handler: offen nur `POST /api/login`, `GET /api/health`,
  `GET /login` und Pfade, die eine echte Datei im eingebetteten Frontend sind (nicht
  `index.html`, exakte Pfadauflösung, kein Präfix-Match); ohne gültige Sitzung `/api/*` → 401
  JSON, sonst `GET` → 302 `/login`; gültige Sitzung → Nutzer in den Request-Context, bei
  Verlängerung Cookie neu setzen; angemeldeter Aufruf von `/login` → 302 `/`. `CSRF` — Methode
  ≠ GET/HEAD/OPTIONS unter `/api/` ohne `X-Requested-With` → 403 (läuft nach der Session-Prüfung,
  gilt auch für `/api/login`). IP für den Limiter nur aus `r.RemoteAddr`. Verdrahtung in der
  Server-Datei von 002 und in `cmd/photocurator` (`serve` öffnet `app.db`). Test
  `internal/api/auth_test.go` (httptest, Fake-Uhr, Verify-Zähler). `docs/concepts/photocurator-tech-spec.md`
  §9 + §6: `users.conf`-Format, Cookie ohne `Secure` (HTTP), `/api/me`, Default-deny-Liste.
  Tests: „without a session every API path but login and health answers 401" (inkl. erfundener
  Pfad `/api/does-not-exist` und `GET /api/me`), „without a session a page request is redirected
  to the login", „login sets an HttpOnly SameSite=Strict session cookie and logout ends the
  session on the server" (Cookie danach bei `/api/me` → 401), „wrong password and unknown user
  get the same answer and both run one password check", „after five failed logins the login
  answers 429 until the lockout ends, even with X-Forwarded-For changed", „a write request
  without X-Requested-With is refused".

- **D4 — Login-Seite, Nutzername, Abmelden, E2E-Flow.**
  Dateien (Svelte-Struktur aus Story 002 übernehmen): `web/src/lib/api.ts` (fetch-Wrapper: setzt
  bei jeder Anfrage `X-Requested-With: fetch`, JSON; bei 401 außerhalb des Logins →
  `location.assign('/login')`), Login-Ansicht (`web/src/routes/Login.svelte` o. ä. je nach
  Routing aus 002; Felder Name/Passwort mit `autocapitalize="off"`, `autocomplete="username"` /
  `"current-password"`, Fehlermeldung aus der API in `role="alert"`; Erfolg → `/`),
  Kopfzeile (`web/src/lib/AppHeader.svelte`: Name aus `GET /api/me` mit
  `data-testid="current-user"`, Knopf „Abmelden" → `POST /api/logout` → `/login`), Einbindung in
  die App-Hülle; ≥ 44 px Touch-Fläche, ab 360 px Breite (Tech Spec §7).
  `web/tests/e2e/helpers/auth.ts` (`login(page, user)`, Testnutzer `anna`/`ben` mit
  `anna-e2e-pass`/`ben-e2e-pass`); bestehender Smoke-Test aus Story 002 ruft vorher `login`
  auf. Neuer Flow `web/tests/e2e/auth.spec.ts` › „two users sign in, see their name and sign
  out": `/` und ein Deep-Link ohne Anmeldung landen auf `/login`; falsches Passwort und
  unbekannter Name zeigen dieselbe Meldung; `anna` meldet sich an, Name sichtbar, Cookie
  `pc_session` ist `httpOnly` und `sameSite: 'Strict'` (`context.cookies()`); Abmelden → `/login`,
  erneuter Aufruf von `/` → `/login`; danach dasselbe für `ben`. Fehlversuche des Flows (2)
  bleiben unter dem Limit.

## Model Hints

- D3 → deliverable-hard — die Default-deny-Middleware umschließt jede Route des 002-Servers:
  eine falsche Ausnahme (Präfix-Match, `index.html` als Asset, Reihenfolge 401/403) öffnet
  entweder die API oder sperrt Healthcheck bzw. die Assets der Login-Seite und bricht damit den
  E2E-Harness und den Compose-Healthcheck aus 003.
- Review: → default — die naheliegenden falschen Umsetzungen (Timing-Leck bei unbekanntem Namen,
  Klartext-Token in der DB, Limit über `X-Forwarded-For` umgehbar, Präfix-Ausnahmen) sind als
  eigene Tests in D2/D3 festgenagelt.

## Acceptance Tests

- AC1 → e2e `web/tests/e2e/auth.spec.ts` › "two users sign in, see their name and sign out"
  (Seiten-Umleitung) + unit `internal/api/auth_test.go` › "without a session every API path but
  login and health answers 401", "without a session a page request is redirected to the login"
  (D3, D4)
- AC2 → e2e `web/tests/e2e/auth.spec.ts` › "two users sign in, see their name and sign out"
  + unit `internal/api/auth_test.go` › "wrong password and unknown user get the same answer and
  both run one password check" (D3, D4)
- AC3 → unit `cmd/photocurator/hash_password_test.go` › "hash-password prints an argon2id hash
  that never contains the password"; `internal/auth/users_test.go` › "a users.conf line with a
  plaintext password stops the start"; `internal/auth/password_test.go` › "a hash verifies the
  right password and rejects a wrong one"; `internal/auth/sessions_test.go` › "the database
  stores only the hash of the session token" (D1, D2)
- AC4 → unit `internal/api/auth_test.go` › "login sets an HttpOnly SameSite=Strict session cookie
  and logout ends the session on the server"; `internal/auth/sessions_test.go` › "a session stays
  valid until 30 days without use and is gone after", "using a session at most once a day pushes
  its expiry forward"; e2e `web/tests/e2e/auth.spec.ts` › "two users sign in, see their name and
  sign out" (Cookie-Attribute, Abmelden) (D2, D3, D4)
- AC5 → unit `internal/auth/ratelimit_test.go` › "the sixth attempt after five failures within a
  minute is refused until the lockout ends", "a successful login clears the failure count";
  `internal/api/auth_test.go` › "after five failed logins the login answers 429 until the lockout
  ends, even with X-Forwarded-For changed" (D2, D3) — Ebene Go-Test statt E2E, weil die Sperrzeit
  nur mit Fake-Uhr ohne Warten prüfbar ist und der Limiter IP-weit gilt (würde parallele E2E-Tests
  sperren).
- AC6 → e2e `web/tests/e2e/auth.spec.ts` › "two users sign in, see their name and sign out" (D4)
- AC7 → unit `internal/api/auth_test.go` › "a write request without X-Requested-With is refused";
  dass das Frontend den Header immer sendet, belegt der E2E-Flow (Login/Logout gingen sonst mit
  403 schief) (D3, D4)

Coverage-Gate: AC1 D3+D4 · AC2 D1+D3+D4 · AC3 D1+D2 · AC4 D2+D3+D4 · AC5 D2+D3 · AC6 D4 ·
AC7 D3+D4 — jedes Kriterium hat Deliverable und Test, keine manual residue.

## Done

Zusammenfassung: Anmeldung für zwei Nutzer — Argon2id + `users.conf` + `photocurator hash-password`, `app.db` (modernc sqlite) mit Session-Store (30 Tage gleitend, nur SHA-256 in der DB), IP-Limiter (5/min → 429), Default-deny-Middleware + CSRF-Header-Pflicht, Login-Seite, Kopfzeile mit Nutzername/Abmelden, E2E-Flow.

Commit: `004: Anmeldung für zwei Nutzer — Argon2id/users.conf/hash-password, Sessions in app.db, Limiter, Default-deny+CSRF, Login-UI, E2E`

Verifikation (enge Gate; `test-story` none → volles `test`): build, `npm run check`, lint (docker + web), `docker build --target test .` + vitest (7), `e2e-story`: `cd web && npx playwright test tests/e2e/auth.spec.ts` — alles grün; nach Review-Fix erneut test/lint/e2e grün. Review: 1 Zyklus (default), Verdikt PASS, 3 Findings behoben. Volles Gate steht beim Sprint aus.
AC → Test: AC1 auth_test.go (401 / Seiten-Umleitung) + e2e auth.spec.ts · AC2 auth_test.go (gleiche Antwort, eine Prüfung) + e2e · AC3 hash_password_test, users_test, password_test, sessions_test (nur Hash in DB) · AC4 auth_test (Cookie, Logout), sessions_test (30 Tage / 1×täglich), e2e (Cookie-Attribute) · AC5 ratelimit_test + auth_test (429 trotz XFF) · AC6 e2e (`current-user`) · AC7 auth_test (403 ohne Header) + e2e. Go-Tests laufen ohne `-v`; Nachweis = Paket ok + Test per grep vorhanden. Keine manual residue.

Decisions: `users.conf` im E2E nach `/run/users.conf` gemountet (nicht unter `/data`, Kollision mit Datenmount), `PC_E2E_USERS_FILE` in `compose.ts`; nicht-GET auf Seitenpfaden ohne Sitzung → 401 JSON (Tech Spec §6); Logout 204; kaputter Login-Body → 400 ohne Fehlversuch; Sitzung eines entfernten Nutzers wird gelöscht; Dummy-Hash eager im Konstruktor. Offen/unbehoben: Limiter-Map räumt nur bei erneutem Zugriff der gleichen IP auf (Heimnetz vernachlässigbar); Session-DB-Lookup auch für Assets (nur Performance); 360-px-Layout nicht visuell geprüft; 30d-Grenze mit zwei Sitzungen statt einem Token getestet.

tiers: D 4 / hard 1 · review default · cycles 1 · agents 7
