package httpapi

import (
	"github.com/gin-gonic/gin"

	"bashkir-school/backend/internal/service"
)

func (h *Handler) saveManualTranslations(c *gin.Context) {
	var in service.SaveManualTranslationsInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "BAD_REQUEST", err.Error())
		return
	}
	result, err := h.app.SaveManualTranslations(c.Request.Context(), parameters(c), in)
	respond(c, result, err)
}

func (h *Handler) mergeCandidates(c *gin.Context) {
	var in service.MergeCandidatesInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "BAD_REQUEST", err.Error())
		return
	}
	result, err := h.app.MergeCandidates(c.Request.Context(), parameters(c), in)
	respond(c, result, err)
}
