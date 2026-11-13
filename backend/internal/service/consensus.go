package service

import (
	"context"
	"fmt"

	"bashkir-school/backend/internal/domain"
)

// reconcileTranslation runs while the fragment row is locked, so concurrent
// answers and moderation cannot overwrite a newer aggregate with stale votes.
func (s *Service) reconcileTranslation(ctx context.Context, tx Transaction, fragmentID, pair string) error {
	rows1, err := tx.ListConsensusCandidates(ctx, fragmentID, pair)
	if err != nil {
		return err
	}
	candidates := []domain.ConsensusCandidate{}
	for _, row := range rows1 {
		var item domain.ConsensusCandidate
		{
			item.ID, item.Text, item.CanonicalID, item.Correct = row.ID, row.Text, row.CanonicalID, row.Correct
		}

		candidates = append(candidates, item)
	}

	if err != nil {
		return err
	}
	roots, groups := domain.GroupCandidates(candidates)
	rows2, err := tx.ListLatestVotes(ctx, fragmentID, pair)
	if err != nil {
		return err
	}
	votes := []string{}
	for _, row := range rows2 {
		var id string
		{
			id = row.ID
		}

		votes = append(votes, id)
	}

	if err != nil {
		return err
	}
	for _, id := range votes {
		if group := groups[roots[id]]; group != nil {
			group.Votes++
		}
	}
	var winner *domain.CandidateGroup
	for _, group := range groups {
		if winner == nil || group.Votes > winner.Votes || (group.Votes == winner.Votes && group.ID < winner.ID) {
			winner = group
		}
	}
	if winner == nil || len(votes) < domain.MinimumVoters {
		return nil
	}
	// Expert decisions are retained until a moderator deliberately revises them.
	var expert bool
	{
		var recordIsExpertConfirmed IsExpertConfirmedRecord
		recordIsExpertConfirmed, err = tx.IsExpertConfirmed(ctx, fragmentID, pair)
		expert = recordIsExpertConfirmed.Expert

		if err != nil {
			return err
		}
	}
	if expert {
		var selected string
		{
			var recordGetExpertTranslation GetExpertTranslationRecord
			recordGetExpertTranslation, err = tx.GetExpertTranslation(ctx, fragmentID, pair)
			selected = recordGetExpertTranslation.Selected

			if err != nil {
				return err
			}
		}
		count := 0
		for _, group := range groups {
			if domain.NormalizeTranslation(group.Text) == domain.NormalizeTranslation(selected) {
				count += group.Votes
			}
		}
		_, err = tx.UpdateConfirmationCount(ctx, fragmentID, pair, count)
		return err
	}
	status := "expert_review"
	if winner.Votes*100 >= len(votes)*domain.ConsensusPercent && (winner.Correct == nil || *winner.Correct) {
		status = "confirmed"
	}
	_, err = tx.SaveConsensus(ctx, fragmentID, pair, winner.Text, winner.Votes, status)
	if err != nil {
		return fmt.Errorf("save consensus: %w", err)
	}
	if status == "expert_review" {
		_, err = tx.OpenModerationCase(ctx, fragmentID, pair)
	} else {
		_, err = tx.CloseConsensusCase(ctx, fragmentID, pair)
	}
	return err
}
