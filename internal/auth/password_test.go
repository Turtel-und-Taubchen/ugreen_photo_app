package auth

import (
	"strings"
	"testing"
)

func TestHashVerifiesTheRightPasswordAndRejectsAWrongOne(t *testing.T) {
	hash, err := HashPassword("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("unexpected PHC format: %s", hash)
	}
	if ok, err := VerifyPassword(hash, "correct horse"); err != nil || !ok {
		t.Fatalf("right password: ok=%v err=%v", ok, err)
	}
	if ok, err := VerifyPassword(hash, "wrong"); err != nil || ok {
		t.Fatalf("wrong password: ok=%v err=%v", ok, err)
	}
	other, _ := HashPassword("correct horse")
	if other == hash {
		t.Fatal("equal passwords produced identical hashes (salt reused)")
	}
	if _, err := VerifyPassword("plaintext", "x"); err == nil {
		t.Fatal("malformed hash accepted")
	}
}
