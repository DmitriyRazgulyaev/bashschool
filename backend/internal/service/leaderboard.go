package service

import (
	"context"

	"bashkir-school/backend/internal/domain"
)

func (s *Service) Leaderboard(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	scope, period, metric := p.Filter("scope", "global"), p.Filter("period", "weekly"), p.Filter("metric", "xp")
	periodStart := map[string]string{"daily": "day", "weekly": "week", "monthly": "month", "yearly": "year", "all": ""}
	_, ok := periodStart[period]
	if !ok {
		return domain.Result{}, domain.Failure(domain.Invalid, "BAD_PERIOD", "Доступны daily, weekly, monthly, yearly, all")
	}
	if scope != "global" && scope != "school" && scope != "class" {
		return domain.Result{}, domain.Failure(domain.Invalid, "BAD_SCOPE", "Недопустимый рейтинг")
	}
	if metric != "xp" && metric != "activity" && metric != "vocabulary" && metric != "corrections" && metric != "errors" {
		return domain.Result{}, domain.Failure(domain.Invalid, "BAD_METRIC", "Недопустимая метрика")
	}
	rows, err := s.store.Leaderboard(ctx, scope, period, metric)
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Не удалось получить рейтинг")
	}

	entries := []domain.Object{}
	for _, entry := range rows {
		entries = append(entries, domain.Object{
			"rank":        len(entries) + 1,
			"entityId":    entry.ID,
			"displayName": entry.Name,
			"score":       entry.Score,
		})
	}

	return domain.OK(domain.Object{
		"scope":   scope,
		"period":  period,
		"metric":  metric,
		"entries": entries,
	}), nil
}
