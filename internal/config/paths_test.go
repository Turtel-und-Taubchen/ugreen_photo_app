package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func libs(t *testing.T) Paths {
	t.Helper()
	root := t.TempDir()
	p := Paths{PhotosDir: filepath.Join(root, "photos"), DataDir: filepath.Join(root, "data"), BackupDir: filepath.Join(root, "backup")}
	for _, d := range []string{p.PhotosDir, p.DataDir, p.BackupDir} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

func TestValidatePathsAcceptsWritableLibrary(t *testing.T) {
	p := libs(t)
	p.PhotosHostPath = `C:\NAS\Meine Fotos\Sortiert`
	if err := ValidatePaths(p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidatePathsRejectsMissingPhotosDir(t *testing.T) {
	p := libs(t)
	p.PhotosDir = filepath.Join(t.TempDir(), "nope")
	err := ValidatePaths(p)
	if err == nil || !strings.Contains(err.Error(), p.PhotosDir) {
		t.Fatalf("want error naming %s, got %v", p.PhotosDir, err)
	}
}

func TestValidatePathsRejectsReadOnlyDirs(t *testing.T) {
	for _, which := range []string{"photos", "data", "backup"} {
		t.Run(which, func(t *testing.T) {
			p := libs(t)
			dir := map[string]string{"photos": p.PhotosDir, "data": p.DataDir, "backup": p.BackupDir}[which]
			if err := os.Chmod(dir, 0o555); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
			// Root bypasses permission checks; the suite must run unprivileged, so fail loudly.
			if os.Geteuid() == 0 {
				t.Fatal("test must run as a non-root user")
			}
			err := ValidatePaths(p)
			if err == nil || !strings.Contains(err.Error(), dir) || !strings.Contains(err.Error(), "PUID/PGID") {
				t.Fatalf("want writability error naming %s, got %v", dir, err)
			}
		})
	}
}

func TestValidatePathsRejectsInboxPaths(t *testing.T) {
	t.Run("container path", func(t *testing.T) {
		p := libs(t)
		p.PhotosDir = filepath.Join(p.PhotosDir, "iNbOx")
		if err := os.Mkdir(p.PhotosDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := ValidatePaths(p); err == nil || !strings.Contains(err.Error(), "Inbox") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("host path", func(t *testing.T) {
		for _, h := range []string{`D:\My Photos\Inbox\2024`, "/volume1/My Photos/inbox"} {
			p := libs(t)
			p.PhotosHostPath = h
			if err := ValidatePaths(p); err == nil || !strings.Contains(err.Error(), fmt.Sprintf("%q", h)) {
				t.Fatalf("host %q: got %v", h, err)
			}
		}
	})
	t.Run("subfolder", func(t *testing.T) {
		p := libs(t)
		if err := os.Mkdir(filepath.Join(p.PhotosDir, "INBOX"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := ValidatePaths(p); err == nil || !strings.Contains(err.Error(), "INBOX") {
			t.Fatalf("got %v", err)
		}
	})
}
