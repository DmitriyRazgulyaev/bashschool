package service

import "context"

// TelegramRepository defines persistence operations for telegram.
type TelegramRepository interface {
	BotSession(context.Context, int64) (BotSessionRecord, error)
	SaveBotSession(context.Context, int64, string, *string) (Command, error)
	BotProfile(context.Context, string) (BotProfileRecord, error)
	BotPair(context.Context, string) (BotPairRecord, error)
	BotSuggested(context.Context, string, string) (BotSuggestedRecord, error)
	BotQuestProgress(context.Context, string) (BotQuestProgressRecord, error)
}
