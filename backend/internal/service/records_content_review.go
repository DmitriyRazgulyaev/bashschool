package service

// Persistence results are application DTOs; no driver types cross this boundary.
type LockDraftForReviewRecord struct {
	Status string
}

type GetMergeGroupRecord struct {
	Fragment     string
	LanguagePair string
}

type LockMergeFragmentRecord struct {
	Locked string
}
