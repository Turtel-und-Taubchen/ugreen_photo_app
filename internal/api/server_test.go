package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"photocurator/internal/api"
	"photocurator/internal/version"
)

const indexHTML = "<html>photocurator-index</html>"

func get(t *testing.T, target string) *httptest.ResponseRecorder {
	t.Helper()
	h := api.NewHandler(fstest.MapFS{
		"index.html":    {Data: []byte(indexHTML)},
		"assets/app.js": {Data: []byte("console.log(1)")},
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func TestHealthReportsStatusAndVersion(t *testing.T) {
	rec := get(t, "/api/health")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content-type = %q", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" || body["version"] != version.Version {
		t.Fatalf("body = %v", body)
	}
}

func TestRootServesEmbeddedIndex(t *testing.T) {
	rec := get(t, "/")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "photocurator-index") {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
}

func TestUnknownClientRouteFallsBackToIndex(t *testing.T) {
	rec := get(t, "/review/42")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "photocurator-index") {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
	if a := get(t, "/assets/app.js"); !strings.Contains(a.Body.String(), "console.log") {
		t.Fatalf("existing asset not served: %q", a.Body.String())
	}
}

func TestUnknownAPIRouteReturnsJSON404(t *testing.T) {
	rec := get(t, "/api/nope")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content-type = %q", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["error"] != "not found" {
		t.Fatalf("body = %q err=%v", rec.Body.String(), err)
	}
}
