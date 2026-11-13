package postgres

import (
	"context"

	"bashkir-school/backend/internal/service"
)

func (q *Queries) LockDraftForReview(ctx context.Context,

	id string, languagePair string) (service.LockDraftForReviewRecord, error) {
	var record service.LockDraftForReviewRecord
	err := q.db.QueryRow(ctx, `SELECT f.review_status
FROM fragments f
JOIN language_pairs lp ON lp.source_lang = f.source_lang
AND lp.code = $2
AND lp.active
WHERE f.id = $1
    FOR
    UPDATE OF f`, id, languagePair).Scan(&record.Status)
	return record, persistenceError(err)
}

func (q *Queries) ResetSeedCandidates(ctx context.Context, id string, languagePair string) (service.Command, error) {
	return q.exec(ctx, `UPDATE candidates
SET is_seed = FALSE,
              is_correct = NULL
WHERE fragment_id = $1
    AND lang_pair = $2`, id, languagePair)
}

func (q *Queries) SaveReviewedCandidate(ctx context.Context, id string, languagePair string, text string, correct bool, value string) (service.Command, error) {
	return q.exec(ctx, `INSERT INTO candidates(fragment_id, lang_pair, TRANSLATION, is_seed, is_correct, normalized_text)
VALUES($1,$2,$3,TRUE,$4,$5) ON CONFLICT(fragment_id, lang_pair, TRANSLATION) DO
UPDATE
SET is_seed = TRUE,
              is_correct = EXCLUDED.is_correct,
              normalized_text = EXCLUDED.normalized_text`,

		id, languagePair, text, correct, value)
}

func (q *Queries) GetMergeGroup(ctx context.Context, candidateID string, canonicalID string, id string) (service.GetMergeGroupRecord, error) {
	var record service.GetMergeGroupRecord
	err := q.db.QueryRow(ctx, `SELECT a.fragment_id,
       a.lang_pair
FROM candidates a
JOIN candidates b ON a.fragment_id = b.fragment_id
AND a.lang_pair = b.lang_pair
WHERE a.id = $1
    AND b.id = $2
    AND a.canonical_id IS NULL
    AND b.canonical_id IS NULL
    AND a.fragment_id = $3`, candidateID, canonicalID, id).Scan(&record.Fragment, &record.LanguagePair)
	return record, persistenceError(err)
}

func (q *Queries) LockMergeFragment(ctx context.Context, fragment string) (service.LockMergeFragmentRecord, error) {
	var record service.LockMergeFragmentRecord
	err := q.db.QueryRow(ctx, `SELECT id
FROM fragments
WHERE id = $1
    FOR
    UPDATE`, fragment).Scan(&record.Locked)
	return record, persistenceError(err)
}

func (q *Queries) MergeSynonymGroup(ctx context.Context, candidateID string, canonicalID string) (service.Command, error) {
	return q.exec(ctx, `UPDATE candidates
SET canonical_id = $2
WHERE (id = $1
       OR canonical_id = $1)
    AND EXISTS
        (SELECT 1
         FROM candidates
         WHERE id = $1
             AND canonical_id IS NULL)
    AND EXISTS
        (SELECT 1
         FROM candidates
         WHERE id = $2
             AND canonical_id IS NULL)`, candidateID, canonicalID)
}
