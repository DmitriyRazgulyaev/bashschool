package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"bashkir-school/backend/internal/domain"
	"bashkir-school/backend/internal/security"

	"golang.org/x/crypto/bcrypt"
)

type RegisterInput struct {
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=8,max=72"`
	DisplayName  string `json:"displayName" binding:"required,min=2"`
	LanguagePair string `json:"languagePair"`
}
type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (s *Service) Register(ctx context.Context, p domain.Parameters, in RegisterInput) (domain.Result, error) {
	if in.LanguagePair == "" {
		in.LanguagePair = "ba-ru"
	}
	if !s.validPair(ctx, in.LanguagePair) {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_PAIR", "Языковая пара не найдена или отключена")
	}
	if len(in.Password) > 72 {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_PASSWORD", "Пароль должен быть не длиннее 72 байт")
	}
	pwd, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "SERVER_ERROR", "Ошибка хеширования")
	}
	var id string
	var recordCreateUser CreateUserRecord
	recordCreateUser, err = s.store.CreateUser(ctx, strings.TrimSpace(in.Email), string(pwd), in.DisplayName, in.LanguagePair)
	id = recordCreateUser.ID

	if err != nil {
		if errors.Is(err, ErrConflict) {
			return domain.Result{}, domain.Failure(domain.Conflict, "EMAIL_EXISTS", "Эта почта уже зарегистрирована. Войдите в аккаунт.")
		} else {
			slog.Error("registration failed", "error", err)
			return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось создать аккаунт. Попробуйте позже.")
		}

	}
	t, err := s.tokens.Issue(id, "student")
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "TOKEN_ERROR", "Не удалось создать сессию")
	}
	return domain.New(domain.Object{"accessToken": t, "user": domain.Object{
		"id":           id,
		"displayName":  in.DisplayName,
		"role":         "student",
		"languagePair": in.LanguagePair,
	}}), nil
}
func (s *Service) Login(ctx context.Context, p domain.Parameters, in LoginInput) (domain.Result, error) {
	var id, hash, role, name, pair string
	recordFindUserByEmail, err := s.store.FindUserByEmail(ctx, strings.TrimSpace(in.Email))
	id, hash, role, name, pair = recordFindUserByEmail.ID, recordFindUserByEmail.PasswordHash, recordFindUserByEmail.Role, recordFindUserByEmail.Name, recordFindUserByEmail.LanguagePair

	if errors.Is(err, ErrNotFound) {
		return domain.Result{}, domain.Failure(domain.NotFound, "EMAIL_NOT_FOUND", "Пользователь с такой почтой не зарегистрирован. Создайте аккаунт.")
	}
	if err != nil {
		slog.Error("login lookup failed", "error", err)
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось выполнить вход. Попробуйте позже.")
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		return domain.Result{}, domain.Failure(domain.Unauthenticated, "INVALID_CREDENTIALS", "Неверный email или пароль")
	}
	t, err := s.tokens.Issue(id, role)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "TOKEN_ERROR", "Не удалось создать сессию")
	}
	return domain.OK(domain.Object{"accessToken": t, "user": domain.Object{
		"id":           id,
		"displayName":  name,
		"role":         role,
		"languagePair": pair,
	}}), nil
}
func (s *Service) Me(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	var id, name, email, role, pair string
	var xp, streak int
	var school *string
	recordGetUserProfile, err := s.store.GetUserProfile(ctx, p.Actor.ID)
	id, name, email, role, pair, xp, streak, school = recordGetUserProfile.ID, recordGetUserProfile.Name, recordGetUserProfile.Email, recordGetUserProfile.Role, recordGetUserProfile.LanguagePair,

		recordGetUserProfile.XP, recordGetUserProfile.Streak, recordGetUserProfile.School

	if err != nil {
		return domain.Result{}, domain.Failure(domain.NotFound, "NOT_FOUND", "Пользователь не найден")
	}
	return domain.OK(domain.Object{
		"id":           id,
		"displayName":  name,
		"email":        email,
		"role":         role,
		"languagePair": pair,
		"xp":           xp,
		"streak":       streak,
		"schoolId":     school,
	}), nil
}
func (s *Service) UpdateMe(ctx context.Context, p domain.Parameters, in UpdateMeInput) (domain.Result, error) {
	if in.DisplayName != nil && len(strings.TrimSpace(*in.DisplayName)) < 2 {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_NAME", "Имя должно содержать минимум 2 символа")
	}
	if in.LanguagePair != nil && !s.validPair(ctx, *in.LanguagePair) {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_PAIR", "Недоступная языковая пара")
	}
	if in.ClearSchool && in.SchoolID != nil {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_SCHOOL", "Нельзя одновременно выбрать и удалить школу")
	}
	updateSchool := in.SchoolID != nil || in.ClearSchool
	_, err := s.store.UpdateUserProfile(ctx, in.DisplayName, in.LanguagePair, updateSchool, in.SchoolID, p.Actor.ID)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Invalid, "INVALID_SCHOOL", "Неверная школа или не удалось обновить профиль")
	}
	return s.Me(ctx, p)
}
func (s *Service) LinkCode(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	code := security.Code()
	_, err := s.store.CreateTelegramLink(ctx, code, p.Actor.ID)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось создать код")
	}
	return domain.OK(domain.Object{
		"code":             code,
		"instruction":      "Отправьте боту /start " + code,
		"expiresInSeconds": 900,
	}), nil
}
func (s *Service) accountByTelegram(ctx context.Context, tg int64) (string, error) {
	var id string
	recordFindUserByTelegram, err := s.store.FindUserByTelegram(ctx, tg)
	id = recordFindUserByTelegram.ID

	return id, err
}
func (s *Service) bindTelegram(ctx context.Context, tg int64, code string) error {
	tx, err := s.store.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var id string
	var recordConsumeTelegramLink ConsumeTelegramLinkRecord
	recordConsumeTelegramLink, err = tx.ConsumeTelegramLink(ctx, code)
	id = recordConsumeTelegramLink.ID

	if err != nil {
		return err
	}
	_, err = tx.DetachTelegramAccount(ctx, tg)
	if err != nil {
		return err
	}
	ct, err := tx.AssignTelegramAccount(ctx, tg, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() != 1 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}
