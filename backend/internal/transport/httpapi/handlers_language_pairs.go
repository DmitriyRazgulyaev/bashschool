package httpapi

import (
	"github.com/gin-gonic/gin"

	"bashkir-school/backend/internal/service"
)

func (h *Handler) listPairs(c *gin.Context) {
	result, err := h.app.ListPairs(c.Request.Context(), parameters(c))
	respond(c, result, err)
}

func (h *Handler) listAllPairs(c *gin.Context) {
	result, err := h.app.ListAllPairs(c.Request.Context(), parameters(c))
	respond(c, result, err)
}

func (h *Handler) createPair(c *gin.Context) {
	var in service.CreatePairInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "BAD_REQUEST", err.Error())
		return
	}
	result, err := h.app.CreatePair(c.Request.Context(), parameters(c), in)
	respond(c, result, err)
}

func (h *Handler) updatePair(c *gin.Context) {
	var in service.UpdatePairInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "BAD_REQUEST", err.Error())
		return
	}
	result, err := h.app.UpdatePair(c.Request.Context(), parameters(c), in)
	respond(c, result, err)
}
