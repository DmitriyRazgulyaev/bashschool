package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
	"time"
)

type taskRecord struct {
	ID, FragmentID, Text, Translation, Pair, Status string
	BaseXP                                          int
}

func getDaily(ctx context.Context, userID, pair string) (taskRecord, error) {
	var t taskRecord
	err := db.QueryRow(ctx, `SELECT t.id,t.fragment_id,f.text_ba,c.translation,t.lang_pair,t.status FROM tasks t JOIN fragments f ON f.id=t.fragment_id JOIN candidates c ON c.id=t.candidate_id WHERE t.user_id=$1 AND t.lang_pair=$2 AND t.assigned_date=CURRENT_DATE AND t.status='pending'`, userID, pair).Scan(&t.ID, &t.FragmentID, &t.Text, &t.Translation, &t.Pair, &t.Status)
	if err == nil {
		t.BaseXP = 10
		return t, nil
	}
	if err != pgx.ErrNoRows {
		return t, err
	}
	err = db.QueryRow(ctx, `WITH chosen AS (SELECT f.id AS fid,c.id AS cid FROM fragments f JOIN candidates c ON c.fragment_id=f.id AND c.lang_pair=$2 JOIN language_pairs lp ON lp.code=c.lang_pair AND lp.source_lang=f.source_lang AND lp.active=true WHERE f.review_status='published' AND NOT EXISTS(SELECT 1 FROM tasks old WHERE old.user_id=$1 AND old.fragment_id=f.id) ORDER BY random() LIMIT 1), fallback AS (SELECT f.id fid,c.id cid FROM fragments f JOIN candidates c ON c.fragment_id=f.id AND c.lang_pair=$2 JOIN language_pairs lp ON lp.code=c.lang_pair AND lp.source_lang=f.source_lang AND lp.active=true WHERE f.review_status='published' ORDER BY random() LIMIT 1), selected AS(SELECT * FROM chosen UNION ALL SELECT * FROM fallback WHERE NOT EXISTS(SELECT 1 FROM chosen) LIMIT 1) INSERT INTO tasks(user_id,fragment_id,candidate_id,lang_pair) SELECT $1,fid,cid,$2 FROM selected ON CONFLICT(user_id,lang_pair,assigned_date) WHERE status='pending' DO UPDATE SET user_id=EXCLUDED.user_id RETURNING id`, userID, pair).Scan(&t.ID)
	if err != nil {
		return t, err
	}
	err = db.QueryRow(ctx, `SELECT t.fragment_id,f.text_ba,c.translation,t.lang_pair,t.status FROM tasks t JOIN fragments f ON f.id=t.fragment_id JOIN candidates c ON c.id=t.candidate_id WHERE t.id=$1`, t.ID).Scan(&t.FragmentID, &t.Text, &t.Translation, &t.Pair, &t.Status)
	t.BaseXP = 10
	return t, err
}
func dailyTask(c *gin.Context) {
	pair := c.DefaultQuery("langPair", "ba-ru")
	if !validPair(c.Request.Context(), pair) {
		fail(c, 422, "INVALID_PAIR", "Пара недоступна")
		return
	}
	t, err := getDaily(c, uid(c), pair)
	if err != nil {
		if err == pgx.ErrNoRows {
			fail(c, 404, "NO_TASKS_FOR_PAIR", "Для этой пары ещё нет фрагментов с предложенными переводами. Администратор должен добавить контент.")
			return
		}
		fail(c, 500, "TASK_ERROR", "Не удалось получить задание")
		return
	}
	jsonOK(c, gin.H{"taskId": t.ID, "fragmentId": t.FragmentID, "sourceText": t.Text, "suggestedTranslation": t.Translation, "languagePair": t.Pair, "status": t.Status, "taskType": "translation_review", "reward": gin.H{"baseXp": t.BaseXP}})
}
func history(c *gin.Context) {
	limit := numeric(c, "limit", 20, 100)
	rows, err := db.Query(c, `SELECT t.id,f.text_ba,c.translation,t.status,COALESCE(t.completed_at,t.assigned_date::timestamptz) FROM tasks t JOIN fragments f ON f.id=t.fragment_id JOIN candidates c ON c.id=t.candidate_id WHERE t.user_id=$1 ORDER BY t.assigned_date DESC LIMIT $2`, uid(c), limit)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка получения истории")
		return
	}
	defer rows.Close()
	items := []gin.H{}
	for rows.Next() {
		var id, text, tr, status string
		var date time.Time
		if rows.Scan(&id, &text, &tr, &status, &date) == nil {
			items = append(items, gin.H{"taskId": id, "sourceText": text, "suggestedTranslation": tr, "status": status, "date": date})
		}
	}
	jsonOK(c, gin.H{"items": items})
}

type answerInput struct {
	AnswerType     string `json:"answerType" binding:"required,oneof=confirmation correction"`
	Translation    string `json:"translation" binding:"required,min=1,max=2000"`
	IdempotencyKey string `json:"idempotencyKey" binding:"required,uuid"`
}

func submitAnswer(c *gin.Context) {
	var in answerInput
	if !bindingJSON(c, &in) {
		return
	}
	out, status, code, msg := answerCore(c, uid(c), c.Param("id"), in)
	if status != 200 {
		fail(c, status, code, msg)
		return
	}
	jsonOK(c, out)
}
func answerCore(ctx context.Context, userID, taskID string, in answerInput) (gin.H, int, string, string) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, 500, "DB_ERROR", "Ошибка транзакции"
	}
	defer tx.Rollback(ctx)
	var status, candidateID, fragmentID, pair, suggested string
	err = tx.QueryRow(ctx, `SELECT t.status,t.candidate_id,t.fragment_id,t.lang_pair,c.translation FROM tasks t JOIN candidates c ON c.id=t.candidate_id WHERE t.id=$1 AND t.user_id=$2 FOR UPDATE OF t`, taskID, userID).Scan(&status, &candidateID, &fragmentID, &pair, &suggested)
	if err != nil {
		return nil, 404, "TASK_NOT_FOUND", "Задание не найдено"
	}
	// Serialize all XP mutations for this user, even if multiple requests arrive together.
	var lockedUser string
	if err = tx.QueryRow(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&lockedUser); err != nil {
		return nil, 500, "DB_ERROR", "Ошибка блокировки прогресса"
	}
	if status == "completed" {
		var key string
		err = tx.QueryRow(ctx, `SELECT idempotency_key FROM answers WHERE task_id=$1`, taskID).Scan(&key)
		if err == nil && key == in.IdempotencyKey {
			var xp, streak int
			_ = tx.QueryRow(ctx, `SELECT xp,streak FROM users WHERE id=$1`, userID).Scan(&xp, &streak)
			return gin.H{"status": "accepted", "xpEarned": 10, "totalXp": xp, "streak": streak, "alreadyProcessed": true}, 200, "", ""
		}
		return nil, 409, "TASK_ALREADY_COMPLETED", "Задание уже выполнено"
	}
	translation := strings.TrimSpace(in.Translation)
	if translation == "" {
		return nil, 422, "EMPTY_TRANSLATION", "Пустой перевод"
	}
	if in.AnswerType == "confirmation" && translation != suggested {
		return nil, 422, "INVALID_CONFIRMATION", "При подтверждении текст должен совпадать с предложенным"
	}
	if in.AnswerType == "correction" {
		err = tx.QueryRow(ctx, `INSERT INTO candidates(fragment_id,lang_pair,translation) VALUES($1,$2,$3) ON CONFLICT(fragment_id,lang_pair,translation) DO UPDATE SET translation=EXCLUDED.translation RETURNING id`, fragmentID, pair, translation).Scan(&candidateID)
		if err != nil {
			return nil, 500, "DB_ERROR", "Ошибка сохранения варианта"
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO answers(task_id,user_id,candidate_id,answer_type,translation,idempotency_key) VALUES($1,$2,$3,$4,$5,$6)`, taskID, userID, candidateID, in.AnswerType, translation, in.IdempotencyKey)
	if err != nil {
		return nil, 409, "DUPLICATE_ANSWER", "Повторный ответ"
	}
	_, err = tx.Exec(ctx, `UPDATE tasks SET status='completed',completed_at=now() WHERE id=$1`, taskID)
	if err != nil {
		return nil, 500, "DB_ERROR", "Ошибка обновления задания"
	}
	_, err = tx.Exec(ctx, `INSERT INTO xp_events(user_id,reason,source_id,amount) VALUES($1,'task_completed',$2,10) ON CONFLICT DO NOTHING`, userID, taskID)
	if err != nil {
		return nil, 500, "DB_ERROR", "Ошибка начисления XP"
	}
	var xp, streak int
	err = tx.QueryRow(ctx, `UPDATE users SET xp=xp+10,streak=CASE WHEN last_active_date=CURRENT_DATE THEN streak WHEN last_active_date=CURRENT_DATE-1 THEN streak+1 ELSE 1 END,longest_streak=GREATEST(longest_streak,CASE WHEN last_active_date=CURRENT_DATE THEN streak WHEN last_active_date=CURRENT_DATE-1 THEN streak+1 ELSE 1 END),last_active_date=CURRENT_DATE WHERE id=$1 RETURNING xp,streak`, userID).Scan(&xp, &streak)
	if err != nil {
		return nil, 500, "DB_ERROR", "Ошибка прогресса"
	}
	// Count votes from independent users for this fragment and language pair.
	// All SQL errors must be handled, otherwise PostgreSQL aborts the transaction.
	var winnerVotes, distinctUsers int
	var winner string
	err = tx.QueryRow(ctx, `
		SELECT COUNT(DISTINCT a.user_id)
		FROM answers a
		JOIN tasks t ON t.id = a.task_id
		WHERE t.fragment_id = $1 AND t.lang_pair = $2
	`, fragmentID, pair).Scan(&distinctUsers)
	if err != nil {
		return nil, 500, "DB_ERROR", "Ошибка подсчёта участников"
	}

	err = tx.QueryRow(ctx, `
		SELECT c.translation, COUNT(DISTINCT a.user_id) AS votes
		FROM answers a
		JOIN tasks t ON t.id = a.task_id
		JOIN candidates c ON c.id = a.candidate_id
		WHERE t.fragment_id = $1 AND t.lang_pair = $2
		GROUP BY c.id, c.translation
		ORDER BY votes DESC, c.id
		LIMIT 1
	`, fragmentID, pair).Scan(&winner, &winnerVotes)
	if err != nil {
		return nil, 500, "DB_ERROR", "Ошибка определения перевода"
	}

	if distinctUsers >= 5 && winnerVotes*100 >= distinctUsers*80 {
		_, err = tx.Exec(ctx, `
			INSERT INTO translations(fragment_id,lang_pair,translation,confirmed,status)
			VALUES($1,$2,$3,$4,'confirmed')
			ON CONFLICT(fragment_id,lang_pair) DO UPDATE
			SET translation=EXCLUDED.translation,
			    confirmed=EXCLUDED.confirmed,
			    status='confirmed',
			    updated_at=now()
		`, fragmentID, pair, winner, winnerVotes)
		if err != nil {
			return nil, 500, "DB_ERROR", "Ошибка согласования"
		}
	}
	if distinctUsers >= 5 && winnerVotes*100 < distinctUsers*80 {
		_, err = tx.Exec(ctx, `
			INSERT INTO moderation_cases(fragment_id,lang_pair)
			VALUES($1,$2) ON CONFLICT DO NOTHING
		`, fragmentID, pair)
		if err != nil {
			return nil, 500, "DB_ERROR", "Ошибка создания заявки модерации"
		}
	}
	bonusXP, unlockedQuests, err := awardDailyQuestBonuses(ctx, tx, userID)
	if err != nil {
		return nil, 500, "QUEST_ERROR", "Не удалось обновить ежедневные цели"
	}
	xp += bonusXP
	if err = tx.Commit(ctx); err != nil {
		return nil, 500, "DB_ERROR", "Ошибка фиксации"
	}
	return gin.H{"status": "accepted", "xpEarned": 10 + bonusXP, "baseXp": 10, "questBonusXp": bonusXP, "unlockedQuests": unlockedQuests, "totalXp": xp, "streak": streak}, http.StatusOK, "", ""
}

func taskDetail(c *gin.Context) {
	var taskID, fragmentID, text, tr, pair, status string
	err := db.QueryRow(c, `SELECT t.id,t.fragment_id,f.text_ba,ca.translation,t.lang_pair,t.status FROM tasks t JOIN fragments f ON f.id=t.fragment_id JOIN candidates ca ON ca.id=t.candidate_id WHERE t.id=$1 AND t.user_id=$2`, c.Param("id"), uid(c)).Scan(&taskID, &fragmentID, &text, &tr, &pair, &status)
	if err != nil {
		fail(c, 404, "NOT_FOUND", "Задание не найдено")
		return
	}
	jsonOK(c, gin.H{"taskId": taskID, "fragmentId": fragmentID, "sourceText": text, "suggestedTranslation": tr, "languagePair": pair, "status": status, "reward": gin.H{"baseXp": 10}})
}
