package integration

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"bashkir-school/backend/internal/repository/postgres"
	"bashkir-school/backend/internal/security"
	"bashkir-school/backend/internal/service"
	"bashkir-school/backend/internal/transport/telegram"

	"github.com/jackc/pgx/v5/pgxpool"
)

type telegramStub func(*http.Request) (*http.Response, error)

func (f telegramStub) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Opt-in integration tests create an isolated schema, never truncate user tables.
func TestBotFSMAndMigrationRestart(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("integration_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(context.Background(), `DROP SCHEMA `+schema+` CASCADE`)
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := postgres.New(pool)
	if err = store.Migrate(ctx, "../../../db"); err != nil {
		t.Fatal(err)
	}
	if err = store.Migrate(ctx, "../../../db"); err != nil {
		t.Fatal("restart:", err)
	}
	tokens, err := security.New("test_only_secret_long_enough_for_hs256_signing")
	if err != nil {
		t.Fatal(err)
	}
	app := service.New(store, tokens, nil)
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM candidates`).Scan(&count); err != nil || count != 24 {
		t.Fatalf("seed candidates=%d, err=%v", count, err)
	}
	var user string
	if err = pool.QueryRow(ctx, `INSERT INTO users(email,password_hash,display_name,telegram_id) VALUES('bot@example.org','test','Bot',123) RETURNING id`).Scan(&user); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: telegramStub(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})}
	bot := telegram.New(app, client)
	bot.Handle(ctx, "test-token", 123, 123, "/daily", false)
	state, task, err := app.BotSession(ctx, 123)
	if err != nil || state != "waiting_answer" || task == nil {
		t.Fatalf("daily state=%s task=%v error=%v", state, task, err)
	}
	// Old buttons must not act on a newly assigned task.
	bot.Handle(ctx, "test-token", 123, 123, "confirm:00000000-0000-4000-a000-000000000000", true)
	state, _, _ = app.BotSession(ctx, 123)
	if state != "waiting_answer" {
		t.Fatal("stale button changed state")
	}
	bot.Handle(ctx, "test-token", 123, 123, "correct:"+*task, true)
	state, _, _ = app.BotSession(ctx, 123)
	if state != "waiting_correction" {
		t.Fatal("correction state was not persisted")
	}
	bot.Handle(ctx, "test-token", 123, 123, "/cancel", false)
	state, task, err = app.BotSession(ctx, 123)
	if err != nil || state != "idle" || task != nil {
		t.Fatal("cancel did not reset persisted state")
	}
	bot.Handle(ctx, "test-token", 123, 123, "/start", false)
}
