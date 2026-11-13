package domain

import "testing"

func TestNormalizeTranslation(t *testing.T) {
	for _, tc := range []struct{ input, want string }{{"  Привет,  мир! ", "привет мир"}, {"Ещё\nкитап!", "еще китап"}, {"Hello — WORLD", "hello world"}} {
		if got := NormalizeTranslation(tc.input); got != tc.want {
			t.Errorf("normalize(%q)=%q, want %q", tc.input, got, tc.want)
		}
	}
	if NormalizeTranslation("Я иду в школу") == NormalizeTranslation("Я иду домой") {
		t.Fatal("different meanings must remain separate")
	}
}

func TestCandidateGroupsRequireReviewedSynonyms(t *testing.T) {
	correct := true
	root := "a"
	candidates := []ConsensusCandidate{{ID: "a", Text: "Здравствуйте!", Correct: &correct}, {ID: "b", Text: "здравствуйте."}, {ID: "c", Text: "Приветствую"}}
	roots, groups := GroupCandidates(candidates)
	if len(groups) != 2 || roots["a"] != roots["b"] || roots["a"] == roots["c"] {
		t.Fatal("typography and synonyms should be handled separately")
	}
	candidates[2].CanonicalID = &root
	roots, groups = GroupCandidates(candidates)
	if len(groups) != 1 || roots["a"] != roots["c"] {
		t.Fatal("reviewed synonyms must share votes")
	}
}
