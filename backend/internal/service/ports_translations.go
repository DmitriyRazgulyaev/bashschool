package service

import "context"

type TranslationsRepository interface {
	ExportConfirmedTranslations(context.Context, string) ([]ExportConfirmedTranslationsRecord, error)
	GetTranslation(context.Context, string) (GetTranslationRecord, error)
	ListConfirmedTranslations(context.Context, string) ([]ListConfirmedTranslationsRecord, error)
}
