package security

import (
	"strings"
	"testing"
)

func TestTokenRoundTrip(t *testing.T) {
	tokens, err := New("test_only_secret_long_enough_for_hs256_signing")
	if err != nil {
		t.Fatal(err)
	}
	token, err := tokens.Issue("user-123", "student")
	if err != nil {
		t.Fatal(err)
	}
	id, role, err := tokens.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if id != "user-123" || role != "student" {
		t.Fatalf("unexpected claims %q, %q", id, role)
	}
}
func TestTamperedJWT(t *testing.T) {
	tokens, err := New("test_only_secret_long_enough_for_hs256_signing")
	if err != nil {
		t.Fatal(err)
	}
	token, _ := tokens.Issue("user-123", "student")
	if _, _, err := tokens.Parse(token + "x"); err == nil {
		t.Fatal("tampered token accepted")
	}
}
func TestRandomUUIDShape(t *testing.T) {
	s := UUID()
	if len(s) != 36 || strings.Count(s, "-") != 4 {
		t.Fatalf("bad generated UUID %q", s)
	}
}
