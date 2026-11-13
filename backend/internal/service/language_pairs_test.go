package service

import "testing"

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
