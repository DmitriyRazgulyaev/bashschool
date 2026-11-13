package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Quality is based on a reviewed reference or a confirmed community translation.
// Unknown answers are excluded from the errors metric rather than called correct.
const qualityAnswersSQL = `WITH quality AS (
 SELECT a.*,t.fragment_id,t.lang_pair,f.text_ba,
 COALESCE(root.is_correct,ca.is_correct,
 CASE WHEN tr.status='confirmed' THEN normalize_translation(COALESCE(root.translation,ca.translation))=normalize_translation(tr.translation) END) AS correct
 FROM answers a JOIN tasks t ON t.id=a.task_id JOIN fragments f ON f.id=t.fragment_id
 JOIN candidates ca ON ca.id=a.candidate_id LEFT JOIN candidates root ON root.id=ca.canonical_id
 LEFT JOIN translations tr ON tr.fragment_id=t.fragment_id AND tr.lang_pair=t.lang_pair
)`

func leaderboard(c *gin.Context) {
	scope, period, metric := c.DefaultQuery("scope", "global"), c.DefaultQuery("period", "weekly"), c.DefaultQuery("metric", "xp")
	periodStart := map[string]string{"daily": "day", "weekly": "week", "monthly": "month", "yearly": "year", "all": ""}
	unit, ok := periodStart[period]
	if !ok {
		fail(c, 422, "BAD_PERIOD", "Доступны daily, weekly, monthly, yearly, all")
		return
	}
	if scope != "global" && scope != "school" && scope != "class" {
		fail(c, 422, "BAD_SCOPE", "Недопустимый рейтинг")
		return
	}
	if metric != "xp" && metric != "activity" && metric != "vocabulary" && metric != "corrections" && metric != "errors" {
		fail(c, 422, "BAD_METRIC", "Недопустимая метрика")
		return
	}
	// Only fixed fragments from the allowlists above enter SQL; user values never do.
	dateFilter := func(column string) string {
		if unit == "" {
			return ""
		}
		return fmt.Sprintf(" AND %s >= date_trunc('%s',now())", column, unit)
	}
	id, name, join := "u.id::text", "u.display_name", ""
	if scope == "school" {
		id, name, join = "s.id::text", "s.name", " JOIN schools s ON s.id=u.school_id "
	}
	if scope == "class" {
		id, name, join = "cl.id::text", "cl.name", " JOIN class_members cm ON cm.user_id=u.id JOIN classes cl ON cl.id=cm.class_id "
	}
	cte := qualityAnswersSQL
	source, score, where := "quality e", "count(e.id)", ""
	switch metric {
	case "xp":
		source, score = "xp_events e", "COALESCE(sum(e.amount),0)"
	case "corrections":
		where = " AND e.answer_type='correction'"
	case "errors":
		where = " AND e.correct=false"
	case "vocabulary":
		source = "(quality e CROSS JOIN LATERAL regexp_split_to_table(lower(e.text_ba),'[^[:alpha:]]+') AS words(word))"
		score = "count(DISTINCT words.word)"
		where = " AND e.correct=true AND words.word<>''"
	}
	order := "DESC"
	// Least-errors ranking includes only users with evaluated answers in the period.
	eligibility := ""
	if metric == "errors" {
		order = "ASC"
		eligibility = " AND EXISTS(SELECT 1 FROM quality q WHERE q.user_id=u.id AND q.correct IS NOT NULL" + dateFilter("q.created_at") + ")"
	}
	query := fmt.Sprintf(`%s SELECT %s,%s,%s AS score FROM users u %s
 LEFT JOIN %s ON e.user_id=u.id %s %s WHERE u.role='student' %s
 GROUP BY %s,%s ORDER BY score %s,%s LIMIT 100`, cte, id, name, score, join, source, dateFilter("e.created_at"), where, eligibility, id, name, order, name)
	rows, err := db.Query(c.Request.Context(), query)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось получить рейтинг")
		return
	}
	defer rows.Close()
	entries := []gin.H{}
	for rows.Next() {
		var id, name string
		var score int64
		if err = rows.Scan(&id, &name, &score); err != nil {
			fail(c, 500, "DB_ERROR", "Ошибка чтения рейтинга")
			return
		}
		entries = append(entries, gin.H{"rank": len(entries) + 1, "entityId": id, "displayName": name, "score": score})
	}
	if rows.Err() != nil {
		fail(c, 500, "DB_ERROR", "Ошибка чтения рейтинга")
		return
	}
	c.JSON(http.StatusOK, gin.H{"scope": scope, "period": period, "metric": metric, "entries": entries})
}
