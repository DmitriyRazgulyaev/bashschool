package postgres

import (
	"context"

	"bashkir-school/backend/internal/service"
)

func (q *Queries) GetTaskSource(ctx context.Context, fragmentID string) (

	service.GetTaskSourceRecord, error) {
	var record service.GetTaskSourceRecord
	err := q.db.QueryRow(ctx, `SELECT m.id,
       m.title,
       COALESCE(f.chapter, ''),
       f.start_offset,
       f.end_offset,
       f.start_ms,
       f.end_ms,
       m.audio_data IS NOT NULL
FROM fragments f
JOIN materials m ON m.id = f.material_id
WHERE f.id = $1`, fragmentID).Scan(&record.MaterialID, &record.Title, &record.Chapter, &record.StartOffset, &record.
		EndOffset, &record.StartTimeMS, &record.EndTimeMS, &record.HasAudio)
	return record, persistenceError(err)
}

func (q *Queries) FindPendingTask(ctx context.Context, userID string, pair string) (service.FindPendingTaskRecord, error) {
	var record service.FindPendingTaskRecord
	err := q.db.QueryRow(ctx, `SELECT t.id,
       t.fragment_id,
       f.text_ba,
       c.translation,
       t.lang_pair,
       t.status
FROM tasks t
JOIN fragments f ON f.id = t.fragment_id
JOIN candidates c ON c.id = t.candidate_id
WHERE t.user_id = $1
    AND t.lang_pair = $2
    AND t.assigned_date = CURRENT_DATE
    AND t.status = 'pending'`, userID, pair).Scan(&record.ID, &record.FragmentID, &record.Text, &record.Translation, &record.Pair, &record.Status)
	return record, persistenceError(err)
}

func (q *Queries) AssignUnseenTask(ctx context.Context, userID string, pair string) (service.AssignUnseenTaskRecord, error) {
	var record service.AssignUnseenTaskRecord
	err := q.db.QueryRow(ctx, `WITH chosen AS
    (SELECT f.id AS fid,
            c.id AS cid
     FROM fragments f
     JOIN candidates c ON c.fragment_id = f.id
     AND c.lang_pair = $2
     JOIN language_pairs lp ON lp.code = c.lang_pair
     AND lp.source_lang = f.source_lang
     AND lp.active = TRUE
     WHERE f.review_status = 'published'
         AND c.is_seed
         AND NOT EXISTS
             (SELECT 1
              FROM tasks OLD
              WHERE old.user_id = $1
                  AND old.fragment_id = f.id
                  AND old.lang_pair = $2)
     ORDER BY random()
     LIMIT 1),
     selected AS
    (SELECT *
     FROM chosen)
INSERT INTO tasks(user_id, fragment_id, candidate_id, lang_pair)
SELECT $1,
       fid,
       cid,
       $2
FROM selected ON CONFLICT(user_id, lang_pair, assigned_date)
WHERE status = 'pending' DO
    UPDATE
    SET user_id = EXCLUDED.user_id RETURNING id`,

		userID, pair).Scan(&record.ID)
	return record, persistenceError(err)
}

func (q *Queries) GetAssignedTask(ctx context.Context, id string) (service.GetAssignedTaskRecord, error) {
	var record service.GetAssignedTaskRecord
	err := q.db.QueryRow(ctx, `SELECT t.fragment_id,
       f.text_ba,
       c.translation,
       t.lang_pair,
       t.status
FROM tasks t
JOIN fragments f ON f.id = t.fragment_id
JOIN candidates c ON c.id = t.candidate_id
WHERE t.id = $1`, id).Scan(&record.FragmentID, &record.Text, &record.Translation, &record.Pair, &record.Status)
	return record, persistenceError(err)
}

func (q *Queries) ListTaskHistory(ctx context.Context, id string, limit int) ([]service.ListTaskHistoryRecord, error) {
	rows, err := q.db.Query(ctx, `SELECT t.id,
       f.text_ba,
       c.translation,
       t.status,
       COALESCE(t.completed_at, t.assigned_date::timestamptz)
FROM tasks t
JOIN fragments f ON f.id = t.fragment_id
JOIN candidates c ON c.id = t.candidate_id
WHERE t.user_id = $1
ORDER BY t.assigned_date DESC
LIMIT $2`, id, limit)
	if err != nil {
		return nil, persistenceError(err)
	}
	defer rows.Close()
	records := []service.ListTaskHistoryRecord{}
	for rows.Next() {
		var record service.ListTaskHistoryRecord
		if err = rows.Scan(&record.ID, &record.Text, &record.Translation, &record.Status, &record.Date); err != nil {
			return nil, persistenceError(err)
		}
		records = append(records, record)
	}
	return records, persistenceError(rows.Err())
}

func (q *Queries) LockOwnedTask(ctx context.Context, taskID string, userID string) (service.LockOwnedTaskRecord, error) {
	var record service.LockOwnedTaskRecord
	err := q.db.QueryRow(ctx, `SELECT t.status,
       t.candidate_id,
       t.fragment_id,
       t.lang_pair,
       c.translation
FROM tasks t
JOIN candidates c ON c.id = t.candidate_id
WHERE t.id = $1
    AND t.user_id = $2
    FOR
    UPDATE OF t`, taskID, userID).Scan(&record.Status, &record.CandidateID, &record.FragmentID, &record.LanguagePair, &record.Suggested)
	return record, persistenceError(err)
}

func (q *Queries) LockUserProgress(ctx context.Context, userID string) (service.LockUserProgressRecord, error) {
	var record service.LockUserProgressRecord
	err := q.db.QueryRow(ctx, `SELECT id
FROM users
WHERE id = $1
    FOR
    UPDATE`, userID).Scan(&record.LockedUser)
	return record, persistenceError(err)
}

func (q *Queries) GetAnswerReceipt(ctx context.Context, taskID string) (service.GetAnswerReceiptRecord, error) {
	var record service.GetAnswerReceiptRecord
	err := q.db.QueryRow(ctx, `SELECT idempotency_key,
       RESULT
FROM answers
WHERE task_id = $1`, taskID).Scan(&record.Key, &record.Receipt)
	return record, persistenceError(err)
}

func (q *Queries) GetUserProgress(ctx context.Context, userID string) (service.GetUserProgressRecord, error) {
	var record service.GetUserProgressRecord
	err := q.db.QueryRow(ctx, `SELECT xp,
       streak
FROM users
WHERE id = $1`, userID).Scan(&record.XP, &record.Streak)
	return record, persistenceError(err)
}

func (q *Queries) LockAnswerFragment(ctx context.Context, fragmentID string) (service.LockAnswerFragmentRecord, error) {
	var record service.LockAnswerFragmentRecord
	err := q.db.QueryRow(ctx, `SELECT id
FROM fragments
WHERE id = $1
    FOR
    UPDATE`, fragmentID).Scan(&record.LockedFragment)
	return record, persistenceError(err)
}

func (q *Queries) UpsertCorrection(ctx context.Context, fragmentID string, pair string, translation string) (service.UpsertCorrectionRecord, error) {
	var record service.UpsertCorrectionRecord
	err := q.db.QueryRow(ctx, `INSERT INTO candidates(fragment_id, lang_pair, TRANSLATION)
VALUES($1,$2,$3) ON CONFLICT(fragment_id, lang_pair, TRANSLATION) DO
UPDATE
SET TRANSLATION = EXCLUDED.translation RETURNING id`, fragmentID, pair, translation).Scan(&record.CandidateID)
	return record, persistenceError(err)
}

func (q *Queries) SaveAnswer(ctx context.Context, taskID string, userID string, candidateID string, answerType string, translation string, idempotencyKey string) (service.Command, error) {
	return q.exec(ctx, `INSERT INTO answers(task_id, user_id, candidate_id, answer_type, TRANSLATION, idempotency_key)
VALUES($1,$2,$3,$4,$5,$6)`, taskID, userID, candidateID, answerType, translation, idempotencyKey)
}

func (q *Queries) CompleteTask(ctx context.Context, taskID string) (service.Command, error) {
	return q.exec(ctx, `UPDATE tasks
SET status = 'completed',
    completed_at = now()
WHERE id = $1`, taskID)
}

func (q *Queries) RecordTaskReward(ctx context.Context, userID string, taskID string) (service.Command, error) {
	return q.exec(ctx, `INSERT INTO xp_events(user_id, reason, source_id, amount)
VALUES($1,'task_completed',$2,10) ON CONFLICT DO NOTHING`, userID, taskID)
}

func (q *Queries) AdvanceUserProgress(ctx context.Context, userID string) (service.AdvanceUserProgressRecord, error) {
	var record service.AdvanceUserProgressRecord
	err := q.db.QueryRow(ctx, `UPDATE users
SET xp = xp + 10,
    streak = CASE
                 WHEN last_active_date = CURRENT_DATE THEN streak
                 WHEN last_active_date = CURRENT_DATE-1 THEN streak + 1
                 ELSE 1
             END,
             longest_streak = GREATEST(longest_streak, CASE
                                                           WHEN last_active_date = CURRENT_DATE THEN streak
                                                           WHEN last_active_date = CURRENT_DATE-1 THEN streak + 1
                                                           ELSE 1
                                                       END),
             last_active_date = CURRENT_DATE
WHERE id = $1 RETURNING xp,
                        streak`, userID).Scan(&record.XP, &record.Streak)
	return record, persistenceError(err)
}

func (q *Queries) SaveAnswerReceipt(ctx context.Context, taskID string, receipt []byte) (service.Command, error) {
	return q.exec(ctx, `UPDATE answers
SET RESULT = $2
WHERE task_id = $1`, taskID, receipt)
}

func (q *Queries) GetOwnedTask(ctx context.Context, id string, id1 string) (service.GetOwnedTaskRecord, error) {
	var record service.GetOwnedTaskRecord
	err := q.db.QueryRow(ctx, `SELECT t.id,
       t.fragment_id,
       f.text_ba,
       ca.translation,
       t.lang_pair,
       t.status
FROM tasks t
JOIN fragments f ON f.id = t.fragment_id
JOIN candidates ca ON ca.id = t.candidate_id
WHERE t.id = $1
    AND t.user_id = $2`, id, id1).Scan(&record.TaskID, &record.FragmentID, &record.Text, &record.Translation, &record.LanguagePair, &record.Status)
	return record, persistenceError(err)
}
