package httpapi

import (
	"github.com/gin-gonic/gin"
)

func (h *Handler) progress(c *gin.Context) {
	result, err := h.app.Progress(c.Request.Context(), parameters(c))
	respond(c, result, err)
}

func (h *Handler) achievements(c *gin.Context) {
	result, err := h.app.Achievements(c.Request.Context(), parameters(c))
	respond(c, result, err)
}
