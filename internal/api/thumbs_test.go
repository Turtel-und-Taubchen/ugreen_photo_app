package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"photocurator/internal/api"
	"photocurator/internal/library"
	"photocurator/internal/thumbs"
)

// writingRenderer creates dst like the real renderer does and counts renders.
type writingRenderer struct {
	mu    sync.Mutex
	calls int
	fail  bool
	hold  chan struct{} // when set, renders block until it is closed
}

func (r *writingRenderer) RenderTo(_ context.Context, _, _, _, dst string) error {
	r.mu.Lock()
	r.calls++
	fail, hold := r.fail, r.hold
	r.mu.Unlock()
	if hold != nil {
		<-hold
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if fail {
		return os.WriteFile(strings.TrimSuffix(dst, ".webp")+".failed", nil, 0o644)
	}
	return os.WriteFile(dst, []byte("RIFFfakeWEBP"), 0o644)
}

func (r *writingRenderer) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

func (r *writingRenderer) failRenders() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fail = true
}

type fakeIndex struct {
	mu      sync.Mutex
	entries map[int64]thumbs.Entry
}

func (f *fakeIndex) lookup(id int64) (thumbs.Entry, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	e, ok := f.entries[id]
	return e, ok, nil
}

func (f *fakeIndex) set(e thumbs.Entry) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries[e.ID] = e
}

func keyOf(e thumbs.Entry) string { return thumbs.Key(e.RelPath, e.Size, e.MtimeNs) }

type thumbFixture struct {
	srv   *testServer
	rend  *writingRenderer
	index *fakeIndex
	sess  reqOpt
}

func newThumbFixture(t *testing.T) *thumbFixture {
	t.Helper()
	rend := &writingRenderer{}
	cache := t.TempDir()
	svc := thumbs.NewService(cache, t.TempDir(), rend, 1)
	svc.Start(context.Background())
	t.Cleanup(svc.Stop)
	idx := &fakeIndex{entries: map[int64]thumbs.Entry{
		1: {ID: 1, RelPath: "2024/07/a.jpg", MediaType: "image", Size: 10, MtimeNs: 100},
		2: {ID: 2, RelPath: "2024/07/b.jpg", MediaType: "image", Size: 20, MtimeNs: 200},
	}}
	s := newTestServerDeps(t, api.LibraryDeps{Thumbs: &api.ThumbDeps{Lookup: idx.lookup, Service: svc, CacheDir: cache}})
	return &thumbFixture{srv: s, rend: rend, index: idx, sess: s.signedIn(t, "anna")}
}

func (f *thumbFixture) get(target string) *httptest.ResponseRecorder {
	return f.srv.do(http.MethodGet, target, "", f.sess)
}

// ready polls until the preview is no longer pending, as the browser does after a 202.
func (f *thumbFixture) ready(t *testing.T, target string) *httptest.ResponseRecorder {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if rec := f.get(target); rec.Code != http.StatusAccepted {
			return rec
		}
	}
	t.Fatalf("%s stayed pending", target)
	return nil
}

func TestThumbnailEndpointAnswersPendingThenReadyAsWebp(t *testing.T) {
	f := newThumbFixture(t)
	wantJSONStatus(t, "no session", f.srv.do(http.MethodGet, "/api/thumbs/1/s", ""), http.StatusUnauthorized)

	// The renderer is held, so the first request cannot be served yet.
	hold := make(chan struct{})
	f.rend.hold = hold
	var release sync.Once
	releaseRender := func() { release.Do(func() { close(hold) }) }
	t.Cleanup(releaseRender)
	first := f.get("/api/thumbs/1/s")
	wantJSONStatus(t, "pending", first, http.StatusAccepted)
	if first.Header().Get("Retry-After") != "1" || first.Header().Get("Cache-Control") != "no-store" ||
		strings.TrimSpace(first.Body.String()) != `{"state":"pending"}` {
		t.Errorf("pending: %v %q", first.Header(), first.Body.String())
	}
	releaseRender()
	rec := f.ready(t, "/api/thumbs/1/s")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/webp" ||
		rec.Header().Get("Cache-Control") != "no-store" || rec.Body.String() != "RIFFfakeWEBP" {
		t.Errorf("ready: %d %v %q", rec.Code, rec.Header(), rec.Body.String())
	}
}

func TestThumbnailEndpointRejectsUnknownSizeAndUnknownId(t *testing.T) {
	f := newThumbFixture(t)
	for _, target := range []string{"/api/thumbs/1/m", "/api/thumbs/1/x", "/api/thumbs/99/s", "/api/thumbs/abc/s"} {
		wantJSONStatus(t, target, f.get(target), http.StatusNotFound)
	}
	if f.rend.count() != 0 {
		t.Errorf("rendered %d times for rejected requests", f.rend.count())
	}
}

func TestThumbnailEndpointReportsFailedRenderAs422(t *testing.T) {
	f := newThumbFixture(t)
	f.rend.failRenders()
	rec := f.ready(t, "/api/thumbs/1/l")
	wantJSONStatus(t, "failed", rec, http.StatusUnprocessableEntity)
	if rec.Header().Get("Cache-Control") != "no-store" || strings.TrimSpace(rec.Body.String()) != `{"state":"failed"}` {
		t.Errorf("failed: %v %q", rec.Header(), rec.Body.String())
	}
}

func TestASecondRequestServesTheCachedThumbnailWithoutRendering(t *testing.T) {
	f := newThumbFixture(t)
	f.ready(t, "/api/thumbs/1/s")
	if rec := f.get("/api/thumbs/1/s"); rec.Code != http.StatusOK {
		t.Fatalf("second request: %d", rec.Code)
	}
	if f.rend.count() != 1 {
		t.Errorf("renders = %d, want 1", f.rend.count())
	}
}

func TestAChangedSizeOrMtimeRendersANewThumbnailUnderANewKey(t *testing.T) {
	f := newThumbFixture(t)
	old, _, _ := f.index.lookup(1)
	f.ready(t, "/api/thumbs/1/s?k="+keyOf(old))

	for name, changed := range map[string]thumbs.Entry{
		"size":  {ID: 1, RelPath: old.RelPath, MediaType: "image", Size: old.Size + 1, MtimeNs: old.MtimeNs},
		"mtime": {ID: 1, RelPath: old.RelPath, MediaType: "image", Size: old.Size, MtimeNs: old.MtimeNs + 1},
	} {
		if keyOf(changed) == keyOf(old) {
			t.Fatalf("%s: key unchanged", name)
		}
		before := f.rend.count()
		f.index.set(changed)
		rec := f.ready(t, "/api/thumbs/1/s?k="+keyOf(changed))
		if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != "private, max-age=31536000, immutable" {
			t.Errorf("%s: %d %v", name, rec.Code, rec.Header())
		}
		if f.rend.count() != before+1 {
			t.Errorf("%s: renders %d -> %d, want one more", name, before, f.rend.count())
		}
		// The old key no longer matches the file behind the id.
		if got := f.get("/api/thumbs/1/s?k=" + keyOf(old)).Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("%s: stale key Cache-Control = %q", name, got)
		}
		f.index.set(old)
	}
}

func TestOnlyARequestWithTheCurrentKeyIsCacheableForever(t *testing.T) {
	f := newThumbFixture(t)
	e, _, _ := f.index.lookup(1)
	key := keyOf(e)
	f.ready(t, "/api/thumbs/1/s")

	for target, want := range map[string]string{
		"/api/thumbs/1/s?k=" + key:       "private, max-age=31536000, immutable",
		"/api/thumbs/1/s":                "no-store",
		"/api/thumbs/1/s?k=":             "no-store",
		"/api/thumbs/1/s?k=" + key + "0": "no-store",
		"/api/thumbs/1/s?k=other":        "no-store",
	} {
		rec := f.get(target)
		if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != want {
			t.Errorf("%s: %d Cache-Control %q, want %q", target, rec.Code, rec.Header().Get("Cache-Control"), want)
		}
	}
}

type fakeLister struct{ rows []library.Photo }

func (fakeLister) Groups(context.Context) (library.Groups, error) { return library.Groups{}, nil }

func (fakeLister) Detail(context.Context, int64) (library.Detail, bool, error) {
	return library.Detail{}, false, nil
}

func (f fakeLister) List(_ context.Context, _ string, after *library.ListCursor, limit int) ([]library.Photo, error) {
	var out []library.Photo
	for _, p := range f.rows {
		if after != nil {
			at := ""
			if p.TakenAt != nil {
				at = *p.TakenAt
			}
			if !(at < after.TakenAt || (at == after.TakenAt && p.ID < after.ID)) {
				continue
			}
		}
		if len(out) < limit {
			out = append(out, p)
		}
	}
	return out, nil
}

func TestPhotoListCarriesMediaTypeAndThumbKeyAndPagesByCursor(t *testing.T) {
	d1, d2, d3 := "2024-07-03T10:00:00", "2024-07-02T10:00:00", "2024-07-01T10:00:00"
	rows := []library.Photo{
		{ID: 3, RelPath: "c.mp4", MediaType: "video", Size: 3, MtimeNs: 30, TakenAt: &d1},
		{ID: 2, RelPath: "b.jpg", MediaType: "image", Size: 2, MtimeNs: 20, TakenAt: &d2},
		{ID: 1, RelPath: "a.jpg", MediaType: "image", Size: 1, MtimeNs: 10, TakenAt: &d3},
	}
	s := newTestServerDeps(t, api.LibraryDeps{Photos: fakeLister{rows}})
	sess := s.signedIn(t, "anna")
	type page struct {
		Items []map[string]any `json:"items"`
		Next  *string          `json:"next"`
	}
	fetch := func(target string) page {
		rec := s.do(http.MethodGet, target, "", sess)
		wantJSONStatus(t, target, rec, http.StatusOK)
		var p page
		if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
			t.Fatal(err)
		}
		return p
	}

	p1 := fetch("/api/photos?limit=2")
	if len(p1.Items) != 2 || p1.Next == nil {
		t.Fatalf("first page: %+v", p1)
	}
	it := p1.Items[0]
	if it["id"] != 3.0 || it["relPath"] != "c.mp4" || it["mediaType"] != "video" ||
		it["thumbKey"] != thumbs.Key("c.mp4", 3, 30) || it["takenAt"] != d1 {
		t.Errorf("item: %v", it)
	}
	p2 := fetch("/api/photos?limit=2&after=" + *p1.Next)
	if len(p2.Items) != 1 || p2.Items[0]["id"] != 1.0 || p2.Next != nil {
		t.Errorf("second page: %+v", p2)
	}
	wantJSONStatus(t, "bad cursor", s.do(http.MethodGet, "/api/photos?after=!!", "", sess), http.StatusBadRequest)
}
