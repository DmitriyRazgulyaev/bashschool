package main

import "github.com/gin-gonic/gin"

func cases(c *gin.Context) {
	rows, err := db.Query(c, `SELECT mc.id,mc.fragment_id,f.text_ba,mc.lang_pair,mc.status,mc.created_at FROM moderation_cases mc JOIN fragments f ON f.id=mc.fragment_id ORDER BY mc.created_at DESC LIMIT 100`)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка модерации")
		return
	}
	defer rows.Close()
	arr := []gin.H{}
	for rows.Next() {
		var id, fragment, text, pair, status string
		var created any
		if err := rows.Scan(&id, &fragment, &text, &pair, &status, &created); err != nil {
			fail(c, 500, "DB_ERROR", "Ошибка чтения данных")
			return
		} else {
			arr = append(arr, gin.H{"id": id, "fragmentId": fragment, "textBa": text, "langPair": pair, "status": status, "createdAt": created})
		}
	}
	if rows.Err() != nil {
		fail(c, 500, "DB_ERROR", "Ошибка чтения данных")
		return
	}

	jsonOK(c, arr)
}
func resolveCase(c *gin.Context) {
	var in struct {
		CandidateID string `json:"candidateId" binding:"required,uuid"`
		Comment     string `json:"comment"`
	}
	if !bindingJSON(c, &in) {
		return
	}
	tx, err := db.Begin(c)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка транзакции")
		return
	}
	defer tx.Rollback(c)
	var fid, pair, tr string
	err = tx.QueryRow(c, `SELECT mc.fragment_id,mc.lang_pair,ca.translation FROM moderation_cases mc JOIN candidates ca ON ca.id=$2 AND ca.fragment_id=mc.fragment_id AND ca.lang_pair=mc.lang_pair WHERE mc.id=$1 AND mc.status='open' `, c.Param("id"), in.CandidateID).Scan(&fid, &pair, &tr)
	if err != nil {
		fail(c, 404, "NOT_FOUND", "Спор или вариант не найден")
		return
	}
	var lockedFragment string
	if err = tx.QueryRow(c, `SELECT id FROM fragments WHERE id=$1 FOR UPDATE`, fid).Scan(&lockedFragment); err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка блокировки")
		return
	}
	var open bool
	if err = tx.QueryRow(c, `SELECT status='open' FROM moderation_cases WHERE id=$1 FOR UPDATE`, c.Param("id")).Scan(&open); err != nil || !open {
		fail(c, 409, "CASE_CHANGED", "Спор уже рассмотрен. Обновите список.")
		return
	}
	var votes int
	err = tx.QueryRow(c, `SELECT count(*) FROM (SELECT DISTINCT ON (a.user_id) a.user_id,a.candidate_id FROM answers a JOIN tasks t ON t.id=a.task_id WHERE t.fragment_id=$1 AND t.lang_pair=$2 ORDER BY a.user_id,a.created_at DESC,a.id DESC) latest JOIN candidates ca ON ca.id=latest.candidate_id LEFT JOIN candidates root ON root.id=ca.canonical_id WHERE normalize_translation(COALESCE(root.translation,ca.translation))=normalize_translation($3)`, fid, pair, tr).Scan(&votes)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка подсчёта голосов")
		return
	}
	if _, err = tx.Exec(c, `UPDATE candidates ca SET is_correct=(normalize_translation(COALESCE((SELECT r.translation FROM candidates r WHERE r.id=ca.canonical_id),ca.translation))=normalize_translation($3)) WHERE ca.fragment_id=$1 AND ca.lang_pair=$2`, fid, pair, tr); err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка обновления эталона")
		return
	}
	_, err = tx.Exec(c, `INSERT INTO translations(fragment_id,lang_pair,translation,confirmed,status,expert_confirmed) VALUES($1,$2,$3,$4,'confirmed',true) ON CONFLICT(fragment_id,lang_pair) DO UPDATE SET translation=EXCLUDED.translation,confirmed=EXCLUDED.confirmed,status='confirmed',expert_confirmed=true,updated_at=now()`, fid, pair, tr, votes)
	if err == nil {
		_, err = tx.Exec(c, `UPDATE moderation_cases SET status='resolved',selected_translation=$1,reviewer_id=$2,resolved_at=now() WHERE id=$3`, tr, uid(c), c.Param("id"))
	}
	if err != nil || tx.Commit(c) != nil {
		fail(c, 500, "DB_ERROR", "Не удалось разрешить спор")
		return
	}
	jsonOK(c, gin.H{"status": "resolved", "translation": tr})
}

func fragmentCandidates(c *gin.Context) {
	rows, err := db.Query(c, `SELECT ca.id,ca.translation,COUNT(DISTINCT a.user_id) AS votes FROM candidates ca LEFT JOIN answers a ON a.candidate_id=ca.id WHERE ca.fragment_id=$1 AND ($2='' OR ca.lang_pair=$2) GROUP BY ca.id,ca.translation ORDER BY votes DESC`, c.Param("id"), c.DefaultQuery("langPair", ""))
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка вариантов")
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, tr string
		var votes int
		if err := rows.Scan(&id, &tr, &votes); err != nil {
			fail(c, 500, "DB_ERROR", "Ошибка чтения данных")
			return
		} else {
			out = append(out, gin.H{"id": id, "translation": tr, "votes": votes})
		}
	}
	if rows.Err() != nil {
		fail(c, 500, "DB_ERROR", "Ошибка чтения данных")
		return
	}

	jsonOK(c, out)
}

func addCandidate(c *gin.Context) {
	var in struct {
		Translation  string `json:"translation" binding:"required,min=1,max=2000"`
		LanguagePair string `json:"languagePair"`
	}
	if !bindingJSON(c, &in) {
		return
	}
	if in.LanguagePair == "" {
		in.LanguagePair = "ba-ru"
	}
	if !validPair(c.Request.Context(), in.LanguagePair) {
		fail(c, 422, "INVALID_PAIR", "Языковая пара не поддерживается")
		return
	}
	var id string
	err := db.QueryRow(c.Request.Context(), `INSERT INTO candidates(fragment_id,lang_pair,translation,is_seed)
 SELECT f.id,lp.code,$3,true FROM fragments f JOIN language_pairs lp ON lp.source_lang=f.source_lang AND lp.code=$2 AND lp.active=true WHERE f.id=$1
 ON CONFLICT(fragment_id,lang_pair,translation) DO UPDATE SET translation=EXCLUDED.translation RETURNING id`, c.Param("id"), in.LanguagePair, in.Translation).Scan(&id)
	if err != nil {
		fail(c, 422, "INVALID_FRAGMENT", "Не удалось добавить перевод")
		return
	}
	c.JSON(201, gin.H{"id": id})
}
