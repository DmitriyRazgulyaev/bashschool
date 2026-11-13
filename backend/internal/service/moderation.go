package service

import (
	"context"

	"bashkir-school/backend/internal/domain"
)

func (s *Service) Cases(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	rows1, err := s.store.ListModerationCases(ctx)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка модерации")
	}

	arr := []domain.Object{}
	for _, row := range rows1 {
		var id, fragment, text, pair, status string
		var created any
		{
			id, fragment, text, pair, status, created = row.ID, row.Fragment, row.Text,
				row.LanguagePair, row.Status, row.Created

			arr = append(arr, domain.Object{
				"id":         id,
				"fragmentId": fragment,
				"textBa":     text,
				"langPair":   pair,
				"status":     status,
				"createdAt":  created,
			})
		}

	}

	return domain.OK(arr), nil
}
func (s *Service) ResolveCase(ctx context.Context, p domain.Parameters, in ResolveCaseInput) (domain.Result, error) {
	tx, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка транзакции")
	}
	defer tx.Rollback(ctx)
	var fid, pair, tr string
	var recordGetCaseCandidate GetCaseCandidateRecord
	recordGetCaseCandidate, err = tx.GetCaseCandidate(ctx, p.ID, in.CandidateID)
	fid, pair, tr = recordGetCaseCandidate.FragmentID,
		recordGetCaseCandidate.LanguagePair, recordGetCaseCandidate.Translation

	if err != nil {
		return domain.Result{}, domain.Failure(domain.NotFound, "NOT_FOUND", "Спор или вариант не найден")
	}
	{
		var recordLockModerationFragment LockModerationFragmentRecord
		recordLockModerationFragment, err = tx.LockModerationFragment(ctx, fid)
		_ = recordLockModerationFragment.LockedFragment

		if err != nil {
			return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка блокировки")
		}
	}
	var open bool
	{
		var recordLockOpenCase LockOpenCaseRecord
		recordLockOpenCase, err = tx.LockOpenCase(ctx, p.ID)
		open = recordLockOpenCase.Open

		if err != nil || !open {
			return domain.Result{}, domain.Failure(domain.Conflict, "CASE_CHANGED", "Спор уже рассмотрен. Обновите список.")
		}
	}
	var votes int
	var recordCountCandidateVotes CountCandidateVotesRecord
	recordCountCandidateVotes, err = tx.CountCandidateVotes(ctx, fid, pair, tr)
	votes = recordCountCandidateVotes.Votes

	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка подсчёта голосов")
	}
	if _, err = tx.MarkReviewedCandidates(ctx, fid, pair, tr); err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка обновления эталона")
	}
	_, err = tx.SaveExpertTranslation(ctx, fid, pair, tr, votes)
	if err == nil {
		_, err = tx.ResolveModerationCase(ctx, tr, p.Actor.ID, p.ID)
	}
	if err != nil || tx.Commit(ctx) != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось разрешить спор")
	}
	return domain.OK(domain.Object{"status": "resolved", "translation": tr}), nil
}

func (s *Service) FragmentCandidates(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	rows1, err := s.store.ListCandidateVotes(ctx, p.ID, p.Filter("langPair", ""))
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка вариантов")
	}

	out := []domain.Object{}
	for _, row := range rows1 {
		var id, tr string
		var votes int
		{
			id, tr, votes = row.ID, row.Translation, row.Votes

			out = append(out, domain.Object{
				"id":          id,
				"translation": tr,
				"votes":       votes,
			})
		}

	}

	return domain.OK(out), nil
}

func (s *Service) AddCandidate(ctx context.Context, p domain.Parameters, in AddCandidateInput) (domain.Result, error) {
	if in.LanguagePair == "" {
		in.LanguagePair = "ba-ru"
	}
	if !s.validPair(ctx, in.LanguagePair) {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_PAIR", "Языковая пара не поддерживается")
	}
	var id string
	recordCreateSeedCandidate, err := s.store.CreateSeedCandidate(ctx, p.ID, in.LanguagePair, in.Translation)
	id = recordCreateSeedCandidate.ID

	if err != nil {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_FRAGMENT", "Не удалось добавить перевод")
	}
	return domain.New(domain.Object{"id": id}), nil
}
