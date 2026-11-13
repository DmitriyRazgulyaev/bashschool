package service

import (
	"context"
	"time"

	"bashkir-school/backend/internal/domain"
)

// Daily challenges are server-owned; clients cannot choose their rewards or progress.
type questDefinition struct {
	Code, Title, Description string
	Goal, RewardXP           int
}

var dailyQuestDefinitions = []questDefinition{
	{"three_translations", "Первые 3 перевода", "Подтвердите или исправьте 3 предложенных перевода", 3, 50},
	{"five_translations", "Мастер переводов", "Выполните 5 переводов за сегодня", 5, 100},
	{"one_correction", "Внимательный переводчик", "Предложите собственное исправление перевода", 1, 25},
}

// beginDailyQuests ensures that each user receives the same challenge list throughout a day.
func (s *Service) beginDailyQuests(ctx context.Context, tx Transaction, userID string) error {
	_, err := tx.EnsureDailyQuests(ctx, userID)
	return err
}

// awardDailyQuestBonuses is called in the same transaction as task completion.
// It receives a locked user row from answerCore, preventing concurrent double rewards.
func (s *Service) awardDailyQuestBonuses(ctx context.Context, tx Transaction, userID string) (int, []string, error) {
	if err := s.beginDailyQuests(ctx, tx, userID); err != nil {
		return 0, nil, err
	}
	var total, corrections int
	recordCountDailyAnswers, err := tx.CountDailyAnswers(ctx, userID)
	total, corrections = recordCountDailyAnswers.Total,
		recordCountDailyAnswers.Corrections

	if err != nil {
		return 0, nil, err
	}

	var bonus int
	unlocked := []string{}
	for _, quest := range dailyQuestDefinitions {
		count := total
		if quest.Code == "one_correction" {
			count = corrections
		}
		if count < quest.Goal {
			continue
		}
		var questID string
		// A row may be rewarded once; RETURNING only occurs on the first update.
		recordClaimQuestReward, err := tx.ClaimQuestReward(ctx, userID, quest.Code)
		questID = recordClaimQuestReward.QuestID

		if err == ErrNotFound {
			continue
		}
		if err != nil {
			return 0, nil, err
		}
		_, err = tx.RecordQuestReward(ctx, userID, questID, quest.RewardXP)
		if err != nil {
			return 0, nil, err
		}
		bonus += quest.RewardXP
		unlocked = append(unlocked, quest.Title)
	}
	if bonus > 0 {
		_, err = tx.AddQuestExperience(ctx, bonus, userID)
		if err != nil {
			return 0, nil, err
		}
	}
	return bonus, unlocked, nil
}

func (s *Service) DailyQuests(ctx context.Context, p domain.Parameters) (domain.Result, error) {

	userID := p.Actor.ID
	tx, err := s.store.Begin(ctx)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось получить ежедневные цели")
	}
	defer tx.Rollback(ctx)
	if err := s.beginDailyQuests(ctx, tx, userID); err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось создать ежедневные цели")
	}
	var total, corrections int
	var recordGetDailyAnswerCounts GetDailyAnswerCountsRecord
	recordGetDailyAnswerCounts, err = tx.GetDailyAnswerCounts(ctx, userID)
	total, corrections = recordGetDailyAnswerCounts.Total,
		recordGetDailyAnswerCounts.Corrections

	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось подсчитать прогресс")
	}

	rows1, err := tx.ListDailyQuestClaims(ctx, userID)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось загрузить цели")
	}
	claimed := map[string]*time.Time{}
	for _, row := range rows1 {
		var code string
		var awarded *time.Time
		{
			code, awarded = row.Code, row.Awarded
		}

		claimed[code] = awarded
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка фиксации целей")
	}

	items := []domain.Object{}
	for _, definition := range dailyQuestDefinitions {
		count := total
		if definition.Code == "one_correction" {
			count = corrections
		}
		if count > definition.Goal {
			count = definition.Goal
		}
		items = append(items, domain.Object{
			"code": definition.Code, "title": definition.Title,
			"description": definition.Description, "goal": definition.Goal,
			"progress": count, "rewardXp": definition.RewardXP,
			"completed": count >= definition.Goal, "rewarded": claimed[definition.Code] != nil,
		})
	}
	return domain.OK(domain.Object{"date": time.Now().UTC().Format("2006-01-02"), "quests": items}), nil
}
