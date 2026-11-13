package httpapi

import (
	"github.com/gin-gonic/gin"

	"bashkir-school/backend/internal/service"
)

func (h *Handler) dailyTask(c *gin.Context) {
	result, err := h.app.DailyTask(c.Request.Context(), parameters(c))
	respond(c, result, err)
}

func (h *Handler) history(c *gin.Context) {
	result, err := h.app.History(c.Request.Context(), parameters(c))
	respond(c, result, err)
}

func (h *Handler) submitAnswer(c *gin.Context) {
	var in service.AnswerInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "BAD_REQUEST", err.Error())
		return
	}
	result, err := h.app.SubmitAnswer(c.Request.Context(), parameters(c), in)
	respond(c, result, err)
}

func (h *Handler) taskDetail(c *gin.Context) {
	result, err := h.app.TaskDetail(c.Request.Context(), parameters(c))
	respond(c, result, err)
}
