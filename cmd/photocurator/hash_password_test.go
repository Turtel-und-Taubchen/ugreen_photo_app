package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"photocurator/internal/auth"
)

func stdinWith(t *testing.T, s string) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(s)
	f.Seek(0, 0)
	return f
}

func TestHashPasswordPrintsAnArgon2idHashThatNeverContainsThePassword(t *testing.T) {
	const pw = "s3cret-pass-word"
	var out, errOut bytes.Buffer
	if code := hashPassword(stdinWith(t, pw+"\n"), &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	got := out.String()
	if !strings.HasPrefix(got, "$argon2id$") || strings.Count(got, "\n") != 1 || !strings.HasSuffix(got, "\n") {
		t.Fatalf("output must be exactly the hash plus newline, got %q", got)
	}
	if strings.Contains(got, pw) || strings.Contains(errOut.String(), pw) {
		t.Fatal("password leaked into output")
	}
	if ok, err := auth.VerifyPassword(strings.TrimSpace(got), pw); err != nil || !ok {
		t.Fatalf("printed hash does not verify: ok=%v err=%v", ok, err)
	}
}

func TestHashPasswordRejectsAnEmptyPassword(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := hashPassword(stdinWith(t, "\n"), &out, &errOut); code == 0 || out.Len() != 0 {
		t.Fatalf("exit %d, stdout %q", code, out.String())
	}
}
