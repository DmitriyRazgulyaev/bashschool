package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5"
)

const minimumVoters = 5
const consensusPercent = 80

// normalizeTranslation only removes typographic differences. Semantic aliases
// must be reviewed by a moderator; similarity alone is not evidence of meaning.
func normalizeTranslation(text string) string {
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

type consensusCandidate struct {
	ID, Text    string
	CanonicalID *string
	Correct     *bool
}
type candidateGroup struct {
	ID, Text string
	Votes    int
	Correct  *bool
}

func groupCandidates(candidates []consensusCandidate) (map[string]string, map[string]*candidateGroup) {
	roots := make(map[string]string, len(candidates))
	normalized := make(map[string]string, len(candidates))
	groups := make(map[string]*candidateGroup, len(candidates))
	// Stable order chooses the same display text on every reconciliation.
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	for _, c := range candidates {
		if c.CanonicalID != nil {
			continue
		}
		key := normalizeTranslation(c.Text)
		root, exists := normalized[key]
		if !exists {
			root = c.ID
			normalized[key] = root
			groups[root] = &candidateGroup{ID: root, Text: c.Text, Correct: c.Correct}
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

// reconcileTranslation runs while the fragment row is locked, so concurrent
// answers and moderation cannot overwrite a newer aggregate with stale votes.
func reconcileTranslation(ctx context.Context, tx pgx.Tx, fragmentID, pair string) error {
	rows, err := tx.Query(ctx, `SELECT id,translation,canonical_id,is_correct FROM candidates WHERE fragment_id=$1 AND lang_pair=$2`, fragmentID, pair)
	if err != nil {
		return err
	}
	candidates, err := pgx.CollectRows(rows, pgx.RowToStructByPos[consensusCandidate])
	if err != nil {
		return err
	}
	roots, groups := groupCandidates(candidates)
	rows, err = tx.Query(ctx, `SELECT candidate_id FROM (
  SELECT DISTINCT ON (a.user_id) a.user_id,a.candidate_id
  FROM answers a JOIN tasks t ON t.id=a.task_id
  WHERE t.fragment_id=$1 AND t.lang_pair=$2
  ORDER BY a.user_id,a.created_at DESC,a.id DESC) latest`, fragmentID, pair)
	if err != nil {
		return err
	}
	votes, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	for _, id := range votes {
		if group := groups[roots[id]]; group != nil {
			group.Votes++
		}
	}
	var winner *candidateGroup
	for _, group := range groups {
		if winner == nil || group.Votes > winner.Votes || (group.Votes == winner.Votes && group.ID < winner.ID) {
			winner = group
		}
	}
	if winner == nil || len(votes) < minimumVoters {
		return nil
	}
	// Expert decisions are retained until a moderator deliberately revises them.
	var expert bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM translations WHERE fragment_id=$1 AND lang_pair=$2 AND expert_confirmed)`, fragmentID, pair).Scan(&expert); err != nil {
		return err
	}
	if expert {
		var selected string
		if err = tx.QueryRow(ctx, `SELECT translation FROM translations WHERE fragment_id=$1 AND lang_pair=$2`, fragmentID, pair).Scan(&selected); err != nil {
			return err
		}
		count := 0
		for _, group := range groups {
			if normalizeTranslation(group.Text) == normalizeTranslation(selected) {
				count += group.Votes
			}
		}
		_, err = tx.Exec(ctx, `UPDATE translations SET confirmed=$3,updated_at=now() WHERE fragment_id=$1 AND lang_pair=$2`, fragmentID, pair, count)
		return err
	}
	status := "expert_review"
	if winner.Votes*100 >= len(votes)*consensusPercent && (winner.Correct == nil || *winner.Correct) {
		status = "confirmed"
	}
	_, err = tx.Exec(ctx, `INSERT INTO translations(fragment_id,lang_pair,translation,confirmed,status)
 VALUES($1,$2,$3,$4,$5) ON CONFLICT(fragment_id,lang_pair) DO UPDATE SET
 translation=EXCLUDED.translation,confirmed=EXCLUDED.confirmed,status=EXCLUDED.status,updated_at=now()`, fragmentID, pair, winner.Text, winner.Votes, status)
	if err != nil {
		return fmt.Errorf("save consensus: %w", err)
	}
	if status == "expert_review" {
		_, err = tx.Exec(ctx, `INSERT INTO moderation_cases(fragment_id,lang_pair) VALUES($1,$2)
  ON CONFLICT(fragment_id,lang_pair) DO UPDATE SET status='open',resolved_at=NULL`, fragmentID, pair)
	} else {
		_, err = tx.Exec(ctx, `UPDATE moderation_cases SET status='consensus',resolved_at=now() WHERE fragment_id=$1 AND lang_pair=$2 AND status='open'`, fragmentID, pair)
	}
	return err
}
