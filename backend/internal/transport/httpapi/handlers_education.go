package httpapi

import (
	"github.com/gin-gonic/gin"

	"bashkir-school/backend/internal/service"
)

func (h *Handler) schools(c *gin.Context) {
	result, err := h.app.Schools(c.Request.Context(), parameters(c))
	respond(c, result, err)
}

func (h *Handler) myClasses(c *gin.Context) {
	result, err := h.app.MyClasses(c.Request.Context(), parameters(c))
	respond(c, result, err)
}

func (h *Handler) createClass(c *gin.Context) {
	var in service.CreateClassInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "BAD_REQUEST", err.Error())
		return
	}
	result, err := h.app.CreateClass(c.Request.Context(), parameters(c), in)
	respond(c, result, err)
}

func (h *Handler) addStudent(c *gin.Context) {
	var in service.AddStudentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "BAD_REQUEST", err.Error())
		return
	}
	result, err := h.app.AddStudent(c.Request.Context(), parameters(c), in)
	respond(c, result, err)
}

func (h *Handler) classStudents(c *gin.Context) {
	result, err := h.app.ClassStudents(c.Request.Context(), parameters(c))
	respond(c, result, err)
}

func (h *Handler) classStatistics(c *gin.Context) {
	result, err := h.app.ClassStatistics(c.Request.Context(), parameters(c))
	respond(c, result, err)
}
