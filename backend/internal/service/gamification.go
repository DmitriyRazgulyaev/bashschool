package service

import (
	"context"

	"bashkir-school/backend/internal/domain"
)

func (s *Service) Progress(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	var xp, streak, longest, completed, confirmed int
	recordGetLearningProgress, err := s.store.GetLearningProgress(ctx, p.Actor.ID)
	xp, streak, longest, completed, confirmed = recordGetLearningProgress.XP, recordGetLearningProgress.Streak, recordGetLearningProgress.Longest, recordGetLearningProgress.Completed, recordGetLearningProgress.Confirmed

	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось получить прогресс")
	}
	level := 1 + xp/200
	return domain.OK(domain.Object{
		"xp":                     xp,
		"level":                  level,
		"xpInLevel":              xp % 200,
		"xpForNextLevel":         200,
		"streak":                 streak,
		"longestStreak":          longest,
		"completedTasks":         completed,
		"confirmedContributions": confirmed,
	}), nil
}

func (s *Service) Achievements(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	var done int
	{
		recordCountCompletedTasks, err := s.store.CountCompletedTasks(ctx, p.Actor.ID)
		done = recordCountCompletedTasks.Done

		if err != nil {
			return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось получить достижения")
		}
	}
	rows1, err := s.store.ListAchievements(ctx)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка достижений")
	}

	out := []domain.Object{}
	for _, row := range rows1 {
		var code, title, desc string
		var threshold int
		{
			code, title, desc, threshold = row.Code, row.Title, row.Desc, row.Threshold

			out = append(out, domain.Object{
				"code":        code,
				"name":        title,
				"description": desc,
				"unlocked":    done >= threshold,
				"threshold":   threshold,
			})
		}

	}

	return domain.OK(out), nil
}
