package writer

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"photocurator/internal/store"
)

func newBackups(t *testing.T, e *env, now *time.Time) *Backups {
	t.Helper()
	db, err := store.OpenAppDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return &Backups{DB: db, Root: e.root, Dir: e.p.BackupDir, Retention: 30, Now: func() time.Time { return *now }}
}

func TestFirstWriteLeavesDatedBackupCopy(t *testing.T) {
	e := newEnv(t, true)
	now := time.Date(2024, 9, 10, 15, 30, 0, 0, time.Local)
	b := newBackups(t, e, &now)

	path, created, err := b.Ensure(sampleRel)
	if err != nil || !created {
		t.Fatalf("Ensure = %q, %v, %v", path, created, err)
	}
	want := filepath.Join(e.p.BackupDir, "2024-09-10", filepath.FromSlash(sampleRel))
	if path != want {
		t.Fatalf("path = %s, want %s", path, want)
	}
	if !bytes.Equal(readFile(t, path), readFile(t, e.abs(sampleRel))) {
		t.Error("backup differs from the original")
	}
	orig, _ := os.Stat(e.abs(sampleRel))
	cp, _ := os.Stat(path)
	if !cp.ModTime().Equal(orig.ModTime()) || cp.Mode().Perm() != orig.Mode().Perm() {
		t.Errorf("copy has mtime %v mode %v, original %v %v", cp.ModTime(), cp.Mode(), orig.ModTime(), orig.Mode())
	}
	if _, _, err := b.Ensure("../escape.jpg"); err == nil {
		t.Error("a path outside the library was accepted")
	}
}

func TestSecondBackupWithinRetentionIsSkipped(t *testing.T) {
	e := newEnv(t, true)
	now := time.Date(2024, 9, 10, 15, 30, 0, 0, time.Local)
	b := newBackups(t, e, &now)
	first, _, err := b.Ensure(sampleRel)
	if err != nil {
		t.Fatal(err)
	}

	now = now.AddDate(0, 0, 10)
	again, created, err := b.Ensure(sampleRel)
	if err != nil || created || again != first {
		t.Fatalf("second Ensure = %q, %v, %v; want %q kept", again, created, err, first)
	}
	if _, err := os.Stat(filepath.Join(e.p.BackupDir, "2024-09-20")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a second dated folder exists: %v", err)
	}

	if err := os.Remove(first); err != nil {
		t.Fatal(err)
	}
	again, created, err = b.Ensure(sampleRel)
	if err != nil || !created {
		t.Fatalf("Ensure after deleting the copy = %q, %v, %v", again, created, err)
	}
	if !bytes.Equal(readFile(t, again), readFile(t, e.abs(sampleRel))) {
		t.Error("recreated copy differs from the original")
	}
}

func TestPruneRemovesOnlyBackupsOlderThanRetention(t *testing.T) {
	e := newEnv(t, true)
	now := time.Date(2024, 9, 10, 15, 30, 0, 0, time.Local)
	b := newBackups(t, e, &now)
	dir := e.p.BackupDir
	put := func(rel string) {
		t.Helper()
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Retention 30 days at 2024-09-10: the cutoff day is 2024-08-11 and still counts as young.
	for _, rel := range []string{
		"2024-08-10/a.jpg", ".inflight/2024-08-10/a.jpg", // old
		"2024-08-11/b.jpg", ".inflight/2024-08-11/b.jpg", // youngest that stays
		"keep-me/c.jpg", "notes.txt", "2024-08-01.txt", // foreign
	} {
		put(rel)
	}
	for rel, at := range map[string]time.Time{
		"old.jpg":   time.Date(2024, 8, 10, 23, 0, 0, 0, time.Local),
		"young.jpg": time.Date(2024, 8, 11, 0, 0, 0, 0, time.Local),
	} {
		if _, err := b.DB.Exec(`INSERT INTO backups VALUES (?, ?, ?)`, rel, "x/"+rel, at.Unix()); err != nil {
			t.Fatal(err)
		}
	}

	if err := b.Prune(now); err != nil {
		t.Fatal(err)
	}
	for rel, want := range map[string]bool{
		"2024-08-10": false, ".inflight/2024-08-10": false,
		"2024-08-11": true, ".inflight/2024-08-11": true,
		"keep-me": true, "notes.txt": true, "2024-08-01.txt": true, ".inflight": true,
	} {
		_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if (err == nil) != want {
			t.Errorf("%s exists = %v, want %v", rel, err == nil, want)
		}
	}
	var rows int
	var left string
	if err := b.DB.QueryRow(`SELECT count(*), min(rel_path) FROM backups`).Scan(&rows, &left); err != nil || rows != 1 || left != "young.jpg" {
		t.Errorf("rows left = %d (%s), err %v; want only young.jpg", rows, left, err)
	}
}

func TestExclusiveWaitsForARunningJob(t *testing.T) {
	e := newEnv(t, true)
	q := newQueue(t, e)
	q.exclusive.Lock()
	ran := make(chan struct{})
	go func() {
		q.Exclusive(context.Background(), func() error { close(ran); return nil })
	}()
	select {
	case <-ran:
		t.Fatal("ran while the queue was busy")
	case <-time.After(50 * time.Millisecond):
	}
	q.exclusive.Unlock()
	select {
	case <-ran:
	case <-time.After(5 * time.Second):
		t.Fatal("never ran after the queue became idle")
	}
}
