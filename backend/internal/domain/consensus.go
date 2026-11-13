package domain

import (
	"sort"
	"strings"
	"unicode"
)

const MinimumVoters = 5
const ConsensusPercent = 80

// NormalizeTranslation only removes typographic differences. Semantic aliases
// must be reviewed by a moderator; similarity alone is not evidence of meaning.
func NormalizeTranslation(text string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(text) {
		if r == 'ё' {
			r = 'е'
		}
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			out.WriteRune(r)
		} else {
			out.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(out.String()), " ")
}

type ConsensusCandidate struct {
	ID, Text    string
	CanonicalID *string
	Correct     *bool
}
type CandidateGroup struct {
	ID, Text string
	Votes    int
	Correct  *bool
}

func GroupCandidates(candidates []ConsensusCandidate) (map[string]string, map[string]*CandidateGroup) {
	roots := make(map[string]string, len(candidates))
	normalized := make(map[string]string, len(candidates))
	groups := make(map[string]*CandidateGroup, len(candidates))
	// Stable order chooses the same display text on every reconciliation.
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	for _, c := range candidates {
		if c.CanonicalID != nil {
			continue
		}
		key := NormalizeTranslation(c.Text)
		root, exists := normalized[key]
		if !exists {
			root = c.ID
			normalized[key] = root
			groups[root] = &CandidateGroup{ID: root, Text: c.Text, Correct: c.Correct}
		}
		roots[c.ID] = root
		if c.Correct != nil {
			groups[root].Correct = c.Correct
		}
	}
	for _, c := range candidates {
		if c.CanonicalID != nil {
			if root, ok := roots[*c.CanonicalID]; ok {
				roots[c.ID] = root
			}
		}
	}
	return roots, groups
}
