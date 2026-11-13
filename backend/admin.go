package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"log"
	"strings"
)

func bootstrapAdmin() {
	email := strings.TrimSpace(env("ADMIN_EMAIL", ""))
	password := env("ADMIN_PASSWORD", "")
	if email == "" || password == "" {
		return
	}
	if len(password) < 12 {
		log.Println("ADMIN_PASSWORD must be at least 12 characters; admin was not created")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println(err)
		return
	}
	_, err = db.Exec(context.Background(), `INSERT INTO users(email,password_hash,display_name,role) VALUES(lower($1),$2,'Администратор','admin') ON CONFLICT (email) DO NOTHING`, email, string(hash))
	if err != nil {
		log.Println("admin bootstrap:", err)
	}
}
func setRole(c *gin.Context) {
	var in struct {
		Role string `json:"role" binding:"required,oneof=student teacher moderator admin"`
	}
	if !bindingJSON(c, &in) {
		return
	}
	ct, err := db.Exec(c, `UPDATE users SET role=$1 WHERE id=$2 AND id<>$3`, in.Role, c.Param("id"), uid(c))
	if err != nil || ct.RowsAffected() != 1 {
		fail(c, 404, "NOT_FOUND", "Пользователь не найден или нельзя менять свою роль")
		return
	}
	jsonOK(c, gin.H{"id": c.Param("id"), "role": in.Role})
}
