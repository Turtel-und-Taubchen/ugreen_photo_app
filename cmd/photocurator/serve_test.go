package main

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"photocurator/internal/config"
)

func TestServeRefusesToStartWithInvalidPaths(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()

	missing := filepath.Join(t.TempDir(), "missing")
	t.Setenv("PC_PHOTOS_DIR", missing)
	t.Setenv("PC_DATA_DIR", missing)
	t.Setenv("PC_BACKUP_DIR", missing)
	t.Setenv("PC_LISTEN", addr)

	if _, err := checkStart(config.PathsFromEnv(os.Getenv)); err == nil {
		t.Fatal("checkStart accepted invalid paths")
	}
	if code := serve(nil); code != 1 {
		t.Fatalf("serve exit code = %d, want 1", code)
	}
	// The port must still be free: serve never opened it.
	ln2, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("port was opened: %v", err)
	}
	ln2.Close()
}

func TestServeRefusesToStartWithoutAValidUsersFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PC_PHOTOS_DIR", dir)
	t.Setenv("PC_DATA_DIR", dir)
	t.Setenv("PC_BACKUP_DIR", dir)
	t.Setenv("PC_USERS_FILE", filepath.Join(dir, "users.conf"))
	if code := serve(nil); code != 1 {
		t.Fatalf("serve exit code = %d, want 1", code)
	}
}
