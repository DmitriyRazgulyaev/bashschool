package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"bashkir-school/backend/internal/domain"
)

type languagePair struct {
	Code       string `json:"code"`
	SourceLang string `json:"sourceLang"`
	TargetLang string `json:"targetLang"`
	Name       string `json:"name"`
	Active     bool   `json:"active"`
}

var langPattern = regexp.MustCompile(`^[a-z]{2,3}$`)

// Pair validity is stored in PostgreSQL rather than hardcoded in Go.
func (s *Service) validPair(ctx context.Context, code string) bool {
	var ok bool
	recordIsPairActive, err := s.store.IsPairActive(ctx, code)
	ok = recordIsPairActive.Ok

	return err == nil && ok
}
func (s *Service) ListPairs(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	rows1, err := s.store.ListActivePairs(ctx)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось загрузить языки")
	}

	result := []languagePair{}
	for _, row := range rows1 {
		var pair languagePair
		{
			pair.Code, pair.SourceLang, pair.TargetLang,
				pair.Name, pair.Active = row.Code, row.SourceLang, row.TargetLang, row.Name, row.Active
		}

		result = append(result, pair)
	}

	return domain.OK(result), nil
}
func (s *Service) ListAllPairs(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	rows1, err := s.store.ListAllPairs(ctx)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка загрузки языков")
	}

	result := []languagePair{}
	for _, row := range rows1 {
		var item languagePair
		item.Code, item.SourceLang, item.TargetLang, item.Name, item.Active = row.Code, row.SourceLang, row.TargetLang, row.Name, row.Active
		result = append(result, item)
	}

	return domain.OK(result), nil
}
func (s *Service) CreatePair(ctx context.Context, p domain.Parameters, in CreatePairInput) (domain.Result, error) {
	in.SourceLang = strings.ToLower(strings.TrimSpace(in.SourceLang))
	in.TargetLang = strings.ToLower(strings.TrimSpace(in.TargetLang))
	if !langPattern.MatchString(in.SourceLang) || !langPattern.MatchString(in.TargetLang) || in.SourceLang == in.TargetLang {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_LANGUAGE", "Укажите разные коды языков длиной 2–3 латинские буквы")
	}
	pair := languagePair{Code: in.SourceLang + "-" + in.TargetLang, SourceLang: in.SourceLang, TargetLang: in.TargetLang, Name: strings.TrimSpace(in.Name), Active: true}
	recordCreatePair, err := s.store.CreatePair(ctx, pair.Code, pair.SourceLang, pair.TargetLang, pair.Name)
	pair.Active = recordCreatePair.Active

	if err != nil {
		if errors.Is(err, ErrConflict) {
			return domain.Result{}, domain.Failure(domain.Conflict, "PAIR_EXISTS", "Такая языковая пара уже существует")
		}
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось создать языковую пару")
	}
	return domain.New(pair), nil
}
func (s *Service) UpdatePair(ctx context.Context, p domain.Parameters, in UpdatePairInput) (domain.Result, error) {
	if in.Name == nil && in.Active == nil {
		return domain.Result{}, domain.Failure(domain.Invalid, "EMPTY_UPDATE", "Передайте name или active")
	}
	if in.Name != nil && len(strings.TrimSpace(*in.Name)) < 3 {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_NAME", "Название слишком короткое")
	}
	var item languagePair
	recordUpdatePair, err := s.store.UpdatePair(ctx, p.Code, in.Name, in.Active)
	item.Code, item.SourceLang, item.TargetLang,
		item.Name, item.Active = recordUpdatePair.Code, recordUpdatePair.SourceLang, recordUpdatePair.TargetLang, recordUpdatePair.Name, recordUpdatePair.Active

	if errors.Is(err, ErrNotFound) {
		return domain.Result{}, domain.Failure(domain.NotFound, "PAIR_NOT_FOUND", "Пара не найдена")
	}
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось изменить пару")
	}
	return domain.OK(item), nil
}
