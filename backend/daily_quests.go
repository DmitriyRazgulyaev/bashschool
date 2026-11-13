package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// Daily challenges are server-owned; clients cannot choose their rewards or progress.
type questDefinition struct {
	Code, Title, Description string
	Goal, RewardXP           int
}

var dailyQuestDefinitions = []questDefinition{
	{"three_translations", "Первые 3 перевода", "Подтвердите или исправьте 3 предложенных перевода", 3, 50},
	{"five_translations", "Мастер переводов", "Выполните 5 переводов за сегодня", 5, 100},
	{"one_correction", "Внимательный переводчик", "Предложите собственное исправление перевода", 1, 25},
}

// beginDailyQuests ensures that each user receives the same challenge list throughout a day.
func beginDailyQuests(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `INSERT INTO daily_quests(user_id,quest_date,code)
        SELECT $1, CURRENT_DATE, q.code
        FROM (VALUES ('three_translations'),('five_translations'),('one_correction')) AS q(code)
        ON CONFLICT(user_id,quest_date,code) DO NOTHING`, userID)
	return err
}

// awardDailyQuestBonuses is called in the same transaction as task completion.
// It receives a locked user row from answerCore, preventing concurrent double rewards.
func awardDailyQuestBonuses(ctx context.Context, tx pgx.Tx, userID string) (int, []string, error) {
	if err := beginDailyQuests(ctx, tx, userID); err != nil {
		return 0, nil, err
	}
	var total, corrections int
	err := tx.QueryRow(ctx, `SELECT COUNT(*)::int,
        COUNT(*) FILTER (WHERE a.answer_type='correction')::int
        FROM answers a JOIN tasks t ON t.id=a.task_id
        WHERE a.user_id=$1 AND t.status='completed' AND t.completed_at::date=CURRENT_DATE`, userID).Scan(&total, &corrections)
	if err != nil {
		return 0, nil, err
	}

	var bonus int
	unlocked := []string{}
	for _, quest := range dailyQuestDefinitions {
		count := total
		if quest.Code == "one_correction" {
			count = corrections
		}
		if count < quest.Goal {
			continue
		}
		var questID string
		// A row may be rewarded once; RETURNING only occurs on the first update.
		err := tx.QueryRow(ctx, `UPDATE daily_quests SET rewarded_at=NOW()
            WHERE user_id=$1 AND quest_date=CURRENT_DATE AND code=$2
                AND rewarded_at IS NULL RETURNING id`, userID, quest.Code).Scan(&questID)
		if err == pgx.ErrNoRows {
			continue
		}
		if err != nil {
			return 0, nil, err
		}
		_, err = tx.Exec(ctx, `INSERT INTO xp_events(user_id,reason,source_id,amount)
            VALUES($1,'daily_quest',$2,$3) ON CONFLICT DO NOTHING`, userID, questID, quest.RewardXP)
		if err != nil {
			return 0, nil, err
		}
		bonus += quest.RewardXP
		unlocked = append(unlocked, quest.Title)
	}
	if bonus > 0 {
		_, err = tx.Exec(ctx, `UPDATE users SET xp=xp+$1 WHERE id=$2`, bonus, userID)
		if err != nil {
			return 0, nil, err
		}
	}
	return bonus, unlocked, nil
}

func dailyQuests(c *gin.Context) {
	ctx := c.Request.Context()
	userID := uid(c)
	tx, err := db.Begin(ctx)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось получить ежедневные цели")
		return
	}
	defer tx.Rollback(ctx)
	if err := beginDailyQuests(ctx, tx, userID); err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось создать ежедневные цели")
		return
	}
	var total, corrections int
	err = tx.QueryRow(ctx, `SELECT COUNT(*)::int,
        COUNT(*) FILTER (WHERE a.answer_type='correction')::int
        FROM answers a JOIN tasks t ON t.id=a.task_id
        WHERE a.user_id=$1 AND t.status='completed' AND t.completed_at::date=CURRENT_DATE`, userID).Scan(&total, &corrections)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось подсчитать прогресс")
		return
	}

	rows, err := tx.Query(ctx, `SELECT code, rewarded_at FROM daily_quests
        WHERE user_id=$1 AND quest_date=CURRENT_DATE`, userID)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось загрузить цели")
		return
	}
	claimed := map[string]*time.Time{}
	for rows.Next() {
		var code string
		var awarded *time.Time
		if err := rows.Scan(&code, &awarded); err != nil {
			rows.Close()
			fail(c, 500, "DB_ERROR", "Ошибка чтения целей")
			return
		}
		claimed[code] = awarded
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		fail(c, 500, "DB_ERROR", "Ошибка чтения целей")
		return
	}
	rows.Close()
	if err := tx.Commit(ctx); err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка фиксации целей")
		return
	}

	items := []gin.H{}
	for _, definition := range dailyQuestDefinitions {
		count := total
		if definition.Code == "one_correction" {
			count = corrections
		}
		if count > definition.Goal {
			count = definition.Goal
		}
		items = append(items, gin.H{
			"code": definition.Code, "title": definition.Title,
			"description": definition.Description, "goal": definition.Goal,
			"progress": count, "rewardXp": definition.RewardXP,
			"completed": count >= definition.Goal, "rewarded": claimed[definition.Code] != nil,
		})
	}
	c.JSON(http.StatusOK, gin.H{"date": time.Now().UTC().Format("2006-01-02"), "quests": items})
}
