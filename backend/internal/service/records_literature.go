package service

// Persistence results are application DTOs; no driver types cross this boundary.
type CreateTextMaterialRecord struct {
	ID string
}

type ListContentDraftsRecord struct {
	ID               string
	Text             string
	Title            string
	Chapter          string
	Candidates       int
	Status           string
	SourceLang       string
	Correct          *string
	Incorrect        *string
	ErrorExplanation *string
}

type GetGenerationSourceRecord struct {
	Text   string
	Source string
	Target string
	Status string
}

type LockDraftForGenerationRecord struct {
	Current string
}

type LockDraftForPublicationRecord struct {
	ReviewStatus string
}

type CountReviewedVariantsRecord struct {
	Count int
}
