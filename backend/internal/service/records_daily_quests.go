package service

import "time"

// Persistence results are application DTOs; no driver types cross this boundary.
type CountDailyAnswersRecord struct {
	Total       int
	Corrections int
}

type ClaimQuestRewardRecord struct {
	QuestID string
}

type GetDailyAnswerCountsRecord struct {
	Total       int
	Corrections int
}

type ListDailyQuestClaimsRecord struct {
	Code    string
	Awarded *time.Time
}
