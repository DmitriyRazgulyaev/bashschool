package service

import "time"

// Persistence results are application DTOs; no driver types cross this boundary.
type GetTaskSourceRecord struct {
	MaterialID  string
	Title       string
	Chapter     string
	StartOffset *int
	EndOffset   *int
	StartTimeMS *int64
	EndTimeMS   *int64
	HasAudio    bool
}

type FindPendingTaskRecord struct {
	ID          string
	FragmentID  string
	Text        string
	Translation string
	Pair        string
	Status      string
}

type AssignUnseenTaskRecord struct {
	ID string
}

type GetAssignedTaskRecord struct {
	FragmentID  string
	Text        string
	Translation string
	Pair        string
	Status      string
}

type ListTaskHistoryRecord struct {
	ID          string
	Text        string
	Translation string
	Status      string
	Date        time.Time
}

type LockOwnedTaskRecord struct {
	Status       string
	CandidateID  string
	FragmentID   string
	LanguagePair string
	Suggested    string
}

type LockUserProgressRecord struct {
	LockedUser string
}

type GetAnswerReceiptRecord struct {
	Key     string
	Receipt []byte
}

type GetUserProgressRecord struct {
	XP     int
	Streak int
}

type LockAnswerFragmentRecord struct {
	LockedFragment string
}

type UpsertCorrectionRecord struct {
	CandidateID string
}

type AdvanceUserProgressRecord struct {
	XP     int
	Streak int
}

type GetOwnedTaskRecord struct {
	TaskID       string
	FragmentID   string
	Text         string
	Translation  string
	LanguagePair string
	Status       string
}
