package geo

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sync"
	"testing"
	"time"

	"photocurator/internal/store"
)

const (
	testContact   = "curator@example.org"
	testUserAgent = "PhotoCurator/1.2.3 (curator@example.org)"
)

// fakeClock never waits: Sleep moves the clock forward instead.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func (c *fakeClock) Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.Advance(d)
	return nil
}

type upstreamRequest struct {
	Path      string
	Query     url.Values
	UserAgent string
	At        time.Time
}

// upstream is a fake Nominatim that records every request it receives with the fake time.
type upstream struct {
	*httptest.Server
	clock    *fakeClock
	mu       sync.Mutex
	requests []upstreamRequest
}

func newUpstream(t *testing.T, clock *fakeClock, respond http.HandlerFunc) *upstream {
	t.Helper()
	u := &upstream{clock: clock}
	u.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u.mu.Lock()
		u.requests = append(u.requests, upstreamRequest{
			Path: r.URL.Path, Query: r.URL.Query(), UserAgent: r.UserAgent(), At: clock.Now(),
		})
		u.mu.Unlock()
		respond(w, r)
	}))
	t.Cleanup(u.Close)
	return u
}

func (u *upstream) Requests() []upstreamRequest {
	u.mu.Lock()
	defer u.mu.Unlock()
	return append([]upstreamRequest(nil), u.requests...)
}

// serveFixtures answers like Nominatim from testdata: reverse with Saalfelden, a search for
// "nowhere" with nothing, every other search with Saalfelden.
func serveFixtures(t *testing.T) http.HandlerFunc {
	search := fixture(t, "search-saalfelden.json")
	reverse := fixture(t, "reverse-saalfelden.json")
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/reverse":
			w.Write(reverse)
		case r.URL.Query().Get("q") == "nowhere":
			w.Write([]byte("[]"))
		default:
			w.Write(search)
		}
	}
}

func openDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := store.OpenAppDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func newTestGeocoder(t *testing.T, clock *fakeClock, baseURL string, db *sql.DB) *Geocoder {
	return New(Options{
		BaseURL: baseURL, Contact: testContact, Version: "1.2.3", DB: db,
		Now: clock.Now, Sleep: clock.Sleep,
	})
}

func cacheRows(t *testing.T, db *sql.DB) map[string]int64 {
	t.Helper()
	rows, err := db.Query(`SELECT key, fetched_at FROM geocode_cache`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	got := map[string]int64{}
	for rows.Next() {
		var key string
		var at int64
		if err := rows.Scan(&key, &at); err != nil {
			t.Fatal(err)
		}
		got[key] = at
	}
	return got
}

func assertSpacedAtLeastOneSecond(t *testing.T, reqs []upstreamRequest) {
	t.Helper()
	for i := 1; i < len(reqs); i++ {
		if gap := reqs[i].At.Sub(reqs[i-1].At); gap < time.Second {
			t.Errorf("request %d started %v after the previous one, want at least 1s", i, gap)
		}
	}
}

func runConcurrently(n int, f func(i int)) {
	var start, done sync.WaitGroup
	start.Add(1)
	for i := 0; i < n; i++ {
		done.Add(1)
		go func() {
			defer done.Done()
			start.Wait()
			f(i)
		}()
	}
	start.Done()
	done.Wait()
}

func TestFiveConcurrentSearchesReachNominatimAtLeastOneSecondApart(t *testing.T) {
	clock := newFakeClock()
	up := newUpstream(t, clock, serveFixtures(t))
	g := newTestGeocoder(t, clock, up.URL, openDB(t))

	queries := []string{"Saalfelden", "Zell am See", "Maria Alm", "Leogang", "Lofer"}
	errs := make([]error, len(queries))
	runConcurrently(len(queries), func(i int) {
		_, errs[i] = g.Search(context.Background(), queries[i])
	})
	for i, err := range errs {
		if err != nil {
			t.Fatalf("search %q: %v", queries[i], err)
		}
	}
	reqs := up.Requests()
	if len(reqs) != len(queries) {
		t.Fatalf("upstream saw %d requests, want %d", len(reqs), len(queries))
	}
	assertSpacedAtLeastOneSecond(t, reqs)
}

func TestConcurrentIdenticalSearchesReachNominatimOnce(t *testing.T) {
	clock := newFakeClock()
	up := newUpstream(t, clock, serveFixtures(t))
	g := newTestGeocoder(t, clock, up.URL, openDB(t))

	results := make([][]Place, 5)
	errs := make([]error, 5)
	runConcurrently(5, func(i int) {
		results[i], errs[i] = g.Search(context.Background(), "Saalfelden")
	})
	for i := range results {
		if errs[i] != nil {
			t.Fatal(errs[i])
		}
		if len(results[i]) == 0 || !reflect.DeepEqual(results[i], results[0]) {
			t.Errorf("caller %d got %v, want the shared answer %v", i, results[i], results[0])
		}
	}
	if n := len(up.Requests()); n != 1 {
		t.Fatalf("upstream saw %d requests, want 1", n)
	}
}

func TestASearchAsksNominatimWithJsonv2AddressAndNameDetailsGermanAndTenResultsAndOurUserAgent(t *testing.T) {
	clock := newFakeClock()
	up := newUpstream(t, clock, func(w http.ResponseWriter, r *http.Request) {
		w.Write(fixture(t, "search-ritzenhofer-saalfelden.json"))
	})
	g := newTestGeocoder(t, clock, up.URL+"/", openDB(t))

	places, err := g.Search(context.Background(), "  Ritzenhofer   Saalfelden ")
	if err != nil {
		t.Fatal(err)
	}
	want, err := ParseSearch(fixture(t, "search-ritzenhofer-saalfelden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(places, want) {
		t.Errorf("places = %v, want %v", places, want)
	}
	reqs := up.Requests()
	if len(reqs) != 1 {
		t.Fatalf("upstream saw %d requests, want 1", len(reqs))
	}
	req := reqs[0]
	if req.Path != "/search" {
		t.Errorf("path = %q, want /search", req.Path)
	}
	wantQuery := url.Values{
		"q": {"Ritzenhofer   Saalfelden"}, "format": {"jsonv2"}, "addressdetails": {"1"},
		"namedetails": {"1"}, "accept-language": {"de"}, "limit": {"10"},
	}
	if !reflect.DeepEqual(req.Query, wantQuery) {
		t.Errorf("query = %v, want %v", req.Query, wantQuery)
	}
	if req.UserAgent != testUserAgent {
		t.Errorf("User-Agent = %q, want %q", req.UserAgent, testUserAgent)
	}
}

func TestARepeatedSearchDifferingOnlyInCaseAndSpacesIsAnsweredFromTheCache(t *testing.T) {
	clock := newFakeClock()
	up := newUpstream(t, clock, serveFixtures(t))
	g := newTestGeocoder(t, clock, up.URL, openDB(t))
	ctx := context.Background()

	first, err := g.Search(ctx, "Saalfelden am See")
	if err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Hour)
	again, err := g.Search(ctx, "  saalfelden \t AM   see ")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, again) {
		t.Errorf("cached answer %v differs from the first %v", again, first)
	}
	if n := len(up.Requests()); n != 1 {
		t.Fatalf("upstream saw %d requests, want 1", n)
	}
}

func TestAnEmptySearchAnswerIsCachedToo(t *testing.T) {
	clock := newFakeClock()
	up := newUpstream(t, clock, serveFixtures(t))
	g := newTestGeocoder(t, clock, up.URL, openDB(t))

	for i := 0; i < 2; i++ {
		places, err := g.Search(context.Background(), "nowhere")
		if err != nil {
			t.Fatal(err)
		}
		if len(places) != 0 {
			t.Fatalf("places = %v, want none", places)
		}
	}
	if n := len(up.Requests()); n != 1 {
		t.Fatalf("upstream saw %d requests, want 1", n)
	}
}

func TestACachedAnswerOlderThanNinetyDaysIsFetchedAgain(t *testing.T) {
	clock := newFakeClock()
	up := newUpstream(t, clock, serveFixtures(t))
	db := openDB(t)
	g := newTestGeocoder(t, clock, up.URL, db)
	ctx := context.Background()
	search := func() {
		t.Helper()
		if _, err := g.Search(ctx, "Saalfelden"); err != nil {
			t.Fatal(err)
		}
	}

	search()
	clock.Advance(CacheTTL - time.Second)
	search()
	if n := len(up.Requests()); n != 1 {
		t.Fatalf("within ninety days upstream saw %d requests, want 1", n)
	}
	clock.Advance(time.Second)
	search()
	if n := len(up.Requests()); n != 2 {
		t.Fatalf("after ninety days upstream saw %d requests, want 2", n)
	}
	rows := cacheRows(t, db)
	if len(rows) != 1 || rows["search:de:saalfelden"] != clock.Now().Unix() {
		t.Errorf("cache rows = %v, want the one entry refreshed at %d", rows, clock.Now().Unix())
	}
	search()
	if n := len(up.Requests()); n != 2 {
		t.Fatalf("the refreshed entry was not served from the cache: %d requests", n)
	}
}

func TestReverseGeocodingSharesTheLimiterAndTheCacheWithSearch(t *testing.T) {
	clock := newFakeClock()
	up := newUpstream(t, clock, serveFixtures(t))
	db := openDB(t)
	g := newTestGeocoder(t, clock, up.URL, db)
	ctx := context.Background()

	if _, err := g.Search(ctx, "Saalfelden"); err != nil {
		t.Fatal(err)
	}
	place, err := g.Reverse(ctx, 47.4259, 12.8467)
	if err != nil {
		t.Fatal(err)
	}
	want, err := ParseReverse(fixture(t, "reverse-saalfelden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(place, want) {
		t.Errorf("place = %v, want %v", place, want)
	}
	again, err := g.Reverse(ctx, 47.425901, 12.846699)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(again, want) {
		t.Errorf("cached place = %v, want %v", again, want)
	}

	reqs := up.Requests()
	if len(reqs) != 2 {
		t.Fatalf("upstream saw %d requests, want 2", len(reqs))
	}
	assertSpacedAtLeastOneSecond(t, reqs)
	rev := reqs[1]
	wantQuery := url.Values{
		"lat": {"47.42590"}, "lon": {"12.84670"}, "format": {"jsonv2"}, "addressdetails": {"1"},
		"namedetails": {"1"}, "accept-language": {"de"},
	}
	if rev.Path != "/reverse" || !reflect.DeepEqual(rev.Query, wantQuery) || rev.UserAgent != testUserAgent {
		t.Errorf("reverse request = %+v, want /reverse with %v and our user agent", rev, wantQuery)
	}
	rows := cacheRows(t, db)
	if _, ok := rows["search:de:saalfelden"]; !ok || len(rows) != 2 {
		t.Errorf("cache keys = %v, want the search and the reverse entry", rows)
	}
	if _, ok := rows["reverse:de:47.42590,12.84670"]; !ok {
		t.Errorf("cache keys = %v, want reverse:de:47.42590,12.84670", rows)
	}
}

func TestAnUnreachableOrFailingNominatimYieldsAnUpstreamErrorAndCachesNothing(t *testing.T) {
	closed := httptest.NewServer(http.NotFoundHandler())
	closed.Close()

	cases := map[string]struct {
		respond http.HandlerFunc
		baseURL string
		timeout time.Duration
	}{
		"unreachable": {baseURL: closed.URL},
		"server error": {respond: func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "overloaded", http.StatusServiceUnavailable)
		}},
		"broken json": {respond: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`[{"lat":`))
		}},
		"timeout": {timeout: 50 * time.Millisecond, respond: func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			clock := newFakeClock()
			db := openDB(t)
			baseURL := tc.baseURL
			if tc.respond != nil {
				baseURL = newUpstream(t, clock, tc.respond).URL
			}
			g := newTestGeocoder(t, clock, baseURL, db)
			if tc.timeout > 0 {
				g.http = &http.Client{Timeout: tc.timeout}
			}

			if _, err := g.Search(context.Background(), "Saalfelden"); !errors.Is(err, ErrUpstream) {
				t.Errorf("search error = %v, want ErrUpstream", err)
			}
			if _, err := g.Reverse(context.Background(), 47.4259, 12.8467); !errors.Is(err, ErrUpstream) {
				t.Errorf("reverse error = %v, want ErrUpstream", err)
			}
			if rows := cacheRows(t, db); len(rows) != 0 {
				t.Errorf("cache rows = %v, want none", rows)
			}
		})
	}
}

func TestWithoutAContactTheGeocoderSendsNothing(t *testing.T) {
	clock := newFakeClock()
	up := newUpstream(t, clock, serveFixtures(t))
	g := New(Options{BaseURL: up.URL, Version: "1.2.3", DB: openDB(t), Now: clock.Now, Sleep: clock.Sleep})

	if g.Enabled() {
		t.Error("Enabled() = true without a contact")
	}
	if _, err := g.Search(context.Background(), "Saalfelden"); !errors.Is(err, ErrDisabled) {
		t.Errorf("search error = %v, want ErrDisabled", err)
	}
	if _, err := g.Reverse(context.Background(), 47.4259, 12.8467); !errors.Is(err, ErrDisabled) {
		t.Errorf("reverse error = %v, want ErrDisabled", err)
	}
	if n := len(up.Requests()); n != 0 {
		t.Errorf("upstream saw %d requests, want none", n)
	}
}
