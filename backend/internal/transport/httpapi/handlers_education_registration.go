package httpapi

import (
	"github.com/gin-gonic/gin"

	"bashkir-school/backend/internal/service"
)

func (h *Handler) registerStudent(c *gin.Context) {
	var in service.RegisterInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "BAD_REQUEST", err.Error())
		return
	}
	result, err := h.app.RegisterStudent(c.Request.Context(), parameters(c), in)
	respond(c, result, err)
}

func (h *Handler) createSchool(c *gin.Context) {
	var in service.CreateSchoolInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "BAD_REQUEST", err.Error())
		return
	}
	result, err := h.app.CreateSchool(c.Request.Context(), parameters(c), in)
	respond(c, result, err)
}
