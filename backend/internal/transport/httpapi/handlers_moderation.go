package httpapi

import (
	"github.com/gin-gonic/gin"

	"bashkir-school/backend/internal/service"
)

func (h *Handler) cases(c *gin.Context) {
	result, err := h.app.Cases(c.Request.Context(), parameters(c))
	respond(c, result, err)
}

func (h *Handler) resolveCase(c *gin.Context) {
	var in service.ResolveCaseInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "BAD_REQUEST", err.Error())
		return
	}
	result, err := h.app.ResolveCase(c.Request.Context(), parameters(c), in)
	respond(c, result, err)
}

func (h *Handler) fragmentCandidates(c *gin.Context) {
	result, err := h.app.FragmentCandidates(c.Request.Context(), parameters(c))
	respond(c, result, err)
}

func (h *Handler) addCandidate(c *gin.Context) {
	var in service.AddCandidateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "BAD_REQUEST", err.Error())
		return
	}
	result, err := h.app.AddCandidate(c.Request.Context(), parameters(c), in)
	respond(c, result, err)
}
