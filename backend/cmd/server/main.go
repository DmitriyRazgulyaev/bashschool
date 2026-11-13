package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"bashkir-school/backend/internal/config"
	"bashkir-school/backend/internal/repository/postgres"
	"bashkir-school/backend/internal/security"
	"bashkir-school/backend/internal/service"
	"bashkir-school/backend/internal/translation"
	"bashkir-school/backend/internal/transport/httpapi"
	"bashkir-school/backend/internal/transport/telegram"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// The executable is the only place that constructs infrastructure dependencies.
func run() error {
	tokens, err := security.New(config.Env("JWT_SECRET", ""))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	pool, err := postgres.Connect(ctx, config.Env("DATABASE_URL", "postgres://bashkir:bashkir@localhost:5432/bashkir?sslmode=disable"))
	cancel()
	if err != nil {
		return err
	}
	defer pool.Close()
	store := postgres.New(pool)
	directory := config.Env("MIGRATIONS_DIR", filepath.Dir(config.Env("MIGRATIONS_PATH", "../db/001_init.sql")))
	if err = store.Migrate(context.Background(), directory); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}
	app := service.New(store, tokens, &translation.Client{})
	if err = app.BootstrapAdmin(context.Background(), config.Env("ADMIN_EMAIL", ""), config.Env("ADMIN_PASSWORD", "")); err != nil {
		return fmt.Errorf("bootstrap administrator: %w", err)
	}
	if token := config.Env("TELEGRAM_BOT_TOKEN", ""); token != "" {
		go telegram.New(app, nil).Run(token)
	}
	return httpapi.Serve(httpapi.New(app))
}
