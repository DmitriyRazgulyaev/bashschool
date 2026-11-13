package main

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

// registerStudent creates an account and class membership atomically. Teachers
// can register students only in classes that they own.
func registerStudent(c *gin.Context) {
	if !canManageClass(c) {
		return
	}
	var in registerInput
	if !bindingJSON(c, &in) {
		return
	}
	if in.LanguagePair == "" {
		in.LanguagePair = "ba-ru"
	}
	if !validPair(c, in.LanguagePair) {
		fail(c, 422, "INVALID_PAIR", "Недоступная языковая пара")
		return
	}
	if len(in.Password) > 72 {
		fail(c, 422, "INVALID_PASSWORD", "Пароль должен быть не длиннее 72 байт")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(c, 422, "INVALID_PASSWORD", "Пароль должен содержать 8–72 байта")
		return
	}
	tx, err := db.Begin(c)
	if err != nil {
		fail(c, 500, "DB_ERROR", "Ошибка транзакции")
		return
	}
	defer tx.Rollback(c)
	var id string
	err = tx.QueryRow(c, `INSERT INTO users(email,password_hash,display_name,language_pair,school_id)
 SELECT lower($1),$2,$3,$4,school_id FROM classes WHERE id=$5 RETURNING id`, strings.TrimSpace(in.Email), string(hash), strings.TrimSpace(in.DisplayName), in.LanguagePair, c.Param("id")).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			fail(c, 409, "EMAIL_EXISTS", "Эта почта уже зарегистрирована. Добавьте существующего ученика по UUID.")
		} else {
			fail(c, 500, "DB_ERROR", "Не удалось создать аккаунт")
		}
		return
	}
	if _, err = tx.Exec(c, `INSERT INTO class_members(class_id,user_id) VALUES($1,$2)`, c.Param("id"), id); err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось добавить ученика")
		return
	}
	if err = tx.Commit(c); err != nil {
		fail(c, 500, "DB_ERROR", "Не удалось сохранить ученика")
		return
	}
	c.JSON(201, gin.H{"id": id, "displayName": in.DisplayName, "email": in.Email, "classId": c.Param("id")})
}

func createSchool(c *gin.Context) {
	var in struct {
		Name string `json:"name" binding:"required,min=2,max=200"`
	}
	if !bindingJSON(c, &in) {
		return
	}
	var id string
	err := db.QueryRow(c, `INSERT INTO schools(name) VALUES($1) RETURNING id`, strings.TrimSpace(in.Name)).Scan(&id)
	if err != nil {
		fail(c, 409, "SCHOOL_EXISTS", "Школа уже существует или не удалось её создать")
		return
	}
	c.JSON(201, gin.H{"id": id, "name": in.Name})
}
