package service

// Persistence results are application DTOs; no driver types cross this boundary.
type ListModerationCasesRecord struct {
	ID           string
	Fragment     string
	Text         string
	LanguagePair string
	Status       string
	Created      any
}

type GetCaseCandidateRecord struct {
	FragmentID   string
	LanguagePair string
	Translation  string
}

type LockModerationFragmentRecord struct {
	LockedFragment string
}

type LockOpenCaseRecord struct {
	Open bool
}

type CountCandidateVotesRecord struct {
	Votes int
}

type ListCandidateVotesRecord struct {
	ID          string
	Translation string
	Votes       int
}

type CreateSeedCandidateRecord struct {
	ID string
}
