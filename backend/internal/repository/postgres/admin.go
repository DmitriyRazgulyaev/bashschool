package postgres

import (
	"context"

	"bashkir-school/backend/internal/service"
)

func (q *Queries) EnsureAdministrator(ctx context.Context, email string,

	value string) (service.Command, error) {
	return q.exec(ctx, `INSERT INTO users(email, password_hash, display_name, ROLE)
VALUES(lower($1),$2,'Администратор','admin') ON CONFLICT (email) DO NOTHING`, email, value)
}

func (q *Queries) ChangeUserRole(ctx context.Context, role string, id string, id2 string) (service.Command, error) {
	return q.exec(ctx, `UPDATE users
SET ROLE = $1
WHERE id = $2
    AND id <> $3`, role, id, id2)
}
