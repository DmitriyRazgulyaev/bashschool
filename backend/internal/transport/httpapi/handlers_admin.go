package httpapi

import (
	"github.com/gin-gonic/gin"

	"bashkir-school/backend/internal/service"
)

func (h *Handler) setRole(c *gin.Context) {
	var in service.SetRoleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "BAD_REQUEST", err.Error())
		return
	}
	result, err := h.app.SetRole(c.Request.Context(), parameters(c), in)
	respond(c, result, err)
}
