package service

import "context"

// LanguagePairsRepository defines persistence operations for language pairs.
type LanguagePairsRepository interface {
	IsPairActive(context.Context, string) (IsPairActiveRecord, error)
	ListActivePairs(context.Context) ([]ListActivePairsRecord, error)
	ListAllPairs(context.Context) ([]ListAllPairsRecord, error)
	CreatePair(context.Context, string, string, string, string) (CreatePairRecord, error)
	UpdatePair(context.Context, string, *string, *bool) (UpdatePairRecord, error)
}
