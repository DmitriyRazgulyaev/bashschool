package httpapi

import (
	"github.com/gin-gonic/gin"
)

func (h *Handler) dailyQuests(c *gin.Context) {
	result, err := h.app.DailyQuests(c.Request.Context(), parameters(c))
	respond(c, result, err)
}
