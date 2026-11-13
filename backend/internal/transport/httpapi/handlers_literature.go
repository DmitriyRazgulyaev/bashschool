package httpapi

import (
	"github.com/gin-gonic/gin"

	"bashkir-school/backend/internal/service"
)

func (h *Handler) importLiterature(c *gin.Context) {
	in, err := readLiteratureUpload(c)
	if err != nil {
		writeError(c, err)
		return
	}
	result, err := h.app.ImportLiterature(c.Request.Context(), parameters(c), in)
	respond(c, result, err)
}

func (h *Handler) literatureDrafts(c *gin.Context) {
	result, err := h.app.LiteratureDrafts(c.Request.Context(), parameters(c))
	respond(c, result, err)
}

func (h *Handler) generateLiteratureCandidates(c *gin.Context) {
	result, err := h.app.GenerateLiteratureCandidates(c.Request.Context(), parameters(c))
	respond(c, result, err)
}

func (h *Handler) publishLiteratureFragment(c *gin.Context) {
	var in service.PublishLiteratureFragmentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "BAD_REQUEST", err.Error())
		return
	}
	result, err := h.app.PublishLiteratureFragment(c.Request.Context(), parameters(c), in)
	respond(c, result, err)
}

func (h *Handler) rejectLiteratureFragment(c *gin.Context) {
	result, err := h.app.RejectLiteratureFragment(c.Request.Context(), parameters(c))
	respond(c, result, err)
}
