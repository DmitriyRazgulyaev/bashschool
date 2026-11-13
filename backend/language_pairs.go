package main

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"regexp"
	"strings"
)

type languagePair struct {
	Code       string `json:"code"`
	SourceLang string `json:"sourceLang"`
	TargetLang string `json:"targetLang"`
	Name       string `json:"name"`
	Active     bool   `json:"active"`
}

var langPattern = regexp.MustCompile(`^[a-z]{2,3}$`)

// Pair validity is stored in PostgreSQL rather than hardcoded in Go.
func validPair(ctx context.Context, code string) bool {
	var ok bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM language_pairs WHERE code=$1 AND active=true)`, code).Scan(&ok)
	return err == nil && ok
}
func listPairs(c *gin.Context) {
	rows, err := db.Query(c.Request.Context(), `SELECT code,source_lang,target_lang,name,active FROM language_pairs WHERE active=true ORDER BY code`)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось загрузить языки")
		return
	}
	defer rows.Close()
	result := []languagePair{}
	for rows.Next() {
		var pair languagePair
		if err := rows.Scan(&pair.Code, &pair.SourceLang, &pair.TargetLang, &pair.Name, &pair.Active); err != nil {
			fail(c, 500, "DB_ERROR", "Ошибка чтения языков")
			return
		}
		result = append(result, pair)
	}
	if rows.Err() != nil {
		fail(c, 500, "DB_ERROR", "Ошибка чтения языков")
		return
	}
	jsonOK(c, result)
}
func listAllPairs(c *gin.Context) {
	rows, err := db.Query(c.Request.Context(), `SELECT code,source_lang,target_lang,name,active FROM language_pairs ORDER BY code`)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка загрузки языков")
		return
	}
	defer rows.Close()
	result := []languagePair{}
	for rows.Next() {
		var item languagePair
		if rows.Scan(&item.Code, &item.SourceLang, &item.TargetLang, &item.Name, &item.Active) != nil {
			fail(c, 500, "DB_ERROR", "Ошибка чтения языков")
			return
		}
		result = append(result, item)
	}
	if rows.Err() != nil {
		fail(c, 500, "DB_ERROR", "Ошибка чтения языков")
		return
	}
	jsonOK(c, result)
}
func createPair(c *gin.Context) {
	var in struct {
		SourceLang string `json:"sourceLang" binding:"required"`
		TargetLang string `json:"targetLang" binding:"required"`
		Name       string `json:"name" binding:"required,min=3,max=120"`
	}
	if !bindingJSON(c, &in) {
		return
	}
	in.SourceLang = strings.ToLower(strings.TrimSpace(in.SourceLang))
	in.TargetLang = strings.ToLower(strings.TrimSpace(in.TargetLang))
	if !langPattern.MatchString(in.SourceLang) || !langPattern.MatchString(in.TargetLang) || in.SourceLang == in.TargetLang {
		fail(c, 422, "INVALID_LANGUAGE", "Укажите разные коды языков длиной 2–3 латинские буквы")
		return
	}
	pair := languagePair{Code: in.SourceLang + "-" + in.TargetLang, SourceLang: in.SourceLang, TargetLang: in.TargetLang, Name: strings.TrimSpace(in.Name), Active: true}
	err := db.QueryRow(c.Request.Context(), `INSERT INTO language_pairs(code,source_lang,target_lang,name) VALUES($1,$2,$3,$4) RETURNING active`, pair.Code, pair.SourceLang, pair.TargetLang, pair.Name).Scan(&pair.Active)
	if err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) && pgerr.Code == "23505" {
			fail(c, 409, "PAIR_EXISTS", "Такая языковая пара уже существует")
			return
		}
		fail(c, 500, "DB_ERROR", "Не удалось создать языковую пару")
		return
	}
	c.JSON(201, pair)
}
func updatePair(c *gin.Context) {
	var in struct {
		Name   *string `json:"name"`
		Active *bool   `json:"active"`
	}
	if !bindingJSON(c, &in) {
		return
	}
	if in.Name == nil && in.Active == nil {
		fail(c, 422, "EMPTY_UPDATE", "Передайте name или active")
		return
	}
	if in.Name != nil && len(strings.TrimSpace(*in.Name)) < 3 {
		fail(c, 422, "INVALID_NAME", "Название слишком короткое")
		return
	}
	var item languagePair
	err := db.QueryRow(c.Request.Context(), `UPDATE language_pairs SET name=COALESCE($2,name),active=COALESCE($3,active) WHERE code=$1 RETURNING code,source_lang,target_lang,name,active`, c.Param("code"), in.Name, in.Active).Scan(&item.Code, &item.SourceLang, &item.TargetLang, &item.Name, &item.Active)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(c, 404, "PAIR_NOT_FOUND", "Пара не найдена")
		return
	}
	if err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось изменить пару")
		return
	}
	jsonOK(c, item)
}
