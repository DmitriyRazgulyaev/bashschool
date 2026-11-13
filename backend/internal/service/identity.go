package service

import (
	"context"

	"bashkir-school/backend/internal/domain"
)

func (s *Service) Authenticate(ctx context.Context, value string) (domain.Actor, error) {
	id, _, err := s.tokens.Parse(value)
	if err != nil {
		return domain.Actor{}, domain.Failure(domain.Unauthenticated, "UNAUTHORIZED", "Недействительный токен")
	}
	role, err := s.store.UserRole(ctx, id)
	if err != nil {
		return domain.Actor{}, domain.Failure(domain.Unauthenticated, "UNAUTHORIZED", "Сессия недоступна. Войдите снова.")
	}
	return domain.Actor{ID: id, Role: role}, nil
}
