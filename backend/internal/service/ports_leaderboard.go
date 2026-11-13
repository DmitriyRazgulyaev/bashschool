package service

import "context"

// LeaderboardRepository defines persistence operations for leaderboard.
type LeaderboardRepository interface {
	Leaderboard(context.Context, string, string, string) ([]LeaderboardRecord, error)
}
