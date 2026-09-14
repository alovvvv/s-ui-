package util

import "testing"

func TestCheckPasswordHashed(t *testing.T) {
	hash, err := HashPassword("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword("correct horse", hash) {
		t.Error("the right password was rejected")
	}
	if CheckPassword("wrong horse", hash) {
		t.Error("the wrong password was accepted")
	}
}

// Rows written before hashing was introduced hold the password verbatim, and
// still have to authenticate -- the comparison just has to be constant time.
func TestCheckPasswordLegacyPlaintext(t *testing.T) {
	if !CheckPassword("admin", "admin") {
		t.Error("a legacy plaintext row stopped authenticating")
	}
	if CheckPassword("admi", "admin") || CheckPassword("adminx", "admin") {
		t.Error("a prefix or extension of the password was accepted")
	}
	if CheckPassword("", "admin") {
		t.Error("an empty password was accepted")
	}
}

// The dummy hash has to be a real bcrypt hash, or the no-such-user path returns
// early and the timing difference it exists to hide comes straight back.
func TestBurnPasswordCheckUsesARealHash(t *testing.T) {
	if !IsHashedPassword(dummyHash) {
		t.Fatalf("dummyHash is not a bcrypt hash: %q", dummyHash)
	}
	if CheckPassword("anything", dummyHash) {
		t.Error("dummyHash matched a password")
	}
	BurnPasswordCheck("anything")
}
