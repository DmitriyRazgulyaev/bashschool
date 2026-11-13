package postgres

import (
	"context"

	"bashkir-school/backend/internal/service"
)

func (q *Queries) GetLearningProgress(ctx context.Context, id string) (service.GetLearningProgressRecord, error) {
	var record service.GetLearningProgressRecord
	err := q.db.QueryRow(ctx, `SELECT xp,
       streak,
       longest_streak,
    (SELECT count(*)
     FROM tasks
     WHERE user_id = $1
         AND status = 'completed'),
    (SELECT count(*)
     FROM answers a
     JOIN translations tr ON tr.fragment_id =
         (SELECT fragment_id
          FROM tasks
          WHERE id = a.task_id)
     AND tr.lang_pair =
         (SELECT lang_pair
          FROM tasks
          WHERE id = a.task_id)
     AND tr.translation = a.translation
     AND tr.status = 'confirmed'
     WHERE a.user_id = $1)
FROM users
WHERE id = $1`, id).Scan(&record.XP, &record.Streak, &record.Longest, &record.Completed, &record.Confirmed)
	return record,

		persistenceError(err)
}

func (q *Queries) CountCompletedTasks(ctx context.Context, id string) (service.CountCompletedTasksRecord, error) {
	var record service.CountCompletedTasksRecord
	err := q.db.QueryRow(ctx, `SELECT count(*)
FROM tasks
WHERE user_id = $1
    AND status = 'completed'`, id).Scan(&record.Done)
	return record, persistenceError(err)
}

func (q *Queries) ListAchievements(ctx context.Context) ([]service.ListAchievementsRecord, error) {
	rows, err := q.db.Query(ctx, `SELECT code,
       title,
       description,
       threshold
FROM achievements
ORDER BY threshold`)
	if err != nil {
		return nil, persistenceError(err)
	}
	defer rows.Close()
	records := []service.ListAchievementsRecord{}
	for rows.Next() {
		var record service.ListAchievementsRecord
		if err = rows.Scan(&record.Code, &record.Title, &record.Desc, &record.Threshold); err != nil {
			return nil, persistenceError(err)
		}
		records = append(records, record)
	}
	return records, persistenceError(rows.Err())
}
