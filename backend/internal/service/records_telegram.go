package service

// Persistence results are application DTOs; no driver types cross this boundary.
type BotSessionRecord struct {
	State string
	Task  *string
}

type BotProfileRecord struct {
	XP     int
	Streak int
}

type BotPairRecord struct {
	LanguagePair string
}

type BotSuggestedRecord struct {
	Text string
}

type BotQuestProgressRecord struct {
	Total       int
	Corrections int
}
