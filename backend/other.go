package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
)

func progress(c *gin.Context) {
	var xp, streak, longest, completed, confirmed int
	err := db.QueryRow(c, `SELECT xp,streak,longest_streak,(SELECT count(*) FROM tasks WHERE user_id=$1 AND status='completed'),(SELECT count(*) FROM answers a JOIN translations tr ON tr.fragment_id=(SELECT fragment_id FROM tasks WHERE id=a.task_id) AND tr.translation=a.translation AND tr.status='confirmed' WHERE a.user_id=$1) FROM users WHERE id=$1`, uid(c)).Scan(&xp, &streak, &longest, &completed, &confirmed)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось получить прогресс")
		return
	}
	level := 1 + xp/200
	jsonOK(c, gin.H{"xp": xp, "level": level, "xpInLevel": xp % 200, "xpForNextLevel": 200, "streak": streak, "longestStreak": longest, "completedTasks": completed, "confirmedContributions": confirmed})
}
func achievements(c *gin.Context) {
	var done int
	_ = db.QueryRow(c, `SELECT count(*) FROM tasks WHERE user_id=$1 AND status='completed'`, uid(c)).Scan(&done)
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
		if rows.Scan(&code, &title, &desc, &threshold) == nil {
			out = append(out, gin.H{"code": code, "name": title, "description": desc, "unlocked": done >= threshold, "threshold": threshold})
		}
	}
	jsonOK(c, out)
}
func leaderboard(c *gin.Context) {
	scope := c.DefaultQuery("scope", "global")
	period := c.DefaultQuery("period", "all")
	metric := c.DefaultQuery("metric", "xp")
	if metric != "xp" && metric != "activity" {
		fail(c, 422, "BAD_METRIC", "Доступны xp и activity")
		return
	}
	if period != "all" && period != "weekly" && period != "monthly" {
		fail(c, 422, "BAD_PERIOD", "Недопустимый период")
		return
	}
	if scope != "global" && scope != "school" && scope != "class" {
		fail(c, 422, "BAD_SCOPE", "Недопустимый рейтинг")
		return
	}
	periodSQL := ""
	if period == "weekly" {
		periodSQL = " AND e.created_at >= date_trunc('week', now())"
	} else if period == "monthly" {
		periodSQL = " AND e.created_at >= date_trunc('month', now())"
	}
	agg := "COALESCE(sum(e.amount),0)"
	if metric == "activity" {
		agg = "count(e.id)"
	}
	groupBy := "u.id,u.display_name"
	join := ""
	idExpr := "u.id::text"
	nameExpr := "u.display_name"
	if scope == "school" {
		join = " JOIN schools s ON s.id=u.school_id "
		groupBy = "s.id,s.name"
		idExpr = "s.id::text"
		nameExpr = "s.name"
	} else if scope == "class" {
		join = " JOIN class_members cm ON cm.user_id=u.id JOIN classes cl ON cl.id=cm.class_id "
		groupBy = "cl.id,cl.name"
		idExpr = "cl.id::text"
		nameExpr = "cl.name"
	}
	q := fmt.Sprintf(`SELECT %s,%s,%s AS score FROM users u %s LEFT JOIN xp_events e ON e.user_id=u.id %s WHERE u.role <> 'admin' GROUP BY %s ORDER BY score DESC,%s LIMIT 30`, idExpr, nameExpr, agg, join, periodSQL, groupBy, nameExpr)
	rows, err := db.Query(c, q)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка рейтинга")
		return
	}
	defer rows.Close()
	out := []gin.H{}
	rank := 0
	for rows.Next() {
		var id, name string
		var score int
		if rows.Scan(&id, &name, &score) == nil {
			rank++
			out = append(out, gin.H{"rank": rank, "entityId": id, "displayName": name, "score": score})
		}
	}
	jsonOK(c, gin.H{"scope": scope, "period": period, "metric": metric, "entries": out})
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
		if rows.Scan(&id, &title, &kind, &license, &body, &created) == nil {
			items = append(items, gin.H{"id": id, "title": title, "type": kind, "license": license, "description": body, "createdAt": created})
		}
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
		if rows.Scan(&id, &text, &title, &chapter) == nil {
			items = append(items, gin.H{"id": id, "textBa": text, "source": title, "chapter": chapter})
		}
	}
	jsonOK(c, items)
}
func translations(c *gin.Context) {
	rows, err := db.Query(c, `SELECT t.id,t.fragment_id,f.text_ba,t.lang_pair,t.translation,t.confirmed,t.status,m.title,COALESCE(f.chapter,''),t.updated_at FROM translations t JOIN fragments f ON f.id=t.fragment_id JOIN materials m ON m.id=f.material_id WHERE ($1='' OR t.lang_pair=$1) ORDER BY t.updated_at DESC LIMIT 500`, c.DefaultQuery("langPair", ""))
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
		if rows.Scan(&id, &fid, &text, &pair, &tr, &count, &status, &title, &chapter, &updated) == nil {
			out = append(out, gin.H{"id": id, "fragmentId": fid, "textBa": text, "langPair": pair, "translation": tr, "confirmed": count, "status": status, "source": gin.H{"title": title, "chapter": chapter}, "updatedAt": updated})
		}
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
	if format == "csv" {
		c.Header("Content-Disposition", "attachment; filename=translations.csv")
		c.Header("Content-Type", "text/csv; charset=utf-8")
		w := csv.NewWriter(c.Writer)
		_ = w.Write([]string{"fragment_id", "text_ba", "lang_pair", "translation", "source", "confirmed", "status"})
		for rows.Next() {
			var x item
			var source string
			if rows.Scan(&x.FragmentID, &x.TextBa, &x.LangPair, &x.Translation, &source, &x.Confirmed, &x.Status) == nil {
				_ = w.Write([]string{x.FragmentID, x.TextBa, x.LangPair, x.Translation, source, strconv.Itoa(x.Confirmed), x.Status})
			}
		}
		w.Flush()
		return
	}
	out := []item{}
	for rows.Next() {
		var x item
		var source string
		if rows.Scan(&x.FragmentID, &x.TextBa, &x.LangPair, &x.Translation, &source, &x.Confirmed, &x.Status) == nil {
			x.Source = json.RawMessage(source)
			out = append(out, x)
		}
	}
	c.Header("Content-Disposition", "attachment; filename=translations.json")
	c.JSON(200, out)
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

var _ = json.Marshal
var _ = strings.TrimSpace
var _ = http.MethodGet

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
