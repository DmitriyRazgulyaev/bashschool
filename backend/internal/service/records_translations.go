package service

type ExportConfirmedTranslationsRecord struct {
	FragmentID  string
	TextBa      string
	LangPair    string
	Translation string
	Source      string
	Confirmed   int
	Status      string
}

type ListConfirmedTranslationsRecord struct {
	ID           string
	FragmentID   string
	Text         string
	LanguagePair string
	Translation  string
	Count        int
	Status       string
	Title        string
	Chapter      string
	Updated      any
}

type GetTranslationRecord struct {
	ID           string
	FragmentID   string
	Text         string
	LanguagePair string
	Translation  string
	Count        int
	Status       string
	Title        string
	Chapter      string
}
