package api_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
	_ "time/tzdata"

	"photocurator/internal/api"
	"photocurator/internal/exif"
	"photocurator/internal/library"
	"photocurator/internal/metadata"
	"photocurator/internal/store"
	"photocurator/internal/testsupport"
)

type tagNode struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	Count    int       `json:"count"`
	Children []tagNode `json:"children"`
}

func tagServer(t *testing.T, idx *sql.DB) *testServer {
	t.Helper()
	app, err := store.OpenAppDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.Close() })
	return newTestServerDeps(t, api.LibraryDeps{Tags: &api.TagDeps{App: app, Index: idx}})
}

func emptyIndex(t *testing.T) *sql.DB {
	t.Helper()
	db, err := store.OpenIndexDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func fetchTags(t *testing.T, s *testServer, sess reqOpt) []tagNode {
	t.Helper()
	rec := s.do(http.MethodGet, "/api/tags", "", sess)
	wantJSONStatus(t, "get tags", rec, http.StatusOK)
	var got struct {
		Tags []tagNode `json:"tags"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Tags == nil {
		t.Fatalf("tags must be an array, body %s", rec.Body.String())
	}
	return got.Tags
}

func findTag(nodes []tagNode, path string) *tagNode {
	for i := range nodes {
		if nodes[i].Path == path {
			return &nodes[i]
		}
		if f := findTag(nodes[i].Children, path); f != nil {
			return f
		}
	}
	return nil
}

func TestDeletingATagThatPhotosCarryIsRefused(t *testing.T) {
	idx := emptyIndex(t)
	if _, err := idx.Exec(`INSERT INTO photos (id, rel_path, folder, media_type, format, size, mtime_ns, taken_at, taken_source, scanned_at)
		VALUES (1, 'a.jpg', '', 'image', 'jpeg', 1, 1, '2024-07-01T10:00:00', 'exif', '2024-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := idx.Exec(`INSERT INTO photo_tags (photo_id, tag_path) VALUES (1, 'Anlass/Geburtstag')`); err != nil {
		t.Fatal(err)
	}
	s := tagServer(t, idx)
	sess := s.signedIn(t, "anna")
	wantJSONStatus(t, "create", s.do(http.MethodPost, "/api/tags", `{"path":"Anlass/Geburtstag"}`, sess, withXRW), http.StatusCreated)

	rec := s.do(http.MethodDelete, "/api/tags?path=Anlass", "", sess, withXRW)
	wantJSONStatus(t, "delete", rec, http.StatusConflict)
	var got struct {
		Error string `json:"error"`
		Count int    `json:"count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Count != 1 || got.Error == "" {
		t.Errorf("body = %s, want count 1 and an error text", rec.Body.String())
	}
	if findTag(fetchTags(t, s, sess), "Anlass/Geburtstag") == nil {
		t.Error("the refused delete removed the tag")
	}
}

func TestAnEmptyCategoryCanBeDeleted(t *testing.T) {
	s := tagServer(t, emptyIndex(t))
	sess := s.signedIn(t, "anna")
	wantJSONStatus(t, "no session", s.do(http.MethodGet, "/api/tags", ""), http.StatusUnauthorized)
	wantJSONStatus(t, "create", s.do(http.MethodPost, "/api/tags", `{"path":"Reisen/Italien"}`, sess, withXRW), http.StatusCreated)
	wantJSONStatus(t, "create again", s.do(http.MethodPost, "/api/tags", `{"path":"Reisen/Italien"}`, sess, withXRW), http.StatusOK)

	if rec := s.do(http.MethodDelete, "/api/tags?path=Reisen", "", sess, withXRW); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: status = %d, want 204 (body %q)", rec.Code, rec.Body.String())
	}
	if got := fetchTags(t, s, sess); len(got) != 0 {
		t.Errorf("tree after delete = %+v, want empty", got)
	}
	wantJSONStatus(t, "delete unknown", s.do(http.MethodDelete, "/api/tags?path=Reisen", "", sess, withXRW), http.StatusNotFound)
}

func TestAnInvalidNameIsRejectedWithItsReason(t *testing.T) {
	s := tagServer(t, emptyIndex(t))
	sess := s.signedIn(t, "anna")
	rec := s.do(http.MethodPost, "/api/tags", `{"path":"Anlass|Geburtstag"}`, sess, withXRW)
	wantJSONStatus(t, "post", rec, http.StatusBadRequest)
	var got struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Error, "|") {
		t.Errorf("error = %q, want it to name the offending character |", got.Error)
	}
}

func TestTheTagTreeSurvivesAnIndexRebuild(t *testing.T) {
	cfg, err := metadata.WriteConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	loc, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		t.Fatal(err)
	}
	root := testsupport.CopyFixture(t, "library")
	scanInto := func(dataDir string) *sql.DB {
		db, err := store.OpenIndexDB(dataDir)
		if err != nil {
			t.Fatal(err)
		}
		sc := &library.Scanner{Root: root, DB: db, Loc: loc,
			Reader: exif.Reader{ConfigPath: cfg, Tags: metadata.ReadTags}}
		if _, err := sc.Run(context.Background(), nil); err != nil {
			t.Fatalf("scan: %v", err)
		}
		return db
	}

	dataDir := t.TempDir()
	idx := scanInto(dataDir)
	app, err := store.OpenAppDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.Close() })

	s1 := newTestServerDeps(t, api.LibraryDeps{Tags: &api.TagDeps{App: app, Index: idx}})
	sess1 := s1.signedIn(t, "anna")
	wantJSONStatus(t, "create", s1.do(http.MethodPost, "/api/tags", `{"path":"Ausflug/Wandern"}`, sess1, withXRW), http.StatusCreated)
	before := fetchTags(t, s1, sess1)

	idx.Close()
	for _, f := range []string{"index.db", "index.db-wal", "index.db-shm"} {
		if err := os.Remove(filepath.Join(dataDir, f)); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	idx2 := scanInto(dataDir)
	t.Cleanup(func() { idx2.Close() })
	s2 := newTestServerDeps(t, api.LibraryDeps{Tags: &api.TagDeps{App: app, Index: idx2}})
	after := fetchTags(t, s2, s2.signedIn(t, "anna"))

	if !reflect.DeepEqual(before, after) {
		t.Errorf("tree changed across the rebuild:\nbefore %+v\nafter  %+v", before, after)
	}
	if n := findTag(after, "Anlass/Geburtstag"); n == nil || n.Count < 1 {
		t.Errorf("Anlass/Geburtstag = %+v, want a node with count >= 1", n)
	}
	if n := findTag(after, "Ausflug/Wandern"); n == nil || n.Count != 0 {
		t.Errorf("Ausflug/Wandern = %+v, want a node with count 0", n)
	}
}
