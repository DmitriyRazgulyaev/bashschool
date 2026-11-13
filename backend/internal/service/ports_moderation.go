package service

import "context"

// ModerationRepository defines persistence operations for moderation.
type ModerationRepository interface {
	ListModerationCases(context.Context) ([]ListModerationCasesRecord, error)
	GetCaseCandidate(context.Context, string, string) (GetCaseCandidateRecord, error)
	LockModerationFragment(context.Context, string) (LockModerationFragmentRecord, error)
	LockOpenCase(context.Context, string) (LockOpenCaseRecord, error)
	CountCandidateVotes(context.Context, string, string, string) (CountCandidateVotesRecord, error)
	MarkReviewedCandidates(context.Context, string, string, string) (Command, error)
	SaveExpertTranslation(context.Context, string, string, string, int) (Command,
		error)
	ResolveModerationCase(context.Context, string, string, string) (Command, error)
	ListCandidateVotes(context.Context, string, string) ([]ListCandidateVotesRecord, error)
	CreateSeedCandidate(context.Context, string, string, string) (CreateSeedCandidateRecord, error)
}
