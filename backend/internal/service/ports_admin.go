package service

import "context"

// AdminRepository defines persistence operations for admin.
type AdminRepository interface {
	EnsureAdministrator(context.Context, string, string) (Command, error)
	ChangeUserRole(context.Context, string, string, string) (Command, error)
}
