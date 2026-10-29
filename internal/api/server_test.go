package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"photocurator/internal/version"
)

const indexHTML = "<html>photocurator-index</html>"

// get requests target as a signed-in user; the unauthenticated view is covered in auth_test.go.
func get(t *testing.T, target string) *httptest.ResponseRecorder {
	t.Helper()
	s := newTestServer(t)
	return s.do(http.MethodGet, target, "", s.signedIn(t, "anna"))
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
