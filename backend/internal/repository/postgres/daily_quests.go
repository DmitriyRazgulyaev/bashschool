package postgres

import (
	"context"

	"bashkir-school/backend/internal/service"
)

func (q *Queries) EnsureDailyQuests(ctx context.Context,

	userID string) (service.Command, error) {
	return q.exec(ctx, `INSERT INTO daily_quests(user_id, quest_date, code)
SELECT $1,
       CURRENT_DATE,
       q.code
FROM (
      VALUES ('three_translations'),('five_translations'),('one_correction')) AS q(code) ON CONFLICT(user_id, quest_date, code) DO NOTHING`, userID)
}

func (q *Queries) CountDailyAnswers(ctx context.Context, userID string) (service.CountDailyAnswersRecord, error) {
	var record service.CountDailyAnswersRecord
	err := q.db.QueryRow(ctx, `SELECT COUNT(*)::int,
       COUNT(*) FILTER (
                        WHERE a.answer_type = 'correction')::int
FROM answers a
JOIN tasks t ON t.id = a.task_id
WHERE a.user_id = $1
    AND t.status = 'completed'
    AND t.completed_at::date = CURRENT_DATE`, userID).Scan(&record.Total, &record.Corrections)
	return record, persistenceError(

		err)
}

func (q *Queries) ClaimQuestReward(ctx context.Context, userID string, code string) (service.ClaimQuestRewardRecord, error) {
	var record service.ClaimQuestRewardRecord
	err := q.db.QueryRow(ctx, `UPDATE daily_quests
SET rewarded_at = NOW()
WHERE user_id = $1
    AND quest_date = CURRENT_DATE
    AND code = $2
    AND rewarded_at IS NULL RETURNING id`, userID, code).Scan(&record.QuestID)
	return record, persistenceError(err)
}

func (q *Queries) RecordQuestReward(ctx context.Context, userID string, questID string, rewardXP int) (service.Command, error) {
	return q.exec(ctx, `INSERT INTO xp_events(user_id, reason, source_id, amount)
VALUES($1,'daily_quest',$2,$3) ON CONFLICT DO NOTHING`, userID, questID, rewardXP)
}

func (q *Queries) AddQuestExperience(ctx context.Context, bonus int, userID string) (service.Command, error) {
	return q.exec(ctx, `UPDATE users
SET xp = xp + $1
WHERE id = $2`, bonus, userID)
}

func (q *Queries) GetDailyAnswerCounts(ctx context.Context, userID string) (service.GetDailyAnswerCountsRecord, error) {
	var record service.GetDailyAnswerCountsRecord
	err := q.db.QueryRow(ctx, `SELECT COUNT(*)::int,
       COUNT(*) FILTER (
                        WHERE a.answer_type = 'correction')::int
FROM answers a
JOIN tasks t ON t.id = a.task_id
WHERE a.user_id = $1
    AND t.status = 'completed'
    AND t.completed_at::date = CURRENT_DATE`, userID).Scan(&record.Total, &record.Corrections)
	return record, persistenceError(err)
}

func (q *Queries) ListDailyQuestClaims(ctx context.Context, userID string) ([]service.ListDailyQuestClaimsRecord, error) {
	rows, err := q.db.Query(ctx, `SELECT code,
       rewarded_at
FROM daily_quests
WHERE user_id = $1
    AND quest_date = CURRENT_DATE`, userID)
	if err != nil {
		return nil, persistenceError(err)
	}
	defer rows.Close()
	records := []service.ListDailyQuestClaimsRecord{}
	for rows.Next() {
		var record service.ListDailyQuestClaimsRecord
		if err = rows.Scan(&record.Code, &record.Awarded); err != nil {
			return nil, persistenceError(err)
		}
		records = append(records, record)
	}
	return records, persistenceError(rows.Err())
}
