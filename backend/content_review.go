package main

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// saveManualTranslations provides a complete publication path without external AI.
func saveManualTranslations(c *gin.Context) {
	var in struct {
		LanguagePair string `json:"languagePair" binding:"required"`
		Correct      string `json:"correct" binding:"required,max=2000"`
		Incorrect    string `json:"incorrect" binding:"required,max=2000"`
	}
	if !bindingJSON(c, &in) {
		return
	}
	in.Correct = strings.TrimSpace(in.Correct)
	in.Incorrect = strings.TrimSpace(in.Incorrect)
	if normalizeTranslation(in.Correct) == "" || normalizeTranslation(in.Incorrect) == "" || normalizeTranslation(in.Correct) == normalizeTranslation(in.Incorrect) {
		fail(c, 422, "INVALID_VARIANTS", "Укажите разные правильный и ошибочный переводы")
		return
	}
	ctx := c.Request.Context()
	tx, err := db.Begin(ctx)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка транзакции")
		return
	}
	defer tx.Rollback(ctx)
	var status string
	err = tx.QueryRow(ctx, `SELECT f.review_status FROM fragments f JOIN language_pairs lp ON lp.source_lang=f.source_lang AND lp.code=$2 AND lp.active WHERE f.id=$1 FOR UPDATE OF f`, c.Param("id"), in.LanguagePair).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(c, 404, "NOT_FOUND", "Фрагмент или пара не найдены")
		return
	}
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка чтения фрагмента")
		return
	}
	if status != "draft" {
		fail(c, 409, "NOT_DRAFT", "Редактировать можно только черновик")
		return
	}
	if _, err = tx.Exec(ctx, `UPDATE candidates SET is_seed=false,is_correct=NULL WHERE fragment_id=$1 AND lang_pair=$2`, c.Param("id"), in.LanguagePair); err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось обновить варианты")
		return
	}
	for _, variant := range []struct {
		Text    string
		Correct bool
	}{{in.Correct, true}, {in.Incorrect, false}} {
		_, err = tx.Exec(ctx, `INSERT INTO candidates(fragment_id,lang_pair,translation,is_seed,is_correct,normalized_text) VALUES($1,$2,$3,true,$4,$5)
   ON CONFLICT(fragment_id,lang_pair,translation) DO UPDATE SET is_seed=true,is_correct=EXCLUDED.is_correct,normalized_text=EXCLUDED.normalized_text`, c.Param("id"), in.LanguagePair, variant.Text, variant.Correct, normalizeTranslation(variant.Text))
		if err != nil {
			fail(c, 500, "DB_ERROR", "Не удалось сохранить варианты")
			return
		}
	}
	if err = tx.Commit(ctx); err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось сохранить варианты")
		return
	}
	jsonOK(c, gin.H{"status": "draft", "message": "Переводы сохранены. Проверьте их и опубликуйте фрагмент."})
}

// mergeCandidates is a human semantic decision, limited to the same fragment
// and language pair. Flattening aliases prevents cycles and chained groups.
func mergeCandidates(c *gin.Context) {
	var in struct {
		CandidateID string `json:"candidateId" binding:"required,uuid"`
		CanonicalID string `json:"canonicalId" binding:"required,uuid"`
	}
	if !bindingJSON(c, &in) {
		return
	}
	if in.CandidateID == in.CanonicalID {
		fail(c, 422, "SAME_CANDIDATE", "Выберите два разных варианта")
		return
	}
	ctx := c.Request.Context()
	tx, err := db.Begin(ctx)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка транзакции")
		return
	}
	defer tx.Rollback(ctx)
	var fragment, pair string
	err = tx.QueryRow(ctx, `SELECT a.fragment_id,a.lang_pair FROM candidates a JOIN candidates b ON a.fragment_id=b.fragment_id AND a.lang_pair=b.lang_pair WHERE a.id=$1 AND b.id=$2 AND a.canonical_id IS NULL AND b.canonical_id IS NULL AND a.fragment_id=$3`, in.CandidateID, in.CanonicalID, c.Param("id")).Scan(&fragment, &pair)
	if err != nil {
		fail(c, 422, "INVALID_GROUP", "Варианты должны быть корневыми вариантами одного фрагмента и языковой пары")
		return
	}
	var locked string
	if err = tx.QueryRow(ctx, `SELECT id FROM fragments WHERE id=$1 FOR UPDATE`, fragment).Scan(&locked); err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка блокировки")
		return
	}
	// Recheck roots after taking the fragment lock, including concurrent merges.
	tag, err := tx.Exec(ctx, `UPDATE candidates SET canonical_id=$2 WHERE (id=$1 OR canonical_id=$1)
 AND EXISTS(SELECT 1 FROM candidates WHERE id=$1 AND canonical_id IS NULL)
 AND EXISTS(SELECT 1 FROM candidates WHERE id=$2 AND canonical_id IS NULL)`, in.CandidateID, in.CanonicalID)
	if err != nil || tag.RowsAffected() == 0 {
		fail(c, 409, "GROUP_CHANGED", "Группа изменилась. Обновите варианты.")
		return
	}
	if err = reconcileTranslation(ctx, tx, fragment, pair); err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось пересчитать согласование")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось сохранить группу")
		return
	}
	jsonOK(c, gin.H{"canonicalId": in.CanonicalID})
}
