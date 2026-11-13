package httpapi

import (
	"bytes"
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"

	"bashkir-school/backend/internal/domain"

	"github.com/gin-gonic/gin"
)

func parameters(c *gin.Context) domain.Parameters {
	filters := map[string]string{}
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			filters[key] = values[0]
		}
	}
	return domain.Parameters{ID: c.Param("id"), Code: c.Param("code"), Actor: domain.Actor{ID: c.GetString("uid"), Role: c.GetString("role")}, Filters: filters}
}
func respond(c *gin.Context, result domain.Result, err error) {
	if err != nil {
		writeError(c, err)
		return
	}
	if f := result.File; f != nil {
		disposition := "attachment"
		if strings.Contains(f.ContentType, "audio") || strings.Contains(f.ContentType, "ogg") {
			disposition = "inline"
		}
		c.Header("Content-Type", f.ContentType)
		c.Header("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": f.Name}))
		c.Header("Cache-Control", "private, max-age=300")
		http.ServeContent(c.Writer, c.Request, f.Name, time.Time{}, bytes.NewReader(f.Data))
		return
	}
	status := http.StatusOK
	if result.Outcome == domain.Created {
		status = http.StatusCreated
	}
	if result.Outcome == domain.NoContent {
		c.Status(http.StatusNoContent)
		return
	}
	c.JSON(status, result.Value)
}
func writeError(c *gin.Context, err error) {
	var failure *domain.Error
	if !errors.As(err, &failure) {
		slog.Error("request failed", "path", c.FullPath(), "error", err)
		fail(c, 500, "INTERNAL_ERROR", "Не удалось выполнить запрос")
		return
	}
	status := map[domain.ErrorKind]int{domain.Invalid: 422, domain.Unauthenticated: 401, domain.Forbidden: 403, domain.NotFound: 404, domain.Conflict: 409, domain.Unavailable: 503, domain.Internal: 500}[failure.Kind]
	fail(c, status, failure.Code, failure.Message)
}
func fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
func (h *Handler) authenticate(c *gin.Context) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		fail(c, 401, "UNAUTHORIZED", "Требуется авторизация")
		c.Abort()
		return
	}
	actor, err := h.app.Authenticate(c.Request.Context(), strings.TrimPrefix(header, "Bearer "))
	if err != nil {
		writeError(c, err)
		c.Abort()
		return
	}
	c.Set("uid", actor.ID)
	c.Set("role", actor.Role)
	c.Next()
}
func roles(allowed ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, role := range allowed {
			if role == c.GetString("role") {
				c.Next()
				return
			}
		}
		fail(c, 403, "FORBIDDEN", "Недостаточно прав")
		c.Abort()
	}
}
