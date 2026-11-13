package main

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"os"
	"strings"
	"time"
)

var db *pgxpool.Pool

func connectDB() {
	var err error
	for i := 0; i < 30; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		db, err = pgxpool.New(ctx, env("DATABASE_URL", "postgres://bashkir:bashkir@localhost:5432/bashkir?sslmode=disable"))
		if err == nil {
			err = db.Ping(ctx)
		}
		cancel()
		if err == nil {
			return
		}
		time.Sleep(2 * time.Second)
	}
	log.Fatal("database: ", err)
}
func seedDB() {
	path := env("MIGRATIONS_PATH", "../db/001_init.sql")
	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(context.Background(), string(b), pgx.QueryExecModeSimpleProtocol)
	if err != nil {
		log.Fatal("migration 001: ", err)
	}
	upgradePath := env("LANGUAGE_MIGRATION_PATH", "../db/002_language_pairs.sql")
	upgrade, err := os.ReadFile(upgradePath)
	if err != nil {
		log.Fatal("migration 002 read: ", err)
	}
	_, err = db.Exec(context.Background(), string(upgrade), pgx.QueryExecModeSimpleProtocol)
	if err != nil {
		log.Fatal("migration 002: ", err)
	}
	questsMigration := env("DAILY_QUESTS_MIGRATION_PATH", "../db/003_daily_quests.sql")
	questsSQL, err := os.ReadFile(questsMigration)
	if err != nil {
		log.Fatal("migration 003 read: ", err)
	}
	if _, err = db.Exec(context.Background(), string(questsSQL), pgx.QueryExecModeSimpleProtocol); err != nil {
		log.Fatal("migration 003: ", err)
	}
	literatureSQL, err := os.ReadFile(env("LITERATURE_MIGRATION_PATH", "../db/004_literature_ai.sql"))
	if err != nil {
		log.Fatal("migration 004 read: ", err)
	}
	if _, err = db.Exec(context.Background(), string(literatureSQL), pgx.QueryExecModeSimpleProtocol); err != nil {
		log.Fatal("migration 004: ", err)
	}
}
func env(k, def string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	return v
}
