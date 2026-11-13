package service

import "context"

// AuthRepository defines persistence operations for auth.
type AuthRepository interface {
	CreateUser(context.Context, string, string, string, string) (CreateUserRecord, error)
	FindUserByEmail(context.Context, string) (FindUserByEmailRecord, error)
	GetUserProfile(context.Context, string) (GetUserProfileRecord, error)
	UpdateUserProfile(context.Context, *string, *string, bool, *string, string) (Command, error)
	CreateTelegramLink(context.Context, string, string) (Command, error)
	FindUserByTelegram(context.Context, int64) (FindUserByTelegramRecord, error)
	ConsumeTelegramLink(context.Context, string) (ConsumeTelegramLinkRecord, error)
	DetachTelegramAccount(context.Context, int64) (Command, error)
	AssignTelegramAccount(context.Context, int64, string) (Command, error)
	UserRole(context.Context, string) (string, error)
}
