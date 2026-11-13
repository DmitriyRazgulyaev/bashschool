package service

// Persistence results are application DTOs; no driver types cross this boundary.
type LeaderboardRecord struct {
	ID, Name string
	Score    int64
}
