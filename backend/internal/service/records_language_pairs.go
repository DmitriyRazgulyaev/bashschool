package service

// Persistence results are application DTOs; no driver types cross this boundary.
type IsPairActiveRecord struct {
	Ok bool
}

type ListActivePairsRecord struct {
	Code       string
	SourceLang string
	TargetLang string
	Name       string
	Active     bool
}

type ListAllPairsRecord struct {
	Code       string
	SourceLang string
	TargetLang string
	Name       string
	Active     bool
}

type CreatePairRecord struct {
	Active bool
}

type UpdatePairRecord struct {
	Code       string
	SourceLang string
	TargetLang string
	Name       string
	Active     bool
}
