package service

type ListAchievementsRecord struct {
	Code      string
	Title     string
	Desc      string
	Threshold int
}

type CountCompletedTasksRecord struct {
	Done int
}

type GetLearningProgressRecord struct {
	XP        int
	Streak    int
	Longest   int
	Completed int
	Confirmed int
}
