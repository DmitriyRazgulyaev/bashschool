package service

import "context"

// DailyQuestsRepository defines persistence operations for daily quests.
type DailyQuestsRepository interface {
	EnsureDailyQuests(context.Context, string) (Command, error)
	CountDailyAnswers(context.Context, string) (CountDailyAnswersRecord, error)
	ClaimQuestReward(context.Context, string, string) (ClaimQuestRewardRecord, error)
	RecordQuestReward(context.Context, string, string, int) (Command, error)
	AddQuestExperience(context.Context, int, string) (Command, error)
	GetDailyAnswerCounts(context.Context, string) (GetDailyAnswerCountsRecord, error)
	ListDailyQuestClaims(context.Context, string) ([]ListDailyQuestClaimsRecord, error)
}
