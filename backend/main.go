package main

import (
	"log"
	"strings"

	"github.com/gin-gonic/gin"
)

func main() {
	initSecret()
	connectDB()
	defer db.Close()
	seedDB()
	bootstrapAdmin()
	if token := env("TELEGRAM_BOT_TOKEN", ""); token != "" {
		go runTelegram(token)
	}
	r := newRouter()
	if err := serveHTTP(r); err != nil {
		log.Print("HTTP server: ", err)
	}
}

func newRouter() *gin.Engine {
	r := gin.Default()
	_ = r.SetTrustedProxies(nil)
	r.Use(requestProtection)
	r.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		for _, allowed := range strings.Split(env("WEB_ORIGIN", "http://localhost:5173,http://127.0.0.1:5173"), ",") {
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
	api.POST("/auth/register", authLimit, register)
	api.POST("/auth/login", authLimit, login)
	api.GET("/language-pairs", listPairs)
	private := api.Group("")
	private.Use(auth)
	private.GET("/users/me", me)
	private.PATCH("/users/me", updateMe)
	private.GET("/users/me/link-code", linkCode)
	private.GET("/tasks/daily", dailyTask)
	private.GET("/daily-quests", dailyQuests)
	private.GET("/tasks/history", history)
	private.GET("/tasks/:id", taskDetail)
	private.POST("/tasks/:id/answers", submitAnswer)
	private.GET("/gamification/me", progress)
	private.GET("/gamification/achievements", achievements)
	private.GET("/leaderboards", leaderboard)
	private.GET("/materials", materials)
	private.GET("/materials/:id/audio", materialAudio)
	private.GET("/materials/:id", materialDetail)
	private.GET("/fragments", fragments)
	private.GET("/fragments/:id", fragmentDetail)
	private.GET("/translations", translations)
	private.GET("/translations/export", exportTranslations)
	private.GET("/translations/:id", translationDetail)
	private.GET("/schools", schools)
	private.GET("/classes", myClasses)
	teacher := private.Group("")
	teacher.Use(roles("teacher", "admin"))
	teacher.POST("/classes", createClass)
	teacher.POST("/classes/:id/register-student", registerStudent)
	teacher.POST("/classes/:id/students", addStudent)
	teacher.GET("/classes/:id/students", classStudents)
	teacher.GET("/classes/:id/statistics", classStatistics)
	teacher.GET("/classes/:id/reports", classStatistics)
	mods := private.Group("")
	mods.Use(roles("moderator", "admin"))
	admin := private.Group("")
	admin.Use(roles("admin"))
	admin.POST("/schools", createSchool)
	admin.GET("/admin/language-pairs", listAllPairs)
	admin.POST("/admin/language-pairs", createPair)
	admin.PATCH("/admin/language-pairs/:code", updatePair)
	admin.POST("/materials", addMaterial)
	admin.POST("/fragments", addFragment)
	admin.POST("/admin/audio/import", importAudio)
	admin.POST("/admin/literature/import", importLiterature)
	admin.GET("/admin/literature/drafts", literatureDrafts)
	admin.POST("/admin/literature/fragments/:id/generate", generateLiteratureCandidates)
	admin.PUT("/admin/literature/fragments/:id/translations", saveManualTranslations)
	admin.POST("/admin/literature/fragments/:id/publish", publishLiteratureFragment)
	admin.POST("/admin/literature/fragments/:id/reject", rejectLiteratureFragment)
	mods.POST("/fragments/:id/candidates", addCandidate)
	mods.GET("/moderation/cases", cases)
	mods.POST("/fragments/:id/equivalences", mergeCandidates)
	mods.GET("/fragments/:id/candidates", fragmentCandidates)
	mods.POST("/users/:id/role", roles("admin"), setRole)
	mods.POST("/moderation/cases/:id/resolve", resolveCase)
	return r
}
