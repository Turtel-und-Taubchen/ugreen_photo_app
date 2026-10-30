package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"photocurator/internal/api"
	"photocurator/internal/geo"
	"photocurator/internal/store"
)

// nominatimFake counts requests and records the fake time each one arrived.
type nominatimFake struct {
	mu    sync.Mutex
	times []time.Time
	clock *fakeClock
	srv   *httptest.Server
	code  int
}

func newNominatimFake(t *testing.T, clock *fakeClock) *nominatimFake {
	t.Helper()
	f := &nominatimFake{clock: clock, code: http.StatusOK}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		f.times = append(f.times, clock.Now())
		code := f.code
		f.mu.Unlock()
		w.WriteHeader(code)
		if code == http.StatusOK {
			_, _ = w.Write([]byte("[]"))
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *nominatimFake) arrivals() []time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]time.Time(nil), f.times...)
}

// geoServer wires a real Geocoder whose clock and sleeping are fake: a wait advances the clock.
func geoServer(t *testing.T, contact string) (*testServer, *nominatimFake) {
	t.Helper()
	app, err := store.OpenAppDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.Close() })
	clock := &fakeClock{t: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)}
	fake := newNominatimFake(t, clock)
	g := geo.New(geo.Options{
		BaseURL: fake.srv.URL,
		Contact: contact,
		Version: "test",
		DB:      app,
		Now:     clock.Now,
		Sleep:   func(_ context.Context, d time.Duration) error { clock.Advance(d); return nil },
	})
	return newTestServerDeps(t, api.LibraryDeps{Geo: g}), fake
}

func TestSearchesOfTwoUsersShareOneLimitOfOneRequestPerSecond(t *testing.T) {
	s, fake := geoServer(t, "a@example.invalid")
	anna, ben := s.signedIn(t, "anna"), s.signedIn(t, "ben")
	for i, c := range []struct {
		q    string
		sess reqOpt
	}{{"Wien", anna}, {"Graz", ben}, {"Linz", anna}} {
		rec := s.do(http.MethodGet, "/api/geo/search?q="+c.q, "", c.sess)
		wantJSONStatus(t, "search "+c.q, rec, http.StatusOK)
		var got struct {
			Results []geo.Place `json:"results"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Results == nil {
			t.Fatalf("search %d: results must be an array, body %s", i, rec.Body.String())
		}
	}
	at := fake.arrivals()
	if len(at) != 3 {
		t.Fatalf("upstream requests = %d, want 3", len(at))
	}
	for i := 1; i < len(at); i++ {
		if gap := at[i].Sub(at[i-1]); gap < time.Second {
			t.Errorf("request %d came %v after the previous, want at least 1s", i, gap)
		}
	}
}

func TestWithoutAContactThePlaceSearchReportsWhyAndSendsNothing(t *testing.T) {
	s, fake := geoServer(t, "")
	sess := s.signedIn(t, "anna")

	rec := s.do(http.MethodGet, "/api/geo/status", "", sess)
	wantJSONStatus(t, "status", rec, http.StatusOK)
	var st struct {
		Enabled bool   `json:"enabled"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &st); err != nil {
		t.Fatal(err)
	}
	if st.Enabled || st.Reason == "" {
		t.Errorf("status = %+v, want disabled with a reason", st)
	}
	for _, target := range []string{"/api/geo/search?q=Wien", "/api/geo/reverse?lat=48.2&lon=16.3"} {
		rec := s.do(http.MethodGet, target, "", sess)
		wantJSONStatus(t, target, rec, http.StatusServiceUnavailable)
		var e struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil || e.Error != st.Reason {
			t.Errorf("%s: error = %q, want %q", target, e.Error, st.Reason)
		}
	}
	if n := len(fake.arrivals()); n != 0 {
		t.Errorf("upstream requests = %d, want 0", n)
	}
}

func TestANominatimOutageAnswers502WithAReadableMessage(t *testing.T) {
	s, fake := geoServer(t, "a@example.invalid")
	fake.code = http.StatusServiceUnavailable
	rec := s.do(http.MethodGet, "/api/geo/search?q=Wien", "", s.signedIn(t, "anna"))
	wantJSONStatus(t, "search", rec, http.StatusBadGateway)
	var e struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatal(err)
	}
	if want := "Ortssuche gerade nicht erreichbar. Du kannst den Ort per Koordinaten und Namen selbst setzen."; e.Error != want {
		t.Errorf("error = %q, want %q", e.Error, want)
	}
}

func TestAnEmptySearchOrInvalidCoordinatesAreRefusedWithoutAskingNominatim(t *testing.T) {
	s, fake := geoServer(t, "a@example.invalid")
	sess := s.signedIn(t, "anna")
	tooLong := make([]byte, 201)
	for i := range tooLong {
		tooLong[i] = 'a'
	}
	for _, target := range []string{
		"/api/geo/search",
		"/api/geo/search?q=",
		"/api/geo/search?q=%20%20",
		"/api/geo/search?q=" + string(tooLong),
		"/api/geo/reverse",
		"/api/geo/reverse?lat=abc&lon=1",
		"/api/geo/reverse?lat=91&lon=1",
		"/api/geo/reverse?lat=1&lon=-181",
		"/api/geo/reverse?lat=NaN&lon=1",
		"/api/geo/reverse?lat=1&lon=Inf",
	} {
		wantJSONStatus(t, target, s.do(http.MethodGet, target, "", sess), http.StatusBadRequest)
	}
	if n := len(fake.arrivals()); n != 0 {
		t.Errorf("upstream requests = %d, want 0", n)
	}
}

func TestThePlaceSearchNeedsASession(t *testing.T) {
	s, fake := geoServer(t, "a@example.invalid")
	wantJSONStatus(t, "search", s.do(http.MethodGet, "/api/geo/search?q=Wien", ""), http.StatusUnauthorized)
	if n := len(fake.arrivals()); n != 0 {
		t.Errorf("upstream requests = %d, want 0", n)
	}
}
