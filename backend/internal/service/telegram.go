package service

import (
	"context"
	"errors"
)

type Task = taskRecord
type Quest = questDefinition

func Quests() []Quest { return append([]Quest(nil), dailyQuestDefinitions...) }
func (s *Service) TelegramAccount(ctx context.Context, tg int64) (string, error) {
	return s.accountByTelegram(ctx, tg)
}
func (s *Service) LinkTelegram(ctx context.Context, tg int64, code string) error {
	return s.bindTelegram(ctx, tg, code)
}
func (s *Service) BotSession(ctx context.Context, tg int64) (string, *string, error) {
	var state string
	var task *string
	recordBotSession, err := s.store.BotSession(ctx, tg)
	state, task = recordBotSession.State, recordBotSession.Task

	if errors.Is(err, ErrNotFound) {
		return "idle", nil, nil
	}
	return state, task, err
}
func (s *Service) SetBotState(ctx context.Context, tg int64, state string, task *string) error {
	_, err := s.store.SaveBotSession(ctx, tg, state, task)
	return err
}
func (s *Service) BotProfile(ctx context.Context, id string) (xp, streak int, err error) {
	var recordBotProfile BotProfileRecord
	recordBotProfile, err = s.store.BotProfile(ctx, id)
	xp, streak = recordBotProfile.XP, recordBotProfile.Streak

	return
}
func (s *Service) BotPair(ctx context.Context, id string) (pair string, err error) {
	var recordBotPair BotPairRecord
	recordBotPair, err = s.store.BotPair(ctx, id)
	pair = recordBotPair.LanguagePair

	return
}
func (s *Service) BotSuggested(ctx context.Context, task, id string) (text string, err error) {
	var recordBotSuggested BotSuggestedRecord
	recordBotSuggested, err = s.store.BotSuggested(ctx, task, id)
	text = recordBotSuggested.Text

	return
}
func (s *Service) BotQuestProgress(ctx context.Context, id string) (total, corrections int, err error) {
	var recordBotQuestProgress BotQuestProgressRecord
	recordBotQuestProgress, err = s.store.BotQuestProgress(ctx, id)
	total, corrections = recordBotQuestProgress.Total,
		recordBotQuestProgress.Corrections

	return
}
func (s *Service) ValidPair(ctx context.Context, pair string) bool { return s.validPair(ctx, pair) }
func (s *Service) Daily(ctx context.Context, id, pair string) (Task, error) {
	return s.getDaily(ctx, id, pair)
}
