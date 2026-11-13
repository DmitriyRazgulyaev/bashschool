package httpapi

import (
	"strings"

	"bashkir-school/backend/internal/config"
	"bashkir-school/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct{ app *service.Service }

func New(app *service.Service) *gin.Engine {
	h := &Handler{app: app}
	r := gin.Default()
	_ = r.SetTrustedProxies(nil)
	r.Use(requestProtection)
	r.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		for _, allowed := range strings.Split(config.Env("WEB_ORIGIN", "http://localhost:5173,http://127.0.0.1:5173"), ",") {
			if origin == strings.TrimSpace(allowed) {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				break
			}
		}
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type,Range")
		c.Header("Access-Control-Expose-Headers", "Content-Range,Accept-Ranges,Content-Length")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PATCH,PUT,OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	api := r.Group("/api/v1")
	authLimit := authenticationLimit()
	api.POST("/auth/register", authLimit, h.register)
	api.POST("/auth/login", authLimit, h.login)
	api.GET("/language-pairs", h.listPairs)
	private := api.Group("")
	private.Use(h.authenticate)
	private.GET("/users/me", h.me)
	private.PATCH("/users/me", h.updateMe)
	private.GET("/users/me/link-code", h.linkCode)
	private.GET("/tasks/daily", h.dailyTask)
	private.GET("/daily-quests", h.dailyQuests)
	private.GET("/tasks/history", h.history)
	private.GET("/tasks/:id", h.taskDetail)
	private.POST("/tasks/:id/answers", h.submitAnswer)
	private.GET("/gamification/me", h.progress)
	private.GET("/gamification/achievements", h.achievements)
	private.GET("/leaderboards", h.leaderboard)
	private.GET("/materials", h.materials)
	private.GET("/materials/:id/audio", h.materialAudio)
	private.GET("/materials/:id", h.materialDetail)
	private.GET("/fragments", h.fragments)
	private.GET("/fragments/:id", h.fragmentDetail)
	private.GET("/translations", h.translations)
	private.GET("/translations/export", h.exportTranslations)
	private.GET("/translations/:id", h.translationDetail)
	private.GET("/schools", h.schools)
	private.GET("/classes", h.myClasses)
	teacher := private.Group("")
	teacher.Use(roles("teacher", "admin"))
	teacher.POST("/classes", h.createClass)
	teacher.POST("/classes/:id/register-student", h.registerStudent)
	teacher.POST("/classes/:id/students", h.addStudent)
	teacher.GET("/classes/:id/students", h.classStudents)
	teacher.GET("/classes/:id/statistics", h.classStatistics)
	teacher.GET("/classes/:id/reports", h.classStatistics)
	mods := private.Group("")
	mods.Use(roles("moderator", "admin"))
	admin := private.Group("")
	admin.Use(roles("admin"))
	admin.POST("/schools", h.createSchool)
	admin.GET("/admin/language-pairs", h.listAllPairs)
	admin.POST("/admin/language-pairs", h.createPair)
	admin.PATCH("/admin/language-pairs/:code", h.updatePair)
	admin.POST("/materials", h.addMaterial)
	admin.POST("/fragments", h.addFragment)
	admin.POST("/admin/audio/import", h.importAudio)
	admin.POST("/admin/literature/import", h.importLiterature)
	admin.GET("/admin/literature/drafts", h.literatureDrafts)
	admin.POST("/admin/literature/fragments/:id/generate", h.generateLiteratureCandidates)
	admin.PUT("/admin/literature/fragments/:id/translations", h.saveManualTranslations)
	admin.POST("/admin/literature/fragments/:id/publish", h.publishLiteratureFragment)
	admin.POST("/admin/literature/fragments/:id/reject", h.rejectLiteratureFragment)
	mods.POST("/fragments/:id/candidates", h.addCandidate)
	mods.GET("/moderation/cases", h.cases)
	mods.POST("/fragments/:id/equivalences", h.mergeCandidates)
	mods.GET("/fragments/:id/candidates", h.fragmentCandidates)
	mods.POST("/users/:id/role", roles("admin"), h.setRole)
	mods.POST("/moderation/cases/:id/resolve", h.resolveCase)
	return r
}
