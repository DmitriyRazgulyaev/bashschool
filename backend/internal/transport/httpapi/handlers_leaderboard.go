package httpapi

import (
	"github.com/gin-gonic/gin"
)

func (h *Handler) leaderboard(c *gin.Context) {
	result, err := h.app.Leaderboard(c.Request.Context(), parameters(c))
	respond(c, result, err)
}
