package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUsersConfLineWithAPlaintextPasswordStopsTheStart(t *testing.T) {
	const h = "$argon2id$v=19$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0c2E$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaGhhc2g"
	cases := []struct {
		name, content, wantErr string
	}{
		{"plaintext", "# c\nanna:" + h + "\nben:secret\n", "line 3"},
		{"no users", "# only a comment\n\n", "no users"},
		{"duplicate", "anna:" + h + "\nanna:" + h + "\n", "line 2"},
		{"empty name", ":" + h + "\n", "line 1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "users.conf")
			if err := os.WriteFile(p, []byte(c.content), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := LoadUsers(p)
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, c.wantErr)
			}
		})
	}
	t.Run("missing file", func(t *testing.T) {
		if _, err := LoadUsers(filepath.Join(t.TempDir(), "nope")); err == nil {
			t.Fatal("missing file accepted")
		}
	})
	t.Run("valid file keeps names case-sensitive", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "users.conf")
		os.WriteFile(p, []byte("anna:"+h+"\nAnna:"+h+"\n"), 0o600)
		u, err := LoadUsers(p)
		if err != nil || len(u) != 2 {
			t.Fatalf("users=%v err=%v", u, err)
		}
	})
}
