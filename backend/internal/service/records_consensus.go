package service

// Persistence results are application DTOs; no driver types cross this boundary.
type ListConsensusCandidatesRecord struct {
	ID          string
	Text        string
	CanonicalID *string
	Correct     *bool
}

type ListLatestVotesRecord struct {
	ID string
}

type IsExpertConfirmedRecord struct {
	Expert bool
}

type GetExpertTranslationRecord struct {
	Selected string
}
