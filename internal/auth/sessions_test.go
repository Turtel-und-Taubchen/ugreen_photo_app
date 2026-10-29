package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"photocurator/internal/store"
)

func newTestSessions(t *testing.T) (*SessionStore, *fakeClock, string) {
	t.Helper()
	dir := t.TempDir()
	db, err := store.OpenAppDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	c := newFakeClock()
	return NewSessionStore(db, c.now), c, dir
}

func TestASessionStaysValidUntil30DaysWithoutUseAndIsGoneAfter(t *testing.T) {
	s, c, _ := newTestSessions(t)
	token, expires, err := s.Create("anna")
	if err != nil {
		t.Fatal(err)
	}
	if want := c.now().Add(30 * 24 * time.Hour); !expires.Equal(want) {
		t.Fatalf("expires = %v, want %v", expires, want)
	}
	c.advance(30*24*time.Hour - time.Second)
	// The lookup renews (last use was > 1 day ago), so check on a second token for the cutoff.
	if u, _, ok := s.Lookup(token); !ok || u != "anna" {
		t.Fatalf("session lost early: %q %v", u, ok)
	}
	other, _, _ := s.Create("bert")
	c.advance(30 * 24 * time.Hour)
	if _, _, ok := s.Lookup(other); ok {
		t.Fatal("expired session accepted")
	}
	var n int
	s.db.QueryRow(`SELECT count(*) FROM sessions WHERE username = 'bert'`).Scan(&n)
	if n != 0 {
		t.Fatalf("expired row kept: %d", n)
	}

	// Unrenewed boundary: a lookup renews, so use two sessions created at the same instant,
	// each looked up exactly once (no earlier lookups): valid at 30d-1s, gone at 30d.
	early, _, _ := s.Create("carla")
	late, _, _ := s.Create("dora")
	c.advance(30*24*time.Hour - time.Second)
	if u, _, ok := s.Lookup(early); !ok || u != "carla" {
		t.Fatalf("unrenewed session lost at 30d-1s: %q %v", u, ok)
	}
	c.advance(time.Second)
	if _, _, ok := s.Lookup(late); ok {
		t.Fatal("unrenewed session still valid at 30d")
	}
}

func TestUsingASessionAtMostOnceADayPushesItsExpiryForward(t *testing.T) {
	s, c, _ := newTestSessions(t)
	token, _, _ := s.Create("anna")

	c.advance(23 * time.Hour)
	if _, renewed, ok := s.Lookup(token); !ok || !renewed.IsZero() {
		t.Fatalf("renewed within a day: %v %v", renewed, ok)
	}
	c.advance(time.Hour)
	_, renewed, ok := s.Lookup(token)
	if want := c.now().Add(SessionTTL); !ok || !renewed.Equal(want) {
		t.Fatalf("renewed = %v, want %v", renewed, want)
	}
	// The pushed-forward expiry holds: still valid 29 days after the renewal.
	c.advance(29 * 24 * time.Hour)
	if _, _, ok := s.Lookup(token); !ok {
		t.Fatal("renewal did not extend the session")
	}
}

func TestADeletedSessionIsGone(t *testing.T) {
	s, _, _ := newTestSessions(t)
	token, _, _ := s.Create("anna")
	s.Delete(token)
	if _, _, ok := s.Lookup(token); ok {
		t.Fatal("deleted session accepted")
	}
}

func TestTheDatabaseStoresOnlyTheHashOfTheSessionToken(t *testing.T) {
	s, _, dir := newTestSessions(t)
	token, _, _ := s.Create("anna")

	var stored string
	if err := s.db.QueryRow(`SELECT id_hash FROM sessions`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != hashToken(token) || stored == token {
		t.Fatalf("stored id = %q", stored)
	}
	files, _ := filepath.Glob(filepath.Join(dir, "app.db*"))
	if len(files) == 0 {
		t.Fatal("no database files found")
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), token) {
			t.Fatalf("raw token found in %s", f)
		}
	}
}
