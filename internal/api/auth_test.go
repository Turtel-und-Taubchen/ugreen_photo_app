package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"photocurator/internal/api"
	"photocurator/internal/auth"
	"photocurator/internal/store"
)

const (
	annaPassword = "anna-pw"
	loginFailed  = "Anmeldung fehlgeschlagen. Name oder Passwort falsch."
)

var annaHash = sync.OnceValue(func() string {
	h, err := auth.HashPassword(annaPassword)
	if err != nil {
		panic(err)
	}
	return h
})

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

type testServer struct {
	h        http.Handler
	clock    *fakeClock
	sessions *auth.SessionStore
	checks   atomic.Int32 // password verifications actually run
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	db, err := store.OpenAppDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s := &testServer{clock: &fakeClock{t: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)}}
	s.sessions = auth.NewSessionStore(db, s.clock.Now)
	s.h = api.NewHandler(fstest.MapFS{
		"index.html":    {Data: []byte(indexHTML)},
		"assets/app.js": {Data: []byte("console.log(1)")},
	}, api.AuthDeps{
		Users:    auth.Users{"anna": annaHash()},
		Sessions: s.sessions,
		Limiter:  auth.NewLoginLimiter(s.clock.Now),
		Verify: func(phc, pw string) (bool, error) {
			s.checks.Add(1)
			ok, err := auth.VerifyPassword(phc, pw)
			if err != nil {
				t.Errorf("password check did not run Argon2: %v", err)
			}
			return ok, err
		},
	})
	return s
}

type reqOpt func(*http.Request)

func withCookie(c *http.Cookie) reqOpt { return func(r *http.Request) { r.AddCookie(c) } }

func withHeader(k, v string) reqOpt { return func(r *http.Request) { r.Header.Set(k, v) } }

var withXRW = withHeader("X-Requested-With", "XMLHttpRequest")

func withRemote(addr string) reqOpt { return func(r *http.Request) { r.RemoteAddr = addr } }

func (s *testServer) do(method, target, body string, opts ...reqOpt) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	for _, o := range opts {
		o(req)
	}
	rec := httptest.NewRecorder()
	s.h.ServeHTTP(rec, req)
	return rec
}

func (s *testServer) login(user, pw string, opts ...reqOpt) *httptest.ResponseRecorder {
	b, _ := json.Marshal(map[string]string{"username": user, "password": pw})
	return s.do(http.MethodPost, "/api/login", string(b), append([]reqOpt{withXRW}, opts...)...)
}

// signedIn creates a session directly in the store, skipping the slow password check.
func (s *testServer) signedIn(t *testing.T, user string) reqOpt {
	t.Helper()
	token, _, err := s.sessions.Create(user)
	if err != nil {
		t.Fatal(err)
	}
	return withCookie(&http.Cookie{Name: "pc_session", Value: token})
}

func sessionCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == "pc_session" {
			return c
		}
	}
	return nil
}

func wantJSONStatus(t *testing.T, label string, rec *httptest.ResponseRecorder, status int) {
	t.Helper()
	if rec.Code != status {
		t.Errorf("%s: status = %d, want %d (body %q)", label, rec.Code, status, rec.Body.String())
		return
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("%s: content-type = %q, want JSON", label, ct)
	}
}

func TestWithoutASessionEveryAPIPathButLoginAndHealthAnswers401(t *testing.T) {
	s := newTestServer(t)
	wantJSONStatus(t, "GET /api/health", s.do(http.MethodGet, "/api/health", ""), http.StatusOK)
	// Reaching the handler is visible from its own answer, not the gate's.
	if rec := s.login("nobody", "x"); rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), loginFailed) {
		t.Errorf("POST /api/login did not reach the login handler: %d %q", rec.Code, rec.Body.String())
	}

	for _, c := range []struct{ method, target string }{
		{http.MethodGet, "/api/me"},
		{http.MethodGet, "/api/does-not-exist"},
		{http.MethodPost, "/api/does-not-exist"},
		{http.MethodPost, "/api/logout"},
		{http.MethodGet, "/api/login"},
		{http.MethodPost, "/api/health"},
		{http.MethodGet, "/api"},
		{http.MethodGet, "/api/"},
		{http.MethodGet, "/api//health"},
		{http.MethodGet, "/api/health/"},
		{http.MethodGet, "/api/./health"},
		{http.MethodGet, "/api%2Fhealth"},
		{http.MethodGet, "/x/../api/me"},
		{http.MethodPost, "/api//login"},
		{http.MethodPost, "/api/login/"},
	} {
		label := c.method + " " + c.target
		rec := s.do(c.method, c.target, "{}", withXRW)
		wantJSONStatus(t, label, rec, http.StatusUnauthorized)
		if strings.Contains(rec.Body.String(), "photocurator-index") {
			t.Errorf("%s leaked the app shell", label)
		}
	}
}

func TestWithoutASessionAPageRequestIsRedirectedToTheLogin(t *testing.T) {
	s := newTestServer(t)
	for _, target := range []string{
		"/", "/review/42", "/index.html", "//index.html", "/assets/../index.html", "/./index.html",
		"/%69ndex.html", "/LOGIN", "/Login", "/login/", "/login/x", "//login", "/login/.",
		"/assets", "/assets/", "/assets//app.js", "/assets/app.js/", "/assets/%61pp.js", "/API/health",
	} {
		rec := s.do(http.MethodGet, target, "")
		if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/login" {
			t.Errorf("GET %s: got %d Location=%q, want 302 /login", target, rec.Code, rec.Header().Get("Location"))
		}
		if strings.Contains(rec.Body.String(), "photocurator-index") {
			t.Errorf("GET %s leaked the app shell", target)
		}
	}
	if rec := s.do(http.MethodGet, "/login", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "photocurator-index") {
		t.Errorf("GET /login: got %d %q, want the app shell", rec.Code, rec.Body.String())
	}
	for _, m := range []string{http.MethodGet, http.MethodHead} {
		if rec := s.do(m, "/assets/app.js", ""); rec.Code != http.StatusOK {
			t.Errorf("%s /assets/app.js: got %d, want 200", m, rec.Code)
		}
	}
	// Non-GET page requests are refused rather than redirected.
	for _, target := range []string{"/", "/login", "/assets/app.js"} {
		wantJSONStatus(t, "POST "+target, s.do(http.MethodPost, target, "", withXRW), http.StatusUnauthorized)
	}
}

func TestLoginSetsAnHttpOnlySameSiteStrictSessionCookieAndLogoutEndsTheSessionOnTheServer(t *testing.T) {
	s := newTestServer(t)
	rec := s.login("anna", annaPassword)
	wantJSONStatus(t, "login", rec, http.StatusOK)
	if !strings.Contains(rec.Body.String(), `"username":"anna"`) {
		t.Errorf("login body = %q", rec.Body.String())
	}
	c := sessionCookie(rec)
	if c == nil || c.Value == "" {
		t.Fatalf("no pc_session cookie in %q", rec.Header().Values("Set-Cookie"))
	}
	if !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Path != "/" || c.Secure ||
		c.MaxAge != int(auth.SessionTTL/time.Second) {
		t.Errorf("cookie attributes: %q", rec.Header().Get("Set-Cookie"))
	}
	session := withCookie(&http.Cookie{Name: c.Name, Value: c.Value})

	me := s.do(http.MethodGet, "/api/me", "", session)
	wantJSONStatus(t, "me", me, http.StatusOK)
	if strings.TrimSpace(me.Body.String()) != `{"username":"anna"}` {
		t.Errorf("me body = %q", me.Body.String())
	}
	if r := s.do(http.MethodGet, "/", "", session); r.Code != http.StatusOK || !strings.Contains(r.Body.String(), "photocurator-index") {
		t.Errorf("signed-in GET /: %d", r.Code)
	}
	if r := s.do(http.MethodGet, "/login", "", session); r.Code != http.StatusFound || r.Header().Get("Location") != "/" {
		t.Errorf("signed-in GET /login: %d Location=%q, want 302 /", r.Code, r.Header().Get("Location"))
	}

	out := s.do(http.MethodPost, "/api/logout", "", session, withXRW)
	if out.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d", out.Code)
	}
	if gone := sessionCookie(out); gone == nil || gone.MaxAge >= 0 {
		t.Errorf("logout did not expire the cookie: %q", out.Header().Get("Set-Cookie"))
	}
	wantJSONStatus(t, "me after logout", s.do(http.MethodGet, "/api/me", "", session), http.StatusUnauthorized)
}

func TestWrongPasswordAndUnknownUserGetTheSameAnswerAndBothRunOnePasswordCheck(t *testing.T) {
	s := newTestServer(t)
	var bodies []string
	for _, c := range []struct{ user, pw string }{
		{"anna", "wrong"},
		{"nobody", annaPassword},
		{"Anna", annaPassword}, // user names are case-sensitive
	} {
		before := s.checks.Load()
		rec := s.login(c.user, c.pw)
		wantJSONStatus(t, c.user, rec, http.StatusUnauthorized)
		if n := s.checks.Load() - before; n != 1 {
			t.Errorf("%s: %d password checks, want exactly 1", c.user, n)
		}
		if sessionCookie(rec) != nil {
			t.Errorf("%s: failed login set a session cookie", c.user)
		}
		bodies = append(bodies, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal([]byte(bodies[0]), &body); err != nil || body["error"] != loginFailed {
		t.Errorf("failure body = %q", bodies[0])
	}
	for _, b := range bodies[1:] {
		if b != bodies[0] {
			t.Errorf("answers differ: %q vs %q", b, bodies[0])
		}
	}
}

func TestAfterFiveFailedLoginsTheLoginAnswers429UntilTheLockoutEndsEvenWithXForwardedForChanged(t *testing.T) {
	s := newTestServer(t)
	for i := range 5 {
		if rec := s.login("anna", "wrong"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401", i+1, rec.Code)
		}
	}
	before := s.checks.Load()
	for _, xff := range []string{"", "10.0.0.9", "192.0.2.200"} {
		rec := s.login("anna", annaPassword, withHeader("X-Forwarded-For", xff), withHeader("X-Real-IP", xff))
		wantJSONStatus(t, "locked, X-Forwarded-For="+xff, rec, http.StatusTooManyRequests)
		if ra := rec.Header().Get("Retry-After"); ra != "60" {
			t.Errorf("Retry-After = %q, want 60", ra)
		}
		if sessionCookie(rec) != nil {
			t.Error("locked login set a session cookie")
		}
	}
	if n := s.checks.Load() - before; n != 0 {
		t.Errorf("locked logins ran %d password checks, want 0", n)
	}
	if rec := s.login("anna", annaPassword, withRemote("192.0.2.77:4000")); rec.Code != http.StatusOK {
		t.Errorf("another IP is locked too: %d", rec.Code)
	}

	s.clock.Advance(30 * time.Second)
	rec := s.login("anna", annaPassword)
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "30" {
		t.Errorf("mid-lockout: %d Retry-After=%q, want 429 30", rec.Code, rec.Header().Get("Retry-After"))
	}
	s.clock.Advance(30 * time.Second)
	if rec := s.login("anna", annaPassword); rec.Code != http.StatusOK {
		t.Errorf("after lockout: status = %d, want 200", rec.Code)
	}
}

func TestAWriteRequestWithoutXRequestedWithIsRefused(t *testing.T) {
	s := newTestServer(t)
	b, _ := json.Marshal(map[string]string{"username": "anna", "password": annaPassword})
	rec := s.do(http.MethodPost, "/api/login", string(b))
	wantJSONStatus(t, "login without header", rec, http.StatusForbidden)
	if sessionCookie(rec) != nil || s.checks.Load() != 0 {
		t.Error("a login without X-Requested-With was evaluated")
	}

	// Unauthenticated writes fail the session check first.
	wantJSONStatus(t, "anonymous logout", s.do(http.MethodPost, "/api/logout", ""), http.StatusUnauthorized)

	session := s.signedIn(t, "anna")
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		wantJSONStatus(t, m+" /api/does-not-exist", s.do(m, "/api/does-not-exist", "{}", session), http.StatusForbidden)
	}
	wantJSONStatus(t, "logout without header", s.do(http.MethodPost, "/api/logout", "", session), http.StatusForbidden)
	wantJSONStatus(t, "me after refused logout", s.do(http.MethodGet, "/api/me", "", session), http.StatusOK)
	wantJSONStatus(t, "write with header", s.do(http.MethodPost, "/api/does-not-exist", "{}", session, withXRW), http.StatusNotFound)
}

func TestASessionInUseIsRenewedOncePerDay(t *testing.T) {
	s := newTestServer(t)
	session := s.signedIn(t, "anna")
	if c := sessionCookie(s.do(http.MethodGet, "/api/me", "", session)); c != nil {
		t.Errorf("fresh session re-issued its cookie: %v", c)
	}
	s.clock.Advance(25 * time.Hour)
	rec := s.do(http.MethodGet, "/api/me", "", session)
	wantJSONStatus(t, "me after a day", rec, http.StatusOK)
	if c := sessionCookie(rec); c == nil || !c.HttpOnly || c.MaxAge != int(auth.SessionTTL/time.Second) {
		t.Errorf("renewed cookie: %q", rec.Header().Get("Set-Cookie"))
	}
}

func TestASessionOfAUserNoLongerInTheUsersFileIsRefused(t *testing.T) {
	s := newTestServer(t)
	wantJSONStatus(t, "removed user", s.do(http.MethodGet, "/api/me", "", s.signedIn(t, "carl")), http.StatusUnauthorized)
}
