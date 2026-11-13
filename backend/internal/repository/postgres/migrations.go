package postgres

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5"
)

// Migrations are ordered, transactional and applied once. The advisory lock also
// serializes startup of multiple API replicas against the same database.
func (s *Store) Migrate(ctx context.Context, directory string) error {
	conn, err := s.pool.Acquire(ctx)
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
