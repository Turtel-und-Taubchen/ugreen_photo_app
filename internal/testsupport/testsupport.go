// Package testsupport holds shared helpers for tests that work on fixture files.
package testsupport

import (
	"os"
	"path/filepath"
	"testing"
)

// RepoRoot returns the module root: the nearest ancestor of the working directory that
// contains go.mod. Go runs each package's tests in the package directory, so this works
// from any package.
func RepoRoot(t testing.TB) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod found above the working directory")
		}
		dir = parent
	}
}

// CopyFixture copies testdata/<rel> (a file or a directory tree) into a fresh t.TempDir()
// and returns the path of the copy. The copy keeps rel as its relative path, so a fixture
// under library/JJJJ/MM stays inside a JJJJ/MM structure. Tests write only to the copy;
// the committed fixtures are never modified.
func CopyFixture(t testing.TB, rel string) string {
	t.Helper()
	src := filepath.Join(RepoRoot(t), "testdata", filepath.FromSlash(rel))
	info, err := os.Stat(src)
	if err != nil {
		t.Fatalf("fixture %s: %v", rel, err)
	}
	dst := filepath.Join(t.TempDir(), filepath.FromSlash(rel))
	if info.IsDir() {
		if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
			t.Fatalf("copy fixture dir %s: %v", rel, err)
		}
		return dst
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("create fixture dir for %s: %v", rel, err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read fixture %s: %v", rel, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write fixture copy %s: %v", rel, err)
	}
	return dst
}
