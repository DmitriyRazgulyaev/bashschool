package httpapi

import (
	"github.com/gin-gonic/gin"

	"bashkir-school/backend/internal/service"
)

func (h *Handler) materials(c *gin.Context) {
	result, err := h.app.Materials(c.Request.Context(), parameters(c))
	respond(c, result, err)
}

func (h *Handler) fragments(c *gin.Context) {
	result, err := h.app.Fragments(c.Request.Context(), parameters(c))
	respond(c, result, err)
}

func (h *Handler) addMaterial(c *gin.Context) {
	var in service.AddMaterialInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "BAD_REQUEST", err.Error())
		return
	}
	result, err := h.app.AddMaterial(c.Request.Context(), parameters(c), in)
	respond(c, result, err)
}

func (h *Handler) addFragment(c *gin.Context) {
	var in service.AddFragmentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "BAD_REQUEST", err.Error())
		return
	}
	result, err := h.app.AddFragment(c.Request.Context(), parameters(c), in)
	respond(c, result, err)
}

func (h *Handler) materialDetail(c *gin.Context) {
	result, err := h.app.MaterialDetail(c.Request.Context(), parameters(c))
	respond(c, result, err)
}

func (h *Handler) fragmentDetail(c *gin.Context) {
	result, err := h.app.FragmentDetail(c.Request.Context(), parameters(c))
	respond(c, result, err)
}
