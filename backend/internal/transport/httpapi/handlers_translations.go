package httpapi

import (
	"github.com/gin-gonic/gin"
)

func (h *Handler) translations(c *gin.Context) {
	result, err := h.app.Translations(c.Request.Context(), parameters(c))
	respond(c, result, err)
}

func (h *Handler) exportTranslations(c *gin.Context) {
	result, err := h.app.ExportTranslations(c.Request.Context(), parameters(c))
	respond(c, result, err)
}

func (h *Handler) translationDetail(c *gin.Context) {
	result, err := h.app.TranslationDetail(c.Request.Context(), parameters(c))
	respond(c, result, err)
}
