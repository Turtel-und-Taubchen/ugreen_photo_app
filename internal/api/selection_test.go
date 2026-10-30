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

func TestSelectionTagsListsEachTagOfTheSelectionWithItsPhotoCount(t *testing.T) {
	idx := selectionIndex(t)
	for _, tg := range []struct {
		id   int
		path string
	}{
		{1, "Urlaub"}, {1, "Anlass/Geburtstag"},
		{2, "Urlaub"},
		{3, "Anlass/Geburtstag"}, {3, "Urlaub"},
		{5, "Nur/Draussen"}, // outside the July selection
	} {
		if _, err := idx.Exec(`INSERT INTO photo_tags (photo_id, tag_path) VALUES (?, ?)`, tg.id, tg.path); err != nil {
			t.Fatal(err)
		}
	}
	s := newTestServerDeps(t, api.LibraryDeps{Index: idx})
	sess := s.signedIn(t, "anna")
	body := `{"filter":{"groups":["2024-07"]},"exclude_ids":[2]}`

	wantJSONStatus(t, "no session", s.do(http.MethodPost, "/api/selection/tags", body, withXRW), http.StatusUnauthorized)

	rec := s.do(http.MethodPost, "/api/selection/tags", body, sess, withXRW)
	wantJSONStatus(t, "tags", rec, http.StatusOK)
	want := `[{"path":"Anlass/Geburtstag","count":2},{"path":"Urlaub","count":2}]`
	if strings.TrimSpace(rec.Body.String()) != want {
		t.Errorf("body = %s, want %s", rec.Body.String(), want)
	}

	rec = s.do(http.MethodPost, "/api/selection/tags", `{"include_ids":[4]}`, sess, withXRW)
	wantJSONStatus(t, "untagged", rec, http.StatusOK)
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("body = %s, want []", rec.Body.String())
	}
	rec = s.do(http.MethodPost, "/api/selection/tags", `{"filter":{"groups":["2024-13"]}}`, sess, withXRW)
	wantJSONStatus(t, "invalid", rec, http.StatusBadRequest)
}
