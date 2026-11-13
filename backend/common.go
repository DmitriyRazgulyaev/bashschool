package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

var jwtSecret []byte

func initSecret() {
	jwtSecret = []byte(env("JWT_SECRET", ""))
	if len(jwtSecret) < 32 {
		panic("JWT_SECRET must contain at least 32 characters")
	}
}
func tokenFor(id, role string) (string, error) {
	h := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, _ := json.Marshal(map[string]any{"sub": id, "role": role, "exp": time.Now().Add(12 * time.Hour).Unix()})
	p := base64.RawURLEncoding.EncodeToString(payload)
	msg := h + "." + p
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(msg))
	return msg + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
func parseToken(t string) (string, string, error) {
	parts := strings.Split(t, ".")
	if len(parts) != 3 {
		return "", "", errors.New("invalid token")
	}
	msg := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(msg))
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(sig, mac.Sum(nil)) {
		return "", "", errors.New("invalid signature")
	}
	b, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", err
	}
	var claims struct {
		Sub  string `json:"sub"`
		Role string `json:"role"`
		Exp  int64  `json:"exp"`
	}
	if err = json.Unmarshal(b, &claims); err != nil || claims.Exp < time.Now().Unix() || claims.Sub == "" {
		return "", "", errors.New("expired or invalid token")
	}
	return claims.Sub, claims.Role, nil
}
func auth(c *gin.Context) {
	h := c.GetHeader("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		fail(c, 401, "UNAUTHORIZED", "Требуется авторизация")
		c.Abort()
		return
	}
	id, role, err := parseToken(strings.TrimPrefix(h, "Bearer "))
	if err != nil {
		fail(c, 401, "UNAUTHORIZED", "Недействительный токен")
		c.Abort()
		return
	}
	c.Set("uid", id)
	c.Set("role", role)
	c.Next()
}
func roles(allowed ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, r := range allowed {
			if r == c.GetString("role") {
				c.Next()
				return
			}
		}
		fail(c, 403, "FORBIDDEN", "Недостаточно прав")
		c.Abort()
	}
}
func fail(c *gin.Context, status int, code, msg string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": msg}})
}
func bad(c *gin.Context, err error) { fail(c, 400, "BAD_REQUEST", err.Error()) }
func uid(c *gin.Context) string     { return c.GetString("uid") }
func randomCode() string            { b := make([]byte, 18); _, _ = rand.Read(b); return hex.EncodeToString(b) }
func bindingJSON(c *gin.Context, v any) bool {
	if err := c.ShouldBindJSON(v); err != nil {
		bad(c, err)
		return false
	}
	return true
}
func bindQuery(c *gin.Context, v any) bool {
	if err := c.ShouldBindQuery(v); err != nil {
		bad(c, err)
		return false
	}
	return true
}
func numeric(c *gin.Context, key string, def, max int) int {
	var n int
	if _, err := fmt.Sscanf(c.DefaultQuery(key, fmt.Sprint(def)), "%d", &n); err != nil || n < 1 {
		return def
	}
	if n > max {
		return max
	}
	return n
}
func jsonOK(c *gin.Context, data any) { c.JSON(http.StatusOK, data) }
