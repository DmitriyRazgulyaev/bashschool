package service

import "context"

// ConsensusRepository defines persistence operations for consensus.
type ConsensusRepository interface {
	ListConsensusCandidates(context.Context, string, string) ([]ListConsensusCandidatesRecord, error)
	ListLatestVotes(context.Context, string, string) ([]ListLatestVotesRecord, error)
	IsExpertConfirmed(context.Context, string, string) (IsExpertConfirmedRecord, error)
	GetExpertTranslation(context.Context, string, string) (GetExpertTranslationRecord, error)
	UpdateConfirmationCount(context.Context, string, string, int) (Command, error)
	SaveConsensus(context.Context, string, string, string, int, string) (Command, error)
	OpenModerationCase(context.Context, string, string) (Command, error)
	CloseConsensusCase(context.Context, string, string) (Command, error)
}
