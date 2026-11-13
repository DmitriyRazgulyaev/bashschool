package service

import "context"

// LearningRepository defines persistence operations for learning.
type LearningRepository interface {
	GetTaskSource(context.Context, string) (GetTaskSourceRecord, error)
	FindPendingTask(context.Context, string, string) (FindPendingTaskRecord, error)
	AssignUnseenTask(context.Context, string, string) (AssignUnseenTaskRecord, error)
	GetAssignedTask(context.Context, string) (GetAssignedTaskRecord, error)
	ListTaskHistory(context.Context, string, int) ([]ListTaskHistoryRecord, error)
	LockOwnedTask(context.Context, string, string) (LockOwnedTaskRecord, error)
	LockUserProgress(context.Context, string) (LockUserProgressRecord, error)
	GetAnswerReceipt(context.Context, string) (GetAnswerReceiptRecord, error)
	GetUserProgress(context.Context, string) (GetUserProgressRecord, error)
	LockAnswerFragment(context.Context, string) (LockAnswerFragmentRecord, error)
	UpsertCorrection(context.Context, string, string, string) (UpsertCorrectionRecord, error)
	SaveAnswer(context.Context, string, string, string, string, string, string) (Command, error)
	CompleteTask(context.Context, string) (Command, error)
	RecordTaskReward(context.Context, string, string) (Command, error)
	AdvanceUserProgress(context.Context, string) (AdvanceUserProgressRecord, error)
	SaveAnswerReceipt(context.Context, string, []byte) (Command, error)
	GetOwnedTask(context.Context, string, string) (GetOwnedTaskRecord, error)
}
