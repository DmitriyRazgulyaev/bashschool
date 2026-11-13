package postgres

import (
	"context"

	"bashkir-school/backend/internal/service"
)

func (q *Queries) BotSession(ctx context.Context, tg int64) (service.BotSessionRecord, error) {
	var record service.BotSessionRecord
	err := q.db.QueryRow(ctx, `SELECT state,
       task_id
FROM bot_sessions
WHERE telegram_id = $1`, tg).Scan(&record.State, &record.Task)
	return record, persistenceError(err)
}
func (q *Queries) SaveBotSession(ctx context.Context, tg int64, state string, task *string) (service.Command, error) {
	return q.exec(ctx, `INSERT INTO bot_sessions (telegram_id, state, task_id)
VALUES ($1, $2, $3) ON CONFLICT (telegram_id) DO
UPDATE
SET state = EXCLUDED.state,
    task_id = EXCLUDED.task_id,
    updated_at = now()`, tg, state, task)
}
func (q *Queries) BotProfile(ctx context.Context, id string) (service.BotProfileRecord, error) {
	var record service.BotProfileRecord
	err := q.db.QueryRow(ctx, `SELECT xp,
       streak
FROM users
WHERE id = $1`, id).Scan(&record.XP, &record.Streak)
	return record, persistenceError(err)
}
func (q *Queries) BotPair(ctx context.Context, id string) (service.BotPairRecord, error) {
	var record service.BotPairRecord
	err := q.db.QueryRow(ctx, `SELECT language_pair
FROM users
WHERE id = $1`, id).Scan(&record.LanguagePair)
	return record, persistenceError(err)
}
func (q *Queries) BotSuggested(ctx context.Context, task string, id string) (service.BotSuggestedRecord, error) {
	var record service.BotSuggestedRecord
	err := q.db.QueryRow(ctx, `SELECT c.translation
FROM tasks t
JOIN candidates c ON c.id = t.candidate_id
WHERE t.id = $1
  AND t.user_id = $2`, task, id).Scan(&record.Text)
	return record, persistenceError(err)
}
func (q *Queries) BotQuestProgress(ctx context.Context, id string) (service.BotQuestProgressRecord, error) {
	var record service.BotQuestProgressRecord
	err := q.db.QueryRow(ctx, `SELECT COUNT(*)::int,
       COUNT(*) FILTER (
                        WHERE a.answer_type = 'correction')::int
FROM answers a
JOIN tasks t ON t.id = a.task_id
WHERE a.user_id = $1
  AND t.status = 'completed'
  AND t.completed_at::date = CURRENT_DATE`, id).Scan(&record.Total, &record.Corrections)
	return record, persistenceError(err)
}
