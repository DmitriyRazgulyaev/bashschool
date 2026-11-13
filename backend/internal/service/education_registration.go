package service

import (
	"errors"
	"strings"

	"context"

	"bashkir-school/backend/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

// registerStudent creates an account and class membership atomically. Teachers
// can register students only in classes that they own.
func (s *Service) RegisterStudent(ctx context.Context, p domain.Parameters, in RegisterInput) (domain.Result, error) {
	if err := s.canManageClass(ctx, p); err != nil {
		return domain.Result{}, err
	}
	if in.LanguagePair == "" {
		in.LanguagePair = "ba-ru"
	}
	if !s.validPair(ctx, in.LanguagePair) {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_PAIR", "Недоступная языковая пара")
	}
	if len(in.Password) > 72 {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_PASSWORD", "Пароль должен быть не длиннее 72 байт")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_PASSWORD", "Пароль должен содержать 8–72 байта")
	}
	tx, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка транзакции")
	}
	defer tx.Rollback(ctx)
	var id string
	var recordCreateClassStudent CreateClassStudentRecord
	recordCreateClassStudent, err = tx.CreateClassStudent(ctx, strings.TrimSpace(in.Email), string(hash), strings.TrimSpace(in.DisplayName), in.LanguagePair, p.ID)
	id = recordCreateClassStudent.ID

	if err != nil {
		if errors.Is(err, ErrConflict) {
			return domain.Result{}, domain.Failure(domain.Conflict, "EMAIL_EXISTS", "Эта почта уже зарегистрирована. Добавьте существующего ученика по UUID.")
		} else {
			return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось создать аккаунт")
		}

	}
	if _, err = tx.EnrollCreatedStudent(ctx, p.ID, id); err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось добавить ученика")
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось сохранить ученика")
	}
	return domain.New(domain.Object{
		"id":          id,
		"displayName": in.DisplayName,
		"email":       in.Email,
		"classId":     p.ID,
	}), nil
}

func (s *Service) CreateSchool(ctx context.Context, p domain.Parameters, in CreateSchoolInput) (domain.Result, error) {
	var id string
	recordCreateSchool, err := s.store.CreateSchool(ctx, strings.TrimSpace(in.Name))
	id = recordCreateSchool.ID

	if err != nil {
		return domain.Result{}, domain.Failure(domain.Conflict, "SCHOOL_EXISTS", "Школа уже существует или не удалось её создать")
	}
	return domain.New(domain.Object{"id": id, "name": in.Name}), nil
}
