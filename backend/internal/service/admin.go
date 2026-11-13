package service

import (
	"context"
	"fmt"
	"strings"

	"bashkir-school/backend/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

// BootstrapAdmin is explicit startup work; environment variables are read only
// by the executable, never by business services.
func (s *Service) BootstrapAdmin(ctx context.Context, email, password string) error {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return nil
	}
	if len(password) < 12 || len(password) > 72 {
		return fmt.Errorf("administrator password must contain 12–72 bytes")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.store.EnsureAdministrator(ctx, email, string(hash))
	return err
}
func (s *Service) SetRole(ctx context.Context, p domain.Parameters, in SetRoleInput) (domain.Result, error) {
	ct, err := s.store.ChangeUserRole(ctx, in.Role, p.ID, p.Actor.ID)
	if err != nil || ct.RowsAffected() != 1 {
		return domain.Result{}, domain.Failure(domain.NotFound, "NOT_FOUND", "Пользователь не найден или нельзя менять свою роль")
	}
	return domain.OK(domain.Object{"id": p.ID, "role": in.Role}), nil
}
