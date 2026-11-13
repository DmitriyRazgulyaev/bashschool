package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type registerInput struct {
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=8,max=72"`
	DisplayName  string `json:"displayName" binding:"required,min=2"`
	LanguagePair string `json:"languagePair"`
}
type loginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func register(c *gin.Context) {
	var in registerInput
	if !bindingJSON(c, &in) {
		return
	}
	if in.LanguagePair == "" {
		in.LanguagePair = "ba-ru"
	}
	if !validPair(c.Request.Context(), in.LanguagePair) {
		fail(c, 422, "INVALID_PAIR", "Языковая пара не найдена или отключена")
		return
	}
	if len(in.Password) > 72 {
		fail(c, 422, "INVALID_PASSWORD", "Пароль должен быть не длиннее 72 байт")
		return
	}
	pwd, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(c, 500, "SERVER_ERROR", "Ошибка хеширования")
		return
	}
	var id string
	err = db.QueryRow(c, `INSERT INTO users(email,password_hash,display_name,language_pair) VALUES(lower($1),$2,$3,$4) RETURNING id`, strings.TrimSpace(in.Email), string(pwd), in.DisplayName, in.LanguagePair).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			fail(c, http.StatusConflict, "EMAIL_EXISTS", "Эта почта уже зарегистрирована. Войдите в аккаунт.")
		} else {
			slog.Error("registration failed", "error", err)
			fail(c, http.StatusInternalServerError, "DB_ERROR", "Не удалось создать аккаунт. Попробуйте позже.")
		}
		return
	}
	t, err := tokenFor(id, "student")
	if err != nil {
		fail(c, 500, "TOKEN_ERROR", "Не удалось создать сессию")
		return
	}
	c.JSON(201, gin.H{"accessToken": t, "user": gin.H{"id": id, "displayName": in.DisplayName, "role": "student", "languagePair": in.LanguagePair}})
}
func login(c *gin.Context) {
	var in loginInput
	if !bindingJSON(c, &in) {
		return
	}
	var id, hash, role, name, pair string
	err := db.QueryRow(c, `SELECT id,password_hash,role,display_name,language_pair FROM users WHERE email=lower($1)`, strings.TrimSpace(in.Email)).Scan(&id, &hash, &role, &name, &pair)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(c, http.StatusNotFound, "EMAIL_NOT_FOUND", "Пользователь с такой почтой не зарегистрирован. Создайте аккаунт.")
		return
	}
	if err != nil {
		slog.Error("login lookup failed", "error", err)
		fail(c, http.StatusInternalServerError, "DB_ERROR", "Не удалось выполнить вход. Попробуйте позже.")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		fail(c, 401, "INVALID_CREDENTIALS", "Неверный email или пароль")
		return
	}
	t, err := tokenFor(id, role)
	if err != nil {
		fail(c, 500, "TOKEN_ERROR", "Не удалось создать сессию")
		return
	}
	jsonOK(c, gin.H{"accessToken": t, "user": gin.H{"id": id, "displayName": name, "role": role, "languagePair": pair}})
}
func me(c *gin.Context) {
	var id, name, email, role, pair string
	var xp, streak int
	var school *string
	err := db.QueryRow(c, `SELECT id,display_name,email,role,language_pair,xp,streak,school_id FROM users WHERE id=$1`, uid(c)).Scan(&id, &name, &email, &role, &pair, &xp, &streak, &school)
	if err != nil {
		fail(c, 404, "NOT_FOUND", "Пользователь не найден")
		return
	}
	jsonOK(c, gin.H{"id": id, "displayName": name, "email": email, "role": role, "languagePair": pair, "xp": xp, "streak": streak, "schoolId": school})
}
func updateMe(c *gin.Context) {
	var in struct {
		DisplayName  *string `json:"displayName"`
		LanguagePair *string `json:"languagePair"`
		SchoolID     *string `json:"schoolId"`
		ClearSchool  bool    `json:"clearSchool"`
	}
	if !bindingJSON(c, &in) {
		return
	}
	if in.DisplayName != nil && len(strings.TrimSpace(*in.DisplayName)) < 2 {
		fail(c, 422, "INVALID_NAME", "Имя должно содержать минимум 2 символа")
		return
	}
	if in.LanguagePair != nil && !validPair(c.Request.Context(), *in.LanguagePair) {
		fail(c, 422, "INVALID_PAIR", "Недоступная языковая пара")
		return
	}
	if in.ClearSchool && in.SchoolID != nil {
		fail(c, 422, "INVALID_SCHOOL", "Нельзя одновременно выбрать и удалить школу")
		return
	}
	updateSchool := in.SchoolID != nil || in.ClearSchool
	_, err := db.Exec(c.Request.Context(), `UPDATE users SET display_name=COALESCE($1,display_name), language_pair=COALESCE($2,language_pair), school_id=CASE WHEN $3 THEN $4::uuid ELSE school_id END WHERE id=$5`, in.DisplayName, in.LanguagePair, updateSchool, in.SchoolID, uid(c))
	if err != nil {
		fail(c, 422, "INVALID_SCHOOL", "Неверная школа или не удалось обновить профиль")
		return
	}
	me(c)
}
func linkCode(c *gin.Context) {
	code := randomCode()
	_, err := db.Exec(c, `INSERT INTO bot_link_codes(code,user_id,expires_at) VALUES($1,$2,now()+interval '15 minutes')`, code, uid(c))
	if err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось создать код")
		return
	}
	jsonOK(c, gin.H{"code": code, "instruction": "Отправьте боту /start " + code, "expiresInSeconds": 900})
}
func accountByTelegram(ctx context.Context, tg int64) (string, error) {
	var id string
	err := db.QueryRow(ctx, `SELECT id FROM users WHERE telegram_id=$1`, tg).Scan(&id)
	return id, err
}
func bindTelegram(ctx context.Context, tg int64, code string) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var id string
	err = tx.QueryRow(ctx, `UPDATE bot_link_codes SET used_at=now() WHERE code=$1 AND used_at IS NULL AND expires_at>now() RETURNING user_id`, code).Scan(&id)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE users SET telegram_id=NULL WHERE telegram_id=$1`, tg)
	if err != nil {
		return err
	}
	ct, err := tx.Exec(ctx, `UPDATE users SET telegram_id=$1 WHERE id=$2 AND (telegram_id IS NULL OR telegram_id=$1)`, tg, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return tx.Commit(ctx)
}
