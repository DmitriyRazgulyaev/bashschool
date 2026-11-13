package service

// Persistence results are application DTOs; no driver types cross this boundary.
type CreateUserRecord struct {
	ID string
}

type FindUserByEmailRecord struct {
	ID           string
	PasswordHash string
	Role         string
	Name         string
	LanguagePair string
}

type GetUserProfileRecord struct {
	ID           string
	Name         string
	Email        string
	Role         string
	LanguagePair string
	XP           int
	Streak       int
	School       *string
}

type FindUserByTelegramRecord struct {
	ID string
}

type ConsumeTelegramLinkRecord struct {
	ID string
}
