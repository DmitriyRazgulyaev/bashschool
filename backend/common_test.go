package main

import (
	"strings"
	"testing"
)

func TestTokenRoundTrip(t *testing.T) {
	jwtSecret = []byte("test_only_secret_long_enough_for_hs256_signing")
	token, err := tokenFor("user-123", "student")
	if err != nil {
		t.Fatal(err)
	}
	id, role, err := parseToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if id != "user-123" || role != "student" {
		t.Fatalf("unexpected claims %q, %q", id, role)
	}
}
func TestTamperedJWT(t *testing.T) {
	jwtSecret = []byte("test_only_secret_long_enough_for_hs256_signing")
	token, _ := tokenFor("user-123", "student")
	if _, _, err := parseToken(token + "x"); err == nil {
		t.Fatal("tampered token accepted")
	}
}
func TestLanguageCodeValidation(t *testing.T) {
	for _, s := range []string{"ba", "ru", "en", "kk", "eng"} {
		if !langPattern.MatchString(s) {
			t.Fatalf("expected valid %s", s)
		}
	}
	for _, s := range []string{"", "ru-RU", "a", "russian", "12"} {
		if langPattern.MatchString(s) {
			t.Fatalf("expected invalid %s", s)
		}
	}
}
func TestRandomUUIDShape(t *testing.T) {
	s := randomUUID()
	if len(s) != 36 || strings.Count(s, "-") != 4 {
		t.Fatalf("bad generated UUID %q", s)
	}
}
