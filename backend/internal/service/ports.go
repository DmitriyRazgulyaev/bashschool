package service

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("record not found")
var ErrConflict = errors.New("record already exists")

type Command interface{ RowsAffected() int64 }
type Queries interface {
	ConsensusRepository
	LearningRepository
	TelegramRepository
	AudioRepository
	EducationRepository
	EducationRegistrationRepository
	AuthRepository
	ModerationRepository
	LiteratureRepository
	LeaderboardRepository
	GamificationRepository
	CatalogRepository
	TranslationsRepository
	LanguagePairsRepository
	DailyQuestsRepository
	AdminRepository
	ContentReviewRepository
}
type Store interface {
	Queries
	Begin(context.Context) (Transaction, error)
}
type Transaction interface {
	Queries
	Commit(context.Context) error
	Rollback(context.Context) error
}
