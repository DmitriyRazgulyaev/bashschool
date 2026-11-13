package postgres

import (
	"context"

	"bashkir-school/backend/internal/service"
)

func (q *Queries) ListModerationCases(ctx context.Context) ([]service.ListModerationCasesRecord, error) {
	rows, err := q.db.Query(ctx, `SELECT mc.id,
       mc.fragment_id,
       f.text_ba,
       mc.lang_pair,
       mc.status,
       mc.created_at
FROM moderation_cases mc
JOIN fragments f ON f.id = mc.fragment_id
ORDER BY mc.created_at DESC
LIMIT 100`)
	if err != nil {
		return nil, persistenceError(err)
	}
	defer rows.Close()
	records := []service.ListModerationCasesRecord{}
	for rows.Next() {
		var record service.ListModerationCasesRecord
		if err = rows.Scan(&record.ID, &record.Fragment, &record.Text, &record.LanguagePair, &record.Status, &record.Created); err != nil {
			return nil, persistenceError(err)
		}
		records = append(records, record)
	}
	return records, persistenceError(rows.Err())
}

func (q *Queries) GetCaseCandidate(ctx context.Context, id string, candidateID string) (service.GetCaseCandidateRecord, error) {
	var record service.GetCaseCandidateRecord
	err := q.db.QueryRow(ctx, `SELECT mc.fragment_id,
       mc.lang_pair,
       ca.translation
FROM moderation_cases mc
JOIN candidates ca ON ca.id = $2
AND ca.fragment_id = mc.fragment_id
AND ca.lang_pair = mc.lang_pair
WHERE mc.id = $1
    AND mc.status = 'open'`, id, candidateID).Scan(&record.FragmentID, &record.LanguagePair, &record.Translation)
	return record, persistenceError(err)
}

func (q *Queries) LockModerationFragment(ctx context.Context, fid string) (service.LockModerationFragmentRecord, error) {
	var record service.LockModerationFragmentRecord
	err := q.db.QueryRow(ctx, `SELECT id
FROM fragments
WHERE id = $1
    FOR
    UPDATE`, fid).Scan(&record.LockedFragment)
	return record, persistenceError(err)
}

func (q *Queries) LockOpenCase(ctx context.Context, id string) (service.LockOpenCaseRecord, error) {
	var record service.LockOpenCaseRecord
	err := q.db.QueryRow(ctx, `SELECT status = 'open'
FROM moderation_cases
WHERE id = $1
    FOR
    UPDATE`, id).Scan(&record.Open)
	return record, persistenceError(err)
}

func (q *Queries) CountCandidateVotes(ctx context.Context, fid string, pair string, tr string) (service.CountCandidateVotesRecord, error) {
	var record service.CountCandidateVotesRecord
	err := q.db.QueryRow(ctx, `SELECT count(*)
FROM
    (SELECT DISTINCT ON (a.user_id) a.user_id,
                        a.candidate_id
     FROM answers a
     JOIN tasks t ON t.id = a.task_id
     WHERE t.fragment_id = $1
         AND t.lang_pair = $2
     ORDER BY a.user_id,
              a.created_at DESC,
              a.id DESC) latest
JOIN candidates ca ON ca.id = latest.candidate_id
LEFT JOIN candidates root ON root.id = ca.canonical_id
WHERE normalize_translation(COALESCE(root.translation, ca.translation)) = normalize_translation($3)`, fid, pair, tr).Scan(&record.Votes)
	return record, persistenceError(err)
}

func (q *Queries) MarkReviewedCandidates(ctx context.Context, fid string, pair string, tr string) (service.Command, error) {
	return q.exec(ctx, `UPDATE candidates ca
SET is_correct = (normalize_translation(COALESCE(
                                                     (SELECT r.translation
                                                      FROM candidates r
                                                      WHERE r.id = ca.canonical_id),ca.translation)) = normalize_translation($3))
WHERE ca.fragment_id = $1
    AND ca.lang_pair = $2`, fid, pair, tr)
}

func (q *Queries) SaveExpertTranslation(ctx context.Context, fid string, pair string, tr string, votes int) (service.Command, error) {
	return q.exec(ctx, `INSERT INTO translations(fragment_id, lang_pair, TRANSLATION, confirmed, status, expert_confirmed)
VALUES($1,$2,$3,$4,'confirmed',TRUE) ON CONFLICT(fragment_id, lang_pair) DO
UPDATE
SET TRANSLATION = EXCLUDED.translation,
                  confirmed = EXCLUDED.confirmed,
                  status = 'confirmed',
                  expert_confirmed = TRUE,
                                     updated_at = now()`, fid, pair, tr, votes)
}

func (q *Queries) ResolveModerationCase(ctx context.Context, tr string, id string, id2 string) (service.Command, error) {
	return q.exec(ctx, `UPDATE moderation_cases
SET status = 'resolved',
    selected_translation = $1,
    reviewer_id = $2,
    resolved_at = now()
WHERE id = $3`, tr, id, id2)
}

func (q *Queries) ListCandidateVotes(ctx context.Context, id string, value string) ([]service.ListCandidateVotesRecord, error) {
	rows, err := q.db.Query(ctx, `SELECT ca.id,
       ca.translation,
       COUNT(DISTINCT a.user_id) AS votes
FROM candidates ca
LEFT JOIN answers a ON a.candidate_id = ca.id
WHERE ca.fragment_id = $1
    AND ($2 = ''
         OR ca.lang_pair = $2)
GROUP BY ca.id,
         ca.translation
ORDER BY votes DESC`, id, value)
	if err != nil {
		return nil, persistenceError(err)
	}
	defer rows.Close()
	records := []service.ListCandidateVotesRecord{}
	for rows.Next() {
		var record service.ListCandidateVotesRecord
		if err = rows.Scan(&record.ID, &record.Translation, &record.Votes); err != nil {
			return nil, persistenceError(err)
		}
		records = append(records, record)
	}
	return records, persistenceError(rows.Err())
}

func (q *Queries) CreateSeedCandidate(ctx context.Context, id string, languagePair string, translation string) (service.CreateSeedCandidateRecord, error) {
	var record service.CreateSeedCandidateRecord
	err := q.db.QueryRow(ctx, `INSERT INTO candidates(fragment_id, lang_pair, TRANSLATION, is_seed)
SELECT f.id,
       lp.code,
       $3,
       TRUE
FROM fragments f
JOIN language_pairs lp ON lp.source_lang = f.source_lang
AND lp.code = $2
AND lp.active = TRUE
WHERE f.id = $1 ON CONFLICT(fragment_id, lang_pair, TRANSLATION) DO
    UPDATE
    SET TRANSLATION = EXCLUDED.translation RETURNING id`, id, languagePair, translation).Scan(&record.ID)
	return record, persistenceError(err)
}
