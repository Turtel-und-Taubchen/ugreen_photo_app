package api_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"photocurator/internal/api"
	"photocurator/internal/store"
)

func selectionIndex(t *testing.T) *sql.DB {
	t.Helper()
	db, err := store.OpenIndexDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	for i := 1; i <= 5; i++ {
		at := fmt.Sprintf("2024-07-%02dT10:00:00", i)
		if i == 5 {
			at = "2024-08-01T10:00:00"
		}
		if _, err := db.Exec(`INSERT INTO photos (id, rel_path, folder, media_type, format, size, mtime_ns, taken_at, taken_source, scanned_at)
			VALUES (?, ?, '', 'image', 'jpeg', 1, 1, ?, 'exif', '2024-01-01T00:00:00Z')`, i, fmt.Sprintf("p%d.jpg", i), at); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func TestCountingAMonthSelectionWithExclusionsAnswersTheResolvedCount(t *testing.T) {
	s := newTestServerDeps(t, api.LibraryDeps{Index: selectionIndex(t)})
	sess := s.signedIn(t, "anna")
	body := `{"filter":{"groups":["2024-07"]},"exclude_ids":[2,5]}`

	wantJSONStatus(t, "no session", s.do(http.MethodPost, "/api/selection/count", body, withXRW), http.StatusUnauthorized)

	rec := s.do(http.MethodPost, "/api/selection/count", body, sess, withXRW)
	wantJSONStatus(t, "count", rec, http.StatusOK)
	var got map[string]int
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["count"] != 3 {
		t.Errorf("count = %v, want 3 (body %s)", got, rec.Body.String())
	}
}

func TestAnInvalidSelectionAnswers400(t *testing.T) {
	s := newTestServerDeps(t, api.LibraryDeps{Index: selectionIndex(t)})
	sess := s.signedIn(t, "anna")
	for name, body := range map[string]string{
		"bad month key": `{"filter":{"groups":["2024-13"]}}`,
		"not json":      `{"filter":`,
		"too large":     `{"pad":"` + strings.Repeat("x", 1<<20+1) + `"}`,
	} {
		rec := s.do(http.MethodPost, "/api/selection/count", body, sess, withXRW)
		wantJSONStatus(t, name, rec, http.StatusBadRequest)
		var e map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil || e["error"] == "" {
			t.Errorf("%s: body %q lacks an error message", name, rec.Body.String())
		}
	}
}
