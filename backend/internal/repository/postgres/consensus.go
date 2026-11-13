package postgres

import (
	"context"

	"bashkir-school/backend/internal/service"
)

func (q *Queries) ListConsensusCandidates(ctx context.Context,

	fragmentID string, pair string) ([]service.ListConsensusCandidatesRecord, error) {
	rows, err := q.db.Query(ctx, `SELECT id,
       TRANSLATION,
       canonical_id,
       is_correct
FROM candidates
WHERE fragment_id = $1
    AND lang_pair = $2`, fragmentID, pair)
	if err != nil {
		return nil, persistenceError(err)
	}
	defer rows.Close()
	records := []service.ListConsensusCandidatesRecord{}
	for rows.Next() {
		var record service.ListConsensusCandidatesRecord
		if err = rows.Scan(&record.ID, &record.Text, &record.CanonicalID, &record.Correct); err != nil {
			return nil, persistenceError(err)
		}
		records = append(records, record)
	}
	return records, persistenceError(rows.Err())
}

func (q *Queries) ListLatestVotes(ctx context.Context, fragmentID string, pair string) ([]service.ListLatestVotesRecord, error) {
	rows, err := q.db.Query(ctx, `SELECT candidate_id
FROM
    (SELECT DISTINCT ON (a.user_id) a.user_id,
                        a.candidate_id
     FROM answers a
     JOIN tasks t ON t.id = a.task_id
     WHERE t.fragment_id = $1
         AND t.lang_pair = $2
     ORDER BY a.user_id,
              a.created_at DESC,
              a.id DESC) latest`, fragmentID, pair)
	if err != nil {
		return nil, persistenceError(err)
	}
	defer rows.Close()
	records := []service.ListLatestVotesRecord{}
	for rows.Next() {
		var record service.ListLatestVotesRecord
		if err = rows.Scan(&record.ID); err != nil {
			return nil, persistenceError(err)
		}
		records = append(records, record)
	}
	return records, persistenceError(rows.Err())
}

func (q *Queries) IsExpertConfirmed(ctx context.Context, fragmentID string, pair string) (service.IsExpertConfirmedRecord, error) {
	var record service.IsExpertConfirmedRecord
	err := q.db.QueryRow(ctx, `SELECT EXISTS
    (SELECT 1
     FROM translations
     WHERE fragment_id = $1
         AND lang_pair = $2
         AND expert_confirmed)`, fragmentID, pair).Scan(&record.Expert)
	return record, persistenceError(err)
}

func (q *Queries) GetExpertTranslation(ctx context.Context, fragmentID string, pair string) (service.GetExpertTranslationRecord, error) {
	var record service.GetExpertTranslationRecord
	err := q.db.QueryRow(ctx, `SELECT TRANSLATION
FROM translations
WHERE fragment_id = $1
    AND lang_pair = $2`, fragmentID, pair).Scan(&record.Selected)
	return record, persistenceError(err)
}

func (q *Queries) UpdateConfirmationCount(ctx context.Context, fragmentID string, pair string, count int) (service.Command, error) {
	return q.exec(ctx, `UPDATE translations
SET confirmed = $3,
    updated_at = now()
WHERE fragment_id = $1
    AND lang_pair = $2`, fragmentID, pair, count)
}

func (q *Queries) SaveConsensus(ctx context.Context, fragmentID string, pair string, text string, votes int, status string) (service.Command, error) {
	return q.exec(ctx, `INSERT INTO translations(fragment_id, lang_pair, TRANSLATION, confirmed, status)
VALUES($1,$2,$3,$4,$5) ON CONFLICT(fragment_id, lang_pair) DO
UPDATE
SET TRANSLATION = EXCLUDED.translation,
                  confirmed = EXCLUDED.confirmed,
                  status = EXCLUDED.status,
                  updated_at = now()`, fragmentID, pair, text, votes, status)
}

func (q *Queries) OpenModerationCase(ctx context.Context, fragmentID string, pair string) (service.Command, error) {
	return q.exec(ctx, `INSERT INTO moderation_cases(fragment_id, lang_pair)
VALUES($1,$2) ON CONFLICT(fragment_id, lang_pair) DO
UPDATE
SET status = 'open',
    resolved_at = NULL`, fragmentID, pair)
}

func (q *Queries) CloseConsensusCase(ctx context.Context, fragmentID string, pair string) (service.Command, error) {
	return q.exec(ctx, `UPDATE moderation_cases
SET status = 'consensus',
    resolved_at = now()
WHERE fragment_id = $1
    AND lang_pair = $2
    AND status = 'open'`, fragmentID, pair)
}
