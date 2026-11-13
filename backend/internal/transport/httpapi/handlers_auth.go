package httpapi

import (
	"github.com/gin-gonic/gin"

	"bashkir-school/backend/internal/service"
)

func (h *Handler) register(c *gin.Context) {
	var in service.RegisterInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "BAD_REQUEST", err.Error())
		return
	}
	result, err := h.app.Register(c.Request.Context(), parameters(c), in)
	respond(c, result, err)
}

func (h *Handler) login(c *gin.Context) {
	var in service.LoginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "BAD_REQUEST", err.Error())
		return
	}
	result, err := h.app.Login(c.Request.Context(), parameters(c), in)
	respond(c, result, err)
}

func (h *Handler) me(c *gin.Context) {
	result, err := h.app.Me(c.Request.Context(), parameters(c))
	respond(c, result, err)
}

func (h *Handler) updateMe(c *gin.Context) {
	var in service.UpdateMeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "BAD_REQUEST", err.Error())
		return
	}
	result, err := h.app.UpdateMe(c.Request.Context(), parameters(c), in)
	respond(c, result, err)
}

func (h *Handler) linkCode(c *gin.Context) {
	result, err := h.app.LinkCode(c.Request.Context(), parameters(c))
	respond(c, result, err)
}
