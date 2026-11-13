package service

import (
	"errors"
	"strings"

	"context"

	"bashkir-school/backend/internal/domain"
)

// saveManualTranslations provides a complete publication path without external AI.
func (s *Service) SaveManualTranslations(ctx context.Context, p domain.Parameters, in SaveManualTranslationsInput) (domain.Result, error) {
	in.Correct = strings.TrimSpace(in.Correct)
	in.Incorrect = strings.TrimSpace(in.Incorrect)
	if domain.NormalizeTranslation(in.Correct) == "" || domain.NormalizeTranslation(in.Incorrect) == "" || domain.NormalizeTranslation(in.Correct) == domain.NormalizeTranslation(in.Incorrect) {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_VARIANTS", "Укажите разные правильный и ошибочный переводы")
	}

	tx, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка транзакции")
	}
	defer tx.Rollback(ctx)
	var status string
	var recordLockDraftForReview LockDraftForReviewRecord
	recordLockDraftForReview, err = tx.LockDraftForReview(ctx, p.ID, in.LanguagePair)
	status = recordLockDraftForReview.Status

	if errors.Is(err, ErrNotFound) {
		return domain.Result{}, domain.Failure(domain.NotFound, "NOT_FOUND", "Фрагмент или пара не найдены")
	}
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка чтения фрагмента")
	}
	if status != "draft" {
		return domain.Result{}, domain.Failure(domain.Conflict, "NOT_DRAFT", "Редактировать можно только черновик")
	}
	if _, err = tx.ResetSeedCandidates(ctx, p.ID, in.LanguagePair); err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось обновить варианты")
	}
	for _, variant := range []struct {
		Text    string
		Correct bool
	}{{in.Correct, true}, {in.Incorrect, false}} {
		_, err = tx.SaveReviewedCandidate(ctx, p.ID, in.LanguagePair, variant.Text, variant.Correct, domain.NormalizeTranslation(variant.Text))
		if err != nil {
			return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось сохранить варианты")
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось сохранить варианты")
	}
	return domain.OK(domain.Object{"status": "draft", "message": "Переводы сохранены. Проверьте их и опубликуйте фрагмент."}), nil
}

// mergeCandidates is a human semantic decision, limited to the same fragment
// and language pair. Flattening aliases prevents cycles and chained groups.
func (s *Service) MergeCandidates(ctx context.Context, p domain.Parameters, in MergeCandidatesInput) (domain.Result, error) {
	if in.CandidateID == in.CanonicalID {
		return domain.Result{}, domain.Failure(domain.Invalid, "SAME_CANDIDATE", "Выберите два разных варианта")
	}

	tx, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка транзакции")
	}
	defer tx.Rollback(ctx)
	var fragment, pair string
	var recordGetMergeGroup GetMergeGroupRecord
	recordGetMergeGroup, err = tx.GetMergeGroup(ctx, in.CandidateID, in.CanonicalID, p.ID)
	fragment, pair = recordGetMergeGroup.Fragment,
		recordGetMergeGroup.LanguagePair

	if err != nil {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_GROUP", "Варианты должны быть корневыми вариантами одного фрагмента и языковой пары")
	}
	{
		var recordLockMergeFragment LockMergeFragmentRecord
		recordLockMergeFragment, err = tx.LockMergeFragment(ctx, fragment)
		_ = recordLockMergeFragment.Locked

		if err != nil {
			return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка блокировки")
		}
	}
	// Recheck roots after taking the fragment lock, including concurrent merges.
	tag, err := tx.MergeSynonymGroup(ctx, in.CandidateID, in.CanonicalID)
	if err != nil || tag.RowsAffected() == 0 {
		return domain.Result{}, domain.Failure(domain.Conflict, "GROUP_CHANGED", "Группа изменилась. Обновите варианты.")
	}
	if err = s.reconcileTranslation(ctx, tx, fragment, pair); err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось пересчитать согласование")
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось сохранить группу")
	}
	return domain.OK(domain.Object{"canonicalId": in.CanonicalID}), nil
}
