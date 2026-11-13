package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("database configuration: %w", err)
	}
	config.MaxConns = 20
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.ConnConfig.RuntimeParams["timezone"] = "UTC"
	config.ConnConfig.RuntimeParams["statement_timeout"] = "15000"
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	for {
		if err = pool.Ping(ctx); err == nil {
			return pool, nil
		}
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, fmt.Errorf("connect database: %w", err)
		case <-time.After(time.Second):
		}
	}
}
