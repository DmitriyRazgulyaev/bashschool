package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"
)

func progress(c *gin.Context) {
	var xp, streak, longest, completed, confirmed int
	err := db.QueryRow(c, `SELECT xp,streak,longest_streak,(SELECT count(*) FROM tasks WHERE user_id=$1 AND status='completed'),(SELECT count(*) FROM answers a JOIN translations tr ON tr.fragment_id=(SELECT fragment_id FROM tasks WHERE id=a.task_id) AND tr.lang_pair=(SELECT lang_pair FROM tasks WHERE id=a.task_id) AND tr.translation=a.translation AND tr.status='confirmed' WHERE a.user_id=$1) FROM users WHERE id=$1`, uid(c)).Scan(&xp, &streak, &longest, &completed, &confirmed)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось получить прогресс")
		return
	}
	level := 1 + xp/200
	jsonOK(c, gin.H{"xp": xp, "level": level, "xpInLevel": xp % 200, "xpForNextLevel": 200, "streak": streak, "longestStreak": longest, "completedTasks": completed, "confirmedContributions": confirmed})
}
func achievements(c *gin.Context) {
	var done int
	if err := db.QueryRow(c, `SELECT count(*) FROM tasks WHERE user_id=$1 AND status='completed'`, uid(c)).Scan(&done); err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось получить достижения")
		return
	}
	rows, err := db.Query(c, `SELECT code,title,description,threshold FROM achievements ORDER BY threshold`)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка достижений")
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var code, title, desc string
		var threshold int
		if err := rows.Scan(&code, &title, &desc, &threshold); err != nil {
			fail(c, 500, "DB_ERROR", "Ошибка чтения данных")
			return
		} else {
			out = append(out, gin.H{"code": code, "name": title, "description": desc, "unlocked": done >= threshold, "threshold": threshold})
		}
	}
	if rows.Err() != nil {
		fail(c, 500, "DB_ERROR", "Ошибка чтения данных")
		return
	}

	jsonOK(c, out)
}
func materials(c *gin.Context) {
	rows, err := db.Query(c, `SELECT id,title,kind,license,LEFT(COALESCE(body,''),180),created_at FROM materials ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка материалов")
		return
	}
	defer rows.Close()
	items := []gin.H{}
	for rows.Next() {
		var id, title, kind, license, body string
		var created any
		if err := rows.Scan(&id, &title, &kind, &license, &body, &created); err != nil {
			fail(c, 500, "DB_ERROR", "Ошибка чтения данных")
			return
		} else {
			items = append(items, gin.H{"id": id, "title": title, "type": kind, "license": license, "description": body, "createdAt": created})
		}
	}
	if rows.Err() != nil {
		fail(c, 500, "DB_ERROR", "Ошибка чтения данных")
		return
	}

	jsonOK(c, items)
}
func fragments(c *gin.Context) {
	rows, err := db.Query(c, `SELECT f.id,f.text_ba,m.title,COALESCE(f.chapter,'') FROM fragments f JOIN materials m ON m.id=f.material_id WHERE f.review_status='published' ORDER BY f.created_at DESC LIMIT 100`)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка фрагментов")
		return
	}
	defer rows.Close()
	items := []gin.H{}
	for rows.Next() {
		var id, text, title, chapter string
		if err := rows.Scan(&id, &text, &title, &chapter); err != nil {
			fail(c, 500, "DB_ERROR", "Ошибка чтения данных")
			return
		} else {
			items = append(items, gin.H{"id": id, "textBa": text, "source": title, "chapter": chapter})
		}
	}
	if rows.Err() != nil {
		fail(c, 500, "DB_ERROR", "Ошибка чтения данных")
		return
	}

	jsonOK(c, items)
}
func translations(c *gin.Context) {
	rows, err := db.Query(c, `SELECT t.id,t.fragment_id,f.text_ba,t.lang_pair,t.translation,t.confirmed,t.status,m.title,COALESCE(f.chapter,''),t.updated_at FROM translations t JOIN fragments f ON f.id=t.fragment_id JOIN materials m ON m.id=f.material_id WHERE t.status='confirmed' AND ($1='' OR t.lang_pair=$1) ORDER BY t.updated_at DESC LIMIT 500`, c.DefaultQuery("langPair", ""))
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка переводов")
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, fid, text, pair, tr, status, title, chapter string
		var count int
		var updated any
		if err := rows.Scan(&id, &fid, &text, &pair, &tr, &count, &status, &title, &chapter, &updated); err != nil {
			fail(c, 500, "DB_ERROR", "Ошибка чтения данных")
			return
		} else {
			out = append(out, gin.H{"id": id, "fragmentId": fid, "textBa": text, "langPair": pair, "translation": tr, "confirmed": count, "status": status, "source": gin.H{"title": title, "chapter": chapter}, "updatedAt": updated})
		}
	}
	if rows.Err() != nil {
		fail(c, 500, "DB_ERROR", "Ошибка чтения данных")
		return
	}

	jsonOK(c, out)
}
func exportTranslations(c *gin.Context) {
	format := c.DefaultQuery("format", "json")
	if format != "csv" && format != "json" {
		fail(c, 422, "BAD_FORMAT", "Доступны csv и json")
		return
	}
	rows, err := db.Query(c, `SELECT t.fragment_id::text,f.text_ba,t.lang_pair,t.translation,jsonb_build_object('materialId',m.id,'title',m.title,'chapter',f.chapter,'startOffset',f.start_offset,'endOffset',f.end_offset,'startTimeMs',f.start_ms,'endTimeMs',f.end_ms)::text,t.confirmed,t.status FROM translations t JOIN fragments f ON f.id=t.fragment_id JOIN materials m ON m.id=f.material_id WHERE t.status='confirmed' AND ($1='' OR t.lang_pair=$1) ORDER BY t.updated_at DESC`, c.DefaultQuery("langPair", ""))
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка выгрузки")
		return
	}
	defer rows.Close()
	type item struct {
		FragmentID  string          `json:"fragment_id"`
		TextBa      string          `json:"text_ba"`
		LangPair    string          `json:"lang_pair"`
		Translation string          `json:"translation"`
		Source      json.RawMessage `json:"source"`
		Confirmed   int             `json:"confirmed"`
		Status      string          `json:"status"`
	}
	out := []item{}
	for rows.Next() {
		var entry item
		var source string
		if err = rows.Scan(&entry.FragmentID, &entry.TextBa, &entry.LangPair, &entry.Translation, &source, &entry.Confirmed, &entry.Status); err != nil {
			fail(c, 500, "DB_ERROR", "Ошибка чтения выгрузки")
			return
		}
		entry.Source = json.RawMessage(source)
		out = append(out, entry)
	}
	if rows.Err() != nil {
		fail(c, 500, "DB_ERROR", "Ошибка чтения выгрузки")
		return
	}
	c.Header("Content-Disposition", "attachment; filename=translations."+format)
	if format == "json" {
		c.JSON(200, out)
		return
	}
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err = writer.Write([]string{"fragment_id", "text_ba", "lang_pair", "translation", "source", "confirmed", "status"}); err != nil {
		fail(c, 500, "EXPORT_ERROR", "Ошибка формирования CSV")
		return
	}
	for _, entry := range out {
		if err = writer.Write([]string{entry.FragmentID, entry.TextBa, entry.LangPair, entry.Translation, string(entry.Source), strconv.Itoa(entry.Confirmed), entry.Status}); err != nil {
			fail(c, 500, "EXPORT_ERROR", "Ошибка формирования CSV")
			return
		}
	}
	writer.Flush()
	if writer.Error() != nil {
		fail(c, 500, "EXPORT_ERROR", "Ошибка формирования CSV")
		return
	}
	c.Data(200, "text/csv; charset=utf-8", buffer.Bytes())
}

func addMaterial(c *gin.Context) {
	var in struct {
		Title       string `json:"title" binding:"required"`
		Type        string `json:"type" binding:"required,oneof=text audio"`
		License     string `json:"license" binding:"required"`
		Description string `json:"description"`
		SourceURL   string `json:"sourceUrl"`
	}
	if !bindingJSON(c, &in) {
		return
	}
	var id string
	err := db.QueryRow(c, `INSERT INTO materials(title,kind,license,body,source_url) VALUES($1,$2,$3,$4,$5) RETURNING id`, in.Title, in.Type, in.License, in.Description, in.SourceURL).Scan(&id)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось создать материал")
		return
	}
	c.JSON(201, gin.H{"id": id})
}
func addFragment(c *gin.Context) {
	var in struct {
		MaterialID  string `json:"materialId" binding:"required,uuid"`
		TextBa      string `json:"textBa" binding:"required"`
		SourceLang  string `json:"sourceLang"`
		Chapter     string `json:"chapter"`
		StartOffset *int   `json:"startOffset"`
		EndOffset   *int   `json:"endOffset"`
		StartTimeMs *int64 `json:"startTimeMs"`
		EndTimeMs   *int64 `json:"endTimeMs"`
	}
	if !bindingJSON(c, &in) {
		return
	}
	if in.SourceLang == "" {
		in.SourceLang = "ba"
	}
	if !langPattern.MatchString(in.SourceLang) {
		fail(c, 422, "INVALID_LANGUAGE", "Код языка должен содержать 2–3 латинские буквы")
		return
	}
	var id string
	err := db.QueryRow(c, `INSERT INTO fragments(material_id,text_ba,source_lang,chapter,start_offset,end_offset,start_ms,end_ms,review_status) VALUES($1,$2,$3,$4,$5,$6,$7,$8,'draft') RETURNING id`, in.MaterialID, in.TextBa, in.SourceLang, in.Chapter, in.StartOffset, in.EndOffset, in.StartTimeMs, in.EndTimeMs).Scan(&id)
	if err != nil {
		fail(c, 422, "INVALID_MATERIAL", "Не удалось создать фрагмент")
		return
	}
	c.JSON(201, gin.H{"id": id})
}

func materialDetail(c *gin.Context) {
	var id, title, kind, license, description string
	err := db.QueryRow(c, `SELECT id,title,kind,license,COALESCE(body,'') FROM materials WHERE id=$1`, c.Param("id")).Scan(&id, &title, &kind, &license, &description)
	if err != nil {
		fail(c, 404, "NOT_FOUND", "Материал не найден")
		return
	}
	jsonOK(c, gin.H{"id": id, "title": title, "type": kind, "license": license, "description": description})
}
func fragmentDetail(c *gin.Context) {
	var id, text, title, chapter string
	var start, end *int
	var startMs, endMs *int64
	err := db.QueryRow(c, `SELECT f.id,f.text_ba,m.title,COALESCE(f.chapter,''),f.start_offset,f.end_offset,f.start_ms,f.end_ms FROM fragments f JOIN materials m ON m.id=f.material_id WHERE f.id=$1 AND f.review_status='published'`, c.Param("id")).Scan(&id, &text, &title, &chapter, &start, &end, &startMs, &endMs)
	if err != nil {
		fail(c, 404, "NOT_FOUND", "Фрагмент не найден")
		return
	}
	jsonOK(c, gin.H{"id": id, "textBa": text, "source": gin.H{"title": title, "chapter": chapter, "startOffset": start, "endOffset": end, "startTimeMs": startMs, "endTimeMs": endMs}})
}
func translationDetail(c *gin.Context) {
	var id, fid, text, pair, tr, status, title, chapter string
	var count int
	err := db.QueryRow(c, `SELECT t.id,t.fragment_id,f.text_ba,t.lang_pair,t.translation,t.confirmed,t.status,m.title,COALESCE(f.chapter,'') FROM translations t JOIN fragments f ON f.id=t.fragment_id JOIN materials m ON m.id=f.material_id WHERE t.id=$1`, c.Param("id")).Scan(&id, &fid, &text, &pair, &tr, &count, &status, &title, &chapter)
	if err != nil {
		fail(c, 404, "NOT_FOUND", "Перевод не найден")
		return
	}
	jsonOK(c, gin.H{"id": id, "fragmentId": fid, "textBa": text, "langPair": pair, "translation": tr, "confirmed": count, "status": status, "source": gin.H{"title": title, "chapter": chapter}})
}
