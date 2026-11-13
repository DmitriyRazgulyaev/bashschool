package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"bashkir-school/backend/internal/config"
	"bashkir-school/backend/internal/domain"

	"github.com/gin-gonic/gin"
)

func Serve(handler http.Handler) error {
	server := &http.Server{
		Addr:              ":" + config.Env("PORT", "8080"),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      45 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	finished := make(chan error, 1)
	go func() { finished <- server.ListenAndServe() }()
	select {
	case err := <-finished:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}

func requestProtection(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, domain.MaxAudioBytes+(2<<20))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Next()
	for _, err := range c.Errors {
		slog.Error("request failed", "method", c.Request.Method, "path", c.FullPath(), "error", err.Error())
	}
}

type rateWindow struct {
	start time.Time
	count int
}

// The bounded, per-process limiter protects authentication without a new service.
// Multi-replica public deployments should enforce a shared limit at the gateway.
func authenticationLimit() gin.HandlerFunc {
	var mu sync.Mutex
	windows := map[string]rateWindow{}
	return func(c *gin.Context) {
		now := time.Now()
		key := c.ClientIP()
		mu.Lock()
		if len(windows) > 10000 {
			for k, v := range windows {
				if now.Sub(v.start) >= time.Minute {
					delete(windows, k)
				}
			}
		}
		w := windows[key]
		if now.Sub(w.start) >= time.Minute {
			w = rateWindow{start: now}
		}
		w.count++
		allowed := w.count <= 20
		if len(windows) < 20000 {
			windows[key] = w
		} else {
			allowed = false
		}
		mu.Unlock()
		if !allowed {
			c.Header("Retry-After", "60")
			fail(c, 429, "RATE_LIMITED", "Слишком много попыток. Повторите через минуту.")
			c.Abort()
			return
		}
		c.Next()
	}
}
