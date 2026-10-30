package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"photocurator/internal/library"
)

type fakeScan struct {
	triggered int
	bad       []library.UnreadableFile
}

func (f *fakeScan) Status() library.Status {
	return library.Status{State: "running", Trigger: "manual", Done: 3, Total: 10,
		StartedAt: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)}
}
func (f *fakeScan) Trigger() bool       { f.triggered++; return true }
func (f *fakeScan) Count() (int, error) { return 7, nil }
func (f *fakeScan) Unreadable(int) ([]library.UnreadableFile, int, error) {
	return f.bad, 120, nil
}

func TestPostScanStartsAScanAndGetScanReportsProgressAndUnreadableFiles(t *testing.T) {
	fake := &fakeScan{bad: []library.UnreadableFile{{RelPath: "2024/07/x.jpg", Error: "broken"}}}
	s := newTestServerWith(t, fake)
	session := s.signedIn(t, "anna")

	wantJSONStatus(t, "get without session", s.do(http.MethodGet, "/api/scan", ""), http.StatusUnauthorized)
	wantJSONStatus(t, "post without session", s.do(http.MethodPost, "/api/scan", "", withXRW), http.StatusUnauthorized)
	wantJSONStatus(t, "post without header", s.do(http.MethodPost, "/api/scan", "", session), http.StatusForbidden)
	if fake.triggered != 0 {
		t.Fatalf("refused requests triggered %d scans", fake.triggered)
	}

	post := s.do(http.MethodPost, "/api/scan", "", session, withXRW)
	wantJSONStatus(t, "post", post, http.StatusAccepted)
	if fake.triggered != 1 {
		t.Errorf("triggered = %d, want 1", fake.triggered)
	}
	get := s.do(http.MethodGet, "/api/scan", "", session)
	wantJSONStatus(t, "get", get, http.StatusOK)
	if fake.triggered != 1 {
		t.Errorf("GET triggered a scan")
	}
	for name, body := range map[string][]byte{"post": post.Body.Bytes(), "get": get.Body.Bytes()} {
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got["state"] != "running" || got["trigger"] != "manual" || got["done"] != 3.0 || got["total"] != 10.0 ||
			got["files"] != 7.0 || got["unreadable_count"] != 120.0 || got["last_error"] != "" {
			t.Errorf("%s: %v", name, got)
		}
		if got["started_at"] != "2026-03-01T12:00:00Z" || got["finished_at"] != nil {
			t.Errorf("%s: times %v / %v", name, got["started_at"], got["finished_at"])
		}
		list, _ := got["unreadable"].([]any)
		if len(list) != 1 {
			t.Fatalf("%s: unreadable %v", name, got["unreadable"])
		}
		if e := list[0].(map[string]any); e["rel_path"] != "2024/07/x.jpg" || e["error"] != "broken" {
			t.Errorf("%s: unreadable %v", name, e)
		}
	}

	fake.bad = nil
	if body := s.do(http.MethodGet, "/api/scan", "", session).Body.String(); !strings.Contains(body, `"unreadable":[]`) {
		t.Errorf("empty list not []: %s", body)
	}
}
