package service

import "context"

type GamificationRepository interface {
	CountCompletedTasks(context.Context, string) (CountCompletedTasksRecord, error)
	GetLearningProgress(context.Context, string) (GetLearningProgressRecord, error)
	ListAchievements(context.Context) ([]ListAchievementsRecord, error)
}
