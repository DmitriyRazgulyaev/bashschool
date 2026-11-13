package service

import "context"

// LiteratureRepository defines persistence operations for literature.
type LiteratureRepository interface {
	CreateTextMaterial(context.Context, string, string, string, string, string, string, string) (CreateTextMaterialRecord, error)
	CreateTextFragment(context.Context, string, string, string, string, int, int) (Command, error)
	ListContentDrafts(context.Context, string) ([]ListContentDraftsRecord, error)
	GetGenerationSource(context.Context, string, string) (GetGenerationSourceRecord, error)
	LockDraftForGeneration(context.Context, string) (LockDraftForGenerationRecord, error)
	SaveGeneratedCandidate(context.Context, string, string, string) (Command, error)
	SaveAISuggestion(context.Context, string, string, string, string, string, string) (Command, error)
	LockDraftForPublication(context.Context, string) (LockDraftForPublicationRecord, error)
	CountReviewedVariants(context.Context, string, string) (CountReviewedVariantsRecord, error)
	PublishReviewedFragment(context.Context, string, string) (Command, error)
	RejectDraft(context.Context, string, string) (Command, error)
}
