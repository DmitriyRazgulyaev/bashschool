package httpapi

import (
	"github.com/gin-gonic/gin"
)

func (h *Handler) importAudio(c *gin.Context) {
	in, err := readAudioUpload(c)
	if err != nil {
		writeError(c, err)
		return
	}
	result, err := h.app.ImportAudio(c.Request.Context(), parameters(c), in)
	respond(c, result, err)
}

func (h *Handler) materialAudio(c *gin.Context) {
	result, err := h.app.MaterialAudio(c.Request.Context(), parameters(c))
	respond(c, result, err)
}
