package service

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strconv"

	"context"

	"bashkir-school/backend/internal/domain"
)

func (s *Service) Translations(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	rows1, err := s.store.ListConfirmedTranslations(ctx, p.Filter("langPair", ""))
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка переводов")
	}

	out := []domain.Object{}
	for _, row := range rows1 {
		var id, fid, text, pair, tr, status, title, chapter string
		var count int
		var updated any
		{
			id, fid, text, pair, tr, count, status, title,
				chapter, updated = row.ID, row.FragmentID,
				row.Text, row.LanguagePair, row.Translation,

				row.Count, row.Status, row.Title, row.Chapter, row.Updated

			out = append(out, domain.Object{
				"id":          id,
				"fragmentId":  fid,
				"textBa":      text,
				"langPair":    pair,
				"translation": tr,
				"confirmed":   count,
				"status":      status,
				"source":      domain.Object{"title": title, "chapter": chapter},
				"updatedAt":   updated,
			})
		}

	}

	return domain.OK(out), nil
}

func (s *Service) ExportTranslations(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	format := p.Filter("format", "json")
	if format != "csv" && format != "json" {
		return domain.Result{}, domain.Failure(domain.Invalid, "BAD_FORMAT", "Доступны csv и json")
	}
	rows1, err := s.store.ExportConfirmedTranslations(ctx, p.Filter("langPair", ""))
	if err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "DB_ERROR", "Ошибка выгрузки")
	}

	type item struct {
		FragmentID  string          `json:"fragment_id"`
		TextBa      string          `json:"text_ba"`
		LangPair    string          `json:"lang_pair"`
		Translation string          `json:"translation"`
		Source      json.RawMessage `json:"source"`
		Confirmed   int             `json:"confirmed"`
		Status      string          `json:"status"`
	}
	out := []item{}
	for _, row := range rows1 {
		var entry item
		var source string
		{
			entry.FragmentID, entry.TextBa, entry.LangPair, entry.Translation, source, entry.Confirmed, entry.Status = row.FragmentID, row.TextBa,

				row.LangPair,

				row.Translation, row.Source, row.Confirmed,
				row.Status
		}

		entry.Source = json.RawMessage(source)
		out = append(out, entry)
	}

	if format == "json" {
		return domain.OK(out), nil
	}
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err = writer.Write([]string{"fragment_id", "text_ba", "lang_pair", "translation", "source", "confirmed", "status"}); err != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "EXPORT_ERROR", "Ошибка формирования CSV")
	}
	for _, entry := range out {
		if err = writer.Write([]string{entry.FragmentID, entry.TextBa, entry.LangPair, entry.Translation, string(entry.Source), strconv.Itoa(entry.Confirmed), entry.Status}); err != nil {
			return domain.Result{}, domain.Failure(domain.Internal, "EXPORT_ERROR", "Ошибка формирования CSV")
		}
	}
	writer.Flush()
	if writer.Error() != nil {
		return domain.Result{}, domain.Failure(domain.Internal, "EXPORT_ERROR", "Ошибка формирования CSV")
	}
	return domain.Download(buffer.Bytes(), "translations.csv", "text/csv; charset=utf-8"), nil
}

func (s *Service) TranslationDetail(ctx context.Context, p domain.Parameters) (domain.Result, error) {
	var id, fid, text, pair, tr, status, title, chapter string
	var count int
	recordGetTranslation, err := s.store.GetTranslation(ctx, p.ID)
	id, fid, text, pair, tr, count, status, title,
		chapter = recordGetTranslation.ID, recordGetTranslation.FragmentID,
		recordGetTranslation.Text, recordGetTranslation.LanguagePair, recordGetTranslation.Translation,

		recordGetTranslation.Count, recordGetTranslation.Status, recordGetTranslation.Title, recordGetTranslation.Chapter

	if err != nil {
		return domain.Result{}, domain.Failure(domain.NotFound, "NOT_FOUND", "Перевод не найден")
	}
	return domain.OK(domain.Object{
		"id":          id,
		"fragmentId":  fid,
		"textBa":      text,
		"langPair":    pair,
		"translation": tr,
		"confirmed":   count,
		"status":      status,
		"source":      domain.Object{"title": title, "chapter": chapter},
	}), nil
}
