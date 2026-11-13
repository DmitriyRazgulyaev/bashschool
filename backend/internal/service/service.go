package service

import (
	"context"
	"strconv"

	"bashkir-school/backend/internal/domain"
	"bashkir-school/backend/internal/security"
)

type Service struct {
	store      Store
	tokens     *security.Tokens
	translator Translator
}

type Translator interface {
	Generate(context.Context, string, string, string) (domain.AITranslation, error)
	Configured() bool
	Model() string
}

func New(store Store, tokens *security.Tokens, translator Translator) *Service {
	return &Service{store: store, tokens: tokens, translator: translator}
}
func filterLimit(value string, fallback, max int) int {
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 {
		return fallback
	}
	if n > max {
		return max
	}
	return n
}
