package postgres

import (
	"context"
	"errors"

	"bashkir-school/backend/internal/service"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type executor interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}
type Queries struct{ db executor }
type Store struct {
	*Queries
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store { return &Store{Queries: &Queries{db: pool}, pool: pool} }
func (s *Store) Begin(ctx context.Context) (service.Transaction, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &transaction{Queries: &Queries{db: tx}, tx: tx}, nil
}

type transaction struct {
	*Queries
	tx pgx.Tx
}

func (t *transaction) Commit(ctx context.Context) error   { return t.tx.Commit(ctx) }
func (t *transaction) Rollback(ctx context.Context) error { return t.tx.Rollback(ctx) }

func persistenceError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return service.ErrNotFound
	}
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23505" {
		return service.ErrConflict
	}
	return err
}
func (q *Queries) exec(ctx context.Context, sql string, args ...any) (service.Command, error) {
	tag, err := q.db.Exec(ctx, sql, args...)
	return tag, persistenceError(err)
}

var _ service.Store = (*Store)(nil)
var _ service.Transaction = (*transaction)(nil)
