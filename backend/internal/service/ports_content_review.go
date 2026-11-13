package service

import "context"

// ContentReviewRepository defines persistence operations for content review.
type ContentReviewRepository interface {
	LockDraftForReview(context.Context, string, string) (LockDraftForReviewRecord, error)
	ResetSeedCandidates(context.Context, string, string) (Command, error)
	SaveReviewedCandidate(context.Context, string, string, string, bool, string) (
		Command, error)
	GetMergeGroup(context.Context, string, string, string) (GetMergeGroupRecord, error)
	LockMergeFragment(context.Context, string) (LockMergeFragmentRecord, error)
	MergeSynonymGroup(context.Context, string, string) (Command, error)
}
