package service

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"bashkir-school/backend/internal/domain"
	"bashkir-school/backend/internal/security"
)

type loginRepository struct {
	Store
	user FindUserByEmailRecord
	err  error
}

func (r *loginRepository) FindUserByEmail(context.Context, string) (FindUserByEmailRecord, error) {
	return r.user, r.err
}

func TestLoginDistinguishesMissingEmailAndDatabaseFailure(t *testing.T) {
	for _, test := range []struct {
		name    string
		failure error
		kind    domain.ErrorKind
		code    string
	}{
		{"missing email", ErrNotFound, domain.NotFound, "EMAIL_NOT_FOUND"},
		{"database failure", errors.New("connection lost"), domain.Internal, "DB_ERROR"},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := New(&loginRepository{err: test.failure}, nil, nil)
			_, err := app.Login(context.Background(), domain.Parameters{}, LoginInput{Email: "missing@example.org", Password: "password"})
			var failure *domain.Error
			if !errors.As(err, &failure) || failure.Kind != test.kind || failure.Code != test.code {
				t.Fatalf("unexpected login error: %v", err)
			}
		})
	}
}
func TestLoginVerifiesPasswordBeforeIssuingToken(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	tokens, err := security.New("test_only_secret_long_enough_for_hs256_signing")
	if err != nil {
		t.Fatal(err)
	}
	store := &loginRepository{user: FindUserByEmailRecord{ID: "user-id", PasswordHash: string(hash), Role: "student", Name: "User", LanguagePair: "ba-ru"}}
	app := New(store, tokens, nil)
	_, err = app.Login(context.Background(), domain.Parameters{}, LoginInput{Email: "user@example.org", Password: "wrong-password"})
	var failure *domain.Error
	if !errors.As(err, &failure) || failure.Kind != domain.Unauthenticated {
		t.Fatalf("expected password error, got %v", err)
	}
	result, err := app.Login(context.Background(), domain.Parameters{}, LoginInput{Email: "user@example.org", Password: "correct-password"})
	if err != nil {
		t.Fatal(err)
	}
	token := result.Value.(domain.Object)["accessToken"].(string)
	id, role, err := tokens.Parse(token)
	if err != nil || id != "user-id" || role != "student" {
		t.Fatal("invalid login token")
	}
}
