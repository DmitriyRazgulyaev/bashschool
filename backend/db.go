package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var db *pgxpool.Pool

func connectDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	config, err := pgxpool.ParseConfig(env("DATABASE_URL", "postgres://bashkir:bashkir@localhost:5432/bashkir?sslmode=disable"))
	if err != nil {
		log.Fatal("database configuration: ", err)
	}
	config.MaxConns = 20
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.ConnConfig.RuntimeParams["timezone"] = "UTC"
	config.ConnConfig.RuntimeParams["statement_timeout"] = "15000"
	db, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatal("database pool: ", err)
	}
	for {
		if err = db.Ping(ctx); err == nil {
			return
		}
		select {
		case <-ctx.Done():
			log.Fatal("database unavailable: ", err)
		case <-time.After(time.Second):
		}
	}
}

// Migrations are ordered, transactional and applied once. The advisory lock also
// serializes startup of multiple API replicas against the same database.
func runMigrations(ctx context.Context, directory string) error {
	conn, err := db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer conn.Release()
	if _, err = conn.Exec(ctx, `SELECT pg_advisory_lock(72938401)`); err != nil {
		return err
	}
	defer conn.Exec(context.Background(), `SELECT pg_advisory_unlock(72938401)`)
	if _, err = conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	paths, err := filepath.Glob(filepath.Join(directory, "[0-9]*_*.sql"))
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no migrations in %s", directory)
	}
	sort.Strings(paths)
	for _, path := range paths {
		name := filepath.Base(path)
		var applied bool
		if err = conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, name).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		tx, err := conn.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, string(body), pgx.QueryExecModeSimpleProtocol); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES($1)`, name); err != nil {
			tx.Rollback(ctx)
			return err
		}
		if err = tx.Commit(ctx); err != nil {
			return err
		}
		log.Printf("migration applied: %s", name)
	}
	return nil
}

func seedDB() {
	directory := env("MIGRATIONS_DIR", filepath.Dir(env("MIGRATIONS_PATH", "../db/001_init.sql")))
	if err := runMigrations(context.Background(), directory); err != nil {
		log.Fatal(err)
	}
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
